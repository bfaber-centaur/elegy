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
| `sc` | SC-001..034 | 680 | scanning: what each player's file shows of fleets, planets, designs and player blocks | `docs/SCANNING.md` |
| `kx001`, `kx002` | KX-001, KX-002 cases | 38 | planet economy: production, Auto Alchemy, growth, research, mining (one year from an edited PG001 file) | `docs/KERNEL.md` |
| `kx003` | KX-003 r1, r2, r3, r3l | 4 | score records, victory flags, slower tech, Claim Adjuster | `docs/KERNEL.md` |
| `kx004` | KX-004 S1..S10 | 10 | random events and Mystery Trader appearance, 3 to 61 streams each | `docs/KERNEL.md` "Random events" |
| `mf` | MF-01..11c | 20 | minefields: hits, sweeping, decay, detonation, speed bumps | `docs/OBJECTS.md` "Minefields" |
| `rp` | RD-P1..P12 | 12 | turn-time race penalty and repairs | `docs/KERNEL.md` (race budget), PARITY "Turn-time penalty" |
| `cb7` | CB-048, CB-049, SC-035, SC-036 | 5 | combat round 7 (Mystery Trader items from battle, movement) and scanning after battles | `docs/COMBAT.md`, `docs/SCANNING.md` |
| `tk3` | TK-201..203 | 3 | takeover round 3: order across planets, unloads, mines floor, scrapping tech | `docs/TAKEOVER.md` |
| `sl` | SL-01..12 (9 setups) | 9 | ship launch: new fleets, route warps, the 512-fleet limit, starbase replacement | `docs/ORDERS.md` |
| `cb` | CB-001..047 | 68 | combat: battle records and everything a battle turn changed | `docs/COMBAT.md` |

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

- `game`: universe size and bounds, player count, and the game record's
  options (`random_events`, `slower_tech`, `public_scores`) and
  `victory_conditions` (each with its value in game units and, for the seven
  that can be switched, `enabled`), when the vector's builder read them.
- `players`: tech levels by field, research accumulated per field, research
  percent and field, relations to each other player (`neutral`, `friend`,
  `enemy`), Mystery Trader items owned, and the race: PRT, LRTs, growth rate,
  habitability (center, low, high per axis, 255 = immune), colonists per
  resource, factory and mine settings, research cost per field, the leftover
  points spend (`UNIVERSE.md` "Leftover advantage points") and race stat 15 (PARITY "Turn-time penalty"). An out-of-range PRT is given
  as its stored number.
- `planets`: position, owner (-1 none), mineral concentrations, environment
  (gravity, temperature, radiation as stored values), original environment,
  surface minerals (ironium, boranium, germanium), population **in hundreds**,
  mines, factories, defenses, planetary scanner, starbase (design slot and
  damage). Planets nobody owns may list only position, concentrations and
  environment.
- `designs`, `starbase_designs`: owner, design slot, hull name, then one entry
  per hull slot (`{"count", "part"}`, or `null` for an empty slot).
- `battle_plans`, `production_queues` (items `{id, count, percent, kind}`;
  `percent` is the part of the first unit already built, omitted when 0).
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
| `production_queue` | planet `planet` | the queue as `{id, count, percent}` items (`percent` omitted when 0) |
| `design` | design `owner`/`slot` | `hull`, `slots` |
| `player` | player `id` | `tech`, `research_accumulated`, `mystery_trader_items`, `ship_design_count`, `score_record` (score, resources, planets, starbases, unarmed/escort/capital ship counts, tech level sum, rank, `victory_conditions_met`; `seen_by` names the player whose file held it) |
| `wormhole` | wormhole end `id` | `known_to`, `destination_known_to` |
| `trader` | Mystery Trader `id` | `x`, `y`, `warp`, `destination`, `met` |
| `packet` | packet `owner`/`id` | `x`, `y` (within `tolerance` ly) |

`tolerance` as an object (`{"surface_minerals": 1}`) allows that much
difference in the named field. Kernel vectors use it for surface minerals,
because mining's +1 remainder is random (`KERNEL.md`).

