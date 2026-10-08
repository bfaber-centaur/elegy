# Peaceful kernel: implementation status

The J-RC3 peaceful turn and ordinary fleet movement are implemented in
`engine/` from the public specification in `bfaber-centaur/stars-elegy`:
`docs/KERNEL.md`, `docs/PARITY.md` (including KX-001 to KX-004) and the public FM-001..004
movement corpus (`experiments/fm00N`), as of stars-elegy `main` at `63635f0` (which includes KX-001 to KX-005, the
OT runs and the turn order and random draw order). The parity vectors
are copied from stars-elegy `main` at `58a9b58` (stars-elegy #111, #113, #116).
Nothing here comes from the private archaeology repositories.

## End-to-end parity milestone

The milestone is `GenerateTurn` running the players' orders, waypoint
tasks, production and the space objects in KERNEL.md's turn order, with
the GT, MF, OB, WT and WU vectors running through it in
`TestParityVectors`, and every failure listed below under "Known parity
failures" with its real reason. In the baseline at main e5e613c: GT 4
pass; MF 23 pass, 2 random; OB 145 pass, 1 random, 4 sample-only; WT 10
pass, 18 sample-only; WU 31 pass. Packet items build and launch through
`SpaceObjects.LaunchPacket`; the harness maps them (items 6, 14–17)
since the vectors carry each planet's packet destination and warp
(stars-elegy 4247afc).

## Tests: ground truth versus predictions

KERNEL.md gives every rule a status. Test names follow it:

- `TestConfirmed*`: CONFIRMED vectors and oracle observations. These are
  ground truth; never change one to make code pass. `go test -run
  Confirmed ./...` runs only these.
- `TestPrediction*`: BINARY-ONLY vectors, worked from the rule with no
  oracle observation. They pin the code to the spec; an oracle result may
  flip one, and then the test (and code) change with the spec.

| Area | Code | Ground truth | Predictions |
|---|---|---|---|
| Habitability | `habitability.go` | center → 100; KX-002 H1–H6 | 70, 50, 50 → 79 |
| Maximum population | `population.go` | hab 100 → 10,000; KX-002 hab 92, 58, 41, 3, hostile; HE, JOAT, OBRM (P1–P3) | JOAT+OBRM, AR |
| Population growth and carry | `population.go` | PG-001..003, 36 years of P and carry; KX-002 H3, H4, P1–P3, G1–G3 (overcrowding factor 4), hostile deaths, HE ×2 | 3 KERNEL vectors (uncrowded, quantization, g < 1000), empty planets |
| Resources, caps | `economy.go` | PG resources, PQ operable caps, AR resources (floating E/R0) and mines (KX-001 Z2, Z3); KX-002 C1–C3 maximums, E above max (G1) | floor of 10, other defense branches, the 2·max limit of E |
| Mining | `mining.go` | PG002 concentration and fraction 2407–2411; KX-002 N2 (1000 mines, homeworld floor, clamp re-evaluated) | random +1 draw, draw order |
| Research | `research.go` | PG003 energy 2408–2436, level cost; KX-002 R1, R2 (cost settings), R5 (GR split), R6 (maxed field); slower tech's half-scale storage (KX-003 S2 vectors, parity KX-003-r2) | several levels, next field, same-year switch, explicit/lowest persistence |
| Production queue | `production.go` | PQ-001 C01–C14 (15 cases); KX-001 A1–A4 (Auto Alchemy before a ×n item), M1–M4 (item costs) | empty queue (no tax), zero resources |
| Movement and fuel | `movement.go` | all 224 fleets of FM-001..004 (position, fuel, waypoints, warp, orbit, events), KERNEL fuel/range/chase vectors | no free warp, more than one engine per ship, cargo ties, chaser fuel per round (R, running dry, top-up, ram scoop) |
| Starbase refuelling | `movement.go` | FM-004 DK | |
| AR colonists in flight | `movement.go` `arColonistLoss` | TK-117, TK-107 (stars-elegy #44) | |
| Under-engined designs: f = 99999 and the 32-bit fuel-term wrap (LEGACY BUG, ruleset switch `Legacy.FuelWrap`) | `movement.go` `engineFactor`, `fuelTermWrap` | FM-105: 200, 50, 500 mg → 7, 1, 19 ly, 0 mg | the float form does not wrap |
| Random events: comet strike (sizes, kills, minerals, environment, queue cut), climate change, new minerals, option off | `randomevents.go` | KX-004 vectors (S2, S3, S5, E0), replayed in KERNEL.md's draw order; comet message axes LEGACY BUG behind `Legacy.CometAxes` | AR owner struck, the 180 cap, the probabilities |
| Score terms, ship classes, rank, flag word | `scores.go` | KX-003 S1 terms (planets, tech, ships, resources), Omega/Cherry class boundaries, flags 0x0ae0 / 0x0021 | capacitors, sappers, speed adjustment; score, resources and highest-score flags |
| Deciding the game, public scores | `scores.go` `decide`, `visibleScores` | public scores from year index 20 (KX-004 E0, E1) | deaths, survivor, winners after the minimum years; decided game and dead players' records |
| Improved Fuel Efficiency factor; fuel generators and fuel transports; Radiating Hydro-Ram Scoop colonist losses; refuelling at a friend's docked starbase | `movement.go` `fleetFactor`, `generateFuel`, `radiatingColonists`, `refuelFleets` | parity vectors FM-101..103 (`TestParityVectors`); KB-4A E, F1–F5, G, H, X and CS-003-W as cited in KERNEL.md "Other movement rules" | the RHRS immune and ≥ 170 exemptions |
| Claim Adjuster year-end step: original-value drift, then every axis to its limit with the reach just researched; terraform reach per axis | `terraform.go` `claimAdjusterYearEnd`, `terraformReach` | KX-003 S3/S3L; capture examples for TK-108, TK-118..121 (KERNEL.md "Terraforming"); parity vectors TK-108-A, TK-118-A, TK-119-A now match | the drift's draw sequence, CONFIRMED by KX-005 replays but tested here only with scripted draws |
| Fuel cannot be unloaded onto a planet | `takeover.go` (no fuel action) | FM-101..105 | |
| Whole turn | `turn.go` | PG homeworld 2407 → 2436 through `GenerateTurn`; PQ C01 and C14 over two years; KX-001 Z2, Z3 | |
| AR without a starbase | `turn.go` | | Elegy decision, below (`TestElegyDecision*`) |

## Turn order implemented

`GenerateTurn` follows KERNEL.md "Turn order" for the steps Elegy
models: the players' orders (`YearOrders`: the player shuffle, each
file in that order, then the gift credits; ORDERS-LAYER-STATUS.md); the
waypoint check (step 1a.3), which first retargets waypoints aimed at a
fleet merged away during the replay (ORDERS-STATUS.md); the before-movement waypoint tasks (unloads,
colonize, route, drops, the research level-up check, loads and merges);
fleet movement (ordinary fleets in fleet order, owner then fleet number,
then fleet chasers in rounds, then waypoint settlement, where reached
waypoints are dropped or, with repeat orders, rotated), Radiating Hydro-Ram Scoop losses and fuel generation;
mining for every planet in id order (population before growth); per
planet in id order: resources, research tax, production queue (caps use
the grown population); population growth for every planet; research
level-ups; random events (when `Game.RandomEvents` is on); starbase
refuelling; battles, bombing and the after-movement waypoint tasks
(unloads, colonize, route, remote mining, drops, the research check,
merges and fleet transfers; COMBAT-STATUS.md, TAKEOVER-STATUS.md);
repair; the Claim Adjuster year-end step; the Orbital Adjusters; the
end-of-year waypoint check (step 7a.2); year
+ 1; scores, victory flags and deciding the game
(`TurnResult.Scores`, each view's visible records). Its random draws
follow KERNEL.md "Random draws" for the same steps.

Waypoint upkeep (`waypoints.go`, ORDERS.md "Waypoint upkeep and the
remaining tasks"): reaching a waypoint with and without repeat orders,
the two fallbacks and patrol waypoints never repeating; the waypoint
check (a waypoint aimed at a fleet takes its position, and one aimed at
a fleet that is gone becomes a plain go-to); the route task; the
transfer-fleet task with its refusals. A transport task whose unloads
have all run becomes no task (MEASURED: TK-501 fleet 4, WP-1-explore).
The orders layer does not accept the patrol and transfer-fleet tasks or
the repeat flag in an order yet; `validTask` in `orders.go` belongs to
that lane.

Waypoint 0 keeps its task while the fleet is in transit (ORDERS.md Q2,
MEASURED WU-ROUTE). When the year's files are written, a waypoint aimed
at another player's fleet becomes a plain position (ORDERS.md Q1,
BINARY-ONLY; WU-A fleet 13 shows the target's current position as
space, and the vector agrees). At the end of the turn, a patrol fleet
with no waypoint takes the nearest enemy fleet it sees within 50 ly,
ties by fleet order, at warp min(10, range ÷ 5), and the new waypoint
carries the patrol task (ORDERS.md Q3, BINARY-ONLY; vectors WU-SW50 and
WU-RNG20). A computer player never accepts a gifted fleet (ORDERS.md,
CONFIRMED WU-AICOMP3/4; `Player.Computer`).

- **W5 (ASSUMPTION).** The enemy-target rewrite is applied to the game
  state itself, not only to the written file, so next year's waypoint
  check sees a plain position.
- **P1 (ASSUMPTION).** "Enemy" for patrol is the player relation enemy;
  neutral fleets are not acquired.

### Space objects

Minefields, wormholes, the Mystery Trader, mineral packets and
stargates are rules in package `objects` (OBJECTS-STATUS.md). The
engine reaches them through the `SpaceObjects` interface
(`engine/objects.go`), held in `Game.Objects` and implemented by
`objects.Space` (`objects/engine_adapter.go`). A nil `Game.Objects` is a
galaxy with no objects; its steps do nothing and draw nothing, so every
earlier result is unchanged. `GenerateTurn` calls the objects where
KERNEL.md "Turn order" puts them:

- 3.2: the Traders move, then packets in flight; then the waypoint check.
- 3.3: each fleet's movement step is checked for a minefield stop and
  the hit applied; a warp-11 waypoint is a stargate jump instead of a
  move; a reached waypoint on a wormhole end takes the fleet through.
- 3a: packet decay, detonations, minefield decay.
- 4c: after the random events, the Trader's appearance.
- 5.1: packets launched this year fly half a year, the wormholes move,
  then the waypoint check (step 5.2 refuelling follows).
- 6b: Trader meetings, after battles and bombing.
- 6c.2: mine laying, with the after-movement unloads.
- 7.1: mine sweeping, before repair. A fleet that jumped through a
  stargate is not repaired that year.
- Before the views: each player's scanners see the space objects
  (SCANNING.md "Space objects"): seen minefields and wormhole ends
  become known, a fleet hit by a minefield learns it, a Packet Physics
  player's packets in flight scan fleets and planets, a Space
  Demolition player's minefields see the fleets inside them, and the
  owners of the minefields, packets and salvage seen become known
  players. `PlayerView.Objects` lists the objects seen.

Salvage (OBJECTS.md "Salvage") lives in `Game.Salvage`. A battle's
deep-space salvage object belongs to the owner of its first addition
and is marked fresh; an overflow object belongs to the owner of the
addition that overflowed and is not marked; each new object takes its
number from `SpaceObjects.SalvageNumber`. A mine hit's salvage joins the
first object at the stop point (`objects.Space.AddMineSalvage`), and
salvage decays at step 3a before packets. A fleet stopped by a
minefield lands at OBJECTS.md's stop point (`objects.StopPoint`).

The race check (RACES.md "In a running game") runs at KERNEL.md step
2a, after the before-movement tasks and before movement, through
`Game.Races` (`races.GameRaces`); a nil `Game.Races` skips it. A
checker that keeps state implements `RaceCloner`, so `GenerateTurn`
leaves the input game's checker unchanged.

Remote mining and the Orbital Adjusters run through `Game.Terraform`
(`terraform.Rules`); a nil `Game.Terraform` skips them. A fleet with
the remote-mining task (`TaskRemoteMine`) that did not move this year
mines in its place in fleet order at step 6c.2 (KERNEL.md "Remote
mining", CONFIRMED T-35, KB-1B); a fleet built this year counts as moved
(CONFIRMED SL-03). An Alternate Reality player's new fleet that can mine
gets the task at its build planet (PRODUCTION-LAUNCH.md "Default task",
CONFIRMED SL-11). The Orbital Adjusters run at step 7.4, after the Claim
Adjuster step (CONFIRMED OT-4); a change of the planet owner's value
tells the fleet owner, and the planet owner when it is someone else
(MESSAGES.md 0x12c, 0x15a). A Packet Physics player whose packet a
starbase caught learns that starbase's design in full that year
(`EventPacketDesignSeen`; OBJECTS.md "Impact" step 4, SCANNING.md
"Designs", BINARY-ONLY).

An Interstellar Traveler gets a normal report of every planet whose
starbase has a stargate within range of one of its own planets' gates
(SCANNING.md, CONFIRMED OB-013).

The parity harness loads `initial_state.objects` and checks the
minefield, wormhole, trader, packet, object and object_gone
expectations through a hook in `engine/objects_parity_test.go`
(package `engine_test`, because `objects` imports the engine).

Choices where OBJECTS.md is silent:

- **O9 (ASSUMPTION).** A fleet that arrived this year onto a "lay
  mines" waypoint moved, so only a Space Demolition fleet lays that
  year, and only half.
- **O10 (ASSUMPTION).** A Space Demolition half lay does not count
  toward the task's duration.
- **O11 (ASSUMPTION).** A fleet stopped by a minefield is charged the
  fuel for the whole step and gets no top-up or ram-scoop fuel for it.
- **O12 (ASSUMPTION).** A waypoint aimed at a wormhole end its owner
  does not know follows the end only while known; one that moved unseen
  becomes a plain position where the end was.
- **O13 (ASSUMPTION).** The space objects are seen, with the Space
  Demolition cloak draws, before the population estimates draw.
- **O14 (ASSUMPTION).** In a game with no space objects (nil
  `Game.Objects`), a new salvage object takes its owner's lowest unused
  salvage number, with no object limit, and salvage does not decay.
- **O15 (ASSUMPTION).** In step 6c.2 each fleet runs one task, so a
  fleet that colonizes there lays nothing, including a Space Demolition
  fleet whose next waypoint is "lay mines".
- **S5 (ASSUMPTION).** The starbase cloak bound for an IT gate report
  uses the gate's range as P; an unlimited gate shows every starbase.

Not modelled yet: following fleets (step 1a.3; Elegy does not keep
waypoint 0's target), the stargate choice of the route task, waypoint
tasks other than unloads, colonize, merge, route, transfer, patrol, lay
mines, remote mining and scrap (load from and unload into salvage only
by task, ASSUMPTION T5; ORDERS-STATUS.md), terraforming production items (the
orders lane adds them through `Game.Terraform`), the messages for
an Orbital Adjuster or remote miner that changed nothing (MESSAGES.md
0x12d, 0x15b), Super Stealth research stealing, the duplicate-serial penalty,
ships/starbases in the queue, and the
BINARY-ONLY movement rules for IFE, Cheap Engines, warp-10 losses,
Radiating Hydro-Ram colonist deaths and transport tasks.

## Corrected upstream

This implementation surfaced three places where KERNEL.md disagreed with
the public corpus. stars-elegy PR #13 (merged) corrects them, and the code and
`TestConfirmed*` tests follow the corrected text:

- **Caps by order kind.** Auto Mines/Factories/Defenses build at most
  `operable − installed`; plain orders are cut to `max(maximum, operable)
  − installed` (PQ C04, C09, C10, C13, C14).
- **Running dry.** Fuel 0, limited by `R` or paid a non-zero cost, could
  not afford the whole leg, and short of the destination (or `R = 0`),
  even when this year's move was paid in full (FM-002 24, 29).
- **Waypoint settlement.** After all movement, waypoints aimed at a fleet
  take its end position and every fleet sitting exactly on its next
  waypoint completes it, so both sides of a mutual chase complete
  (FM-001 71/72, FM-002 39/40, FM-003 21/22, 29/30, ...).

## Answered upstream

stars-elegy PR #14 answered these questions as BINARY-ONLY rules; the code
follows them and `kernel_pr14_test.go` pins each as a `TestPrediction*`:

- **Chasers and fuel** (chase rule 6): each round applies the ordinary fuel
  rules with `R` reduced by the distance already moved, running dry stops
  the chaser for the year, top-up and ram scoop apply per round. Changed.
- **Research switching**: the new field is checked the same year, an
  explicit next field resets to "same field" after use, "lowest field"
  stays, and only the current field switches (also with Generalized
  Research). Changed. With Generalized Research the code checks every
  field before any switch; KERNEL.md does not give that order.
- **Depletion clamp**: re-evaluated every repetition. Already so.
- **Empty queue**: no research tax either. Already so.
- **Empty planets**: no growth or deaths on 0 population. Already so.
- **Cargo ties**: the fleet's own design order. Already so.
- **Mining draws**: every planet mined before any production, planets in id
  order, ironium, boranium, germanium. Mining is now its own pass; the
  draw sequence is the same as before.

stars-elegy PR #16 (KX-001) then settled the three that remained:

- **Auto Alchemy before a ×n item**: one unit at a time; a mineral-short
  unit takes its partial, then alchemy buys its shortfall (an auto item
  skips the partial); if alchemy runs short, the unit keeps its
  percentage, the rest becomes a Mineral Alchemy partial at the front and
  the queue stops. Changed (`kernel_kx001_test.go`, CONFIRMED).
- **Cost modifiers**: factory and mine resource costs from the race,
  factories 3 kT germanium with "factories cost 1 kT less germanium",
  Inner Strength defenses 9 + 3/3/3, alchemy 25 with Mineral Alchemy
  (`ItemCost`). Changed. Terraforming costs are not modelled (no
  terraforming yet). The game's degrading of an over-budget race is not
  modelled; KX-001 M3 is tested from the degraded race.
- **Zero maximum population**: see the Elegy decision below.

KX-001 M4 also showed that the partial percentage is PARITY.md's formula
`max(trunc((a+1)·100/c) − 1, trunc(a·100/c))`, which is one less than "the
largest p with trunc(c·p/100) ≤ a" whenever `c` does not divide
`(a+1)·100` (9 resources: 54%, not 55%). The code now uses the formula;
PQ-001's costs (4, 5, 10, 100) never separated the two. stars-elegy #18
states the formula as the rule.

stars-elegy #18 also settled the limiting component (code follows it;
`limiting` in `production.go`):

- Components are compared Fe, Bo, Ge, then resources; one replaces the
  lowest only if strictly lower, so the first of tied minerals wins and a
  mineral wins a tie with resources (BINARY-ONLY).
- An auto item with any mineral short is mineral-blocked even when
  resources give the lower percentage: skipped without a prefix (KX-001
  A7), straight to alchemy for the lowest component's shortfall with one
  (A6). Changed; A6 failed before.
- A prefix stays in front of an auto item after it builds (A5). Already so.

KX-002 (stars-elegy PR #35) confirmed many of the BINARY-ONLY rules above;
their vectors are now `TestConfirmed*` tests in `kernel_kx002_test.go`. It
also corrected one: overcrowding deaths are `g = 4·max(−300, trunc(c/−10)
+ 99)`, not `2·…` (G1: 12,000 → 11,899 carry 20). Changed. KX-002's
research switching cases (R3, R4) and several levels a year (R2, G1) agree
with the rules already pinned by `TestPrediction*` tests; KERNEL.md does
not give their full starting state, so they stay predictions here.
Terraforming (KX-002 T1–T3) is not modelled.

## Elegy decisions

Places where the original has no behavior to copy, and Elegy chose one.

- **Alternate Reality planet with population, habitability ≥ 0 and no
  starbase** (maximum population 0). The original stops turn generation
  with an integer divide by zero (KERNEL.md, CONFIRMED KX-001 Z1, LEGACY
  BUG). Elegy's ruleset switch `Legacy.ZeroMaxPopulationStop` decides:
  on (`jrc3-faithful`, `elegy` v1), `GenerateTurn` returns a
  `*ZeroMaxPopulationError` naming the planet before changing anything;
  off (`elegy` v2 and later, the default; INTENTIONALLY DIFFERENT, Bobby's "Keep
  going"), the year is generated and the planet's growth treats it as
  maximally overcrowded (`crowdingPermille`): more than 10 units lose 12%
  a year, 10 or fewer stay. See RULESET.md. A hostile planet in the same
  state generates normally (hostile deaths, 1 resource; KX-001 Z3).

Waypoint tasks (`waypoints.go`), where ORDERS.md is silent:

- **W1 (ASSUMPTION).** When several transfer-fleet refusals apply, the
  first in ORDERS.md's order is reported: recipient absent, recipient
  treats the giver as an enemy, colonists aboard, no design slot or fleet
  room.
- **W2 (ASSUMPTION).** A recipient's "matching" design slot is one holding
  the same `Game.Designs` entry. A copy never is, so in practice every
  design of a gift takes a free slot (CONFIRMED that a differing existing
  design is not matched).
- **W3 (ASSUMPTION).** A gifted fleet has no name and repeat orders off.
- **W4 (ASSUMPTION).** The route task runs only for a fleet with no
  further waypoint, so a fleet already routed is not routed again before
  it leaves.
- **W7 (ASSUMPTION).** When a merge order removes a fleet that waypoints
  name, they are retargeted to the fleet it joined. ORDERS.md "Targets
  that moved, died or were captured" prefers the replaying player's own
  fleets at that spot and does not pin which one.
- **W8 (ASSUMPTION).** When a chase order names a fleet merged away
  earlier in the replay, the waypoint check after the orders picks among
  the owner's fleets at the order's coordinates uniformly. Each candidate
  draws as ORDERS.md says, but its battle-plan preference and weight are
  not modelled, because ORDERS.md defines neither. The research lane may
  pin them.
- **Known issue: a stale chase id can be reused.** `Fleet.ID` is Elegy's
  internal key, and a new fleet takes the highest id plus one
  (`Game.newFleetID`). If the merged-away fleet held the highest id, a
  fleet split later in the same replay takes that id. A chase order naming
  the merged fleet then resolves to the split fleet and skips W8's pick.
  The fleet number, which the specs count, still follows
  PRODUCTION-LAUNCH.md "The new fleet" (lowest unused, CONFIRMED SL-02).

## Year invariants

`engine.CheckYear(prev, result)` checks one generated year's structural
invariants (engine/invariants.go): ids, owners and indexes; planets
unmoved, and an unowned planet without population, defenses, starbase or
queue; fleets with ships, unique numbers, and fuel and cargo within their
tanks and holds; at most 512 fleets a player; research levels in 0..26
that never fall; and each player's view holding exactly their own planets
at the own-planet level, with every report and sighting naming a real
object at its real position. The doc comment gives each check its
stars-elegy citation or its label (ELEGY CHOICE for the data-model checks,
with what each protects; ASSUMPTION for levels never falling). An owned
planet with 0 colonists is allowed: a load after movement can take every
colonist, and the planet is lost only at the next year's growth
(TAKEOVER.md "Unload and load amounts", CONFIRMED TK-201 G). Mass
conservation is not checked: mining, production, scrapping and battles
all change it. `TestCheckYearReports` checks that each kind of corruption
is reported with its own problem, and `TestCheckYearAllowsOwnedEmptyPlanet`
the allowed empty planet. Long multi-seed runs that call it every year
belong to the game loop.

## Known parity failures

`TestParityVectors -v` prints every case that does not pass. These are
the failing cases with a cause found or the note that none was looked
for. A case that is "random" (its result changes with the seed) is not
listed; its comparison is in the test output.

The harness treats an expectation marked `sample: true` (vectors
README.md: one stream's random outcome) as one sample: a match counts
as a check, a mismatch is skipped as "sample", never failed. That made
the battle cases CB-036, CB-039 and CB-042 to CB-044 pass on what is
deterministic (battle token lists, everything away from the battles),
and every CB case that was "random" now passes in every seed. A
production queue on a planet that changed owner is skipped: it takes
the new owner's default queue, which the vectors do not carry (TK-108-A
and TK-108-C).

- **WU-CAP** passes. A colony whose owner's default queue (after the
  Alternate Reality and Claim Adjuster skips) is empty gets no queue, so
  all its resources go to research; Elegy gave it an empty queue, which
  contributes nothing (KERNEL.md "Production"). MEASURED in WU-CAP only
  (29 resources, message 0x03f).
- A minefield's `radius` is not compared: SCANNING.md defines no
  per-player known radius.

The harness checks `view` expectations (vectors README.md) from each
year's `PlayerView`s: a planet's report level (0 none, 1 position, 3
normal, 4 detailed, PARITY.md "Co-location and orbit reports") and
starbase visibility, a fleet's level (3 seen, 4 with its cargo) and
`known`, and whether a minefield, packet, wormhole end or Trader was
seen that year. Other view fields are skipped. SC028-T75-out, a setup
artifact (PARITY.md), is no longer in the vectors (stars-elegy #116).

## Integration handoff list

The durable list for the next integration owner, as of the merge of
elegy #40. `TestParityVectors -v` prints each case's status and reason.

**Parity failures** (not in the baseline):

- FO-03-E, FO-06-G: "differs" only because `Legacy.MergeOverflow` is off
  by default; they pass with the LEGACY BUG switch on.

**Whole corpora or case groups skipped, and why:**

- rp (race penalty): no longer skipped; RD-P1..RD-P21 pass and are in
  the baseline. Race expectations compare the whole race and the wizard
  settings (leftover spend, stat 15, techs start high) after the year's
  race check. The harness reads stored habitat values as signed bytes
  (RACES.md "Repairs": 251 is −5, a low of 255 the immune marker) and an
  out-of-range PRT as an invalid one, which the check clamps (RD-P7).
- Computer players without orders in the vector: their fleets, planets
  and production queues are not compared ("a computer player's planned
  orders"), since the host plans them each year and the harness runs no
  planner. 19 expectations, each skipped alone inside a case that is
  otherwise checked: RD-P11 (2), RD-P12 (2), RD-P20 (9), RD-P21 (6).
  Their race expectations are compared.

- CB-009 and CB-010 load designs written straight into the oracle's
  state with a Beam Deflector in a shield-or-armor slot; the harness keeps
  such a part. ASSUMPTION K10: TAKEOVER.md "Design parts dropped when the
  year is generated" says parity is "keep and use" for parts beyond the
  owner's tech or race, and extending it to a part in a slot of the wrong
  kind is inferred; CB-010 passing with the part kept supports it.
  CB-010 passes.
  CB-009 passes on all 8 seeds. Its salvage at (1060,1230) follows which
  ships the battle's draws killed, and stars-elegy #119 (4587f7d) marks
  that expectation `sample`, as its `fleet_gone` and `fleet` ones already
  were. Of its 63 expectations, 32 carry a `stream` and are not compared
  and 1 is a `battle` kind the harness does not compare. The pass rests
  on the 9 compared exact expectations (7 salvage amounts, 2 planets),
  which match on every seed. The 21 compared samples (16 `fleet_gone`,
  4 `fleet`, the 1 salvage) never count toward the pass; a sample that
  misses is skipped. Misses per seed, in seed order: 2, 2, 2, 2, 2, 1, 1,
  1 (the sampled salvage misses on 7 seeds, a `fleet_gone` on 6). The
  samples are not evidence of which fleets the battle destroys or of the
  sampled salvage amount.

**Expectations skipped inside checked cases** (reference seed, as of this
list; `TestParityVectors -v` prints the current counts as "expectations
skipped in checked cases"): other oracle random streams 4222, `message`
754 (Elegy's events are its own), `score_record` 319, samples that did
not match 318, player default queue settings 103, `battle` 54 and
`battle_actions` 42, planet `starbase` 22, transport `set_amount_to` 5
and `fuel` 5, computer players' planned orders 19, Super Stealth
research stealing 13, `battle_plan` 13, wormhole objects 10,
`no_battle` 9, player `research_field` 6, original environment null 5,
`mystery_trader_items` 5, and 1 to 4 each for planet `concentrations`,
`planetary_scanner` and `starbase_design`, player `counts` and
`ship_design_count`, `no_new_fleets`, `sample` and the design read.
- UG01-A..UG30-A: "player not in the state" (the expectation names a
  player the initial state lacks).
- RD-1..RD-7 and RW08 (the `rw` vectors): new games built from race
  files, which `newgame/vectors_test.go` checks (elegy #42); this harness has no
  players for them and skips them ("player not in the state", samples).
- `client_estimate` readings other than a fleet's "Est. range" and
  "Est. fuel usage", and `sample` mismatches (155): those client
  estimates are not modelled, and one stream's random outcome is never a
  failure (above). The two fleet estimates are compared
  (`Game.EstRange`, `Game.EstFuelUsage`; ESTIMATES.md): ES-001's 23
  ranges and 21 fuel readings and ES-002's 18 fuel readings all pass,
  61 of them CONFIRMED and ES-002-F07-w1 MEASURED.
- `view` of `other_players` (96), `player` (56) and `design` (4), and
  the view fields `population_estimate`, `defense_estimate`, `heading`,
  `mass_shown` and `environment_visible`: not compared yet.
- `battle` and `battle_actions` expectations (18, CS-003-C): battle
  records are compared only through their effects.
- Orders and tasks Elegy does not load: fleet-to-fleet transport (15),
  transport fuel and the fill, wait, set-amount and set-waypoint actions
  ("load all" and "load exactly" are loaded), `design` and
  `design_delete` orders.
- Production queues with unmapped planetary items: 10 (unused), 13
  (the Genesis Device) and the scanners 18–27. Packets launched from such
  a queue are skipped too ("a packet from a production queue").
- Stream cycles beyond the harness's seeds, "original environment null",
  "design read needs the order layer's design source", and KX-002-R3/R4
  (the next research field choice is not in the vector).

**Sample-only and random cases:** a case listed as `random k` in the
baseline passes with k of 8 seeds; it is evidence of the rule, never of
exact parity. A `sample: true` expectation counts only when it matches.
A case whose compared expectations are all samples is reported as
`sample-only`, not as a pass, and listed so in the baseline (`sample-only
k` when every sample missed on the other seeds): 25 cases (CB 3, OB 4,
WT 18) as of this list.

**Unmodelled steps:** see "Not modelled yet" above.

**Open questions:** K2 and K7 below; ASSUMPTIONs O9–O14, W1–W5, P1 and
S5 above.

## Open spec questions

- **K2 (ASSUMPTION), random event options.** `Game.RandomEvents` and
  `Game.Size` (0 tiny .. 4 huge) carry the game's option and universe
  size until new-game settings land; their names are Elegy's own.
- **K7 (ASSUMPTION), growth estimate for the caps.** Growth (step 4a)
  runs after every planet's production, with the environment production
  terraformed (KX-002 T1, T2 populations). The grown population the
  production caps read is estimated before production, from the
  environment before its terraforming; that is not measured.

- **K1, K3–K6** are answered by KERNEL.md (OT-6, KX-003): the AR
  loss applies whenever the next waypoint's warp is above 0; ship power
  uses the design's own speed code without the War Monger bonus; Orbital
  Forts are left out of the starbase count; a tie for the top score flags
  nobody for the lead; the needed count is capped at the enabled
  conditions (0: nobody wins by conditions); the game is decided again
  every year the conditions hold.

Choices the code makes where KERNEL.md is silent:

- **Generalized Research order.** Every field is checked for level-ups
  before any switch.
- **PG race settings.** KERNEL.md does not give the PG race's factories and
  mines operated; 10 per 10,000 colonists fits every PQ-001 cap.
