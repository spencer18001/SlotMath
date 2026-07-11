package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"slotmath/internal/app"
	"slotmath/internal/probe"
)

type options struct {
	GamePath            string
	Spins               int
	Seed                int64
	ProbeLineRuleID     int
	ProbeScatterRuleID  int
	HasLineRuleProbe    bool
	HasScatterRuleProbe bool
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
	if opts.HasScatterRuleProbe {
		result, err := game.RunScatterPayRuleProbe(opts.Spins, opts.Seed, opts.ProbeScatterRuleID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printScatterPayRuleProbe(result, opts, time.Since(startedAt))
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
	flag.IntVar(&opts.ProbeScatterRuleID, "probe-scatter-pay-rule", -1, "paytable.scatter rule id to probe; -1 disables probe")
	flag.Parse()
	opts.HasLineRuleProbe = opts.ProbeLineRuleID >= 0
	opts.HasScatterRuleProbe = opts.ProbeScatterRuleID >= 0
	return opts
}

func validateOptions(opts options) error {
	if opts.GamePath == "" {
		return fmt.Errorf("game path is required")
	}
	if opts.Spins <= 0 {
		return fmt.Errorf("spins must be greater than zero")
	}
	if opts.HasLineRuleProbe && opts.HasScatterRuleProbe {
		return fmt.Errorf("choose either --probe-line-pay-rule or --probe-scatter-pay-rule, not both")
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
	fmt.Printf("Scatter pays: %d\n", summary.ScatterPays)
	fmt.Printf("Total bet: %d\n", summary.TotalBet)
	fmt.Printf("Total line win: %d\n", summary.TotalLineWin)
	fmt.Printf("Total scatter win: %d\n", summary.TotalScatterWin)
	fmt.Printf("Total win: %d\n", summary.TotalWin)
	fmt.Printf("Line RTP: %.8f%%\n", ratio(summary.TotalLineWin, summary.TotalBet)*100)
	fmt.Printf("Scatter RTP: %.8f%%\n", ratio(summary.TotalScatterWin, summary.TotalBet)*100)
	fmt.Printf("Total RTP: %.8f%%\n", ratio(summary.TotalWin, summary.TotalBet)*100)
	fmt.Printf("Hit count: %d\n", summary.HitCount)
	printPayHitSummary(summary)
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
	if len(summary.FirstScatterWins) > 0 {
		fmt.Println("First scatter wins:")
		for _, win := range summary.FirstScatterWins {
			fmt.Printf("  %s x%d pays %d\n", win.Symbol, win.Count, win.Payout)
		}
	}
	if len(summary.FirstLineWins) > 0 {
		fmt.Println("First line wins:")
		for _, win := range summary.FirstLineWins {
			fmt.Printf("  line %d: %-2s x%d pays %d [%s]\n", win.LineIndex, win.Symbol, win.Count, win.Payout, strings.Join(win.Symbols, " "))
		}
	}
	fmt.Printf("Status: %s\n", summary.Status)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printPayHitSummary(summary *app.SimulationSummary) {
	if len(summary.PayHits) == 0 {
		return
	}
	fmt.Println("Pay hit summary:")
	for _, hit := range summary.PayHits {
		probability := ratio(hit.Hits, int64(summary.Spins))
		if hit.ExpectedProbability == 0 {
			fmt.Printf(
				"  %-7s %-2s x%d pays %-4d hits %-8d probability %.8f expected - error - zScore -\n",
				hit.Kind,
				hit.Symbol,
				hit.Count,
				hit.Payout,
				hit.Hits,
				probability,
			)
			continue
		}
		zScore, hasZScore := probabilityZScore(probability, hit.ExpectedProbability, summary.Spins)
		fmt.Printf(
			"  %-7s %-2s x%d pays %-4d hits %-8d probability %.8f expected %.8f error % .4f%% zScore %s\n",
			hit.Kind,
			hit.Symbol,
			hit.Count,
			hit.Payout,
			hit.Hits,
			probability,
			hit.ExpectedProbability,
			relativeError(probability, hit.ExpectedProbability)*100,
			formatZScore(zScore, hasZScore),
		)
	}
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
	printProbeComparison(result.Probability, result.Rule.ExpectedProbability, result.Spins)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printScatterPayRuleProbe(result *probe.ScatterPayRuleResult, opts options, elapsed time.Duration) {
	fmt.Println("SlotMath scatter pay rule probe")
	fmt.Printf("Game path: %s\n", opts.GamePath)
	fmt.Printf("Spins: %d\n", result.Spins)
	printSeed(opts.Seed)
	fmt.Printf("Rule ID: %d\n", result.RuleID)
	fmt.Printf("Rule: %s x%d pays %d\n", result.Rule.Symbol, result.Rule.Count, result.Rule.Payout)
	fmt.Println("Wild: excluded")
	fmt.Printf("Hits: %d\n", result.Hits)
	fmt.Printf("Probability: %.8f\n", result.Probability)
	printProbeComparison(result.Probability, result.Rule.ExpectedProbability, result.Spins)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printProbeComparison(actual, expected float64, spins int) {
	if expected == 0 {
		fmt.Println("Expected: -")
		fmt.Println("Error: -")
		fmt.Println("Z-score: -")
		return
	}
	fmt.Printf("Expected: %.8f\n", expected)
	fmt.Printf("Error: % .4f%%\n", relativeError(actual, expected)*100)
	zScore, ok := probabilityZScore(actual, expected, spins)
	fmt.Printf("Z-score: %s\n", formatZScore(zScore, ok))
}

func ratio(numerator int64, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func relativeError(actual, expected float64) float64 {
	if expected == 0 {
		return 0
	}
	return (actual - expected) / expected
}

func probabilityZScore(actual, expected float64, spins int) (float64, bool) {
	if spins <= 0 || expected <= 0 || expected >= 1 {
		return 0, false
	}
	standardError := math.Sqrt(expected * (1 - expected) / float64(spins))
	if standardError == 0 {
		return 0, false
	}
	return (actual - expected) / standardError, true
}

func formatZScore(zScore float64, ok bool) string {
	if !ok {
		return "-"
	}
	return fmt.Sprintf("% .3f", zScore)
}

func printSeed(seed int64) {
	if seed == 0 {
		fmt.Println("Seed: random")
		return
	}
	fmt.Printf("Seed: %d\n", seed)
}
