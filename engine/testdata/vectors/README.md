# Parity vectors

A copy of `vectors/` from `bfaber-centaur/stars-elegy` at `main` bd46dfc:
public, behavior-level oracle cases. `FORMAT.md` is that
directory's README. `engine/parity_test.go` runs them; `baseline.txt`
lists the cases Elegy passes, and the test fails if one of them stops
passing. Production queues are loaded through FORMAT.md "Queue items"; a queue
holding an item Elegy does not build yet (designs, terraforming, packets,
scanners) is not, and checks it could affect are skipped. Submitted
orders are converted to Elegy orders (`parity_orders_test.go`); a vector
with an order kind the harness does not convert (designs, splits, ship
moves) is skipped. Refresh by copying the directory again and updating the commit
above.
