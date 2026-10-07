# Handoff

## Current question

Grow the engine from the public stars-elegy specification. The peaceful
kernel and ordinary fleet movement from `docs/KERNEL.md` (stars-elegy PR #12)
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
- Where KERNEL.md and the corpus disagree or KERNEL.md is silent, the code
  follows the corpus and the choice is listed in KERNEL-STATUS.md "Spec
  gaps" (auto-item caps, running dry with exact fuel, mutual chases).
- `PlayerOrders` and `Ruleset` are still stubs; RNG is injected as
  `engine.Rand`.

## What still matters

- Send the KERNEL-STATUS.md spec gaps back to stars-elegy so KERNEL.md can
  be corrected or an experiment run.
- Not modelled: orders, waypoint tasks, colonization, ships and starbases
  in production, terraforming, remote mining, random events, combat, and
  the BINARY-ONLY LRT/engine movement rules.

## Best next move

When KERNEL.md answers a gap, flip the matching test or code. Otherwise add
the next specified subsystem (waypoint tasks and colonization are the
natural step toward an expansion slice) with the same Confirmed/Prediction
test split.
