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
	wayEvaluator     *WayEvaluator
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
	Mode         Mode          // 使用的 Reel Mode
	Stops        []int         // 初始停輪位置
	Board        Board         // 初始盤面
	LineWins     []LineWin     // 初始盤面的 Line 獎項
	WayWins      []WayWin      // 初始盤面的 Way 獎項
	ScatterWins  []ScatterWin  // 初始盤面的 Scatter 獎項
	CascadeSteps []CascadeStep // 每次 Tumble 後的盤面與計分結果
	TotalWin     int64         // 初始盤面與所有 Cascade 的總得分
	FreeSpins    int           // 初始盤面與所有 Cascade 觸發的 Free Spins 總數
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
	scatterPayBet := int64(1)
	if len(gameSetting.paytable.Way) > 0 {
		scatterPayBet = gameSetting.baseSetting.WayPayBet
	}

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
		scatterEvaluator: newScatterEvaluator(gameSetting.paytable, scatterPayBet),
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
		Mode: mode, Stops: drawResult.Stops, Board: drawResult.Board,
		LineWins: eval.lineWins, WayWins: eval.wayWins, ScatterWins: eval.scatterWins,
		TotalWin: eval.totalWin, FreeSpins: eval.freeSpins,
	}, nil
}

func (e *Engine) evaluateCascading(mode Mode, generator *Generator, drawn DrawResult, bet Bet) (SpinResult, error) {
	initialEval, err := e.evaluateBoard(drawn.Board, bet)
	if err != nil {
		return SpinResult{}, err
	}

	result := SpinResult{
		Mode:        mode,
		Stops:       drawn.Stops,
		Board:       drawn.Board,
		LineWins:    initialEval.lineWins,
		WayWins:     initialEval.wayWins,
		ScatterWins: initialEval.scatterWins,
		TotalWin:    initialEval.totalWin,
		FreeSpins:   initialEval.freeSpins,
	}

	board := drawn.Board
	stops := drawn.Stops
	remove := removeMask(board, initialEval.lineWins, initialEval.wayWins, initialEval.scatterWins)
	for stepIndex := 0; !maskEmpty(remove); stepIndex++ {
		if stepIndex >= maxCascadeSteps {
			return SpinResult{}, fmt.Errorf("cascade exceeded max steps %d", maxCascadeSteps)
		}

		after, nextStops, removed, err := generator.tumble(board, stops, remove)
		if err != nil {
			return SpinResult{}, err
		}
		eval, err := e.evaluateBoard(after, bet)
		if err != nil {
			return SpinResult{}, err
		}

		result.CascadeSteps = append(result.CascadeSteps, CascadeStep{
			Index:            stepIndex,
			Board:            after,
			RemovedPositions: removed,
			LineWins:         eval.lineWins,
			WayWins:          eval.wayWins,
			ScatterWins:      eval.scatterWins,
			TotalWin:         eval.totalWin,
			FreeSpins:        eval.freeSpins,
		})
		result.TotalWin += eval.totalWin
		result.FreeSpins += eval.freeSpins

		board = after
		stops = nextStops
		remove = removeMask(board, eval.lineWins, eval.wayWins, eval.scatterWins)
	}
	return result, nil
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
