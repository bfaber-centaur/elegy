# Handoff

## Current question

Is main `f09acd9` a parity release candidate? That means knowing what is compared and passes, what is random-dependent, sampled, absent or untested, and what stands between Elegy and a complete engine.

## State

- The release candidate is main `f09acd9`'s code, pinned as #260's merge commit (docs only on top). It is the base for lane F's out-of-sample game, RG-005. The RG vectors are stars-elegy main `104e8f5`'s, including #206's rebuilt year 6.
- The coverage report is [docs/COVERAGE.md](docs/COVERAGE.md). The per-corpus detail is in [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md) "Current status".
- Tested at #260's head `b78c03c` (tree `59a494d`), a merge of main `f09acd9` with this PR's docs-only commits. Outside docs/ and HANDOFF.md its files are identical to `f09acd9`'s (`git diff f09acd9 b78c03c -- . ':!docs' ':!HANDOFF.md'` is empty), so the code tested is main `f09acd9`'s (tree `b7e0e77`). No test reads the files this PR changes. The candidate is #260's merge commit, whose tree differs from `f09acd9`'s only in docs/ and HANDOFF.md. On that head:
  - `go test ./...` passes.
  - The PR-subset long-game matrix passes: 24 games of 60 years, with replay and save/reload.
  - The nightly matrix: 96 of 96 games with no failure (12 seeds × 2 rulesets × 4 opponent sets, 150 years on a medium map, replayed and reloaded after 77 years; 444 s).
  - The RG ledgers and `TestBattleRecords` match the run exactly, and regenerating them moves no line.
- Each ledgered game registers from its files (`engine/testdata/vectors/rg/GAMES`, internal/vectors `RGGames`). A PR that moves RG output regenerates the ledgers rather than editing them by hand:
  - `RG002_LEDGER=write go test ./internal/vectors -run 'TestRG00.Ledger'`
  - `RG002_LEDGER=write go test ./engine -run 'TestBattleRecords$'`
  - `PARITY_BASELINE=write` for the baseline.

## What we know

- **0 known discrepancies in RG-002, RG-003 and RG-004** on the reference seed: no incorrect-mechanic line and no unresolved-randomness line.
  - Every compared difference is a sample, or random-driven: it matches on at least one of the 8 seed variants.
  - The chained runs diverge in year 1, and everything after year 1 is a sample.
- **Ledger lines** (chained / restarts):

  | corpus | matched | random-driven | sample | not compared | total |
  |---|---|---|---|---|---:|
  | RG-002 | 8 / 44 | 0 / 35 | 1019 / 199 | 0 / 0 | 1305 |
  | RG-003 | 8 / 100 | 0 / 76 | 5287 / 450 | 0 / 0 | 5921 |
  | RG-004 | 4 / 47 | 1 / 17 | 2519 / 263 | 2 / 2 | 2855 |

- **Orders:** every recorded order is loaded and accounted for, and no vector has a load error.
  - The chained runs refuse 37, 220 and 84 orders, all samples.
  - No restart refuses an order.
- **Random-driven lines with a named draw.** RG-002 and RG-003 each have the same 3, in the Y6 restart:
  - one is lane D's movement tie;
  - two are economy's germanium mining +1 draw.
- **Not compared:** RG-004's `design_gone` (lane B) and a sequence of sample entries only. `alt_battle` is judged by `TestBattleRecords`.
- **Zero known discrepancies is not complete equivalence.** COVERAGE.md "Biggest blind spots" covers:
  - random parity, which is 0 exact;
  - computer players, with no oracle comparison;
  - client estimates, 216 expectations;
  - 61 unmapped message ids;
  - fleet-to-fleet cargo.

## Engine completion

The full checklist, by subsystem and column, is docs/SPRINT.md "Engine-completion checklist".

| column | state | what is missing |
|---|---|---|
| playable | done for the vertical slice | Every step works from the CLI with readable output (`elegy show`, `order`, `check`, #266), including design and battle-plan authoring by name (#268, `TestOrderDesigns`, `TestOrderPlans`), and a saved game reloads identically. Deferred: derived numbers and a front end. |
| behaviorally faithful | partial | 0 known discrepancies inside the compared set. Not compared: AI decisions, client estimates, 61 message ids and fleet-to-fleet cargo. |
| bit-exact RNG parity | missing | The original's generator is not in the harness (SPRINT.md P1-1, pending Bobby). |

- **RG-005 (out of sample), 2026-10-14:** the zero did not hold. On the candidate's first run, 25 lines were UNCLASSIFIED (8 in restarts years 1–6 and chained year 1). Fix A (harness: Trader vector ids) leaves 19, all in the restarts: 12 strict expectations that miss on all 8 seeds, hand-classed `state-import error [lane B]` with cause UNRESOLVED and still counted as deterministic misses. The leading candidate is that the vectors do not carry the computer players' level (lane D retrodiction). PARITY-STATUS.md "RG-005" has the tallies and the PREDICTION.md scoring.

## What still matters

- COVERAGE.md "Remaining comparator gaps, ranked" lists what to route next. The top three:
  - minefield messages (35, lane D);
  - fleet-to-fleet transport (38, lane A, a mechanic);
  - the Mystery Trader (15, lane D).
- Viewer reports (156) are compared since 2026-10-14 and all pass.
- Random-driven lines stay open until the original's generator is in the harness.
- The parity case tally lists most RG cases as `fail`, although there are 0 known discrepancies.
  - Restarts fail because no single seed matches all their random-driven expectations at once.
  - Chained years after the first continue from a diverged year 1.
  - PARITY-STATUS.md "Current status" has the per-seed check behind this.

## Best next move

Load the computer players' level once lane F's vectors carry it (`computer_level`), with an explicit skip where a computer player has none, and regenerate RG-005 as a fourth tally. Then route the ranked gaps, starting with the minefield messages.
