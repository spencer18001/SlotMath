package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"slotmath/flow"
	"slotmath/simulator"
	"slotmath/spin"
)

type options struct {
	GamePath                 string
	Spins                    int
	Seed                     int64
	Bet                      int64
	ProbeLinePayRuleIndex    int
	ProbeScatterPayRuleIndex int
	HasLineRuleProbe         bool
	HasScatterRuleProbe      bool
}

func main() {
	startedAt := time.Now()
	opts := parseFlags()
	if err := validateOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: %v\n", err)
		os.Exit(1)
	}

	game, err := spin.Load(opts.GamePath, opts.Seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: load game: %v\n", err)
		os.Exit(1)
	}

	if opts.HasLineRuleProbe {
		result, err := simulator.RunLinePayRuleProbe(game, opts.Spins, opts.ProbeLinePayRuleIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printLinePayRuleProbe(result, opts, time.Since(startedAt))
		return
	}
	if opts.HasScatterRuleProbe {
		result, err := simulator.RunScatterPayRuleProbe(game, opts.Spins, opts.ProbeScatterPayRuleIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printScatterPayRuleProbe(result, opts, time.Since(startedAt))
		return
	}

	bet := opts.Bet
	if bet == 0 {
		bet = game.DefaultBet().Total
	}
	gameFlow := flow.New(game)
	sim := simulator.New(gameFlow)
	summary, err := sim.Run(simulator.Request{Spins: opts.Spins, Bet: bet})
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: run sims: %v\n", err)
		os.Exit(1)
	}
	printSummary(game, summary, time.Since(startedAt))
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.GamePath, "game", "games/sample_lines", "path to a game definition folder")
	flag.IntVar(&opts.Spins, "spins", 100000, "number of spins to simulate")
	flag.Int64Var(&opts.Seed, "seed", 0, "base random seed; 0 means random")
	flag.Int64Var(&opts.Bet, "bet", 0, "total bet per spin; 0 activates all configured paylines")
	flag.IntVar(&opts.ProbeLinePayRuleIndex, "probe-line-pay-rule", -1, "paytable.line rule index to probe on payline 0; -1 disables probe")
	flag.IntVar(&opts.ProbeScatterPayRuleIndex, "probe-scatter-pay-rule", -1, "paytable.scatter rule index to probe; -1 disables probe")
	flag.Parse()
	opts.HasLineRuleProbe = opts.ProbeLinePayRuleIndex >= 0
	opts.HasScatterRuleProbe = opts.ProbeScatterPayRuleIndex >= 0
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
	if opts.HasLineRuleProbe && opts.HasScatterRuleProbe {
		return fmt.Errorf("choose either --probe-line-pay-rule or --probe-scatter-pay-rule, not both")
	}
	return nil
}

