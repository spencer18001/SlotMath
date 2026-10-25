package spin

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 遊戲模式
type Mode string

const (
	ModeBase Mode = "base"
	ModeFree Mode = "free"
)

type GameSetting struct {
	path        string
	baseSetting GameBaseSetting
	paylines    [][]int // [line][reel] -> rowIndex
	paytable    Paytable
	reels       map[Mode][][]string // [mode][reel][stop] -> symbol
}

type GameBaseSetting struct {
	GameID      string
	Description string
	BetPerLine  int64 // 押一條線要幾分 (可以由當次總押注算出押幾條線)
	WayPayBet   int64
	NumReels    int // 盤面有幾軸
	NumRows     int // 盤面每軸有幾列
	DrawMode    string
	Cascading   bool
	WildSymbols []string
}

type Paytable struct {
	Line    []PaytableEntry
	Way     []PaytableEntry
	Scatter []PaytableEntry
}

type PaytableEntry struct {
	Index     int `json:"-"`
	Symbol    string
	Count     int
	Odds      int64
	FreeSpins int
}

func Load(gamePath string, seed int64) (*Engine, error) {
	gameSetting, err := loadGameSetting(gamePath)
	if err != nil {
		return nil, err
	}
	return newEngine(gameSetting, seed)
}

func loadGameSetting(gamePath string) (GameSetting, error) {
	if gamePath == "" {
		return GameSetting{}, fmt.Errorf("game path is required")
	}

	var cfg GameBaseSetting
	if err := loadJSON(filepath.Join(gamePath, "config.json"), &cfg); err != nil {
		return GameSetting{}, err
	}
	var paylines [][]int
	if err := loadJSON(filepath.Join(gamePath, "paylines.json"), &paylines); err != nil {
		return GameSetting{}, err
	}
	var paytable Paytable
	if err := loadJSON(filepath.Join(gamePath, "paytable.json"), &paytable); err != nil {
		return GameSetting{}, err
	}
	setPaytableIndexes(&paytable)

	baseReels, err := loadReelsCSV(filepath.Join(gamePath, "reels.csv"))
	if err != nil {
		return GameSetting{}, fmt.Errorf("load base reels: %w", err)
	}
	reels := map[Mode][][]string{ModeBase: baseReels}

	freeReels, err := loadReelsCSV(filepath.Join(gamePath, "reels_free.csv"))
	if err == nil {
		reels[ModeFree] = freeReels
	} else if !errors.Is(err, os.ErrNotExist) {
		return GameSetting{}, fmt.Errorf("load free reels: %w", err)
	}

	gameSetting := GameSetting{path: gamePath, baseSetting: cfg, paylines: paylines, paytable: paytable, reels: reels}
	if err := validateGameSetting(gameSetting); err != nil {
		return GameSetting{}, err
	}
	return gameSetting, nil
}

func setPaytableIndexes(paytable *Paytable) {
	for index := range paytable.Line {
		paytable.Line[index].Index = index
	}
	for index := range paytable.Way {
		paytable.Way[index].Index = index
	}
	for index := range paytable.Scatter {
		paytable.Scatter[index].Index = index
	}
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

func loadReelsCSV(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open reels csv %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read reels csv %s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("reels csv %s is empty", path)
	}

	numReels := len(records[0])
	if numReels == 0 {
		return nil, fmt.Errorf("reels csv %s has no reels", path)
	}

	reels := make([][]string, numReels)
	for rowIndex, record := range records {
		if len(record) != numReels {
			return nil, fmt.Errorf("reels csv %s row %d has %d columns, expected %d", path, rowIndex+1, len(record), numReels)
		}

		for reelIndex, value := range record {
			symbol := strings.TrimSpace(value)
			if symbol == "" {
				continue
			}

			reels[reelIndex] = append(reels[reelIndex], symbol)
		}
	}

	for reelIndex, reel := range reels {
		if len(reel) == 0 {
			return nil, fmt.Errorf("reels csv %s reel %d is empty", path, reelIndex)
		}
	}

	return reels, nil
}

func validateGameSetting(gameSetting GameSetting) error {
	baseSetting := gameSetting.baseSetting
	if baseSetting.GameID == "" {
		return fmt.Errorf("config GameID is required")
	}
	if baseSetting.BetPerLine <= 0 {
		return fmt.Errorf("BetPerLine must be greater than zero")
	}
	drawMode, err := parseDrawMode(baseSetting.DrawMode)
	if err != nil {
		return err
	}
	if baseSetting.Cascading && drawMode != drawModeReelStrips {
		return fmt.Errorf("Cascading requires DrawMode %q", drawModeReelStrips)
	}
	if baseSetting.NumReels <= 0 {
		return fmt.Errorf("NumReels must be greater than zero")
	}
	if baseSetting.NumRows <= 0 {
		return fmt.Errorf("NumRows must be greater than zero")
	}

	for lineIndex, line := range gameSetting.paylines {
		if len(line) != baseSetting.NumReels {
			return fmt.Errorf("payline %d has %d entries, expected %d", lineIndex, len(line), baseSetting.NumReels)
		}
		for reelIndex, row := range line {
			if row < 0 || row >= baseSetting.NumRows {
				return fmt.Errorf("payline %d reel %d row %d is outside 0..%d", lineIndex, reelIndex, row, baseSetting.NumRows-1)
			}
		}
	}

	if err := validatePaytableEntries("line", gameSetting.paytable.Line); err != nil {
		return err
	}
	for index, pay := range gameSetting.paytable.Line {
		if pay.FreeSpins != 0 {
			return fmt.Errorf("line pay %d cannot award FreeSpins", index)
		}
	}
	if err := validatePaytableEntries("way", gameSetting.paytable.Way); err != nil {
		return err
	}
	if len(gameSetting.paytable.Way) > 0 && baseSetting.WayPayBet <= 0 {
		return fmt.Errorf("WayPayBet must be greater than zero when way pays are configured")
	}
	for index, pay := range gameSetting.paytable.Way {
		if pay.FreeSpins != 0 {
			return fmt.Errorf("way pay %d cannot award FreeSpins", index)
		}
	}
	if err := validatePaytableEntries("scatter", gameSetting.paytable.Scatter); err != nil {
		return err
	}
	for _, pay := range gameSetting.paytable.Scatter {
		if pay.FreeSpins > 0 {
			if _, ok := gameSetting.reels[ModeFree]; !ok {
				return fmt.Errorf("free reels are required when scatter pays award FreeSpins")
			}
			break
		}
	}

	if len(gameSetting.reels) == 0 {
		return fmt.Errorf("config reels is required")
	}
	if _, ok := gameSetting.reels[ModeBase]; !ok {
		return fmt.Errorf("base reels are required")
	}
	for mode, reel := range gameSetting.reels {
		if mode != ModeBase && mode != ModeFree {
			return fmt.Errorf("unsupported reel mode %q", mode)
		}
		if len(reel) != baseSetting.NumReels {
			return fmt.Errorf("%s reels count %d does not match NumReels %d", mode, len(reel), baseSetting.NumReels)
		}
		for reelIndex, reel := range reel {
			if len(reel) < baseSetting.NumRows {
				return fmt.Errorf("%s reel %d has %d stops, needs at least NumRows %d", mode, reelIndex, len(reel), baseSetting.NumRows)
			}
		}
	}
	return nil
}

func validatePaytableEntries(kind string, entries []PaytableEntry) error {
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
