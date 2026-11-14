package simulator_test

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"slotmath/flow"
	"slotmath/simulator"
	"slotmath/spin"
)

// go test ./simulator -run TestGameRTP -v -count=1
func TestGameRTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping RTP simulation in short mode")
	}

	const (
		spins       = 10_000_000
		seed  int64 = 0
	)

	testCases := []struct {
		game        string
		bet         int64
		freeSpins   int
		expectedRTP float64
	}{
		{game: "test1", bet: 1, expectedRTP: 63.18},
		{game: "test2", bet: 1, expectedRTP: 163.09},
		{game: "test3", bet: 1, expectedRTP: 104.76},
		{game: "test4", bet: 50, expectedRTP: 134.15},
		{game: "test5", bet: 1, expectedRTP: 96.85},
		{game: "test6", bet: 1, expectedRTP: 172.50},
		{game: "test7", bet: 1, freeSpins: 10, expectedRTP: 26533.96},
	}

	for _, testCase := range testCases {
		t.Run(testCase.game, func(t *testing.T) {
			engine, err := spin.Load(filepath.Join("..", "games", testCase.game), seed)
			require.NoError(t, err)

			sim := simulator.New(flow.New(engine))
			summary, err := sim.Run(simulator.RunRequest{
				Spins: spins, Bet: testCase.bet, FreeSpins: testCase.freeSpins,
			})
			require.NoError(t, err)

			actualRTP := float64(summary.TotalWin) / float64(summary.TotalBet) * 100
			t.Logf("RTP: %.2f%%, expected: %.2f%%", actualRTP, testCase.expectedRTP)

			// 最少允許 ±1 個百分點
			// 高 RTP 時允許理論值的 ±0.1%
			tolerance := math.Max(1, testCase.expectedRTP*0.001)
			require.InDelta(t, testCase.expectedRTP, actualRTP, tolerance)
		})
	}
}
