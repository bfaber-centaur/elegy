# Handoff

## Current question

Does Elegy reproduce a representative late-game run with every recorded order (RG-002), and what is left between it and the original?

## State

- elegy main `8426a6a` carries the seven RG-002 runs: the chained 2463→2468 and each year's restart. Every recorded order is loaded, and no year is left out.
- The discrepancy ledger (`engine/testdata/vectors/rg/rg002-ledger.txt`) is checked exactly by `TestRG002Ledger` and `TestBattleRecords`. A PR that changes RG-002 output rewrites it with `RG002_LEDGER=write` (both tests) and classifies any UNCLASSIFIED line.
- The result is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "RG-002", and the line-by-line breakdown is in [docs/RG002-REPORT.md](docs/RG002-REPORT.md).

## What we know

- **MEASURED on main `f6beefa` plus the messages comparator.**
  - On the reference seed there are 21 deterministic discrepancies, all missing-mechanic field lines: messages Elegy has no event for (below). #197 cleared Y4 planet 88's `packet_warp`. Incorrect mechanic, state-import error, unsupported order and unresolved randomness are 0 there.
  - Across the other seed variants there is one incorrect mechanic, an order refused only on variants 1 and 6 (below). It makes 2 ledger lines.
  - The restarts refuse no order, and no strict battle record fails.
  - 30 field lines are random-driven: each matches on some seed variant, not on the reference one.
  - Everything in the chained run after its first year is a sample. With the restarts' stream-marked random outcomes, samples are 991 fields, 43 refused orders and 16 missed battle records.
- #189 (lane A's fleet retargets) and #190 (economy's remote mining) cleared all 78 earlier deterministic lines. RG002-REPORT.md lists them.
- Planet `packet_destination` and `packet_warp` and player `research_percent` are compared now (#194). A null `packet_destination` stays not compared (ELEGY CHOICE, RG002-REPORT.md).
- **Messages are compared now** (PARITY-STATUS.md "Messages compared"). RG-002's 68 message lines: 32 match, 15 are samples, and 21 are missing mechanic for ids Elegy sends no event for: economy 10 (0x026, 0x050, 0x05f), lane D 9 (battle summaries) and lane A 2 (0x02e).
- **Not seen.** Of the 276 not-compared lines, 176 are battle expectations that `TestBattleRecords` compares record by record and 8 are null in the original's file. The other 92 are comparator gaps for quantities Elegy models, where a mismatch would go unseen:
  - `object`-form wormholes (72);
  - design reads (14);
  - player counts (6).

## What still matters

- Missing messages. Elegy has no event for overcrowding (0x026), a research level (0x050), a new component (0x05f), an unload of colonists (0x02e) or a per-player battle summary (0x095, 0x09f, 0x0a1, 0x116). Owners: economy, lane A and lane D. In other corpora, Elegy also sends no 0x04e after a refused gift at the last waypoint (WU-B1, WU-B2, WU-AICOMP3, WU-AICOMP4). Owner: lane A.
- Those comparator gaps. Until they are closed, "no deterministic discrepancy" covers only what is compared.
- Exact random parity needs the original's generator in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby). Until then the restarts' random-driven lines and the chained years stay unresolved.
- A variant-only order refusal. In RG-002 Y1, variants 1 and 6 refuse player 2's waypoint-0 change aimed at a fleet that player 6's `move_ships` empties earlier in the replay. stars-elegy #170 (a1c4c91, BINARY-ONLY) says the original never rejects a waypoint-0 follow whose target was merged away, so Elegy's refusal is the bug. Owner: lane A. It is 2 incorrect-mechanic ledger lines (PARITY-STATUS.md, point 2).
- stars-elegy #169 (ET-2c): a transport task aimed at another player's packet loads its minerals with no 0x11e (CONFIRMED by prediction). Elegy sends 0x11e and loads nothing. Owner: lane A.
- RG-002 has 6 years. Milestone 2 asks for 10.

## Best next move

Compare the 72 `object`-form wormhole expectations. Only the `wormhole` expectation kind is compared now, so this is harness work in internal/vectors. Then rewrite the ledger and see what it exposes.
