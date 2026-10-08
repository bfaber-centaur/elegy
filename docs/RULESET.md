# Rulesets

A game's ruleset says which compatibility behaviours the game follows.
It is part of the game: `engine.Game.Rules`, set at creation from
`newgame.Settings.Rules` and passed on unchanged by every
`GenerateTurn`. There is no process-wide compatibility setting, so two
games with different rulesets can run in one process.

The type is `engine.Ruleset` (`engine/ruleset.go`):

```go
type Ruleset struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Legacy  Legacy `json:"legacy"` // one bool per switch
}
```

It is plain data (no pointers, slices, maps or funcs), so a copy is
independent, `==` compares two rulesets exactly, and it round-trips
exactly through `encoding/json`. A save stores the whole value, settings
included, not just the ID.

## Built-in rulesets

| ID | Version | Settings |
|---|---|---|
| `elegy` | 2 | `engine.ElegyRules()`, the default: version 1 with `zero_max_population_stop` off, so an Alternate Reality planet with population and no starbase no longer stops the year (Bobby's "Keep going", 2026-10-08). |
| `elegy` | 1 | The behaviour Elegy had before rulesets existed. Every switch on except `merge_overflow`, `keep_unentitled_parts` and `field_limit_511`, where Elegy has chosen its own rule. Kept for games saved under it. |
| `jrc3-faithful` | 2 | `engine.FaithfulRules()`: every legacy switch on. Only as faithful as the switches go; behaviour Elegy models differently without a switch is the same in both. Adds `cybertron_packet_mark_next_id` and `cybertron_scanner_shot_overflow` (both on) to version 1. |
| `jrc3-faithful` | 1 | Every switch that existed before the two Cybertron switches on. Kept for games saved under it. |

A built-in ruleset's settings never change once released. A change is a
new version, and the old version stays in `engine.Rulesets()` so a game
saved under it still loads and replays. `Ruleset.Validate` refuses a
ruleset that claims a built-in ID and version with other settings, a
built-in ID with an unknown version, an empty ID, or a version below 1.
Any other ID is a custom ruleset, whose settings are whatever it carries.
The zero `Ruleset` is no ruleset: `GenerateTurn` and `newgame.Generate`
refuse it (`engine.ErrNoRuleset`).

Adding a switch adds a `Legacy` field. A save written before the field
existed decodes it as `false`. So a built-in version released before a
switch has it off: a new switch that is on in a built-in ruleset makes
a new version of that ruleset, and one that is off leaves the version
alone (the two Cybertron switches made `jrc3-faithful` v2 and left `elegy` at
v2).

## Switch inventory

Each switch, when on, reproduces a behaviour of the original game that
Elegy would otherwise do differently. The spec column is the rule's place
in the stars-elegy behavioural record and its evidence; the code comment
at the use site has the detail.

Every switch that has a use is read from the game's own ruleset
(`g.Rules.Legacy`, or the new-game settings' `Rules` during generation);
no package-level compatibility constant or variable remains. The two
Cybertron switches have no use yet. A function that has no game
takes the switch as a parameter from a caller that does.

| Saved name | Field | `elegy` v2 | `jrc3-faithful` v2 | Read in | Spec and evidence |
|---|---|---|---|---|---|
| `fuel_wrap` | `FuelWrap` | on | on | `engine/movement.go` `Game.fuelTerm` | KERNEL.md "Designs without a full set of engines", LEGACY BUG, CONFIRMED FM-105 |
| `colocation` | `Colocation` | on | on | `engine/scanning.go` `seesFleet` | SCANNING.md "Co-location", LEGACY BUG, CONFIRMED SC-002, SC-014 |
| `drop_scan` | `DropScan` | on | on | `engine/takeover.go` `Game.dropWinner` | TAKEOVER.md "Several players dropping at once", LEGACY BUG, CONFIRMED T-32 |
| `starbase_armed_class` | `StarbaseArmedClass` | on | on | `engine/combat_battle.go`, `starbaseClass` | COMBAT.md "Starbases in battle", LEGACY BUG, CONFIRMED CB-011..013 S4/S5 |
| `observer_tech_mask` | `ObserverTechMask` | on | on | `engine/combat.go` `battle.techAttempts` | COMBAT.md "Tech from battle", LEGACY BUG, CONFIRMED CB-031-obs, CB-037 |
| `plan0_recipient` | `Plan0Recipient` | on | on | `engine/combat_who.go` `legacyPlan0Recipient` | COMBAT.md "LEGACY BUG: plan 0 ...", CONFIRMED CB-011..013, CB-022, CB-035 |
| `comet_axes` | `CometAxes` | on | on | `engine/randomevents.go` `Game.cometStrike` | KERNEL.md "Comet strike", LEGACY BUG, CONFIRMED KX-004 S2 (message only) |
| `merge_dilution` | `MergeDilution` | on | on | `engine/fleetops.go` `Game.taskMergeRule` | ORDERS.md "Merge", LEGACY BUG, CONFIRMED FO-01..07 |
| `merge_overflow` | `MergeOverflow` | off | on | `engine/fleetops.go` `Game.mergeTask` | ORDERS.md "Merge", LEGACY BUG, CONFIRMED FO; parity cases FO-03-E and FO-06-G differ under `elegy` |
| `keep_unentitled_parts` | `KeepUnentitledParts` | off | on | `engine/fleetops.go` `Catalog.ReadDesign` (takes the ruleset) | ORDERS.md "Design legality (Mystery Trader parts kept)", LEGACY BUG, CONFIRMED |
| `zero_max_population_stop` | `ZeroMaxPopulationStop` | off (on in v1) | on | `engine/turn.go` `GenerateTurn` (`checkGenerable`) | KERNEL.md "Maximum population", LEGACY BUG, CONFIRMED KX-001 Z1; see below |
| `field_limit_511` | `FieldLimit511` | off | on | `objects/minefields.go` `Space.Lay` | OBJECTS.md "Laying", LEGACY BUG, MEASURED MF-13 |
| `empty_fleet_salvage` | `EmptyFleetSalvage` | on | on | `objects/minefields.go` `mineCargo` | OBJECTS.md "Hits on moving fleets", LEGACY BUG candidate, MEASURED OB-024 |
| `due_north_south_cut` | `DueNorthSouthCut` | on | on | `objects/minefields.go` `CheckStep` | OBJECTS.md "Due-north and due-south legs", LEGACY BUG, MEASURED MF-15 |
| `mine_survivor_salvage` | `MineSurvivorSalvage` | on | on | `objects/minefields.go` `mineCargo` | OBJECTS.md "Cargo when ships are destroyed", LEGACY BUG candidate, MEASURED MF-14 |
| `gate_mixed_fleet_loss` | `GateMixedFleetLoss` | on | on | `objects/stargates.go` `Jump` | OBJECTS.md "Mixed fleets", LEGACY BUG, MEASURED GT-001 H2, GT-002, CONFIRMED GT-003 W0–W5 |
| `trader_last_redraw` | `TraderLastRedraw` | on | on | `objects/trader.go` `Space.reward` | OBJECTS.md "Encounters", LEGACY BUG, BINARY-ONLY |
| `shared_homeworld_minerals` | `SharedHomeworldMinerals` | on | on | `newgame/players.go` `setUpHomeworld` | UNIVERSE.md "Shared starting minerals", LEGACY BUG, CONFIRMED UG16–UG21 |
| `second_planet_fallback` | `SecondPlanetFallback` | on | on | `newgame/players.go` `setUpSecondPlanet` | UNIVERSE.md "Second planet", LEGACY BUG, CONFIRMED UG29, UG30 (success on exactly the 100th redraw BINARY-ONLY) |
| `cybertron_packet_mark_next_id` | `CybertronPacketMarkNextID` | off | on (off in v1) | `ai/packets.go` scannerShot, from `Report.Rules` | stars-elegy `docs/ai/cybertron.md` §6 "Packet marks": the scanner shot marks the planet one id above its destination, LEGACY BUG, MEASURED AI-24 |
| `cybertron_scanner_shot_overflow` | `CybertronScannerShotOverflow` | off | on (off in v1) | `ai/packets.go` scannerShot, from `Report.Rules` | stars-elegy `docs/ai/cybertron.md` §6 "Scanner shot" step 5: for `w` ≥ 14 the `w⁴` distance test overflows and passes every planet, LEGACY BUG, BINARY-ONLY |

Not every off setting has been exercised. One known gap: with
`colocation` off, a fleet with no scanner still sees a cloaked fleet at its
own position, because the range test passes at distance 0, so the off
setting does not yet do what its comment says (found while writing the
coexistence test; the off path was never reachable before rulesets).

## Alternate Reality with maximum population 0

An Alternate Reality planet with population, habitability ≥ 0 and no
starbase has maximum population 0. The original divides by it in
population growth and stops with an integer divide by zero, writing no
file (KERNEL.md "Maximum population", LEGACY BUG, CONFIRMED KX-001 Z1).
There is no original behaviour to copy, so each setting is a choice:

- `zero_max_population_stop` on (`jrc3-faithful`, `elegy` v1): the
  closest Elegy comes to the original's stop. `GenerateTurn` returns a
  `*engine.ZeroMaxPopulationError` naming the planet and changes nothing.
- Off (`elegy` v2), INTENTIONALLY DIFFERENT: the year is generated with
  every rule as KERNEL.md states it, and the one division takes its limit
  as the maximum goes to 0. The crowding permille is unbounded, so a
  planet with more than 10 units is overcrowded at the cap,
  `g = 4·(−300)`, and loses 12% a year (KERNEL.md "Population growth");
  10 units or fewer are within 10 of the maximum and stay as they are.
  Other rules see the maximum of 0 as they would any maximum (for
  example, effective population above the maximum).

A hostile planet (habitability < 0) in the same state takes the hostile
death rule and generates normally under every ruleset (CONFIRMED, KX-001
Z3).

## Related behaviour that is not a switch yet

- **`legacy_ai_state_leak`** (AI-STATUS.md, `ai/doc.go`) is named but not
  implemented: Elegy's computer players never leak state between AIs.
