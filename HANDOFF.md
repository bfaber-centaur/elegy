# Handoff

## Current question

Land the battle phase (elegy PR #5), which implements stars-elegy
`docs/COMBAT.md`. Then open the scanning PR, which is ready in a local
commit and implements `docs/SCANNING.md`.

## State

- Kernel and movement are merged (elegy #1–#4).
- This branch adds the battle phase: `engine/ships.go` (parts, hulls,
  design cost), `battleplan.go`, `combat_*.go`, and
  `docs/COMBAT-STATUS.md`. It follows COMBAT.md on stars-elegy `main` at
  `a6ac76e` plus open #31 (answers to the A-list) and the owner-cost rule
  of COMPONENTS.md in open #30. Merge order: stars-elegy #30, #31, then
  elegy #5.
- `go vet ./...` and `go test ./...` are green. `go test -run Confirmed
  ./...` runs the CB vectors.

## What we know

- Every CB vector in COMBAT.md/PARITY.md that needs no part catalogue
  passes, including the 132% capacitor of CB-002 C4. Capacitors compound
  per item, as current COMBAT.md says.
- Where COMBAT.md is unsettled, one explicit placeholder remains: A6
  (plan-0 X on a turn's first location, an open experiment). It is
  marked in the code and listed in COMBAT-STATUS.md. The others are now
  cited rules, including the observer tech-attempt LEGACY BUG (switch
  `legacyObserverTechMask`).

## What still matters

- A6 only matters with three or more players.
- No part catalogue yet. Plug the public component table into `Design`
  when stars-elegy `docs/COMPONENTS.md` lands.

## Best next move

Once #5 merges, open the scanning PR from a fresh branch off main. Then
plug the component table (stars-elegy #30) into `Design`.
