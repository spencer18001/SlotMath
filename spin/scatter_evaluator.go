package spin

import "fmt"

type scatterResult struct {
	wins      []ScatterWin
	freeSpins int
}

type scatterEvaluator struct {
	pays    []PaytableEntry
	symbols map[string]bool
	payBet  int64
}

func newScatterEvaluator(scatterSymbols []string, paytable Paytable, payBet int64) *scatterEvaluator {
	if payBet <= 0 {
		payBet = 1
	}
	symbols := make(map[string]bool)
	for _, symbol := range scatterSymbols {
		symbols[symbol] = true
	}
	return &scatterEvaluator{pays: paytable.Scatter, symbols: symbols, payBet: payBet}
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
		pay, ok := findPaytableEntry(e.pays, symbol, len(positions))
		if !ok || (pay.Odds <= 0 && pay.FreeSpins <= 0) {
			continue
		}
		win := ScatterWin{
			PaytableEntryIndex: pay.Index,
			Payout:             pay.Odds * totalBet / e.payBet,
			Positions:          append([]Position(nil), positions...),
		}
		result.wins = append(result.wins, win)
		result.freeSpins += pay.FreeSpins
	}
	return result, nil
}
