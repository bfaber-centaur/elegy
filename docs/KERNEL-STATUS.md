# Peaceful kernel: implementation status

The J-RC3 peaceful turn and ordinary fleet movement are implemented in
`engine/` from the public specification in `bfaber-centaur/stars-elegy`:
`docs/KERNEL.md`, `docs/PARITY.md`, and the public FM-001..004 movement
corpus (`experiments/fm00N`), all as of stars-elegy `main` at `b927632`. Nothing here comes from
the private archaeology repositories.

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
| Population growth and carry | `population.go` | PG-001..003, 36 years of P and carry | 6 KERNEL vectors (g < 1000, frozen, overcrowded, zero growth, ...), hostile deaths, HE ×2 |
| Resources, caps | `economy.go` | PG resources, PQ operable caps | maximums, over-max E, AR resources and mines |
| Mining | `mining.go` | PG002 concentration and fraction 2407–2411 | random +1 draw, homeworld floor |
| Research | `research.go` | PG003 energy 2408–2436, level cost | other cost settings, slower tech, several levels, next field, GR, maxed field |
| Production queue | `production.go` | PQ-001 C01–C14 (15 cases) | empty queue, zero resources |
| Movement and fuel | `movement.go` | all 224 fleets of FM-001..004 (position, fuel, waypoints, warp, orbit, events), KERNEL fuel/range/chase vectors | no free warp, more than one engine per ship |
| Starbase refuelling | `movement.go` | FM-004 DK | |
| Whole turn | `turn.go` | PG homeworld 2407 → 2436 through `GenerateTurn`; PQ C01 and C14 over two years | |

## Turn order implemented

`GenerateTurn` runs: fleet movement (ordinary fleets in id order, then
fleet chasers in rounds); per planet in id order: mining (population before
growth), resources, research tax, production queue (caps use the grown
population); population growth for every planet; starbase refuelling;
research level-ups; year + 1.

Not modelled yet: order application, waypoint tasks (load, unload, colonize,
scrap), space objects, random events, battles, mine sweeping, repair,
terraforming, remote mining, scores, Super Stealth research stealing, the
duplicate-serial penalty, ships/starbases in the queue, fuel generators,
friends' starbases, and the BINARY-ONLY movement rules for IFE, Cheap
Engines, warp-10 losses, AR colonist losses, Radiating Hydro-Ram colonist
deaths and transport/lay-mines tasks.

## Spec gaps and choices

Questions for stars-elegy, with what the code does meanwhile. Items marked
*corpus* are where the public corpus decides something KERNEL.md does not
state.

1. **Auto item caps.** KERNEL.md "Caps" says production caps are
   `max(maximum, operable) − installed` for both auto items and plain
   installation orders. PQ-001 C04, C09, C13 and C14 only fit `operable −
   installed` for auto items (C09: 55 mines, not all the resources' worth).
   Code: auto items use `operable − installed`, plain orders are clipped to
   `max(maximum, operable) − installed` (C10). *corpus*
2. **Running dry with exactly enough fuel.** KERNEL.md lowers the warp only
   when the allowed distance exceeds `R`. FM-002 fleet 24 (fuel 6, cost of
   36 ly = 6, R 37) moved the full 36 ly and still got the out-of-fuel event
   and warp 1. Code: a fleet that could not pay for the whole leg and ends
   the year with 0 mg short of its destination has run dry. Unknown: a
   fleet that cannot pay for the leg, is not range-limited, and keeps
   some fuel (no corpus case). *corpus*
3. **Mutual chases.** When a chaser lands on a target that is chasing it,
   the target's waypoint is also completed (event "completed orders"), even
   if the target had already used its movement (FM-001 72, 73; FM-002 40;
   FM-003 28, 30, 32, 33). KERNEL.md says only that the target stops.
   *corpus*
4. **Chasers and fuel.** KERNEL.md gives no fuel limit, running-dry,
   top-up or ram-scoop rule for fleets chasing fleets. Code charges fuel on
   the year's total distance only, never below 0.
5. **Auto Alchemy before an item.** The PQ text covers a ×1 item. Code:
   the shortfall is for the item's whole remaining count; when resources
   cover the shortfall, alchemy buys it and the item is processed normally;
   when they do not, the item first takes its partial on current stock and
   alchemy then converts what is left (the order C07's 49% and 46% require).
6. **Research field switching.** After a level-up moves the leftover to the
   next field, are further level-ups in the new field checked the same year?
   (Code: no.) Does the choice persist? What does "cheapest field" do (not
   implemented)? With Generalized Research, code checks level-ups in every
   field, and only the current field triggers a switch.
7. **Depletion clamp.** Is the clamped concentration (`cc`) re-evaluated as
   the concentration drops within one year? It only matters crossing 5
   (25 → 10). Code: re-evaluated.
8. **Empty queue.** "Contributes nothing to research": does that include the
   research tax? Code: nothing at all.
9. **Zero maximum population** (Alternate Reality without a starbase):
   the crowding permille divides by zero. Code: treated as fully
   overcrowded.
10. **Empty planets.** Hostile deaths of `max(1, …)` would act on a planet
    with no population. Code: a planet with 0 population does not change.
11. **Cargo ties.** Order of designs with equal `f(w)` when assigning cargo.
    Code: fleet stack order.
12. **Mining draws.** Order of the `rand(100)` draws across minerals and
    planets. Code: planets in id order, ironium, boranium, germanium.
13. **PG race settings.** KERNEL.md does not state the PG race's factories
    and mines operated; 10 per 10,000 colonists fits every PQ-001 cap.
14. **Fixed costs.** Defense (15 + 5/5/5) and alchemy (100) costs are
    constants; the Mineral Alchemy LRT and other cost modifiers are not
    modelled.
