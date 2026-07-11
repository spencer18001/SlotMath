package evaluator

import (
	"fmt"

	"slotmath/internal/board"
	"slotmath/internal/config"
)

// ScatterWin describes one scatter-symbol payout on one board.
type ScatterWin struct {
	Symbol string
	Count  int
	Payout int64
}

// ScatterResult is the full scatter-pay result for one board.
type ScatterResult struct {
	Wins     []ScatterWin
	TotalWin int64
}

// ScatterEvaluator evaluates scatter pays by counting configured scatter symbols anywhere on the board.
type ScatterEvaluator struct {
	paytable map[payKey]int64
	symbols  map[string]bool
	bet      int64
}

// NewScatterEvaluator creates an evaluator for scatter-symbol count pays.
func NewScatterEvaluator(scatterSymbols []string, paytable config.Paytable, bet int64) (*ScatterEvaluator, error) {
	if bet <= 0 {
		return nil, fmt.Errorf("bet must be greater than zero")
	}

	symbols := make(map[string]bool)
	for _, symbol := range scatterSymbols {
		symbols[symbol] = true
	}

	scatterPays := make(map[payKey]int64)
	for _, pay := range paytable.Scatter {
		scatterPays[payKey{symbol: pay.Symbol, count: pay.Count}] = pay.Payout
	}

	return &ScatterEvaluator{
		paytable: scatterPays,
		symbols:  symbols,
		bet:      bet,
	}, nil
}

// Evaluate returns scatter pays for exact symbol counts on one board.
func (e *ScatterEvaluator) Evaluate(b board.Board) ScatterResult {
	counts := make(map[string]int)
	for _, reel := range b {
		for _, symbol := range reel {
			if e.symbols[symbol] {
				counts[symbol]++
			}
		}
	}

	var result ScatterResult
	for symbol, count := range counts {
		payout := e.paytable[payKey{symbol: symbol, count: count}]
		if payout <= 0 {
			continue
		}
		win := ScatterWin{
			Symbol: symbol,
			Count:  count,
			Payout: payout * e.bet,
		}
		result.Wins = append(result.Wins, win)
		result.TotalWin += win.Payout
	}
	return result
}
