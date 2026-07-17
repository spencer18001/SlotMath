package simulator

import (
	"fmt"
	"sort"
	"strings"

	"slotmath/spin"
)

type PatternKind string

const (
	PatternLine    PatternKind = "line"
	PatternScatter PatternKind = "scatter"
)

type PatternCondition struct {
	Any     bool
	Negated bool
	Symbols []string
}

type Pattern struct {
	Raw        string
	Kind       PatternKind
	Conditions []PatternCondition
}

type PatternProbeResult struct {
	Pattern     Pattern
	Mode        spin.Mode
	LineIndex   int
	Payline     []int
	Spins       int
	Hits        int64
	Probability float64
}

func ParsePattern(raw string, reelCount int, symbols []string) (Pattern, error) {
	kindText, conditionsText, ok := strings.Cut(strings.TrimSpace(raw), ".")
	if !ok {
		return Pattern{}, fmt.Errorf("pattern must start with line. or scatter.")
	}
	kind := PatternKind(strings.ToLower(kindText))
	if kind != PatternLine && kind != PatternScatter {
		return Pattern{}, fmt.Errorf("unsupported pattern kind %q", kindText)
	}
	tokens := strings.Split(conditionsText, "|")
	if len(tokens) != reelCount {
		return Pattern{}, fmt.Errorf("pattern has %d reel conditions, expected %d", len(tokens), reelCount)
	}
	known := append([]string(nil), symbols...)
	sort.Slice(known, func(i, j int) bool { return len(known[i]) > len(known[j]) })
	pattern := Pattern{Raw: raw, Kind: kind, Conditions: make([]PatternCondition, len(tokens))}
	for index, token := range tokens {
		condition, err := parsePatternCondition(strings.TrimSpace(token), known)
		if err != nil {
			return Pattern{}, fmt.Errorf("reel %d: %w", index, err)
		}
		pattern.Conditions[index] = condition
	}
	return pattern, nil
}

func parsePatternCondition(token string, symbols []string) (PatternCondition, error) {
	if token == "-" {
		return PatternCondition{Any: true}, nil
	}
	condition := PatternCondition{}
	if strings.HasPrefix(token, "!") {
		condition.Negated = true
		token = strings.TrimPrefix(token, "!")
	}
	if token == "" {
		return PatternCondition{}, fmt.Errorf("symbol condition is empty")
	}
	parsed, count := splitKnownSymbols(token, symbols)
	if count == 0 {
		return PatternCondition{}, fmt.Errorf("%q cannot be split into configured symbols", token)
	}
	if count > 1 {
		return PatternCondition{}, fmt.Errorf("%q has an ambiguous symbol split", token)
	}
	condition.Symbols = parsed
	return condition, nil
}

func splitKnownSymbols(value string, symbols []string) ([]string, int) {
	if value == "" {
		return []string{}, 1
	}
	var first []string
	count := 0
	for _, symbol := range symbols {
		if !strings.HasPrefix(value, symbol) {
			continue
		}
		rest, restCount := splitKnownSymbols(strings.TrimPrefix(value, symbol), symbols)
		if restCount == 0 {
			continue
		}
		if count == 0 {
			first = append([]string{symbol}, rest...)
		}
		count += restCount
		if count > 1 {
			return first, 2
		}
	}
	return first, count
}

func RunPatternProbe(engine *spin.Engine, spins int, mode spin.Mode, lineIndex int, rawPattern string) (*PatternProbeResult, error) {
	if spins <= 0 {
		return nil, fmt.Errorf("spins must be greater than zero")
	}
	pattern, err := ParsePattern(rawPattern, engine.Info().ReelCount, engine.Symbols())
	if err != nil {
		return nil, err
	}
	result := &PatternProbeResult{Pattern: pattern, Mode: mode, LineIndex: lineIndex, Spins: spins}
	if pattern.Kind == PatternLine {
		payline, ok := engine.Payline(lineIndex)
		if !ok {
			return nil, fmt.Errorf("line index %d is outside configured paylines", lineIndex)
		}
		result.Payline = payline
	} else if lineIndex != 0 {
		return nil, fmt.Errorf("line index only applies to line patterns")
	}
	bet := engine.DefaultBet().Total
	req := spin.Request{Bet: bet, Mode: mode}
	for index := 0; index < spins; index++ {
		spinResult, err := engine.Spin(req)
		if err != nil {
			return nil, err
		}
		if matchesPattern(pattern, spinResult.InitialBoard, result.Payline) {
			result.Hits++
		}
	}
	if result.Spins > 0 {
		result.Probability = float64(result.Hits) / float64(result.Spins)
	}
	return result, nil
}

func matchesPattern(pattern Pattern, board spin.Board, payline []int) bool {
	for reel, condition := range pattern.Conditions {
		matched := false
		if pattern.Kind == PatternLine {
			matched = condition.matches(board[reel][payline[reel]])
		} else {
			for _, symbol := range board[reel] {
				if condition.matchesPositive(symbol) {
					matched = true
					break
				}
			}
			if condition.Any {
				matched = true
			} else if condition.Negated {
				matched = !matched
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (c PatternCondition) matches(symbol string) bool {
	if c.Any {
		return true
	}
	matched := c.matchesPositive(symbol)
	if c.Negated {
		return !matched
	}
	return matched
}

func (c PatternCondition) matchesPositive(symbol string) bool {
	for _, candidate := range c.Symbols {
		if symbol == candidate {
			return true
		}
	}
	return false
}
