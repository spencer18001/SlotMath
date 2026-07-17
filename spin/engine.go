package spin

import (
	"fmt"
	"math/rand"
	"time"
)

const maxCascadeSteps = 100

type Engine struct {
	info             Info
	rng              *rand.Rand
	generators       map[Mode]*generator
	lineEvaluator    *lineEvaluator
	wayEvaluator     *wayEvaluator
	scatterEvaluator *scatterEvaluator
	paylines         [][]int
	paytable         Paytable
	symbols          []string
	wayPayBet        int64
	cascading        bool
}

func newEngine(data loadedGame, seed int64) (*Engine, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}
	drawMode, err := parseDrawMode(data.config.DrawMode)
	if err != nil {
		return nil, err
	}
	generators := make(map[Mode]*generator, len(data.reels))
	for mode, reels := range data.reels {
		generator, err := newGenerator(reels, data.config.NumRows, drawMode)
		if err != nil {
			return nil, fmt.Errorf("create %s generator: %w", mode, err)
		}
		generators[mode] = generator
	}
	baseReels := data.reels[ModeBase]
	symbolSet := make(map[string]bool)
	for _, reelSet := range data.reels {
		for _, reel := range reelSet {
			for _, symbol := range reel {
				symbolSet[symbol] = true
			}
		}
	}
	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}
	return &Engine{
		info: Info{
			GameID: data.config.GameID, Path: data.path, Seed: seed,
			BetPerLine: data.config.BetPerLine, WayPayBet: data.config.WayPayBet,
			DrawMode: string(drawMode), Cascading: data.config.Cascading,
			ReelCount: len(baseReels), PaylineCount: len(data.paylines),
		},
		rng: rand.New(rand.NewSource(actualSeed)), generators: generators,
		lineEvaluator:    newLineEvaluator(data.paylines, data.paytable, data.config.WildSymbols),
		wayEvaluator:     newWayEvaluator(data.paytable, data.config.WildSymbols, data.config.WayPayBet),
		scatterEvaluator: newScatterEvaluator(data.config.ScatterSymbols, data.paytable, data.config.WayPayBet),
		paylines:         data.paylines, paytable: data.paytable, symbols: symbols,
		wayPayBet: data.config.WayPayBet, cascading: data.config.Cascading,
	}, nil
}

func (g *Engine) ResolveBet(totalBet int64) (Bet, error) {
	if totalBet <= 0 {
		return Bet{}, fmt.Errorf("bet must be greater than zero")
	}
	if g.wayPayBet > 0 && totalBet%g.wayPayBet != 0 {
		return Bet{}, fmt.Errorf("bet %d must be a multiple of wayPayBet %d", totalBet, g.wayPayBet)
	}
	if len(g.paytable.Line) == 0 {
		return Bet{Total: totalBet, PerLine: g.info.BetPerLine, ActiveLines: 0}, nil
	}
	if totalBet%g.info.BetPerLine != 0 {
		return Bet{}, fmt.Errorf("bet %d must be a multiple of bet per line %d", totalBet, g.info.BetPerLine)
	}
	activeLines := int(totalBet / g.info.BetPerLine)
	if activeLines > g.info.PaylineCount {
		return Bet{}, fmt.Errorf("bet %d activates %d lines, maximum is %d", totalBet, activeLines, g.info.PaylineCount)
	}
	return Bet{Total: totalBet, PerLine: g.info.BetPerLine, ActiveLines: activeLines}, nil
}

func (g *Engine) DefaultBet() Bet {
	if len(g.paytable.Line) == 0 {
		if len(g.paytable.Way) > 0 && g.wayPayBet > 0 {
			return Bet{Total: g.wayPayBet, PerLine: g.info.BetPerLine, ActiveLines: 0}
		}
		return Bet{Total: g.info.BetPerLine, PerLine: g.info.BetPerLine, ActiveLines: 0}
	}
	return Bet{Total: g.info.BetPerLine * int64(g.info.PaylineCount), PerLine: g.info.BetPerLine, ActiveLines: g.info.PaylineCount}
}

func (g *Engine) Spin(request Request) (Result, error) {
	bet, err := g.ResolveBet(request.Bet)
	if err != nil {
		return Result{}, err
	}
	mode, generator, err := g.generatorFor(request.Mode)
	if err != nil {
		return Result{}, err
	}
	drawn := generator.draw(g.rng)
	if !g.cascading {
		return g.evaluateSingle(mode, drawn, bet)
	}
	return g.evaluateCascading(mode, generator, drawn, bet)
}

func (g *Engine) evaluateSingle(mode Mode, drawn drawResult, bet Bet) (Result, error) {
	eval, err := g.evaluateBoard(drawn.Board, bet)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Mode: mode, Stops: drawn.Stops, Board: drawn.Board, InitialBoard: drawn.Board.Clone(),
		LineWins: eval.lineWins, WayWins: eval.wayWins, ScatterWins: eval.scatterWins,
		TotalWin: eval.totalWin, FreeSpins: eval.freeSpins,
	}, nil
}

