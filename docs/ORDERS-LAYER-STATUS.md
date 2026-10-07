# Orders layer status

The orders layer (`engine/orders.go`, `orders_cargo.go`,
`orders_design.go`) is what a player may order for a year and the checks
Elegy applies when it accepts the orders. It follows the public
specification stars-elegy on main (`b654cb3`): `docs/ORDERS.md`, with
battle plans and their order validation from `docs/COMBAT.md`, research
and the replay shuffle from `docs/KERNEL.md`, manual transfers to other
players and the planet side of cargo from `docs/TAKEOVER.md`, and queue
replace, setting orders, names and design slots from `docs/LIMITS.md`. Nothing else was used.

The merge order and the design read are the kernel lane's
`Game.MergeFleets` and `Catalog.ReadDesign` (`engine/fleetops.go`,
[ORDERS-STATUS.md](ORDERS-STATUS.md)); the orders layer calls them.

`YearOrders` draws the replay order with `ShufflePlayers` (KERNEL.md
"Turn order", 1. Orders, step 2: a forward shuffle,
one draw per player, the first draws of the year) and applies the files.
It is not called from `GenerateTurn` yet: that, and handing the orders'
drops and gifts to the waypoint phases, is a turn change for the kernel
lane.

## Orders and checks

| Order | Checks and effect | Spec | Status | Test |
|---|---|---|---|---|
| (file) | another game, an earlier year or a later year: the whole file is refused | ORDERS.md "Wrong game or wrong year" | BINARY-ONLY | `TestPredictionFileAcceptance` |
| (file) | players replayed in a given order, later wins; no file keeps standing orders | ORDERS.md "Conflicts between players", "A player who submits nothing" | BINARY-ONLY | `TestPredictionReplayOrder` |
| (year) | replay order: forward shuffle, one draw per player, first draws of the year | KERNEL.md step 1 | draw count CONFIRMED (KX-004), permutation BINARY-ONLY | `TestPredictionPlayerShuffle` |
| (every order) | an order naming another player's fleet or planet is rejected | ORDERS.md "Ownership" | chosen rule | `TestPredictionOwnershipEveryOrder` |
| `ResearchOrder` | budget 0..100, a field, a next-field choice; anything else rejects the order | ORDERS.md "Research allocation" | BINARY-ONLY | `TestPredictionResearchOrder` |
| `BattlePlanOrder` | replaces a plan or adds the next one, at most 16; tactic and targets in range | COMBAT.md "Adding, replacing and deleting"; ORDERS.md "Battle-plan fields" | BINARY-ONLY; chosen rule | `TestPredictionBattlePlanOrders` |
| `DeletePlanOrder` | never plan 0; later plans and fleets on them move down one | COMBAT.md (BP-1); ORDERS.md | CONFIRMED; chosen rule | `TestConfirmedDeletePlanRenumbers` |
| `FleetPlanOrder` | own fleet, a plan the player has | ORDERS.md "Battle-plan fields" | chosen rule | `TestPredictionBattlePlanOrders` |
| `DesignOrder` | read by `ReadDesign` against the player's race and tech; a slot whose design has ships or a starbase is refused; any other filled slot is overwritten in place | ORDERS.md "Design legality", "Design change into an occupied slot" | queue-only edit MEASURED (CO-08); refusal is Elegy's rule (the client cannot edit such a design) | `TestPredictionDesignOrders`, `TestMeasuredDesignEditQueueOnly` |
| `DesignOrder`, malformed fills | a part the slot does not take dropped, a count over capacity cut, parts above tech stripped, an empty engine slot back-filled with Quick Jump 5 | ORDERS.md "Design read, four malformed cases" (Q12) | BINARY-ONLY | `TestDesignMalformedFills` |
| `DeleteDesignOrder` | the design's ships, fleets left empty and starbase removed, the population stays; the slot cleared in place, later slots keep their numbers; removed ships take ⌊amount × their capacity ÷ fleet capacity⌋ of the fleet's fuel and cargo | ORDERS.md "Design delete effect"; KERNEL.md "Maximum population" | MEASURED (CO-07, CO-07b, CO-07c); population BINARY-ONLY | `TestPredictionDesignOrders`, `TestMeasuredDesignDeleteInPlace`, `TestMeasuredDesignDeleteSharesFuel` |
| `CargoOrder`, same owner | clamped to source, hold and tank (independent); own planet takes colonists at once | ORDERS.md "Cargo amounts and clamps"; TAKEOVER.md "Unload and load amounts" | CONFIRMED (FO-01..07, TK-201) | `TestConfirmedCargoClamps`, `TestConfirmedCargoOwnPlanet` |
| `CargoOrder`, own fleets | the amount given moves (not a rebalance), capped by what the source holds and the receiver's free space | ORDERS.md "Transfer between the player's own fleets" | CONFIRMED (CO-04) | `TestConfirmedOwnFleetTransferExplicit` |
| `CargoOrder`, preconditions | fleet at the target or refused; no jettison; planet fuel dropped, the rest moves; any fleet with free hold carries colonists | ORDERS.md "Elegy implementation Q3", "Q4" | chosen rules | `TestCargoChecks` |
| `CargoOrder`, another owner's planet | giving only; colonists are a drop (another player's planet without a starbase) or lost (unowned, or a starbase); minerals credited after every file, no message; relations do not matter | ORDERS.md "Cross-owner cargo"; TAKEOVER.md "Manual cargo transfers to other players" | CONFIRMED (TK-501, TK-502), minerals MEASURED (TK-405, TK-412) | `TestConfirmedManualTransfersToOthers`, `TestPredictionGiftsCreditedAfterReplay` |
| `CargoOrder`, another player's fleet | giving only; colonists rejected (no legal client writes them, TK-408, TK-414); debits first, then credits in place during the orders step; what does not fit is lost and the giver told; no relation check | ORDERS.md "Cross-owner cargo"; TAKEOVER.md "Manual cargo transfers to other players" | MEASURED (TK-406, TK-407, TK-409); over-full gift CONFIRMED; colonists chosen rule | `TestPredictionCargoToForeignFleet`, `TestGiftToFleetMergedAway` |
| `WaypointOrder` | own fleet; coordinates clamped to the galaxy box; planet and fleet targets take their position; a warp outside 0..11, a missing target or a negative transport amount rejected; warp 11 (stargate) not modelled | ORDERS.md "Waypoint coordinates", "Waypoint warp, target and transport" (Q13); UNIVERSE.md | BINARY-ONLY; chosen rule | `TestPredictionWaypointClamp` |
| `RenameOrder` | own fleet | ORDERS.md "Ownership" | chosen rule | `TestRenameOrder` |
| `MergeOrder` | `Game.MergeFleets` | ORDERS.md "Merge" | see ORDERS-STATUS.md | (kernel lane) |
| `DetonateOrder` | rejected: Elegy has no minefields yet | ORDERS.md "Minefield detonate-setting" | not modelled | `TestDetonateNotModelled` |
| `QueueOrder` | own planet; empty list removes the queue; otherwise replaced as sent, a sent percentage kept only against an unused old item of the same kind with exactly that percentage (chosen rule) | LIMITS.md "Production-queue replace" | CONFIRMED (LQ-1..LQ-6); chosen rule | `TestConfirmedQueueReplace`, `TestPredictionQueueNoNewProgress` |
| `PlanetSettingsOrder` | own planet; leftover-only and route destination | LIMITS.md "Setting orders" | BINARY-ONLY | `TestPredictionSettingOrders` |
| `RelationsOrder` | only the sender's row | LIMITS.md "Setting orders" | BINARY-ONLY | `TestPredictionSettingOrders` |

