package spin

type Request struct{ Bet int64 }

type Bet struct {
	Total       int64
	PerLine     int64
	ActiveLines int
}

type Board [][]string

func (b Board) Rows() [][]string {
	if len(b) == 0 {
		return nil
	}
	rowCount := len(b[0])
	rows := make([][]string, rowCount)
	for row := 0; row < rowCount; row++ {
		rows[row] = make([]string, len(b))
		for reel := range b {
			rows[row][reel] = b[reel][row]
		}
	}
	return rows
}

type LineWin struct {
	LineIndex    int
	PayRuleIndex int
	Payout       int64
}

type ScatterWin struct {
	PayRuleIndex int
	Payout       int64
}

type Result struct {
	Stops           []int
	Board           Board
	LineWins        []LineWin
	ScatterWins     []ScatterWin
	TotalLineWin    int64
	TotalScatterWin int64
	TotalWin        int64
}

type PayEntry struct {
	Symbol              string  `json:"symbol"`
	Count               int     `json:"count"`
	Odds                int64   `json:"odds"`
	ExpectedProbability float64 `json:"expectedProbability"`
}

type Paytable struct {
	Line    []PayEntry `json:"line"`
	Scatter []PayEntry `json:"scatter"`
}

type Info struct {
	GameID       string
	Path         string
	Seed         int64
	BetPerLine   int64
	ReelCount    int
	PaylineCount int
}
