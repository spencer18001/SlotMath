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

type Summary struct {
	Spins           int
	Bet             spin.Bet
	TotalBet        int64
	TotalLineWin    int64
	TotalScatterWin int64
	TotalWin        int64
	HitCount        int
	PayHits         []PayHitSummary
	First           *spin.Result
	Status          string
}

func New(gameFlow *flow.Flow) *Simulator { return &Simulator{flow: gameFlow} }

func (s *Simulator) Run(request Request) (*Summary, error) {
	if request.Spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}
	game := s.flow.Game()
	bet, err := game.ResolveBet(request.Bet)
	if err != nil {
		return nil, err
	}
	paytable := game.Paytable()
	summary := &Summary{
		Spins:    request.Spins,
		Bet:      bet,
		TotalBet: int64(request.Spins) * bet.Total,
		PayHits:  initialPayHitSummaries(paytable),
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
	if summary.First == nil {
		first := result
		summary.First = &first
	}
	if result.TotalWin > 0 {
		summary.HitCount++
	}
	for _, win := range result.LineWins {
		if win.LineIndex == 0 && win.PayRuleIndex >= 0 && win.PayRuleIndex < linePayCount {
			summary.PayHits[win.PayRuleIndex].Hits++
		}
	}
	for _, win := range result.ScatterWins {
		index := linePayCount + win.PayRuleIndex
		if win.PayRuleIndex >= 0 && index < len(summary.PayHits) {
			summary.PayHits[index].Hits++
		}
	}
	summary.TotalLineWin += result.TotalLineWin
	summary.TotalScatterWin += result.TotalScatterWin
	summary.TotalWin += result.TotalWin
}

func initialPayHitSummaries(paytable spin.Paytable) []PayHitSummary {
	var summaries []PayHitSummary
	appendPay := func(kind string, pay spin.PayEntry) {
		summaries = append(summaries, PayHitSummary{
			Kind: kind, Symbol: pay.Symbol, Count: pay.Count, Odds: pay.Odds,
			ExpectedProbability: pay.ExpectedProbability,
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
