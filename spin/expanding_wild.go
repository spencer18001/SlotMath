package spin

type ExpandingWildResult struct {
	Reel   int    // 展開的軸
	Symbol string // 展開使用的 Wild
}

func applyExpandingWilds(board Board, wilds map[string]bool) []ExpandingWildResult {
	var results []ExpandingWildResult
	for reelIndex, reel := range board {
		wild := ""
		for _, symbol := range reel {
			if wilds[symbol] {
				wild = symbol
				break
			}
		}
		if wild == "" {
			continue
		}

		for row := range reel {
			reel[row] = wild
		}
		results = append(results, ExpandingWildResult{Reel: reelIndex, Symbol: wild})
	}
	return results
}
