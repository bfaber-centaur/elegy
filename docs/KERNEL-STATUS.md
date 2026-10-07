# Peaceful kernel: implementation status

The J-RC3 peaceful turn and ordinary fleet movement are implemented in
`engine/` from the public specification in `bfaber-centaur/stars-elegy`:
`docs/KERNEL.md`, `docs/PARITY.md` (including KX-001 to KX-004) and the public FM-001..004
movement corpus (`experiments/fm00N`), as of stars-elegy `main` at `9569af3` (PR #44: scores, random events, AR colonists in flight, FM round 2; earlier PR #35:
KX-002 and the overcrowding correction).
Nothing here comes from the private archaeology repositories.

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
| Research | `research.go` | PG003 energy 2408–2436, level cost; KX-002 R1, R2 (cost settings), R5 (GR split), R6 (maxed field) | slower tech, several levels, next field, same-year switch, explicit/lowest persistence |
| Production queue | `production.go` | PQ-001 C01–C14 (15 cases); KX-001 A1–A4 (Auto Alchemy before a ×n item), M1–M4 (item costs) | empty queue (no tax), zero resources |
| Movement and fuel | `movement.go` | all 224 fleets of FM-001..004 (position, fuel, waypoints, warp, orbit, events), KERNEL fuel/range/chase vectors | no free warp, more than one engine per ship, cargo ties, chaser fuel per round (R, running dry, top-up, ram scoop) |
| Starbase refuelling | `movement.go` | FM-004 DK | |
| AR colonists in flight | `movement.go` `arColonistLoss` | TK-117, TK-107 (stars-elegy #44) | |
| Under-engined designs: f = 99999 and the 32-bit fuel-term wrap (LEGACY BUG, switch `legacyFuelWrap`) | `movement.go` `engineFactor`, `fuelTerm` | FM-105: 200, 50, 500 mg → 7, 1, 19 ly, 0 mg | the float form does not wrap |
| Random events: comet strike (sizes, kills, minerals, environment, queue cut), climate change, new minerals, option off | `randomevents.go` | KX-004 vectors (S2, S3, S5, E0), replayed in KERNEL.md's draw order; comet message axes LEGACY BUG behind `legacyCometAxes` | AR owner struck, the 180 cap, the probabilities |
| Score terms, ship classes, rank, flag word | `scores.go` | KX-003 S1 terms (planets, tech, ships, resources), Omega/Cherry class boundaries, flags 0x0ae0 / 0x0021 | capacitors, sappers, speed adjustment; score, resources and highest-score flags |
| Deciding the game, public scores | `scores.go` `decide`, `visibleScores` | public scores from year index 20 (KX-004 E0, E1) | deaths, survivor, winners after the minimum years; decided game and dead players' records |
| Improved Fuel Efficiency factor; fuel generators and fuel transports; Radiating Hydro-Ram Scoop colonist losses; refuelling at a friend's docked starbase | `movement.go` `fleetFactor`, `generateFuel`, `radiatingColonists`, `refuelFleets` | parity vectors FM-101..103 (`TestParityVectors`); KB-4A E, F1–F5, G, H, X and CS-003-W as cited in stars-elegy #53 "Other movement rules" | the RHRS immune and ≥ 170 exemptions |
| Claim Adjuster year-end step: original-value drift, then every axis to its limit with the reach just researched; terraform reach per axis | `terraform.go` `claimAdjusterYearEnd`, `terraformReach` | KX-003 S3/S3L; capture examples for TK-108, TK-118..121 (stars-elegy #53 24d4c09); parity vectors TK-108-A, TK-118-A, TK-119-A now match | the drift's draw sequence, CONFIRMED by KX-005 replays but tested here only with scripted draws |
| Fuel cannot be unloaded onto a planet | `takeover.go` (no fuel action) | FM-101..105 | |
| Whole turn | `turn.go` | PG homeworld 2407 → 2436 through `GenerateTurn`; PQ C01 and C14 over two years; KX-001 Z2, Z3 | |
| AR without a starbase | `turn.go` | | Elegy decision, below (`TestElegyDecision*`) |

## Turn order implemented

`GenerateTurn` runs: fleet movement (ordinary fleets in id order, then
fleet chasers in rounds, then waypoint settlement); mining for every planet
in id order (population before growth); per planet in id order: resources,
research tax, production queue (caps use the grown population); population growth for every planet; starbase refuelling;
research level-ups; random events (when `Game.RandomEvents` is on); battles, bombing and the after-movement takeover tasks
(COMBAT-STATUS.md, TAKEOVER-STATUS.md); repair; year + 1; scores, victory flags and deciding the game
(`TurnResult.Scores`, each view's visible records). The
before-movement takeover tasks (unloads, colonize, drops) come first.

Not modelled yet: order application, waypoint tasks other than unloads and
colonize and merge (load, scrap, transfer; ORDERS-STATUS.md), space objects and the Mystery Trader, mine
sweeping,
terraforming other than the Claim Adjuster's year-end step (production
items, Orbital Adjusters), remote mining, Super Stealth research stealing, the
duplicate-serial penalty, ships/starbases in the queue, fuel generators,
friends' starbases, and the BINARY-ONLY movement rules for IFE, Cheap
Engines, warp-10 losses, Radiating Hydro-Ram colonist
deaths and transport/lay-mines tasks.

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
  BUG). Elegy: `GenerateTurn` returns a `*ZeroMaxPopulationError` naming
  the planet before changing anything. The rule lives in `checkGenerable`
  (`turn.go`), so a different choice is a change there and in the
  population rule. A hostile planet in the same state generates normally
  (hostile deaths, 1 resource; KX-001 Z3). Calling `GrowPopulation`
  directly with maximum 0 still treats the planet as fully overcrowded.

## Open spec questions

- **K2 (ASSUMPTION), random event options.** `Game.RandomEvents` and
  `Game.Size` (0 tiny .. 4 huge) carry the game's option and universe
  size until new-game settings land; their names are Elegy's own.

- **K1, K3–K6** were answered by stars-elegy #53 (OT-6, KX-003): the AR
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
