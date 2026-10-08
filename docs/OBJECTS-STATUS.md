# Space objects: implementation status

`objects/` implements the space objects of stars-elegy `docs/OBJECTS.md`
(as of stars-elegy `main` at `0779d2f`) and its Mystery Trader appearance
in `KERNEL.md`, with the part statistics of
`COMPONENTS.md` (the component table the engine embeds). Nothing here
comes from the private archaeology repositories.

The package reads the engine's exported state and exposes step and query
functions. The turn engine owns the year's order (OBJECTS.md "Turn
placement") and calls them through `engine.SpaceObjects`, which
`objects/engine_adapter.go` implements (elegy #27; see "Turn wiring"
below and KERNEL-STATUS.md "Space objects").

## Minefields

| Rule (OBJECTS.md) | Function | Status |
|---|---|---|
| Size and inside test | `Minefield.Contains` | CONFIRMED (OB-002, OB-018-C) |
| Decay, incl. SD and detonation | `DecayLoss`, `Space.Decay` | CONFIRMED (OB-002, OB-014-A, OB-015, OB-016, MF-4) |
| Lay amounts, hull multiplier, MCM | `LayAmounts` | CONFIRMED (OB-002, OB-024) |
| Merging, merge cap, laying order | `Space.Lay` | CONFIRMED (OB-002-G, MF-10, MF-12) |
| SD half lay while moving | `Layer.Half` | CONFIRMED (OB-014-C) |
| 512-field limit | `Space.Lay` | MEASURED (MF-11); the 511 case is LEGACY BUG `Legacy.FieldLimit511` (off in the `elegy` ruleset, Elegy's chosen rule) |
| 4050-object limit | `Space.Lay` | BINARY-ONLY |
| Sweep rating and amount | `SweepRating`, `Space.Sweep` | CONFIRMED (OB-001, OB-007, OB-008, OB-010-S); ratings agree with the table's mines-swept column |
| Effective warp | `EffectiveWarp` | CONFIRMED (OB-010, MF-3) |
| Stop checks along a step | `CheckStep` | CONFIRMED as rates (MF-1, MF-5, MF-6); safe-warp bonuses for SS and SD BINARY-ONLY |
| Path cut: whole-ly foot, entry and exit | `cut` | BINARY-ONLY; east legs CONFIRMED as rates (MF-1, MF-3) |
| Due-north and due-south legs | `Legacy.DueNorthSouthCut` (on in the `elegy` ruleset) | LEGACY BUG, MEASURED (MF-15) |
| Stretches: eight per kind, merging, draw order, stop offset | `addStretch`, `CheckStep` | BINARY-ONLY; stop offsets MEASURED (OB-010-S, MF-15) |
| Stop point | `StopPoint` | BINARY-ONLY; exact on east legs |
| Hit damage | `MineDamage` | CONFIRMED (MF-9, MF-8, OB-010-S, OB-002-M); damage on existing damage BINARY-ONLY |
| Cargo and fuel lost with destroyed ships | `ApplyHit`, `Space.Detonate` | MEASURED (MF-14); detonation BINARY-ONLY |
| Survivors' minerals dropped as salvage | `Legacy.MineSurvivorSalvage` (on in the `elegy` ruleset) | LEGACY BUG candidate, MEASURED (MF-14) |
| Mines lost, paying field | `MinesLost`, `PayingField`, `ApplyHit` | CONFIRMED (OB-010-S, OB-024, MF-4) |
| Salvage at the stop point, none at a planet; all minerals when the whole fleet dies; empty fleet `rand(10)` | `ApplyHit` | MEASURED (OB-024, MF-14); the whole-fleet case BINARY-ONLY; the empty-fleet drop is a LEGACY BUG candidate, `Legacy.EmptyFleetSalvage` (on in the `elegy` ruleset) |
| SD disclosure on hits and detonations | `Hit.Disclosed`, `Detonation.Disclosed` | CONFIRMED for detonations (MF-7) |
| Detonate setting: owner only, Space Demolition only, standard fields only | `Space.SetDetonate` | BINARY-ONLY; OBJECTS.md's chosen rule (the original accepts any field, LEGACY BUG). `engine.DetonateOrder` applies it through `SpaceObjects.SetDetonate` and is rejected with its error (`TestDetonateOrderThroughEngine`) |
| Detonation | `Space.Detonate` | CONFIRMED (OB-002-M, MF-7, MF-8); order and marking of several fields BINARY-ONLY |

Tests follow the engine's naming: `TestConfirmed*` use the OB and MF
vectors OBJECTS.md and PARITY.md give; `TestPrediction*` cover the rest.
The MF runs' "Tank" is a Destroyer with two Superlatanium (3200 armor).

### Assumptions (spec gaps)

OBJECTS.md now answers O1–O8 ("Laying", "Limits", "Hits on moving
fleets", "Arithmetic details", "Detonation"; MF-14 MEASURED, MF-15
MEASURED, the rest BINARY-ONLY):
- An SD half lay halves each kind's amount; amounts are multiples of 10,
  so it is exact.
- Equally near own fields: the first in object order (lower number).
- A new field takes the lowest number unused across all three kinds.
- The path cut uses a whole-ly foot of the perpendicular; due-north and
  due-south legs are a LEGACY BUG behind `Legacy.DueNorthSouthCut`.
- Stretches: at most eight per kind, merged when overlapping, touching or
  ending 1 ly before another; equal entries go standard, heavy, speed
  bump; draws `k = 0..b − a − 1`, the stop `a + k` ly from the start.
- Damage on existing damage counts `⌊pct·n/100⌋` damaged ships and
  stores `max(1, ⌊avg·500/A⌋)` units.
- Destroyed ships' cargo and fuel shares are lost, and the survivors'
  minerals become salvage behind `Legacy.MineSurvivorSalvage`.
- Detonating fields go in object order, and the first containing field
  marks the fleet.

Still open:

1. **O6** A design's ships are the fleet's stacks of that design, with
   the first stack's stored damage; exempt layers are left out of the
   fleet minimum.
2. **O9** A new stretch merges into the first stretch it may join, in
   entry order, and the result is not merged again.
3. **O10** The empty-fleet `rand(10)` draws happen only where salvage can
   form (not at a planet) and only when ships were destroyed.

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
| 25th redraw finds an unowned part but gives a ship | `Legacy.TraderLastRedraw` (on in the `elegy` ruleset) | LEGACY BUG, BINARY-ONLY |
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
| PP terraforming | impact → `terraform.PacketTerraform`, `Impact.Terraform` | axes CONFIRMED (OB-029-T1..T3); draws BINARY-ONLY; see TERRAFORM-STATUS.md |
| PP starbase design disclosure | `Impact.DiscloseDesign` | BINARY-ONLY; the turn engine records the design (`EventPacketDesignSeen`) |

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

1. **P5** is answered (OBJECTS.md "Numbering", "Salvage", BINARY-ONLY): a
   player's packets and salvage share one number pool (`packetNumber`),
   and only higher players' packets and salvage, wormholes and the
   Trader block 511.
2. **P6** OBJECTS.md "The settings" leaves a destination that names no
   planet undefined and asks Elegy for a rule: Elegy treats it as no
   destination (`Launch.NoDriver`).


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
| Mixed-fleet count | `Legacy.GateMixedFleetLoss` (on in the `elegy` ruleset) | LEGACY BUG, MEASURED (GT-001 H2, GT-002), CONFIRMED (GT-003 W0–W5) |

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


## Salvage

Salvage objects live in `engine.Game.Salvage`; the engine creates battle
and scrap salvage. `engine.Salvage` carries `Owner`, `Number`, `Fresh`
and `Steps` for these rules (OBJECTS.md "Salvage", BINARY-ONLY except
where marked).

| Rule | Function | Status |
|---|---|---|
| Owner's packet number pool, 511 rule, object limit | `Space.SalvageNumber` | BINARY-ONLY |
| New object, the 30,000 kT overflow | `Space.NewSalvage` | BINARY-ONLY; overflow CONFIRMED (CB-040) |
| Mine-hit salvage joins the first object at the stop point | `Space.AddMineSalvage` | BINARY-ONLY |
| Decay: skip once after a make or add; `max(10, ⌊m/10⌋)` | `DecaySalvage` | BINARY-ONLY; fits T-34 |
| Loading capped by contents; unloading by the stored size | `SalvageLoad`, `SalvageRoom` | BINARY-ONLY |
| Seen as a packet; PP sees all; owner made known | `Space.Scan` | BINARY-ONLY |

### Assumptions (spec gaps)

1. **S1** With no free number or object slot, no salvage object is made
   and its minerals are lost (LIMITS.md lists this as UNKNOWN).

## Visibility

Who sees which space object comes from stars-elegy `docs/SCANNING.md`,
"Space objects" (as of stars-elegy `main` at `0779d2f`). The turn engine
keeps the fleet and planet sightings; these functions cover the objects.

| Rule (SCANNING.md) | Function | Status |
|---|---|---|
| Minefields: own; `P`, `R/4`; inside (fleets only); known within `R` | `Space.Scan` | CONFIRMED (OB-018, OB-018-C, E–G); fleets-only inside BINARY-ONLY |
| Minefield knowledge from sight, sweeps and hits | `Space.Scan`, `Space.Sweep`, `Space.LearnHit`, `Space.Detonate` | sight CONFIRMED; sweeps and hits BINARY-ONLY |
| Ownership does not make a field known; own fields still listed in the owner's view | `Space.Scan` | MEASURED (MF-13a, MF-13c; stars-elegy #108); with `SeeObjects` calling `Scan` and the harness comparing `known_to`, both cases pass locally |
| Wormhole ends: within `R` only; known, `P` or `R/4`; known once seen | `Space.Scan` | CONFIRMED (OB-011-H, OB-017, OB-018 H, I, WT batch); the known band BINARY-ONLY |
| Packets within `R`; PP sees all | `Space.Scan` | CONFIRMED (OB-018 J, K, OB-012) |
| Mystery Trader seen by all | `Space.Scan` | CONFIRMED (OB-011-J) |
| Owners made known by minefields and packets | `Sightings.Owners` | CONFIRMED (OB-011, OB-017, OB-018) |
| PP packet scanners, `R = P = warp²` | `Space.PacketScanners` | CONFIRMED (OB-012) |
| SD minefields see fleets inside, not orbiting | `Space.DemolitionSightings` | CONFIRMED (OB-014-B); the cloak draw BINARY-ONLY |
| What a sighting shows: the whole object, whoever owns it; only the viewer's own "known" entry; seen this year only | `Space.Report`, `ObjectReport` | MEASURED (SC-038; SCANNING.md "Space objects" under "Disclosure"); reported as `game.Report.Objects` |

### Assumptions (spec gaps)

1. **V1** Every packet a PP player owns at the end of the year scans,
   including one launched this year.
2. **V2** Overlapping fields of one kind are one stretch in `CheckStep`,
   so a hit teaches every field of that kind that contains the stop point
   and could stop the fleet.
3. **V3** For SD fields, "enemy" means any other player's fleet, as
   elsewhere in SCANNING.md. A fleet at a planet's position is in orbit.
   Fleets are taken in fleet order, and each makes at most one cloak draw
   however many fields it is inside.
4. **V4** A player's own packets are always in its report
   (`Space.Scan`; own minefields are, MEASURED MF-13). SC-038 saw own
   packets beyond every scanner missing from their owner's file once
   (OB-017 D–F); one observation, not adopted. A packet's "decay state
   and whether it has moved" (SC-038) are Elegy's decay class and its
   launched-this-year mark. Elegy keeps no per-year "seen" marker, so
   none is reported.
5. **V5** A detonation that damaged at least one of a fleet's ships is a
   hit, so the fleet's owner learns the field (`Space.Detonate`); a
   fleet the field marked without damage learns nothing.

Not covered yet in objects: dropping or keeping waypoints on objects no
longer seen (SCANNING.md "Orders that depend on sight"), which belongs to
the turn engine's waypoint handling.

## Turn wiring

`engine.GenerateTurn` reaches these rules through `engine.SpaceObjects`
(`Game.Objects`), implemented by `objects/engine_adapter.go`, which the
kernel owns (elegy #27). `newgame.Generate` sets `Game.Objects` to an
`objects.Space` holding the new game's wormholes.

| Turn step (KERNEL.md "Turn order") | Adapter method | Rules called |
|---|---|---|
| 3.2 Traders, then packets in flight | `MoveObjects` | `Space.MoveTraders`, `Space.MovePackets` |
| 3.3 each movement step | `MineCheck`, `MineHit` | `CheckStep`, `ApplyHit` |
| 3.3 warp-11 waypoint | `Stargate` | `Jump` |
| 3.3 reached wormhole end | `TransitWormhole` | `Space.Transit` |
| 3a | `DecayObjects` | `Space.DecayPackets`, `Space.Detonate`, `Space.Decay` |
| 4 production, a packet item | `LaunchPacket` | `Space.Launch` |
| 4c | `TraderAppears` | `Space.Appear` |
| 5.1 | `MoveObjectsAgain` | `Space.FlyLaunched`, `Space.MoveWormholes` |
| 6b | `MeetTraders` | `Space.Meet` |
| 6c.2 | `LayMines` | `Space.Lay` |
| 7.1 | `SweepMines` | `Space.Sweep` (the sweeper's owner learns the field) |
| before the views | `SeeObjects` | `Space.Scan` per player, `PacketScanners`, `DemolitionSightings` |

Also wired: minefield knowledge (`MineHit`
calls `LearnHit`; `SeeObjects` calls `Space.Scan` per player), fleet and
player sight from PP packets and SD minefields, the salvage call sites
(`AddMineSalvage`, `DecaySalvage`) and PP starbase design disclosure
(`objects/engine_adapter.go`; KERNEL-STATUS.md "Space objects"). Tests:
the OB-011..OB-018 parity cases pass through the whole turn;
`TestPredictionMinefieldKnowledge`, `TestConfirmedPacketScanners`,
`TestConfirmedDemolitionSight`, `TestPredictionSalvageDecay`,
`TestPredictionMineSalvage` and `TestPredictionPacketDesignSeen` cover
the rules each call uses.

Not wired yet:

- **Salvage loads and unloads:** `SalvageLoad` and `SalvageRoom` wait
  for the engine's load and unload-into-salvage tasks, which are not
  modelled.
- **Computer players' planet trades:** the adapter passes
  `engine.Player.Level` to `Space.Meet`, but nothing sets the level yet,
  so every computer player counts as Easy and no planet trades. A new
  game will set it from its computer players' levels.
