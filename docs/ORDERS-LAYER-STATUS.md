# Orders layer status

The orders layer (`engine/orders.go`, `orders_cargo.go`,
`orders_design.go`) is what a player may order for a year and the checks
Elegy applies when it accepts the orders. It follows the public
specification stars-elegy on main (`b654cb3`): `docs/ORDERS.md`, with
battle plans and their order validation from `docs/COMBAT.md`, research
and the replay shuffle from `docs/KERNEL.md`, manual transfers to other
players and the planet side of cargo from `docs/TAKEOVER.md`, and queue
replace, setting orders, names and design slots from `docs/LIMITS.md`,
and the packet settings and items from `docs/OBJECTS.md` "The settings"
and `docs/KERNEL.md` "Packet items" (stars-elegy `c0bee4b`). Nothing else was used.

The merge order and the design read are the kernel lane's
`Game.MergeFleets` and `Catalog.ReadDesign` (`engine/fleetops.go`,
[ORDERS-STATUS.md](ORDERS-STATUS.md)); the orders layer calls them.

`YearOrders` draws the replay order with `ShufflePlayers` (KERNEL.md
"Turn order", 1. Orders, step 2: a forward shuffle,
one draw per player, the first draws of the year) and applies the files.
`GenerateTurn` calls it first (KERNEL.md step 1) and returns each
order's result in `TurnResult.Orders`. The orders' colonist drops go to
the front of the before-movement drop queue, in the order given
(TAKEOVER.md "Order inside a phase", MEASURED TK-501); gifts are credited
at the end of the replay, before any waypoint task.

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
| `DesignOrder` | read by `ReadDesign` against the player's race and tech; the slot records the year as `Created` and the order's `Picture` (AI.md "Storing a design"; L25, L26), and production counts its ships ever built as `Built` (L29); a slot whose design has ships or a starbase is refused; any other filled slot is overwritten in place | ORDERS.md "Design legality", "Design change into an occupied slot" | queue-only edit MEASURED (CO-08); refusal is Elegy's rule (the client cannot edit such a design) | `TestPredictionDesignOrders`, `TestMeasuredDesignEditQueueOnly`, `TestDesignOrderCreatedAndPicture` |
| `DesignOrder`, malformed fills | a part the slot does not take dropped, a count over capacity cut, parts above tech stripped, an empty engine slot back-filled with Quick Jump 5 | ORDERS.md "Design read, four malformed cases" (Q12) | BINARY-ONLY | `TestDesignMalformedFills` |
| `DeleteDesignOrder` | the design's ships, fleets left empty and starbase removed, the population stays; the slot cleared in place, later slots keep their numbers; removed ships take ⌊amount × their capacity ÷ fleet capacity⌋ of the fleet's fuel and cargo | ORDERS.md "Design delete effect"; KERNEL.md "Maximum population" | MEASURED (CO-07, CO-07b, CO-07c); population BINARY-ONLY | `TestPredictionDesignOrders`, `TestMeasuredDesignDeleteInPlace`, `TestMeasuredDesignDeleteSharesFuel` |
| `CargoOrder`, same owner | clamped to source, hold and tank (independent); own planet takes colonists at once | ORDERS.md "Cargo amounts and clamps"; TAKEOVER.md "Unload and load amounts" | CONFIRMED (FO-01..07, TK-201) | `TestConfirmedCargoClamps`, `TestConfirmedCargoOwnPlanet` |
| `CargoOrder`, own fleets | the amount given moves (not a rebalance), capped by what the source holds and the receiver's free space | ORDERS.md "Transfer between the player's own fleets" | CONFIRMED (CO-04) | `TestConfirmedOwnFleetTransferExplicit` |
| `CargoOrder`, preconditions | fleet at the target or refused; no jettison; planet fuel dropped, the rest moves; any fleet with free hold carries colonists | ORDERS.md "Elegy implementation Q3", "Q4" | chosen rules | `TestCargoChecks` |
| `CargoOrder`, another owner's planet | giving only; colonists are a drop (another player's planet without a starbase) or lost (unowned, or a starbase); minerals credited in place as the order applies, no message; relations do not matter | ORDERS.md "Cross-owner cargo"; TAKEOVER.md "Manual cargo transfers to other players" | CONFIRMED (TK-501, TK-502), minerals MEASURED (TK-405, TK-412) | `TestConfirmedManualTransfersToOthers`, `TestGiftCreditedInPlace` |
| `CargoOrder`, another player's fleet | giving only; colonists rejected (no legal client writes them, TK-408, TK-414); credited in place as the order applies; what does not fit is lost and the giver told; no relation check; a later merge or design delete disposes of the gift as of the fleet's own cargo | ORDERS.md "Cross-owner cargo"; TAKEOVER.md "Manual cargo transfers to other players" | MEASURED (TK-406, TK-407, TK-409); over-full gift CONFIRMED; colonists chosen rule | `TestPredictionCargoToForeignFleet`, `TestGiftReceiverRemovedEarlier`, `TestGiftToFleetMergedAway`, `TestMeasuredGiftLostWithDeletedDesign` |
| `WaypointOrder` | own fleet; coordinates clamped to the galaxy box; planet, fleet, wormhole-end and Trader targets take their position (L27); a warp outside 0..11, a missing target or a negative transport amount rejected, except a fleet a merge order removed earlier in the replay, which is kept with the order's coordinates (ORDERS-STATUS.md); warp 11 (the stargate hop, GT-004) accepted, the gate checked at the jump; route, patrol and transfer-fleet tasks accepted, a transfer to any player (refused when the task runs), a negative patrol range rejected (L18); remote-mining and lay-mines tasks accepted, refused when they run, a lay-mines duration below one year other than indefinitely rejected (L24) | ORDERS.md "Waypoint coordinates", "Waypoint warp, target and transport" (Q13); OBJECTS.md "Stargates", "Laying", "Waypoint upkeep and the remaining tasks", "Transfer fleet"; KERNEL.md "Remote mining"; UNIVERSE.md | BINARY-ONLY; chosen rule | `TestPredictionWaypointClamp`, `TestWaypointOrderUpkeepTasks`, `TestWaypointOrderMiningAndLayingTasks`, `TestWaypointOrderObjectTargets` |
| `RepeatOrder` | own fleet; sets or clears the repeat-orders flag (the host checks no owner, LEGACY BUG) | ORDERS.md "Reaching a waypoint", "Ownership"; LIMITS.md "No owner check" | chosen rule | `TestRepeatOrder` |
| `RenameOrder` | own fleet | ORDERS.md "Ownership" | chosen rule | `TestRenameOrder` |
| `MergeOrder` | `Game.MergeFleets` | ORDERS.md "Merge" | see ORDERS-STATUS.md | (kernel lane) |
| `SplitOrder` | own fleet; a new fleet of the named ships with the source's plan, waypoints and task; cargo and fuel shared by capacity, rounded down, the remainder on the source; lowest free number (L20); repeat flag copied (L21); refused at 512 fleets; a NewFleet name below 0 stands for the new fleet in the file's later orders (L28) | ORDERS.md "Split" | CONFIRMED (CO-01, CO-02); FC-1 | `TestConfirmedSplitSharesByCapacity`, `TestConfirmedSplitAllRemainderOnSource`, `TestSplitNewFleetName` |
| `MoveShipsOrder` | two own fleets at one place; ships move with their capacity share of cargo and fuel; damage combined by the merge order's rule; at most 32765 per stack, the rest lost; damaged stacks split keep their damage (L22); an emptied fleet removed (L23) | ORDERS.md "Split", "Merge" | CONFIRMED (CO-03); stack limit MEASURED (CO-06) | `TestConfirmedMoveShips`, `TestMoveShipsChecks` |
| `DetonateOrder` | sets the detonate setting of the player's own minefield through `SpaceObjects.SetDetonate`: owner only, Space Demolition only, standard fields only; a refusal is the order's error; no space objects: `ErrNoSuchObject` | ORDERS.md "Minefield detonate-setting", OBJECTS.md "The detonate setting" | BINARY-ONLY (chosen rule) | `TestDetonateWithoutObjects`, `TestDetonateOrderThroughEngine` (objects) |
| `QueueOrder` | own planet; empty list removes the queue; otherwise replaced as sent, a sent percentage kept only against an unused old item of the same kind with exactly that percentage (chosen rule) | LIMITS.md "Production-queue replace" | CONFIRMED (LQ-1..LQ-6); chosen rule | `TestConfirmedQueueReplace`, `TestPredictionQueueNoNewProgress` |
| `PlanetSettingsOrder` | own planet; leftover-only, route destination, packet destination (unchecked) and packet speed (0 or 4..19, L19) | LIMITS.md "Setting orders"; OBJECTS.md "The settings" | BINARY-ONLY; chosen rule | `TestPredictionSettingOrders`, `TestPacketSettingsOrder` |
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
| L9 | (settled: ORDERS.md "Receiver removed after the credit, same turn": the gift is credited in place, carries no gift record, and a later removal disposes of it as of the fleet's own cargo) | |
| L10 | (settled: LIMITS.md, 16 ship and 10 starbase slots) | |
| L11 | A hull of the wrong kind for the slot rejects a design; a fill naming a slot the hull lacks, or a count below 1, is dropped; two fills for one slot, or an engine slot short of capacity, reject it. (Occupied slot and the four malformed cases settled: ORDERS.md Q10, Q12.) | ORDERS.md's four cases do not cover these. |
| L12 | (settled: ORDERS.md "Design delete effect", Q11) | |
| L14 | A queue item of unknown kind, a count outside 1..1023, or more than 255 items refuses the queue order. | The host checks none of them (LIMITS.md "Production queue"). |
| L15 | A route destination that is not a planet refuses the planet-settings order. | The host does not check it exists. |
| L16 | A relations row needs one entry per player in range; the sender's own entry stays friend. | LIMITS.md says only the sender's row changes. |
| L17 | (settled: ORDERS.md "Cross-owner cargo", a recipient whose orders replay after the giver's can use the gift the same year, BINARY-ONLY) | |
| L18 | A patrol task with a negative range rejects the waypoint order. | ORDERS.md gives no check on the patrol range. |
| L19 | A packet speed other than 0 (unset) or 4..19 refuses the planet-settings order. | OBJECTS.md "The settings": the stored field holds warps 4..19. |
| L20 | A split's new fleet takes its owner's lowest unused fleet number; a player at 512 fleets cannot split. | ORDERS.md "Split" does not say; CO-02 and FC-1 agree with the new-ship rule. |
| L21 | A split's new fleet copies the source's repeat-orders flag and has no name. | ORDERS.md "Split" names the plan and waypoint list only. |
| L22 | Ships moved out of a damaged stack keep its damage percentage and units, as do those left. | ORDERS.md does not say which ships of a stack are damaged. |
| L23 | A ship move takes ships out of the order's fleet first, then out of the other; a fleet left with no ships is removed. | As merged fleets are (ORDERS.md "Merge"). |
| L24 | A lay-mines task whose duration is neither indefinitely nor at least one year rejects the waypoint order. | OBJECTS.md "Duration" names the three durations and no order-time check. |
| L25 | A design picture outside 0..3 rejects the design order. | AI.md "Picture" names four pictures per hull and no order-time check. |
| L26 | A design edited in place takes the edit's year as its creation year, and the order's picture. | AI.md "Storing a design" covers a delete and a new design only. |
| L27 | A wormhole-end or Mystery Trader waypoint target need not be known to the player. | As planet and fleet targets; ORDERS.md gives no knowledge check. |
| L28 | A split may name its new fleet with a value below 0 that the file's later orders use for it; a name above 0 or used twice rejects the split, and a rejected split's name binds nothing. | ORDERS.md has no such names (the original's client knows a new fleet's number at once); computer players need one (AI.md §10). |
| L29 | A design edited in place starts its built count (ships ever built) again at 0; ships lost at the fleet limit still count; starbase slots count nothing. | ai/turindrone.md defines the built count but not what an edit or the fleet limit does to it. |

## Ships leaving production

`engine/launch.go` follows stars-elegy `docs/PRODUCTION-LAUNCH.md` on main.
Production builds ship and starbase items through `Game.PlanetProduction`
(`engine/production.go`): a design item names one of the owner's design
slots, is spent on like any non-auto item (`KERNEL.md` "Production", the
PQ-001 model), at the design's owner cost (ships), `StarbaseBuildCost`, or
`StarbaseReplacementCost` where a starbase stands. The units an item
completes in a year are one `Launch`; each starbase unit is built as it
completes. `GenerateTurn` calls `PlanetProduction` for each planet;
`RunProduction`, which has no game, still stops the queue at a design
item.

| Rule | Code | Status | Test |
|---|---|---|---|
| Ideal warp per engine | `idealWarp` | CONFIRMED for Long Hump 6 and Quick Jump 5 (SL-04..07), BINARY-ONLY otherwise | `TestIdealWarpPerEngine` |
| Route warp: dock rule, step down, fuel | `routeWarp` | CONFIRMED (SL-04..07, every non-gate row) | `TestConfirmedRouteWarp`, `TestConfirmedRouteWarpTankScout` |
| Route warp: the gate (warp 11) when both own planets are gated, the fleet carries no cargo and the jump is within the source range and both mass limits; new ships and the route task | `routeWarp`, `gateSafe` | CONFIRMED (SL gate rows); route task MEASURED (ORDERS.md wuRSG2) | `TestConfirmedRouteWarpGates`, `TestRouteWarpGateConditions` |
| One fleet per build event, the owner's lowest free `Number`, full tanks, plan 0, no task, no name | `Launch` | CONFIRMED (SL-01, SL-02) | `TestConfirmedLaunchNewFleets` |
| Route destination: a second waypoint with the route task | `Launch` | CONFIRMED (SL-04..07) | `TestConfirmedLaunchRouted` |
| No starbase, or no tech: nothing built | `Launch` | BINARY-ONLY | `TestPredictionLaunchNeedsStarbaseAndTech` |
| 512 fleets: join the first fleet at the planet within 32765, or lose the ships | `joinAtLimit` | CONFIRMED (SL-08..10) | `TestConfirmedFleetLimit` |
| Damage of the joined stack | `joinDamage` | CONFIRMED (SL-10) | `TestConfirmedJoinDamage` |
| Dock check at order validation | `DockAllows` | chosen rule (host has none, SL-12) | `TestDockAllows` |
| New starbase keeps damage units; the owner is told no ships / up to N kT / any size | `BuildStarbase` | CONFIRMED (SL-12); message BINARY-ONLY | `TestConfirmedStarbaseKeepsDamage` |
| At the limit, ships of a design the fleet lacks take their design slot's place | `insertStack` | BINARY-ONLY | `TestPredictionJoinTakesSlotPlace` |
| Replacement cost, different hull | `StarbaseReplacementCost` | MEASURED (SL-12) | `TestMeasuredStarbaseReplacementCost`, `TestMeasuredProductionChargesReplacement` |
| Replacement cost, same hull: slot by slot | `StarbaseReplacementCost` | BINARY-ONLY | `TestSameHullReplacementSameParts` |
| One build event per item per year; a ship item spent on like any item | `PlanetProduction` | CONFIRMED (SL-01); spending KERNEL.md PQ-001 model | `TestConfirmedProductionOneFleetPerItem`, `TestProductionShipPartial` |
| Without a starbase: resources spent, nothing built, item removed | `PlanetProduction`, `Launch` | BINARY-ONLY | `TestPredictionShipWithoutStarbase` |
| An earlier hull removes queued ships and resets starbase items | `afterEarlierHull` | CONFIRMED (SL-12); earlier items built MEASURED once | `TestConfirmedEarlierHullClearsShips` |
| Design delete drops the slot's queue entries | `DeleteDesignOrder` | MEASURED (CO-07) | `TestMeasuredDesignDeleteDropsQueue` |
| Design edit: a queue entry builds the edited design | queue items name the slot | MEASURED (CO-08) | `TestMeasuredDesignEditQueueBuildsEdited` |
| Ship item dock check at order validation | `QueueOrder`, `DockAllows` | chosen rule | `TestQueueOrderDesignItems` |
| Packet item costs | `ItemCost` | minerals MEASURED (OB-028, OB-029) except IT mixed; resources BINARY-ONLY (KERNEL.md "Item costs") | `TestPacketItemCostMatchesLaunch` |
| Packet items: unit loop, one launch per item per year | `PlanetProduction`, `launchPackets` | BINARY-ONLY (KERNEL.md "Packet items") | `TestPacketItemUnitLoop` |
| No driver or destination: the item removed, cancel then "completed its orders" | `plain` | MEASURED (OB-028-F) | `TestMeasuredPacketItemNoDestination` |
| Auto Mineral Packets: mixed units up to the count (at most 1000), auto skip and partial rules, nothing without a driver | `autoInstall` | BINARY-ONLY (KERNEL.md "Packet items") | `TestAutoMineralPackets` |
| Terraform Environment: unit cost, the order cut to the capacity with a message or removed at 0 when the queue reaches it, one improving click per unit at the levels before research | `plain`, `production_terraform.go` via `Game.Terraform` (`terraform.Rules`) | CONFIRMED (KX-002 T1–T3, KB-2C, KX-005; KERNEL.md "Terraforming") | `TestConfirmedTerraformItemClipAndClicks`, `TestConfirmedTerraformItemNoCapacity` |
| Auto Max / Auto Min Terraform: `AutoUnits` with this year's population change (production runs before the year's population is written, engine/turn.go) | `autoInstall` | CONFIRMED (KX-005) | `TestConfirmedAutoTerraform` |

Production assumptions:

| Id | What Elegy does | Why |
|---|---|---|
| P1 | A design item whose slot holds no design is removed with nothing spent. | Deleting a design removes its items, so only a state built without design orders reaches it. |
| P2 | A same-hull starbase replacement whose designs do not record slot positions is charged the fresh cost. | Designs from `NewDesign` record them (`Design.SlotPos`). |
| P3 | A queue order with a design item naming an empty or out-of-range slot is refused. | The host checks no item ids (LIMITS.md "Production-queue replace"). |
| P4 | A queue that a design delete leaves empty is removed. | KERNEL.md says a zero-item queue does not arise in play. |
| P6 | A packet destination naming no planet counts as no destination. | OBJECTS.md "The settings": the original reads past its planet table; Elegy needs a chosen rule. |
| W6 | Without space objects (`Game.Objects`) routing never chooses a gate. | A gate jump does nothing without them. |
| P7 | An Auto Alchemy prefix before a packet item removed for want of a driver or destination stays and stands before the next item. | KERNEL.md does not say what happens to the prefix. |

Not modelled there: the Alternate Reality remote-mining
task, the "did not move" mark (GenerateTurn must not mark a fleet built
this year as stationary), and the route task on arrival. Packet items
need `Game.Objects` and terraform items `Game.Terraform`; without them,
and in `RunProduction`, they stop the queue.

## Open items for the next owner

A durable list as of the orders lane's last PR. Each item names what a
successor needs before it can change.

### Orders Elegy does not implement

- **Mystery Trader items**: no order reaches them (the player owns none).
- **The queued cross-player credit routine**: no legal order is known to
  reach it (ORDERS.md, stars-elegy #87, UNRESOLVED HYPOTHESIS).
- **Production side, not orders**: the Alternate Reality remote-mining
  task on a new fleet, the "did not move" mark (GenerateTurn must not
  mark a fleet built this year as stationary), and the route task on
  arrival. Packet items need `Game.Objects` and terraform items
  `Game.Terraform`; without them, and in `RunProduction`, they stop the
  queue.

### Orders Elegy refuses on purpose

- **Cargo to deep space (jettison)**: refused under ORDERS.md "Deep-space
  jettison" (Elegy's chosen rule; settled, no open question).

### Labelled choices

Every Elegy choice is in the tables above: L2, L4, L6, L7, L11, L14–L16,
L18–L29 (orders), P1–P4, P6, P7 (production), W6 (routing). Each is an
`ASSUMPTION` in the code. L1, L3, L5, L8–L10, L12 and L17 are
settled by the specs (L13 was never used).

### Open questions for stars-elegy

- L22: which ships of a damaged stack a split or ship move takes, and
  how the damage divides.
- L23: the order of the two directions in one ship move, and whether an
  emptied fleet is removed at once.
- L20 / L21: the new fleet's number, the 512-fleet case, its repeat flag
  and name on a split.
- L19: whether a packet speed outside 4..19 can be stored.
- P6: a packet destination that names no planet.
- P7: the Auto Alchemy prefix before a removed packet item.
- The two stack caps: the merge order keeps 32766 per design, which is
  Elegy's chosen rule and unconfirmed because no case reached the merge
  order's own clamp (ORDERS.md "Merge order"), while a ship move keeps
  32765 (MEASURED CO-06). A case reaching the merge clamp would settle
  whether the two differ.
