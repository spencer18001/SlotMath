package spin

import (
	"fmt"
	"math/rand"
)

type drawResult struct {
	Stops []int
	Board Board
}

type generator struct {
	reels [][]string
	rows  int
}

func newGenerator(reels [][]string, rows int) (*generator, error) {
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
	return &generator{reels: reels, rows: rows}, nil
}

func (g *generator) draw(rng *rand.Rand) drawResult {
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
	return drawResult{Stops: stops, Board: visible}
}
