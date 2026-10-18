package spin

import "fmt"

// 連線資訊
type LineMatch struct {
	baseSymbol            string
	matchCount            int
	leadingWildMatchCount int
}

// 線獎結果
type LineResult struct {
	wins []LineWin // 線獎列表
}

// 線獎計算
type LineEvaluator struct {
	paylines      [][]int
	pays          []PaytableEntry
	wilds         map[string]bool
	wildPaySymbol string
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

// 計算所有線獎
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

// 由盤面取得線上的 symbol
func symbolsForPayline(b Board, payline []int) []string {
	symbols := make([]string, len(payline))
	for reel, row := range payline {
		symbols[reel] = b[reel][row]
	}
	return symbols
}

// 取得最好的賠付規則
func (e *LineEvaluator) bestPay(match LineMatch) (PaytableEntry, bool) {
	if match.baseSymbol == "" {
		return e.bestRuleForCount(match.matchCount)
	}

	basePay, hasBasePay := e.bestRuleForSymbol(match.baseSymbol, match.matchCount)
	wildPay, hasWildPay := e.bestRuleForSymbol(e.wildPaySymbol, match.leadingWildMatchCount)
	if !hasBasePay && !hasWildPay {
		return PaytableEntry{}, false
	}
	if hasWildPay && (!hasBasePay || wildPay.Odds > basePay.Odds) {
		return wildPay, true
	}
	return basePay, true
}

// 取得指定 symbol 在連線數內最好的賠付規則
func (e *LineEvaluator) bestRuleForSymbol(symbol string, count int) (PaytableEntry, bool) {
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

// 取得連線數內最好的賠付規則
func (e *LineEvaluator) bestRuleForCount(count int) (PaytableEntry, bool) {
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

// 分析連線資訊
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
