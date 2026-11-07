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
	Index            int          // Cascade 索引，第一個 Tumble 為 0
	Board            Board        // Tumble 後的盤面
	RemovedPositions []Position   // 為產生此盤面而從前一盤面移除的位置
	LineWins         []LineWin    // 此盤面的 Line 獎項
	WayWins          []WayWin     // 此盤面的 Way 獎項
	ScatterWins      []ScatterWin // 此盤面的 Scatter 獎項
	TotalWin         int64        // 此盤面的總得分
	FreeSpins        int          // 此盤面觸發的 Free Spins 數量
}
