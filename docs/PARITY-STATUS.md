# Parity status

Elegy's end-to-end results against the original, one entry per full-order regression. The behavioral record these rest on is stars-elegy `docs/PARITY.md`. Other elegy docs mean that file when they cite "docs/PARITY.md".

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
  - **lane A, 2 lines:** 0x02e, colonists beamed down by an unload task (Y1 and the chained year 1).

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
- **0x04e after a refused gift: WU-AICOMP3, WU-AICOMP4, WU-B1, WU-B2 (lane A).** A transfer task to another player at the fleet's last waypoint is refused (Elegy's `EventFleetGiftRefused`), and the original then sends 0x04e: a refusal cancels the task, and 0x04e follows at the last waypoint (MESSAGES.md 0x04e (a)). Elegy sends no `EventFleetArrived` after the refusal.

### Battle summaries (lane D)

Elegy now sends each battle player its summary (`EventBattleSummary`, engine/combat_summary.go), and `pvMessages` maps it to MESSAGES.md's 0x07e, 0x091–0x0a8 and 0x113–0x116. All of those ids are compared now.
- **RG-002:** lane D's 9 missing-mechanic lines match on the reference seed (0x095 ×4, 0x116 ×3, 0x09f, 0x0a1), and so do the 5 summary samples of the chained run's years 2–6. No new line appears. The ledger is missing mechanic 13 (economy 10, lane A 2, Y4 planet 88's `packet_warp`), incorrect mechanic 2 (orders), random-driven 30, not compared 276, and samples 987 fields / 43 orders / 16 battle records.
- **Other corpora:** every summary expectation outside RG-002 matches: CB-048 and CB-048-ctl (0x098, 0x099), CB-049 (0x09b, 0x09c) and SC-036 (0x0a4, 0x0a7). The parity baseline is unchanged. Every expectation of these ids in the corpora is "present"; none checks that a wrong id is absent, so `TestBattleSummaryRules` pins the choice among the ids.
- **What a "stack" is.** MESSAGES.md's one-stack texts (0x096–0x09a) say "exactly 2 stacks". CB-039 (140 one-ship fleets a side) sent 0x09a, and CB-048 (80 ships in one fleet against 30 one-ship fleets) sent 0x098 and 0x099. So a stack there is one player's ships of one design, whatever fleets hold them (MEASURED). Elegy counts it that way; read as fleet stacks, CB-048 fails.

### Object-form wormholes compared (after the final run)

Measured on elegy main `9088500` (#198) plus the wormhole branch; counts re-measured on main `387c622` (#199, lane D's battle summaries) plus the wormhole branch.

`object` expectations of kind `wormhole` are compared now (internal/vectors `wormholeFields`), with the fields the `wormhole` form already used plus `id` and `partner`, the other end's vector id.
- **RG-002: 72 expectations, all marked sample by the vectors.** A wormhole end jiggles or jumps on draws every year (OBJECTS.md). All 72 miss, and they become 160 sample field lines: x (72), y (65), `years_since_jump` (13) and `known_to` (10). `partner`, `stability_class` and `destination_known_to` match in all 72. No deterministic line is added.
- Counts after this, on main `387c622`: missing mechanic 12 (economy 10, lane A 2), incorrect mechanic 2 (orders), random-driven 30, not compared 204 (20 comparator gaps: 14 design reads and 6 player counts; 176 battle; 8 null), and samples 1146 fields (986 + 160), 43 orders and 16 battle records.
- **Other corpora: RD-P20 and RD-P21.** Their 10 `object` wormhole expectations are year-end positions after a jiggle, which is draw-decided. Before stars-elegy #181 the vectors did not mark them as samples, so they missed on position (and RD-P21 on one `known_to`) and left the parity baseline. stars-elegy #181 (`b4c0086`) flags them `sample` (6 in rp20, 4 in rp21); with those two vectors re-synced, both pass and are back in the baseline. The re-synced files also carry the galaxy `size`, `bounds` and `density` fields; Elegy's vectors loader does not read them yet, so they change no result.

