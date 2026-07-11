# SlotMath

SlotMath provides three public layers for line and scatter slot math:

```text
simulator -> flow -> spin
```

- `spin`: draws and evaluates one independent line/scatter spin.
- `flow`: advances a round with `Next`; the current round completes after one spin.
- `simulator`: runs complete rounds repeatedly and aggregates RTP, pay hits, and probes.

The current scope intentionally includes only line pays and scatter pays. Additional
round state and rules should be added when a game actually requires them.

## Structure

```text
SlotMath/
├─ spin/                 # Public single-spin math
├─ flow/                 # Public round flow
├─ simulator/            # Public simulation and probe tools
├─ cmd/slotmath/         # CLI wiring and report output
└─ games/sample_lines/   # Sample line/scatter definition
```

## Basic usage

```go
engine, err := spin.Load("games/sample_lines", seed)
if err != nil {
	return err
}

slotFlow := flow.New(engine)
sim := simulator.New(slotFlow)
summary, err := sim.Run(simulator.Request{Spins: 1_000_000, Bet: 5})
```

For one round, callers can drive the flow directly:

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

The spin game owns its random stream. A seed of `0` selects a random seed; a
non-zero seed makes a run reproducible. A game instance should not be used
concurrently.
The configured `betPerLine` is the betting unit. A total bet activates the
first `bet / betPerLine` paylines. Line wins pay `odds * betPerLine`, while
scatter wins pay `odds * totalBet`.

The simulator's line pay hit summary records payline 0 only, so its probability
and z-score remain directly comparable with each rule's single-line
`expectedProbability`.

## CLI

```text
go run ./cmd/slotmath -game games/sample_lines -spins 100000 -seed 1 -bet 5
```

Rule probes remain available:

```text
go run ./cmd/slotmath -game games/sample_lines -spins 100000 -seed 1 -probe-line-pay-rule 0
go run ./cmd/slotmath -game games/sample_lines -spins 100000 -seed 1 -probe-scatter-pay-rule 0
```
