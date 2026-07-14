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
		count, ways := e.countWays(pay.Symbol, b)
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
		})
	}
	return result, nil
}

func (e *wayEvaluator) countWays(symbol string, b Board) (int, int64) {
	ways := int64(1)
	for reelIndex, reel := range b {
		hits := e.countReelHits(symbol, reel)
		if hits == 0 {
			return reelIndex, ways
		}
		ways *= int64(hits)
	}
	return len(b), ways
}

func (e *wayEvaluator) countReelHits(symbol string, visibleReel []string) int {
	hits := 0
	for _, visible := range visibleReel {
		if visible == symbol || e.wilds[visible] {
			hits++
		}
	}
	return hits
}
