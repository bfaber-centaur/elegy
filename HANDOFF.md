# Handoff

## Current question

The engine has just been split out of `stars-elegy`. There is no active feature mission yet; establish this repository as the canonical implementation and grow it only from settled behavioral knowledge.

## State

- Branch: `main`
- Engine: tiny prototype moved from `stars-elegy`
- Implemented: year advance, standard positive-world population capacity, simple uncrowded growth scaling
- Not implemented: J-RC3 growth carry, crowding, hostile-world deaths, economy, fleets, or other major systems

## What we know

The current `PopulationGrowth` function is only a skeleton. Public research has already established a persistent population-growth remainder in J-RC3; do not mistake the current implementation for the behavioral specification.

## Best next move

When the population-growth research in `stars-elegy` is sufficiently settled, promote the exact state representation and uncrowded/crowded behavior here with parity-backed tests. Until then, keep engine changes small and explicit.
