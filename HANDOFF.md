# Handoff

## Current question

Does Elegy reproduce a representative late-game run with every recorded order (RG-002), and what is left between it and the original?

## State

- elegy main `e1798ac` carries the seven RG-002 runs: the chained 2463→2468 and each year's restart. Every recorded order is loaded, and no year is left out.
- The discrepancy ledger (`engine/testdata/vectors/rg/rg002-ledger.txt`) is checked exactly by `TestRG002Ledger` and `TestBattleRecords`. A PR that changes RG-002 output rewrites it with `RG002_LEDGER=write` (both tests) and classifies any UNCLASSIFIED line.
- The result is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "RG-002", and the line-by-line breakdown is in [docs/RG002-REPORT.md](docs/RG002-REPORT.md).

## What we know

- **MEASURED on `e1798ac`.**
  - There is no deterministic discrepancy left: missing mechanic, incorrect mechanic, state-import error, unsupported order and unresolved randomness are all 0.
  - The restarts refuse no order, and no strict battle record fails.
  - 30 field lines are random-driven: each matches on some seed variant, not on the reference one.
  - Everything in the chained run after its first year is a sample: 976 fields, 43 refused orders and 16 missed battle records.
- #189 (lane A's fleet retargets) and #190 (economy's remote mining) cleared all 78 earlier deterministic lines. RG002-REPORT.md lists them.
- **Not seen.** 272 not-compared lines are comparator gaps for quantities Elegy models. A mismatch there would go unseen:
  - packet destination and warp (110);
  - `object`-form wormholes (72);
  - messages (68);
  - design reads (14);
  - player counts and research percent (8).

## What still matters

- Those comparator gaps. Until they are closed, "no deterministic discrepancy" covers only what is compared.
- Exact random parity needs the original's generator in the harness (SPRINT.md P1-1, the seed-table publication pending Bobby). Until then the restarts' random-driven lines and the chained years stay unresolved.
- A variant-only order refusal. In RG-002 Y1, variants 1 and 6 refuse player 2's waypoint-0 change aimed at a fleet that player 6's `move_ships` empties earlier in the replay. ORDERS.md covers the merged-away target only for index 1 or later, so the original's behavior for waypoint 0 is UNRESOLVED. Owner: lane A. The ledger checks variant 0 only (PARITY-STATUS.md, point 2).
- stars-elegy #169 (ET-2): a transport loading from another player's packet is UNRESOLVED; the own-packet case is MEASURED. Owner: lane A.
- RG-002 has 6 years. Milestone 2 asks for 10.

## Best next move

Close the largest comparator gap: planet `packet_destination` and `packet_warp`. Elegy models `PacketDest` and `PacketSpeed`, so this is harness work in internal/vectors. Then rewrite the ledger and see what it exposes. Messages are next.
