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

// Paytable groups pay rules by evaluator type.
type Paytable struct {
	Line    []PayEntry `json:"line"`
	Scatter []PayEntry `json:"scatter"`
}

// PayEntry describes one payout rule, such as K pays 100 for 5 on a line.
type PayEntry struct {
	Symbol              string  `json:"symbol"`
	Count               int     `json:"count"`
	Payout              int64   `json:"payout"`
	ExpectedProbability float64 `json:"expectedProbability"`
}
