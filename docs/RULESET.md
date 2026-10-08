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

Every switch is read from the game's own ruleset (`g.Rules.Legacy`, or
the new-game settings' `Rules` during generation); no package-level
compatibility constant or variable remains. A function that has no game
takes the switch as a parameter from a caller that does.

| Saved name | Field | `elegy` v1 | `jrc3-faithful` v1 | Read in | Spec and evidence |
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
| `field_limit_511` | `FieldLimit511` | off | on | `objects/minefields.go` `Space.Lay` | OBJECTS.md "Laying", LEGACY BUG, MEASURED MF-13 |
| `empty_fleet_salvage` | `EmptyFleetSalvage` | on | on | `objects/minefields.go` `mineCargo` | OBJECTS.md "Hits on moving fleets", LEGACY BUG candidate, MEASURED OB-024 |
| `due_north_south_cut` | `DueNorthSouthCut` | on | on | `objects/minefields.go` `CheckStep` | OBJECTS.md "Due-north and due-south legs", LEGACY BUG, MEASURED MF-15 |
| `mine_survivor_salvage` | `MineSurvivorSalvage` | on | on | `objects/minefields.go` `mineCargo` | OBJECTS.md "Cargo when ships are destroyed", LEGACY BUG candidate, MEASURED MF-14 |
| `gate_mixed_fleet_loss` | `GateMixedFleetLoss` | on | on | `objects/stargates.go` `Jump` | OBJECTS.md "Mixed fleets", LEGACY BUG, MEASURED GT-001 H2, GT-002, CONFIRMED GT-003 W0–W5 |
| `trader_last_redraw` | `TraderLastRedraw` | on | on | `objects/trader.go` `Space.reward` | OBJECTS.md "Encounters", LEGACY BUG, BINARY-ONLY |
| `shared_homeworld_minerals` | `SharedHomeworldMinerals` | on | on | `newgame/players.go` `setUpHomeworld` | UNIVERSE.md "Shared starting minerals", LEGACY BUG, CONFIRMED UG16–UG21 |
| `second_planet_fallback` | `SecondPlanetFallback` | on | on | `newgame/players.go` `setUpSecondPlanet` | UNIVERSE.md "Second planet", LEGACY BUG, CONFIRMED UG29, UG30 (success on exactly the 100th redraw BINARY-ONLY) |

Not every off setting has been exercised. One known gap: with
`colocation` off, a fleet with no scanner still sees a cloaked fleet at its
own position, because the range test passes at distance 0, so the off
setting does not yet do what its comment says (found while writing the
coexistence test; the off path was never reachable before rulesets).

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
