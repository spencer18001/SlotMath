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

// PayHitSummary counts how often one configured pay rule appears during simulation.
type PayHitSummary struct {
	Kind   string
	Symbol string
	Count  int
	Payout int64
	Hits   int64
}

type payHitKey struct {
	kind   string
	symbol string
	count  int
}

// SimulationSummary is the high-level result returned by a simulation run.
type SimulationSummary struct {
	GameID           string
	GamePath         string
	Spins            int
	Seed             int64
	ReelCount        int
	Paylines         int
	LinePays         int
	ScatterPays      int
	TotalBet         int64
	TotalLineWin     int64
	TotalScatterWin  int64
	TotalWin         int64
	HitCount         int
	PayHits          []PayHitSummary
	FirstStops       []int
	FirstBoard       board.Board
	FirstLineWins    []evaluator.LineWin
	FirstScatterWins []evaluator.ScatterWin
	FirstWin         int64
	Status           string
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
	scatterEvaluator, err := evaluator.NewScatterEvaluator(
		g.data.Config.ScatterSymbols,
		g.data.Paytable,
		g.data.Config.Bet,
	)
	if err != nil {
		return nil, err
	}
	payHits, payHitIndex := g.initialPayHitSummaries()

	summary := &SimulationSummary{
		GameID:      g.data.Config.GameID,
		GamePath:    g.data.Path,
		Spins:       spins,
		Seed:        seed,
		ReelCount:   len(g.data.Reels),
		Paylines:    len(g.data.Paylines),
		LinePays:    len(g.data.Paytable.Line),
		ScatterPays: len(g.data.Paytable.Scatter),
		TotalBet:    int64(spins) * g.data.Config.Bet,
		PayHits:     payHits,
		Status:      "generated boards and evaluated line/scatter pays",
	}

	for spin := 0; spin < spins; spin++ {
		spinResult := generator.Draw(rng)
		lineResult := lineEvaluator.Evaluate(spinResult.Board)
		scatterResult := scatterEvaluator.Evaluate(spinResult.Board)
		spinWin := lineResult.TotalWin + scatterResult.TotalWin

		if spin == 0 {
			summary.FirstStops = spinResult.Stops
			summary.FirstBoard = spinResult.Board
			summary.FirstLineWins = lineResult.Wins
			summary.FirstScatterWins = scatterResult.Wins
			summary.FirstWin = spinWin
		}
		if spinWin > 0 {
			summary.HitCount++
		}
		recordLinePayHits(summary.PayHits, payHitIndex, lineResult.Wins)
		recordScatterPayHits(summary.PayHits, payHitIndex, scatterResult.Wins)
		summary.TotalLineWin += lineResult.TotalWin
		summary.TotalScatterWin += scatterResult.TotalWin
		summary.TotalWin += spinWin
	}

	return summary, nil
}

func (g *LineGame) initialPayHitSummaries() ([]PayHitSummary, map[payHitKey]int) {
	var summaries []PayHitSummary
	index := make(map[payHitKey]int)

	for _, pay := range g.data.Paytable.Line {
		key := payHitKey{kind: "line", symbol: pay.Symbol, count: pay.Count}
		index[key] = len(summaries)
		summaries = append(summaries, PayHitSummary{
			Kind:   key.kind,
			Symbol: pay.Symbol,
			Count:  pay.Count,
			Payout: pay.Payout,
		})
	}
	for _, pay := range g.data.Paytable.Scatter {
		key := payHitKey{kind: "scatter", symbol: pay.Symbol, count: pay.Count}
		index[key] = len(summaries)
		summaries = append(summaries, PayHitSummary{
			Kind:   key.kind,
			Symbol: pay.Symbol,
			Count:  pay.Count,
			Payout: pay.Payout,
		})
	}

	return summaries, index
}

func recordLinePayHits(summaries []PayHitSummary, index map[payHitKey]int, wins []evaluator.LineWin) {
	for _, win := range wins {
		key := payHitKey{kind: "line", symbol: win.Symbol, count: win.Count}
		if hitIndex, ok := index[key]; ok {
			summaries[hitIndex].Hits++
		}
	}
}

func recordScatterPayHits(summaries []PayHitSummary, index map[payHitKey]int, wins []evaluator.ScatterWin) {
	for _, win := range wins {
		key := payHitKey{kind: "scatter", symbol: win.Symbol, count: win.Count}
		if hitIndex, ok := index[key]; ok {
			summaries[hitIndex].Hits++
		}
	}
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

// RunScatterPayRuleProbe estimates the appearance probability of paytable.scatter[ruleID].
func (g *LineGame) RunScatterPayRuleProbe(spins int, seed int64, ruleID int) (*probe.ScatterPayRuleResult, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}

	rng := rand.New(rand.NewSource(actualSeed))
	generator, err := board.NewGenerator(g.data.Reels, g.data.Config.NumRows)
	if err != nil {
		return nil, err
	}
	scatterProbe, err := probe.NewScatterPayRuleProbe(
		g.data.Config.ScatterSymbols,
		g.data.Paytable,
		ruleID,
	)
	if err != nil {
		return nil, err
	}

	var hits int64
	for spin := 0; spin < spins; spin++ {
		spinResult := generator.Draw(rng)
		if scatterProbe.Observe(spinResult.Board) {
			hits++
		}
	}

	result := scatterProbe.Result(spins, hits)
	return &result, nil
}
