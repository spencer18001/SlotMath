package spin

type Bet struct {
	Total       int64
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

type LineWin struct {
	LineIndex          int
	PaytableEntryIndex int
	Payout             int64
	Positions          []Position
}

type ScatterWin struct {
	PaytableEntryIndex int
	Payout             int64
	Positions          []Position
}

type WayWin struct {
	PaytableEntryIndex int
	Count              int
	Ways               int64
	Payout             int64
	Positions          []Position
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
