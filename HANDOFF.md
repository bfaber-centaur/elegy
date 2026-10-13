# Handoff

## Current question

Does Elegy reproduce a representative late-game run with every recorded order (RG-002), and what is left between it and the original?

## State

- elegy main `8426a6a` carries the seven RG-002 runs: the chained 2463→2468 and each year's restart. Every recorded order is loaded, and no year is left out.
- The discrepancy ledger (`engine/testdata/vectors/rg/rg002-ledger.txt`) is checked exactly by `TestRG002Ledger` and `TestBattleRecords`. A PR that changes RG-002 output rewrites it with `RG002_LEDGER=write` (both tests) and classifies any UNCLASSIFIED line.
- The result is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "RG-002", and the line-by-line breakdown is in [docs/RG002-REPORT.md](docs/RG002-REPORT.md).

## What we know

- **MEASURED on main `f6beefa` plus the messages comparator.**
  - On the reference seed (main `387c622`) there are 12 deterministic discrepancies, all missing-mechanic field lines: messages Elegy has no event for (below). #197 cleared Y4 planet 88's `packet_warp`. Incorrect mechanic, state-import error, unsupported order and unresolved randomness are 0 there.
  - Across the other seed variants there is one incorrect mechanic, an order refused only on variants 1 and 6 (below). It makes 2 ledger lines.
  - The restarts refuse no order, and no strict battle record fails.
  - 30 field lines are random-driven: each matches on some seed variant, not on the reference one.
  - Everything in the chained run after its first year is a sample. With the restarts' stream-marked random outcomes, samples are 1146 fields, 43 refused orders and 16 missed battle records.
- #189 (lane A's fleet retargets) and #190 (economy's remote mining) cleared all 78 earlier deterministic lines. RG002-REPORT.md lists them.
- Planet `packet_destination` and `packet_warp` and player `research_percent` are compared now (#194). A null `packet_destination` stays not compared (ELEGY CHOICE, RG002-REPORT.md).
- **Messages are compared now** (PARITY-STATUS.md "Messages compared"). RG-002's 68 message lines: 32 match, 15 are samples, and 21 were missing mechanic for ids Elegy sent no event for: economy 10 (0x026, 0x050, 0x05f), lane D 9 (battle summaries) and lane A 2 (0x02e). Lane D's battle summaries (#199) cleared its 9, leaving 12.
- **`object`-form wormholes are compared now.** All 72 are samples (a wormhole moves on draws every year), and all 72 miss on position, so they add 160 sample lines and no deterministic one.
- **Designs and player counts are compared now**, and all 20 RG-002 lines match.
- **Message order between ids is compared now** (PARITY-STATUS.md "Message order compared"): 1534 comparisons over all corpora and seed variants; the only mismatch is TK-203's draw-decided 0x050 (lane F, flagging).
- **Nothing Elegy models is left uncompared in RG-002.** The 184 not-compared lines are 176 battle expectations that `TestBattleRecords` compares record by record and 8 nulls in the original's file.

## What still matters

- Missing messages. Elegy has no event for overcrowding (0x026), a research level (0x050), a new component (0x05f), an unload of colonists (0x02e) Owners: economy and lane A. In other corpora, Elegy also sends no 0x04e after a refused gift at the last waypoint (WU-B1, WU-B2, WU-AICOMP3, WU-AICOMP4). Owner: lane A.
- Other corpora's unmapped message ids. The harness compares only the ids it maps (PARITY-STATUS.md "Messages compared"). The rest are skipped with a reason; the largest are 0x03f (no production queue, 298 expectations), 0x03e (queue finished, 41) and 0x0fe (28).
- Message sequences (MS-1) are compared now, with order, repeats and slots (PARITY-STATUS.md "Message sequences compared"). Some slots are not in Elegy's events: 0x050's next field, the 0x08f/0x090 bombing fleet, and 0x05b's salvage. 0x05f's part slots need a public category-mask table, which stars-elegy has only for terraform (lane F/G).
- T-37 (stars-elegy PARITY.md, CONFIRMED): after generation the original removes from a design any part the owner can no longer build (TK-104-A, TK-105-A: LBU-17 and LBU-32 at electronics 7). Elegy keeps them. The design comparison now shows it. Owner: lane A.
- Exact random parity needs the original's generator in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby). Until then the restarts' random-driven lines and the chained years stay unresolved.
- A variant-only order refusal. In RG-002 Y1, variants 1 and 6 refuse player 2's waypoint-0 change aimed at a fleet that player 6's `move_ships` empties earlier in the replay. stars-elegy #170 (a1c4c91, BINARY-ONLY) says the original never rejects a waypoint-0 follow whose target was merged away, so Elegy's refusal is the bug. Owner: lane A. It is 2 incorrect-mechanic ledger lines (PARITY-STATUS.md, point 2).
- stars-elegy #169 (ET-2c): a transport task aimed at another player's packet loads its minerals with no 0x11e (CONFIRMED by prediction). Elegy sends 0x11e and loads nothing. Owner: lane A.
- RG-003 (12 years) and RG-004 (a second game, 12 years) have their own ledgers (PARITY-STATUS.md "RG-003: 12 chained years", "RG-004: a second game, 12 chained years"). RG-003 has 1 deterministic lane A gap left (GC-1; #237 cleared the y12 battle); its y9 extra battle is a sample since #236. RG-004 has 16 deterministic lines: lane A's empty design-slot delete refused (9) and new-fleet numbering in y4 (5), and lane D's minefield `known_to` (2).

## Best next move

Map more message ids, starting with the most common in other corpora: 0x03f (no production queue) and 0x03e (queue finished). Each needs an Elegy event to map from, or it becomes a missing-mechanic finding for its owner. This is harness work in internal/vectors; rewrite the ledger and baseline and see what it exposes.
