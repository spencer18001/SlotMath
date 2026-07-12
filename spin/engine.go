package spin

import (
	"fmt"
	"math/rand"
	"time"
)

type Engine struct {
	info             Info
	rng              *rand.Rand
	generators       map[Mode]*generator
	lineEvaluator    *lineEvaluator
	scatterEvaluator *scatterEvaluator
	paylines         [][]int
	paytable         Paytable
	symbols          []string
}

func newEngine(data loadedGame, seed int64) (*Engine, error) {
	actualSeed := seed
	if actualSeed == 0 {
		actualSeed = time.Now().UnixNano()
	}
	generators := make(map[Mode]*generator, len(data.reels))
	for mode, reels := range data.reels {
		generator, err := newGenerator(reels, data.config.NumRows)
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
			BetPerLine: data.config.BetPerLine, ReelCount: len(baseReels), PaylineCount: len(data.paylines),
		},
		rng: rand.New(rand.NewSource(actualSeed)), generators: generators,
		lineEvaluator:    newLineEvaluator(data.paylines, data.paytable, data.config.WildSymbols),
		scatterEvaluator: newScatterEvaluator(data.config.ScatterSymbols, data.paytable),
		paylines:         data.paylines, paytable: data.paytable, symbols: symbols,
	}, nil
}

func (g *Engine) ResolveBet(totalBet int64) (Bet, error) {
	if totalBet <= 0 {
		return Bet{}, fmt.Errorf("bet must be greater than zero")
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
	lineResult, err := g.lineEvaluator.evaluate(drawn.Board, bet.ActiveLines, bet.PerLine)
	if err != nil {
		return Result{}, err
	}
	scatterResult, err := g.scatterEvaluator.evaluate(drawn.Board, bet.Total)
	if err != nil {
		return Result{}, err
	}
	var totalWin int64
	for _, win := range lineResult.wins {
		totalWin += win.Payout
	}
	for _, win := range scatterResult.wins {
		totalWin += win.Payout
	}
	return Result{
		Mode:  mode,
		Stops: drawn.Stops, Board: drawn.Board,
		LineWins: lineResult.wins, ScatterWins: scatterResult.wins,
		TotalWin:  totalWin,
		FreeSpins: scatterResult.freeSpins,
	}, nil
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
