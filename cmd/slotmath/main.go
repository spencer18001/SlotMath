package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"slotmath/internal/app"
)

type options struct {
	GamePath string
	Spins    int
	Seed     int64
}

func main() {
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
	summary, err := game.RunSims(opts.Spins, opts.Seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slotmath: run sims: %v\n", err)
		os.Exit(1)
	}

	printSummary(summary)
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.GamePath, "game", "games/sample_lines", "path to a game definition folder")
	flag.IntVar(&opts.Spins, "spins", 100000, "number of spins to simulate")
	flag.Int64Var(&opts.Seed, "seed", 0, "base random seed; 0 means random")
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
	return nil
}

func printSummary(summary *app.SimulationSummary) {
	fmt.Println("SlotMath line-game simulator")
	fmt.Printf("Game ID: %s\n", summary.GameID)
	fmt.Printf("Game path: %s\n", summary.GamePath)
	fmt.Printf("Spins: %d\n", summary.Spins)
	if summary.Seed == 0 {
		fmt.Println("Seed: random")
	} else {
		fmt.Printf("Seed: %d\n", summary.Seed)
	}
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
}
