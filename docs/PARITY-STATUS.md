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
- **Other corpora:** the parity baseline is unchanged. TK-104-A and TK-105-A were skipped before ("design read needs the order layer's design source") and now fail on T-37 (lane A), so tk2 goes from fail 0 / skip 4 to fail 2 / skip 2. Stars-elegy PARITY.md T-37 (CONFIRMED) says the original removes LBU-17 and LBU-32 from designs after generation, at electronics 7. Elegy keeps them (lane A).

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
- **Not mapped yet: 0x078, 0x145, 0x157** (a level made a ship hull, planetary defense or planetary scanner available). With them mapped, every presence check matches, but the order check fails in KX-003 r2, KX-005 T0–T2 and RG-001 Y4: the vectors list a 0x05f of the same level before the hull or planetary message (energy 5: Temp Terraform ±7 before Missile Battery; construction 2: Robo-Mini-Miner before Mini-Miner). Elegy announces a level's components in table order. The original's order within one level is not in MESSAGES.md, so these ids wait for that rule.

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
  - 0x02e's mineral and location slots, because MESSAGES.md does not say which word of a planet location holds the planet;
  - a battle summary's focus and slots;
  - 0x04e's and 0x02e's fleet when the fleet is gone by the end of the year.
- **Result.**
  - Of the 249 sequences in the reference streams, 183 are compared and 66 are skipped by named reason (engine/testdata/vectors/skips.txt).
  - The parity baseline does not move. The new mismatches are all in RG cases that already fail.
- **RG-002 ledger:**
  - Not compared 268 → 190: 84 lines become 6, the 0x05b, 0x05f and 0x08f slots, each in the chained run and its restart. Y1's 0x02e (#213) matches with its fleet and amount.
  - Random-driven 30 → 31: Y3's 0x036 count, 11 against Elegy's 10 (5 of 8 variants match).
  - Sample 1191 → 1197: the chained run's 0x036/0x038 counts in years 3, 4 and 6, which follow the factory and mine counts that already differ.

