# RG-002 regression report

RG-002 is a state-edited late-game AIX start in 2463 (stars-elegy `vectors/rg/rg002*.json`, synced unchanged from stars-elegy main). It has seven runs:
- the chained six years, 2463→2468;
- each year's restart from the recorded state (Y1..Y6).

Every recorded order is loaded, and no year is left out.

The ledger, [`engine/testdata/vectors/rg/rg002-ledger.txt`](../engine/testdata/vectors/rg/rg002-ledger.txt), lists every difference the reference run (seed variant 0) has from the original, one line each. A line covers one of these:
- a mismatched field;
- a field the harness does not compare;
- a refused order;
- a battle record that does not match.

Two tests check the ledger exactly: `internal/vectors` `TestRG002Ledger` for fields and orders, and `engine` `TestBattleRecords` for battle records. Either one fails on a line the run produces that the ledger lacks, and on a ledger line the run no longer produces. So a fix fails until its lines leave the ledger, and nothing is skipped.

Run on: elegy main `cf41aab` plus #188 and this change. The final run on the integrated main SHA is still to come and will replace these numbers.

## Classes

These classes are computed:
- **sample**: one of two things.
  - An expectation the vector marks as one stream's random outcome.
  - Anything in the chained run after its first year. Those years follow Elegy's own draws (docs/RANDOM-PARITY.md "Samples"), and that includes the chained run's later order refusals and battle records.
- **random-driven**: a field whose result differs between Elegy's 8 seed variants, so Elegy's draws decide it. The line says how many variants match.
- **not compared**: an expectation or field the harness does not compare, with the reason.

These are classified by hand, with an owner. They cover a field that differs the same way on all 8 variants, an order Elegy refuses, and a strict battle record that fails:
- **missing mechanic**;
- **incorrect mechanic**;
- **state-import error**;
- **unsupported order**;
- **random-driven**: used by hand for a strict battle record that follows a random-driven field.

## Counts

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| incorrect mechanic | 41 | 9 | 1 |
| missing mechanic | 3 | 0 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| random-driven | 63 | 0 | 1 (hand) |
| not compared | 457 | 0 | 0 |
| sample | 1746 | 62 | 18 |

No order fails to load. Of the restarts' orders, Elegy refuses 9. All 9 are the same mechanic (below). The chained run refuses 62 orders, all samples, from years 3–6.

## Deterministic lines, by owner

### Lane A: a waypoint aimed at a fleet merged away earlier in the replay (incorrect mechanic)

- **Refused orders:**
  - Y2: player 1 orders 39, 42, 59 and 72, aimed at 2/14.
  - Y4: player 1 orders 30, 35 and 45, aimed at 2/15.
  - Y5: player 1 orders 45 and 75, aimed at 6/8.
- **The cause.** Each target fleet is emptied that year by its owner's `move_ships` into another fleet at the same spot. Elegy refuses the later waypoint order with "no such object".
- **The spec.** stars-elegy ORDERS.md, "A fleet target merged away during order replay" (BINARY-ONLY), says the waypoint is neither dropped nor rejected. The replay order then decides between two paths:
  - the start-of-year retarget to a co-located fleet of the owner, with draws;
  - the removal-time retarget, with no draws.
- **Lines that follow from it (inferred):**
  - Y2 fleet 1/57: position and waypoint 1.
  - Y4 fleet 1/12: no waypoint after waypoint 0.
  - The Y2 battle at (1367,1327), which Elegy does not fight.

### Lane A: retarget of a removed target fleet (missing mechanic)

- **The lines.**
  - Y3 fleets 1/26 and 1/32: the waypoint at (1481,1304) should name planet 79 (Elegy id 80). Elegy has deep space.
  - Y4 fleet 1/60: the waypoint at (1293,1597) should name planet 51 (Elegy id 52).
- **The spec.** ORDERS.md "Targets that moved, died or were captured", the B-first bullet: removing a fleet retargets its waypoints to what stands at its position. Elegy applies the plain "target gone" rule instead.

### Economy: planet 35's production and player research (incorrect mechanic)

- **Y1 and chained 2463.**
  - Fleet 6/12 has 4 ships instead of 5, and fuel 1120 instead of 1400.
  - Player 6's research is +8 per field and +26 energy. That is consistent with one ship's resources going to research.
- **Y4.**
  - The planet-35 ship is missing. Fleet 6/12 is the planet-46 build, which the original numbered 15.
  - Fleet 6/15 is reported gone.
- **Y5.** Player 6's energy research is 59 against 23.
- **Player 2's research.** It is off by 1 or 2 in some fields in Y1, Y3 and Y4, the same on all 8 variants. The cause is not established. RG-001 shows similar ±1s that follow mining draws, but here they do not vary with the seed.

## Not compared (lane B comparator gaps)

| Reason | Lines |
|---|---|
| planet `packet_destination` (incl. null) and `packet_warp` | 112 |
| `battle`, `battle_actions` expectations | 176 |
| `object` wormhole | 72 |
| `message` | 68 |
| `starbase_design` and design reads | 15 |
| player `counts`, `research_percent` | 8 |
| planet `defenses` null | 6 |

The 176 `battle` and `battle_actions` expectations are compared record by record by TestBattleRecords, and its RG-002 results are the battle lines above. The other gaps are comparator work for lane B.

## Random-driven

- **Restarts:** 55 field lines. They are mostly mining and surface minerals, minefield decay, production percentages, and fleets whose movement rides on random merges, battles and target picks. The breakdown is Y1 8, Y3 9, Y4 15 and Y5 23.
- **Chained year 1:** 8 more field lines.
- **The Y5 battle at (1335,1320).** Elegy does not fight it on the reference seed because fleets 2/18 and 2/20 end elsewhere, and their positions match on 5 of 8 variants. Lane D is reviewing this record's class.
