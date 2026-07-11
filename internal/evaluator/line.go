package evaluator

import (
	"fmt"

	"slotmath/internal/board"
	"slotmath/internal/config"
)

type payKey struct {
	symbol string
	count  int
}

// LineWin describes one winning payline on one board.
type LineWin struct {
	LineIndex int
	Payline   []int
	Symbols   []string
	Symbol    string
	Count     int
	Payout    int64
}

// LineResult is the full line-pay result for one board.
type LineResult struct {
	Wins     []LineWin
	TotalWin int64
}

// LineEvaluator evaluates left-to-right line pays.
type LineEvaluator struct {
	paylines [][]int
	paytable map[payKey]int64
	symbols  []string
	wilds    map[string]bool
	bet      int64
}

// NewLineEvaluator creates an evaluator from paylines and line pay rules.
func NewLineEvaluator(paylines [][]int, paytable config.Paytable, wildSymbols []string, bet int64) (*LineEvaluator, error) {
	if bet <= 0 {
		return nil, fmt.Errorf("bet must be greater than zero")
	}

	linePays := make(map[payKey]int64)
	seenSymbols := make(map[string]bool)
	var symbols []string
	for _, pay := range paytable.Line {
		key := payKey{symbol: pay.Symbol, count: pay.Count}
		linePays[key] = pay.Payout
		if !seenSymbols[pay.Symbol] {
			seenSymbols[pay.Symbol] = true
			symbols = append(symbols, pay.Symbol)
		}
	}

	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}

	return &LineEvaluator{
		paylines: paylines,
		paytable: linePays,
		symbols:  symbols,
		wilds:    wilds,
		bet:      bet,
	}, nil
}

// Evaluate returns all winning line pays for one board.
func (e *LineEvaluator) Evaluate(b board.Board) LineResult {
	var result LineResult
	for lineIndex, payline := range e.paylines {
		symbolsOnLine := symbolsForPayline(b, payline)
		win, ok := e.bestWinForLine(lineIndex, payline, symbolsOnLine)
		if !ok {
			continue
		}
		result.Wins = append(result.Wins, win)
		result.TotalWin += win.Payout
	}
	return result
}

func (e *LineEvaluator) bestWinForLine(lineIndex int, payline []int, symbolsOnLine []string) (LineWin, bool) {
	var best LineWin
	found := false

	for _, target := range e.symbols {
		count := e.matchCount(symbolsOnLine, target)
		payout, ok := e.paytable[payKey{symbol: target, count: count}]
		if !ok || payout <= 0 {
			continue
		}

		linePayout := payout * e.bet
		if !found || linePayout > best.Payout {
			best = LineWin{
				LineIndex: lineIndex,
				Payline:   cloneInts(payline),
				Symbols:   cloneStrings(symbolsOnLine),
				Symbol:    target,
				Count:     count,
				Payout:    linePayout,
			}
			found = true
		}
	}

	return best, found
}

func (e *LineEvaluator) matchCount(symbolsOnLine []string, target string) int {
	count := 0
	for _, symbol := range symbolsOnLine {
		if symbol != target && !e.wilds[symbol] {
			break
		}
		count++
	}
	return count
}

func symbolsForPayline(b board.Board, payline []int) []string {
	symbols := make([]string, len(payline))
	for reel, row := range payline {
		symbols[reel] = b[reel][row]
	}
	return symbols
}

func cloneInts(values []int) []int {
	clone := make([]int, len(values))
	copy(clone, values)
	return clone
}

func cloneStrings(values []string) []string {
	clone := make([]string, len(values))
	copy(clone, values)
	return clone
}
