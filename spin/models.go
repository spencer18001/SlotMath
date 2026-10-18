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

type Position struct {
	Reel int
	Row  int
}

type Board [][]string // 盤面 ([reel][row] -> symbol)

func (b Board) Clone() Board {
	if len(b) == 0 {
		return nil
	}
	clone := make(Board, len(b))
	for reel := range b {
		clone[reel] = append([]string(nil), b[reel]...)
	}
	return clone
}

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

// 線獎
type LineWin struct {
	LineIndex    int   // 線索引
	PayRuleIndex int   // 賠付規則索引
	Payout       int64 // 賠付
	Positions    []Position
}

type ScatterWin struct {
	PayRuleIndex int
	Payout       int64
	Positions    []Position
}

type WayWin struct {
	PayRuleIndex int
	Count        int
	Ways         int64
	Payout       int64
	Positions    []Position
}

type CascadeStep struct {
	Index            int
	BoardBefore      Board
	BoardAfter       Board
	RemovedPositions []Position
	LineWins         []LineWin
	WayWins          []WayWin
	ScatterWins      []ScatterWin
	TotalWin         int64
}

type Result struct {
	Mode         Mode
	Stops        []int
	Board        Board
	InitialBoard Board
	LineWins     []LineWin
	WayWins      []WayWin
	ScatterWins  []ScatterWin
	CascadeSteps []CascadeStep
	TotalWin     int64
	FreeSpins    int
}

// 賠付項目
type PayEntry struct {
	Symbol    string // Symbol
	Count     int    // 個數
	Odds      int64  // 賠率
	FreeSpins int
}

// 賠付表
type Paytable struct {
	Line    []PayEntry // 線獎
	Way     []PayEntry
	Scatter []PayEntry
}

type Info struct {
	GameID       string
	Path         string
	Seed         int64
	BetPerLine   int64
	WayPayBet    int64
	DrawMode     string
	Cascading    bool
	ReelCount    int
	PaylineCount int
}
