# Orders layer status

The orders layer (`engine/orders.go`, `orders_cargo.go`,
`orders_design.go`) is what a player may order for a year and the checks
Elegy applies when it accepts the orders. It follows the public
specification stars-elegy `docs/ORDERS.md` at PR #51 head `14e3c10`, with
battle plans from `docs/COMBAT.md` at PR #59 head `1026471`, research
from `docs/KERNEL.md` "Research" and the planet side of cargo from
`docs/TAKEOVER.md` on main `09e94c4`. Nothing else was used.

The merge order and the design read are the kernel lane's
`Game.MergeFleets` and `Catalog.ReadDesign` (`engine/fleetops.go`,
[ORDERS-STATUS.md](ORDERS-STATUS.md)); the orders layer calls them.

`ApplyOrders` is not called from `GenerateTurn` yet. The replay order is
a random draw whose place in the year's random sequence belongs to the
turn (KERNEL.md "Turn order" step 1), and the drops and gifts the orders
make have to be handed to the waypoint phases. That wiring is a turn
change for the kernel lane.

## Orders and checks

| Order | Checks and effect | Spec | Status | Test |
|---|---|---|---|---|
| (file) | another game, an earlier year or a later year: the whole file is refused | ORDERS.md "Wrong game or wrong year" | BINARY-ONLY | `TestPredictionFileAcceptance` |
| (file) | players replayed in a given order, later wins; no file keeps standing orders | ORDERS.md "Conflicts between players", "A player who submits nothing" | BINARY-ONLY | `TestPredictionReplayOrder` |
| (every order) | an order naming another player's fleet or planet is rejected | ORDERS.md "Ownership" | chosen rule | `TestPredictionOwnershipEveryOrder` |
| `ResearchOrder` | budget 0..100, a field, a next-field choice; anything else rejects the order | ORDERS.md "Research allocation" | BINARY-ONLY | `TestPredictionResearchOrder` |
| `BattlePlanOrder` | replaces a plan or adds the next one, at most 16; tactic and targets in range | COMBAT.md "Adding, replacing and deleting"; ORDERS.md "Battle-plan fields" | BINARY-ONLY; chosen rule | `TestPredictionBattlePlanOrders` |
| `DeletePlanOrder` | never plan 0; later plans and fleets on them move down one | COMBAT.md (BP-1); ORDERS.md | CONFIRMED; chosen rule | `TestConfirmedDeletePlanRenumbers` |
| `FleetPlanOrder` | own fleet, a plan the player has | ORDERS.md "Battle-plan fields" | chosen rule | `TestPredictionBattlePlanOrders` |
| `DesignOrder` | read by `ReadDesign` against the player's race and tech | ORDERS.md "Design legality" | chosen rule | `TestPredictionDesignOrders` |
| `DeleteDesignOrder` | the design's starbase is removed, the population stays | KERNEL.md "Maximum population" | BINARY-ONLY | `TestPredictionDesignOrders` |
| `CargoOrder`, same owner | clamped to source, hold and tank (independent); own planet takes colonists at once | ORDERS.md "Cargo amounts and clamps"; TAKEOVER.md "Unload and load amounts" | CONFIRMED (FO-01..07, TK-201) | `TestConfirmedCargoClamps`, `TestConfirmedCargoOwnPlanet` |
| `CargoOrder`, other owner | giving only; colonists to a planet become a drop; minerals and fuel are taken now and credited by `DeliverGifts`, the rest lost if it does not fit | ORDERS.md "Cross-owner cargo" | BINARY-ONLY | `TestPredictionCrossOwnerCargo`, `TestPredictionCargoToForeignFleet` |
| `CargoOrder` to another player's fleet | no colonists; nothing to a receiver that regards the giver as an enemy | TAKEOVER.md "Other waypoint tasks" | as there | `TestPredictionCargoToForeignFleet` |
| `WaypointOrder` | own fleet; coordinates clamped to the galaxy box; planet and fleet targets take their position | ORDERS.md "Waypoint coordinates"; UNIVERSE.md | BINARY-ONLY | `TestPredictionWaypointClamp` |
| `RenameOrder` | own fleet | ORDERS.md "Ownership" | chosen rule | `TestRenameOrder` |
| `MergeOrder` | `Game.MergeFleets` | ORDERS.md "Merge" | see ORDERS-STATUS.md | (kernel lane) |
| `DetonateOrder` | rejected: Elegy has no minefields yet | ORDERS.md "Minefield detonate-setting" | not modelled | `TestPlaceholderOrders` |
| `QueueOrder`, `PlanetFlagsOrder` | ownership checked, then rejected | ORDERS.md "Ownership"; queue replace spec pending | PLACEHOLDER | `TestPlaceholderOrders` |
| `SettingsOrder` | rejected | settings-orders spec pending | PLACEHOLDER | `TestPlaceholderOrders` |

## Implementation assumptions

Elegy's own choices where the specs are silent. They are **not**
established Stars! behavior; each is marked `ASSUMPTION Ln` in the code
and has been sent to stars-elegy as a question.

| Id | What Elegy does | Why |
|---|---|---|
| L1 | Fleet, design and battle-plan names are at most 31 characters, any text. | No spec gives a name limit; PRODUCTION-LAUNCH.md only says the client shows 28 characters of a design name. |
| L2 | A player's second file in a year is refused; the first applies. | ORDERS.md has one file per player. |
| L3 | At most 16 battle plans (the host's limit, not the client's 15). | COMBAT.md gives both and does not say which to enforce. |
| L4 | An attack-who value outside its five values, or "attack a player" naming no other existing player, rejects the plan. | ORDERS.md names only the tactic and target ranges. |
| L5 | A battle-plan number beyond the next one is rejected. | COMBAT.md: a new plan "takes the next number". |
| L6 | A warp outside 0..10, a negative transport amount, an unmodelled task, or a target planet or fleet that does not exist rejects a waypoint order. | ORDERS.md gives no other waypoint checks. |
| L7 | A cargo order needs the fleet at its target; fuel to or from a planet rejects it; any failed check rejects the whole order; any fleet with a hold may carry colonists. | ORDERS.md and KERNEL.md cover the waypoint unload of fuel, not a direct transfer, and leave the colonist condition unpinned. |
| L8 | The enemy check on cargo for another player's fleet is made when the order applies. | TAKEOVER.md says nothing moves, not when it is checked. |
| L9 | Cargo given to a fleet or planet that no longer exists is lost. | ORDERS.md "Cross-owner cargo" does not say. |
| L10 | 16 ship and 10 starbase design slots per player. | No spec gives the number yet (COVERAGE.md lists it among the limits). |
| L11 | A design slot out of range, a hull of the wrong kind for the slot, and a new design for a slot whose design is in play are rejected. | ORDERS.md does not say what replacing a design in use does. |
| L12 | Deleting a design removes its ships, and fleets left empty. | No spec says what happens to them. |

## Not modelled

- the replay-order draw and calling `ApplyOrders` from `GenerateTurn`;
  `DeliverGifts` is not called either (ORDERS.md puts the credit after
  movement, TAKEOVER.md step 2 before it);
- splits, the transfer-balance between the player's own fleets, and
  cargo to deep space;
- Mystery Trader items (the player owns none), minefields, stargates;
- production queue replace, planet flags and settings orders, until their
  spec lands.
