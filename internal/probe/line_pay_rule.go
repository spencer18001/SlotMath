package probe

import (
	"fmt"

	"slotmath/internal/board"
	"slotmath/internal/config"
	"slotmath/internal/evaluator"
)

// LinePayRuleResult is the probability result for one line pay rule on payline 0.
type LinePayRuleResult struct {
	RuleID      int
	Rule        config.PayEntry
	Payline     []int
	Spins       int
	Hits        int64
	Probability float64
}

// LinePayRuleProbe checks whether payline 0 resolves to one configured line pay rule.
type LinePayRuleProbe struct {
	ruleID    int
	rule      config.PayEntry
	payline   []int
	evaluator *evaluator.LineEvaluator
}

// NewLinePayRuleProbe creates a probe for paytable.line[ruleID] on payline 0.
func NewLinePayRuleProbe(paylines [][]int, paytable config.Paytable, wildSymbols []string, ruleID int) (*LinePayRuleProbe, error) {
	if len(paylines) == 0 {
		return nil, fmt.Errorf("payline 0 is required for line pay rule probe")
	}
	if ruleID < 0 || ruleID >= len(paytable.Line) {
		return nil, fmt.Errorf("line pay rule id %d is outside 0..%d", ruleID, len(paytable.Line)-1)
	}

	payline := make([]int, len(paylines[0]))
	copy(payline, paylines[0])

	lineEvaluator, err := evaluator.NewLineEvaluator(paylines[:1], paytable, wildSymbols, 1)
	if err != nil {
		return nil, err
	}

	return &LinePayRuleProbe{
		ruleID:    ruleID,
		rule:      paytable.Line[ruleID],
		payline:   payline,
		evaluator: lineEvaluator,
	}, nil
}

// Observe returns true when the board resolves to the target line pay rule.
func (p *LinePayRuleProbe) Observe(b board.Board) bool {
	win, ok := p.evaluator.EvaluateLine(0, p.payline, b)
	return ok && win.Symbol == p.rule.Symbol && win.Count == p.rule.Count
}

// Result creates a probability result from a finished hit count.
func (p *LinePayRuleProbe) Result(spins int, hits int64) LinePayRuleResult {
	probability := 0.0
	if spins > 0 {
		probability = float64(hits) / float64(spins)
	}
	return LinePayRuleResult{
		RuleID:      p.ruleID,
		Rule:        p.rule,
		Payline:     cloneInts(p.payline),
		Spins:       spins,
		Hits:        hits,
		Probability: probability,
	}
}

func cloneInts(values []int) []int {
	clone := make([]int, len(values))
	copy(clone, values)
	return clone
}
