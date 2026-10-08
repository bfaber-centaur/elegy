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
| `elegy` | 1 | `engine.ElegyRules()`: the behaviour Elegy had before rulesets existed. Every switch on except `merge_overflow`, `keep_unentitled_parts` and `field_limit_511`, where Elegy has chosen its own rule. |
| `jrc3-faithful` | 1 | `engine.FaithfulRules()`: every legacy switch on. Only as faithful as the switches go; behaviour Elegy models differently without a switch is the same in both. |

A built-in ruleset's settings never change once released. A change is a
new version, and the old version stays in `engine.Rulesets()` so a game
saved under it still loads and replays. `Ruleset.Validate` refuses a
ruleset that claims a built-in ID and version with other settings, a
built-in ID with an unknown version, an empty ID, or a version below 1.
Any other ID is a custom ruleset, whose settings are whatever it carries.
The zero `Ruleset` is no ruleset: `GenerateTurn` and `newgame.Generate`
refuse it (`engine.ErrNoRuleset`).

Adding a switch adds a `Legacy` field. A save written before the field
existed decodes it as `false`, so the save format's loader has to know
which ruleset fields its version had.

## Switch inventory

Each switch, when on, reproduces a behaviour of the original game that
Elegy would otherwise do differently. The spec column is the rule's place
in the stars-elegy behavioural record and its evidence; the code comment
at the use site has the detail.

Status (2026-10-08): the inventory and the type are in place, and every
game carries its ruleset, but the switch sites still read the
package-level constants and variables listed under "Today's source".
Tests check that `ElegyRules()` equals those values. Moving each site to
read `Game.Rules` and deleting the package-level values is the next step.

| Saved name | Field | `elegy` v1 | `jrc3-faithful` v1 | Today's source | Spec and evidence |
|---|---|---|---|---|---|
| `fuel_wrap` | `FuelWrap` | on | on | `engine/movement.go` `legacyFuelWrap` | KERNEL.md "Designs without a full set of engines", LEGACY BUG, CONFIRMED FM-105 |
| `colocation` | `Colocation` | on | on | `engine/scanning.go` `legacyColocation` | SCANNING.md "Co-location", LEGACY BUG, CONFIRMED SC-002, SC-014 |
| `drop_scan` | `DropScan` | on | on | `engine/takeover.go` `legacyDropScan` | TAKEOVER.md "Several players dropping at once", LEGACY BUG, CONFIRMED T-32 |
| `starbase_armed_class` | `StarbaseArmedClass` | on | on | `engine/combat_token.go` `legacyStarbaseArmedClass` | COMBAT.md "Starbases in battle", LEGACY BUG, CONFIRMED CB-011..013 S4/S5 |
| `observer_tech_mask` | `ObserverTechMask` | on | on | `engine/combat.go` `legacyObserverTechMask` | COMBAT.md "Tech from battle", LEGACY BUG, CONFIRMED CB-031-obs, CB-037 |
| `plan0_recipient` | `Plan0Recipient` | on | on | `engine/combat_who.go` `legacyPlan0` | COMBAT.md "LEGACY BUG: plan 0 ...", CONFIRMED CB-011..013, CB-022, CB-035 |
| `comet_axes` | `CometAxes` | on | on | `engine/randomevents.go` `legacyCometAxes` | KERNEL.md "Comet strike", LEGACY BUG, CONFIRMED KX-004 S2 (message only) |
| `merge_dilution` | `MergeDilution` | on | on | `engine/fleetops.go` `legacyMergeDilution` (also read by `taskMergeRule`) | ORDERS.md "Merge", LEGACY BUG, CONFIRMED FO-01..07 |
| `merge_overflow` | `MergeOverflow` | off | on | `engine/fleetops.go` `legacyMergeOverflow` | ORDERS.md "Merge", LEGACY BUG, CONFIRMED FO; parity cases FO-03-E and FO-06-G differ under `elegy` |
| `keep_unentitled_parts` | `KeepUnentitledParts` | off | on | `engine/fleetops.go` `legacyKeepUnentitledParts` | ORDERS.md "Design legality (Mystery Trader parts kept)", LEGACY BUG, CONFIRMED |
| `field_limit_511` | `FieldLimit511` | off | on | `objects/minefields.go` `LegacyFieldLimit511` | OBJECTS.md "Laying", LEGACY BUG, MEASURED MF-13 |
| `empty_fleet_salvage` | `EmptyFleetSalvage` | on | on | `objects/minefields.go` `LegacyEmptyFleetSalvage` | OBJECTS.md "Hits on moving fleets", LEGACY BUG candidate, MEASURED OB-024 |
| `due_north_south_cut` | `DueNorthSouthCut` | on | on | `objects/minefields.go` `LegacyDueNorthSouthCut` | OBJECTS.md "Due-north and due-south legs", LEGACY BUG, MEASURED MF-15 |
| `mine_survivor_salvage` | `MineSurvivorSalvage` | on | on | `objects/minefields.go` `LegacyMineSurvivorSalvage` | OBJECTS.md "Cargo when ships are destroyed", LEGACY BUG candidate, MEASURED MF-14 |
| `gate_mixed_fleet_loss` | `GateMixedFleetLoss` | on | on | `objects/stargates.go` `LegacyGateMixedFleetLoss` | OBJECTS.md "Mixed fleets", LEGACY BUG, MEASURED GT-001 H2, GT-002, CONFIRMED GT-003 W0–W5 |
| `trader_last_redraw` | `TraderLastRedraw` | on | on | `objects/trader.go` `LegacyTraderLastRedraw` | OBJECTS.md "Encounters", LEGACY BUG, BINARY-ONLY |
| `shared_homeworld_minerals` | `SharedHomeworldMinerals` | on | on | `newgame/players.go` `legacySharedHomeworldMinerals` | UNIVERSE.md "Shared starting minerals", LEGACY BUG, CONFIRMED UG16–UG21 |
| `second_planet_fallback` | `SecondPlanetFallback` | on | on | `newgame/players.go` `legacySecondPlanetFallback` | UNIVERSE.md "Second planet", LEGACY BUG, CONFIRMED UG29, UG30 (success on exactly the 100th redraw BINARY-ONLY) |

## Related behaviour that is not a switch yet

- **AR planet with population, habitability ≥ 0 and no starbase.** The
  original stops generation with an integer divide by zero (KERNEL.md
  "Maximum population", LEGACY BUG, CONFIRMED KX-001 Z1). Today `GenerateTurn` refuses such a game
  (`*engine.ZeroMaxPopulationError`) under every ruleset. It will become a
  switch: the faithful setting keeps the stop, and Elegy's ruleset keeps
  generating with the closest behaviour the public spec supports,
  INTENTIONALLY DIFFERENT (Bobby's decision, 2026-10-08).
- **`legacy_ai_state_leak`** (AI-STATUS.md, `ai/doc.go`) is named but not
  implemented: Elegy's computer players never leak state between AIs.
