# Handoff

## Current question

Does Elegy reproduce a representative late-game run with every recorded order (RG-002), and what is left between it and the original?

## State

- elegy main `8426a6a` carries the seven RG-002 runs: the chained 2463→2468 and each year's restart. Every recorded order is loaded, and no year is left out.
- The discrepancy ledger (`engine/testdata/vectors/rg/rg002-ledger.txt`) is checked exactly by `TestRG002Ledger` and `TestBattleRecords`. A PR that changes RG-002 output rewrites it with `RG002_LEDGER=write` (both tests) and classifies any UNCLASSIFIED line.
- The result is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "RG-002", and the line-by-line breakdown is in [docs/RG002-REPORT.md](docs/RG002-REPORT.md).

## What we know

- **MEASURED on main `8426a6a` plus elegy #194 (docs, the variant-refusal test change and economy's packet-settings and research-percent comparator).**
  - On the reference seed there is one deterministic discrepancy, a field: Y4 planet 88's `packet_warp` is 0 in Elegy and 8 in the original (missing mechanic [economy], below). Incorrect mechanic, state-import error, unsupported order and unresolved randomness are 0 there.
  - Across the other seed variants there is one incorrect mechanic, an order refused only on variants 1 and 6 (below). It makes 2 ledger lines.
  - The restarts refuse no order, and no strict battle record fails.
  - 30 field lines are random-driven: each matches on some seed variant, not on the reference one.
  - Everything in the chained run after its first year is a sample: 977 fields, 43 refused orders and 16 missed battle records.
- #189 (lane A's fleet retargets) and #190 (economy's remote mining) cleared all 78 earlier deterministic lines. RG002-REPORT.md lists them.
- Planet `packet_destination` and `packet_warp` and player `research_percent` are compared now (#194). A null `packet_destination` stays not compared (ELEGY CHOICE, RG002-REPORT.md).
- **Not seen.** 160 not-compared lines are comparator gaps for quantities Elegy models. A mismatch there would go unseen:
  - `object`-form wormholes (72);
  - messages (68);
  - design reads (14);
  - player counts (6).

## What still matters

- A missing mechanic. Y4 planet 88's starbase with a mass driver completes, and the original's file then has launch warp 8 where Elegy has 0. Economy infers that completing such a starbase sets the packet warp, from this one observation. Owner: economy.
- Those comparator gaps. Until they are closed, "no deterministic discrepancy" covers only what is compared.
- Exact random parity needs the original's generator in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby). Until then the restarts' random-driven lines and the chained years stay unresolved.
- A variant-only order refusal. In RG-002 Y1, variants 1 and 6 refuse player 2's waypoint-0 change aimed at a fleet that player 6's `move_ships` empties earlier in the replay. stars-elegy #170 (a1c4c91, BINARY-ONLY) says the original never rejects a waypoint-0 follow whose target was merged away, so Elegy's refusal is the bug. Owner: lane A. It is 2 incorrect-mechanic ledger lines (PARITY-STATUS.md, point 2).
- stars-elegy #169 (ET-2c): a transport task aimed at another player's packet loads its minerals with no 0x11e (CONFIRMED by prediction). Elegy sends 0x11e and loads nothing. Owner: lane A.
- RG-002 has 6 years. Milestone 2 asks for 10.

## Best next move

Compare messages: the 68 `message` expectations, starting with economy's message-id comparison. Elegy's turn events have no event-to-message mapping in the harness yet, so this is harness work in internal/vectors. Then rewrite the ledger and see what it exposes.
