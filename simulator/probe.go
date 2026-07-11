package simulator

import (
	"fmt"

	"slotmath/spin"
)

// LinePayRuleProbeResult is the probability result for one line pay rule on payline 0.
type LinePayRuleProbeResult struct {
	PayRuleIndex int
	Rule         spin.PayEntry
	Payline      []int
	Spins        int
	Hits         int64
	Probability  float64
}

// ScatterPayRuleProbeResult is the probability result for one scatter pay rule.
type ScatterPayRuleProbeResult struct {
	PayRuleIndex int
	Rule         spin.PayEntry
	Spins        int
	Hits         int64
	Probability  float64
}

// RunLinePayRuleProbe estimates the appearance probability of paytable.line[payRuleIndex] on payline 0.
func RunLinePayRuleProbe(game *spin.Game, spins int, payRuleIndex int) (*LinePayRuleProbeResult, error) {
	if spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}
	paytable := game.Paytable()
	if payRuleIndex < 0 || payRuleIndex >= len(paytable.Line) {
		return nil, fmt.Errorf("line pay rule index %d is outside 0..%d", payRuleIndex, len(paytable.Line)-1)
	}
	payline, ok := game.Payline(0)
	if !ok {
		return nil, fmt.Errorf("payline 0 is required for line pay rule probe")
	}
	rule := paytable.Line[payRuleIndex]
	var hits int64
	for index := 0; index < spins; index++ {
		result, err := game.SpinLine(0)
		if err != nil {
			return nil, err
		}
		for _, win := range result.LineWins {
			if win.PayRuleIndex == payRuleIndex {
				hits++
				break
			}
		}
	}
	return &LinePayRuleProbeResult{
		PayRuleIndex: payRuleIndex,
		Rule:         rule,
		Payline:      payline,
		Spins:        spins,
		Hits:         hits,
		Probability:  probability(hits, spins),
	}, nil
}

// RunScatterPayRuleProbe estimates the appearance probability of paytable.scatter[payRuleIndex].
func RunScatterPayRuleProbe(game *spin.Game, spins int, payRuleIndex int) (*ScatterPayRuleProbeResult, error) {
	if spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}
	paytable := game.Paytable()
	if payRuleIndex < 0 || payRuleIndex >= len(paytable.Scatter) {
		return nil, fmt.Errorf("scatter pay rule index %d is outside 0..%d", payRuleIndex, len(paytable.Scatter)-1)
	}
	rule := paytable.Scatter[payRuleIndex]
	var hits int64
	for index := 0; index < spins; index++ {
		result, err := game.SpinScatter()
		if err != nil {
			return nil, err
		}
		for _, win := range result.ScatterWins {
			if win.PayRuleIndex == payRuleIndex {
				hits++
				break
			}
		}
	}
	return &ScatterPayRuleProbeResult{
		PayRuleIndex: payRuleIndex,
		Rule:         rule,
		Spins:        spins,
		Hits:         hits,
		Probability:  probability(hits, spins),
	}, nil
}

func probability(hits int64, spins int) float64 {
	if spins <= 0 {
		return 0
	}
	return float64(hits) / float64(spins)
}
