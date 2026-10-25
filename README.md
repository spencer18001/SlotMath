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
|-- simulator/         # Simulation and probability reports
|-- cmd/slotmath/      # CLI and report output
`-- games/fg/          # Base/free reel example
```

## Game definition

A game folder contains its engine configuration, reel strips, paylines, and paytable:

```text
games/fg/
|-- config.json
|-- reels.csv
|-- reels_free.csv
|-- paylines.json
`-- paytable.json
```

Game files use fixed names: `reels.csv` contains the base-game reel strips, `paylines.json` contains paylines, and `paytable.json` contains the paytable. Add `reels_free.csv` when the game has free-game reel strips.

```json
{
  "GameID": "fg",
  "Description": "Free Game Sample",
  "BetPerLine": 1,
  "NumReels": 5,
  "NumRows": 3,
  "WildSymbols": ["W"]
}
```

Reel CSV columns may have different lengths. An empty trailing cell means that reel has no stop on that row.

Set `Cascading` to `true` for tumbling/cascading games. A spin still appears as one completed `Engine.Spin` result to callers, but internally it repeats evaluate -> remove winning positions -> refill from reel strips until no win remains. Cascade steps do not add bet; they are reported separately as `CascadeSteps`.
`DrawMode` controls how a visible board is drawn:

- Omitted or `reelStrips`: each reel draws one stop, then visible rows are consecutive symbols from that reel strip.
- `independentRows`: each visible row independently draws one stop from that reel's strip. This models independent-reel samples where the same column uses one strip distribution, but every cell is sampled independently.

A scatter rule may award free spins. `spin.Result.FreeSpins` is the number newly awarded by that spin; remaining free spins belong to `flow.State`.

```json
{
  "Symbol": "S",
  "Count": 3,
  "Odds": 2,
  "FreeSpins": 10
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

The configured `BetPerLine` is the betting unit. Total bet activates the first `bet / BetPerLine` paylines.

- Line wins pay `Odds * BetPerLine`.
- Scatter wins pay `odds * totalBet`.
- Free spins use the original round bet for payout calculation but do not increase the paid `TotalBet`.

Simulator reports are separated into `base` and `free`:

- Mode RTP uses `mode spins * bet` as its denominator.
- Free RTP contribution uses paid `TotalBet` as its denominator.
- Overall RTP uses all base/free wins divided by paid `TotalBet`.
- Pay-hit probability uses the spin count for that mode.
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
-spins     number of paid base spins
-seed      base random seed; 0 means random
-bet       total bet per spin; 0 activates all configured paylines
```

