# Handoff

## Current question

Land per-player knowledge (scanning), which implements stars-elegy
`docs/SCANNING.md`, on top of the merged battle phase.

## State

- Kernel, movement and battles are merged (elegy #1–#5). Combat follows
  stars-elegy `main` at `df57443` (`docs/COMBAT-STATUS.md`).
- This branch adds `engine/scanning.go`: a pure `Views` function from the
  post-turn game to one `PlayerView` per player, plus
  `PopulationEstimates`. `GenerateTurn` returns the views in
  `TurnResult.Views`. `docs/SCANNING-STATUS.md` maps rules, tests and gaps.
- `go vet ./...` and `go test ./...` are green. `go test -run Confirmed
  ./...` runs the CB and SC vectors.

## What we know

- Every SC vector that needs no part catalogue passes. BINARY-ONLY rules
  (population and defense estimates, headings, AR planet scanners) are
  `TestPrediction*` tests. The co-location LEGACY BUG is behind
  `legacyColocation`.
- Combat keeps one marked placeholder: A6, the plan-0 X on a turn's first
  location, which is an open experiment in COMBAT.md.

## What still matters

- Scanner, cloak and planetary-scanner values are caller-supplied. The
  component table (stars-elegy `data/components.json`, `COMPONENTS.md`)
  has merged and can now fill `Design`, `Game.PlanetScanners` and
  `Game.Defenses`.
- Space objects (minefields, packets, wormholes, Mystery Trader,
  stargates) are specified in stars-elegy `OBJECTS.md` and SCANNING.md
  "Space objects", but Elegy has none yet.

## Best next move

Load the component table into the engine. Then implement TAKEOVER.md
(bombing, invasion, colonization) and the universe objects.
