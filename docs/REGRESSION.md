# Long-game regression

`regress/` plays long Elegy games against the computer players and
checks every year that each game stays valid and deterministic. It
reports what it finds by category. It does not judge whose bug a finding
is.

## A case

A case is one game: a seed, a built-in ruleset (`elegy` or
`jrc3-faithful`) and a set of computer opponents at expert level. The
game is played on a small map with normal density and moderate
positions. Player 0 is an idle human with the default race. The
computer players follow, using their built-in races and `ai.Driver`.

Each case is played three times in lockstep:

- **primary:** the game;
- **replay:** a second game from the same seed, which must give the
  same state hash every year;
- **reload:** the primary saved and loaded after `SaveAt` years, with
  fresh drivers, which must give the primary's hash every year after
  that, and right after the load.

Every primary year must pass `game.Check` (`game.Advance` refuses a year
that fails it) and the extra year checks in `regress/checks.go`: the
engine's structural invariants from the year before to the year
generated (`engine.CheckYear`, each invariant cited in its doc).

## Categories

Failures, any of which fails the case:

| Category | Meaning |
|---|---|
| setup failed | the game could not be created |
| driver error | a computer player's driver returned an error |
| turn refused | the engine refused a year |
| invariant failed | the new state failed `game.Check` |
| year check failed | the new state failed an extra year check |
| replay differs | the replay's hash differs from the primary's |
| reload differs | the reloaded copy's hash differs from the primary's |
| save or load failed | the save or the load at `SaveAt` failed |
| computer order rejected | the engine rejected a computer player's order (the first one, with the count) |

One observation is counted but is not a failure:

- **unsupported computer steps:** the total of `ai.Driver.Unsupported`,
  the steps a computer player could not order yet (AI-STATUS).
  A case that stops early keeps the count of the years it played.

A failure gives the game year being generated, the detail, the first
JSON paths where two copies differ (`game.Diff`), and the command that
replays the case alone. A case stops at its primary's first failure:
a refused year or a failed year check. A rejected computer order does
not stop it; the case plays on and counts the rejections. The replay and
reload copies stop at their own first difference.

## Matrices

The PR subset runs in CI's "long-game regression" job on every PR. It
has 3 seeds (1, 2, 3), both rulesets, and 4 opponent sets: Robotoid,
Rototill and Cybertron each alone, then all three. That makes 24 games
of 60 years, reloaded after 23 years. 60 years reach Cybertron's first
warship group. The cases run in parallel; the job takes about 30
seconds.

The scheduled "Long games" workflow runs nightly, and on demand from
the Actions tab. It has 12 seeds (1–12), both rulesets and the same 4
opponent sets: 96 games of 150 years on a medium map, reloaded after 77
years. On demand, its seeds and years can be changed.

To replay a case or run another matrix:

```sh
go test ./regress -run TestLongGames -count=1 -v -args -seeds=7 -rules=elegy -ai=rototill -years=40 -save-at=17 -size=1
go test ./regress -run TestLongGames -count=1 -v -args -seeds=1,2,3,4,5,6,7,8 -years=100 -save-at=41
```

`-ai` takes comma-separated opponent sets, each `name+name`. `-size`
takes 0 tiny to 4 huge.
