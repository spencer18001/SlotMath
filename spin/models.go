package spin

type Mode string

const (
	ModeBase Mode = "base"
	ModeFree Mode = "free"
)

type Request struct {
	Bet  int64
	Mode Mode
}

type Bet struct {
	Total       int64
	PerLine     int64
	ActiveLines int
}

type Board [][]string // [reel][row] -> symbol

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

type WayWin struct {
	PayRuleIndex int
	Count        int
	Ways         int64
	Payout       int64
}

type Result struct {
	Mode        Mode
	Stops       []int
	Board       Board
	LineWins    []LineWin
	WayWins     []WayWin
	ScatterWins []ScatterWin
	TotalWin    int64
	FreeSpins   int
}

type PayEntry struct {
	Symbol                string           `json:"symbol"`
	Count                 int              `json:"count"`
	Odds                  int64            `json:"odds"`
	ExpectedProbabilities map[Mode]float64 `json:"expectedProbabilities,omitempty"`
	FreeSpins             int              `json:"freeSpins,omitempty"`
}

func (p PayEntry) ExpectedProbabilityFor(mode Mode) float64 {
	if expected, ok := p.ExpectedProbabilities[mode]; ok {
		return expected
	}
	return 0
}

type Paytable struct {
	Line    []PayEntry `json:"line"`
	Way     []PayEntry `json:"way"`
	Scatter []PayEntry `json:"scatter"`
}

type Info struct {
	GameID       string
	Path         string
	Seed         int64
	BetPerLine   int64
	WayPayBet    int64
	ReelCount    int
	PaylineCount int
}
