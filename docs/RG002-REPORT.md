# RG-002 regression report

RG-002 is a state-edited late-game AIX start in 2463 (stars-elegy `vectors/rg/rg002*.json`, synced unchanged from stars-elegy main). It has seven runs:
- the chained six years, 2463→2468;
- each year's restart from the recorded state (Y1..Y6).

Every recorded order is loaded, and no year is left out.

The ledger, [`engine/testdata/vectors/rg/rg002-ledger.txt`](../engine/testdata/vectors/rg/rg002-ledger.txt), lists every difference the reference run (seed variant 0) has from the original, one line each. A line covers one of these:
- a mismatched field;
- a field the harness does not compare;
- a refused order;
- a battle record, matched or not (one line each);
- a battle Elegy fights where the original recorded none.

Every order is accounted for. Lane F's coverage check (`/mnt/project-files/rg002/orders-coverage.md`) found that all 1867 order records in the 42 order files decode into the vectors, and it lists the two known vector omissions.

Two tests check the ledger exactly: `internal/vectors` `TestRG002Ledger` for fields and orders, and `engine` `TestBattleRecords` for battle records. Either one fails on a line the run produces that the ledger lacks, and on a ledger line the run no longer produces. So a fix fails until its lines leave the ledger, and nothing is skipped.

Run on: elegy main `8426a6a` plus this branch, which adds docs, the variant-refusal test change (internal/vectors tests and two ledger lines) and economy's comparator for planet `packet_destination` and `packet_warp` and player `research_percent` (internal/vectors), with the full test set, the ledger, the 8-seed matrix and the long-game matrix. That run is in [PARITY-STATUS.md](PARITY-STATUS.md) "RG-002".

## Classes

These classes are computed:
- **sample**: one of two things.
  - An expectation the vector marks as one stream's random outcome.
  - Anything in the chained run after its first year. Those years follow Elegy's own draws (docs/RANDOM-PARITY.md "Samples"), and that includes the chained run's later order refusals and battle records.
- **random-driven**: a field that misses on the reference seed but matches on at least one of Elegy's other 7 seed variants, so Elegy's draws decide it. The line says how many variants match. For a restart's battle record, it is lane D's in-year rule: Elegy drew in that year before its battles, and at least one other seed variant fights the recorded battle. The line keeps the count of matching variants, so a change in that count fails too.
- **not compared**: an expectation or field the harness does not compare, with the reason.
- **matched**: a strict battle record (a restart's, or the chained run's first year) whose tokens, players, planet, hits and moves all matched. The line says what was compared: hits after a torpedo roll answered from the record are counted as conditioned, which is not random parity.

These are classified by hand, with an owner. They cover a field that misses on all 8 variants, an order Elegy refuses, and a strict battle record that fails:
- **missing mechanic**;
- **incorrect mechanic**;
- **state-import error**;
- **unsupported order**;
- **unresolved randomness**, for a field that misses on all 8 variants by an amount that varies between them, when its owner argues it is draw-driven all the same. The class carries the owner and a one-line reason (`unresolved randomness [lane D]: reason`), and the test refuses it on a line whose value is the same on every variant. No line has it.

A field that misses on all 8 variants by a varying amount says so in its detail: "(0/8 variants match, values vary)". Its value depends on Elegy's draws, but no draw gets it right, so it is a mechanic difference until its owner shows otherwise.

## Counts

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| incorrect mechanic | 0 | 0 | 0 |
| missing mechanic | 0 | 0 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| unresolved randomness | 0 | 0 | 0 |
| random-driven | 30 | 0 | 1 |
| not compared | 92 | 0 | 0 |
| sample | 1138 | 37 | 41 |
| matched | 0 | 0 | 52 |

These counts include the message, `object`-form wormhole, design and player-count comparisons added after the final run ("Messages" and "Not compared, by reason" below), lane D's battle summaries, which cleared 9 missing-mechanic lines and 5 samples ("Closed after the final run: battle summaries"), and economy's level-up and overcrowding messages, which cleared 10 missing-mechanic lines and 8 samples ("Closed after the final run: level-up and overcrowding messages"). Samples total 1216: 1138 fields and 37 orders, checked by `TestRG002Ledger` (which logs them together as 1175), and 41 battle lines, checked by `TestBattleRecords` (36 chained records, 20 matched and 16 missed, and 5 battles Elegy fights where the chained record has none). Of the 92 fields not compared, 84 are `message_sequence` expectations, which wait on lane B's within-id comparator, and 8 are null values ("Not compared, by reason" below).

