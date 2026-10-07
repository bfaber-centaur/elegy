# Handoff

## Current question

Implement the public combat specification (stars-elegy `docs/COMBAT.md`,
PR #19) as a battle phase of `GenerateTurn`, in the kernel's style.

## State

- Kernel and movement are merged (elegy #1–#4, following stars-elegy
  through #18).
- This branch adds the battle phase: `engine/ships.go` (parts, hulls),
  `battleplan.go`, `combat_*.go`, and `docs/COMBAT-STATUS.md` (rules,
  statuses, tests, choices). It merges after stars-elegy #19.
- `go test ./...` is green; `go test -run Confirmed ./...` runs the
  CB-000..CB-019 vectors.

## What we know

- Every CB vector stated in COMBAT.md/PARITY.md that needs no part
  catalogue passes: token values, hit chances and 202-torpedo hit counts,
  missile damage and one kill per missile, misses on shields, beam
  dropoff (starbase uses the part range), carry R', gatling, sappers,
  target choice, starbase damage steps, salvage, repair rates, plan-0
  joining, and the same-turn tech gain of CB-018.
- Choices where COMBAT.md is silent are listed in COMBAT-STATUS.md and
  were raised with the spec author.

## What still matters

- Answers from the spec author may flip a default (COMBAT-STATUS.md
  "Choices"); the capacitor 132% vs per-item compounding is the one that
  contradicts a CONFIRMED number.
- Round 3 (R-8..R-10) may change BINARY-ONLY statuses.
- No part catalogue: CB battles cannot be replayed end to end publicly.

## Best next move

Fold in COMBAT.md corrections as they land. After that, waypoint tasks,
colonization and ships in production move toward a playable slice.
