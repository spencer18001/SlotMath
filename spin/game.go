package spin

import (
	"fmt"
	"math/rand"
	"time"
)

type Game struct {
	info             Info
	rng              *rand.Rand
	generator        *generator
	lineEvaluator    *lineEvaluator
	scatterEvaluator *scatterEvaluator
	paylines         [][]int
	paytable         Paytable
}

func newGame(data loadedGame, seed int64) (*Game, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}
	generator, err := newGenerator(data.reels, data.config.NumRows)
	if err != nil {
		return nil, err
	}
	return &Game{
		info: Info{
			GameID: data.config.GameID, Path: data.path, Seed: seed,
			BetPerLine: data.config.BetPerLine, ReelCount: len(data.reels), PaylineCount: len(data.paylines),
		},
		rng: rand.New(rand.NewSource(actualSeed)), generator: generator,
		lineEvaluator:    newLineEvaluator(data.paylines, data.paytable, data.config.WildSymbols),
		scatterEvaluator: newScatterEvaluator(data.config.ScatterSymbols, data.paytable),
		paylines:         clonePaylines(data.paylines), paytable: clonePaytable(data.paytable),
	}, nil
}

func (g *Game) ResolveBet(totalBet int64) (Bet, error) {
	if totalBet <= 0 {
		return Bet{}, fmt.Errorf("bet must be greater than zero")
	}
	if totalBet%g.info.BetPerLine != 0 {
		return Bet{}, fmt.Errorf("bet %d must be a multiple of bet per line %d", totalBet, g.info.BetPerLine)
	}
	activeLines := int(totalBet / g.info.BetPerLine)
	if activeLines > g.info.PaylineCount {
		return Bet{}, fmt.Errorf("bet %d activates %d lines, maximum is %d", totalBet, activeLines, g.info.PaylineCount)
	}
	return Bet{Total: totalBet, PerLine: g.info.BetPerLine, ActiveLines: activeLines}, nil
}

func (g *Game) DefaultBet() Bet {
	return Bet{Total: g.info.BetPerLine * int64(g.info.PaylineCount), PerLine: g.info.BetPerLine, ActiveLines: g.info.PaylineCount}
}

func (g *Game) Spin(request Request) (Result, error) {
	bet, err := g.ResolveBet(request.Bet)
	if err != nil {
		return Result{}, err
	}
	drawn := g.generator.draw(g.rng)
	lineResult, err := g.lineEvaluator.evaluate(drawn.Board, bet.ActiveLines, bet.PerLine)
	if err != nil {
		return Result{}, err
	}
	scatterResult, err := g.scatterEvaluator.evaluate(drawn.Board, bet.Total)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Stops: drawn.Stops, Board: drawn.Board,
		LineWins: lineResult.wins, ScatterWins: scatterResult.wins,
		TotalLineWin: lineResult.totalWin, TotalScatterWin: scatterResult.totalWin,
		TotalWin: lineResult.totalWin + scatterResult.totalWin,
	}, nil
}

func (g *Game) SpinLine(lineIndex int) (Result, error) {
	if lineIndex < 0 || lineIndex >= len(g.paylines) {
		return Result{}, fmt.Errorf("line index %d is outside 0..%d", lineIndex, len(g.paylines)-1)
	}
	drawn := g.generator.draw(g.rng)
	result := Result{Stops: drawn.Stops, Board: drawn.Board}
	win, ok := g.lineEvaluator.evaluateLine(lineIndex, g.paylines[lineIndex], drawn.Board, g.info.BetPerLine)
	if !ok {
		return result, nil
	}
	result.LineWins = []LineWin{win}
	result.TotalLineWin = win.Payout
	result.TotalWin = win.Payout
	return result, nil
}

func (g *Game) SpinScatter() (Result, error) {
	drawn := g.generator.draw(g.rng)
	scatterResult, err := g.scatterEvaluator.evaluate(drawn.Board, g.DefaultBet().Total)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Stops: drawn.Stops, Board: drawn.Board, ScatterWins: scatterResult.wins,
		TotalScatterWin: scatterResult.totalWin, TotalWin: scatterResult.totalWin,
	}, nil
}

func (g *Game) Info() Info         { return g.info }
func (g *Game) Paytable() Paytable { return clonePaytable(g.paytable) }
func (g *Game) Payline(index int) ([]int, bool) {
	if index < 0 || index >= len(g.paylines) {
		return nil, false
	}
	return cloneInts(g.paylines[index]), true
}

func clonePaytable(value Paytable) Paytable {
	return Paytable{Line: append([]PayEntry(nil), value.Line...), Scatter: append([]PayEntry(nil), value.Scatter...)}
}
func clonePaylines(values [][]int) [][]int {
	result := make([][]int, len(values))
	for index, value := range values {
		result[index] = cloneInts(value)
	}
	return result
}
func cloneInts(values []int) []int { return append([]int(nil), values...) }
