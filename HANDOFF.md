# Handoff

## Current question

Grow the engine from the public stars-elegy specification. The peaceful
kernel and ordinary fleet movement from stars-elegy `docs/KERNEL.md` (on `main`)
are implemented; the next work is either closing the spec gaps or the next
subsystem the specification covers.

## State

- Branch: `claude/project-thread-8c4kqn` (PR open, not merged).
- `engine/` holds habitability, population, economy, mining, research,
  production, movement and `GenerateTurn`. `docs/KERNEL-STATUS.md` is the
  map: rules, tests, statuses, gaps.
- `go test ./...` is green. `go test -run Confirmed ./...` runs only ground
  truth; `TestPrediction*` tests pin BINARY-ONLY rules.

## What we know

- Every CONFIRMED vector in KERNEL.md passes, plus the full public oracle
  data behind them: PG-001..003 population/carry 2400–2436, PG research
  and mining, all 15 PQ-001 production cases, and all 224 fleets of
  FM-001..004 (copied into `engine/testdata/fm`).
- Three KERNEL.md/corpus disagreements found here (auto-item caps, running
  dry with exact fuel, mutual chases) are corrected upstream in stars-elegy
  PR #13; the code follows that text. Remaining open questions and the
  code's interim choices are in KERNEL-STATUS.md.
- `PlayerOrders` and `Ruleset` are still stubs; RNG is injected as
  `engine.Rand`.

## What still matters

- When stars-elegy PR #13 merges, re-point the KERNEL-STATUS.md citation at
  `main`. Send the remaining open questions back to stars-elegy.
- Not modelled: orders, waypoint tasks, colonization, ships and starbases
  in production, terraforming, remote mining, random events, combat, and
  the BINARY-ONLY LRT/engine movement rules.

## Best next move

When KERNEL.md answers a gap, flip the matching test or code. Otherwise add
the next specified subsystem (waypoint tasks and colonization are the
natural step toward an expansion slice) with the same Confirmed/Prediction
test split.
