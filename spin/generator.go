package spin

import (
	"fmt"
	"math/rand"
)

// 抽取盤面的模式
type DrawMode string

const (
	drawModeReelStrips      DrawMode = "reelStrips"      // 每軸只停一次, 每列為同一條輪帶的連續符號
	drawModeIndependentRows DrawMode = "independentRows" // 同一軸的不同列為獨立滾輪, 共用 reel strip, 但各自抽一個停輪位置
)

// 盤面生成
type Generator struct {
	reels [][]string // 滾輪帶
	rows  int        // 盤面的每一軸有幾列
	mode  DrawMode
}

// 抽取盤面的結果
type DrawResult struct {
	Stops []int // 每個滾輪的停輪索引
	Board Board
}

func parseDrawMode(value string) (DrawMode, error) {
	if value == "" {
		return drawModeReelStrips, nil
	}

	mode := DrawMode(value)
	switch mode {
	case drawModeReelStrips, drawModeIndependentRows:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported DrawMode %q", value)
	}
}

func newGenerator(reels [][]string, rows int, mode DrawMode) (*Generator, error) {
	if rows <= 0 {
		return nil, fmt.Errorf("rows must be greater than zero")
	}
	if len(reels) == 0 {
		return nil, fmt.Errorf("at least one reel is required")
	}
	for reelIndex, reel := range reels {
		if len(reel) < rows {
			return nil, fmt.Errorf("reel %d has %d stops, needs at least %d", reelIndex, len(reel), rows)
		}
	}
	return &Generator{reels: reels, rows: rows, mode: mode}, nil
}

// 抽取盤面
func (g *Generator) draw(rng *rand.Rand) DrawResult {
	switch g.mode {
	case drawModeIndependentRows:
		return g.drawIndependentRows(rng)
	default:
		return g.drawReelStrips(rng)
	}
}

func (g *Generator) drawReelStrips(rng *rand.Rand) DrawResult {
	stops := make([]int, len(g.reels))
	board := make(Board, len(g.reels))
	for reelIndex, reel := range g.reels {
		stop := rng.Intn(len(reel))
		stops[reelIndex] = stop

		boardReel := make([]string, g.rows)
		for row := 0; row < g.rows; row++ {
			boardReel[row] = reel[(stop+row)%len(reel)]
		}
		board[reelIndex] = boardReel
	}
	return DrawResult{Stops: stops, Board: board}
}

func (g *Generator) drawIndependentRows(rng *rand.Rand) DrawResult {
	stops := make([]int, len(g.reels)*g.rows)
	board := make(Board, len(g.reels))
	for reelIndex, reel := range g.reels {
		boardReel := make([]string, g.rows)
		for row := 0; row < g.rows; row++ {
			stop := rng.Intn(len(reel))
			stops[reelIndex*g.rows+row] = stop

			boardReel[row] = reel[stop]
		}
		board[reelIndex] = boardReel
	}
	return DrawResult{Stops: stops, Board: board}
}

func (g *Generator) tumble(board Board, stops []int, remove [][]bool) (Board, []int, []Position, error) {
	if g.mode != drawModeReelStrips {
		return nil, nil, nil, fmt.Errorf("cascading tumble requires DrawMode %q", drawModeReelStrips)
	}
	if len(stops) != len(g.reels) {
		return nil, nil, nil, fmt.Errorf("stops count %d does not match reels count %d", len(stops), len(g.reels))
	}
	nextStops := append([]int(nil), stops...)
	nextBoard := make(Board, len(board))
	var removed []Position
	for reelIndex, reel := range board {
		if len(reel) != g.rows {
			return nil, nil, nil, fmt.Errorf("board reel %d has %d rows, expected %d", reelIndex, len(reel), g.rows)
		}
		if len(remove) <= reelIndex || len(remove[reelIndex]) != g.rows {
			return nil, nil, nil, fmt.Errorf("remove mask reel %d does not match row count %d", reelIndex, g.rows)
		}
		removeCount := 0
		remaining := make([]string, 0, g.rows)
		for rowIndex, symbol := range reel {
			if remove[reelIndex][rowIndex] {
				removeCount++
				removed = append(removed, Position{Reel: reelIndex, Row: rowIndex})
				continue
			}
			remaining = append(remaining, symbol)
		}
		if removeCount == 0 {
			nextBoard[reelIndex] = append([]string(nil), reel...)
			continue
		}
		reelStrip := g.reels[reelIndex]
		nextStop := mod(nextStops[reelIndex]-removeCount, len(reelStrip))
		newReel := make([]string, 0, g.rows)
		for offset := 0; offset < removeCount; offset++ {
			newReel = append(newReel, reelStrip[(nextStop+offset)%len(reelStrip)])
		}
		newReel = append(newReel, remaining...)
		if len(newReel) != g.rows {
			return nil, nil, nil, fmt.Errorf("tumbled reel %d has %d rows, expected %d", reelIndex, len(newReel), g.rows)
		}
		nextStops[reelIndex] = nextStop
		nextBoard[reelIndex] = newReel
	}
	return nextBoard, nextStops, removed, nil
}

func mod(value, divisor int) int {
	remainder := value % divisor
	if remainder < 0 {
		return remainder + divisor
	}
	return remainder
}
