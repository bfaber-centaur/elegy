# Space objects: implementation status

`objects/` implements the space objects of stars-elegy `docs/OBJECTS.md`
(as of stars-elegy `main` at `e0b7d1b`) and its Mystery Trader appearance
in `KERNEL.md`, with the part statistics of
`COMPONENTS.md` (the component table the engine embeds). Nothing here
comes from the private archaeology repositories.

The package reads the engine's exported state and exposes step and query
functions. It does not change `engine/`, and `engine.GenerateTurn` does
not call it yet: the turn engine owns the year's order (OBJECTS.md "Turn
placement") and wires each step in. Object families land one at a time:
minefields, wormholes, the Mystery Trader, then mass-driver packets.

## Minefields

| Rule (OBJECTS.md) | Function | Status |
|---|---|---|
| Size and inside test | `Minefield.Contains` | CONFIRMED (OB-002, OB-018-C) |
| Decay, incl. SD and detonation | `DecayLoss`, `Space.Decay` | CONFIRMED (OB-002, OB-014-A, OB-015, OB-016, MF-4) |
| Lay amounts, hull multiplier, MCM | `LayAmounts` | CONFIRMED (OB-002, OB-024) |
| Merging, merge cap, laying order | `Space.Lay` | CONFIRMED (OB-002-G, MF-10, MF-12) |
| SD half lay while moving | `Layer.Half` | CONFIRMED (OB-014-C) |
| 512-field limit | `Space.Lay` | MEASURED (MF-11); the 511 case is LEGACY BUG `LegacyFieldLimit511` (off, Elegy's chosen rule) |
| 4050-object limit | `Space.Lay` | BINARY-ONLY |
| Sweep rating and amount | `SweepRating`, `Space.Sweep` | CONFIRMED (OB-001, OB-007, OB-008, OB-010-S); ratings agree with the table's mines-swept column |
| Effective warp | `EffectiveWarp` | CONFIRMED (OB-010, MF-3) |
| Stop checks along a step | `CheckStep` | CONFIRMED as rates (MF-1, MF-5, MF-6); safe-warp bonuses for SS and SD BINARY-ONLY |
| Hit damage | `MineDamage` | CONFIRMED (MF-9, MF-8, OB-010-S, OB-002-M) |
| Mines lost, paying field | `MinesLost`, `PayingField`, `ApplyHit` | CONFIRMED (OB-010-S, OB-024, MF-4) |
| Salvage from destroyed ships; empty fleet `rand(10)` | `ApplyHit` | MEASURED (OB-024); the empty-fleet drop is a LEGACY BUG candidate, `LegacyEmptyFleetSalvage` (on) |
| SD disclosure on hits and detonations | `Hit.Disclosed`, `Detonation.Disclosed` | CONFIRMED for detonations (MF-7) |
| Detonation | `Space.Detonate` | CONFIRMED (OB-002-M, MF-7, MF-8) |

Tests follow the engine's naming: `TestConfirmed*` use the OB and MF
vectors OBJECTS.md and PARITY.md give; `TestPrediction*` cover the rest.
The MF runs' "Tank" is a Destroyer with two Superlatanium (3200 armor).

### Assumptions (spec gaps)

Marked `ASSUMPTION On` in the code and sent to the spec owner:

1. **O1** A Space Demolition half lay halves each kind's amount,
   truncating.
2. **O2** Ties for the nearest own field when merging go to the first
   field in object order.
3. **O3** A new field takes the owner's lowest unused number.
4. **O4** A step's stretch inside a field runs from the entry distance
   rounded up to the exit distance rounded down, one draw per whole ly from
   entry up to (not including) exit; a stop `k` ly into it is `entry + k`
   from the step's start.
5. **O5** A kind whose safe warp is at or above the effective warp makes
   no draws; stretches with equal entries go standard, heavy, speed bump.
6. **O6** Damage bookkeeping: groups are a fleet's stacks of one design;
   shields absorb ships × the design's shield; existing damage is
   `units·armor/500` per damaged ship, `damaged = ⌈pct·ships/100⌉`; new
   damage is stored as pct 100 and `units = ⌊average·500/armor⌋`; a design
   is destroyed when its average is greater than its armor; exempt layers
   are left out of the fleet minimum.
7. **O7** A destroyed group's share of the fleet's cargo is proportional
   to its cargo capacity (to ship count when the fleet has none).
8. **O8** A fleet inside several detonating fields is hit by the first in
   object order.

### What the turn engine needs to call (minefields)

In OBJECTS.md "Turn placement" order:

- movement: `CheckStep` for each movement step of a fleet moving at warp
  1–10, not through a stargate and not already at its waypoint, then
  `ApplyHit` at the stop point (the fleet stops, gets no ram-scoop fuel,
  and has spent its whole leg's fuel);
- after movement: `Space.Detonate`, then `Space.Decay`;
- waypoint tasks after battles: `Space.Lay` for the fleets laying this
  year (a "lay mines" task with its duration, and the SD half lay);
- then `Space.Sweep`;
- fleets left with no stacks are destroyed; salvage objects and messages
  come from the returned records.

The engine has no minefield list, "lay mines" task or minefield
visibility yet; those belong to the turn engine and scanning code.

## Wormholes

| Rule (OBJECTS.md) | Function | Status |
|---|---|---|
| Pair counts, classes, placement badness at creation | `WormholePairs`, `Surroundings.Badness`, `Surroundings.Place`; used by `newgame` | CONFIRMED (OB-006, UG01–UG21) |
| Jump chance | `JumpChance` | MEASURED (OB-025) |
| Stability name | `StabilityName` | BINARY-ONLY |
| Jump: years reset, knowledge cleared, uniform placement | `Space.MoveWormholes` | MEASURED (OB-005-B), CONFIRMED (OB-025) |
| Jiggle: ±12 per axis, never the old position, class kept | `Space.MoveWormholes` | CONFIRMED (OB-005, OB-017, OB-025) |
| Transit to the partner's start-of-year position, knowledge | `Space.Transit` | CONFIRMED (OB-005 C, D, WT-001) |
| Destination knowledge, shown while the partner is seen | `WormholeEnd.DestKnown`, `Space.Destination` | CONFIRMED (WT-001, WT-004); report BINARY-ONLY |

`newgame` now places wormholes through `objects`; its tests and the
creation results are unchanged.

OBJECTS.md now answers the two wormhole gaps (W1, W2), so they are no
longer assumptions; both answers are BINARY-ONLY:

- Outside the galaxy is a coordinate below 1000 or above 1000 + W; a
  coordinate equal to 1000 + W is inside (`Surroundings.Badness`).
- Jump and jiggle tries use the creation badness, with fleets, minefield
  centres and Traders as objects; each try draws x before y. A jiggle try
  on the old position is skipped without a score but uses up one of the
  100 tries. If every scored try is rejected, the end moves to the first
  of them; it does not stay put (`Surroundings.Place`).

### What the turn engine needs to call (wormholes)

- Fleet movement: a fleet whose reached waypoint targets a wormhole end
  (not a plain position on it, and not one it passes over) calls
  `Space.Transit`; fleets following it lose it.
- A waypoint aimed at an end keeps following it only while its owner knew
  the end (`WormholeEnd.KnownBy`) at the start of the year; otherwise,
  after the wormhole's first move, it becomes a plain position at the
  end's old position (CONFIRMED OB-025-F, OB-027).
- After fleets move (OBJECTS.md "Turn placement" step 7):
  `Space.MoveWormholes`.
- Scanning marks sightings with `WormholeEnd.MarkKnown` and shows
  destinations with `Space.Destination`.

## Mystery Trader

| Rule | Function | Status |
|---|---|---|
| Appearance: chance, warp, start, destination, item, in draw order | `Space.Appear` | CONFIRMED (KERNEL.md, KX-004 S6–S10) |
| Part reroll (bits 6, 7, 10, 11) and conversion of the second draw (7 before 120, 10 before 150, 11 before 180) | `Space.Appear` | BINARY-ONLY |
| Movement: warp rise, then the 1/3 new destination; arrival, leaving | `Space.MoveTraders` | CONFIRMED in part (OB-023, OB-026, OB-031); draws BINARY-ONLY |
| New destination on any of the four edges | `newDestination` | CONFIRMED in part (OB-023); draw order BINARY-ONLY |
| Each year's step, shared with packets | `StepToward` | BINARY-ONLY in detail; agrees with OB-003, OB-023, OB-028 |
| Trade threshold, one trade per player per Trader, fleet consumed | `Space.Meet` | CONFIRMED (OB-004, OB-023, OB-030-T, WT-002..WT-004) |
| Part reward; parts owned per player | `TraderParts`, `TraderParts.Items` | CONFIRMED (WT-003 A) |
| Research reward: L from cargo and tech sum, field choice | `researchReward` | CONFIRMED (WT-002 A, B); field odds MEASURED; draws BINARY-ONLY |
| Every field at 26: nothing with 1/5, else a part or a ship | `Space.Meet` | CONFIRMED (WT-004 C) |
| 25th redraw finds an unowned part but gives a ship | `LegacyTraderLastRedraw` (on) | LEGACY BUG, BINARY-ONLY |
| Ship gifts: designs, first empty design slot | `shipGift`, `giftDesign` | CONFIRMED (WT-003 B, WT-004, OB-026) |
| Gift draw order, matching (earlier gifts only, name not compared), counts, new fleet | `shipGift`, `giftSlot` | BINARY-ONLY; fleet number from 1 as a launch (SL-02) |
| Computer players' planets: after each Trader's fleets, planet-number order, scan stop, d² ≤ 10,000, ship item as bit 12 | `Space.Meet` | CONFIRMED in part (TP-001, TP-002); Harder 3,500 kT CONFIRMED (O-53); order, range and ship items BINARY-ONLY |

Traders sit in `Space.Traders` and count toward the object limit and the
wormhole placement objects. `Space.GiftDesigns` marks the designs a gift
created.

OBJECTS.md now answers every Trader gap of the first drafts (T1–T10):
the gift loadouts' slot layout is MEASURED (WT-004), the rest
BINARY-ONLY. A gift fleet that lands later in fleet order is offered to
the Trader like any fleet and refused (no minerals).

### What the turn engine needs to call (Mystery Trader)

- End of production, after new minerals: `Space.Appear`, then the
  appearance message to every player.
- Before fleets move: `Space.MoveTraders`, then the waypoint refresh, so
  waypoints on a Trader follow its new position. A Trader that left turns
  them into plain positions, and the owner is told.
- After battles: `Space.Meet`, which runs each Trader's fleets and then
  its planets; messages come from the returned records.
- A gift fleet (`Reward.NewFleet`): mark it as not moved this year (it
  gets no refusal message), set its waypoint 0 at the trade point, and
  have it orbit the planet the traded fleet orbited, if any.
- Designs: pass `TraderParts.Items(player)` to `ReadDesign`; drop a
  deleted design's index from `Space.GiftDesigns`.
- Scanning shows Traders as objects; the engine has no Trader visibility
  yet.

## Mass-driver packets

| Rule | Function | Status |
|---|---|---|
| Driver warp `Dw` and the two-driver bonus `t` | `DriverWarp` | CONFIRMED (OB-028-C, D) |
| Packet warp from the speed setting | `PacketWarp` | CONFIRMED (OB-028-B, C, D) |
| Decay class, +1 for an IT launcher, at most 3 | `DecayClass` | CONFIRMED (OB-028-A, H, I) |
| Launch and spend per item, mixed items | `PacketItem` | MEASURED (OB-028, OB-029) |
| No driver or destination: nothing built | `Space.Launch` | CONFIRMED (OB-028-F) |
| Merging, the 32,760 kT cap, the 16,300 kT merge limit | `Space.Launch` | CONFIRMED (OB-028-B, E); cap and limit BINARY-ONLY |
| Flight: ⌊W²/2⌋ on the launch year, then W² | `Space.FlyLaunched`, `Space.MovePackets` | CONFIRMED (OB-003 J, K, OB-028 A–I) |
| Decay by class, PP rates, minimum loss, partial years | `Decay`, `Space.DecayPackets` | CONFIRMED (OB-003, OB-023, OB-028) |
| Catch share, minerals added | `CatcherWarp`, impact | CONFIRMED (OB-003, OB-009, OB-022) |
| Damage, kill, defenses lost; AR immune; own packets | impact | CONFIRMED (OB-009, OB-022, OB-030-A, OB-009-E) |
| PP terraforming and starbase design disclosure | none yet | not modelled; `Impact.Unhandled` marks a PP launcher's uncaught impact |

Packets sit in `Space.Packets` in object order (owner, then number) and
count toward the object limit.

### Assumptions (spec gaps)

OBJECTS.md now answers P1–P4 (BINARY-ONLY):
- An IT mixed item spends 48 kT of each mineral.
- New cargo joins any packet of the same owner lying exactly at the
  planet with the same warp, destination and class, whatever its
  minerals. The merge test is Σ⌈m/10⌉ < 1,630 on that packet before the
  add.
  A merged mineral above 32,767 becomes 32,760.
- Packet numbers run 0..510, and 511 is used only when nothing sorts after
  the owner's packets. With no number or object slot left, the item is
  built but no packet appears (`Launch.NoRoom`).
- Decay is integer: `min(m, max(floor, ⌊m·r·p/10000⌋))`, with arrival
  `p = round(trunc(d)·100/move)`, halved on the launch year. A packet
  with nothing left is removed.

Still open:

1. **P5** A player's packets and salvage share one 0..511 number pool,
   and only higher players' packets and salvage, wormholes and the Trader
   block 511. Elegy's salvage has no owner or number yet, so it takes no
   packet number and blocks nothing.

### What the turn engine needs to call (packets)

- Production (orders layer): a packet item calls `Space.Launch` with the
  planet's packet destination and speed (the engine's `Planet` has
  neither yet), spends `Launch.Spend` from the surface, and sends the
  no-driver message for `Launch.NoDriver`.
- Movement step 2 (objects move): `Space.MovePackets` with an
  `ImpactContext` whose `DefenseShare` is the planet's normal-bomb
  defense share (TAKEOVER.md); then the waypoint check.
- Step 3a: `Space.DecayPackets` with salvage decay.
- Step 5: `Space.FlyLaunched`, before battles and bombing.
- For every `Impact` with `Emptied`, empty the planet as after bombing.
- Packet visibility belongs to scanning.

## Stargates

| Rule | Function | Status |
|---|---|---|
| What makes a gate; limits from the component table | `StarbaseGate`, `PlanetGate` | CONFIRMED (GT-004) |
| Source and destination gates, one-sided friendship, Jump Gates | `Jump` | CONFIRMED (GT-001 A–E, M) |
| Refusal order | `Jump`, `GateRefusal` | CONFIRMED (GT-001 F3, GT-003 R1–R6); the destination planet check BINARY-ONLY in its place |
| Cargo unloaded before the checks; foreign colonists | `Jump` | LEGACY BUG MEASURED (OB-021); CONFIRMED (GT-001 F, F2) |
| Range and mass refusals at 5× | `Jump` | CONFIRMED (GT-001) |
| Danger | `GateDanger` | CONFIRMED (GT-001 N1–N6) |
| Losses, damage; IT destroys no ships | `Jump` | CONFIRMED (OB-021, OB-022) |
| Mixed-fleet count | `LegacyGateMixedFleetLoss` (on) | LEGACY BUG, MEASURED (GT-001 H2, GT-002), CONFIRMED (GT-003 W0–W5) |

### Assumptions (spec gaps)

OBJECTS.md now answers G1–G3 (BINARY-ONLY; fits OB-021, GT-001):
- Designs go in design-number order, with every pct worked out first.
- A design at 0% makes no draws; one at 100% is removed.
- Each ship draws its loss roll, and a destroyed ship draws `rand(500) < u`
  while damaged ships remain.
- Old damage counts only the `D` damaged ships, and `D` more ships go when
  `Nw + Dm ≥ A`.
- Lost ships take `⌊fuel·L/T⌋` of the fuel, and the same share of any cargo
  still aboard.

Still open:

1. **G4** When the extra destruction would take more ships than are left
   (`D > s`, which corrupts the original's record), every ship of the
   design is destroyed. A design's ships are taken as one stack with its
   first stack's damage.

### What the turn engine needs to call (stargates)

- Fleet movement: a fleet whose next waypoint has warp `GateWarp` (11)
  calls `Jump` instead of moving. On `GateOK` the fleet is at the
  destination: consume the waypoint, mark it moved, give it no heading
  or warp for others' scans, skip its repair this year, never apply
  Cheap Engines failure, and make other players' chasers stop at its
  departure point (the owner's own follow it).
- `FleetLost`: delete the fleet with the "fleet lost" message.
- A refusal: one message to the owner for `Refused`; the fleet keeps its
  waypoints and counts as stationary. An unload (`Unloaded`) also
  messages the source planet's owner.
- Orders: accept waypoint warp 11 (now refused as not modelled), and let
  routing pick it when both ends are gated and the jump is safe.
