# Development direction

## Goal

Build a deterministic simulation engine, then grow it into a complete 4X game.

A J-RC3-compatible ruleset is a major target, but compatibility behavior should be based on measured/documented evidence rather than intuition. Explicitly fixed or modernized rulesets coexist with it as separate per-game rulesets.

The current goal is a self-contained game that Elegy plays by itself, first with scripted players and then with the approved computer opponents (see "Milestones"). UI parity, original-file compatibility, networking and a Steam-ready application are not current goals.

## Core architecture

```text
Game (state + ruleset) + order files + explicit RNG state
                         ↓
                engine.GenerateTurn
                         ↓
      TurnResult: next Game, events, per-player views, scores
```

The simulation does not depend on:

- a graphical or terminal UI;
- wall-clock time;
- ambient filesystem state;
- networking;
- original Stars! save formats;
- hidden global randomness or process-wide settings.

Packages:

| Package | Role | Status |
|---|---|---|
| `engine/` | state types, orders, `GenerateTurn`, scanning and views, the component table, rulesets | [KERNEL-STATUS.md](KERNEL-STATUS.md), [ORDERS-LAYER-STATUS.md](ORDERS-LAYER-STATUS.md), [ORDERS-STATUS.md](ORDERS-STATUS.md), [COMBAT-STATUS.md](COMBAT-STATUS.md), [SCANNING-STATUS.md](SCANNING-STATUS.md), [TAKEOVER-STATUS.md](TAKEOVER-STATUS.md), [COMPONENTS-STATUS.md](COMPONENTS-STATUS.md), [RULESET.md](RULESET.md) |
| `objects/` | minefields, wormholes, the Mystery Trader, packets, stargates, salvage, reached through `engine.SpaceObjects` | [OBJECTS-STATUS.md](OBJECTS-STATUS.md) |
| `terraform/` | terraforming, Orbital Adjusters, PP packet terraforming, remote mining | [TERRAFORM-STATUS.md](TERRAFORM-STATUS.md) |
| `races/` | race points, repairs, Random races, built-in computer races, the yearly race check | [RACES-STATUS.md](RACES-STATUS.md) |
| `newgame/` | a new game from settings and a seed | [UNIVERSE-STATUS.md](UNIVERSE-STATUS.md) |
| `game/` | the game loop: drivers, reports, save/load, hash, diff, order files | [GAME-LOOP.md](GAME-LOOP.md) |
| `ai/` | Robotoid, Rototill and Cybertron as game drivers | [AI-STATUS.md](AI-STATUS.md) |
| `cmd/elegy` | the command line | [GAME-LOOP.md](GAME-LOOP.md) "Command line" |

Each status file says what is implemented, which tests are ground truth and which are predictions, and the open spec gaps. Packages depend downward: `objects/`, `terraform/` and `races/` import `engine/` and plug into it through interfaces on `engine.Game`; `game/` builds on all of them; `ai/` sees only `game.Report`.

### Truth versus player knowledge

Authoritative state and player-visible information are separate. `GenerateTurn` returns one `engine.PlayerView` per player, and `game.Report` gives a driver that view plus the player's own objects and history, never the authoritative game. Computer players, scripted players and front ends all work from a `Report`.

### Rulesets

Each game carries an immutable typed `engine.Ruleset` (identity, version and one setting per compatibility switch). It is set at creation and saved with the game. There is no process-wide compatibility setting, so games with different rulesets run in one process. The built-in rulesets are `elegy` (the default) and `jrc3-faithful`. [RULESET.md](RULESET.md) is the inventory of every switch, its default in each ruleset and its evidence. A new compatibility difference is a new ruleset setting, not a constant or package variable.

### Data first

Favor straightforward data structures and testable transformations. Split packages when real subsystem boundaries appear rather than pre-creating a large architecture.

## Milestones

### Done

- **Kernel and economy:** population, economy, research and production, with a homeworld simulated 2407–2436 against oracle values.
- **Expansion:** galaxy generation, designs, fleets, movement and fuel, cargo, colonization.
- **Information model:** scanning and `PlayerView`, and per-player reports with planet history.
- **Combat and the long tail:** battles, bombing and invasion, starbases, minefields, terraforming, packets, gates, wormholes, the Mystery Trader and random events.
- **Per-game rulesets** replacing compile-time and process-wide switches.
- **The game loop, save/load and a command line.**

### Next: Elegy plays a long game by itself

The headline milestone is an Elegy game generated and advanced for many years entirely without the original executable, with deterministic replay, valid player views and actionable diagnostics. The original remains the independent parity oracle.

What the tests show today:

