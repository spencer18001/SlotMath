package spin

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// loadReelsCSV reads a reel-strip CSV where each column is one reel and each row is one stop.
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
