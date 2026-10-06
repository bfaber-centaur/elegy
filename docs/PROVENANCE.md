# Provenance and repository boundary

Elegy is intended to be an independently written public implementation.

This document is an engineering boundary, not a claim that any particular legal doctrine or formal clean-room process has been satisfied.

## Repository roles

### Elegy

`bfaber-centaur/elegy` contains:

- original engine/application source written for Elegy;
- tests derived from documented behavioral facts;
- original documentation and assets owned or appropriately licensed by the project.

It must not contain original Stars! binaries/assets, registration material, raw disassembly/decompiler output, or copied source-like reconstructions of the original executable.

### Stars' Elegy research

`bfaber-centaur/stars-elegy` is the public behavioral/specification layer. It records:

- controlled oracle experiments;
- parity notes;
- documented formulas and uncertainties;
- public tooling and sanitized fixtures.

This is the preferred source of compatibility requirements for Elegy.

### Private archaeology

`bfaber-centaur/stars-oracle-apparatus` and `bfaber-centaur/stars-decomp` are private research workspaces.

They may contain raw evidence or binary-analysis material that should not be copied into this public repository.

## Promotion path

Prefer this flow:

```text
original behavior / private analysis
            ↓
behavior-level claim, measurement, or testable algorithm
            ↓
stars-elegy documentation / validation
            ↓
independently written Elegy implementation + tests
```

White-box analysis can tell us what question to ask and can inform a behavioral specification. Do not paste decompiler output or transcribe original assembly/source-like code into Elegy.

When a mechanic is uncertain, preserve the uncertainty rather than making the engine look more authoritative than the evidence.
