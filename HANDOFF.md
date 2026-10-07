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
  `a6ac76e`, which includes the implementer answers merged in #24.
- `go vet ./...` and `go test ./...` are green. `go test -run Confirmed
  ./...` runs the CB vectors.

## What we know

- Every CB vector in COMBAT.md/PARITY.md that needs no part catalogue
  passes, including the 132% capacitor of CB-002 C4. Capacitors compound
  per item, as current COMBAT.md says.
- Where COMBAT.md is silent or unsettled, the code makes nine explicit
  assumptions or placeholders (A1–A9). They are marked in the code and
  listed in COMBAT-STATUS.md. None is claimed as Stars! behavior.

## What still matters

- Spec decisions are needed for A4 (destroyed ships' fuel) and A8 (IT vs
  IS repair wording). The other items only matter for untested cases
  (three or more players, dumped cargo, the Bleeding Edge flag).
- No part catalogue yet. Plug the public component table into `Design`
  when stars-elegy `docs/COMPONENTS.md` lands.

## Best next move

Once #5 merges, open the scanning PR from a fresh branch off main. Then
fold in spec answers for A1–A9 as they land.
