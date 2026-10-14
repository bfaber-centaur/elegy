# Handoff

## Current question

Is Elegy ready for a human playtest? The release candidate is meant to be stable and reproducible, and playable from the command line. The next evidence comes from people playing it ([docs/PLAYTEST.md](docs/PLAYTEST.md)), not from another research round.

## State

Release-candidate checkpoint, 2026-10-14.

- **Engine:** elegy main after #279. The tested code is `b2642b4` (tree `533a530`), which merges main `2943788` (#278) with #279's command-line changes. #279's last commit adds only this file, so the code tested is exactly the code merged.
- **Integrated on main, in order:**
  - #276 (`97fb5d2`): RG-005 ingest, the Trader-id harness fix and computer levels from the vectors.
  - #277 (`d64c936`): the patrol target message 0x0ff.
  - #278 (`2943788`): the Mystery Trader part messages and player Trader items.
- **Specification and vectors:** stars-elegy main `d3d436b`. The RG-005 vectors are `419229a` (#212); the computer levels come from `bdb4921` (#214).
- **Other repositories** (private, not used by elegy's build): stars-oracle-apparatus `1a9f1c2`, stars-decomp `a1af617`.
- **Ruleset** `elegy` v7 (default) and `jrc3-faithful` v6. SaveVersion 13. Event kinds up to 103.

### Verification on `b2642b4`

| check | command | result |
|---|---|---|
| full suite | `go test ./... -count=1` | pass, every package (2 min 37 s on 4 cores) |
| computer players across save and reload | `go test ./ai -run TestDriversSaveReload -count=1` | pass, both rulesets |
| release matrix | `go test ./regress -run '^TestLongGames$' -count=1 -args -seeds=1,2,3,4,5,6,7,8 -years=150 -save-at=77 -size=2` | 64 games, 64 with no failure (4 min 42 s) |
| CLI walkthrough, save copy continued | `TestPlaytestScript`, `TestCLIPlayableSlice` (in the full suite) | pass |

The release matrix covers 8 seeds × 2 rulesets × 4 opponent sets: 64 games of 150 years on a medium map. Each game is replayed from its seed and reloaded from its year-77 save, and every year's hash must match ([docs/REGRESSION.md](docs/REGRESSION.md)). The steward also ran this matrix on #278's trial merge, with 64 of 64 games and no failure, and `TestDriversSaveReload` passed there.

## What we know

**RG status** (ledgers `engine/testdata/vectors/rg/`; detail in [docs/COVERAGE.md](docs/COVERAGE.md) and [docs/PARITY-STATUS.md](docs/PARITY-STATUS.md)):

| game | what it is | status on main |
|---|---|---|
| RG-001 | 7 players, 2429–2435 | Held strict, no ledger. Restarts Y2 and Y4 pass. Y1, Y3, Y5, Y6 and the chained run pass on some seed variants (random). No order is refused on any variant (`TestRGNoRejectedOrders`). |
| RG-002 | AIX start, 2461 | 0 deterministic misses. Ledger: 52 matched, 36 random-driven, 1218 sample. |
| RG-003 | 12-year chain, 2462–2474 | 0 deterministic misses. Ledger: 108 matched, 77 random-driven, 5741 sample. |
| RG-004 | 7 custom races, 2466–2477 | 0 deterministic misses. Ledger: 51 matched, 18 random-driven, 2802 sample, 4 not compared (`design_gone` and a samples-only sequence). |
| RG-005 | out-of-sample game, 5 custom races, 2466–2477 | **First result, kept as recorded:** on the frozen candidate `8584f92` (record commit `9bb2cb6`), the pre-registered prediction "RC does not hold out of sample" was CONFIRMED. 25 lines were UNCLASSIFIED, 8 of them early, and they missed on all 8 seeds. **Corrected afterwards (harness, not engine):** fix A (Trader vector ids) and loading each computer player's level from the vectors leave 0 deterministic misses. Ledger now: 21 matched, 11 random-driven, 2363 sample, 15 not compared (Trader slots Elegy cannot name). RG-005 is no longer out of sample. |

"0 deterministic misses" means every compared difference is either a sample or matches on at least one of the 8 seed variants. That is weaker than equivalence: see the blind spots below. The parity tally lists most RG cases as `fail` or `random`, because no single seed matches all of a case's random outcomes at once.

**Coverage figures on main `2943788`** (from the coordinator's census; COVERAGE.md holds the recount):
- all corpora: pass 1162, skip 383;
- class (e), untested: 336 expectations;
- messages with no event mapping: 84 expectations over 52 ids;
- unproven labels: BINARY-ONLY 192, ASSUMPTION 84, UNRESOLVED 16, NOT RUN 22.

**Known mechanical discrepancies and untested behaviour:**
- **Randomness.** Elegy uses its own generator. Random outcomes are never replayed bit for bit: exact random parity is 0 in every corpus, and random outcomes are compared as samples or across 8 seeds. Bit-exact replay of the original's generator is a separate milestone (SPRINT.md P1-1), not a blocker for playing.
- **Computer players.** No decision of theirs is compared with the original (AI-STATUS.md). The long-game matrix shows only that they run deterministically and survive save and reload.
- **Client estimates.** 216 estimates are not compared (milestone 3). The CLI shows resources, maximum population and operable installations. It does not show range, ETA or cost.
- **Messages.** 52 message ids are not mapped. The CLI prints Elegy's event names, not the original's texts.
- **Not modelled.** Robber Baron and Pick Pocket cargo stealing, and the fuel action "load optimal" (TAKEOVER-STATUS.md).
- **Mystery Trader slots.** Elegy keeps Traders densely, without the original's slot, so 15 RG-005 Trader expectations are not compared.
- **INTENTIONALLY DIFFERENT under `elegy`:** merge overflow (FO-03-E, FO-06-G) and `zero_max_population_stop` off (RULESET.md). `jrc3-faithful` keeps the original's behaviour.

**Playable** (docs/PLAYTEST.md): creating a game, inspecting it, research, production, designing and queueing a ship, moving, colonizing, meeting and fighting the computer player, advancing years, and saving and continuing all work from `cmd/elegy`. `TestPlaytestScript` runs the guide's literal walkthrough. The command line has no `order` kind for merge, split, rename, transfers between fleets, follow, repeat, planet settings, relations, the default queue or detonation. Those orders can be written into the JSON order file by hand.

## Rebuild and run

```sh
git clone https://github.com/bfaber-centaur/elegy && cd elegy
git checkout b2642b4        # or main
go test ./...              # about 3 minutes on 4 cores
go build -o elegy ./cmd/elegy
mkdir -p playtest && cd playtest
../elegy new -seed 1 -players 1 -size tiny -ai robotoid
../elegy show
```

Then follow [docs/PLAYTEST.md](docs/PLAYTEST.md). It covers the walkthrough, the command list, rejected orders, keeping a failing game, the bug-report format and the regression commands. Go version: `go.mod`.

## What still matters

- The human playtest, starting with what PLAYTEST.md "What a human should test first" lists. Use the bug-report format.
- Playtest bugs are judged against the behavioural record (stars-elegy PARITY.md) before anyone changes a rule.
- Deferred, not blockers:
  - the missing `order` kinds above;
  - fleet range and cost estimates in `show`;
  - the original's message texts and planet names;
  - any TUI or GUI;
  - bit-exact random replay;
  - the ranked comparator gaps (COVERAGE.md "Remaining comparator gaps, ranked").

## Best next move

Play. Start with PLAYTEST.md's walkthrough, then a 30-year game on `-size small`, and file what breaks in the bug-report format. No new investigation is open.
