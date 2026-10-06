# Elegy

This is the canonical implementation repository for Elegy.

## Start here

Read:

1. `README.md`
2. `docs/DEVELOPMENT.md`
3. `docs/PROVENANCE.md`
4. relevant code and tests

Then run:

```sh
go test ./...
```

## Working principles

Keep the simulation deterministic and independent from UI, filesystem layout, networking, and original Stars! file formats.

Prefer simple data and pure transformations over speculative abstractions.

Do not silently invent Stars! mechanics. When compatibility behavior matters, use the behavioral record in `bfaber-centaur/stars-elegy` as the specification and add a focused regression test.

This repository is not an archaeology workspace. Do not add original binaries/assets, raw registered-copy evidence, disassembly, or decompiler output here. See `docs/PROVENANCE.md`.

It is fine to follow implementation clues far enough to understand the current problem; avoid unrelated refactors or speculative subsystem design.

Before handing work back:

```sh
gofmt -w <changed-go-files>
go test ./...
git status
git diff
```

Use a branch/PR for substantial work.
