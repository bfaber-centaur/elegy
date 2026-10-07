# Peaceful kernel: implementation status

The J-RC3 peaceful turn and ordinary fleet movement are implemented in
`engine/` from the public specification in `bfaber-centaur/stars-elegy`:
`docs/KERNEL.md`, `docs/PARITY.md` and the public FM-001..004 movement
corpus (`experiments/fm00N`), as of stars-elegy `main` at `57610ab` (after PR #14).
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
| Habitability | `habitability.go` | center → 100 | 7 KERNEL vectors |
| Maximum population | `population.go` | hab 100 → 10,000 | HE, JOAT, OBRM, hab < 5, AR |
| Population growth and carry | `population.go` | PG-001..003, 36 years of P and carry | 6 KERNEL vectors (g < 1000, frozen, overcrowded, zero growth, ...), hostile deaths, empty planets, HE ×2 |
| Resources, caps | `economy.go` | PG resources, PQ operable caps | maximums, over-max E, AR resources and mines |
| Mining | `mining.go` | PG002 concentration and fraction 2407–2411 | random +1 draw, homeworld floor, clamp re-evaluated, draw order |
| Research | `research.go` | PG003 energy 2408–2436, level cost | other cost settings, slower tech, several levels, next field, same-year switch, explicit/lowest persistence, GR, maxed field |
| Production queue | `production.go` | PQ-001 C01–C14 (15 cases) | empty queue (no tax), zero resources |
| Movement and fuel | `movement.go` | all 224 fleets of FM-001..004 (position, fuel, waypoints, warp, orbit, events), KERNEL fuel/range/chase vectors | no free warp, more than one engine per ship, cargo ties, chaser fuel per round (R, running dry, top-up, ram scoop) |
| Starbase refuelling | `movement.go` | FM-004 DK | |
| Whole turn | `turn.go` | PG homeworld 2407 → 2436 through `GenerateTurn`; PQ C01 and C14 over two years | |

## Turn order implemented

`GenerateTurn` runs: fleet movement (ordinary fleets in id order, then
fleet chasers in rounds, then waypoint settlement); mining for every planet
in id order (population before growth); per planet in id order: resources,
research tax, production queue (caps use the grown population); population growth for every planet; starbase refuelling;
research level-ups; year + 1.

Not modelled yet: order application, waypoint tasks (load, unload, colonize,
scrap), space objects, random events, battles, mine sweeping, repair,
terraforming, remote mining, scores, Super Stealth research stealing, the
duplicate-serial penalty, ships/starbases in the queue, fuel generators,
friends' starbases, and the BINARY-ONLY movement rules for IFE, Cheap
Engines, warp-10 losses, AR colonist losses, Radiating Hydro-Ram colonist
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

## Open spec questions

Each of these is one of KERNEL.md's "Open experiments", waiting on an
oracle case; the code's current choice is given.

1. **Auto Alchemy before a multi-count item** (Open experiments, "Auto
   Alchemy before a multi-count item"). Code: the shortfall is for the
   item's whole remaining count; when resources cover the shortfall,
   alchemy buys it and the item is processed normally; when they do not,
   the item first takes its partial on current stock and alchemy then
   converts what is left (the order C07's 49% and 46% require).
2. **Zero maximum population** (Open experiments, "Zero maximum
   population"): Alternate Reality without a starbase. Code: treated as
   fully overcrowded.
3. **Cost modifiers** (Open experiments, "Cost modifiers"). Code: defense
   (15 + 5/5/5) and alchemy (100) costs are constants; the Mineral Alchemy
   LRT and other cost modifiers are not modelled.

Also unstated, not an open experiment: KERNEL.md does not give the PG race's
factories and mines operated; 10 per 10,000 colonists fits every PQ-001 cap.