The reference run of RG-002 has no deterministic field discrepancy. The last two, lane A's 0x02e message for colonists beamed down by an unload task (Y1 and the chained year 1), cleared when Elegy began sending it (`EventColonistsBeamedDown`). With them gone, the Y1 restart (RG-002-Y1-2463) and the chained run's first year (RG-002-2463) pass on 1 of the 8 seed variants (baseline.txt `random 1`); every field they miss on the other variants is random-driven or a sample. Before them, Y4 planet 88's `packet_warp` (missing mechanic [economy]) was cleared by the starbase mass-driver fix ("Effect of the packet-warp fix" below), lane D's 9 battle-summary lines by the battle summaries ("Closed after the final run: battle summaries"), and economy's 10 message lines by the level-up and overcrowding messages. Every other field, order and battle record of the seven runs either matches the original on the reference seed, matches on at least one other seed variant (random-driven), is a sample, or is not compared.

No order fails to load. Lane F's coverage check found that the 42 order files hold 1867 order records and the vectors carry 1867 orders, so a refused order is always Elegy's gap, never the decoder's. Two fields of the host files are not in the vectors, and their meaning is UNRESOLVED: the lay-mines task's second word and the high nibble of the waypoint target type. The ledger header lists both.

- **Restarts:** on the reference seed Elegy refuses none of the orders, and all 44 battle records match. Elegy fights one battle the original did not record, Y4 at (1362,1284): random-driven, as player 2's fleet 13 ends there on the reference seed and on 2 other variants, and fights nothing there on the 5 variants that put it at the original's (1367,1305).
- **Refusals on other variants:** none. The ledger lists an order refused on another seed variant but not on the reference one, outside the chained run's samples. There was one, in Y1 and in the chained run's year 1: player 2's order 10, a waypoint-0 follow of player 6's fleet 10, which player 6's `move_ships` empties; variants 1 and 6 replay player 6 first and refused it (incorrect mechanic [lane A]). The waypoint-0 follow change (ORDERS.md "Waypoint 0, following", stars-elegy a1c4c91, BINARY-ONLY) accepts it, and both lines cleared.
- **Chained run:** 37 orders are refused in years 2–6, all samples, and 16 battle records miss, all samples ("Closed: battle records line by line" below). The same change removed 6 refused follow orders (Y2 p1 o35, Y4 p5 o64, Y5 p5 o30, o67, Y5 p6 o14, Y6 p6 o10); no other line moved.

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

## Effect of the packet-warp fix

Economy's change (a new starbase's mass driver sets the planet's packet speed to `Dw + t`, PRODUCTION-LAUNCH.md "Starbases", stars-elegy 86ca470) on main `bd5d67b`, ledger rewritten with `RG002_LEDGER=write`:

- Cleared, missing mechanic: Y4 restart planet 88 `packet_warp = 0, want 8` now matches on the reference seed.
- Cleared, sample: the chained run's year 4, planet 88 `packet_warp = 0, want 8`.
- No other line moved.

| Class | Before: fields / orders / battles | After: fields / orders / battles |
|---|---|---|
| missing mechanic | 1 / 0 / 0 | 0 / 0 / 0 |
| sample | 977 / 43 / 16 | 976 / 43 / 16 |

## Not compared, by reason

92 lines are not compared. None of the 8 null lines below is an RG-002 mechanic Elegy is known to lack. The other 84 are the `message_sequence` expectations from lane B's vector re-sync, which wait on lane B's within-id message-order comparator. The other groups are empty now: quantities Elegy models but the harness did not compare (the "Closed" notes below), and the 176 `battle` and `battle_actions` expectations, which are ledger lines of their own now ("Closed: battle records line by line").

**1. No value in the original's file.** The vectors write null where the file holds nothing to compare: 6 lines of planet `defenses` null and 2 of `packet_destination` null. Keeping null `packet_destination` out of the comparison is an ELEGY CHOICE: null means the file holds no destination record, which is not the same claim as -1 ("none"), so it is not compared as -1. Reversible if stars-elegy documents null as "none".

Every RG-002 expectation of a quantity Elegy models is now compared, by the ledger's tests or by `TestBattleRecords`.

**Closed: battle records line by line (lane D).** The 176 `battle` and `battle_actions` lines left "not compared" for one line per battle record from `TestBattleRecords`:
- the 88 records' tokens, as before, and now also their players (as a set) and planet;
- whether Elegy fights a battle at a place the year's records do not have;
- a guard that every one of the 88 records has both expectations and a ledger line.

