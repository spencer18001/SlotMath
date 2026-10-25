package spin

import "fmt"

type ScatterEvaluator struct {
	pays      []PaytableEntry
	symbols   []string
	symbolSet map[string]bool
}

type ScatterResult struct {
	wins      []ScatterWin
	freeSpins int
}

func newScatterEvaluator(paytable Paytable) *ScatterEvaluator {
	var symbols []string
	symbolSet := make(map[string]bool)
	for _, pay := range paytable.Scatter {
		if symbolSet[pay.Symbol] {
			continue
		}

		symbolSet[pay.Symbol] = true
		symbols = append(symbols, pay.Symbol)
	}
	return &ScatterEvaluator{pays: paytable.Scatter, symbols: symbols, symbolSet: symbolSet}
}

func (e *ScatterEvaluator) evaluate(b Board, totalBet int64) (ScatterResult, error) {
	if totalBet <= 0 {
		return ScatterResult{}, fmt.Errorf("total bet must be greater than zero")
	}

	positionsBySymbol := make(map[string][]Position)
	for reelIndex, reel := range b {
		for rowIndex, symbol := range reel {
			if e.symbolSet[symbol] {
				positionsBySymbol[symbol] = append(positionsBySymbol[symbol], Position{Reel: reelIndex, Row: rowIndex})
			}
		}
	}

	var result ScatterResult
	for _, symbol := range e.symbols {
		positions, ok := positionsBySymbol[symbol]
		if !ok {
			continue
		}

		pay, ok := e.bestScatterPay(symbol, len(positions))
		if !ok || (pay.Odds <= 0 && pay.FreeSpins <= 0) {
			continue
		}

		win := ScatterWin{
			PaytableEntryIndex: pay.Index,
			Payout:             pay.Odds * totalBet,
			Positions:          positions,
		}
		result.wins = append(result.wins, win)
		result.freeSpins += pay.FreeSpins
	}
	return result, nil
}

func (e *ScatterEvaluator) bestScatterPay(symbol string, count int) (PaytableEntry, bool) {
	var best PaytableEntry
	found := false
	for _, pay := range e.pays {
		if pay.Symbol != symbol || pay.Count > count {
			continue
		}

		if !found || pay.Count > best.Count {
			best = pay
			found = true
		}
	}
	return best, found
}