## Implementation assumptions

Elegy's own choices where the specs are silent. They are **not**
established Stars! behavior; each is marked `ASSUMPTION Ln` in the code
and has been sent to stars-elegy as a question.

| Id | What Elegy does | Why |
|---|---|---|
| L1 | (settled: LIMITS.md "Names", at most 31 characters of any text) | |
| L2 | A player's second file in a year is refused; the first applies. | ORDERS.md has one file per player. |
| L3 | (settled: COMBAT.md sets Elegy's limit at 16) | |
| L4 | An attack-who value outside Elegy's five rejects the plan. ("Attack a player" naming itself or no player in the game is now COMBAT.md's Elegy rule.) | COMBAT.md describes the host's stored values; Elegy's type has no others. |
| L5 | (settled: COMBAT.md, a definition is accepted only for k ≤ count) | |
| L6 | An unmodelled task rejects a waypoint order. (Warp, target and transport checks settled: ORDERS.md Q13.) | Elegy does not model every task. |
| L7 | Any failed cargo check other than planet fuel rejects the whole order. (Explicit own-fleet amounts settled: CONFIRMED CO-04. Co-location, planet fuel, jettison and the colonist gate settled: ORDERS.md Q3, Q4.) | ORDERS.md does not say whether one bad cargo kind spoils the rest. |
| L8 | (settled: TAKEOVER.md, a legal client never gives colonists to another player's fleet; Elegy rejects it under "Ownership") | |
| L9 | **Elegy choice, not observed behaviour** (switch `giftReturnsToGiver`): a receiving fleet that a later order in the same replay removed (merge, design delete) counts as a missing endpoint; the cargo goes back to the giver's fleet, as much as fits, and the rest is lost with a message. | No measurement covers a receiver removed between debit and credit. BINARY-ONLY analysis on stars-elegy says the cargo is lost; that cargo path is being reconciled there, and Elegy follows the settled rule when it lands. |
| L10 | (settled: LIMITS.md, 16 ship and 10 starbase slots) | |
| L11 | A hull of the wrong kind for the slot rejects a design; a fill naming a slot the hull lacks, or a count below 1, is dropped; two fills for one slot, or an engine slot short of capacity, reject it. (Occupied slot and the four malformed cases settled: ORDERS.md Q10, Q12.) | ORDERS.md's four cases do not cover these. |
| L12 | (settled: ORDERS.md "Design delete effect", Q11) | |
| L14 | A queue item of unknown kind, a count outside 1..1023, or more than 255 items refuses the queue order. | The host checks none of them (LIMITS.md "Production queue"). |
| L15 | A route destination that is not a planet refuses the planet-settings order. | The host does not check it exists. |
| L16 | A relations row needs one entry per player in range; the sender's own entry stays friend. | LIMITS.md says only the sender's row changes. |

## Ships leaving production

`engine/launch.go` follows stars-elegy `docs/PRODUCTION-LAUNCH.md` on main. Production has no ship or starbase items yet, so
`Launch` and `BuildStarbase` are the build steps such an item will call.

| Rule | Code | Status | Test |
|---|---|---|---|
| Ideal warp per engine | `idealWarp` | CONFIRMED for Long Hump 6 and Quick Jump 5 (SL-04..07), BINARY-ONLY otherwise | `TestIdealWarpPerEngine` |
| Route warp: dock rule, step down, fuel | `routeWarp` | CONFIRMED (SL-04..07, every non-gate row) | `TestConfirmedRouteWarp`, `TestConfirmedRouteWarpTankScout` |
| One fleet per build event, the owner's lowest free `Number`, full tanks, plan 0, no task, no name | `Launch` | CONFIRMED (SL-01, SL-02) | `TestConfirmedLaunchNewFleets` |
| Route destination: a second waypoint with the route task | `Launch` | CONFIRMED (SL-04..07) | `TestConfirmedLaunchRouted` |
| No starbase, or no tech: nothing built | `Launch` | BINARY-ONLY | `TestPredictionLaunchNeedsStarbaseAndTech` |
| 512 fleets: join the first fleet at the planet within 32765, or lose the ships | `joinAtLimit` | CONFIRMED (SL-08..10) | `TestConfirmedFleetLimit` |
| Damage of the joined stack | `joinDamage` | CONFIRMED (SL-10) | `TestConfirmedJoinDamage` |
| Dock check at order validation | `DockAllows` | chosen rule (host has none, SL-12) | `TestDockAllows` |
| New starbase keeps damage units; the owner is told no ships / up to N kT / any size | `BuildStarbase` | CONFIRMED (SL-12); message BINARY-ONLY | `TestConfirmedStarbaseKeepsDamage` |
| At the limit, ships of a design the fleet lacks take their design slot's place | `insertStack` | BINARY-ONLY | `TestPredictionJoinTakesSlotPlace` |
| Replacement cost, different hull | `StarbaseReplacementCost` | MEASURED (SL-12) | `TestMeasuredStarbaseReplacementCost` |

Not modelled there: stargate routing, the Alternate Reality remote-mining
task, the "did not move" mark (GenerateTurn must not mark a fleet built
this year as stationary), the queue changes when a starbase is replaced
by an earlier hull, the same-hull replacement cost (Elegy's designs do not
record slot positions), mass drivers, and the route task on arrival.

## Not modelled

- calling `YearOrders` from `GenerateTurn`, and putting its drops at the
  front of the before-movement drop queue;
- splits, and cargo to deep space;
- Mystery Trader items (the player owns none), minefields, stargates;
- mass drivers and packets in planet settings, and ship and starbase
  items in the queue: so a queue entry is not dropped when its design is
  deleted (CO-07), and the queue-only design edit (CO-08) is reached only
  as a slot with nothing in play;
- stargate hops (waypoint warp 11).
