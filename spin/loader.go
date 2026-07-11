package spin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type gameConfig struct {
	GameID         string   `json:"gameId"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	BetPerLine     int64    `json:"betPerLine"`
	NumReels       int      `json:"numReels"`
	NumRows        int      `json:"numRows"`
	ReelsFile      string   `json:"reelsFile"`
	PaylinesFile   string   `json:"paylinesFile"`
	PaytableFile   string   `json:"paytableFile"`
	WildSymbols    []string `json:"wildSymbols"`
	ScatterSymbols []string `json:"scatterSymbols"`
}

type loadedGame struct {
	path     string
	config   gameConfig
	reels    [][]string
	paylines [][]int
	paytable Paytable
}

func Load(gamePath string, seed int64) (*Engine, error) {
	data, err := loadGame(gamePath)
	if err != nil {
		return nil, err
	}
	return newEngine(data, seed)
}

func loadGame(gamePath string) (loadedGame, error) {
	if gamePath == "" {
		return loadedGame{}, fmt.Errorf("game path is required")
	}
	var cfg gameConfig
	if err := loadJSON(filepath.Join(gamePath, "config.json"), &cfg); err != nil {
		return loadedGame{}, err
	}
	var paytable Paytable
	if err := loadJSON(filepath.Join(gamePath, cfg.PaytableFile), &paytable); err != nil {
		return loadedGame{}, err
	}
	var paylines [][]int
	if err := loadJSON(filepath.Join(gamePath, cfg.PaylinesFile), &paylines); err != nil {
		return loadedGame{}, err
	}
	reelSet, err := loadReelsCSV(filepath.Join(gamePath, cfg.ReelsFile))
	if err != nil {
		return loadedGame{}, err
	}
	data := loadedGame{path: gamePath, config: cfg, reels: reelSet, paylines: paylines, paytable: paytable}
	if err := validateGame(data); err != nil {
		return loadedGame{}, err
	}
	return data, nil
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func validateGame(data loadedGame) error {
	cfg := data.config
	if cfg.GameID == "" {
		return fmt.Errorf("config gameId is required")
	}
	if cfg.BetPerLine <= 0 {
		return fmt.Errorf("betPerLine must be greater than zero")
	}
	if cfg.NumReels <= 0 {
		return fmt.Errorf("numReels must be greater than zero")
	}
	if cfg.NumRows <= 0 {
		return fmt.Errorf("numRows must be greater than zero")
	}
	if len(data.reels) != cfg.NumReels {
		return fmt.Errorf("reels count %d does not match numReels %d", len(data.reels), cfg.NumReels)
	}
	for reelIndex, reel := range data.reels {
		if len(reel) < cfg.NumRows {
			return fmt.Errorf("reel %d has %d stops, needs at least numRows %d", reelIndex, len(reel), cfg.NumRows)
		}
	}
	for lineIndex, line := range data.paylines {
		if len(line) != cfg.NumReels {
			return fmt.Errorf("payline %d has %d entries, expected %d", lineIndex, len(line), cfg.NumReels)
		}
		for reelIndex, row := range line {
			if row < 0 || row >= cfg.NumRows {
				return fmt.Errorf("payline %d reel %d row %d is outside 0..%d", lineIndex, reelIndex, row, cfg.NumRows-1)
			}
		}
	}
	if err := validatePayEntries("line", data.paytable.Line); err != nil {
		return err
	}
	return validatePayEntries("scatter", data.paytable.Scatter)
}

func validatePayEntries(kind string, entries []PayEntry) error {
	for index, pay := range entries {
		if pay.Symbol == "" {
			return fmt.Errorf("%s pay %d symbol is required", kind, index)
		}
		if pay.Count <= 0 {
			return fmt.Errorf("%s pay %d count must be greater than zero", kind, index)
		}
		if pay.Odds < 0 {
			return fmt.Errorf("%s pay %d odds cannot be negative", kind, index)
		}
		if pay.ExpectedProbability < 0 {
			return fmt.Errorf("%s pay %d expectedProbability cannot be negative", kind, index)
		}
	}
	return nil
}
