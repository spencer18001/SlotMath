package app

import (
	"fmt"
	"path/filepath"

	"slotmath/internal/config"
	"slotmath/internal/reels"
)

// GameData contains the loaded game definition needed to build a simulator.
type GameData struct {
	Path     string
	Config   config.GameConfig
	Reels    [][]string
	Paylines [][]int
	Paytable config.Paytable
}

// LoadGame is the top-level game loading entry point.
func LoadGame(gamePath string) (*GameData, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path is required")
	}

	cfg, err := config.LoadGameConfig(filepath.Join(gamePath, "config.json"))
	if err != nil {
		return nil, err
	}
	paytable, err := config.LoadPaytable(filepath.Join(gamePath, cfg.PaytableFile))
	if err != nil {
		return nil, err
	}
	paylines, err := config.LoadPaylines(filepath.Join(gamePath, cfg.PaylinesFile))
	if err != nil {
		return nil, err
	}
	reelSet, err := reels.LoadCSV(filepath.Join(gamePath, cfg.ReelsFile))
	if err != nil {
		return nil, err
	}

	data := &GameData{
		Path:     gamePath,
		Config:   cfg,
		Reels:    reelSet,
		Paylines: paylines,
		Paytable: paytable,
	}
	if err := validateGameData(data); err != nil {
		return nil, err
	}
	return data, nil
}

func validateGameData(data *GameData) error {
	cfg := data.Config
	if cfg.GameID == "" {
		return fmt.Errorf("config gameId is required")
	}
	if cfg.Bet <= 0 {
		return fmt.Errorf("bet must be greater than zero")
	}
	if cfg.NumReels <= 0 {
		return fmt.Errorf("numReels must be greater than zero")
	}
	if cfg.NumRows <= 0 {
		return fmt.Errorf("numRows must be greater than zero")
	}
	if len(data.Reels) != cfg.NumReels {
		return fmt.Errorf("reels count %d does not match numReels %d", len(data.Reels), cfg.NumReels)
	}
	for reelIndex, reel := range data.Reels {
		if len(reel) < cfg.NumRows {
			return fmt.Errorf("reel %d has %d stops, needs at least numRows %d", reelIndex, len(reel), cfg.NumRows)
		}
	}

	for lineIndex, line := range data.Paylines {
		if len(line) != cfg.NumReels {
			return fmt.Errorf("payline %d has %d entries, expected %d", lineIndex, len(line), cfg.NumReels)
		}
		for reelIndex, row := range line {
			if row < 0 || row >= cfg.NumRows {
				return fmt.Errorf("payline %d reel %d row %d is outside 0..%d", lineIndex, reelIndex, row, cfg.NumRows-1)
			}
		}
	}
	for index, pay := range data.Paytable.Line {
		if pay.Symbol == "" {
			return fmt.Errorf("line pay %d symbol is required", index)
		}
		if pay.Count <= 0 {
			return fmt.Errorf("line pay %d count must be greater than zero", index)
		}
		if pay.Payout < 0 {
			return fmt.Errorf("line pay %d payout cannot be negative", index)
		}
	}
	return nil
}
