# Handoff

## Current question

Grow the engine from the public stars-elegy specification. The peaceful
kernel and ordinary fleet movement from stars-elegy `docs/KERNEL.md` (on `main`)
are implemented; the next work is either closing the spec gaps or the next
subsystem the specification covers.

## State

- Merged to `main`: elegy PR #1 (kernel), PR #2 (stars-elegy PR #14
  rules) and PR #3 (KX-001: Auto Alchemy before a ×n item, item costs, AR
  resources, the AR zero-maximum decision). The follow-up for stars-elegy
  PR #18 (limiting component; KX-001 A5–A7) is on branch
  `claude/project-thread-8c4kqn`, in review; it should merge after #18.
- `engine/` holds habitability, population, economy, mining, research,
  production, movement and `GenerateTurn`. `docs/KERNEL-STATUS.md` is the
  map: rules, tests, statuses, gaps.
- `go test ./...` is green. `go test -run Confirmed ./...` runs only ground
  truth; `TestPrediction*` tests pin BINARY-ONLY rules.

## What we know

- Every CONFIRMED vector in KERNEL.md passes, plus the full public oracle
  data behind them: PG-001..003 population/carry 2400–2436, PG research
  and mining, all 15 PQ-001 production cases, KX-001 (Auto Alchemy, item
  costs, AR), and all 224 fleets of
  FM-001..004 (copied into `engine/testdata/fm`).
- Three KERNEL.md/corpus disagreements found here (auto-item caps, running
  dry with exact fuel, mutual chases) are corrected upstream in stars-elegy
  PR #13 (merged); the code follows that text. PR #14 and PR #16 (KX-001)
  settled every open question. The one Elegy decision (an AR planet with
  maximum population 0 makes `GenerateTurn` return
  `*ZeroMaxPopulationError`) and the code's remaining small choices are in
  KERNEL-STATUS.md.
- `PlayerOrders` and `Ruleset` are still stubs; RNG is injected as
  `engine.Rand`.

## What still matters

- Bobby has not yet picked the rule for the AR zero-maximum state; the
  typed error is the recommended default. If he picks another, change
  `checkGenerable`.
- Combat is the likely next subsystem here, once stars-elegy has a public
  combat spec; build it only from that public text, like the kernel.
- Not modelled: orders, waypoint tasks, colonization, ships and starbases
  in production, terraforming, remote mining, random events, combat, and
  the BINARY-ONLY LRT/engine movement rules.

## Best next move

When KERNEL.md answers a gap, flip the matching test or code. When the
public combat spec lands, implement it with the same Confirmed/Prediction
test split; otherwise waypoint tasks and colonization are the natural step
toward an expansion slice.