- `game` `TestSmokeGame`: a three-player galaxy played for 40 years by a scripted driver that researches, builds, colonizes and scouts, with every order accepted.
- `TestSmokeDeterministic` and `TestSmokeSaveLoadContinues`: the same seed gives the same state hash every year, and a game saved and loaded mid-way continues to the same hashes. With computer players, `ai` `TestDriversSaveReload` shows the same: a game with Robotoid, Rototill and Cybertron saved and reloaded into fresh drivers, mid-way or every year, continues to the uninterrupted game's hashes for 50 years, under both built-in rulesets.
- `TestRulesetsCoexist`: two games with different rulesets in one process.
- `TestReportHoldsOnlyOwnObjects`: a report holds only the player's own objects in full.
- `game.Check` after every year, plus typed `DriverError`, `TurnError` and `InvariantError`. `game.Diff` names the first differing JSON paths when two runs diverge.
- `cmd/elegy` `TestCLIComputerPlayers`: `elegy play -ai ...` plays a human against Robotoid, Rototill and Cybertron, alone and together, for 40 years, with no rejected orders and the same output on a second run. Save/reload continuation with computer players is `TestDriversSaveReload` above.
- `engine` `TestConfirmedAlternateRealityColonyStarbase` and `TestAlternateRealityColonyGeneratesYears`: a new Alternate Reality colony gets its owner's first starbase design and the game keeps generating years.

Known gaps that a long game can reach:

- Waypoint tasks that load cargo or scrap ships are not modelled, and a minefield detonate order is refused (`engine/orders.go` `validTask`, `DetonateOrder`).
- An Alternate Reality colony whose owner has no starbase design gets none (TAKEOVER-STATUS.md, UNRESOLVED).

### After that: games against the computer opponents

Robotoid, Rototill and Cybertron play first one at a time, then together. The command line already plays them (`elegy play -ai robotoid,rototill,cybertron`, GAME-LOOP.md "Command line"), and they play 40- and 60-year smoke games in the tests. Their turns are still partial: AI-STATUS.md "Not implemented yet" lists the missing steps. A game with computer players replays identically, uninterrupted or across a save and reload into fresh drivers (`TestDriversSaveReload`, [#70](https://github.com/bfaber-centaur/elegy/issues/70)). Turindrone, Automitron and Macinti are reference behavior only. The roster changes only by project decision.

## Front ends

The engine should support more than one client.

Near-term experimentation may include a keyboard-first TUI. A bespoke graphical desktop client can become the mainstream product later. Neither should own simulation rules. Today the only front end is `cmd/elegy`, which reads and writes save files and order files.

## Testing

Use several layers:

- formula/unit tests;
- subsystem interaction tests;
- complete turn tests;
- multi-turn deterministic scenarios (the `game` and `ai` smoke games);
- J-RC3 parity tests promoted from the research repository.

Test names carry their evidence: `TestConfirmed*` is ground truth from oracle observations and must not be changed to make code pass, `TestPrediction*` pins a BINARY-ONLY rule from the spec, and `TestElegyDecision*` pins one of Elegy's own choices. `go test -run Confirmed ./...` runs only the ground truth.

### Parity vectors

The stars-elegy parity vectors are copied into `engine/testdata/vectors` and run by `TestParityVectors` (`engine/parity_test.go`) under the `elegy` ruleset. Each vector's start becomes an Elegy game; the harness generates the run's years and checks every case's expectations. Each vector runs with 8 seed variants, because the harness's random stream is not the original's. A case gets one result:

- **pass:** every compared expectation matches with every seed. CONFIRMED and LEGACY BUG cases are exact-match targets. MEASURED passes are counted in their own column: the value is the original's, but the prediction made in advance missed or the outcome is random (vectors README "Tags").
- **sample-only:** every compared expectation is a sample (one stream's random outcome) and matched. It is evidence of the rule, not of an exact value.
- **random:** the result changes with the seed. It is reported with how many seeds pass and the reference seed's comparison. A case like this matches only by chance.
- **differs:** a LEGACY BUG case whose setting is off in `elegy`.
- **fail:** a compared expectation does not match. MEASURED cases fail here too.
- **skip:** the case needs an order, object, expectation or state Elegy does not model or the vector does not carry. The output names the reason. A sample mismatch is skipped, never failed. An expectation tagged with one oracle stream is skipped while the case's other expectations are still checked. When a year's orders include one Elegy cannot load, every case of that vector is skipped.

`go test -run TestParityVectors -v ./engine` prints the per-corpus tally and every case that does not pass. The known failures and the skipped groups are listed in [KERNEL-STATUS.md](KERNEL-STATUS.md) "Known parity failures" and "Integration handoff list". `baseline.txt` lists the passing cases, with each random and sample-only case's count. The test fails if a listed case regresses. After a change that moves the results, check each change and then refresh the list with `PARITY_BASELINE=write go test -run TestParityVectors ./engine`. A change in how many draws the turn makes moves every random case's count.

New-game vectors (`ug`, `rw`) run through `newgame` in `TestUGVectors` instead, because the engine harness has no new-game path.

Passing tests are not proof of exact parity. Never turn a skip into a pass, or count a sample as proof.

For uncertain compatibility behavior:

```text
hypothesis → experiment / evidence → specification → test → implementation
```

## Persistence

A saved game is Elegy's own deterministic, versioned JSON document (`"format": "elegy-save"`), described in [GAME-LOOP.md](GAME-LOOP.md) "Saved games". It holds the whole game state, the ruleset and the random stream, and a game saves back byte for byte. Computer players keep no state of their own: their designs' creation years and pictures are design-slot fields the save holds. Order files are Elegy's own JSON too.

The original binary formats are not Elegy's save model. If they are ever supported, it will be as import/export compatibility.
