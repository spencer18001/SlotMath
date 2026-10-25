package spin

import (
	"fmt"
	"math/rand"
	"time"
)

const maxCascadeSteps = 100

type Engine struct {
	info             GameInfo
	rng              *rand.Rand
	paylines         [][]int
	paytable         Paytable
	generators       map[Mode]*Generator
	lineEvaluator    *LineEvaluator
	wayEvaluator     *wayEvaluator
	scatterEvaluator *ScatterEvaluator
}

type GameInfo struct {
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

type SpinRequest struct {
	Bet  int64
	Mode Mode
}

type SpinResult struct {
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

type boardEvaluation struct {
	lineWins    []LineWin
	wayWins     []WayWin
	scatterWins []ScatterWin
	totalWin    int64
	freeSpins   int
}

func newEngine(gameSetting GameSetting, seed int64) (*Engine, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}

	drawMode, err := parseDrawMode(gameSetting.baseSetting.DrawMode)
	if err != nil {
		return nil, err
	}

	generators := make(map[Mode]*Generator, len(gameSetting.reels))
	for mode, reels := range gameSetting.reels {
		generator, err := newGenerator(reels, gameSetting.baseSetting.NumRows, drawMode)
		if err != nil {
			return nil, fmt.Errorf("create %s generator: %w", mode, err)
		}

		generators[mode] = generator
	}

	baseReels := gameSetting.reels[ModeBase]

	return &Engine{
		info: GameInfo{
			GameID: gameSetting.baseSetting.GameID, Path: gameSetting.path, Seed: seed,
			BetPerLine: gameSetting.baseSetting.BetPerLine, WayPayBet: gameSetting.baseSetting.WayPayBet,
			DrawMode: string(drawMode), Cascading: gameSetting.baseSetting.Cascading,
			ReelCount: len(baseReels), PaylineCount: len(gameSetting.paylines),
		},
		rng: rand.New(rand.NewSource(actualSeed)), generators: generators,
		paylines: gameSetting.paylines, paytable: gameSetting.paytable,
		lineEvaluator:    newLineEvaluator(gameSetting.paylines, gameSetting.paytable, gameSetting.baseSetting.WildSymbols),
		wayEvaluator:     newWayEvaluator(gameSetting.paytable, gameSetting.baseSetting.WildSymbols, gameSetting.baseSetting.WayPayBet),
		scatterEvaluator: newScatterEvaluator(gameSetting.paytable),
	}, nil
}

func (e *Engine) ResolveBet(totalBet int64) (Bet, error) {
	if totalBet <= 0 {
		return Bet{}, fmt.Errorf("bet must be greater than zero")
	}
	if e.info.WayPayBet > 0 && totalBet%e.info.WayPayBet != 0 {
		return Bet{}, fmt.Errorf("bet %d must be a multiple of WayPayBet %d", totalBet, e.info.WayPayBet)
	}
	if len(e.paytable.Line) == 0 {
		return Bet{Total: totalBet, ActiveLines: 0}, nil
	}
	if totalBet%e.info.BetPerLine != 0 {
		return Bet{}, fmt.Errorf("bet %d must be a multiple of bet per line %d", totalBet, e.info.BetPerLine)
	}
	activeLines := int(totalBet / e.info.BetPerLine)
	if activeLines > e.info.PaylineCount {
		return Bet{}, fmt.Errorf("bet %d activates %d lines, maximum is %d", totalBet, activeLines, e.info.PaylineCount)
	}
	return Bet{Total: totalBet, ActiveLines: activeLines}, nil
}

func (e *Engine) DefaultBet() Bet {
	if len(e.paytable.Line) == 0 {
		if len(e.paytable.Way) > 0 && e.info.WayPayBet > 0 {
			return Bet{Total: e.info.WayPayBet, ActiveLines: 0}
		}
		return Bet{Total: e.info.BetPerLine, ActiveLines: 0}
	}
	return Bet{Total: e.info.BetPerLine * int64(e.info.PaylineCount), ActiveLines: e.info.PaylineCount}
}

func (e *Engine) Spin(request SpinRequest) (SpinResult, error) {
	bet, err := e.ResolveBet(request.Bet)
	if err != nil {
		return SpinResult{}, err
	}

	mode, generator, err := e.generatorFor(request.Mode)
	if err != nil {
		return SpinResult{}, err
	}

	drawResult := generator.draw(e.rng)
	if !e.info.Cascading {
		return e.evaluateSingle(mode, drawResult, bet)
	}
	return e.evaluateCascading(mode, generator, drawResult, bet)
}

func (e *Engine) generatorFor(mode Mode) (Mode, *Generator, error) {
	if mode == "" {
		mode = ModeBase
	}
	generator, ok := e.generators[mode]
	if !ok {
		return "", nil, fmt.Errorf("reels for mode %q are not configured", mode)
	}
	return mode, generator, nil
}

func (e *Engine) evaluateSingle(mode Mode, drawResult DrawResult, bet Bet) (SpinResult, error) {
	eval, err := e.evaluateBoard(drawResult.Board, bet)
	if err != nil {
		return SpinResult{}, err
	}

	return SpinResult{
		Mode: mode, Stops: drawResult.Stops, Board: drawResult.Board, InitialBoard: drawResult.Board.Clone(),
		LineWins: eval.lineWins, WayWins: eval.wayWins, ScatterWins: eval.scatterWins,
		TotalWin: eval.totalWin, FreeSpins: eval.freeSpins,
	}, nil
}

func (e *Engine) evaluateCascading(mode Mode, generator *Generator, drawn DrawResult, bet Bet) (SpinResult, error) {
	initialStops := append([]int(nil), drawn.Stops...)
	stops := append([]int(nil), drawn.Stops...)
	board := drawn.Board.Clone()
	result := SpinResult{Mode: mode, Stops: initialStops, InitialBoard: drawn.Board.Clone()}
	for stepIndex := 0; stepIndex < maxCascadeSteps; stepIndex++ {
		before := board.Clone()
		eval, err := e.evaluateBoard(board, bet)
		if err != nil {
			return SpinResult{}, err
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
			return SpinResult{}, err
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
	return SpinResult{}, fmt.Errorf("cascade exceeded max steps %d", maxCascadeSteps)
}

func (e *Engine) evaluateBoard(board Board, bet Bet) (boardEvaluation, error) {
	var eval boardEvaluation

	if len(e.paytable.Line) > 0 {
		lineResult, err := e.lineEvaluator.evaluate(board, bet.ActiveLines, e.info.BetPerLine)
		if err != nil {
			return boardEvaluation{}, err
		}

		eval.lineWins = lineResult.wins
	}

	wayResult, err := e.wayEvaluator.evaluate(board, bet.Total)
	if err != nil {
		return boardEvaluation{}, err
	}
	eval.wayWins = wayResult.wins

	scatterResult, err := e.scatterEvaluator.evaluate(board, bet.Total)
	if err != nil {
		return boardEvaluation{}, err
	}
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

func (e *Engine) Info() GameInfo     { return e.info }
func (e *Engine) Paytable() Paytable { return e.paytable }
func (e *Engine) Modes() []Mode {
	modes := []Mode{ModeBase}
	if _, ok := e.generators[ModeFree]; ok {
		modes = append(modes, ModeFree)
	}
	return modes
}
func (e *Engine) Payline(index int) ([]int, bool) {
	if index < 0 || index >= len(e.paylines) {
		return nil, false
	}
	return e.paylines[index], true
}
