package simulator

import (
	"fmt"

	"slotmath/flow"
	"slotmath/spin"
)

type Simulator struct {
	flow *flow.Flow
}

type RunRequest struct {
	Spins int
	Bet   int64
}

type Summary struct {
	Spins           int
	FreeSpins       int
	CascadeSteps    int
	Bet             spin.Bet
	TotalBet        int64
	TotalLineWin    int64
	TotalWayWin     int64
	TotalScatterWin int64
	TotalWin        int64
	HitCount        int
	CascadeHitCount int
	Modes           []ModeSummary
}

type ModeSummary struct {
	Mode            spin.Mode
	Spins           int
	CascadeSteps    int
	TotalLineWin    int64
	TotalWayWin     int64
	TotalScatterWin int64
	TotalWin        int64
}

func New(slotFlow *flow.Flow) *Simulator { return &Simulator{flow: slotFlow} }

func (s *Simulator) Run(request RunRequest) (*Summary, error) {
	if request.Spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}

	engine := s.flow.Engine()
	bet, err := engine.ResolveBet(request.Bet)
	if err != nil {
		return nil, err
	}
	summary := &Summary{
		Spins:    request.Spins,
		Bet:      bet,
		TotalBet: int64(request.Spins) * bet.Total,
		Modes:    initialModeSummaries(engine.Modes()),
	}

	for round := 0; round < request.Spins; round++ {
		state := flow.State{Bet: bet.Total}
		for !state.Completed {
			stepResult, err := s.flow.Next(state)
			if err != nil {
				return nil, err
			}
			state = stepResult.State

			observe(summary, stepResult.Spin)
		}
	}
	return summary, nil
}

func initialModeSummaries(modes []spin.Mode) []ModeSummary {
	summaries := make([]ModeSummary, 0, len(modes))
	for _, mode := range modes {
		summaries = append(summaries, ModeSummary{Mode: mode})
	}
	return summaries
}

func observe(summary *Summary, spinResult spin.SpinResult) {
	if spinResult.Mode == spin.ModeFree {
		summary.FreeSpins++
	}
	if spinResult.TotalWin > 0 {
		summary.HitCount++
	}
	summary.CascadeSteps += len(spinResult.CascadeSteps)
	for _, step := range spinResult.CascadeSteps {
		if step.TotalWin > 0 {
			summary.CascadeHitCount++
		}
	}
	modeSummary := modeSummaryFor(summary, spinResult.Mode)
	modeSummary.Spins++
	modeSummary.CascadeSteps += len(spinResult.CascadeSteps)
	var lineWin, wayWin, scatterWin int64
	for _, win := range spinResult.LineWins {
		lineWin += win.Payout
	}
	for _, win := range spinResult.WayWins {
		wayWin += win.Payout
	}
	for _, win := range spinResult.ScatterWins {
		scatterWin += win.Payout
	}
	modeSummary.TotalLineWin += lineWin
	modeSummary.TotalWayWin += wayWin
	modeSummary.TotalScatterWin += scatterWin
	modeSummary.TotalWin += spinResult.TotalWin
	summary.TotalLineWin += lineWin
	summary.TotalWayWin += wayWin
	summary.TotalScatterWin += scatterWin
	summary.TotalWin += spinResult.TotalWin
}

func modeSummaryFor(summary *Summary, mode spin.Mode) *ModeSummary {
	for index := range summary.Modes {
		if summary.Modes[index].Mode == mode {
			return &summary.Modes[index]
		}
	}
	panic(fmt.Sprintf("simulator observed unconfigured mode %q", mode))
}
