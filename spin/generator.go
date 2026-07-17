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

func (g *generator) tumble(board Board, stops []int, remove [][]bool) (Board, []int, []Position, error) {
	if g.mode != drawModeReelStrips {
		return nil, nil, nil, fmt.Errorf("cascading tumble requires drawMode %q", drawModeReelStrips)
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
