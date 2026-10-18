package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"slotmath/flow"
	"slotmath/simulator"
	"slotmath/spin"
)

type options struct {
	GamePath string
	Spins    int
	Seed     int64
	Bet      int64
	Verbose  bool
}

func main() {
	startedAt := time.Now()
	opts := parseFlags()
	if err := validateOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: %v\n", err)
		os.Exit(1)
	}

	engine, err := spin.Load(filepath.Join("games", opts.GamePath), opts.Seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: load game: %v\n", err)
		os.Exit(1)
	}
	bet := opts.Bet
	if bet == 0 {
		bet = engine.DefaultBet().Total
	}
	slotFlow := flow.New(engine)
	sim := simulator.New(slotFlow)
	summary, err := sim.Run(simulator.Request{Spins: opts.Spins, Bet: bet})
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: run sims: %v\n", err)
		os.Exit(1)
	}
	printSummary(engine, summary, opts, time.Since(startedAt))
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.GamePath, "game", "line", "path to a game definition folder")
	flag.IntVar(&opts.Spins, "spins", 10000000, "number of spins to simulate")
	flag.Int64Var(&opts.Seed, "seed", 0, "base random seed; 0 means random")
	flag.Int64Var(&opts.Bet, "bet", 0, "total bet per spin; 0 activates all configured paylines")
	flag.BoolVar(&opts.Verbose, "v", false, "show detailed verbose pay hit summaries")
	flag.Parse()
	return opts
}

func validateOptions(opts options) error {
	if opts.GamePath == "" {
		return fmt.Errorf("game path is required")
	}
	if opts.Spins <= 0 {
		return fmt.Errorf("spins must be greater than zero")
	}
	if opts.Bet < 0 {
		return fmt.Errorf("bet cannot be negative")
	}
	return nil
}

func printSummary(engine *spin.Engine, summary *simulator.Summary, opts options, elapsed time.Duration) {
	info := engine.Info()
	paytable := engine.Paytable()
	fmt.Printf("Game ID: %s\n", info.GameID)
	fmt.Printf("Game path: %s\n", info.Path)
	fmt.Printf("Base spins: %d\n", summary.Spins)
	fmt.Printf("Free spins: %d\n", summary.FreeSpins)
	printSeed(info.Seed)
	fmt.Printf("Reels: %d\n", info.ReelCount)
	fmt.Printf("Paylines: %d\n", info.PaylineCount)
	fmt.Printf("Line pays: %d\n", len(paytable.Line))
	fmt.Printf("Way pays: %d\n", len(paytable.Way))
	fmt.Printf("Scatter pays: %d\n", len(paytable.Scatter))
	fmt.Printf("Bet per line: %d\n", summary.Bet.PerLine)
	if info.WayPayBet > 0 {
		fmt.Printf("Way pay bet: %d\n", info.WayPayBet)
	}
	if info.Cascading {
		fmt.Println("Cascading: true")
	}
	fmt.Printf("Active lines: %d\n", summary.Bet.ActiveLines)
	fmt.Printf("Bet per spin: %d\n", summary.Bet.Total)
	fmt.Printf("Total bet: %d\n", summary.TotalBet)
	for _, modeSummary := range summary.Modes {
		printModeWinSummary(modeSummary, summary.Bet.Total, summary.TotalBet)
	}
	fmt.Printf("Total line RTP: %.2f%%\n", ratio(summary.TotalLineWin, summary.TotalBet)*100)
	fmt.Printf("Total way RTP: %.2f%%\n", ratio(summary.TotalWayWin, summary.TotalBet)*100)
	fmt.Printf("Total scatter RTP: %.2f%%\n", ratio(summary.TotalScatterWin, summary.TotalBet)*100)
	fmt.Printf("Total RTP: %.2f%%\n", ratio(summary.TotalWin, summary.TotalBet)*100)
	fmt.Printf("Hit count: %d\n", summary.HitCount)
	if info.Cascading || summary.CascadeSteps > 0 {
		fmt.Printf("Cascade steps: %d\n", summary.CascadeSteps)
		fmt.Printf("Average cascade steps per generated spin: %.4f\n", ratioInt(summary.CascadeSteps, summary.GeneratedSpins))
		fmt.Printf("Cascade hit count: %d\n", summary.CascadeHitCount)
	}
	if opts.Verbose {
		for _, modeSummary := range summary.Modes {
			printPayHitSummary(modeSummary)
		}
	}
	fmt.Printf("Elapsed: %.1fs\n", elapsed.Seconds())
}

func printModeWinSummary(summary simulator.ModeSummary, betPerSpin int64, totalBet int64) {
	modeBet := int64(summary.Spins) * betPerSpin
	fmt.Printf("%s line RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalLineWin, modeBet)*100)
	fmt.Printf("%s way RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalWayWin, modeBet)*100)
	fmt.Printf("%s scatter RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalScatterWin, modeBet)*100)
	fmt.Printf("%s RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalWin, modeBet)*100)
	if summary.CascadeSteps > 0 {
		fmt.Printf("%s average cascade steps: %.4f\n", summary.Mode, ratioInt(summary.CascadeSteps, summary.Spins))
	}
	if summary.Mode != spin.ModeBase {
		fmt.Printf("%s RTP contribution: %.2f%%\n", summary.Mode, ratio(summary.TotalWin, totalBet)*100)
	}
}

func printPayHitSummary(summary simulator.ModeSummary) {
	if len(summary.PayHits) == 0 || summary.Spins == 0 {
		return
	}
	fmt.Printf("%s initial-board pay hit summary (line pays use payline 0):\n", summary.Mode)
	for _, hit := range summary.PayHits {
		probability := ratio(hit.Hits, int64(summary.Spins))
		fmt.Printf(
			"  %-7s %-2s x%d odds %-4d probability %.8f\n",
			hit.Kind, hit.Symbol, hit.Count, hit.Odds, probability,
		)
	}
}

func ratio(numerator int64, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func ratioInt(numerator int, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func printSeed(seed int64) {
	if seed == 0 {
		fmt.Println("Seed: random")
		return
	}
	fmt.Printf("Seed: %d\n", seed)
}
