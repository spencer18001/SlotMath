package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
	Pattern                  string
	Mode                     string
	LineIndex                int
	Expected                 float64
	Verbose                  bool
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
	if opts.Pattern != "" {
		result, err := simulator.RunPatternProbe(engine, opts.Spins, spin.Mode(opts.Mode), opts.LineIndex, opts.Pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run pattern probe: %v\n", err)
			os.Exit(1)
		}
		printPatternProbe(result, opts, time.Since(startedAt))
		return
	}

	if opts.HasLineRuleProbe {
		result, err := simulator.RunLinePayRuleProbe(engine, opts.Spins, opts.ProbeLinePayRuleIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printLinePayRuleProbe(result, opts, time.Since(startedAt))
		return
	}
	if opts.HasScatterRuleProbe {
		result, err := simulator.RunScatterPayRuleProbe(engine, opts.Spins, opts.ProbeScatterPayRuleIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "slotmath: run probe: %v\n", err)
			os.Exit(1)
		}
		printScatterPayRuleProbe(result, opts, time.Since(startedAt))
		return
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
	flag.IntVar(&opts.Spins, "spins", 100000, "number of spins to simulate")
	flag.Int64Var(&opts.Seed, "seed", 0, "base random seed; 0 means random")
	flag.Int64Var(&opts.Bet, "bet", 0, "total bet per spin; 0 activates all configured paylines")
	flag.IntVar(&opts.ProbeLinePayRuleIndex, "probe-line-pay-rule", -1, "paytable.line rule index to probe on payline 0; -1 disables probe")
	flag.IntVar(&opts.ProbeScatterPayRuleIndex, "probe-scatter-pay-rule", -1, "paytable.scatter rule index to probe; -1 disables probe")
	flag.StringVar(&opts.Pattern, "pattern", "", `board pattern, for example "line.WK|WK|WK|WK|!WK"`)
	flag.StringVar(&opts.Mode, "mode", string(spin.ModeBase), "reel mode for pattern probe: base or free")
	flag.IntVar(&opts.LineIndex, "line", 0, "payline index for a line pattern")
	flag.Float64Var(&opts.Expected, "expected", 0, "expected pattern probability; 0 disables z-score comparison")
	flag.BoolVar(&opts.Verbose, "v", false, "show detailed verbose pay hit summaries")
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
	probeCount := 0
	if opts.Pattern != "" {
		probeCount++
	}
	if opts.HasLineRuleProbe {
		probeCount++
	}
	if opts.HasScatterRuleProbe {
		probeCount++
	}
	if probeCount > 1 {
		return fmt.Errorf("choose only one pattern or pay-rule probe")
	}
	if opts.Mode != string(spin.ModeBase) && opts.Mode != string(spin.ModeFree) {
		return fmt.Errorf("mode must be base or free")
	}
	if opts.LineIndex < 0 {
		return fmt.Errorf("line index cannot be negative")
	}
	if opts.Expected < 0 || opts.Expected >= 1 {
		return fmt.Errorf("expected probability must be in [0, 1)")
	}
	return nil
}

func printPatternProbe(result *simulator.PatternProbeResult, opts options, elapsed time.Duration) {
	fmt.Println("SlotMath pattern probe")
	fmt.Printf("Game: %s\n", opts.GamePath)
	fmt.Printf("Mode: %s\n", result.Mode)
	fmt.Printf("Type: %s\n", result.Pattern.Kind)
	if result.Pattern.Kind == simulator.PatternLine {
		fmt.Printf("Payline index: %d\n", result.LineIndex)
		fmt.Printf("Payline: %v\n", result.Payline)
	}
	fmt.Printf("Pattern: %s\n", result.Pattern.Raw)
	fmt.Printf("Spins: %d\n", result.Spins)
	printSeed(opts.Seed)
	fmt.Printf("Hits: %d\n", result.Hits)
	fmt.Printf("Probability: %.8f\n", result.Probability)
	printProbeComparison(result.Probability, opts.Expected, result.Spins)
	fmt.Printf("Elapsed: %s\n", elapsed)
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
	fmt.Printf("Scatter pays: %d\n", len(paytable.Scatter))
	fmt.Printf("Bet per line: %d\n", summary.Bet.PerLine)
	fmt.Printf("Active lines: %d\n", summary.Bet.ActiveLines)
	fmt.Printf("Bet per spin: %d\n", summary.Bet.Total)
	fmt.Printf("Total bet: %d\n", summary.TotalBet)
	for _, modeSummary := range summary.Modes {
		printModeWinSummary(modeSummary, summary.Bet.Total, summary.TotalBet)
	}
	fmt.Printf("Overall line RTP: %.2f%%\n", ratio(summary.TotalLineWin, summary.TotalBet)*100)
	fmt.Printf("Overall scatter RTP: %.2f%%\n", ratio(summary.TotalScatterWin, summary.TotalBet)*100)
	fmt.Printf("Total RTP: %.2f%%\n", ratio(summary.TotalWin, summary.TotalBet)*100)
	fmt.Printf("Hit count: %d\n", summary.HitCount)
	if opts.Verbose {
		for _, modeSummary := range summary.Modes {
			printPayHitSummary(modeSummary)
		}
	}
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
		fmt.Printf("First free spins awarded: %d\n", summary.First.FreeSpins)
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
	fmt.Printf("Elapsed: %s\n", elapsed)
}

func printModeWinSummary(summary simulator.ModeSummary, betPerSpin int64, totalBet int64) {
	modeBet := int64(summary.Spins) * betPerSpin
	fmt.Printf("%s line RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalLineWin, modeBet)*100)
	fmt.Printf("%s scatter RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalScatterWin, modeBet)*100)
	fmt.Printf("%s RTP: %.2f%%\n", summary.Mode, ratio(summary.TotalWin, modeBet)*100)
	if summary.Mode != spin.ModeBase {
		fmt.Printf("%s RTP contribution: %.2f%%\n", summary.Mode, ratio(summary.TotalWin, totalBet)*100)
	}
}

func printPayHitSummary(summary simulator.ModeSummary) {
	if len(summary.PayHits) == 0 || summary.Spins == 0 {
		return
	}
	fmt.Printf("%s pay hit summary (line pays use payline 0):\n", summary.Mode)
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
	printProbeComparison(result.Probability, result.Rule.ExpectedProbabilityFor(spin.ModeBase), result.Spins)
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
	printProbeComparison(result.Probability, result.Rule.ExpectedProbabilityFor(spin.ModeBase), result.Spins)
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
