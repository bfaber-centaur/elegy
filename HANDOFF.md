# Handoff

## Current question

Load the component table into the engine, then implement TAKEOVER.md.

## State

- Kernel, movement, battles and scanning are merged (elegy #1–#6).
- This branch adds `engine/catalog.go` and a verbatim copy of stars-elegy
  `data/components.json` (`engine/data/`). Parts, hulls, engines,
  planetary scanners and defenses, build rules and owner costs come from
  it. `docs/COMPONENTS-STATUS.md` maps rules, tests and unused columns.
- The FM movement corpus now takes its engine fuel tables from the table
  (CS-002) instead of third-party values.
- `go vet ./...` and `go test ./...` are green.

## What we know

- The table's CONFIRMED vectors pass: starbase build costs, defense
  coverage, token values from catalogue parts. COMBAT.md's battle-warp
  rule agrees with the table's (BINARY-ONLY) battle_warp column.
- Combat keeps one marked placeholder: A6, the plan-0 X on a turn's first
  location (COMBAT-STATUS.md).

## What still matters

- Columns for mines, remote mining, bombs, colonizing and stargates are
  loaded but unused (`deferredStats`); TAKEOVER.md needs bombs and
  colonizers next.
- KERNEL.md's BINARY-ONLY movement rules for IFE, Cheap Engines and others
  are still not applied (KERNEL-STATUS.md); the LRT fields now exist.

## Best next move

Implement TAKEOVER.md (bombing, invasion, colonization, capture) on top of
the catalogue's bomb and colonizer values.