| `salvage_at` | salvage at (`x`, `y`) | `minerals`; `observed: "none"` if there was none |
| `message` | player `player` got message `message_id` | `present` |
| `object` | a new or changed map object | the object, as in `initial_state.objects` |
| `object_gone` | the object `subject` (`{kind, owner, id}`) no longer exists, e.g. a swept minefield | |
| `minefield` | minefield `owner`/`id` | `x`, `y`, `mines`, `type`, `detonating`; `radius` when the case checks it |
| `view` | what player `viewer` knows of `subject` (`{kind, owner, id}`: a planet, fleet, wormhole, minefield, design or player) | the fields the case checks, e.g. `level` (report level), `known`, `starbase_visible`, `heading`, `design_count` |
| `battle` | the battle at (`x`, `y`) | `players`, `planet`, and `tokens`: per token its `owner`, `fleet` or `planet`, `kind`, `design`, `start_square`, `initiative`, `mass`, `shield`, `jammer`, `computer`, `capacitor`, `deflector`, `ships`, `damage` |
| `battle_actions` | the moves and shots of that battle | `actions`: per action `round`, `token`, `to` (square, or `null` for leaving the board) and `hits` (`token`, `flags`, `kills`, `shield_damage`, `armor_damage`) |
| `no_battle` | no battle record was written | |
| `client_estimate` | a value the original client shows on `screen` for `subject` after the year | `field` names the value; `equals` holds it as a number or a short value with its unit, never the client's sentence |
| `sample` | a random outcome | see below |

### Tags

- `CONFIRMED`: the original matched a prediction made in advance from the
  spec rule. If the case also `varies_by_stream`, the rule held in every
  stream and the values are each stream's exact outcome.
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

### Combat vectors

The combat rounds have no per-case checker output, so a `cb` vector has one
case per experiment setup and its expectations are the whole observed turn:
every battle record, decoded, and every fleet, planet, player, queue and
salvage change in the host file. The setup's predictions are in
`experiments/cbNNN/README.md`; `verdict` summarizes how they fared and the
tag follows it. Rounds from CB-009 on ran under pinned random streams, so the
same start gave the same record within a stream; CB-001..008 were not
pinned. A battle's token list is often identical in every stream while its
actions differ, which is why `battle` and `battle_actions` are separate.

### New-game vectors

A universe-generation vector has `new_game` (`settings` and `races`) in place
of `initial_state` and `years: 0`; its expectations (`year: 0`) describe the
generated starting game. Values that only the original's random stream
decides, such as planet positions, appear as samples or counts, not as exact
expectations.

### Not vectors

Static part data (CS-001, CS-002 readouts) is in `data/components.json`,
which is already machine-readable, so it has no vectors.

## Building

`tools/vectors/build.py` regenerates a corpus from the private raw evidence
(start files and check output in `stars-oracle-apparatus`). It writes only the
decoded quantities above. Implementers never need to run it.

```sh
python3 tools/vectors/build.py fm2 ../stars-oracle-apparatus/evidence/fm2
python3 tools/vectors/build.py wt  ../stars-oracle-apparatus/evidence/wt
python3 tools/vectors/build.py cb  ../stars-oracle-apparatus/evidence/cb   # reads every cb* round
python3 tools/vectors/build.py sc  ../stars-oracle-apparatus/evidence/sc
python3 tools/vectors/build.py sl  ../stars-oracle-apparatus/evidence/sl   # also cb7, tk3
python3 tools/vectors/build.py mf  ../stars-oracle-apparatus/evidence/mf
python3 tools/vectors/build.py rp  ../stars-oracle-apparatus/evidence/rd
python3 tools/vectors/build.py kx004 ../stars-oracle-apparatus/evidence/kx004   # also kx001..kx003
go test ./internal/vectors
```

## Not converted yet

- FM-000..004 (round 1, `experiments/fm00N`) and TK-001..007 (round 1,
  `experiments/tk/gen.py`): recorded as prose and TSV tables rather than
  checker output.
- CB-000 (ship designer readouts, no turn), CB-018 batch 1 (confounded by
  research), the superseded CB-046 morph v1, and CB-017's second year.
- GT-001/002 (stargates): waiting for their public record in stars-elegy.
  SC-035 v1 (did not test the claim) and the SL tooling check.
- SC-015 (the tamper check fired; SC-015L repeats it) and SC-021 (invalid
  setup).
- KX-001 M3 (the prediction was void: the race edit tripped the tamper
  check), KX-001 Z1/Z1h (the original crashed: no year was generated; see
  PARITY "KX-001 Z"), KX-004 E0/E1 (the long runs that made the start
  states), KX-005 (not merged yet).
- RD-1..RD-7 and RW (new games: they wait for the `new_game` form) and the
  MF-07 verdicts (tagged MEASURED until the owner says whether OBSERVED or
  held is canonical). Universe generation (`ug`), objects (`ob`), `pg`, `pq` and `cs`:
  being converted by their own lane.
