package spin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type gameConfig struct {
	GameID         string
	Description    string
	BetPerLine     int64
	WayPayBet      int64
	DrawMode       string
	Cascading      bool
	NumReels       int
	NumRows        int
	ReelFiles      map[Mode]string
	PaylinesFile   string
	PaytableFile   string
	WildSymbols    []string
	ScatterSymbols []string
}

type loadedGame struct {
	path     string
	config   gameConfig
	reels    map[Mode][][]string // [mode][reel][stop] -> symbol
	paylines [][]int             // [line][reel] -> row index
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
	reelFiles := cfg.ReelFiles
	reelSets := make(map[Mode][][]string, len(reelFiles))
	for mode, file := range reelFiles {
		reelSet, err := loadReelsCSV(filepath.Join(gamePath, file))
		if err != nil {
			return loadedGame{}, fmt.Errorf("load %s reels: %w", mode, err)
		}
		reelSets[mode] = reelSet
	}
	data := loadedGame{path: gamePath, config: cfg, reels: reelSets, paylines: paylines, paytable: paytable}
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
		return fmt.Errorf("config GameID is required")
	}
	if cfg.BetPerLine <= 0 {
		return fmt.Errorf("BetPerLine must be greater than zero")
	}
	drawMode, err := parseDrawMode(cfg.DrawMode)
	if err != nil {
		return err
	}
	if cfg.Cascading && drawMode != drawModeReelStrips {
		return fmt.Errorf("Cascading requires DrawMode %q", drawModeReelStrips)
	}
	if cfg.NumReels <= 0 {
		return fmt.Errorf("NumReels must be greater than zero")
	}
	if cfg.NumRows <= 0 {
		return fmt.Errorf("NumRows must be greater than zero")
	}
	if len(data.reels) == 0 {
		return fmt.Errorf("config reels is required")
	}
	if _, ok := data.reels[ModeBase]; !ok {
		return fmt.Errorf("base reels are required")
	}
	for mode, reelSet := range data.reels {
		if mode != ModeBase && mode != ModeFree {
			return fmt.Errorf("unsupported reel mode %q", mode)
		}
		if len(reelSet) != cfg.NumReels {
			return fmt.Errorf("%s reels count %d does not match NumReels %d", mode, len(reelSet), cfg.NumReels)
		}
		for reelIndex, reel := range reelSet {
			if len(reel) < cfg.NumRows {
				return fmt.Errorf("%s reel %d has %d stops, needs at least NumRows %d", mode, reelIndex, len(reel), cfg.NumRows)
			}
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
	for index, pay := range data.paytable.Line {
		if pay.FreeSpins != 0 {
			return fmt.Errorf("line pay %d cannot award FreeSpins", index)
		}
	}
	if err := validatePayEntries("way", data.paytable.Way); err != nil {
		return err
	}
	if len(data.paytable.Way) > 0 && cfg.WayPayBet <= 0 {
		return fmt.Errorf("WayPayBet must be greater than zero when way pays are configured")
	}
	for index, pay := range data.paytable.Way {
		if pay.FreeSpins != 0 {
			return fmt.Errorf("way pay %d cannot award FreeSpins", index)
		}
	}
	if err := validatePayEntries("scatter", data.paytable.Scatter); err != nil {
		return err
	}
	for _, pay := range data.paytable.Scatter {
		if pay.FreeSpins > 0 {
			if _, ok := data.reels[ModeFree]; !ok {
				return fmt.Errorf("free reels are required when scatter pays award FreeSpins")
			}
			break
		}
	}
	return nil
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
		if pay.FreeSpins < 0 {
			return fmt.Errorf("%s pay %d FreeSpins cannot be negative", kind, index)
		}
	}
	return nil
}
