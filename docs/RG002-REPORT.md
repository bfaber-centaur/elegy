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

Every order is accounted for. Lane F's coverage check (`/mnt/project-files/rg002/orders-coverage.md`) found that all 1867 order records in the 42 order files decode into the vectors, and it lists the two known vector omissions.

Two tests check the ledger exactly: `internal/vectors` `TestRG002Ledger` for fields and orders, and `engine` `TestBattleRecords` for battle records. Either one fails on a line the run produces that the ledger lacks, and on a ledger line the run no longer produces. So a fix fails until its lines leave the ledger, and nothing is skipped.

Run on: elegy main `546dacc` plus this change. The final run on the integrated main SHA is still to come and will replace these numbers.

## Classes

These classes are computed:
- **sample**: one of two things.
  - An expectation the vector marks as one stream's random outcome.
  - Anything in the chained run after its first year. Those years follow Elegy's own draws (docs/RANDOM-PARITY.md "Samples"), and that includes the chained run's later order refusals and battle records.
- **random-driven**: a field that misses on the reference seed but matches on at least one of Elegy's other 7 seed variants, so Elegy's draws decide it. The line says how many variants match. For a restart's battle record, it is lane D's in-year rule: Elegy drew in that year before its battles, and at least one other seed variant fights the recorded battle. The line keeps the count of matching variants, so a change in that count fails too.
- **not compared**: an expectation or field the harness does not compare, with the reason.

These are classified by hand, with an owner. They cover a field that misses on all 8 variants, an order Elegy refuses, and a strict battle record that fails:
- **missing mechanic**;
- **incorrect mechanic**;
- **state-import error**;
- **unsupported order**;
- **unresolved randomness**, for a field that misses on all 8 variants by an amount that varies between them, when its owner argues it is draw-driven all the same. The class carries the owner and a one-line reason (`unresolved randomness [lane D]: reason`), and the test refuses it on a line whose value is the same on every variant. No line has it yet.

A field that misses on all 8 variants by a varying amount says so in its detail: "(0/8 variants match, values vary)". Its value depends on Elegy's draws, but no draw gets it right, so it is a mechanic difference until its owner shows otherwise.

## Counts

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| incorrect mechanic | 64 | 9 | 2 |
| missing mechanic | 3 | 0 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| unresolved randomness | 0 | 0 | 0 |
| random-driven | 40 | 0 | 0 |
| not compared | 457 | 0 | 0 |
| sample | 1746 | 62 | 18 |

No order fails to load. Lane F's coverage check found that the 42 order files hold 1867 order records and the vectors carry 1867 orders, so a refused order is always Elegy's gap, never the decoder's. Two fields of the host files are not in the vectors, and their meaning is UNRESOLVED: the lay-mines task's second word and the high nibble of the waypoint target type. The ledger header lists both. Of the restarts' orders, Elegy refuses 9. All 9 are the same mechanic (below). The chained run refuses 62 orders, all samples, from years 3–6.

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
  - Y4 fleet 1/12: no waypoint after waypoint 0, and its position and waypoint 0 (values vary).
  - Y4 fleet 2/13: x and waypoint 0 (values vary).
  - Y5 fleets 2/18 and 2/20: waypoint 1 at (1363,1323), where the original has (1335,1320), the Y5 battle's place (values vary).
  - The Y2 battle at (1367,1327) and the Y5 battle at (1335,1320), which Elegy does not fight. Under lane D's rule, no other seed variant fights either one, so both are deterministic.

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
- **Planet 35's surface minerals** (values vary): 37–52 iron too many and up to 37 too many of each other mineral, in chained 2463 and restarts Y1, Y3, Y4 and Y5.
- **Y5.**
  - Player 6's energy research is 59 against 23, and 5 other fields are 10 too high (values vary).
  - Planet 35's queue keeps an item the original finished.
  - Fleet 6/4 has 2 ships of design 45 instead of 3, and fuel 560 instead of 840 (values vary). Inferred: the missing ship is planet 35's build.
- **Player 2's research.** It is off by 1 or 2 in some fields in chained 2463 and in Y1, Y3 and Y4, on all 8 variants. Three of those lines vary with the seed and the rest do not. The cause is not established. RG-001 shows similar ±1s that follow mining draws.

## Not compared, by reason

None of the 457 lines is an RG-002 mechanic Elegy lacks. Each falls in one of three groups.

**1. The harness has no comparator, but Elegy models the quantity (lane B).**

| Reason | Lines | What Elegy has |
|---|---|---|
| planet `packet_destination`, `packet_warp` | 110 | `Planet.HasPacketDest`, `PacketDest`, `PacketSpeed` (the mass-driver setting) |
| `object` wormhole (the whole-object form) | 72 | wormholes; only the `wormhole` expectation kind is compared, not `object` |
| `message` | 68 | turn events; there is no event-to-message mapping in the harness |
| `starbase_design`, design reads ("needs the order layer's design source") | 14 | designs and design orders; the design read is not wired into the harness |
| player `counts`, `research_percent` | 8 | the player's research percentage and per-design counts |
| planet starbase in a design slot the comparator cannot place (`starbase design 0`) | 1 | the starbase; the slot lookup misses after a design change |

**2. Compared by another test.** The 176 `battle` and `battle_actions` expectations are compared record by record by TestBattleRecords. Its RG-002 results are the ledger's battle record lines.

**3. No value in the original's file.** The vectors write null where the file holds nothing to compare: 6 lines of planet `defenses` null and 2 of `packet_destination` null.

Group 1 is comparator work. Until it is done, a mismatch in those 273 lines would go unseen, so their count is the honest size of the gap.

## Random-driven

Each of these lines matches on at least one of the 8 seed variants.
- **Restarts:** 34 field lines. They are mostly mining and surface minerals, minefield decay, production percentages, and fleets whose movement rides on random merges, battles and target picks. The breakdown is Y1 6, Y3 7, Y4 9 and Y5 12.
- **Chained year 1:** 6 more field lines.

Until 5c1ca55, a line counted as random-driven whenever its value differed between variants, even when no variant matched. Those 23 lines (planet 35's minerals, player 2's and player 6's research, planet 35's queue, and fleets 1/12, 2/13, 2/18, 2/20 and 6/4) are now incorrect-mechanic lines with their owners, above.
