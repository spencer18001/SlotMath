package spin

import "fmt"

type payKey struct {
	symbol string
	count  int
}
type payRule struct {
	index     int
	odds      int64
	freeSpins int
}
type lineMatch struct {
	baseSymbol       string
	baseCount        int
	leadingWildCount int
}
type lineResult struct {
	wins     []LineWin
	totalWin int64
}

type lineEvaluator struct {
	paylines      [][]int
	paytable      map[payKey]payRule
	wilds         map[string]bool
	wildPaySymbol string
}

func newLineEvaluator(paylines [][]int, paytable Paytable, wildSymbols []string) *lineEvaluator {
	linePays := make(map[payKey]payRule)
	for index, pay := range paytable.Line {
		linePays[payKey{symbol: pay.Symbol, count: pay.Count}] = payRule{index: index, odds: pay.Odds}
	}
	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}
	wildPaySymbol := ""
	if len(wildSymbols) > 0 {
		wildPaySymbol = wildSymbols[0]
	}
	return &lineEvaluator{paylines: paylines, paytable: linePays, wilds: wilds, wildPaySymbol: wildPaySymbol}
}

func (e *lineEvaluator) evaluate(b Board, activeLines int, betPerLine int64) (lineResult, error) {
	if activeLines <= 0 || activeLines > len(e.paylines) {
		return lineResult{}, fmt.Errorf("active lines %d is outside 1..%d", activeLines, len(e.paylines))
	}
	if betPerLine <= 0 {
		return lineResult{}, fmt.Errorf("bet per line must be greater than zero")
	}
	var result lineResult
	for lineIndex := 0; lineIndex < activeLines; lineIndex++ {
		win, ok := e.evaluateLine(lineIndex, e.paylines[lineIndex], b, betPerLine)
		if !ok {
			continue
		}
		result.wins = append(result.wins, win)
		result.totalWin += win.Payout
	}
	return result, nil
}

func (e *lineEvaluator) evaluateLine(lineIndex int, payline []int, b Board, betPerLine int64) (LineWin, bool) {
	match := analyzeLine(symbolsForPayline(b, payline), e.wilds)
	baseRule, hasBaseRule := e.lookup(match.baseSymbol, match.baseCount)
	wildRule, hasWildRule := e.lookup(e.wildPaySymbol, match.leadingWildCount)
	if !hasBaseRule && !hasWildRule {
		return LineWin{}, false
	}
	if hasWildRule && (!hasBaseRule || wildRule.odds > baseRule.odds) {
		return buildLineWin(lineIndex, wildRule, betPerLine), true
	}
	return buildLineWin(lineIndex, baseRule, betPerLine), true
}

func analyzeLine(symbols []string, wilds map[string]bool) lineMatch {
	baseSymbol := ""
	baseMatches := 0
	leadingWildCount := 0
	for _, symbol := range symbols {
		if baseSymbol == "" {
			if wilds[symbol] {
				leadingWildCount++
				continue
			}
			baseSymbol = symbol
			baseMatches++
			continue
		}
		if symbol != baseSymbol && !wilds[symbol] {
			break
		}
		baseMatches++
	}
	baseCount := 0
	if baseSymbol != "" {
		baseCount = leadingWildCount + baseMatches
	}
	return lineMatch{baseSymbol: baseSymbol, baseCount: baseCount, leadingWildCount: leadingWildCount}
}

func (e *lineEvaluator) lookup(symbol string, count int) (payRule, bool) {
	if symbol == "" || count <= 0 {
		return payRule{}, false
	}
	rule, ok := e.paytable[payKey{symbol: symbol, count: count}]
	return rule, ok && rule.odds > 0
}

func buildLineWin(lineIndex int, rule payRule, betPerLine int64) LineWin {
	return LineWin{LineIndex: lineIndex, PayRuleIndex: rule.index, Payout: rule.odds * betPerLine}
}

func symbolsForPayline(b Board, payline []int) []string {
	symbols := make([]string, len(payline))
	for reel, row := range payline {
		symbols[reel] = b[reel][row]
	}
	return symbols
}
