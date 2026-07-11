package config

// GameConfig is the entry-point game definition loaded from config.json.
type GameConfig struct {
	GameID         string   `json:"gameId"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Bet            int64    `json:"bet"`
	NumReels       int      `json:"numReels"`
	NumRows        int      `json:"numRows"`
	ReelsFile      string   `json:"reelsFile"`
	PaylinesFile   string   `json:"paylinesFile"`
	PaytableFile   string   `json:"paytableFile"`
	WildSymbols    []string `json:"wildSymbols"`
	ScatterSymbols []string `json:"scatterSymbols"`
}

// Paytable groups pay rules by evaluator type. The first version only loads
// line pays, but this shape can later add scatter, ways, and cluster sections.
type Paytable struct {
	Line []PayEntry `json:"line"`
}

// PayEntry describes one payout rule, such as H1 pays 200 for 5 on a line.
type PayEntry struct {
	Symbol string `json:"symbol"`
	Count  int    `json:"count"`
	Payout int64  `json:"payout"`
}
