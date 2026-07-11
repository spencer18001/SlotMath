package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadGameConfig(path string) (GameConfig, error) {
	var cfg GameConfig
	if err := loadJSON(path, &cfg); err != nil {
		return GameConfig{}, err
	}
	return cfg, nil
}

func LoadPaytable(path string) (Paytable, error) {
	var paytable Paytable
	if err := loadJSON(path, &paytable); err != nil {
		return Paytable{}, err
	}
	return paytable, nil
}

func LoadPaylines(path string) ([][]int, error) {
	var paylines [][]int
	if err := loadJSON(path, &paylines); err != nil {
		return nil, err
	}
	return paylines, nil
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
