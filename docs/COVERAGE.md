# Parity coverage (release candidate)

What Elegy's comparison with the original covers, what it leaves out, and how much of each. Measured on elegy main `f09acd9`, with stars-elegy vectors `104e8f5`, after every RG ledger was regenerated with `RG002_LEDGER=write` (no line moved). [PARITY-STATUS.md](PARITY-STATUS.md) holds the per-corpus detail and the history behind these numbers.

Updated 2026-10-14 (viewer reports PR): the sc corpus's 156 viewer-report expectations (views of `other_players` 96, `player` 56, `design` 4) are now compared, and all pass. All-corpora pass rose from 981 to 1137 and skip fell from 565 to 409. Class (e) fell from 584 (after the 0x12c/0x12d mapping, below) to 428, and the gap ranking lost its old rank 1, "viewer reports: `other_players`, `player`, `design` views | 156 | lane B (harness)". The RG numbers are unchanged from `f09acd9`. One reading is an ASSUMPTION: `starbase_designs_full` lists the subject's own starbase designs, while the original's file names no owner for a partial starbase design and the sc experiment's reader counts every such design for the subject. The sc cases have two players, so they cannot tell the two readings apart (pinned by `internal/vectors` `TestDesignViews`).

Updated 2026-10-14 (fleet heading and mass PR): the sc corpus's 7 fleet `heading` and 3 `mass_shown` expectations are now compared, and all pass. All-corpora pass rose from 1137 to 1147 and skip fell from 409 to 399. Class (e) fell from 428 to 418, the "other fields" row from 33 to 23, and the gap ranking lost its old rank 5, "fleet heading and `mass_shown` | 10 | lane B (harness)".

Tested at #260's head `b78c03c` (tree `59a494d`), a merge of main `f09acd9` with this PR's docs-only commits. Outside docs/ and HANDOFF.md its files are identical to `f09acd9`'s (`git diff f09acd9 b78c03c -- . ':!docs' ':!HANDOFF.md'` is empty), so the code tested is main `f09acd9`'s (tree `b7e0e77`). No test reads the files this PR changes. The candidate is #260's merge commit, whose tree differs from `f09acd9`'s only in docs/ and HANDOFF.md. On that head: `go test ./...` passes; the PR-subset matrix (24 games of 60 years, replayed and reloaded after 23) has 24 with no failure; the nightly matrix has 96 of 96 games with no failure (12 seeds × 2 rulesets × 4 opponent sets, 150 years on a medium map, replayed and reloaded after 77 years; 444 s).

**Zero known discrepancies is not complete equivalence.** No compared value in RG-002, RG-003 or RG-004 differs from the original in a way no seed variant reproduces, and no targeted corpus case fails. That claim covers only what is compared:
- most random outcomes are compared as samples or across 8 seed variants, never replayed bit for bit;
- 7,324 skipped expectations are samples of another random stream;
- 418 more expectations test mechanics or views the harness does not compare yet (class e);
- whole subsystems, such as the computer players' choices and the client's estimates, have no oracle comparison at all.

The biggest blind spots are listed at the end.

## How to read the classes

- **(a) Strict pass.** A strict expectation (the vector flags it as one stream-independent outcome) matches on the reference seed.
- **(b) Random-dependent.** It misses on the reference seed but matches on at least one of the 8 seed variants (random-driven), or a whole case passes on some seeds only (`random k`).
- **(c) Sample.** The vector flags it as one random stream's outcome, or it is a chained year after the first, which continues from a state that already differs on the draws. It is compared and tallied, never counted as parity.
- **(d) Absent source value.** The original's file holds no value to compare: a null record, a player outside the state, or an input the vector does not carry (a computer player's planned orders).
- **(e) Untested.** The value exists, but Elegy does not model it or the harness does not map it: messages with no event mapping, uncompared slots, client estimates, fleet-to-fleet transport.

## RG corpora (representative games)

Field expectations on the reference seed; chained run / 12 (RG-002: 6) restarts. Battle records are counted separately below.