All 88 records' players and planets match.

| Class | Before: lines | After: lines |
|---|---|---|
| not compared | 268 (176 battle, 84 `message_sequence`, 8 null) | 92 (84 `message_sequence`, 8 null) |
| matched | 0 | 52 (44 restarts, 8 chained year 1) |
| sample (battle lines) | 16 | 41 (36 chained records: 20 matched, 16 missed; 5 extra Elegy battles) |
| random-driven (battle lines) | 0 | 1 (Y4 (1362,1284)) |

**Closed on this branch: packet settings and research percent.** Economy's comparator (internal/vectors `planetEquals`, `playerEquals`; `TestPacketSettings`, `TestResearchPercent`) compares planet `packet_destination` (the target id, or -1 for none) and `packet_warp` (below 5 is unset on both sides), and player `research_percent` against the research budget. A malformed expected value fails the check rather than reading as absent, as for `starbase` (#191). That removed 112 lines from group 1 (108 `packet_destination`, 2 `packet_warp`, 2 `research_percent`). 110 of them match on the reference seed. The other two were Y4 planet 88's `packet_warp`, a missing mechanic, and the same field in the chained run's year 4, a sample; the packet-warp fix cleared both.

**Closed after the final run: battle summaries (lane D).** Elegy now sends each battle player its summary (`EventBattleSummary`; PARITY-STATUS.md "Battle summaries"). These lines cleared, all now matching on the reference seed:
- missing mechanic [lane D], 9: 0x116 in chained 2463, Y1 and Y2; 0x095 in chained 2463, Y1, Y4 and Y6; 0x09f in Y3; 0x0a1 in Y4;
- samples, 5: the chained run's 2464 0x116, 2465 0x09f, 2466 0x0a1 and 0x095, and 2468 0x095.

**Closed after the final run: level-up and overcrowding messages (economy).** Elegy now sends 0x026 when growth shrinks a planet over capacity that survives with a hab value of 0 or more (`EventOvercrowded`), and 0x050 and 0x05f at each research level-up check (`EventLevelUp`, `EventPartAvailable`; MESSAGES.md at stars-elegy b4c0086; PARITY-STATUS.md "Level-up and overcrowding messages"). These lines cleared, all now matching on the reference seed:
- missing mechanic [economy], 10: 0x026 in chained 2463 and Y1–Y6; 0x050 in Y2 and Y3; 0x05f in Y3;
- samples, 8: the chained run's 0x026 in 2464–2468, 0x050 in 2464 and 2465, and 0x05f in 2465.

No line was added.

**Closed after the final run: messages.** The 68 `message` lines are compared now. 32 match on the reference seed, 15 are samples, and 21 were missing-mechanic lines for ids Elegy sent no event for. Lane D's battle summaries cleared 9, economy's level-up and overcrowding messages 10, and lane A's 0x02e the last 2. The mapping, the per-owner split and the effect on other corpora are in [PARITY-STATUS.md](PARITY-STATUS.md) "Messages compared".

**Closed after the final run: `object`-form wormholes.** The 72 `object` expectations of kind `wormhole` are compared now, with the fields the `wormhole` form already used plus `id` and `partner` (the other end's vector id). The vectors mark all 72 as samples: a wormhole end jiggles or jumps on draws every year (OBJECTS.md). All 72 miss, making 160 sample field lines: x (72), y (65), `years_since_jump` (13, a jump on one side and not the other) and `known_to` (10). `partner`, `stability_class` and `destination_known_to` match in all 72.

**Closed after the final run: designs and player counts.** `design` and `starbase_design` expectations are compared with the design in the owner's slot at the end of the year: hull, mass per ship, and the slots in hull-slot order (null for an empty one). Player `counts` compares the numbers of ship and starbase designs. All 20 RG-002 lines match on the reference seed: 6 design reads, 8 starbase designs (samples, all matching) and 6 counts.

## Random-driven

Each of these lines matches on at least one of the 8 seed variants.
- **Restarts:** 30 field lines: Y2 4, Y3 10, Y4 11 and Y5 5. They are mostly mining and surface minerals, production queues, and fleets whose movement rides on random merges, battles and target picks.
- **Chained year 1:** none.

Until 5c1ca55, a line counted as random-driven whenever its value differed between variants, even when no variant matched. On main `546dacc` there were 23 such lines, and they were classed as incorrect mechanics with their owners. All 23 are among the cleared lines above.
