# Parity vectors

Machine-readable test cases taken from oracle runs of the original Stars!
J-RC3. Each file is one run: a starting position, the number of years the
original game generated from it, and what the original produced. An
implementation passes a case when it reaches the same observable values from
the same start.

Every value here is a behavior-level quantity (positions, cargo, tech levels,
design slots, who knows a wormhole). The files carry no game-file bytes,
serial numbers or screenshots. The rule each case tests is written up in the
spec document the vector cites; the vector is the evidence, not the rule.

## Files

```text
vectors/<corpus>/<run>.json      one oracle run
```

| Corpus | Runs | Cases | Topic | Spec |
|---|---|---|---|---|
| `fm2` | FM-101..105 | 49 | fleet movement, fuel, warp 10 losses, engine overflow | `docs/KERNEL.md` "Fleet movement" |
| `fo` | FO-01..07 | 48 | transport tasks, merges, scrapping, transfers between players | `docs/ORDERS.md` "Fleet operations" |
| `tk2` | TK-101..121 | 64 | planet takeover: bombing, invasion, what the new owner gets | `docs/TAKEOVER.md` |
| `wt` | WT-001..005 | 31 | wormholes and the Mystery Trader | `docs/OBJECTS.md` |

`internal/vectors` holds the same format as Go types and a test that
decodes every file strictly, so `go test ./...` fails on a malformed vector.

## A vector

```jsonc
{
 "schema": "stars-elegy-vector/1",
 "id": "FM-101",
 "title": "...",
 "source": {"experiment": "experiments/fm2", "spec_rules": "docs/KERNEL.md \"Fleet movement\"",
            "parity": "docs/PARITY.md ...", "raw_evidence": "(private)"},
 "years": 1,                     // years generated from the start
 "random": "single_stream",      // or "several_streams" (below)
 "streams": 1,
 "initial_state": {...},
 "cases": [...]
}
```

### `initial_state`

The whole position the run started from, read from the original's own start
file, so nothing in it is a default you have to guess.

- `game`: universe bounds, player count, whether random events are on.
- `players`: tech levels by field, research accumulated per field, research
  percent and field, relations to each other player (`neutral`, `friend`,
  `enemy`), Mystery Trader items owned, and the race: PRT, LRTs, growth rate,
  habitability (center, low, high per axis, 255 = immune), colonists per
  resource, factory and mine settings, research cost per field.
- `planets`: position, owner (-1 none), mineral concentrations, environment
  (gravity, temperature, radiation as stored values), original environment,
  surface minerals (ironium, boranium, germanium), population **in hundreds**,
  mines, factories, defenses, planetary scanner, starbase (design slot and
  damage). Planets nobody owns may list only position, concentrations and
  environment.
- `designs`, `starbase_designs`: owner, design slot, hull name, then one entry
  per hull slot (`{"count", "part"}`, or `null` for an empty slot).
- `battle_plans`, `production_queues`.
- `fleets`: owner, id, position, the planet orbited, ships (`design` slot and
  `count`, with damage when damaged), cargo, fuel, and the waypoints. A
  waypoint has a position, warp, target (`planet`, `fleet`, `space`,
  `wormhole`, `trader`) and task. Waypoint 0 is where the fleet is now. Transport tasks list one order per cargo type:
  `load_all`, `unload_all`, `fill_to_percent`, `wait_for_percent`,
  `load_optimal`, `set_amount_to`, `set_waypoint_to` with a value.
- `objects`: wormhole ends (partner, stability class, years since the last
  jump, players who know it, players who know where it leads), Mystery
  Traders (destination, warp, players met), mineral packets, minefields.

The orders a case tests are the fleets' waypoints and tasks in this state.
No other orders were submitted for the generated years.

### `cases`

```jsonc
{
 "id": "FM-101-A",
 "rule": "KERNEL IFE",             // the rule label the experiment used
 "setup": "QJ5 scout, warp 6, +100 x, 300 mg: ...",
 "tag": "CONFIRMED",
 "prediction_held": true,
 "varies_by_stream": false,
 "expect": [
  {"year": 1, "kind": "fleet", "owner": 0, "id": 0,
   "equals": {"x": 1041, "y": 1005, "fuel": 295, "next_waypoint_warp": 6}}
 ]
}
```

