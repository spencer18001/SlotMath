package app

import (
	"math/rand"
	"time"

	"slotmath/internal/board"
	"slotmath/internal/evaluator"
	"slotmath/internal/probe"
)

// LineGame owns the line-game simulation flow.
type LineGame struct {
	data *GameData
}

// SimulationSummary is the high-level result returned by a simulation run.
type SimulationSummary struct {
	GameID        string
	GamePath      string
	Spins         int
	Seed          int64
	ReelCount     int
	Paylines      int
	LinePays      int
	TotalBet      int64
	TotalWin      int64
	HitCount      int
	FirstStops    []int
	FirstBoard    board.Board
	FirstLineWins []evaluator.LineWin
	FirstWin      int64
	Status        string
}

// NewLineGame wires a loaded game definition into a line-game simulator.
func NewLineGame(data *GameData) *LineGame {
	return &LineGame{data: data}
}

// RunSims runs the line-game simulation.
func (g *LineGame) RunSims(spins int, seed int64) (*SimulationSummary, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}

	rng := rand.New(rand.NewSource(actualSeed))
	generator, err := board.NewGenerator(g.data.Reels, g.data.Config.NumRows)
	if err != nil {
		return nil, err
	}
	lineEvaluator, err := evaluator.NewLineEvaluator(
		g.data.Paylines,
		g.data.Paytable,
		g.data.Config.WildSymbols,
		g.data.Config.Bet,
	)
	if err != nil {
		return nil, err
	}

	summary := &SimulationSummary{
		GameID:    g.data.Config.GameID,
		GamePath:  g.data.Path,
		Spins:     spins,
		Seed:      seed,
		ReelCount: len(g.data.Reels),
		Paylines:  len(g.data.Paylines),
		LinePays:  len(g.data.Paytable.Line),
		TotalBet:  int64(spins) * g.data.Config.Bet,
		Status:    "generated boards and evaluated line pays",
	}

	for spin := 0; spin < spins; spin++ {
		spinResult := generator.Draw(rng)
		lineResult := lineEvaluator.Evaluate(spinResult.Board)

		if spin == 0 {
			summary.FirstStops = spinResult.Stops
			summary.FirstBoard = spinResult.Board
			summary.FirstLineWins = lineResult.Wins
			summary.FirstWin = lineResult.TotalWin
		}
		if lineResult.TotalWin > 0 {
			summary.HitCount++
		}
		summary.TotalWin += lineResult.TotalWin
	}

	return summary, nil
}

// RunLinePayRuleProbe estimates the appearance probability of paytable.line[ruleID] on payline 0.
func (g *LineGame) RunLinePayRuleProbe(spins int, seed int64, ruleID int) (*probe.LinePayRuleResult, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}

	rng := rand.New(rand.NewSource(actualSeed))
	generator, err := board.NewGenerator(g.data.Reels, g.data.Config.NumRows)
	if err != nil {
		return nil, err
	}
	lineProbe, err := probe.NewLinePayRuleProbe(
		g.data.Paylines,
		g.data.Paytable,
		g.data.Config.WildSymbols,
		ruleID,
	)
	if err != nil {
		return nil, err
	}

	var hits int64
	for spin := 0; spin < spins; spin++ {
		spinResult := generator.Draw(rng)
		if lineProbe.Observe(spinResult.Board) {
			hits++
		}
	}

	result := lineProbe.Result(spins, hits)
	return &result, nil
}