func (g *Engine) evaluateCascading(mode Mode, generator *generator, drawn drawResult, bet Bet) (Result, error) {
	initialStops := append([]int(nil), drawn.Stops...)
	stops := append([]int(nil), drawn.Stops...)
	board := drawn.Board.Clone()
	result := Result{Mode: mode, Stops: initialStops, InitialBoard: drawn.Board.Clone()}
	for stepIndex := 0; stepIndex < maxCascadeSteps; stepIndex++ {
		before := board.Clone()
		eval, err := g.evaluateBoard(board, bet)
		if err != nil {
			return Result{}, err
		}
		result.LineWins = append(result.LineWins, eval.lineWins...)
		result.WayWins = append(result.WayWins, eval.wayWins...)
		result.ScatterWins = append(result.ScatterWins, eval.scatterWins...)
		result.TotalWin += eval.totalWin
		result.FreeSpins += eval.freeSpins
		remove := removeMask(board, eval.lineWins, eval.wayWins, eval.scatterWins)
		if maskEmpty(remove) {
			result.Board = board.Clone()
			return result, nil
		}
		after, nextStops, removed, err := generator.tumble(board, stops, remove)
		if err != nil {
			return Result{}, err
		}
		result.CascadeSteps = append(result.CascadeSteps, CascadeStep{
			Index: stepIndex, BoardBefore: before, BoardAfter: after.Clone(),
			RemovedPositions: removed,
			LineWins:         append([]LineWin(nil), eval.lineWins...),
			WayWins:          append([]WayWin(nil), eval.wayWins...),
			ScatterWins:      append([]ScatterWin(nil), eval.scatterWins...),
			TotalWin:         eval.totalWin,
		})
		board = after
		stops = nextStops
	}
	return Result{}, fmt.Errorf("cascade exceeded max steps %d", maxCascadeSteps)
}

type boardEvaluation struct {
	lineWins    []LineWin
	wayWins     []WayWin
	scatterWins []ScatterWin
	totalWin    int64
	freeSpins   int
}

func (g *Engine) evaluateBoard(board Board, bet Bet) (boardEvaluation, error) {
	var eval boardEvaluation
	if len(g.paytable.Line) > 0 {
		lineResult, err := g.lineEvaluator.evaluate(board, bet.ActiveLines, bet.PerLine)
		if err != nil {
			return boardEvaluation{}, err
		}
		eval.lineWins = lineResult.wins
	}
	wayResult, err := g.wayEvaluator.evaluate(board, bet.Total)
	if err != nil {
		return boardEvaluation{}, err
	}
	scatterResult, err := g.scatterEvaluator.evaluate(board, bet.Total)
	if err != nil {
		return boardEvaluation{}, err
	}
	eval.wayWins = wayResult.wins
	eval.scatterWins = scatterResult.wins
	eval.freeSpins = scatterResult.freeSpins
	for _, win := range eval.lineWins {
		eval.totalWin += win.Payout
	}
	for _, win := range eval.wayWins {
		eval.totalWin += win.Payout
	}
	for _, win := range eval.scatterWins {
		eval.totalWin += win.Payout
	}
	return eval, nil
}

func removeMask(board Board, lineWins []LineWin, wayWins []WayWin, scatterWins []ScatterWin) [][]bool {
	mask := make([][]bool, len(board))
	for reel := range board {
		mask[reel] = make([]bool, len(board[reel]))
	}
	mark := func(positions []Position) {
		for _, position := range positions {
			if position.Reel < 0 || position.Reel >= len(mask) {
				continue
			}
			if position.Row < 0 || position.Row >= len(mask[position.Reel]) {
				continue
			}
			mask[position.Reel][position.Row] = true
		}
	}
	for _, win := range lineWins {
		mark(win.Positions)
	}
	for _, win := range wayWins {
		mark(win.Positions)
	}
	for _, win := range scatterWins {
		mark(win.Positions)
	}
	return mask
}

func maskEmpty(mask [][]bool) bool {
	for _, reel := range mask {
		for _, removed := range reel {
			if removed {
				return false
			}
		}
	}
	return true
}

func (g *Engine) generatorFor(mode Mode) (Mode, *generator, error) {
	if mode == "" {
		mode = ModeBase
	}
	generator, ok := g.generators[mode]
	if !ok {
		return "", nil, fmt.Errorf("reels for mode %q are not configured", mode)
	}
	return mode, generator, nil
}

func (g *Engine) Info() Info         { return g.info }
func (g *Engine) Paytable() Paytable { return g.paytable }
func (g *Engine) Symbols() []string  { return append([]string(nil), g.symbols...) }
func (g *Engine) Modes() []Mode {
	modes := []Mode{ModeBase}
	if _, ok := g.generators[ModeFree]; ok {
		modes = append(modes, ModeFree)
	}
	return modes
}
func (g *Engine) Payline(index int) ([]int, bool) {
	if index < 0 || index >= len(g.paylines) {
		return nil, false
	}
	return g.paylines[index], true
}
