package spin

type wayResult struct {
	wins []WayWin
}

type wayEvaluator struct {
	paytable map[payKey]payRule
	pays     []PayEntry
	wilds    map[string]bool
	payBet   int64
}

func newWayEvaluator(paytable Paytable, wildSymbols []string, payBet int64) *wayEvaluator {
	wayPays := make(map[payKey]payRule)
	for index, pay := range paytable.Way {
		wayPays[payKey{symbol: pay.Symbol, count: pay.Count}] = payRule{index: index, odds: pay.Odds}
	}
	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}
	return &wayEvaluator{paytable: wayPays, pays: paytable.Way, wilds: wilds, payBet: payBet}
}

func (e *wayEvaluator) evaluate(b Board, totalBet int64) (wayResult, error) {
	if totalBet <= 0 || len(e.pays) == 0 {
		return wayResult{}, nil
	}
	var result wayResult
	seen := make(map[string]bool)
	for _, pay := range e.pays {
		if seen[pay.Symbol] {
			continue
		}
		seen[pay.Symbol] = true
		count, ways, positions := e.countWays(pay.Symbol, b)
		if count <= 0 || ways <= 0 {
			continue
		}
		rule, ok := e.paytable[payKey{symbol: pay.Symbol, count: count}]
		if !ok || rule.odds <= 0 {
			continue
		}
		result.wins = append(result.wins, WayWin{
			PayRuleIndex: rule.index,
			Count:        count,
			Ways:         ways,
			Payout:       rule.odds * ways * totalBet / e.payBet,
			Positions:    positions,
		})
	}
	return result, nil
}

func (e *wayEvaluator) countWays(symbol string, b Board) (int, int64, []Position) {
	ways := int64(1)
	var positions []Position
	for reelIndex, reel := range b {
		reelPositions := e.reelHitPositions(symbol, reelIndex, reel)
		if len(reelPositions) == 0 {
			return reelIndex, ways, positions
		}
		ways *= int64(len(reelPositions))
		positions = append(positions, reelPositions...)
	}
	return len(b), ways, positions
}

func (e *wayEvaluator) reelHitPositions(symbol string, reelIndex int, visibleReel []string) []Position {
	var positions []Position
	for rowIndex, visible := range visibleReel {
		if visible == symbol || e.wilds[visible] {
			positions = append(positions, Position{Reel: reelIndex, Row: rowIndex})
		}
	}
	return positions
}
