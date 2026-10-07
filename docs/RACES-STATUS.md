# Race design: implementation status

`races/` implements stars-elegy `docs/RACES.md` (stars-elegy PR #49,
`5637e34`): the advantage points rule, repairs and legality at game
creation, the wizard's Random race, and the yearly check of a running
game. `newgame/` applies the creation rules to every player. Test vectors
come from the public RD corpus (`experiments/rd/races.tsv`, copied to
`races/testdata/`) and the PARITY.md "Race design" records.

## Tests

| Rule (RACES.md) | Code | Ground truth |
|---|---|---|
| Advantage points | `points.go` | every RD race with a points value (RD-1..RD-4, 60 races) and the default race (25) |
| Repairs | `points.go` | RD-4 d, e, f (centre, stat 15, growth 0); RD-P8 (low −5 → 0–85, centre 42) |
| At game creation | `check.go`, `newgame/players.go` | RD-4 b, c (illegal → default race, L 25); computer races unchecked; L cap; spends 5 and 6 |
| Yearly check and penalty | `check.go` | RD-P1..RD-P10 on the PG001 race (245 points): points after the edit, punished or not, colonists, growth and points after |
| Random race | `random.go` | outcome only: 500 Elegy-seeded races all score 0..50 and keep their name |

RD-P3's trait word `a000006d` is read as IFE, ARM, ISB, UR, MA, the
"expensive fields start at 3" bit (29) and the germanium bit (31); that
reading gives exactly the corpus's −3667 and 1457 points.

## Assumptions (RACES.md leaves these open)

Marked `ASSUMPTION` in the code and sent to the spec author:

1. RACES.md says an axis with a value outside 0..100 is made immune
   (BINARY-ONLY), but RD-P8 clamped a low of −5 instead. Elegy clamps and
   never makes an axis immune in a repair.
2. Colonists per resource that are not a multiple of 100 are truncated to
   one; a research cost outside the three settings becomes normal.
3. The yearly check does nothing to a computer player's race beyond the
   silent clamps (RACES.md names only human players for the punishment).
4. Random race adjust step: research, economy and growth move in a
   direction picked by a fair coin, with no change at a limit; an LRT is
   toggled; "a setting among colonists, factories and mines" is uniform
   over the seven economy settings.
5. Random race economy: colonists per resource uniform over 700..2500 in
   steps of 100.

## Not wired yet

- The yearly check (`races.YearlyCheck`) is not called by
  `engine.GenerateTurn`: the engine's `Player` has no race design, name or
  tampered flag yet. Wiring it needs that shared-type change and belongs
  with the turn engine.
- Race files, their checksums and the corrupt-file refusal: Elegy has no
  race files.
- The wizard's own limits (20-wide ranges, AR resets, no saving below 0)
  are user-interface rules (BINARY-ONLY).