func printSummary(game *spin.Game, summary *simulator.Summary, elapsed time.Duration) {
	info := game.Info()
	paytable := game.Paytable()
	fmt.Println("SlotMath line-game simulator")
	fmt.Printf("Game ID: %s\n", info.GameID)
	fmt.Printf("Game path: %s\n", info.Path)
	fmt.Printf("Spins: %d\n", summary.Spins)
	printSeed(info.Seed)
	fmt.Printf("Reels: %d\n", info.ReelCount)
	fmt.Printf("Paylines: %d\n", info.PaylineCount)
	fmt.Printf("Line pays: %d\n", len(paytable.Line))
	fmt.Printf("Scatter pays: %d\n", len(paytable.Scatter))
	fmt.Printf("Bet per line: %d\n", summary.Bet.PerLine)
	fmt.Printf("Active lines: %d\n", summary.Bet.ActiveLines)
	fmt.Printf("Bet per spin: %d\n", summary.Bet.Total)
	fmt.Printf("Total bet: %d\n", summary.TotalBet)
	fmt.Printf("Total line win: %d\n", summary.TotalLineWin)
	fmt.Printf("Total scatter win: %d\n", summary.TotalScatterWin)
	fmt.Printf("Total win: %d\n", summary.TotalWin)
	fmt.Printf("Line RTP: %.8f%%\n", ratio(summary.TotalLineWin, summary.TotalBet)*100)
	fmt.Printf("Scatter RTP: %.8f%%\n", ratio(summary.TotalScatterWin, summary.TotalBet)*100)
	fmt.Printf("Total RTP: %.8f%%\n", ratio(summary.TotalWin, summary.TotalBet)*100)
	fmt.Printf("Hit count: %d\n", summary.HitCount)
	printPayHitSummary(summary)
	if summary.First != nil {
		fmt.Printf("First stops: %v\n", summary.First.Stops)
		fmt.Println("First board:")
		for _, row := range summary.First.Board.Rows() {
			fmt.Print("  ")
			for index, symbol := range row {
				if index > 0 {
					fmt.Print(" | ")
				}
				fmt.Printf("%2s", symbol)
			}
			fmt.Println()
		}
		fmt.Printf("First win: %d\n", summary.First.TotalWin)
		if len(summary.First.ScatterWins) > 0 {
			fmt.Println("First scatter wins:")
			for _, win := range summary.First.ScatterWins {
				rule := paytable.Scatter[win.PayRuleIndex]
				fmt.Printf("  %s x%d odds %d payout %d\n", rule.Symbol, rule.Count, rule.Odds, win.Payout)
			}
		}
		if len(summary.First.LineWins) > 0 {
			fmt.Println("First line wins:")
			for _, win := range summary.First.LineWins {
				rule := paytable.Line[win.PayRuleIndex]
				fmt.Printf("  line %d: %-2s x%d odds %d payout %d\n", win.LineIndex, rule.Symbol, rule.Count, rule.Odds, win.Payout)
			}
		}
	}
	fmt.Printf("Status: %s\n", summary.Status)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printPayHitSummary(summary *simulator.Summary) {
	if len(summary.PayHits) == 0 {
		return
	}
	fmt.Println("Pay hit summary (line pays use payline 0):")
	for _, hit := range summary.PayHits {
		probability := ratio(hit.Hits, int64(summary.Spins))
		if hit.ExpectedProbability == 0 {
			fmt.Printf(
				"  %-7s %-2s x%d odds %-4d hits %-8d probability %.8f expected - z-score -\n",
				hit.Kind, hit.Symbol, hit.Count, hit.Odds, hit.Hits, probability,
			)
			continue
		}
		zScore, hasZScore := probabilityZScore(probability, hit.ExpectedProbability, summary.Spins)
		fmt.Printf(
			"  %-7s %-2s x%d odds %-4d hits %-8d probability %.8f expected %.8f z-score %s\n",
			hit.Kind, hit.Symbol, hit.Count, hit.Odds, hit.Hits, probability,
			hit.ExpectedProbability, formatZScore(zScore, hasZScore),
		)
	}
}

func printLinePayRuleProbe(result *simulator.LinePayRuleProbeResult, opts options, elapsed time.Duration) {
	fmt.Println("SlotMath line pay rule probe")
	fmt.Printf("Game path: %s\n", opts.GamePath)
	fmt.Printf("Spins: %d\n", result.Spins)
	printSeed(opts.Seed)
	fmt.Println("Payline index: 0")
	fmt.Printf("Payline: %v\n", result.Payline)
	fmt.Printf("Pay rule index: %d\n", result.PayRuleIndex)
	fmt.Printf("Rule: %s x%d odds %d\n", result.Rule.Symbol, result.Rule.Count, result.Rule.Odds)
	fmt.Println("Wild: included")
	fmt.Printf("Hits: %d\n", result.Hits)
	fmt.Printf("Probability: %.8f\n", result.Probability)
	printProbeComparison(result.Probability, result.Rule.ExpectedProbability, result.Spins)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printScatterPayRuleProbe(result *simulator.ScatterPayRuleProbeResult, opts options, elapsed time.Duration) {
	fmt.Println("SlotMath scatter pay rule probe")
	fmt.Printf("Game path: %s\n", opts.GamePath)
	fmt.Printf("Spins: %d\n", result.Spins)
	printSeed(opts.Seed)
	fmt.Printf("Pay rule index: %d\n", result.PayRuleIndex)
	fmt.Printf("Rule: %s x%d odds %d\n", result.Rule.Symbol, result.Rule.Count, result.Rule.Odds)
	fmt.Println("Wild: excluded")
	fmt.Printf("Hits: %d\n", result.Hits)
	fmt.Printf("Probability: %.8f\n", result.Probability)
	printProbeComparison(result.Probability, result.Rule.ExpectedProbability, result.Spins)
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printProbeComparison(actual, expected float64, spins int) {
	if expected == 0 {
		fmt.Println("Expected: -")
		fmt.Println("Z-score: -")
		return
	}
	fmt.Printf("Expected: %.8f\n", expected)
	zScore, ok := probabilityZScore(actual, expected, spins)
	fmt.Printf("Z-score: %s\n", formatZScore(zScore, ok))
}

func ratio(numerator int64, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
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
