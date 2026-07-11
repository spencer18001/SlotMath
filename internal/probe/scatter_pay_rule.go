package probe

import (
	"fmt"

	"slotmath/internal/board"
	"slotmath/internal/config"
	"slotmath/internal/evaluator"
)

// ScatterPayRuleResult is the probability result for one scatter pay rule.
type ScatterPayRuleResult struct {
	RuleID      int
	Rule        config.PayEntry
	Spins       int
	Hits        int64
	Probability float64
}

// ScatterPayRuleProbe checks whether a board resolves to one configured scatter pay rule.
type ScatterPayRuleProbe struct {
	ruleID    int
	rule      config.PayEntry
	evaluator *evaluator.ScatterEvaluator
}

// NewScatterPayRuleProbe creates a probe for paytable.scatter[ruleID].
func NewScatterPayRuleProbe(scatterSymbols []string, paytable config.Paytable, ruleID int) (*ScatterPayRuleProbe, error) {
	if ruleID < 0 || ruleID >= len(paytable.Scatter) {
		return nil, fmt.Errorf("scatter pay rule id %d is outside 0..%d", ruleID, len(paytable.Scatter)-1)
	}

	scatterEvaluator, err := evaluator.NewScatterEvaluator(scatterSymbols, paytable, 1)
	if err != nil {
		return nil, err
	}

	return &ScatterPayRuleProbe{
		ruleID:    ruleID,
		rule:      paytable.Scatter[ruleID],
		evaluator: scatterEvaluator,
	}, nil
}

// Observe returns true when the board resolves to the target scatter pay rule.
func (p *ScatterPayRuleProbe) Observe(b board.Board) bool {
	result := p.evaluator.Evaluate(b)
	for _, win := range result.Wins {
		if win.Symbol == p.rule.Symbol && win.Count == p.rule.Count {
			return true
		}
	}
	return false
}

// Result creates a probability result from a finished hit count.
func (p *ScatterPayRuleProbe) Result(spins int, hits int64) ScatterPayRuleResult {
	probability := 0.0
	if spins > 0 {
		probability = float64(hits) / float64(spins)
	}
	return ScatterPayRuleResult{
		RuleID:      p.ruleID,
		Rule:        p.rule,
		Spins:       spins,
		Hits:        hits,
		Probability: probability,
	}
}
