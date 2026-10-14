# Parity status

Elegy's end-to-end results against the original, one entry per full-order regression. The release-candidate coverage report, which separates strict passes, random-dependent results, samples, absent source values and untested mechanics across every corpus, is [COVERAGE.md](COVERAGE.md). The behavioral record these rest on is stars-elegy `docs/PARITY.md`. Other elegy docs mean that file when they cite "docs/PARITY.md".

## Current status: RG-002, RG-003 and RG-004 on main `f09acd9`

Updated 2026-10-14 (RG-005 ingest PR): RG-005, the out-of-sample game, is in the rg corpus, and its result is in section "RG-005" below, not in the tables of this section, which stay the candidate's in-sample result. RG-005 has 12 strict expectations that miss on all 8 seeds, cause UNRESOLVED (leading candidate: the computer players' level is not loaded). With it, the rg corpus case tally is fail 62, random 20, random with the reference seed failing 12, and baseline.txt keeps 27 RG entries (RG-005 10, 7 on the first run). No RG-002, RG-003 or RG-004 line moved.

The combined run on elegy main `f09acd9` (the release candidate, [COVERAGE.md](COVERAGE.md); the RG vectors are stars-elegy main `104e8f5`'s, with #206's year-6 rebuild) is the result the three corpora rest on. Run on that SHA with no local change, the ledgers regenerated with `RG002_LEDGER=write` (no line moved):
- Updated 2026-10-14: this section was first measured on main `5a735e1` (after #242), then on `215cd38` (after #251–#257). The changes since are in the dated notes below, which quote the earlier values.
- `go test ./...`: every package passes.
- `go test ./regress -run TestLongGames`, the PR subset: 3 seeds × 2 rulesets × 4 opponent sets = 24 games of 60 years on a small map, each played three times (primary, replay, and reload after 23 years). 24 of 24 have no failure.
- The nightly matrix on the same SHA: 12 seeds × 2 rulesets × 4 opponent sets = 96 games of 150 years on a medium map, each replayed and reloaded after 77 years: 96 with no failure (444 s).
- `TestRG002Ledger`, `TestRG003Ledger`, `TestRG004Ledger` and `TestBattleRecords` pass: every ledger matches the run exactly, with no UNCLASSIFIED line.
- `TestParityVectors`, rg corpus: pass 0, measured pass 2, fail 48, random 13, random with the reference seed failing 9. baseline.txt keeps 17 RG entries: RG-002 and RG-003 chained year 1 and Y1 (`random 1` each), and RG-004's 12 restarts and chained year 1 (`random 1` to `random 7`).
- **Why the failing cases fail, given 0 deterministic misses.** A case fails on a seed when any of its strict field expectations misses there, and it is listed `fail` when no seed matches all of them at once. Every RG-002, RG-003 and RG-004 case that fails (10, 22 and 11) fails for one of two reasons (per-seed check of every strict non-battle field expectation, all 8 seeds, main `f09acd9`):
  - **Restarts (RG-002 Y2–Y6, RG-003 Y2–Y12):** every expectation that misses on any seed matches on at least one other seed. These cases have random-driven expectations only, and no single seed matches them all together. RG-004's restarts all pass on some seed.
  - **Chained runs after year 1** (RG-002 years 2–6, RG-003 years 2–12, RG-004 years 2–12): these years continue from Elegy's own year-1 state, which already differs from the original's on the draws. Some strict expectations miss on all 8 seeds here: RG-002 5 in year 2 rising to 47 in year 6, RG-003 to 169 in year 12, RG-004 3 in year 2 to 50 in year 12. (Updated 2026-10-13: on `5a735e1` this read "RG-003 to 166 in year 12, RG-004 3 in year 2 to 48 in year 12".) The ledger classes every chained difference after year 1 as a sample, as downstream of that divergence.
  - Chained year 1 and every restart: no strict expectation misses on all 8 seeds.
  - Battle records are not in this check. `TestBattleRecords` and the ledgers judge them, with no hand line.

**Classification.** Each corpus has 0 deterministic misses: 0 incorrect-mechanic lines and 0 unresolved-randomness lines. Every compared difference from the original is a sample or random-driven (it matches on at least one seed variant). Ledger lines, chained run / restarts:

| class | RG-002 | RG-003 | RG-004 |
|---|---|---|---|
| matched | 8 / 44 | 8 / 100 | 4 / 47 |
| strict miss (incorrect mechanic, any owner) | 0 / 0 | 0 / 0 | 0 / 0 |
| random-driven | 0 / 35 | 0 / 76 | 1 / 17 |
| of which hand-classed with the draw | 0 / 3 | 0 / 3 | 0 / 0 |
| sample | 1019 / 199 | 5287 / 450 | 2519 / 263 |
| not compared | 0 / 0 | 0 / 0 | 2 / 2 |
| total | 1305 | 5921 | 2855 |

- Updated 2026-10-13 (#247): the 0x08f/0x090 bombing fleet is now compared, which removes 2 RG-002 and 4 RG-003 not-compared lines. The row was "3 / 3 | 11 / 11 | 19 / 19" and the totals 1316, 5947, 2893.
- Updated 2026-10-13 (#248): a null `packet_destination` on a planet that lost its starbase is now compared (PK-1), which removes 2 RG-002 and 4 RG-003 not-compared lines. The row was "2 / 2 | 9 / 9 | 19 / 19" and the totals 1314, 5943, 2893.
- Updated 2026-10-13 (0x040 PR): 0x040 and 0x023 are mapped, which removes RG-004's 8 0x040 lines (4 chained, 4 restart). Two RG-004 sequences (y4 #224, chained and Y4 restart) now stop at their next unmapped entry, 0x007, which adds 2 lines. The row was "1 / 1 | 7 / 7 | 19 / 19" and the totals 1312, 5939, 2893.
- Updated 2026-10-13 (lane A, invasion messages): 0x004's slots and 0x007 are now compared (section "Invasion messages"). RG-003 loses 3 not-compared lines in the chained run and 3 in the restarts, which all match. RG-004 loses 13 and 13: the restarts' 0x007 match, and the chained run gains 8 sample lines (years 3, 4, 8, 9, 10, after the year-1 divergence). The rows were "not compared 1 / 1 | 7 / 7 | 16 / 16", "sample 1024 / 199 | 5291 / 451 | 2523 / 263" and the totals 1312, 5939, 2887.
- Updated 2026-10-13 (FS-1 PR): 0x05b's fleet slot is now compared (design·512 + fleet number, stars-elegy FS-1, 28a3175) and matches in all 4 entries (regenerated on main cbc3354, #251). The counts do not move: the same year-1 sequence entries stay not compared for 0x02e's mineral and location slots (lane A), which the ledger line now names. The not-compared row was "0x05b fleet slot … lane D".
- Updated 2026-10-13 (0x0cd PR): 0x0cd is mapped, which removes RG-003's 4 0x0cd lines (2 chained, 2 restart), all matching (regenerated on main c3916f8, #253). The row was "1 / 1 | 4 / 4 | 3 / 3" and the totals 1312, 5933, 2869.
- Updated 2026-10-13 (lane A, chase): chasers of a fleet destroyed during movement go to its start position (stars-elegy #198 de1bd6f, Z-MD; regenerated on main 8cf4236, #254). RG-003-Y11's battle at (1323,1300) moves from sample to random-driven: on 1 of the 7 other variants fleet 3/12 dies in the field and its chasers fight there (section "Battle records the vector flags sample"). The other moves are chained samples whose values changed or went: RG-002 5, RG-003 5, RG-004 12 fewer. The rows were "random-driven 0 / 35 | 0 / 75 | 1 / 17", "sample 1024 / 199 | 5291 / 451 | 2531 / 263" and the totals 1312, 5929, 2869.
- Updated 2026-10-13 (alt_battle PR): the 4 `alt_battle` expectations (RG-003 y9 (1388,1386), RG-004 y3 (1365,1400), chained and restart) get their lines from `TestBattleRecords` only, which judges each as a sample, and no longer also a not-compared line in the vectors ledger. Three already had a battle-set sample line (Elegy fights there with the alternative stream's players). The fourth, RG-003's chained y9, where Elegy does not fight, gains "sample matched: no battle here, as in the recorded stream". An `alt_battle` in a year that failed to run fails, and one where Elegy fights in a year with no battle record is judged as an extra battle would be; none of the 4 is in either case. The rows were "sample … | 5286 / 450 | …", "not compared 1 / 1 | 2 / 2 | 3 / 3" and the totals 1307, 5924, 2857.
- Updated 2026-10-14 (lane A, 0x02e slots): 0x02e's cargo type and location slots are compared (cargo type 3 for colonists; a planet as 65535 then the planet number; MEASURED, stars-elegy CM-1, 4f59d8d; regenerated on main 0237862, #261). The year-1 sequence entries that held them (RG-002 and RG-003, chained and restart) now match in full, which removes their 4 not-compared lines. The fleet slot is compared as before; its owner·512 part for players other than 0 is an ASSUMPTION (consistent with FS-1 and 0x028, not measured for 0x02e). The row was "1 / 1 | 1 / 1 | 2 / 2" and the totals 1307, 5923, 2855.

- "matched" ledger lines are battle records. Matching fields get no line. Matched field expectations are in the next table.
- Random-driven lines by object: RG-002 14 fleet, 11 planet, 5 production queue, 3 player, 1 battle, 1 message sequence. RG-003 26 fleet, 22 planet, 18 production queue, 3 minefield, 3 player, 3 battle (2 before the chase PR), 1 message sequence. RG-004 10 fleet, 3 planet, 2 battle, 1 minefield, 1 player, 1 production queue.
- The chained runs diverge in year 1, and the ledger classes every later chained difference as a sample. RG-004's one chained random-driven line is in year 1.

Field expectations (one per compared field; battle records are in `TestBattleRecords`), reference seed:

| corpus | run | strict matched | strict random-driven | strict missed | sample matched | sample missed | not compared |
|---|---|---:|---:|---:|---:|---:|---:|
| RG-002 | chained | 356 | 0 | 0 | 1663 | 562 | 0 |
| RG-002 | restarts | 2227 | 25 | 0 | 185 | 144 | 0 |
| RG-003 | chained | 356 | 0 | 0 | 3112 | 2196 | 1 |
| RG-003 | restarts | 4833 | 59 | 0 | 452 | 320 | 1 |
| RG-004 | chained | 197 | 1 | 0 | 1732 | 898 | 3 |
| RG-004 | restarts | 2517 | 12 | 0 | 139 | 160 | 3 |

Recomputed on main `f09acd9` (2026-10-14). The 0x02e slots (CM-1) moved one year-1 sequence entry per run from not compared to strict matched in RG-002 and RG-003. The not-compared expectations left are `alt_battle` (RG-003, RG-004; judged by `TestBattleRecords`), `design_gone` and one sequence of sample entries only (RG-004). On `215cd38` the table read:

| corpus | run | strict matched | strict random-driven | strict missed | sample matched | sample missed | not compared |
|---|---|---:|---:|---:|---:|---:|---:|
| RG-002 | chained | 355 | 0 | 0 | 1663 | 562 | 1 |
| RG-002 | restarts | 2226 | 25 | 0 | 185 | 144 | 1 |
| RG-003 | chained | 355 | 0 | 0 | 3112 | 2196 | 2 |
| RG-003 | restarts | 4832 | 59 | 0 | 452 | 320 | 2 |
| RG-004 | chained | 197 | 1 | 0 | 1732 | 898 | 3 |
| RG-004 | restarts | 2517 | 12 | 0 | 139 | 160 | 3 |

Recomputed on main `215cd38` (2026-10-13). On `5a735e1` the table read:

| corpus | run | strict matched | strict random-driven | strict missed | sample matched | sample missed | not compared |
|---|---|---:|---:|---:|---:|---:|---:|
| RG-002 | chained | 355 | 0 | 0 | 1662 | 562 | 2 |
| RG-002 | restarts | 2224 | 25 | 0 | 185 | 144 | 3 |
| RG-003 | chained | 355 | 0 | 0 | 3106 | 2196 | 8 |
| RG-003 | restarts | 4823 | 59 | 0 | 452 | 320 | 11 |
| RG-004 | chained | 197 | 1 | 0 | 1724 | 890 | 19 |
| RG-004 | restarts | 2501 | 12 | 0 | 139 | 160 | 19 |

Every expectation that left "not compared" since (0x08f/0x090 #247, packet settings #248, 0x040/0x023, 0x004/0x007, 0x0cd) is now strict matched or a sample. Strict random-driven and strict missed do not move, and every row's total is unchanged (RG-002 2581, RG-003 5665, RG-004 2831 expectations per run).

Not recomputed for the alt_battle PR (2026-10-13): the `alt_battle` expectations counted here as not compared (RG-003 and RG-004, one per run) are judged in `TestBattleRecords`, like the battle records this table leaves out.

"Chained, strict" counts year 1 only. Later chained years count as samples, as in the ledger.

**Open owners (random-driven lines with a named draw).** RG-002 and RG-003 share them. Each has 3 lines, all in the Y6 restart (section "RG-002 and RG-003 year 6 rebuilt"):

| line | owner | draw |
|---|---|---|
| fleet 6/12, waypoint 1 | lane D | movement tie draws (#247): the target, player 3's fleet 2, dies in Elegy's battle at (1296,1356) on 6 of 8 seeds |
| planet 102, surface minerals | economy | mining's +1 draw on germanium (54 kT on seed 0, 53 kT on seed 7) |
| planet 102, production queue | economy | the same germanium draw (53 kT gives 93%, 54 kT gives 99%) |

The other random-driven lines have no named draw. They match on some seed variant, so they are not counted as misses. Exact random parity needs the original's generator in the harness (SPRINT.md P1-1, pending Bobby).

**Order accounting.** Every recorded order is loaded. Each is accepted, or refused and recorded as an `order` ledger line. No RG vector has a load error. Per-year tables are in "Order accounting" below.

| corpus | orders | refused, chained run (all samples) | refused, restarts |
|---|---:|---:|---:|
| RG-002 | 1867 | 37 | 0 |
| RG-003 | 3731 | 220 | 0 |
| RG-004 | 2049 | 84 | 0 |

RG-004's 9 restart refusals and 9 chained ones (the delete of empty design slot 0, years 4–12) are gone since lane A's #244, which accepts the delete.

**Not compared, by owner.**

| cause | RG-002 | RG-003 | RG-004 | owner |
|---|---:|---:|---:|---|
| `design_gone` expectation not compared | 0 | 0 | 2 | lane B (harness) |
| message sequence of sample entries only (nothing strict to compare) | 0 | 0 | 2 | none needed |
| total | 0 | 0 | 4 | |

- Updated 2026-10-13 (alt_battle PR): the `alt_battle` row (0 | 2 | 2, lane D) is gone; those expectations are battle-ledger samples (Current status notes). The total was 2 | 4 | 6.
- Updated 2026-10-14 (lane A, 0x02e slots): the row "0x02e mineral and location slots (MESSAGES.md does not say which word of a planet location holds the planet), in the year-1 sequence entry that also holds 0x05b | 2 | 2 | 0 | lane A" is gone (stars-elegy CM-1, 4f59d8d), and the total was 2 | 2 | 4.

- stars-elegy MESSAGES.md "Invasion slots" now gives 0x004's and 0x007's slots (MEASURED: the player slot is the player number + 0x30, the troops are the total dropped). That answers the slot question this section used to leave to lanes F and G. Updated 2026-10-13: the mapping is in (lane A, section "Invasion messages"), and the table's 0x004 row (RG-003 2) and 0x007 row (RG-003 4, RG-004 26) are gone; the total was 2 | 14 | 32.
- 0x040 against 0x023 depends on the previous planet processed (MESSAGES.md 0x040 notes), so mapping it needs that rule as well as an event.
  - Updated 2026-10-13 (0x040 PR): mapped with that rule (Legacy `lost_planet_message_carry`, ruleset v6). The 0x007 row was 24 (RG-004) and the total 2 | 14 | 38; two sequences now stop at 0x007 instead of 0x040.
- Updated 2026-10-13 (0x0cd PR): the row "0x0cd starbase built: id rule contradicted by two vectors (message and sequence entry) | 0 | 4 | 0 | economy (mapping); lane F supports with the dock-rule evidence (stars-elegy #204)" is gone, and the total was 2 | 8 | 6. stars-elegy 898c526 MESSAGES.md now records 0x0cd for every starbase built of the four hulls seen, Orbital Fort, Space Dock, Space Station and Ultra Station (MEASURED), and keeps the dock rule as a preserved contradiction. Death Star is NOT RUN; Elegy sends 0x0cd for it too (inferred).

## RG-002: full-order six-year regression

Measured on elegy main `8426a6a` plus this branch, which adds docs, the variant-refusal test change (internal/vectors tests and two ledger lines) and economy's comparator for planet `packet_destination` and `packet_warp` and player `research_percent` (internal/vectors). Main `8426a6a` carries #191, #193, #192 and #195. That is the final run Bobby's report rests on. **Since then** #197 cleared point 3's `packet_warp` line, and the harness compares messages and `object`-form wormholes (the last two sections), which adds 21 missing-mechanic message lines to point 3 and changes the counts below it. Vectors: stars-elegy `vectors/rg/rg002*.json` (the chained 2463→2468 and each year's restart, Y1..Y6), synced unchanged. Details and the line-by-line breakdown are in [RG002-REPORT.md](RG002-REPORT.md). The ledger is [`engine/testdata/vectors/rg/rg002-ledger.txt`](../engine/testdata/vectors/rg/rg002-ledger.txt).

Bobby's five requirements, in order:

1. **All recorded orders loaded: MEASURED, yes.**
   - Lane F's coverage check (stars-elegy fe0ccc5) found that the 42 order files hold 1867 order records and the vectors carry 1867 orders.
   - The loader converts every one of them: the ledger has no `unsupported order` line, and no vector stops loading.
   - Two host-file fields are not in the vectors. Both are constant across RG-002, and their meaning is UNRESOLVED: the lay-mines task's second word, and the high nibble of the waypoint target type. The ledger header lists them.
2. **No silent rejections: MEASURED. Every refusal, on every seed variant, is a ledger line.**
   - Every order Elegy refuses is a ledger line, so a refusal is never silent.
   - On the reference seed (variant 0), the six restarts and the chained run's first year refuse none.
   - The ledger also lists any order refused on another variant but not on the reference one, outside the chained run's samples, with the variants that refuse it. `TestRGNoRejectedOrders` checks every variant of RG-001, which refuses nothing on any of them.
   - The chained run refuses 43 orders in years 2–6. All 43 are samples: those years follow Elegy's own draws (RANDOM-PARITY.md "Samples").
   - **One refusal on other variants: incorrect mechanic [lane A], 2 ledger lines.** In Y1, and in the chained run's year 1, variants 1 and 6 refuse player 2's order 10. That order is a waypoint-0 change aimed at player 6's fleet 10, which player 6's order 1 (`move_ships`) empties. When the year's player shuffle replays player 6 first, the target is gone and Elegy refuses the order ("no such object"). The reference seed replays player 2 first.
     - stars-elegy #170 (a1c4c91, BINARY-ONLY) says the original never rejects a waypoint-0 follow whose target was merged away: a waypoint edit checks only the ordering fleet and the index. The stale id is resolved later, by the start-of-year waypoint check or by the removal-time retarget. So Elegy's refusal is the bug. #170 also says the player shuffle itself matches the original: the year's first draws, Fisher-Yates.
     - It is classed `incorrect mechanic [lane A]: waypoint-0 follow of a merged-away fleet is refused (stars-elegy ORDERS 'Waypoint 0, following', a1c4c91, BINARY-ONLY)`, one line each for Y1 and the chained run.
     - Cleared since by the follow change: a follow order is never refused for its leader, so both lines and 6 refused FollowOrder samples of the chained run leave the ledger (RG002-REPORT.md "Counts").
3. **Exact deterministic field diffs: MEASURED, one.**
   - The reference run (seed variant 0) has one field line of a deterministic class, and no order or battle record line of one. It appeared when this branch began comparing packet settings (below).
   - **Missing mechanic [economy], 1 field line.** In the Y4 restart, planet 88's `packet_warp` is 0 in Elegy and 8 in the original. The planet's starbase with a mass driver completes that year. Economy infers that the original sets the launch warp when such a starbase completes, from this one observation: "packet warp not set when a starbase with a mass driver completes (inferred from one observation)". The same field in the chained run's year 4 is a sample.
   - Across all variants, the other hand lines are the 2 incorrect-mechanic refusal lines above: orders, not fields.
   - `TestRG002Ledger` and `TestBattleRecords` fail on any new line and on any ledger line no longer produced.
   - Before #189 and #190 there were 78 such lines. RG002-REPORT.md "Effect of #189 and #190" lists them.
4. **Random results classified: yes.**
   - 30 field lines are random-driven. Each misses on the reference seed and matches on at least one other seed variant. The line keeps that count.
   - A line that misses on every variant is a hand line with an owner, never random-driven.
   - **Checked on every variant, not only the reference.** A scratch check ran all 8 variants of the seven runs and classed every failing field. Each one is a ledger random-driven line, a sample, or a field that matches on the reference seed (so random by the same rule). On main `8426a6a` plus this branch the 56 runs have 10,931 failing field results: 94 on random-driven lines, 157 on fields that match on the reference seed, 10,672 samples, and 8 on the missing-mechanic line above (it fails on every variant). No variant of any run has a deterministic field mismatch beyond that line.
   - No restart battle record needed lane D's in-year random rule.
5. **Every test on the final main SHA: yes, on main `8426a6a` plus this branch** (below).

### Counts per class

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| missing mechanic | 1 | 0 | 0 |
| incorrect mechanic | 0 | 2 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| unresolved randomness | 0 | 0 | 0 |
| random-driven | 30 | 0 | 0 |
| not compared | 344 | 0 | 0 |
| sample | 977 | 43 | 16 |

Samples total 1036, by kind: 977 fields and 43 orders, checked by `TestRG002Ledger` (which logs them together as 1020), and 16 battle records, checked by `TestBattleRecords`. All are lines of the same ledger file.

The 344 not-compared lines are:
- 160 comparator gaps for quantities Elegy models (72 `object`-form wormholes, 68 messages, 14 design reads, 6 player counts);
- 176 battle expectations that `TestBattleRecords` compares record by record instead;
- 8 null in the original's file, among them 2 planet `packet_destination` null (ELEGY CHOICE, below).

The 160 are where a mismatch could still go unseen (RG002-REPORT.md "Not compared, by reason").

**Packet settings and research percent are now compared.** This branch folds in economy's comparator (internal/vectors `planetEquals` and `playerEquals`, tests `TestPacketSettings` and `TestResearchPercent`). A malformed expected value of any of the three fails the check, as a malformed `starbase` does (#191); it never reads as absent. It removed 112 not-compared lines (108 `packet_destination`, 2 `packet_warp`, 2 `research_percent`). Of those, 110 match on the reference seed, 1 is the missing mechanic above and 1 is its chained-run sample.
- `packet_destination` is the target planet id, or -1 for none (vectors/README.md "planet_settings").
- `packet_warp` below 5 means unset on both sides (vectors/README.md "initial_state"; OBJECTS.md "Launch"), so 0 and 4 count as equal and 5 does not.
- **ELEGY CHOICE: a null `packet_destination` stays not compared.** The vectors write null where the original's file holds no destination record, which is not the same claim as "no destination" (-1). Comparing null as -1 would read an absent record as a value. So the 2 null lines stay in the "null in the original's file" group, as `defenses` null does. Reversible if stars-elegy documents null as "none".
  - Superseded by #248: a null now compares as no destination, warp 4 (PK-1).

### The run on main `8426a6a`

- `go test ./... -count=1` passes in every package: ai, cmd/elegy, engine, game, internal/vectors, newgame, objects, oraclerng, races, regress and terraform.
- **The ledger.** `go test -run 'TestRG002Ledger|TestRGNoRejectedOrders' -v ./internal/vectors` passes, logging not compared 344, random-driven 30, sample 1020 (977 fields and 43 orders; the 16 battle samples are `TestBattleRecords`'), incorrect mechanic 2 and missing mechanic 1. `TestBattleRecords` passes. `RG002_LEDGER=write` and `PARITY_BASELINE=write` both leave the tree unchanged.
- **Battle records.** No strict RG-002 record fails: the restarts' 44 and the chained run's 8 in its first year. Of the chained run's 36 later records, all samples, 20 match.
- **The long-game matrix**, separate from the RG-002 variants: `go test ./regress -run '^TestLongGames$' -count=1 -v -args -seeds=1,2,3,4,5,6,7,8 -years=150 -save-at=77 -size=2`. That is 8 seeds × 2 rulesets × 4 opponent sets = 64 games of 150 years on a medium map. Each game is replayed and reloaded after 77 years. Result: 64 of 64 games played 150 of 150 years with no failure, in 238 s.
- **The RG-002 8-seed matrix** (`go test -run TestParityVectors -v ./engine`, case level: a case passes on a seed when every compared expectation matches):

  | Case | Seeds that pass |
  |---|---|
  | RG-002-Y1-2463 | 1 of 8 (the reference seed among them) |
  | RG-002-Y2-2464 to Y5-2467 | 0 of 8 |
  | RG-002-Y6-2468 | 8 of 8 |
  | RG-002-2463 (chained, year 1) | 1 of 8 (the reference seed among them) |
  | RG-002-2464 to 2467 (chained) | 0 of 8 |
  | RG-002-2468 (chained) | 6 of 8 |

  - **How to read it.** A case passes on a variant only when every compared expectation matches, including every draw-decided one. "0 of 8" for Y2–Y5 does not mean a deterministic failure: every field that fails, on every variant, is random-driven or a sample (point 4). Each variant misses a different few draw-decided fields, so no single variant gets them all.
  - The chained years 2–6 are samples. RG-002-2468 passing on 6 seeds is a sample result, not replay parity.
  - Exact random outcomes need the original's generator in the harness (SPRINT.md P1-1).

### Messages compared (after the final run)

Measured on elegy main `f6beefa` (#194 and economy's packet-warp fix #197) plus the messages-comparator branch.

The harness now compares `message` expectations (vectors/README.md "message": player `player` got message `message_id`, `present`). Elegy's events carry no message id, so internal/vectors `pvMessages` maps each year's events to the ids stars-elegy MESSAGES.md gives them:
- 0x004 from `EventDropRepelled`, sent to the planet's owner;
- 0x035–0x038 from `EventBuilt` of factories or mines, with MESSAGES.md's merge rule for build counts;
- 0x04e from `EventFleetArrived`;
- 0x02e from `EventColonistsBeamedDown` (added later; see below);
- 0x05b from `EventFleetScrapped` in deep space;
- 0x083–0x086 from `EventCometStrike`, by size;
- 0x08f and 0x090 from `EventBombed` followed by `EventPlanetEmptied`.

Some ids have no Elegy event at all: 0x026, 0x02e, 0x050, 0x05f, 0x095, 0x09f, 0x0a1 and 0x116. They are compared too, so each one the original sends is a mismatch. Every other id is skipped ("message 0x… has no event mapping"), so an id joins the comparison only with its mapping.

**RG-002's 68 message lines:**
- 32 match on the reference seed: 0x004 (2), 0x036 (12), 0x038 (12), 0x04e (2), 0x05b (2) and 0x08f (2).
- 15 are samples: 14 in the chained run's years 2–6, and Y2's huge comet (0x086).
- 21 are new missing-mechanic lines. Each fails on every seed variant:
  - **economy, 10 lines:** 0x026 overcrowding (7: each restart and the chained run's first year), 0x050 research level gained (2: Y2, Y3) and 0x05f component available (1: Y3);
  - **lane D, 9 lines:** no per-player battle summary: 0x095 (4), 0x116 (3), 0x09f (1) and 0x0a1 (1);
  - **lane A, 2 lines:** 0x02e, colonists beamed down by an unload task (Y1 and the chained year 1). Cleared since: an unload task that drops colonists on the fleet owner's own planet sends `EventColonistsBeamedDown`, and both lines match on the reference seed.

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| missing mechanic | 21 | 0 | 0 |
| incorrect mechanic | 0 | 2 | 0 |
| random-driven | 30 | 0 | 0 |
| not compared | 276 | 0 | 0 |
| sample | 991 | 43 | 16 |

The other classes stay 0. The 21 missing-mechanic field lines are the message lines; #197 cleared Y4 planet 88's `packet_warp` and its chained sample. Lane D's battle summaries have since cleared 9 of them ("Battle summaries" below). The 276 not-compared lines are 92 comparator gaps (72 `object`-form wormholes, 14 design reads, 6 player counts), 176 battle expectations that `TestBattleRecords` compares, and 8 nulls. The scratch check of all 56 variant runs finds no unclassified failing field: 168 results on the 21 missing-mechanic lines (8 each), 94 random-driven, 157 that match on the reference seed, and 10,784 samples.

**Other corpora.** Comparing messages takes 38 cases out of the parity baseline (`engine/testdata/vectors/baseline.txt`). Every one is a known gap above, except one new finding:
- 0x050 or 0x05f (research level, component available): 29 cases, KX-002 (12), KX-003, KX-004 (2), KX-005 (5), PQ-001 (3), RD-P15, RG-001 Y2–Y6 and TK-203.
- 0x026: also in KX-002 G1 and KX-005 T0–T2.
- RG-002: Y1, Y6, and the chained 2463 and 2468.
- **0x04e after a refused gift: WU-AICOMP3, WU-AICOMP4, WU-B1, WU-B2 (lane A).** A transfer task to another player at the fleet's last waypoint is refused (Elegy's `EventFleetGiftRefused`), and the original then sends 0x04e: a refusal cancels the task, and 0x04e follows at the last waypoint (MESSAGES.md 0x04e (a)). Elegy sent no `EventFleetArrived` after the refusal. Cleared since: a refused gift ends the task, and a fleet with no waypoints ahead gets `EventFleetArrived` after the refusal; a fleet given away gets none. The four cases are back in the baseline.

### Battle summaries (lane D)

Elegy now sends each battle player its summary (`EventBattleSummary`, engine/combat_summary.go), and `pvMessages` maps it to MESSAGES.md's 0x07e, 0x091–0x0a8 and 0x113–0x116. All of those ids are compared now.
- **RG-002:** lane D's 9 missing-mechanic lines match on the reference seed (0x095 ×4, 0x116 ×3, 0x09f, 0x0a1), and so do the 5 summary samples of the chained run's years 2–6. No new line appears. The ledger is missing mechanic 13 (economy 10, lane A 2, Y4 planet 88's `packet_warp`), incorrect mechanic 2 (orders), random-driven 30, not compared 276, and samples 987 fields / 43 orders / 16 battle records.
- **Other corpora:** every summary expectation outside RG-002 matches: CB-048 and CB-048-ctl (0x098, 0x099), CB-049 (0x09b, 0x09c) and SC-036 (0x0a4, 0x0a7). The parity baseline is unchanged. Every expectation of these ids in the corpora is "present"; none checks that a wrong id is absent, so `TestBattleSummaryRules` pins the choice among the ids.
- **CB-059 (stars-elegy #184, 3bd93e1).** Two of Elegy's readings failed and are fixed: "hit" in the one-ship texts is armor damage only (CONFIRMED by prediction), and the 2-player texts apply whenever two players fought, with every other ship at the location counted as the enemy's (MEASURED). 0x0a2, 0x0a3, 0x113 and 0x114 are CONFIRMED by prediction, and no summary when an AR starbase dies is MEASURED. COMBAT-STATUS.md "CB-059 results" has the cases. No corpus expectation moved.
- **CB-060 (stars-elegy #185, a564ab4), late-tech weapons.** Two of Elegy's rules failed and are fixed: a gatling shot hits the targets of both plan types, not only the primary type, while a primary-type target is in reach (G1, CONFIRMED by prediction), and a sapper shot that strips a target's last shields carries on by the beam carry rule (S2, MEASURED). The other cases match within the distances seen; COMBAT-STATUS.md "CB-060 results" lists the NOT RUN parts.
- **CB-061 (stars-elegy #187, 19896cd).** In a multi-race summary the races count is the number of players who fought, not the recipients: three fought at a fourth player's planet whose starbase took no part, and all four got 0x0a8 with 3 (CONFIRMED by prediction, PQ-1b, the starbase owner's summary included). Elegy had used the recipients (4) and is fixed; the message's other counts match (MEASURED).
- **RG-002 battle records, one ledger line each.** The 176 `battle` and `battle_actions` expectations are no longer "not compared". `TestBattleRecords` writes one line per record, and now also compares each record's players (as a set) and planet, and whether Elegy fights a battle the year's records lack. All 88 records' players and planets match. The ledger is incorrect mechanic 0, missing mechanic 0, random-driven 31 (30 fields, plus the Y4 restart's extra battle at (1362,1284), which follows player 2's fleet 13: Elegy fights there on the 3 variants that put the fleet there, and on none of the 5 that put it where the original did), not compared 92 (84 `message_sequence`, 8 null), matched 52 battle records (44 restarts, 8 chained year 1), and samples 1138 fields / 37 orders / 41 battle lines. RG002-REPORT.md has the table.
- **Battle records the vector flags sample.** A restart's battle record that the vector marks `sample` (it differed between the original's own random streams) is a sample when Elegy misses it, or random-driven when another seed variant matches it. A record the vector does not flag stays strict: a miss with no matching variant fails (`TestBattleRecordRandomInYear`, `TestBattleRecordUnflaggedVectorStaysStrict`). The sample line carries the comparison's first difference, so a gap stays visible in the ledger.
  - **`start_sample` false (stars-elegy e3619ff, vectors README).** On such a record only the token order and start squares were drawn: every alternative stream had the same stacks. So start tokens that differ in anything else are a gap, not a sample, and the miss fails ("start tokens differ though the record's start is the same in every stream"). This covers a token missing or extra, any other token field, no battle, and other players or planet. A difference in start squares only, or in hits, stays a sample. An absent `start_sample` (outside `rg`) is not false. Tests: `TestBattleRecordHarness` (checkStartSample), `TestBattleRecordSampleMissIsNotFailure`, `TestBattleRecordStartSampledIsSample`, `TestBattleRecordNoStartSampleIsSample`.
  - RG-001 and RG-002 records all have `start_sample` false, and RG-003 has 100 of 102. Only RG-003-Y12 at (1288,1347) goes strict: its token 0 starts at damage 281 in Elegy against 235 in the record, from the ship-move damage rule (lane A).
  - RG-003-Y11 at (1323,1300) has `start_sample` true (neither other stream fought there) and stays a sample. It still hides a lane A gap (combat hidden-gaps audit): the original's fleet 3/12 died to mines at (1328,1302), and its chasers went to its start position, while Elegy's go to where it died. Updated 2026-10-13 (lane A, chase, stars-elegy #198 de1bd6f Z-MD): Elegy's chasers of a fleet destroyed during movement now go to its start position too. On the reference seed Elegy's 3/12 is not destroyed by the field (the mine draws), so the miss stays; on 1 of the 7 other variants it is, its chasers end at (1323,1300) and Elegy fights the record's battle there. The line is now random-driven ("random, 1 of 7 other variants match").
  - RG-002's ledger is unchanged.
  - **`alt_battle` (stars-elegy 41d3c86, #202; vectors README "Battle records").** It marks a place where an alternative stream of the original fought and the recorded stream did not. When Elegy fights there with exactly the alt_battle's players (as a set), the battle-set check logs "sample (alt_battle in the vector) missed" instead of failing. Other players, another year or another place keep the strict rule. Only RG-003 year 9 at (1388,1386), players 3 and 6, carries one. Its restart line moves from the hand class `random-driven [lane D]` (#229, oracle evidence stars-elegy 045bc90) to an ordinary sample, and the hand line is gone. TestParityVectors leaves the two alt_battle expectations (chained and restart) to the battle harness, as it does `battle`: skips.txt gains "expectation alt_battle" 2. Measured on elegy main `7aa28f7` (#233) plus this branch. Test: `TestBattleRecordHarness` (same players in another order, other players, another year, another place).
- **What a "stack" is.** MESSAGES.md's one-stack texts (0x096–0x09a) say "exactly 2 stacks". CB-039 (140 one-ship fleets a side) sent 0x09a, and CB-048 (80 ships in one fleet against 30 one-ship fleets) sent 0x098 and 0x099. So a stack there is one player's ships of one design, whatever fleets hold them (MEASURED). Elegy counts it that way; read as fleet stacks, CB-048 fails.

### Object-form wormholes compared (after the final run)

Measured on elegy main `9088500` (#198) plus the wormhole branch; counts re-measured on main `387c622` (#199, lane D's battle summaries) plus the wormhole branch.

`object` expectations of kind `wormhole` are compared now (internal/vectors `wormholeFields`), with the fields the `wormhole` form already used plus `id` and `partner`, the other end's vector id.
- **RG-002: 72 expectations, all marked sample by the vectors.** A wormhole end jiggles or jumps on draws every year (OBJECTS.md). All 72 miss, and they become 160 sample field lines: x (72), y (65), `years_since_jump` (13) and `known_to` (10). `partner`, `stability_class` and `destination_known_to` match in all 72. No deterministic line is added.
- Counts after this, on main `387c622`: missing mechanic 12 (economy 10, lane A 2), incorrect mechanic 2 (orders), random-driven 30, not compared 204 (20 comparator gaps: 14 design reads and 6 player counts; 176 battle; 8 null), and samples 1146 fields (986 + 160), 43 orders and 16 battle records.
- **Other corpora: RD-P20 and RD-P21.** Their 10 `object` wormhole expectations are year-end positions after a jiggle, which is draw-decided. Before stars-elegy #181 the vectors did not mark them as samples, so they missed on position (and RD-P21 on one `known_to`) and left the parity baseline. stars-elegy #181 (`b4c0086`) flags them `sample` (6 in rp20, 4 in rp21); with those two vectors re-synced, both pass and are back in the baseline. The re-synced files also carry the galaxy `size`, `bounds` and `density` fields; Elegy's vectors loader does not read them yet, so they change no result.

### Designs and player counts compared (after the final run)

Measured on elegy main `2ba78f1` (#201) plus the design branch.

- `design` and `starbase_design` expectations are compared with the design in the owner's ship or starbase slot at the end of the year (internal/vectors `designEquals`). The comparison covers the hull, the mass per ship, and the slots in hull-slot order, with null for an empty slot.
- Player `counts` is compared with the numbers of ship and starbase designs the player holds.
- **RG-002: all 20 lines match on the reference seed.** That is 6 design reads, 8 starbase designs (samples, all matching) and 6 counts.
- Not compared is now 184: 176 battle expectations that `TestBattleRecords` compares, and 8 nulls. No quantity Elegy models is left uncompared in RG-002.
- **Other corpora:** the parity baseline is unchanged. TK-104-A and TK-105-A were skipped before ("design read needs the order layer's design source") and now fail on T-37 (lane A), so tk2 goes from fail 0 / skip 4 to fail 2 / skip 2. Stars-elegy PARITY.md T-37 (CONFIRMED) says the original removes LBU-17 and LBU-32 from designs after generation, at electronics 7. Elegy keeps them (lane A). Since the T-37 design check (`checkDesigns`, TAKEOVER-STATUS.md), Elegy removes them too, and TK-104-A and TK-105-A pass (tk2 fail 0).

### Level-up and overcrowding messages (economy)

Elegy now sends the three economy messages it had no event for (MESSAGES.md at stars-elegy b4c0086), and `pvMessages` maps them:
- **0x026** from `EventOvercrowded`: growth shrank a planet with a hab value of 0 or more, and it survived. A planet that dies out gets no 0x026 (MESSAGES.md 0x023 vs 0x026).
- **0x050** from `EventLevelUp`, for a player without Generalized Research (0x136 otherwise, not mapped), one per level gained, at every level-up check: the year's first, the research step, research stolen by Super Stealth, and the check after drops and battles.
- **0x05f** from `EventPartAvailable`, after its level's 0x050: one per component the race may use (a Mystery Trader part only when owned) whose requirement in the field is the new level and whose other requirements the player meets (MEASURED, KX-005 R1). Hulls, starbase hulls and planetary scanners and defenses get their own ids (0x078, 0x0d0, 0x157, 0x145), which are not mapped.
- **Order within a year.** The research step announces its levels in the order the research chain gained them: the current field's levels, then the field the leftover moved to, each level followed by its own 0x05f parts. MEASURED (MESSAGES.md "Clarified 2026-10-13 (EM-1)" at stars-elegy d89b8a3; RG-002 2468: biotech 7, its parts, then weapons 8, its parts). The other checks raise each field in turn, so they announce in field order. ASSUMPTION: the part check sees the levels reached so far in that order, so a part needing two fields that rose in one year is announced at the later one.
- **Terraforming parts** get 0x05f by the same rule, from any of their fields (MEASURED, MESSAGES.md "Clarified 2026-10-13 (EM-1)" at stars-elegy d89b8a3: Gravity ±7 on propulsion 5, Radiation ±7 on weapons 5).
- Not modelled: the level messages of a Mystery Trader reward (objects/ raises those levels directly, with no 0x050 or 0x05f) and the position of the level messages from scrapping at a starbase (0x13d, then 0x050, before the production messages). EM-1 places each in its own position; they are left for the objects and production owners. Corrected 2026-10-13: an earlier version said Elegy gains no levels from scrapping. Scrapping at an owned starbase does add a level's research (`techAttempt`, engine/scrap.go), and the year's first level-up check then gains the level and announces its 0x050 and its parts.
- The comparator checked only whether a player received an id, not the order or the count (it now checks the order between ids, "Message order compared" below, but still not within one id), so the order above is pinned by engine tests (`TestLevelEventsChainOrder`, `TestTurnLevelEventsChainOrder`), not by the vectors. That is why field order also matched.

**RG-002:** economy's 10 missing-mechanic lines match on the reference seed (0x026 ×7, 0x050 ×2, 0x05f), and so do 8 chained samples. No new line appears. The ledger is missing mechanic 2 (lane A's 0x02e), incorrect mechanic 2 (orders), random-driven 30, not compared 184, and samples 1138 fields / 43 orders / 16 battle records (on main `1f03d6a`, after the `object`-form wormholes, designs and player counts).

**Other corpora:** 31 cases join the parity baseline: KX-002 G1–G3, H4, N2, P1–P3 and R1–R4; KX-003 r2; KX-004 S4 and S8; KX-005 R1, R3 and T0–T2; PQ-001 C02, C03 and C10; RD-P15; RG-001 Y2–Y6; RG-002 Y6; and TK-203. Every case "Other corpora" above lists for 0x026, 0x050 or 0x05f now passes.

### Production queue messages (economy)

Measured on elegy main `217ab4b` plus this branch.

- **0x03f** from `EventNoQueue`: production sends it to the owner of every owned planet with no production queue (MESSAGES.md 0x03f at stars-elegy d89b8a3, CONFIRMED). A planet with a queue gets none. A present but empty queue also gets none; that keeps the existing empty-queue rule in production.go, which is BINARY-ONLY.
- **0x03e** from the existing `EventQueueCompleted` (MESSAGES.md 0x03e, CONFIRMED), now mapped.
- A planet whose queue the harness cannot load runs with no queue. So for a player with such a planet, both ids are skipped ("a queue message from a production queue not loaded"), as the build messages already are. No current vector reaches this guard.
- **Other corpora:** 339 expectations that were skipped as unmapped (0x03f 298, 0x03e 41) are compared now, and every one matches on the reference seed. Every per-corpus tally and the parity baseline are unchanged.
- **RG-002:** it carries none of these ids, so the ledger is unchanged.

### Message order compared

Measured on elegy main `133816c` (#207) plus the order branch.

- **What the vectors record.** A player's `message` expectations list each id once, in the order of its first place in the year's message file. This is how stars-elegy's observed-state builders write them (tools/vectors/build_kx.py `messages`, shared by the rg, kx, pq and other builders). vectors/README.md does not state an order, so for a vector built another way the order is an ASSUMPTION.
- **What is compared** (internal/vectors `pvMessageSeq`, `messageOrderCheck`). Elegy's events are turned into each player's message sequence. Messages are appended as the year sends them (MESSAGES.md "Order"). A step that deletes an earlier message takes it out of its place: a fleet's earlier 0x04e (MESSAGES.md 0x04e), and a planet's single-unit build count folded into a later count. Each id is then kept at its first place. Each present message is checked against the nearest message listed before it that Elegy also sends; that message must come first. A message that is not a sample is placed only against one that is not a sample either, so an order that draws decide stays a sample.
- **Result: one order mismatch, draw-dependent.** Over the 8 seed variants of every corpus there are 1534 order comparisons: the seven RG-002 vectors 554, KX-002 232, KX-005 200, PQ-001 104, MF 104, CB7 68, SL 56, GT 40, WU 40, KX-003 32, KX-004 32, and KX-001, RD-P, TK-203, XF 16 each, TK5 8. 728 of them place a 0x050 or 0x05f and 772 a 0x03e or 0x03f. Of the corpora outside build_kx's, only TK-203's order is in question, so the ASSUMPTION above matters only there.
  - **TK-203 (random 6 of 8, was pass):** the vector lists player 1's 0x050 before 0x03f, with no stream label and no sample flag. On seed variants 0 and 6 Elegy sends 0x050 after 0x03f. Elegy's level comes from scrapping at the starbase only when the tech attempt succeeds (`techAttempt`, a draw; TAKEOVER.md scrapping, "as in battle", COMBAT.md "Tech from battle"). When the attempt fails, the level comes from the research step after production. The original's attempt is a draw too: it gave the level in 8 of the 12 cycle settings (TAKEOVER.md, CONFIRMED TK-203). So in the original, too, the place of 0x050 depends on draws. tk3 is not a build_kx vector: its unlabelled list merges 12 streams, so its order is the ASSUMPTION case above. This is a flagging question for lane F (mark TK-203's 0x050 a sample, or compare its order only through MS-1's per-stream `message_sequence`), not an Elegy mechanic.
  - RG-002's ledger is unchanged, so its per-lane count is 0.
- **What the order check cannot see.** Only ids Elegy maps can be placed. The vectors also list an id only once, without its parameters, so the order among several messages of one id is not recorded. Which field's level-up comes first is one such order, which is why the economy section above pins it with engine tests. MS-1's `message_sequence` now carries that sequence; see "Message sequences compared" below.

### Vector re-sync from stars-elegy main (19896cd)

Every corpus was copied verbatim from stars-elegy main `19896cd`. The only differences from the earlier copies are new fields: `default_queue` and `default_leftover_to_research` on 866 player records, 542 `message_sequence` expectations (MS-1, stars-elegy #186), and the galaxy `size`, `bounds` and `density` in `initial_state.game` (#180). Removing those fields, each file equals its earlier copy. Measured on elegy main `397cd2e` (#206) plus the re-sync.
- **Default queues now load**, so a captured planet or new colony gets its owner's default queue in the harness:
  - **TK-106-A2 joins the parity baseline.** War Monger keeps all five default items (stars-elegy PARITY.md T-26, CONFIRMED).
  - **TK-108-A and TK-108-C leave it: incorrect mechanic (lane A).** T-26 says a Claim Adjuster owner skips items 4–5, the auto minimum and maximum terraform, and an Alternate Reality owner skips items 0–2 (auto mines, factories, defenses). These are item ids. `newColony` (engine/takeover.go) skips queue positions instead. TK-108's default queue is factories, mines, defenses, minimum terraform, maximum terraform, so Elegy drops only the maximum terraform (position 4) and keeps the minimum terraform: `queue = [... {16 2 0 0}], want [...]`.
- **`message_sequence` was not compared yet** (compared since; see "Message sequences compared"). In RG-002 it adds 84 not-compared ledger lines, one per player and year. The ledger's other lines are unchanged apart from their expectation numbers.
- The galaxy fields are not read yet (lane A's clamp PR).

### Alchemy message (economy)

Measured on elegy main `3d33192` (#211) plus this branch.

- **0x08c** from the existing `EventAlchemy`, now mapped (MESSAGES.md 0x08c at stars-elegy d89b8a3, CONFIRMED KX-001, PQ-001). Production emits the event for completed alchemy items and for alchemy bought ahead of the next item.
- **Other corpora:** 27 expectations that were skipped as unmapped are compared now. All 27 are "present", and all match on the reference seed. Every per-corpus tally and the parity baseline are unchanged.
- **Order:** with 0x08c mapped, the order check (section above) makes 1662 comparisons over the 8 seed variants, up from 1534. The 128 new ones each place a 0x035 (40) or 0x036 (88) after a 0x08c, and every one holds: no per-corpus tally changes.
- **RG-002:** it carries no 0x08c, so the ledger is unchanged.

### Orders reduced to their limit (economy)

Measured on elegy main `fd606a1` (#209) plus this branch.

- **0x12a** from the existing `EventOrderClipped` of a factory, mine or defense order, and **0x12f** from the same event of a terraform order (MESSAGES.md 0x12a, CONFIRMED KX-002, PQ-001; 0x12f, CONFIRMED KX-002; both at stars-elegy d89b8a3). Production emits the event when it cuts an order to what still fits, including a cut to 0.
- **Other corpora:** 6 expectations that were skipped as unmapped are compared now (0x12a 4, 0x12f 2). All 6 are "present", and all match on the reference seed. Every per-corpus tally and the parity baseline are unchanged.
- **Order:** the order check makes 1742 comparisons over the 8 seed variants, up from 1710 on main. The 32 new ones place a 0x036 (16) or 0x038 (8) after a 0x12a, and a 0x03e (8) after a 0x12f. Every one holds: no per-corpus tally changes.
- **RG-002:** it carries neither id, so the ledger is unchanged.

### Default queue settings (economy)

Measured on elegy main `fd606a1` (#209) plus this branch.

- With #211's re-sync, 9 of the 11 "player default queue settings" skips are compared (section "Vector re-sync" above).
- The other 2 were one RG-001 planet that was owned at the start of the year and had no owner at its end, counted in two cases. A planet left with no owner takes no default (Elegy clears its queue and setting when it empties, engine `emptyPlanet`), so the harness now compares its queue and `leftover_to_research` as it does an unchanged owner's. Both expectations match on every seed variant.
- No skip of this kind is left. Every per-corpus tally, the parity baseline and the RG-002 ledger are unchanged.

### Research stolen and Generalized Research level messages (economy)

Measured on elegy main `426bb35` (#216) plus this branch.

- **0x159** from the existing `EventResearchStolen`, one per field (MESSAGES.md 0x159 at stars-elegy d89b8a3, CONFIRMED tk2/tk113, KX-003).
- **0x136** from `EventLevelUp` for a player with Generalized Research, in place of 0x050 (MESSAGES.md 0x136 and "0x050 vs 0x136" at d89b8a3; the id is BINARY-ONLY there, and this vector match is its first comparison here).
- **Other corpora:** 21 expectations that were skipped as unmapped (0x159 20, 0x136 1) are compared now, and all match on the reference seed. The order check makes 1910 comparisons over the 8 seed variants, up from 1742 on main. 244 of them have 0x159 or 0x136 on one side, and every one holds. Every per-corpus tally, the parity baseline and the RG-002 ledger are unchanged.
- **0x078, 0x145, 0x157** were left unmapped here because of the order within a level. They are mapped in "Component messages within a level" below.

### Component messages within a level (economy)

Measured on elegy main `97ac630` (#217) plus this branch.

- **Order.** After a level message, Elegy now announces that level's components in the order of MESSAGES.md "Order of the component messages within one level" (stars-elegy f5bffd8, which adds AO-1 to #192's f2778b3). The order is: engines, scanners, shields, armor, beam weapons, torpedoes, bombs, mining robots, mine layers and orbital parts; then starbase hulls; then electrical, mechanical and terraforming parts; then ship hulls; then planetary scanners, planetary defenses and the Genesis Device. Within each category, items go in index order. It used to be catalog order (`announceOrder`, engine/research.go). Labels, sentence by sentence:
  - The category order is BINARY-ONLY as a whole, and the pairs observed in one level are MEASURED (KX-005 R1, KX-003 r2, KX-005 T0, RG-001 2432–2435).
  - CONFIRMED by prediction (AO-1, 2 of 2 streams):
    - shields before mining robots;
    - the starbase hull after the mining robots and before the ship hulls;
    - electrical before terraforming;
    - planetary scanners before planetary defenses.
  - Index order within a category is BINARY-ONLY in general. It is MEASURED for ship hulls (RG-001 2434) and CONFIRMED by prediction for scanners and ship hulls (AO-1).
  - A Total Terraforming race is not told about Gravity, Temp or Radiation Terraform ±3 (`unannounced`). This is CONFIRMED by prediction for AO-1 (c) and BINARY-ONLY beyond it.
  - The Genesis Device's place is BINARY-ONLY (NOT RUN in AO-1).
  - `TestLevelEventsAnnounceOrder` reproduces the MEASURED energy 10 and construction 3 orders and the four AO-1 starts exactly. `TestAnnounceOrder` pins the placements AO-1 did not run.
- **Mapped:** 0x078 for a ship hull, 0x145 for a planetary defense, and 0x157 for a planetary scanner. The Genesis Device now sends 0x05f. Starbase hulls (0x0d0) stay unmapped: the MESSAGES.md table still labels that id BINARY-ONLY, although AO-1 (b) observed it, and no vector expectation names that id.
- **Other corpora:** 23 expectations that were skipped as unmapped (0x145 17, 0x078 4, 0x157 2) are now compared, and all match on the reference seed. The order check makes 2094 comparisons over the 8 seed variants, up from 1910 on main. 184 of them have 0x078, 0x145 or 0x157 on one side, and every one holds. That includes the KX-003 r2, KX-005 T0–T2 and RG-001 Y4 orders that failed under catalog order.
- Every per-corpus tally, the parity baseline and the RG-002 ledger are unchanged. No compared expectation is decided by a BINARY-ONLY sentence alone: no vector has a starbase hull, a Genesis Device or a Total Terraforming race's ±3 announcement.

### Mine-sweep messages (lane D)

Measured on elegy main `bd0ed07` (#231) plus this branch.

- **Mapped** from Elegy's existing sweep step (objects `Sweep`, OBJECTS.md "Sweeping"), by MESSAGES.md at stars-elegy 1d4c97b:
  - **0x0f4**: a starbase swept mines. `EventMinesSwept` with a planet, to its owner (CONFIRMED ob/ob007, mf/mf01).
  - **0x0c2**: a fleet swept mines. `EventMinesSwept` with a fleet, to its owner (CONFIRMED ob/ob010, cs).
  - **0x0be**: someone swept your field. The new `EventMinefieldSwept` (kind 96) goes to the field's owner right after each sweeper's event (CONFIRMED ob/ob010, cs, mf/mf01). One per sweeper per field, so two sweepers give two, each with its own amount: BINARY-ONLY (MESSAGES.md "Duplicates"; no run with two sweepers in one field is cited). A fleet or starbase that cannot sweep sends none: ASSUMPTION (the triggers name a sweeper that sweeps; no vector has a non-sweeper in an enemy field). `TestSweepMinesEvents` pins both.
- **Slots.** The sweeper (planet or fleet) and the amount are compared. ~~The field owner, type and centre slots are not compared.~~ Corrected 2026-10-13: every slot is now compared, by the table in MESSAGES.md "Sweep slots" (stars-elegy e1b2ad9, MEASURED SW-1). The sweep events carry the field's owner, type, centre and number (`Event.Axes`).
  - 0x0c2 and 0x0f4: field owner (the plain player number), type (0 standard, 1 heavy, 2 speed bump) and the centre's x and y.
  - 0x0be: the field's object id, owner·512 + number (MESSAGES.md "Object ids in slots"), then type, x and y.
  - Not compared: 0x0f4's type slot when a starbase sweeps a speed bump (NOT RUN in the oracle), and every slot of a 0x0c2 whose fleet is gone. That the owner slot never carries a flag is UNRESOLVED; Elegy writes the plain number.
  - RG-003-Y11's restart 0x0f4 now matches in full: [95 82 0 5 0 1548 1298], player 5's standard field at (1548,1298). Its "not compared" line is gone from the ledger and the census.
- **Other corpora:** 32 expectations that were skipped as unmapped (0x0f4 18, 0x0be 14) are now compared. The MF lane's 28 match on the reference seed. Both `message_sequence` entries of 0x0f4 that were skipped as unmapped (RG-003 year 11, chained and restart) are now compared: the restart's sweeper and amount (82) match exactly, and the chained one is a sample (below). The restart's unknown slots stay a "not compared" tally. No vector names 0x0c2. Every per-corpus tally and the parity baseline are unchanged.
- **RG-003:** the restarts' 0x0f4 match (y11, y12). In the chained run, y11 and y12 have no starbase sweep in Elegy: 3 sample lines, from the chained state after earlier draws.

### Salvage and bombing message slots (lane D)

Measured on this branch, by MESSAGES.md at stars-elegy e1b2ad9.

- **0x05b** (a fleet scrapped in deep space): the salvage slot is now compared. `EventFleetScrapped` carries the new salvage's number (`Axes`), and the slot is its object id, 0x2000 + owner·512 + number (MESSAGES.md "Object ids in slots", MEASURED SV-1). When the minerals overflow into several objects the event names the first: ASSUMPTION (no run has overflowed a scrapped fleet's salvage). The fleet slot stays not compared: its owner bits are UNRESOLVED (the contradiction in MESSAGES.md). RG-002/RG-003 year 1's salvage 8192 matches in all 4 entries.
- **0x08f / 0x090** (bombed empty): the fleet slot and 0x08f's fleet focus are now compared. `EventBombed` names the fleet whose plan started the pass (`Fleet`) and carries its number and the count of the bomber's fleets in orbit (`Axes`), saved at bombing time because the fleet can be gone by the end of the year (RG-002 year 4's was). All 16 sequence entries that skipped the slot match (RG-002 y4, RG-003 y4 and y10, chained and restart).
  - With two or more of the bomber's fleets in orbit, the message is 0x17c/0x17d (MESSAGES.md, CONFIRMED tk2/tk121), with the fleet slot not compared. Which fleets count (all in orbit, or those with bombs) and which fleet 0x08f names when the pass was started by a fleet without bombs are ASSUMPTION: every vector case had one fleet in orbit. No vector names 0x17c or 0x17d.
- Census: "0x05b salvage and fleet slots" (4) becomes "0x05b fleet slot" (4); "0x08f and 0x090 bombing fleet" (16) is gone.
- **Updated 2026-10-13 (FS-1):** 0x05b's fleet slot is compared too. For a fleet of one design it is design·512 + fleet number, the design being the fleet's ship-design slot (CONFIRMED by prediction, stars-elegy FS-1, 28a3175; the bits are not the owner). `EventFleetScrapped` carries the fleet's number, its lowest design slot and its count of designs. A fleet of several designs gave 0x2000 + 2·512 + 6 once (MEASURED); which design it names and what bit 13 means are not known, so that slot stays not compared ("0x05b fleet slot of a fleet of several designs"). RG-002/RG-003 year 1's fleet slot 6 (design 0) matches in all 4 entries. Census: "0x05b fleet slot" (4) becomes "0x02e mineral and location slots" (4), the next unknown slot in the same sequence entries.
- **Updated 2026-10-13 (0x059):** 0x059, a fleet scrapped onto the surface of a planet without a starbase whose owner lacks Ultimate Recycling (MESSAGES.md, CONFIRMED tk2), is mapped for the scrap. `EventFleetScrapped` at a planet now carries the same fleet axes as in deep space and how the planet recycled it (starbase and Ultimate Recycling bits), which picks 0x059. Its fleet slot is design·512 + fleet number for a fleet of one design (CONFIRMED by prediction, stars-elegy FS-1 p and q, 28a3175); a fleet of several designs and a design with no slot are not compared, as for 0x05b. The kT and planet slots are not compared: the slot count (fleet, kT, planet) is MESSAGES.md's slot column, and no public record gives their values. The original also sends 0x059 on a colonization (cs/cs-003); Elegy does not, and no vector carries one, so it is not compared (a vector that had one would fail as not sent). 0x05a, 0x05c, 0x05d and the planet owner's 0x140–0x143 stay unmapped. Other corpora: tk3/tk203's one 0x059 presence expectation is compared now and matches; the parity baseline and every RG ledger are unchanged. Census: skips.txt loses "message 0x059 has no event mapping" (1).
- **Updated 2026-10-13 (0x05a, 0x140, 0x141):** the scrap's second message, to the planet's owner, and the starbase scrap are mapped. `EventFleetScrapped` at a planet now also carries the planet's owner at the scrap and whether the starbase's tech roll gave that owner a technology.
  - **0x140** goes with 0x059 to the planet's owner, unless the planet is unowned. Its fleet slot is owner·512 + fleet number: MEASURED (FS-1, 2 values: 7 for 0/7 and 518 for 1/6). In both cases the fleet's owner, the recipient and the planet's owner were the same player, so which of them the bits name is not known; Elegy uses the fleet's owner.
  - **0x05a** goes to the fleet's owner at a starbase whose owner lacks Ultimate Recycling. **0x141** goes with it to the planet's owner when the scrap gave no technology (0x13c/0x13d otherwise, still unmapped). No fleet slot of either is measured, so only their presence and planet focus are compared.
  - Fleets of several designs, the kT and planet slots, and the Ultimate Recycling messages (0x05c, 0x05d, 0x142, 0x143) are not compared.
  - Other corpora: the 5 presence expectations (0x05a 2, 0x141 2, 0x140 1, in tk3/tk203 and xf/wp-1) are compared now and match. The parity baseline and every RG ledger are unchanged. Census: skips.txt loses "message 0x05a has no event mapping" (2), "message 0x140 …" (1) and "message 0x141 …" (2).

### Invasion messages (lane A)

Measured on elegy main `030875d` (#252).

- **Mapped** by MESSAGES.md "Invasion slots" (stars-elegy #204, e1b2ad9; MEASURED: TK-301, TK-302, GC-1, RG-003, RG-004):
  - **0x004** keeps its trigger (`EventDropRepelled`, to the planet's owner) and now compares every slot: planet, troops (two slots), attacker + 0x30. Troops are what the attacker dropped there that year, which the event already counted.
  - **0x00c** to the winner (former owner + 0x20, planet) and **0x007** to the former owner (winner + 0x30, planet, troops in two slots), from the new `EventPlanetCaptured` (kind 99). Its troops are everything the winner dropped there that year (GC-1: 560 for 395 + 165).
  - The player flags 0x30 and 0x20 are MEASURED values; reading them as article flags over the player number is BINARY-ONLY, and the 0x0f mask is inferred.
- **Order.** A fleet whose unload task dropped colonists on another player's planet is told it has finished (0x04e) after every drop of the phase is resolved. MEASURED, one run each: in TK-501 and WU-CAP that fleet's 0x04e came after the capture's 0x00c, while in TK-201 the mineral unloads' 0x04e came before the capture. Inferred: that the drop is what decides it. TK-201, TK-501 and WU-CAP pin it: without the hold, TK-501 and WU-CAP fail on order once 0x00c is compared; holding every 0x04e back fails TK-201.
- **Other corpora:** 14 expectations that were skipped are now compared (0x007 6, 0x00c 4, and the `message_sequence` skips of 0x004's slots 2 and 0x007 2). Every per-corpus tally and the parity baseline are unchanged.
- **RG-003:** the six "not compared" lines go: y7's 0x004 [51, 550, 0, 49] and y12's 0x007 [49, 49, 600, 0] match, in the chained run and the restarts.
- **RG-004:** the 26 "not compared" 0x007 lines go (24, plus the two y4 #224 sequences that stop at 0x007 since the 0x040 PR). The 13 in the restarts (years 2, 3, 4, 5, 8, 9, 10) match. Of the 13 in the chained run, 6 match (y2 #233 #234, y3 #241, y5 #226 #227, y9 #231) and 7 become 8 sample lines, from the chained state after the year-1 divergence: y3 #242 (sent, want not sent), y4 #223, y4 #224's two 0x007 entries, y8 #238 and #239, y9 #232, y10 #241 (not sent).
- **Census:** skips.txt loses "message 0x007 has no event mapping" (20), "message 0x00c has no event mapping" (4), "message_sequence 0x004 colonists and player slots not compared" (2) and "message_sequence entry 0x007 has no event mapping" (14). No case changes status; the chained RG-004 cases that already failed (2468, 2469, 2473, 2474, 2475) list the new sample differences. baseline.txt is unchanged.

### Deferred: client estimators (milestone 3)

The economy area's 69 `client_estimate` expectations stay skipped, by the coordinator's decision of 2026-10-13: they are display estimators, and they belong with the Report and UI work of milestone 3, not with engine parity. The harness reports them as "client estimate <screen> <field>".

| Count | Reading |
|---:|---|
| 19 | production_tile completion |
| 5 | production_dialog completion |
| 10 | status_tile population |
| 10 | planet_summary_report value |
| 10 | planet_summary_report mining_rate |
| 15 | research_dialog: resources_needed 4, annual_resources_from_all_planets 3, estimated_time_to_completion 3, next_year_projected_research_budget 3, next_field_choice_after_the_year 1, total_resources_spent_on_research_last_year 1 |

Counts are from the reference seed on main `217ab4b`.

### Message sequences compared

`message_sequence` (vectors README.md "Message sequences", MS-1) is now compared: each player's messages of the year, in file order, with focus and slot values. Measured on elegy main `426bb35` (#216) plus #214 and this change.
- **The rule (vectors README.md, order rule).**
  - The entries without `sample` must all be sent, in the same relative order, with the same focus and slots.
  - When no entry is a sample, nothing else may be sent.
  - A sample entry is not compared.
- **What Elegy can compare.**
  - Only the ids Elegy maps (`pvMessageIDs`). An entry of another id makes the expectation a skip, named by its first such id ("message_sequence entry 0x07b has no event mapping"), and any mismatch is still reported.
  - Slots follow MESSAGES.md "Catalogue" in stored order. A colonist amount takes two slots, low word then high word.
  - Planets are the vector's planet numbers. A fleet is `owner << 9 | number` (ORACLE.md, "Fleet parameters are object ids"), and its focus sets bit 15.
  - A computer player keeps only 0x007, 0x023, 0x040 and 0x08f (MESSAGES.md "Computer players"). The one setting that keeps everything is read as unused (ASSUMPTION).
- **Slots Elegy's events do not carry.** These are skipped by name, and the rest of the message is still compared:
  - 0x050's third slot, the field researched next;
  - 0x05f's part focus and category-mask slots, because only terraform's mask (0x2000, EM-1) is published;
  - the bombing fleet of 0x08f and 0x090;
  - 0x05b's salvage and fleet slots;
  - 0x004's colonists and player slots;
  - 0x02e's mineral and location slots, because MESSAGES.md does not say which word of a planet location holds the planet (updated 2026-10-14: compared since stars-elegy CM-1, 4f59d8d);
  - a battle summary's focus and slots;
  - 0x04e's and 0x02e's fleet when the fleet is gone by the end of the year.
- **Result.**
  - Of the 249 sequences in the reference streams, 183 are compared and 66 are skipped by named reason (engine/testdata/vectors/skips.txt).
  - The parity baseline does not move. The new mismatches are all in RG cases that already fail.
- **RG-002 ledger:**
  - Not compared 268 → 190: 84 lines become 6, the 0x05b, 0x05f and 0x08f slots, each in the chained run and its restart. Y1's 0x02e (#213) matches with its fleet and amount.
  - Random-driven 30 → 31: Y3's 0x036 count, 11 against Elegy's 10 (5 of 8 variants match).
  - Sample 1191 → 1197: the chained run's 0x036/0x038 counts in years 3, 4 and 6, which follow the factory and mine counts that already differ.


### RG-003: 12 chained years

RG-003 (vectors README.md "rg") is RG-002 continued to 12 chained years, 2462 to 2474. Player 0's orders in years 7 to 12 were injected (stars-elegy experiments/rg003/README.md). The vectors are copied verbatim from stars-elegy main `f38c843` (#191, #197, #203). The ledger is `engine/testdata/vectors/rg/rg003-ledger.txt`, with RG-002's classes. `TestRG003Ledger` checks it, `TestBattleRecords` checks its battle lines, and `RG002_LEDGER=write` rewrites it. Years 1–6 match RG-002 line for line once the names are replaced. Measured on elegy main `896a546` (after #232 WK-1, #234, #236, #237 ship moves, #238, #235, #240 GC-1 and #239) with the year-6 rebuild below.

**Totals (5947 lines).** Corrected 2026-10-13: this heading said "5946 lines"; the table below sums to 5947.

| class | chained run | 12 restarts | total |
|---|---:|---:|---:|
| sample | 5291 | 451 | 5742 |
| matched | 8 | 100 | 108 |
| not compared | 11 | 11 | 22 |
| random-driven | 0 | 75 | 75 |

- **No deterministic hand lines remain.** Every compared difference in the restarts is random-driven (it matches on some seed variant) or a sample.
- **Chained run:** every compared difference is a sample. The run diverges in year 1 and samples carry it forward. Its only matched lines are 8 battle records.
- **Restarts:** "matched" counts battle records. Three random-driven lines carry their draw (Y6 restart, "RG-002 and RG-003 year 6 rebuilt" below).
- **GC-1 (#240, `0ca6c3b`)** cleared the last hand line, y7 planet 51's population (15083 against 15084, the per-drop ground-combat floor), and the 11 chained sample lines that carried it forward (planet 51's population and excess in years 7–12).
- **Ship moves (#237, `6cde505`)** cleared the y12 battle at (1288,1347), which now matches: 3 tokens, 19 of 19 hits.
- **WK-1 (#232, `0959671`)** cleared the 6 waypoint lines (y7 fleet 45, y8 fleets 81, 82, 84, 86 and 91).
- **#238** compares the 0x0f4 slots, which removes the y11 restart's not-compared line.
- **stars-elegy #203** adds an `alt_battle` expectation at RG-003 y9 (1388,1386), in the chained run and in the restart. The vectors comparator leaves it to the battle harness, so it stays 2 not-compared lines there. Lane D's #236 consumes it in the battle harness: the extra battle at (1388,1386) is now an ordinary sample, and its random-driven hand line is gone.

"No deterministic hand lines" holds for compared fields only. The 22 not-compared lines stay unjudged, and the section below sorts them by cause and owner.

### RG-004: a second game, 12 chained years

RG-004 (vectors README.md "rg", stars-elegy experiments/rg004/README.md) is a second representative game. Seven custom races (WM, SD, IT, JoaT, PP, IS, HE) are played by expert computer players. It runs 12 chained years, 2466 to 2477, after 65 unrecorded warm-up years, with no injected orders. Its 13 files are copied verbatim from stars-elegy main `f38c843` (#203). The ledger is `engine/testdata/vectors/rg/rg004-ledger.txt`, with RG-002's classes. `TestRG004Ledger` checks it, `TestBattleRecords` checks its 110 battle records (55 in the chained run, 55 in the restarts), and `RG002_LEDGER=write` rewrites it. Measured on elegy main `40b27ba`.

**One harness change.** In year 7, player 6's order file deletes waypoint 1 of fleet 20. In the chained run, Elegy's fleet 20 has no waypoint 1 at that point. That used to stop the whole chained run ("order waypoint_delete index", 1447 not-compared lines). Now such a delete is recorded as a refused order, as an engine refusal is, and the year goes on (ELEGY CHOICE, harness). A delete of waypoint 0 still fails the load. The chained year-7 refusal is a sample, and no restart hits it. The same refusal changes RG-003's baseline: chained years 10–12 (2472–2474) used to stop on this load error with 2 of the 8 seeds and fail with the rest, which the baseline listed as `random 0`. They now fail with every seed, so those 3 entries leave the baseline. None of them passed before or passes now.

**Totals (2893 lines, main `513a146` plus lane D's #242).**

| class | chained run | 12 restarts | total |
|---|---:|---:|---:|
| sample | 2523 | 263 | 2786 |
| matched | 4 | 47 | 51 |
| not compared | 19 | 19 | 38 |
| random-driven | 1 | 17 | 18 |
| incorrect mechanic | 0 | 0 | 0 |

- **Battle records:** 51 matched, 63 samples and 2 random-driven. None is a hand line. Since #236, the y3 restart's extra battle at (1365,1400) is a sample, not random-driven, because the vector has an `alt_battle` there.
- **Not compared (38):** 0x007 (24 lines: 14 message, 10 sequence entry) and 0x040 (8 lines) have no event mapping. Also 2 `alt_battle`, 2 `design_gone`, and 2 sequences of sample entries only.
  - Updated 2026-10-13 (0x040 PR): 0x040 is mapped, so RG-004 has 32 not-compared lines (16 / 16) and 2887 in all. 0x007 has 26 (14 message, 12 sequence entry).
- **Baseline:** RG-004 has 13 entries. All 12 restarts and chained year 1 (2466) pass with some seeds (`random k`; Y6 with 1 of 8, Y10 with 7 of 8, Y4 with 5 of 8). Chained years 2–12 fail on samples.

**Hand lines: 0 deterministic.** The 16 incorrect-mechanic lines first found, all in restarts, are cleared: lane A's 14 by #244, lane D's 2 by #242.

| year | object | lines | cause | owner |
|---|---|---:|---|---|
| y4–y12 | player 0's order: delete ship design slot 0 | 9 | Cleared (lane A): Elegy refused the delete ("no such object") because the slot has been empty since y4. The original's planner sends it every year and nothing records a refusal; the state is unchanged either way (MEASURED). Elegy now accepts a delete of an empty slot and changes nothing. That the original accepts it is inferred. | lane A |
| y4 | fleets 10 and 23 (player 6) | 5 | Cleared (lane A): Elegy's fleet numbers matched; its internal fleet key did not. A ship move in the orders empties start fleet 23, and the new fleet 10 at planet 11 took that fleet's internal id, so the harness read it as fleet 23. Elegy no longer reuses, within a year, the id of a fleet that went away that year (ELEGY CHOICE; KERNEL-STATUS.md). | lane A |
| y6, y10 | player 6's minefields 2 and 3 | 2 | Cleared (lane D): `known_to` lacked player 0. Each field is centred on a player 0 planet with no scanner, which now sees it (co-location; see below). | lane D |

**Resolved: a minefield laid on a planet with no scanner (lane D).** Two hand lines were player 6's minefields 2 (restart Y6) and 3 (restart Y10): the original added player 0 to `known_to` and Elegy did not. Each field was laid that year by a player 6 fleet in orbit, so its centre is exactly the position of a player 0 planet with no planetary scanner (planet 47 at (1208,1314) in Y6, planet 23 at (1118,1277) in Y10). No other player 0 viewer was within range.
- **Reading.** Under the co-location rule (SCANNING.md "Co-location", LEGACY BUG, `Legacy.Colocation`), every test passes at distance 0. Elegy now treats an owned planet with no scanner as a viewer with ranges 0 for space objects, so it sees a minefield centred on it. The restart results are MEASURED. The mechanism is CONFIRMED by prediction (stars-elegy `898c526`, MV-1: player 0 at tech 3 with electronics 0, standard fields, one year, one planet per case): with no fleet anywhere near, a field centred exactly on the planet was seen, and fields at d² 1 and 1280 containing it were not. So it is co-location, not sight of the laying fleet.
- **Not seen when merely inside.** RG-002 and RG-003 hold the counter-case. Player 3's field 2 contains player 6's scannerless planet 6 at d² 1280. In 15 restarts (RG-002 Y1–Y6, RG-003 Y1–Y9) it was unknown to player 6 at the start of the year and stayed unknown (MEASURED; SCANNING.md already says planets do not get the inside rule). In RG-003 Y10 player 6 learns it because its fleet 13 ends the year inside the field, 22 ly from the centre (the fleet inside rule).
- **Effect.** Both hand lines go, and so do 10 chained sample lines on the same fields. Restarts Y6 and Y10 now pass with some seeds (baseline `random 1` and `random 7`). No RG-002 or RG-003 line moves. Packets, salvage and wormholes at a scannerless planet's exact position would also be seen; MV-1 did not run them and no vector has one (ASSUMPTION).

### RG-005: the out-of-sample validation game

RG-005 (vectors README.md "rg"; stars-elegy experiments/rg005/README.md and PARITY.md "Representative game (RG-005)") is the out-of-sample check of the release candidate. Five custom races (CA, SS, AR, WM, SD) are played by expert computer players on a medium map, 12 chained years (2466 to 2477) after 65 unrecorded warm-up years, with no injected orders. Its 13 files are copied verbatim from stars-elegy main `419229a` (#212). The pre-run record, stars-elegy experiments/rg005/PREDICTION.md, was pushed alone in `1019fa7` at 02:19:31Z on 2026-10-14, and the oracle run started at 02:19:47Z. The ledger is `engine/testdata/vectors/rg/rg005-ledger.txt` (12 restarts, 36 battle records), checked by `TestRG00xLedger` and `TestBattleRecords`.

**Three tallies.** The first run is the record commit `9bb2cb6`, whose parent is the candidate `8584f92`: testdata and registration only, every line as found. Fix A is a harness change (below), regenerated on main `d1edb22`. The hand classes then class every UNCLASSIFIED line. "Early" is the restarts of years 1–6 plus chained year 1, the only lines a CONFIRMED-by-prediction verdict may rest on; the rest are MEASURED.

| ledger lines | first run (`9bb2cb6`) | after fix A | after hand classes |
|---|---|---|---|
| UNCLASSIFIED (total / early) | 25 / 8 | 19 / 2 | 0 / 0 |
| state-import error [lane B], cause UNRESOLVED | 0 / 0 | 0 / 0 | 19 / 2 |
| random-driven | 12 / 5 | 11 / 4 | 11 / 4 |
| not compared | 4 / 1 | 19 / 3 | 19 / 3 |
| sample (fields, orders, battles) | 2424 / 116 | 2369 / 111 | 2369 / 111 |
| matched (battle records) | 21 / 12 | 21 / 12 | 21 / 12 |
| total | 2486 | 2439 | 2439 |

Field expectations, chained / restarts (reference seed, 8 seeds for random-driven):

| | strict matched | strict random-driven | strict missed | sample matched | sample missed | not compared |
|---|---:|---:|---:|---:|---:|---:|
| first run | 182 / 2449 | 2 / 9 | 2 / 17 | 1652 / 145 | 873 / 91 | 2 / 2 |
| after fix A | 184 / 2453 | 2 / 9 | 0 / 12 | 1654 / 144 | 859 / 90 | 14 / 5 |

- `TestParityVectors`, rg corpus: first run fail 65, random 19, random with the reference seed failing 10 (pass 0, measured pass 2); after fix A fail 62, random 20, random with the reference seed failing 12.
- No load error, no panic, and no recorded order refused in a restart or in chained year 1. The chained runs refuse 56 orders after year 1, all samples.
- The record commit's message gave the case tally as "(17 before RG-005…)". Corrected 2026-10-14: the rg corpus went from fail 48 to 65 and random 13 to 19.

**Fix A (harness): a Trader by its vector id.** A vector Trader id is the original's object slot. RG-005 year 1 has slots 2 and 3 only, and the harness indexed Elegy's dense `objects.Space.Traders` by the id, so both loaded Traders read as missing (7 early misses and 1 early random-driven line, all "trader 2/3 missing"). A loaded id now maps to its rank among the loaded ids (`pvTraderIndex`, `TestTraderVectorIDs`). Where the dense list cannot say which slot a Trader holds, because a Trader left during the run or the compared id was not loaded and one appeared, the comparison is not compared with that reason (15 expectations), never compared against the wrong Trader. The original gives a new Trader the lowest free slot (RG-005 y4: id 0 after slots 2 and 3 left; y7: id 1), while Elegy appends. Classes that moved (old → new):
- early, `trader 2/3 missing` UNCLASSIFIED (7) and random-driven (Y2 #191) → matched, except Y2 #192, which now compares Trader 3 and misses on `met` (UNCLASSIFIED → UNCLASSIFIED, new detail);
- chained y2 and y3 #191/#192 samples → matched, or the same `met` sample miss;
- Y4 #180 sample (Elegy's loaded Trader compared against the original's new Trader 0) → sample `trader 0 missing`; Y4 #181/#182 `object_gone` (silently passed before) → not compared, a Trader left;
- Y7 #190 sample → not compared, a Trader appeared; chained y7–y12 Trader samples (14) → not compared, a Trader left.

**Hand classes: 19 lines, 12 expectations, every one missing on all 8 seeds.** Each is `state-import error [lane B]` with cause UNRESOLVED, and each stays counted as a deterministic miss. The leading candidate: the vectors do not carry a computer player's level, so the harness loads every computer player at level 0 (Easy), and an Easy player makes no Mystery Trader planet trade. Lane D's retrodiction (MEASURED, level 3 for every computer player, on the first run's mapping) turns 13 expectations from miss to match: Y2 #121 and #151, and Y11 #100, #118, #177, #179, #213, #214, #219, #225, #231, #237 and #243. Y2 #192, compared only since fix A, matches in lane B's run with the same change. The 12:
- Y2 #121 planet 218 `surface_minerals`; Y2 #192 Trader 3 `met` = [], want [1];
- Y11 #100 planet 71 and #118 planet 162 `surface_minerals`; #177 player 2 `tech` (biotechnology, construction, weapons); #213 and #214 Traders 0 and 1 `met`; #219, #225, #231, #237 and #243 player 2's `score_record` (score and tech level sum) as players 0–4 see it.

**Scored against PREDICTION.md, on the first comparison as found.**
- "The original creates RG5 and every chain repeat is byte-identical." Lane F's; scored in stars-elegy PARITY.md "Representative game (RG-005)".
- "RC does not hold out of sample: at least one deterministic miss or refused order in the restarts or chained year 1." CONFIRMED by prediction: 8 early expectations missed on all 8 seeds (chained y1 #180 and #181, Y1 #180 and #181, Y2 #192, Y3 #191 and #192: "trader 2/3 missing"; Y2 #121: planet 218). The record defines a deterministic miss by the harness's result, so the verdict stands as pre-registered, although 7 of the 8 were the harness's Trader ids. After fix A, the early support rests on Y2 #121 (planet 218) and Y2 #192 (Trader 3 `met`), both on the Player.Level candidate. Y11's 10 are MEASURED only.
- "At least one random-driven line, and at least one not-compared line." CONFIRMED by prediction in years 1–6: random-driven Y1 #142 (player 1 electronics research, 6 of 8) and #161 (planet 202's queue, 6 of 8), chained and restart; not compared Y2 #151 (player 1's `mystery_trader_items`, a field the harness does not map; PREDICTION.md:124–125 asks for exactly this report).
- An outcome nobody predicted, MEASURED: the deterministic misses cluster on Mystery Trader planet trades, and 7 of the first run's 8 early misses were a harness defect rather than an engine one.

**Not compared (19 after fix A):** 4 `mystery_trader_items` (Y2 player 1 and Y11 player 4, chained and restart) and 15 Trader comparisons whose slot Elegy cannot name. RG-005 is no longer out of sample for an engine or harness that carries fix A or a later change made for it.

### RG-002 and RG-003 year 6 rebuilt (stars-elegy #206, DA-1)

stars-elegy #206 (`104e8f5`) rebuilt the year-6 vectors of RG-002 and RG-003 (`rg002.json`, `rg002-y6.json`, `rg003.json`, `rg003-y6.json`). The alternative-stream host file behind them had been truncated mid-write, which flagged 459 of year 6's 476 expectations as samples; the rebuilt year has 493, 82 of them samples, so 394 expectations became strict. The four files are copied verbatim. RG-002 and RG-003 move identically.
- **No new deterministic miss.** Every newly strict expectation matches on the reference seed except three, and each of those matches on some seed variant.
- **The three, in the Y6 restart, each with the draw behind it** (`random-driven [owner]: evidence`; the vectors ledger now accepts that form on a random-driven line, as the battle ledger does since #229):
  - **fleet 6/12, waypoint 1 (lane D):** its target, player 3's fleet 2, is shot down in Elegy's battle at (1296,1356) on 6 of 8 seeds (102 of 200 fresh seeds), and the waypoint then loses its fleet target. On seeds 1 and 5 the fleet survives and the waypoint matches. The recorded battle has 0 hits, and the vector flags fleet 3/2's survival a sample.
    - **The draws are movement ties.** Each of the fleeing token's 7 moves has 7 or 8 squares tied: every square not next to the chaser scores take 30, the estimate's per-weapon floor (3 lasers × 10 ships). On 6 of the chaser's 8 moves, a diagonal and a straight step tie.
    - **The deterministic parts match the record.** Both tokens make no move in rounds 2 and 6, and the heavier chaser moves first. The fleeing token makes 7 moves and leaves on its 8th, in round 9. After round 8 the two are 2 apart, out of the lasers' reach 1, so there are no hits. Initiative and targeting do not apply, since only one side is armed. All 16 recorded moves are in Elegy's tie sets.
    - **No simple tie rule.** The record's picks are not always the first or last tie in Elegy's scan order. Once the fleeing token steps toward the chaser, from (1,3) to (2,3) with the chaser at (6,5). That is consistent with flat scoring and not with a rule that maximises distance (MEASURED, one battle).
  - **planet 102, surface minerals (economy):** mining's +1 draw (KERNEL.md "Mining", BINARY-ONLY). Germanium enters production at 54 kT on seed 0 and 53 kT on seed 7, the only seed that matches. Germanium limits the ship being built, so the ironium and boranium spent follow it (666/1249 against 668/1251).
  - **planet 102, production queue (economy):** follows the same draw: 53 kT gives 93%, as recorded, and 54 kT gives 99%.
- **Samples:** two new sample lines in the restart (planets 29 and 79, surface minerals) and two in the chained run (planets 29 and 85).
- **Baseline:** RG-002-Y6-2468 and RG-003-Y6-2468 leave it. They passed only because every year-6 miss was a sample; with the three lines above strict, the reference seed fails them. This is the honest result of the rebuild, not a regression.
- **Census:** "sample: one stream's random outcome" 2241 → 2001.
- **Before and after, per expectation.** [DA1-Y6-RECLASS.tsv](DA1-Y6-RECLASS.tsv) lists every non-battle year-6 expectation of the four rebuilt vectors with its class under the old vector and the new one: the vector's flag (sample or strict) and Elegy's result on the reference seed, or k of 8 seeds for random-driven. The Y6 restart (`rg002-y6.json`; `rg003-y6.json` is identical):

  | old class | new class | expectations |
  |---|---|---:|
  | sample, matched | strict, matched | 385 |
  | sample, missed | strict, matched | 6 |
  | sample, missed | strict, random-driven (1/8, 2/8) | 3 |
  | sample, matched | sample, matched | 34 |
  | sample, missed | sample, matched | 1 |
  | sample, missed | sample, missed | 14 |
  | strict, matched | strict, matched | 15 |
  | absent | sample, matched / missed | 8 / 9 |

  In the chained runs, year 6 gains 47 strict expectations that miss on all 8 seeds, but the ledger classes every chained year after the first as a sample, so they stay samples there (the file's header says so).

### Order accounting (RG-002, RG-003, RG-004)

Every order in the RG vectors is accounted for: it is either accepted by the engine, or refused and recorded as a ledger line (an `order` line). **No RG vector has a load error.** Counts are from the reference seed on main `f09acd9`; they are the same as on `5a735e1`. A restart's year is the year it restarts at, and its orders are the same as the chained run's for that year.

Refusal classes: N, no such object (fleet, waypoint, design slot or salvage); P, objects not at the same place; O, the object belongs to another player; R, a value out of range (a ship count); H, the harness refusal of a waypoint_delete past the end of the fleet's list (ELEGY CHOICE, harness).

**RG-002**

| year | orders | chained: refused (N/P/O/R/H) | restart: refused |
|---:|---:|---|---|
| 1 | 287 | 0 | 0 |
| 2 | 309 | 7 (7/0/0/0/0) | 0 |
| 3 | 308 | 2 (1/1/0/0/0) | 0 |
| 4 | 307 | 3 (2/1/0/0/0) | 0 |
| 5 | 361 | 9 (8/1/0/0/0) | 0 |
| 6 | 295 | 16 (15/1/0/0/0) | 0 |
| total | 1867 | 37 | 0 |

**RG-003**

| year | orders | chained: refused (N/P/O/R/H) | restart: refused |
|---:|---:|---|---|
| 1 | 287 | 0 | 0 |
| 2 | 309 | 7 (7/0/0/0/0) | 0 |
| 3 | 308 | 2 (1/1/0/0/0) | 0 |
| 4 | 307 | 3 (2/1/0/0/0) | 0 |
| 5 | 361 | 9 (8/1/0/0/0) | 0 |
| 6 | 295 | 16 (15/1/0/0/0) | 0 |
| 7 | 311 | 22 (17/5/0/0/0) | 0 |
| 8 | 301 | 18 (13/5/0/0/0) | 0 |
| 9 | 298 | 26 (18/6/1/1/0) | 0 |
| 10 | 308 | 35 (27/8/0/0/0) | 0 |
| 11 | 335 | 39 (25/13/1/0/0) | 0 |
| 12 | 311 | 43 (22/19/1/1/0) | 0 |
| total | 3731 | 220 | 0 |

**RG-004**

| year | orders | chained: refused (N/P/O/R/H) | restart: refused |
|---:|---:|---|---|
| 1 | 158 | 0 | 0 |
| 2 | 157 | 0 | 0 |
| 3 | 190 | 6 (4/2/0/0/0) | 0 |
| 4 | 162 | 6 (5/1/0/0/0) | 0 |
| 5 | 171 | 11 (7/2/2/0/0) | 0 |
| 6 | 173 | 12 (9/0/2/1/0) | 0 |
| 7 | 183 | 9 (4/2/2/0/1) | 0 |
| 8 | 161 | 7 (5/0/2/0/0) | 0 |
| 9 | 164 | 7 (3/2/2/0/0) | 0 |
| 10 | 173 | 7 (3/1/3/0/0) | 0 |
| 11 | 172 | 8 (4/0/4/0/0) | 0 |
| 12 | 185 | 11 (4/2/4/1/0) | 0 |
| total | 2049 | 84 | 0 |

- **Chained runs:** every refusal is a sample line. It follows earlier divergence, because the order names an object that Elegy's diverged state no longer has, or has elsewhere. RG-002 has 37, RG-003 220 and RG-004 84.
- **Restarts:** no corpus refuses an order in a restart.
- **Updated 2026-10-13 (main `5a735e1`):** this section first gave RG-004 93 chained and 9 restart refusals (102 order lines), the 9 restart ones being "the lane A hand lines 'empty design-slot delete refused', one in each restart from Y4 to Y12". Lane A's #244 accepts a delete of an empty design slot, which removes those 9 and the same order's 9 chained refusals (one N a year, years 4–12). RG-004 now has 84 order lines, all chained samples.

### Planet records without installations block

stars-elegy ORACLE.md (main `1d4c97b`, the `dump` notes) says: "The game omits a planet's 8-byte installations block when growth carry, mines, factories, defenses and the leftover-only box are all 0 and there is no scanner." The vector diff then writes null for `excess`, `mines`, `factories`, `defenses`, `planetary_scanner` and `leftover_to_research` where the start state had them. The comparator used to skip those fields as "planet field … null". It now reads them as 0, no scanner and false whenever the record has a population, and compares them. The sentence names all six fields (DOCUMENTED). Reading a missing block as those values is its converse, which is an ASSUMPTION: that the block is written whenever any of them is non-zero. Every field read this way matches except one (corrected below). A record whose population is also null stays skipped. The second ORACLE.md note ("Populations above 0 with no mines, factories, defenses or carry are written without an installations section") covers populated planets only, so it says nothing about such a record. Measured on elegy main `7c4f6b0` (#227):
- **skips.txt:** "planet field defenses null" 121 → 2 (cb7/sc035 and tk3/tk201, both population null); "planet field excess null" 4 → 0; "planet field planetary_scanner null" 2 → 0. baseline.txt does not move.
- **Corrected 2026-10-13:** this section first said every newly compared field matches. One does not. RG-001-2435 (`rg001.json`, the chained run) gains a clause in year 6, planet 41: "excess = 65, want 0". The case already failed there on population (2011, want 1912), so its baseline line does not move. The clause is downstream of that population divergence, not a counterexample to the ASSUMPTION (MEASURED). The y6 restart RG-001-Y6-2435 (`rg001-y6.json`) starts from the oracle's year-5 state for the planet (population 1665, carry 58). It compares the same expectation (population 1912 and the omitted block read as carry 0), and its reference seed matches it. The chained run reaches year 6 with a different population and keeps a carry of 65. The census and ledgers did not show the clause because RG-001's chained run has no ledger and was failing already.
- **Ledgers:** RG-002 loses its 6 "not compared … planet field defenses null" lines (14 → 8): y2 planet 65, y3 planet 66 and y4 planet 41, in the chained run and in the restarts. RG-003 loses 16 (50 → 34): the same six plus y7 planet 49, y8 planet 46 and y10 planet 20. No line is added and no hand line moves.

### Not-compared lines sorted by cause (main `1de8ab6`)

The RG ledgers' "not compared" lines (RG-002: 6, RG-003: 32, on `1de8ab6` before RG-004) were sorted by cause. This change fixes the comparator-side ones, except 0x0f4/0x0c2/0x0be, which lane D's #233 maps:
- **`battle_plan` expectations** are compared now (vectors/README.md "battle_plan": every field of the plan in the slot, `attack_who` numbered as the state's battle_plans). The loader now keeps a state plan's name. The census drops "expectation battle_plan" (15). RG-003 y9's plan 5 matches, in the chained run and in the restart.
- **0x039/0x03a** (one or more defenses built) map to EventBuilt of defenses, with the merge rule (MESSAGES.md 0x039, 0x03a, "Merging of build counts").
- **packet_destination/packet_warp null** is read as absent (vectors/README.md "initial_state": no destination, warp 4), but only when the planet still has a starbase. All 6 records so far are planets whose starbase was lost in the year. Elegy keeps the planet's settings there, and the restarts would miss. Whether the original cleared them or only does not write them for a planet with no starbase is UNRESOLVED, so those records stay skipped under a new reason.
  - Superseded by #248: Elegy clears the settings with a lost starbase (ELEGY CHOICE matching PK-1), and the 6 records compare and match.

Left as they were, with the reason:
- **0x0cd–0x0cf (starbase built).** MESSAGES.md picks the id by the new hull's dock (BINARY-ONLY): 0x0cf for an unlimited dock. Two vectors contradict that. SL-starbases (Space Station builds, design slot 2) and RG-003 y8 (planet 51, Space Station, `params [51 16]`) both record 0x0cd and no 0x0cf. No mapping is added until lanes F and G settle the rule. Mapping by the documented rule fails SL-starbases and SL-starbases-ctl.
  - Superseded by the 0x0cd PR: MESSAGES.md (stars-elegy 898c526) now gives 0x0cd for every starbase built, MEASURED for Orbital Fort, Space Dock, Space Station and Ultra Station (Death Star NOT RUN), with the design slot as owner·32 + 16 + slot. Elegy maps it that way, sending 0x0cd for a Death Star too (inferred).
- **Engine-side (lane D):**
  - 0x05b: the salvage and fleet slots are not compared, because EventFleetScrapped names neither the salvage object nor a fleet that still exists.
  - 0x08f/0x090: EventBombed names the bomber, not its fleet.
  - 0x0f4: lane D's #233 maps it. Its field owner, type and centre slots wait for lane D's follow-up.
- **Slot meaning open (lanes F and G):**
  - 0x004 and 0x007 each hold 49 in the slot MESSAGES.md calls "player" (RG-003 y7 `[51 550 0 49]`, y12 `[49 49 600 0]`). That is not a player number.
  - 0x007's colonists slot is the winner's landed troops, which no event carries.