`expect` lists what the original produced, observed after `year` generated
years. Check only the fields listed; everything else in the game is
unconstrained by that case.

| kind | subject | `equals` fields |
|---|---|---|
| `fleet` | fleet `owner`/`id` | `x`, `y`, `fuel`, `cargo`, `ships`, `damage`, `next_waypoint_warp`, `first_waypoint_task` |
| `fleet_gone` | fleet `owner`/`id` no longer exists | |
| `fleet_at` | a fleet of `owner` at (`x`, `y`) | as `fleet` |
| `no_fleet_at` | no fleet of `owner` at (`x`, `y`) | |
| `no_new_fleets` | `owner` gained no fleet | |
| `planet` | planet `id` | `owner`, `population` (hundreds), `surface_minerals`, `environment`, `original_environment`, `defenses`, `starbase_design`, ... |
| `production_queue` | planet `planet` | the queue as `{id, count}` items |
| `design` | design `owner`/`slot` | `hull`, `slots` |
| `player` | player `id` | `tech`, `research_accumulated`, `mystery_trader_items`, `ship_design_count` |
| `wormhole` | wormhole end `id` | `known_to`, `destination_known_to` |
| `trader` | Mystery Trader `id` | `x`, `y`, `warp`, `destination`, `met` |
| `packet` | packet `owner`/`id` | `x`, `y` (within `tolerance` ly) |
| `salvage_at` | salvage at (`x`, `y`) | `minerals`; `observed: "none"` if there was none |
| `message` | player `player` got message `message_id` | `present` |
| `sample` | a random outcome | see below |

### Tags

- `CONFIRMED`: the original matched a prediction made in advance from the
  spec rule, and the outcome does not depend on the random stream.
- `MEASURED`: the value is what the original did, but either the
  prediction missed (the cited PARITY section says what the miss showed) or
  the outcome is random. The observed value is still the original's behavior.
- `LEGACY BUG`: the original's behavior here is a defect that the spec
  documents. Elegy's default is to reproduce such behavior behind a named
  switch; see the cited spec.

`prediction_held` is kept separately so you can see which cases changed the
spec.

### Randomness

The original seeds its random numbers from the clock, so a run cannot give
you the original's random stream. Vectors handle this two ways:

- `random: "several_streams"`: the same start was run under several seeds.
  An expectation that came out the same in every stream is listed once; one
  that differed is listed once per stream with a `stream` label, and the case
  has `varies_by_stream: true`. Treat those as samples of a distribution.
- `kind: "sample"`: the case tests a random outcome. `observed` is what the
  original produced (per stream when there are several) and `constraint` says
  what the rule allows, as a short description or the bounds the experiment
  checked. Check that your result satisfies the constraint and, over many
  seeds, that the observed values look typical.

A case with `void_streams` lists streams where its setup was spoiled by an
unrelated random event (a Mystery Trader changing speed before the staged
meeting); those streams are left out of `expect`.

## Building

`tools/vectors/build.py` regenerates a corpus from the private raw evidence
(start files and check output in `stars-oracle-apparatus`). It writes only the
decoded quantities above. Implementers never need to run it.

```sh
python3 tools/vectors/build.py fm2 ../stars-oracle-apparatus/evidence/fm2
python3 tools/vectors/build.py wt  ../stars-oracle-apparatus/evidence/wt
go test ./internal/vectors
```

## Not converted yet

- FM-000..004 (round 1, `experiments/fm00N`) and TK-001..007 (round 1,
  `experiments/tk/gen.py`): recorded as prose and TSV tables rather than
  checker output.
- Combat (`cb*`), scanning (`sc`), universe generation (`ug`), races (`rd`),
  objects (`ob`), minefields (`mf`), kernel (`kx*`), and the other corpora:
  left for their owners.
