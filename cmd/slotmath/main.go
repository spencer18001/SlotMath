package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"slotmath/internal/app"
	"slotmath/internal/probe"
)

type options struct {
	GamePath         string
	Spins            int
	Seed             int64
	ProbeLineRuleID  int
	HasLineRuleProbe bool
}

func main() {
	startedAt := time.Now()
	opts := parseFlags()
	if err := validateOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: %v\n", err)
		os.Exit(1)
	}

	gameData, err := app.LoadGame(opts.GamePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: load game: %v\n", err)
		os.Exit(1)
	}

	game := app.NewLineGame(gameData)
	if opts.HasLineRuleProbe {
		result, err := game.RunLinePayRuleProbe(opts.Spins, opts.Seed, opts.ProbeLineRuleID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printLinePayRuleProbe(result, opts, time.Since(startedAt))
		return
	}

	summary, err := game.RunSims(opts.Spins, opts.Seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: run sims: %v\n", err)
		os.Exit(1)
	}

	printSummary(summary, time.Since(startedAt))
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.GamePath, "game", "games/sample_lines", "path to a game definition folder")
	flag.IntVar(&opts.Spins, "spins", 100000, "number of spins to simulate")
	flag.Int64Var(&opts.Seed, "seed", 0, "base random seed; 0 means random")
	flag.IntVar(&opts.ProbeLineRuleID, "probe-line-pay-rule", -1, "paytable.line rule id to probe on payline 0; -1 disables probe")
	flag.Parse()
	opts.HasLineRuleProbe = opts.ProbeLineRuleID >= 0
	return opts
}

func validateOptions(opts options) error {
	if opts.GamePath == "" {
		return fmt.Errorf("game path is required")
	}
	if opts.Spins <= 0 {
		return fmt.Errorf("spins must be greater than zero")
	}
	return nil
}

func printSummary(summary *app.SimulationSummary, elapsed time.Duration) {
	fmt.Println("SlotMath line-game simulator")
	fmt.Printf("Game ID: %s\n", summary.GameID)
	fmt.Printf("Game path: %s\n", summary.GamePath)
	fmt.Printf("Spins: %d\n", summary.Spins)
	printSeed(summary.Seed)
	fmt.Printf("Reels: %d\n", summary.ReelCount)
	fmt.Printf("Paylines: %d\n", summary.Paylines)
	fmt.Printf("Line pays: %d\n", summary.LinePays)
	fmt.Printf("Total bet: %d\n", summary.TotalBet)
	fmt.Printf("Total win: %d\n", summary.TotalWin)
	fmt.Printf("Hit count: %d\n", summary.HitCount)
	fmt.Printf("First stops: %v\n", summary.FirstStops)
	fmt.Println("First board:")
	for _, row := range summary.FirstBoard.Rows() {
		fmt.Print("  ")
		for index, symbol := range row {
			if index > 0 {
				fmt.Print(" | ")
			}
			fmt.Printf("%2s", symbol)
		}
		fmt.Println()
	}
	fmt.Printf("First win: %d\n", summary.FirstWin)
	if len(summary.FirstLineWins) > 0 {
		fmt.Println("First line wins:")
		for _, win := range summary.FirstLineWins {
			fmt.Printf("  line %d: %-2s x%d pays %d [%s]\n", win.LineIndex, win.Symbol, win.Count, win.Payout, strings.Join(win.Symbols, " "))
		}
	}
	fmt.Printf("Status: %s\n", summary.Status)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printLinePayRuleProbe(result *probe.LinePayRuleResult, opts options, elapsed time.Duration) {
	fmt.Println("SlotMath line pay rule probe")
	fmt.Printf("Game path: %s\n", opts.GamePath)
	fmt.Printf("Spins: %d\n", result.Spins)
	printSeed(opts.Seed)
	fmt.Println("Payline ID: 0")
	fmt.Printf("Payline: %v\n", result.Payline)
	fmt.Printf("Rule ID: %d\n", result.RuleID)
	fmt.Printf("Rule: %s x%d pays %d\n", result.Rule.Symbol, result.Rule.Count, result.Rule.Payout)
	fmt.Println("Wild: included")
	fmt.Printf("Hits: %d\n", result.Hits)
	fmt.Printf("Probability: %.8f\n", result.Probability)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printSeed(seed int64) {
	if seed == 0 {
		fmt.Println("Seed: random")
		return
	}
	fmt.Printf("Seed: %d\n", seed)
}
