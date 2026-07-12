# SlotMath

SlotMath provides three public layers for line and scatter slot math:

```text
simulator -> flow -> spin
```

- `spin`: draws and evaluates one stateless spin using the requested reel mode.
- `flow`: advances a complete round, including base-game triggers and free-game retriggers.
- `simulator`: runs paid rounds repeatedly and reports RTP and pay-hit statistics by mode.

The current math rules are line pays and scatter pays. New evaluators can be added when a game requires additional rules.

## Structure

```text
SlotMath/
|-- spin/              # Stateless spin math
|-- flow/              # Round and free-spin state
|-- simulator/         # Simulation and probability probes
|-- cmd/slotmath/      # CLI and report output
`-- games/fg/          # Base/free reel example
```

## Game definition

A game folder contains its engine configuration, reel strips, paylines, and paytable:

```text
games/fg/
|-- config.json
|-- reels_base.csv
|-- reels_free.csv
|-- paylines.json
`-- paytable.json
```

`config.json` maps each spin mode to its reel strip:

```json
{
  "gameId": "fg",
  "description": "Free Game Sample",
  "betPerLine": 1,
  "numReels": 5,
  "numRows": 3,
  "reelFiles": {
    "base": "reels_base.csv",
    "free": "reels_free.csv"
  },
  "paylinesFile": "paylines.json",
  "paytableFile": "paytable.json",
  "wildSymbols": ["W"],
  "scatterSymbols": ["S"]
}
```

Reel CSV columns may have different lengths. An empty trailing cell means that reel has no stop on that row.

A scatter rule may award free spins. `spin.Result.FreeSpins` is the number newly awarded by that spin; remaining free spins belong to `flow.State`.

```json
{
  "symbol": "S",
  "count": 3,
  "odds": 2,
  "freeSpins": 10,
  "expectedProbabilities": {
    "base": 0.024384375,
    "free": 0.025063815789473684
  }
}
```

## Basic usage

```go
engine, err := spin.Load("games/fg", seed)
if err != nil {
	return err
}

slotFlow := flow.New(engine)
sim := simulator.New(slotFlow)
summary, err := sim.Run(simulator.Request{
	Spins: 1_000_000, // paid base rounds
	Bet:   5,
})
```

Callers can also drive one complete round directly:

```go
state := flow.State{Bet: 5}
for !state.Completed {
	step, err := slotFlow.Next(state)
	if err != nil {
		return err
	}
	state = step.State
}
```

The initial zero-value mode means `base`. When a spin awards free spins, the flow switches to `free` and continues until `FreeSpinsRemaining` reaches zero. A retrigger is applied as:

```text
remaining = remaining - 1 + spinResult.FreeSpins
```

## Betting and RTP

The configured `betPerLine` is the betting unit. Total bet activates the first `bet / betPerLine` paylines.

- Line wins pay `odds * betPerLine`.
- Scatter wins pay `odds * totalBet`.
- Free spins use the original round bet for payout calculation but do not increase the paid `TotalBet`.

Simulator reports are separated into `base` and `free`:

- Mode RTP uses `mode spins * bet` as its denominator.
- Free RTP contribution uses paid `TotalBet` as its denominator.
- Overall RTP uses all base/free wins divided by paid `TotalBet`.
- Pay-hit probability and z-score use the spin count and expected probability for that mode.
- Line-pay hit summaries observe payline 0 only.

## Randomness

Each engine owns its random stream. Seed `0` selects a random seed; a non-zero seed makes a run reproducible. An engine instance should not be used concurrently.

## CLI

Run complete paid rounds, including all triggered free games:

```text
go run ./cmd/slotmath -game fg -spins 1000000 -seed 1 -bet 5
```

`-game` is the folder name under `games/`, so use `-game fg` rather than `-game games/fg`.

Useful flags:

```text
-spins     number of paid base spins or pattern-probe spins
-seed      base random seed; 0 means random
-bet       total bet per spin; 0 activates all configured paylines
-v         show detailed pay-hit summaries
```

Run raw board-pattern probes. Pattern probes draw boards directly in one reel mode and do not run the free-spin flow. Quote patterns because PowerShell treats `|` as a pipeline:

```text
go run ./cmd/slotmath -game fg -spins 10000000 -pattern "line.WK|WK|WK|WK|!WK" -expected 0.00032
go run ./cmd/slotmath -game fg -spins 10000000 -pattern "scatter.S|S|S|!S|!S" -expected 0.002438438
go run ./cmd/slotmath -game line -spins 10000000 -pattern "scatter.-|S|S|S|-" -expected 0.003375
```

Pattern-probe flags:

```text
-pattern   pattern to test, prefixed by line. or scatter.
-mode      reel mode: base or free
-line      payline index for line patterns
-expected  expected probability; 0 disables z-score comparison
```

For line patterns, each condition checks the selected payline (`-line 0` by default). For scatter patterns, each condition checks the complete visible reel window. `-` accepts anything, a concatenated token such as `WK` accepts either configured symbol, and `!WK` excludes both.
