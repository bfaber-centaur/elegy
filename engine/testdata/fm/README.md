# FM-001..004 fleet-movement corpus

Copied unchanged from the public `bfaber-centaur/stars-elegy` repository,
`experiments/fm00N/fm00N.spec` and `experiments/fm00N/results.tsv`
(PR #11 head `4766d27`). Each `.spec` line is one fleet's starting state and
orders; each `results.tsv` row is the original game's observed state of that
fleet one year later. FM-004's results use a different column layout (read
by `TestConfirmedFleetMovementCorpusFM004`). See `docs/PARITY.md`, "Fleet
Movement", in that repository.

The design and engine table used to read the specs (masses, fuel tables,
fuel capacities) is in `movement_corpus_test.go`, taken from the same
repository's `experiments/fmlib.py`.
