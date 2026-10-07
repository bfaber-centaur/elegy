# Parity vectors

A copy of `vectors/` from `bfaber-centaur/stars-elegy` at `main` 63635f0:
public, behavior-level oracle cases. `FORMAT.md` is that
directory's README. `engine/parity_test.go` runs them; `baseline.txt`
lists the cases Elegy passes, and the test fails if one of them stops
passing. A planet's production queue is not loaded (FORMAT.md gives no
planetary item ids), so checks it could affect are skipped. Refresh by copying the directory again and updating the commit
above.
