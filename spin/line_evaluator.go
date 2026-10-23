package spin

import "fmt"

type LineEvaluator struct {
	paylines      [][]int
	pays          []PaytableEntry
	wilds         map[string]bool
	wildPaySymbol string
}

type LineResult struct {
	wins []LineWin
}

type LineMatch struct {
	baseSymbol            string
	matchCount            int // leadingWildMatchCount + baseMatchCount
	leadingWildMatchCount int
}

func newLineEvaluator(paylines [][]int, paytable Paytable, wildSymbols []string) *LineEvaluator {
	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}
	wildPaySymbol := ""
	if len(wildSymbols) > 0 {
		wildPaySymbol = wildSymbols[0]
	}
	return &LineEvaluator{paylines: paylines, pays: paytable.Line, wilds: wilds, wildPaySymbol: wildPaySymbol}
}

func (e *LineEvaluator) evaluate(b Board, activeLines int, betPerLine int64) (LineResult, error) {
	if activeLines <= 0 || activeLines > len(e.paylines) {
		return LineResult{}, fmt.Errorf("active lines %d is outside 1..%d", activeLines, len(e.paylines))
	}

	var result LineResult
	for lineIndex := 0; lineIndex < activeLines; lineIndex++ {
		win, ok := e.evaluateLine(lineIndex, e.paylines[lineIndex], b, betPerLine)
		if !ok {
			continue
		}

		result.wins = append(result.wins, win)
	}
	return result, nil
}

// 計算線獎
func (e *LineEvaluator) evaluateLine(lineIndex int, payline []int, b Board, betPerLine int64) (LineWin, bool) {
	match := analyzeLine(symbolsForPayline(b, payline), e.wilds)
	pay, ok := e.bestPay(match)
	if !ok {
		return LineWin{}, false
	}
	return LineWin{
		LineIndex:          lineIndex,
		PaytableEntryIndex: pay.Index,
		Payout:             pay.Odds * betPerLine,
		Positions:          linePositions(payline, pay.Count),
	}, true
}

func symbolsForPayline(b Board, payline []int) []string {
	symbols := make([]string, len(payline))
	for reel, row := range payline {
		symbols[reel] = b[reel][row]
	}
	return symbols
}

func analyzeLine(symbols []string, wilds map[string]bool) LineMatch {
	baseSymbol := ""
	baseMatchCount := 0
	leadingWildMatchCount := 0
	for _, symbol := range symbols {
		if baseSymbol == "" {
			if wilds[symbol] {
				leadingWildMatchCount++
				continue
			}

			baseSymbol = symbol
			baseMatchCount++
			continue
		}

		if symbol != baseSymbol && !wilds[symbol] {
			break
		}

		baseMatchCount++
	}

	return LineMatch{
		baseSymbol:            baseSymbol,
		matchCount:            leadingWildMatchCount + baseMatchCount,
		leadingWildMatchCount: leadingWildMatchCount,
	}
}

func (e *LineEvaluator) bestPay(match LineMatch) (PaytableEntry, bool) {
	if match.baseSymbol == "" {
		return e.bestPayForCount(match.matchCount)
	}

	basePay, hasBasePay := e.bestPayForSymbol(match.baseSymbol, match.matchCount)
	wildPay, hasWildPay := e.bestPayForSymbol(e.wildPaySymbol, match.leadingWildMatchCount)
	if !hasBasePay && !hasWildPay {
		return PaytableEntry{}, false
	}
	if hasWildPay && (!hasBasePay || wildPay.Odds > basePay.Odds) {
		return wildPay, true
	}
	return basePay, true
}

func (e *LineEvaluator) bestPayForCount(count int) (PaytableEntry, bool) {
	var best PaytableEntry
	found := false
	for _, pay := range e.pays {
		if pay.Count > count {
			continue
		}

		if !found || pay.Odds > best.Odds {
			best = pay
			found = true
		}
	}
	return best, found
}

func (e *LineEvaluator) bestPayForSymbol(symbol string, count int) (PaytableEntry, bool) {
	var best PaytableEntry
	found := false
	for _, pay := range e.pays {
		if pay.Symbol != symbol || pay.Count > count {
			continue
		}

		if !found || pay.Odds > best.Odds {
			best = pay
			found = true
		}
	}
	return best, found
}

func linePositions(payline []int, count int) []Position {
	if count > len(payline) {
		count = len(payline)
	}
	positions := make([]Position, 0, count)
	for reel := 0; reel < count; reel++ {
		positions = append(positions, Position{Reel: reel, Row: payline[reel]})
	}
	return positions
}