| class | RG-002 | RG-003 | RG-004 |
|---|---|---|---|
| (a) strict pass | 356 / 2227 | 356 / 4833 | 197 / 2517 |
| strict miss, no seed matches | 0 / 0 | 0 / 0 | 0 / 0 |
| (b) random-driven | 0 / 25 | 0 / 59 | 1 / 12 |
| (c) sample, matched | 1663 / 185 | 3112 / 452 | 1732 / 139 |
| (c) sample, missed | 562 / 144 | 2196 / 320 | 898 / 160 |
| not compared | 0 / 0 | 1 / 1 | 3 / 3 |
| expectations per run | 2581 | 5665 | 2831 |

- **Ledger lines** (one per difference or battle record): RG-002 1305, RG-003 5921, RG-004 2855. Not-compared lines: 0 / 0, 0 / 0 and 2 / 2.
  - None is an incorrect mechanic or unresolved randomness.
  - Random-driven lines: RG-002 35, RG-003 76, RG-004 18.
  - Six of them carry a named draw. Lane D's is a movement tie; economy's is the germanium mining +1 draw. Each of RG-002 and RG-003 has 1 of lane D's and 2 of economy's.
- **Orders:** every recorded order is loaded and accounted for: RG-002 1867, RG-003 3731, RG-004 2049.
  - The chained runs refuse 37, 220 and 84 orders, all samples.
  - No restart refuses an order, and no vector has a load error.
- **Battle records** (`TestBattleRecords`, rg corpus): 418 records.
  - 220 were generated and replayed: 146 of 146 hits matched, 2432 of 2432 conditioned hits matched, and 3957 of 3957 recorded moves checked.
  - 1569 moves are unchecked. The battle samples match 75 of 195, and 3 records are random.
  - Ledger classes: RG-002 52 matched, 41 sample and 1 random-driven; RG-003 108, 108 and 3; RG-004 51, 63 and 2.
- **Case tally** (`TestParityVectors`): 0 pass, 2 measured pass, 48 fail, 13 random and 9 random with the reference seed failing.
  - Every failing case fails on samples or random-driven expectations (PARITY-STATUS.md "Current status", per-seed check).
- **Not compared here:**
  - `design_gone` (RG-004, 1 per run): lane B, untested (e);
  - a sequence of sample entries only (RG-004, 1 per run), which has nothing strict to compare;
  - `alt_battle` (RG-003 and RG-004, 1 per run), which the battle harness judges instead (a battle-ledger sample), so it has no vectors-ledger line.

## All corpora

`TestParityVectors` over 31 corpora, 1827 cases:

| status | cases | class |
|---|---:|---|
| pass | 1147 | (a) |
| measured pass | 178 | (a), with a MEASURED tag |
| random, passes on some seeds | 16 | (b) |
| random, reference seed fails | 12 | (b) |
| sample-only | 25 | (c) |
| fail | 48 | all in RG chained years and restarts: (b) and (c), as above |
| differs | 2 | INTENTIONALLY DIFFERENT under the `elegy` ruleset (FO-03-E, FO-06-G, `Legacy.MergeOverflow` off) |
| skip | 399 | every expectation of the case is skipped, by the reasons below |

Expectations skipped, by class (engine/testdata/vectors/skips.txt):

| class | expectations | reasons |
|---|---:|---|
| (c) another stream, or a stream's random outcome | 7,324 | "stream" 5129, "sample: one stream's random outcome" 2001, "expectation sample" 192, sample-only sequences 2 |
| judged by `TestBattleRecords`, not here | 994 | battle 497, battle_actions 484, no_battle 9, alt_battle 4 |
| (d) absent source value | 334 | player not in the state 291; computer players' planned orders 24, and 16 score records that depend on them; null planet records 3 |
| (e) client estimates (milestone 3) | 216 | 21 reasons: ETA, travel time, mining rate, completion, research dialog, ship designer, population and defense estimates |
| (e) messages with no event mapping | 148 | 63 ids (145 expectations) and 2 ids' sequence entries (3) |
| (e) fleet-to-fleet transport | 38 | the task 15, its target 15, transport fuel 8 |
| (e) other fields and expectations | 23 | `design_gone` 5, `starbase_design_gone` 1, `no_new_fleets` 1; planet `environment_visible` 2, `starbase_design` 1; player Mystery Trader items 5, relations 1, `ship_design_count` 1; Super Stealth research stealing 6 |
| (e) message slots not compared | 1 | 0x182 player 1 |

