package spin

import "fmt"

type scatterResult struct {
	wins      []ScatterWin
	freeSpins int
}

type scatterEvaluator struct {
	paytable map[payKey]payRule
	symbols  map[string]bool
}

func newScatterEvaluator(scatterSymbols []string, paytable Paytable) *scatterEvaluator {
	symbols := make(map[string]bool)
	for _, symbol := range scatterSymbols {
		symbols[symbol] = true
	}
	scatterPays := make(map[payKey]payRule)
	for index, pay := range paytable.Scatter {
		scatterPays[payKey{symbol: pay.Symbol, count: pay.Count}] = payRule{index: index, odds: pay.Odds, freeSpins: pay.FreeSpins}
	}
	return &scatterEvaluator{paytable: scatterPays, symbols: symbols}
}

func (e *scatterEvaluator) evaluate(b Board, totalBet int64) (scatterResult, error) {
	if totalBet <= 0 {
		return scatterResult{}, fmt.Errorf("total bet must be greater than zero")
	}
	counts := make(map[string]int)
	for _, reel := range b {
		for _, symbol := range reel {
			if e.symbols[symbol] {
				counts[symbol]++
			}
		}
	}
	var result scatterResult
	for symbol, count := range counts {
		rule, ok := e.paytable[payKey{symbol: symbol, count: count}]
		if !ok || (rule.odds <= 0 && rule.freeSpins <= 0) {
			continue
		}
		win := ScatterWin{PayRuleIndex: rule.index, Payout: rule.odds * totalBet}
		result.wins = append(result.wins, win)
		result.freeSpins += rule.freeSpins
	}
	return result, nil
}
