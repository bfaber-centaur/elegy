# Parity status

Elegy's end-to-end results against the original, one entry per full-order regression. The behavioral record these rest on is stars-elegy `docs/PARITY.md`. Other elegy docs mean that file when they cite "docs/PARITY.md".

## RG-002: full-order six-year regression

Measured on elegy main `e1798ac` (#191 merged). Vectors: stars-elegy `vectors/rg/rg002*.json` (the chained 2463→2468 and each year's restart, Y1..Y6), synced unchanged. Details and the line-by-line breakdown are in [RG002-REPORT.md](RG002-REPORT.md). The ledger is [`engine/testdata/vectors/rg/rg002-ledger.txt`](../engine/testdata/vectors/rg/rg002-ledger.txt).

Bobby's five requirements, in order:

1. **All recorded orders loaded: MEASURED, yes.**
   - Lane F's coverage check (stars-elegy fe0ccc5) found that the 42 order files hold 1867 order records and the vectors carry 1867 orders.
   - The loader converts every one of them: the ledger has no `unsupported order` line, and no vector stops loading.
   - Two host-file fields are not in the vectors. Both are constant across RG-002, and their meaning is UNRESOLVED: the lay-mines task's second word, and the high nibble of the waypoint target type. The ledger header lists them.
2. **No silent rejections: MEASURED, none in the restarts.**
   - Every order Elegy refuses is a ledger line, so a refusal is never silent.
   - The six restarts refuse none. `TestRGNoRejectedOrders` fails on any refusal in a restart or in the chained run's first year.
   - The chained run refuses 43 orders in years 2–6. All 43 are samples: those years follow Elegy's own draws (RANDOM-PARITY.md "Samples").
3. **Exact deterministic field diffs: MEASURED, none.**
   - The reference run (seed variant 0) has no field, order or battle record line of a deterministic class.
   - `TestRG002Ledger` and `TestBattleRecords` fail on any new line and on any ledger line no longer produced.
   - Before #189 and #190 there were 78 such lines. RG002-REPORT.md "Effect of #189 and #190" lists them.
4. **Random results classified: yes.**
   - 30 field lines are random-driven. Each misses on the reference seed and matches on at least one other seed variant. The line keeps that count.
   - A line that misses on every variant is a hand line with an owner, never random-driven.
   - No restart battle record needed lane D's in-year random rule.
5. **Every test on the final main SHA: yes, on `e1798ac`** (below).

### Counts per class

| Class | Fields | Orders | Battle records |
|---|---|---|---|
| missing mechanic | 0 | 0 | 0 |
| incorrect mechanic | 0 | 0 | 0 |
| state-import error | 0 | 0 | 0 |
| unsupported order | 0 | 0 | 0 |
| unresolved randomness | 0 | 0 | 0 |
| random-driven | 30 | 0 | 0 |
| not compared | 456 | 0 | 0 |
| sample | 976 | 43 | 16 |

The 456 not-compared lines are:
- 272 comparator gaps for quantities Elegy models (packet destinations, `object`-form wormholes, messages, design reads, player counts);
- 176 battle expectations that `TestBattleRecords` compares record by record instead;
- 8 null in the original's file.

The 272 are where a mismatch could still go unseen (RG002-REPORT.md "Not compared, by reason").

### The run on `e1798ac`

- `go test ./... -count=1` passes in every package: ai, cmd/elegy, engine, game, internal/vectors, newgame, objects, oraclerng, races, regress and terraform.
- **The ledger.** `go test -run 'TestRG002Ledger|TestRGNoRejectedOrders' -v ./internal/vectors` passes, logging not compared 456, random-driven 30 and sample 1019 (fields and orders). `TestBattleRecords` passes. `RG002_LEDGER=write` and `PARITY_BASELINE=write` both leave the tree unchanged.
- **Battle records.** No strict RG-002 record fails: the restarts' 44 and the chained run's 8 in its first year. Of the chained run's 36 later records, all samples, 20 match.
- **The 8-seed matrix** (`go test -run TestParityVectors -v ./engine`, case level: a case passes on a seed when every compared expectation matches):

  | Case | Seeds that pass |
  |---|---|
  | RG-002-Y1-2463 | 1 of 8 (the reference seed among them) |
  | RG-002-Y2-2464 to Y5-2467 | 0 of 8 |
  | RG-002-Y6-2468 | 8 of 8 |
  | RG-002-2463 (chained, year 1) | 1 of 8 (the reference seed among them) |
  | RG-002-2464 to 2467 (chained) | 0 of 8 |
  | RG-002-2468 (chained) | 6 of 8 |

  - Every field in the Y2–Y5 restarts that fails on the reference seed matches on some other seed: they are the ledger's random-driven lines. A whole case can still fail on every seed, because each seed misses a different draw-decided field.
  - The chained years 2–6 are samples. RG-002-2468 passing on 6 seeds is a sample result, not replay parity.
