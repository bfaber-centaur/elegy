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

Run on: elegy main `e1798ac`, the merge of #191, with the full test set, the ledger and the 8-seed matrix. That run is in [PARITY-STATUS.md](PARITY-STATUS.md) "RG-002".

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
- **unresolved randomness**, for a field that misses on all 8 variants by an amount that varies between them, when its owner argues it is draw-driven all the same. The class carries the owner and a one-line reason (`unresolved randomness [lane D]: reason`), and the test refuses it on a line whose value is the same on every variant. Two lines have it: the variant-only refusal below.

A field that misses on all 8 variants by a varying amount says so in its detail: "(0/8 variants match, values vary)". Its value depends on Elegy's draws, but no draw gets it right, so it is a mechanic difference until its owner shows otherwise.

## Counts

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| incorrect mechanic | 0 | 0 | 0 |
| missing mechanic | 0 | 0 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| unresolved randomness | 0 | 2 | 0 |
| random-driven | 30 | 0 | 0 |
| not compared | 456 | 0 | 0 |
| sample | 976 | 43 | 16 |

Samples total 1035: 976 fields and 43 orders, checked by `TestRG002Ledger` (which logs them together as 1019), and 16 battle records, checked by `TestBattleRecords`.

The reference run of RG-002 has no deterministic discrepancy left. Every field, order and battle record of the seven runs either matches the original on the reference seed, matches on at least one other seed variant (random-driven), is a sample, or is not compared.

No order fails to load. Lane F's coverage check found that the 42 order files hold 1867 order records and the vectors carry 1867 orders, so a refused order is always Elegy's gap, never the decoder's. Two fields of the host files are not in the vectors, and their meaning is UNRESOLVED: the lay-mines task's second word and the high nibble of the waypoint target type. The ledger header lists both.

- **Restarts:** on the reference seed Elegy refuses none of the orders, and none of the 44 battle records fails.
- **Refusals on other variants:** the ledger lists an order refused on another seed variant but not on the reference one, outside the chained run's samples. There is one, in Y1 and in the chained run's year 1. Player 2's order 10 is a waypoint-0 follow of player 6's fleet 10, which player 6's `move_ships` empties; variants 1 and 6 replay player 6 first and refuse it. ORDERS.md "A fleet target merged away during order replay" covers only waypoints at index 1 or later, so both lines are `unresolved randomness [lane A]`.
- **Chained run:** 43 orders are refused in years 2–6, all samples, and 16 battle records miss, all samples.

## Effect of #189 and #190

"Before" is this change's ledger on main `546dacc`, which had neither fix. "After" is the same change rebased on `fcf16e0`, which has both: #189, lane A's merged-away and removed-fleet retargets, and #190, economy's remote-mining fix (only the remote-mining task mines).

| Class | Before: fields / orders / battles | After: fields / orders / battles |
|---|---|---|
| incorrect mechanic | 64 / 9 / 2 | 0 / 0 / 0 |
| missing mechanic | 3 / 0 / 0 | 0 / 0 / 0 |
| random-driven | 40 / 0 / 0 | 30 / 0 / 0 |
| not compared | 457 / 0 / 0 | 456 / 0 / 0 |
| sample | 1746 / 62 / 18 | 976 / 43 / 16 |

### Cleared deterministic lines

All 78 deterministic lines left those classes. 71 now match on the reference seed. 7 still miss there but match on some other seed variant, so they are random-driven now (listed last). The split below follows each line's owner. The two fixes were not measured separately.

**Lane A, #189 (merged-away and removed-fleet targets).**
- Refused orders, all now accepted:
  - Y2: player 1 orders 39, 42, 59 and 72, aimed at 2/14.
  - Y4: player 1 orders 30, 35 and 45, aimed at 2/15.
  - Y5: player 1 orders 45 and 75, aimed at 6/8.
- Battle records, now fought as recorded: Y2 (1367,1327) and Y5 (1335,1320).
- Fleets:
  - Y2 1/57: 4 lines (position and waypoint 1);
  - Y4 1/12: 4 lines;
  - Y4 2/13: 2 lines;
  - Y5 2/18 and 2/20: waypoint 1 at (1335,1320);
  - the missing-mechanic retargets: Y3 1/26 and 1/32 now name planet 79, and Y4 1/60 names planet 51.

