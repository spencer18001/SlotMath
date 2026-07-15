package spin

import (
	"fmt"
	"math/rand"
)

type drawMode string

const (
	drawModeReelStrips      drawMode = "reelStrips"
	drawModeIndependentRows drawMode = "independentRows"
)

type drawResult struct {
	Stops []int
	Board Board
}

type generator struct {
	reels [][]string
	rows  int
	mode  drawMode
}

func parseDrawMode(value string) (drawMode, error) {
	if value == "" {
		return drawModeReelStrips, nil
	}
	mode := drawMode(value)
	switch mode {
	case drawModeReelStrips, drawModeIndependentRows:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported drawMode %q", value)
	}
}

func newGenerator(reels [][]string, rows int, mode drawMode) (*generator, error) {
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
	return &generator{reels: reels, rows: rows, mode: mode}, nil
}

func (g *generator) draw(rng *rand.Rand) drawResult {
	switch g.mode {
	case drawModeIndependentRows:
		return g.drawIndependentRows(rng)
	default:
		return g.drawReelStrips(rng)
	}
}

func (g *generator) drawReelStrips(rng *rand.Rand) drawResult {
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

func (g *generator) drawIndependentRows(rng *rand.Rand) drawResult {
	stops := make([]int, len(g.reels)*g.rows)
	visible := make(Board, len(g.reels))
	for reelIndex, reel := range g.reels {
		visibleReel := make([]string, g.rows)
		for row := 0; row < g.rows; row++ {
			stop := rng.Intn(len(reel))
			stops[reelIndex*g.rows+row] = stop
			visibleReel[row] = reel[stop]
		}
		visible[reelIndex] = visibleReel
	}
	return drawResult{Stops: stops, Board: visible}
}
