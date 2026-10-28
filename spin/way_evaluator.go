package spin

type WayEvaluator struct {
	pays      []PaytableEntry
	symbols   []string
	symbolSet map[string]bool
	wilds     map[string]bool
	payBet    int64
}

type WayResult struct {
	wins []WayWin
}

func newWayEvaluator(paytable Paytable, wildSymbols []string, payBet int64) *WayEvaluator {
	var symbols []string
	symbolSet := make(map[string]bool)
	for _, pay := range paytable.Way {
		if symbolSet[pay.Symbol] {
			continue
		}

		symbolSet[pay.Symbol] = true
		symbols = append(symbols, pay.Symbol)
	}

	wilds := make(map[string]bool)
	for _, symbol := range wildSymbols {
		wilds[symbol] = true
	}
	return &WayEvaluator{
		pays:      paytable.Way,
		symbols:   symbols,
		symbolSet: symbolSet,
		wilds:     wilds,
		payBet:    payBet,
	}
}

func (e *WayEvaluator) evaluate(b Board, totalBet int64) (WayResult, error) {
	if totalBet <= 0 || len(e.pays) == 0 {
		return WayResult{}, nil
	}

	var result WayResult
	for _, symbol := range e.symbols {
		count, ways, positions := e.countWays(symbol, b)
		if count <= 0 || ways <= 0 {
			continue
		}

		matchedPay, ok := e.bestWayPay(symbol, count)
		if !ok || matchedPay.Odds <= 0 {
			continue
		}

		if matchedPay.Count < count {
			ways, positions = wayMatchForCount(matchedPay.Count, positions)
		}
		result.wins = append(result.wins, WayWin{
			PaytableEntryIndex: matchedPay.Index,
			Count:              matchedPay.Count,
			Ways:               ways,
			Payout:             matchedPay.Odds * ways * totalBet / e.payBet,
			Positions:          positions,
		})
	}
	return result, nil
}

func (e *WayEvaluator) countWays(symbol string, b Board) (int, int64, []Position) {
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

func (e *WayEvaluator) reelHitPositions(symbol string, reelIndex int, visibleReel []string) []Position {
	var positions []Position
	for rowIndex, visible := range visibleReel {
		if visible == symbol || e.wilds[visible] {
			positions = append(positions, Position{Reel: reelIndex, Row: rowIndex})
		}
	}
	return positions
}

func (e *WayEvaluator) bestWayPay(symbol string, count int) (PaytableEntry, bool) {
	var best PaytableEntry
	found := false
	for _, pay := range e.pays {
		if pay.Symbol != symbol || pay.Count > count {
			continue
		}

		if !found || pay.Count > best.Count {
			best = pay
			found = true
		}
	}
	return best, found
}

func wayMatchForCount(count int, positions []Position) (int64, []Position) {
	ways := int64(1)
	end := 0
	for reelIndex := 0; reelIndex < count; reelIndex++ {
		start := end
		for end < len(positions) && positions[end].Reel == reelIndex {
			end++
		}
		ways *= int64(end - start)
	}
	return ways, positions[:end]
}