**Economy, #190 (only the remote-mining task mines).**
- Planet 35's surface minerals: chained 2463, Y1, Y3, Y4 and Y5.
- Planet 35's build: fleet 6/12 in chained 2463, Y1 and Y4 (10 lines), fleet 6/15 in Y4, fleet 6/4 in Y5 (2 lines), and planet 35's Y5 queue.
- Player 6's research: chained 2463, Y1 and Y5 (18 lines).
- Player 2's research: chained 2463, Y1, Y3 and Y4 (15 lines). Its cause was not established before. These lines cleared along with planet 35's, so the ±1s probably followed planet 35's extra minerals and production (inferred).

**Now random-driven (7 of the 78).** Each of these matches on some variants after the fixes:
- Y3 player 2 biotechnology research (5/8 variants match);
- Y4 fleet 1/12 position and waypoint 0 (3 lines, 4/8);
- Y4 fleet 2/13 x and waypoint 0 (2 lines, 5/8);
- Y5 planet 35's surface minerals (5/8).

### Other lines that moved

- **Not compared:** chained 2468 planet 49's starbase in a design slot the comparator could not place cleared, since the chained run now has that design.
- **Random-driven:** of the 40 lines before, 2 are unchanged, and the other 38 cleared or changed their value or match count. Of the 30 now, 7 are the former deterministic lines above, and 21 are new or changed. The fixes change how many draws come before each later draw, so a different set of draw-decided fields lands on each variant. The new ones are in the Y2–Y5 restarts: mining and surface minerals, production queues, Y3 planet 42's population and queue, Y4 fleet 1/49 and planet 35's concentrations, and Y5 fleet 6/12's waypoint task.
- **Parity baseline:** RG-002-Y1-2463 and the chained run's RG-002-2463 now pass on 1 of 8 seed variants (`random 1` in engine/testdata/vectors/baseline.txt). Before, they failed on all 8.
- **Sample:** the chained run now follows the original much longer before Elegy's own draws take over, so 770 fewer field lines, 19 fewer refused orders and 2 fewer missed battle records are samples.

## Not compared, by reason

None of the 456 lines is an RG-002 mechanic Elegy lacks. Each falls in one of three groups.

**1. The harness has no comparator, but Elegy models the quantity (lane B).**

| Reason | Lines | What Elegy has |
|---|---|---|
| planet `packet_destination`, `packet_warp` | 110 | `Planet.HasPacketDest`, `PacketDest`, `PacketSpeed` (the mass-driver setting) |
| `object` wormhole (the whole-object form) | 72 | wormholes; only the `wormhole` expectation kind is compared, not `object` |
| `message` | 68 | turn events; there is no event-to-message mapping in the harness |
| `starbase_design`, design reads ("needs the order layer's design source") | 14 | designs and design orders; the design read is not wired into the harness |
| player `counts`, `research_percent` | 8 | the player's research percentage and per-design counts |

**2. Compared by another test.** The 176 `battle` and `battle_actions` expectations are compared record by record by TestBattleRecords. Its RG-002 results are the ledger's battle record lines.

**3. No value in the original's file.** The vectors write null where the file holds nothing to compare: 6 lines of planet `defenses` null and 2 of `packet_destination` null.

Group 1 is comparator work. Until it is done, a mismatch in those 272 lines would go unseen, so their count is the honest size of the gap.

## Random-driven

Each of these lines matches on at least one of the 8 seed variants.
- **Restarts:** 30 field lines: Y2 4, Y3 10, Y4 11 and Y5 5. They are mostly mining and surface minerals, production queues, and fleets whose movement rides on random merges, battles and target picks.
- **Chained year 1:** none.

Until 5c1ca55, a line counted as random-driven whenever its value differed between variants, even when no variant matched. On main `546dacc` there were 23 such lines, and they were classed as incorrect mechanics with their owners. All 23 are among the cleared lines above.