Updated 2026-10-14 (economy, 0x12c/0x12d mapping): "messages with no event mapping" was "148 | 63 ids (145 expectations) and 2 ids' sequence entries (3)" on `f09acd9`; 0x12c (2 + 1 sequence entry) and 0x12d (3 + 2) are now compared and match, so it is 140 (61 ids, no sequence entries) and class (e) is 584. The other numbers on this page are still `f09acd9`'s.

The most frequent unmapped message ids are:
- 0x0c6 and 0x0ca (fleet stopped and damaged in a minefield, both sides): 11 each;
- 0x10b (Mystery Trader trade): 10;
- 0x0ff (patrol picks a target): 9;
- 0x0c3 and 0x0c4 (mines laid, new or added): 5 each;
- 0x17e (mines not laid, object limit): 3.

## Unproven claims in the status documents

Lines in elegy's `docs/*-STATUS.md` and RANDOM-PARITY.md that carry a label weaker than CONFIRMED or MEASURED (a count of labelled lines, `grep -c`, not of distinct rules):

| label | lines | heaviest in |
|---|---:|---|
| BINARY-ONLY (from analysis, no oracle run) | 188 | OBJECTS 44, ORDERS-LAYER 24, KERNEL 20, AI 17, COMBAT 17, TAKEOVER 18 |
| ASSUMPTION | 80 | KERNEL 33, PARITY 12, TAKEOVER 12, COMBAT 7 |
| NOT RUN | 22 | COMBAT 9, PARITY 5, TAKEOVER 4 |
| UNRESOLVED | 12 | PARITY 4, TAKEOVER 3 |

## Biggest blind spots

1. **Random parity.** Exact parity is 0 in every corpus (RANDOM-PARITY.md).
   - Random outcomes are compared as samples or across 8 seeds, never replayed.
   - The 7,324 other-stream expectations and the chained years after year 1 test only plausibility.
   - This needs the original's generator in the harness (SPRINT.md P1-1, pending Bobby).
2. **Computer players.** No AI decision is compared with the oracle (AI-STATUS.md "Implemented"; 17 BINARY-ONLY lines). Their planned orders are absent from the vectors (24 expectations, plus 16 score records).
3. **Client estimates.** 216 estimates are not compared (milestone 3).
4. **Message coverage.** 63 ids have no event mapping, and minefield messages are the largest group.
5. **Fleet-to-fleet cargo.** Loads between fleets, from another player's fleet, Robber Baron stealing and transport fuel actions are not modelled (TAKEOVER-STATUS.md "Not modelled"; 38 expectations).
6. **Labels.** 188 BINARY-ONLY and 80 ASSUMPTION lines are claims no oracle run has checked.
7. **Breadth.** RG covers two representative games (RG-002/003 share years 1–6). An out-of-sample game (RG-005, lane F) will test whether the zero holds.

## Remaining comparator gaps, ranked

Ranked by expectations unblocked, then by whether a mapping exists to build on. Owners are proposals for the coordinator.

| rank | gap | expectations | proposed owner |
|---:|---|---:|---|
| 1 | minefield messages 0x0c3, 0x0c4, 0x0c6, 0x0ca, 0x17e | 35 | lane D |
| 2 | fleet-to-fleet transport and transport fuel | 38 | lane A (mechanic, not only the comparator) |
| 3 | Mystery Trader trade 0x10b and player Trader items | 15 | lane D (objects) |
| 4 | patrol target 0x0ff | 9 | lane A |
| 5 | planet value messages 0x12c, 0x12d (and their sequence entries); Updated 2026-10-14: mapped, all 8 match (rows BINARY-ONLY) | 8 | economy |
| 6 | `design_gone`, `starbase_design_gone`, `no_new_fleets` | 7 | lane B (harness) |
| 7 | 0x182 player slot | 1 | economy |

Viewer reports (156: `other_players`, `player` and `design` views) are compared since 2026-10-14 and all pass (sc corpus, below). Client estimates (216) stay deferred to milestone 3 (ELEGY CHOICE, PARITY-STATUS.md "Deferred: client estimators").
