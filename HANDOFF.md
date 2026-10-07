# Handoff

## Current question

Implement TAKEOVER.md (bombing, invasion, colonization, capture) on top of
the component table.

## State

- Kernel, movement, battles, scanning and the component table are merged
  (elegy #1–#7).
- This branch (elegy #8) adds `engine/takeover.go`, following stars-elegy
  TAKEOVER.md with the answers in stars-elegy #34 (order inside a phase,
  unload amounts, capture tech) and the corrections in #37 (deep-space
  unloads; colonize is tried once).
  `docs/TAKEOVER-STATUS.md` maps rules, tests and assumptions.
- Fleets now carry a waypoint-0 task (`Fleet.Task`, `Waypoint.Task`):
  transport unload actions and colonize.
- `go vet ./...` and `go test ./...` are green. Every TK vector that
  Elegy's state can express is a `TestConfirmed*` test.

## What we know

- One Elegy assumption remains (TAKEOVER-STATUS.md T1): away from a
  planet, cargo stays aboard where another fleet or salvage shares the
  position, since Elegy's task does not record the waypoint's target.
- Combat keeps one placeholder: A6, the plan-0 X on a turn's first
  location (COMBAT-STATUS.md).

## What still matters

- Load actions, merges, transfers, scrap and remote mining are not
  modelled; TAKEOVER.md and #34 specify them.
- An Alternate Reality colony gets no starbase yet (designs have no owner),
  so the next year is refused.
- KERNEL.md's BINARY-ONLY movement rules for IFE, Cheap Engines and others
  are still not applied (KERNEL-STATUS.md).

## Best next move

Universe objects (stars-elegy OBJECTS.md), then turn orders (ORDERS.md),
which would also give designs an owner.
