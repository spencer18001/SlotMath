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

// LineMatch is the left-to-right match state for one payline.
type LineMatch struct {
	BaseSymbol       string
	BaseCount        int
	LeadingWildCount int
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
	paylines      [][]int
	paytable      map[payKey]int64
	wilds         map[string]bool
	wildPaySymbol string
	bet           int64
}

// NewLineEvaluator creates an evaluator from paylines and line pay rules.
func NewLineEvaluator(paylines [][]int, paytable config.Paytable, wildSymbols []string, bet int64) (*LineEvaluator, error) {
	if bet <= 0 {
		return nil, fmt.Errorf("bet must be greater than zero")
	}

	linePays := make(map[payKey]int64)
	for _, pay := range paytable.Line {
		linePays[payKey{symbol: pay.Symbol, count: pay.Count}] = pay.Payout
	}

	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}

	wildPaySymbol := ""
	if len(wildSymbols) > 0 {
		wildPaySymbol = wildSymbols[0]
	}

	return &LineEvaluator{
		paylines:      paylines,
		paytable:      linePays,
		wilds:         wilds,
		wildPaySymbol: wildPaySymbol,
		bet:           bet,
	}, nil
}

// Evaluate returns all winning line pays for one board.
func (e *LineEvaluator) Evaluate(b board.Board) LineResult {
	var result LineResult
	for lineIndex, payline := range e.paylines {
		win, ok := e.EvaluateLine(lineIndex, payline, b)
		if !ok {
			continue
		}
		result.Wins = append(result.Wins, win)
		result.TotalWin += win.Payout
	}
	return result
}

// EvaluateLine returns the best payable win for one payline on one board.
func (e *LineEvaluator) EvaluateLine(lineIndex int, payline []int, b board.Board) (LineWin, bool) {
	symbolsOnLine := symbolsForPayline(b, payline)
	match := analyzeLine(symbolsOnLine, e.wilds)

	basePayout := e.lookup(match.BaseSymbol, match.BaseCount)
	wildPayout := e.lookup(e.wildPaySymbol, match.LeadingWildCount)

	if basePayout <= 0 && wildPayout <= 0 {
		return LineWin{}, false
	}

	if wildPayout > basePayout {
		return e.buildLineWin(lineIndex, payline, symbolsOnLine, e.wildPaySymbol, match.LeadingWildCount, wildPayout), true
	}
	return e.buildLineWin(lineIndex, payline, symbolsOnLine, match.BaseSymbol, match.BaseCount, basePayout), true
}

// analyzeLine scans one payline from left to right, following Stake-style line rules.
func analyzeLine(symbolsOnLine []string, wilds map[string]bool) LineMatch {
	baseSymbol := ""
	baseMatches := 0
	leadingWildCount := 0

	for _, symbol := range symbolsOnLine {
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
	return LineMatch{
		BaseSymbol:       baseSymbol,
		BaseCount:        baseCount,
		LeadingWildCount: leadingWildCount,
	}
}

func (e *LineEvaluator) lookup(symbol string, count int) int64 {
	if symbol == "" || count <= 0 {
		return 0
	}
	return e.paytable[payKey{symbol: symbol, count: count}]
}

func (e *LineEvaluator) buildLineWin(lineIndex int, payline []int, symbolsOnLine []string, symbol string, count int, payout int64) LineWin {
	return LineWin{
		LineIndex: lineIndex,
		Payline:   cloneInts(payline),
		Symbols:   cloneStrings(symbolsOnLine),
		Symbol:    symbol,
		Count:     count,
		Payout:    payout * e.bet,
	}
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
