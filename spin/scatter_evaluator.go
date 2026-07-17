package spin

import "fmt"

type scatterResult struct {
	wins      []ScatterWin
	freeSpins int
}

type scatterEvaluator struct {
	paytable map[payKey]payRule
	symbols  map[string]bool
	payBet   int64
}

func newScatterEvaluator(scatterSymbols []string, paytable Paytable, payBet int64) *scatterEvaluator {
	if payBet <= 0 {
		payBet = 1
	}
	symbols := make(map[string]bool)
	for _, symbol := range scatterSymbols {
		symbols[symbol] = true
	}
	scatterPays := make(map[payKey]payRule)
	for index, pay := range paytable.Scatter {
		scatterPays[payKey{symbol: pay.Symbol, count: pay.Count}] = payRule{index: index, odds: pay.Odds, freeSpins: pay.FreeSpins}
	}
	return &scatterEvaluator{paytable: scatterPays, symbols: symbols, payBet: payBet}
}

func (e *scatterEvaluator) evaluate(b Board, totalBet int64) (scatterResult, error) {
	if totalBet <= 0 {
		return scatterResult{}, fmt.Errorf("total bet must be greater than zero")
	}
	positionsBySymbol := make(map[string][]Position)
	for reelIndex, reel := range b {
		for rowIndex, symbol := range reel {
			if e.symbols[symbol] {
				positionsBySymbol[symbol] = append(positionsBySymbol[symbol], Position{Reel: reelIndex, Row: rowIndex})
			}
		}
	}
	var result scatterResult
	for symbol, positions := range positionsBySymbol {
		rule, ok := e.paytable[payKey{symbol: symbol, count: len(positions)}]
		if !ok || (rule.odds <= 0 && rule.freeSpins <= 0) {
			continue
		}
		win := ScatterWin{
			PayRuleIndex: rule.index,
			Payout:       rule.odds * totalBet / e.payBet,
			Positions:    append([]Position(nil), positions...),
		}
		result.wins = append(result.wins, win)
		result.freeSpins += rule.freeSpins
	}
	return result, nil
}
