package spin

import "math/rand"

type megaSymbolFeature struct {
	mode         Mode
	cases        []MegaSymbolCase
	caseWeight   int
	symbols      []MegaSymbolWeight
	symbolWeight int
}

type MegaSymbolResult struct {
	CaseIndex int        // 抽中的 Case 索引
	Symbol    string     // 抽中的 Mega Symbol；不覆蓋時為空
	Positions []Position // 實際被 Mega Symbol 覆蓋的位置
}

func newMegaSymbolFeature(setting *MegaSymbolSetting) *megaSymbolFeature {
	if setting == nil {
		return nil
	}

	mode := setting.Mode
	if mode == "" {
		mode = ModeBase
	}

	feature := &megaSymbolFeature{
		mode:    mode,
		cases:   setting.Cases,
		symbols: setting.Symbols,
	}
	for _, megaCase := range feature.cases {
		feature.caseWeight += megaCase.Weight
	}
	for _, symbol := range feature.symbols {
		feature.symbolWeight += symbol.Weight
	}
	return feature
}

func (f *megaSymbolFeature) apply(rng *rand.Rand, board Board) *MegaSymbolResult {
	caseIndex := weightedIndex(rng, f.caseWeight, func(index int) int {
		return f.cases[index].Weight
	}, len(f.cases))
	megaCase := f.cases[caseIndex]
	result := &MegaSymbolResult{CaseIndex: caseIndex}
	if len(megaCase.Rows) == 0 {
		return result
	}

	symbolIndex := weightedIndex(rng, f.symbolWeight, func(index int) int {
		return f.symbols[index].Weight
	}, len(f.symbols))
	result.Symbol = f.symbols[symbolIndex].Symbol

	for reel := 1; reel < len(board)-1; reel++ {
		for _, row := range megaCase.Rows {
			board[reel][row] = result.Symbol
			result.Positions = append(result.Positions, Position{Reel: reel, Row: row})
		}
	}
	return result
}

func weightedIndex(rng *rand.Rand, totalWeight int, weight func(int) int, count int) int {
	draw := rng.Intn(totalWeight)
	for index := 0; index < count; index++ {
		draw -= weight(index)
		if draw < 0 {
			return index
		}
	}
	return count - 1
}
