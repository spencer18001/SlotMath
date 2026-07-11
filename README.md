# SlotMath

A small Go slot math simulator project.

Initial goal: simulate a basic line game from reel strips, paylines, and a paytable, then report RTP and related statistics.

## Planned Structure

```text
SlotMath/
├─ go.mod
├─ README.md
├─ cmd/
│  └─ slotmath/
├─ internal/
│  ├─ config/
│  ├─ reels/
│  ├─ board/
│  ├─ evaluator/
│  ├─ simulation/
│  └─ report/
├─ games/
│  └─ sample_lines/
└─ tests/
```

## First Milestone

```text
reels.csv + paylines + paytable
-> draw natural boards
-> evaluate line wins
-> simulate N spins
-> report RTP / hit rate / max win / volatility
```

No implementation yet. This repository is currently only the project skeleton.
