package board

import (
	"fmt"
	"math/rand"
)

// Board stores symbols as board[reel][row].
type Board [][]string

// SpinResult contains one generated board and the reel stops that produced it.
type SpinResult struct {
	Stops []int
	Board Board
}

// Generator draws visible boards from reel strips.
type Generator struct {
	reels [][]string
	rows  int
}

// NewGenerator creates a board generator for a fixed reel set and visible row count.
func NewGenerator(reels [][]string, rows int) (*Generator, error) {
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
	return &Generator{reels: reels, rows: rows}, nil
}

// Draw chooses one stop per reel and returns the visible symbols from that stop downward.
func (g *Generator) Draw(rng *rand.Rand) SpinResult {
	stops := make([]int, len(g.reels))
	visible := make(Board, len(g.reels))

	for reelIndex, reel := range g.reels {
		stop := rng.Intn(len(reel))
		stops[reelIndex] = stop

		visibleReel := make([]string, g.rows)
		for row := 0; row < g.rows; row++ {
			visibleReel[row] = reel[(stop+row)%len(reel)]
		}
		visible[reelIndex] = visibleReel
	}

	return SpinResult{Stops: stops, Board: visible}
}

// Rows returns the board as rows first, which is easier to print and inspect.
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
