# Race design: implementation status

`races/` implements stars-elegy `docs/RACES.md` (stars-elegy PR #49,
`bf303f4`): the advantage points rule, repairs and legality at game
creation, the wizard's Random race, and the yearly check of a running
game. `newgame/` applies the creation rules to every player. Test vectors
come from the public RD corpus (`experiments/rd/races.tsv`, copied to
`races/testdata/`) and the PARITY.md "Race design" records.

## Tests

| Rule (RACES.md) | Code | Ground truth |
|---|---|---|
| Advantage points | `points.go` | every RD race with a points value (RD-1..RD-4, 60 races) and the default race (25) |
| Repairs | `points.go` | RD-4 d, e, f (centre, stat 15, growth 0); RD-P8 (low −5 → 0–85, centre 42); immune marker and research clamp as predictions |
| At game creation | `check.go`, `newgame/players.go` | RD-4 b, c (illegal → default race, L 25); computer races unchecked; L cap; spends 5 and 6 |
| Yearly check and penalty | `check.go` | RD-P1..RD-P10 on the PG001 race (245 points): points after the edit, punished or not, colonists, growth and points after; computer races repaired and flagged, never punished (prediction) |
| Random race | `random.go` | outcome only: 500 Elegy-seeded races all score 0..50 and keep their name |

RD-P3's trait word `a000006d` is IFE, ARM, ISB, UR, MA, the "expensive
fields start at 3" bit (29, confirmed by RACES.md "Race file") and the
germanium bit (31); it gives exactly the corpus's −3667 and 1457 points.

## Settled by stars-elegy #49 `bf303f4`

The five open points sent to the spec author are now cited rules:

1. Only the immune marker in the low makes an axis immune; any other
   out-of-range value is clamped (RD-P8). Elegy represents the marker as
   `ImmuneMarker` (−1) and an immune axis as `EnvRange{Immune: true}`.
2. Research costs outside the settings go to the nearest end; colonists
   per resource are stored in hundreds, so Elegy truncates.
3. A computer race in a running game gets the silent clamps and the
   scoring repair, which sets the tampered flag, but no penalty.
4. The Random race adjust step: one draw for the kind of change (3/10
   research, 3/10 LRT, 3/10 economy, 1/20 axis, 1/20 growth), one for the
   item, options tried in order, the first that brings the points strictly
   closer to 0..50 kept.
5. The Random race economy: the default economy with probability 1/3,
   otherwise each setting uniform over its range.

## Not wired yet

- The yearly check (`races.YearlyCheck`) is not called by
  `engine.GenerateTurn`: the engine's `Player` has no race design, name or
  tampered flag yet. Wiring it needs that shared-type change and belongs
  with the turn engine.
- Race files, their checksums and the corrupt-file refusal: Elegy has no
  race files.
- The wizard's own limits (20-wide ranges, AR resets, no saving below 0)
  are user-interface rules (BINARY-ONLY).
