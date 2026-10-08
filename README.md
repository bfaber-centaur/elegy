# Elegy

Elegy is a modern, deterministic 4X game engine inspired by *Stars!*.

The long-term goal is a complete game with a faithful J-RC3-compatible ruleset, room for explicitly modernized rulesets, and multiple front ends. The code here is the **canonical implementation**. It is intentionally separate from the archaeology used to learn how the original game behaves.

## Status

Elegy can generate a galaxy and play it for many years without the original program. The turn is complete enough for that, but it is not complete: there is no front end, and several rules are partial or missing. Each package's `docs/*-STATUS.md` file lists what it implements and what it does not.

- **The turn** (`engine/`): orders, movement and fuel, production of installations, ships, starbases, packets and terraforming, population, research, random events, battles, bombing, invasion, colonization, scanning and per-player views, scores and victory. Each rule is tested against the stars-elegy specification, and many against observations of the original game.
- **Space objects** (`objects/`, `terraform/`): minefields, wormholes, the Mystery Trader, mineral packets, stargates, salvage, terraforming, Orbital Adjusters and remote mining.
- **New games** (`newgame/`, `races/`): galaxy, homeworlds, starting players and fleets, race scoring and repair, the 24 built-in computer races.
- **Rulesets**: each game carries its own typed ruleset (`elegy` by default, or `jrc3-faithful`). Every compatibility switch is a setting of it. See [docs/RULESET.md](docs/RULESET.md).
- **Game loop** (`game/`): advance years with per-player drivers, per-player reports, a versioned save format with a state hash, deterministic replay and diffs between games. See [docs/GAME-LOOP.md](docs/GAME-LOOP.md).
- **Computer players** (`ai/`): Robotoid, Rototill and Cybertron, as game drivers. They play smoke games inside the tests. Their turns are partial: the steps not implemented yet are listed in [docs/AI-STATUS.md](docs/AI-STATUS.md) "Not implemented yet". They are not yet selectable from the command line.
- **Command line** (`cmd/elegy`): new games, order files, turns, reports and hashes. There is no other front end yet.

Parity with the original game is measured by the parity vectors in `engine/testdata/vectors`. `go test -run TestParityVectors -v ./engine` prints each corpus's passes, samples, failures and skips, with every skip's reason. [engine/testdata/vectors/README.md](engine/testdata/vectors/README.md) explains what a sample is. Passing those tests does not mean exact parity in every case.

## Quick start

Requires Go 1.23+.

```sh
go test ./...                                                        # everything
go run ./cmd/elegy play -seed 1 -players 3 -years 40                 # idle players; one state hash per year
go run ./cmd/elegy new -seed 1 -players 3 -size small -o game.json   # -rules jrc3-faithful for every legacy switch on
go run ./cmd/elegy orders -game game.json -player 0 > orders0.json   # an empty order file to fill in
go run ./cmd/elegy turn -game game.json orders0.json                 # generate the next year
go run ./cmd/elegy report -game game.json -player 0                  # what player 0 knows, as JSON
```

## Project split

- **`bfaber-centaur/elegy`**: the canonical public implementation.
- **`bfaber-centaur/stars-elegy`**: public behavioral research, oracle tooling, parity notes, and experiments against the original game.
- **`bfaber-centaur/stars-oracle-apparatus`**: private raw apparatus and experimental evidence.
- **`bfaber-centaur/stars-decomp`**: private binary-analysis workspace.

Behavioral findings should become explicit specifications/tests before they become engine behavior. Original binaries, proprietary assets, raw decompiler output, and registration-bearing evidence do not belong in this repository.

## Design direction

The engine should be:

- deterministic from explicit state, orders, ruleset, and RNG state;
- independent of UI, filesystem layout, networking, and original Stars! file formats;
- explicit about compatibility rulesets versus deliberate changes;
- structured around authoritative game state and player-visible information as separate concepts.

The TUI and eventual graphical client should consume the same engine rather than embedding simulation rules.

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the architecture, roadmap and testing, and [docs/PROVENANCE.md](docs/PROVENANCE.md) for the repository boundary.

## License

Apache-2.0. Game assets may use separate licenses later.
