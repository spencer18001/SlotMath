package simulator

import (
	"fmt"

	"slotmath/flow"
	"slotmath/spin"
)

type Simulator struct{ flow *flow.Flow }

type Request struct {
	Spins int
	Bet   int64
}

type PayHitSummary struct {
	Kind                string
	Symbol              string
	Count               int
	Odds                int64
	ExpectedProbability float64
	Hits                int64
}

type ModeSummary struct {
	Mode            spin.Mode
	Spins           int
	TotalLineWin    int64
	TotalScatterWin int64
	TotalWin        int64
	HitCount        int
	PayHits         []PayHitSummary
}

type Summary struct {
	Spins           int
	GeneratedSpins  int
	FreeSpins       int
	Bet             spin.Bet
	TotalBet        int64
	TotalLineWin    int64
	TotalScatterWin int64
	TotalWin        int64
	HitCount        int
	Modes           []ModeSummary
	First           *spin.Result
	Status          string
}

func New(slotFlow *flow.Flow) *Simulator { return &Simulator{flow: slotFlow} }

func (s *Simulator) Run(request Request) (*Summary, error) {
	if request.Spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}
	engine := s.flow.Engine()
	bet, err := engine.ResolveBet(request.Bet)
	if err != nil {
		return nil, err
	}
	paytable := engine.Paytable()
	summary := &Summary{
		Spins:    request.Spins,
		Bet:      bet,
		TotalBet: int64(request.Spins) * bet.Total,
		Modes:    initialModeSummaries(engine.Modes(), paytable),
		Status:   "generated boards and evaluated active line/scatter pays",
	}

	for round := 0; round < request.Spins; round++ {
		state := flow.State{Bet: bet.Total}
		for !state.Completed {
			step, err := s.flow.Next(state)
			if err != nil {
				return nil, err
			}
			state = step.State
			observe(summary, len(paytable.Line), step.Spin)
		}
	}
	return summary, nil
}

func observe(summary *Summary, linePayCount int, result spin.Result) {
	summary.GeneratedSpins++
	if result.Mode == spin.ModeFree {
		summary.FreeSpins++
	}
	if summary.First == nil {
		first := result
		summary.First = &first
	}
	if result.TotalWin > 0 {
		summary.HitCount++
	}
	modeSummary := modeSummaryFor(summary, result.Mode)
	modeSummary.Spins++
	if result.TotalWin > 0 {
		modeSummary.HitCount++
	}
	observePayHits(modeSummary.PayHits, linePayCount, result)
	modeSummary.TotalLineWin += result.TotalLineWin
	modeSummary.TotalScatterWin += result.TotalScatterWin
	modeSummary.TotalWin += result.TotalWin
	summary.TotalLineWin += result.TotalLineWin
	summary.TotalScatterWin += result.TotalScatterWin
	summary.TotalWin += result.TotalWin
}

func observePayHits(payHits []PayHitSummary, linePayCount int, result spin.Result) {
	for _, win := range result.LineWins {
		if win.LineIndex == 0 && win.PayRuleIndex >= 0 && win.PayRuleIndex < linePayCount {
			payHits[win.PayRuleIndex].Hits++
		}
	}
	for _, win := range result.ScatterWins {
		index := linePayCount + win.PayRuleIndex
		if win.PayRuleIndex >= 0 && index < len(payHits) {
			payHits[index].Hits++
		}
	}
}

func modeSummaryFor(summary *Summary, mode spin.Mode) *ModeSummary {
	for index := range summary.Modes {
		if summary.Modes[index].Mode == mode {
			return &summary.Modes[index]
		}
	}
	panic(fmt.Sprintf("simulator observed unconfigured mode %q", mode))
}

func initialModeSummaries(modes []spin.Mode, paytable spin.Paytable) []ModeSummary {
	summaries := make([]ModeSummary, 0, len(modes))
	for _, mode := range modes {
		summaries = append(summaries, ModeSummary{
			Mode: mode, PayHits: initialPayHitSummaries(mode, paytable),
		})
	}
	return summaries
}

func initialPayHitSummaries(mode spin.Mode, paytable spin.Paytable) []PayHitSummary {
	var summaries []PayHitSummary
	appendPay := func(kind string, pay spin.PayEntry) {
		summaries = append(summaries, PayHitSummary{
			Kind: kind, Symbol: pay.Symbol, Count: pay.Count, Odds: pay.Odds,
			ExpectedProbability: pay.ExpectedProbabilityFor(mode),
		})
	}
	for _, pay := range paytable.Line {
		appendPay("line", pay)
	}
	for _, pay := range paytable.Scatter {
		appendPay("scatter", pay)
	}
	return summaries
}
