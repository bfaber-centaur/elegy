# Elegy

Elegy is a modern, deterministic 4X game engine inspired by *Stars!*.

The long-term goal is a complete game with a faithful J-RC3-compatible ruleset, room for explicitly modernized rulesets, and multiple front ends. The code here is the **canonical implementation**. It is intentionally separate from the archaeology used to learn how the original game behaves.

Status: early engine. The J-RC3 peaceful turn (population, economy, research, production) and ordinary fleet movement are implemented and tested against original-game observations; see [docs/KERNEL-STATUS.md](docs/KERNEL-STATUS.md).

## Project split

- **`bfaber-centaur/elegy`** — canonical public implementation.
- **`bfaber-centaur/stars-elegy`** — public behavioral research, oracle tooling, parity notes, and experiments against the original game.
- **`bfaber-centaur/stars-oracle-apparatus`** — private raw apparatus and experimental evidence.
- **`bfaber-centaur/stars-decomp`** — private binary-analysis workspace.

Behavioral findings should become explicit specifications/tests before they become engine behavior. Original binaries, proprietary assets, raw decompiler output, and registration-bearing evidence do not belong in this repository.

## Design direction

The engine should be:

- deterministic from explicit state, orders, ruleset, and RNG state;
- independent of UI, filesystem layout, networking, and original Stars! file formats;
- explicit about compatibility rulesets versus deliberate changes;
- structured around authoritative game state and player-visible information as separate concepts.

The TUI and eventual graphical client should consume the same engine rather than embedding simulation rules.

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the current architecture/roadmap and [docs/PROVENANCE.md](docs/PROVENANCE.md) for the repository boundary.

## Development

Requires Go 1.23+.

```sh
go test ./...
```

## License

Apache-2.0. Game assets may use separate licenses later.
