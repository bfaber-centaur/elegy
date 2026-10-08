# Computer players: status

`ai/` holds Elegy's computer-player planners. The specification is
stars-elegy `docs/AI.md` (shared core) and `docs/ai/robotoid.md`,
`docs/ai/rototill.md` and `docs/ai/cybertron.md`. Only those three
personalities are implemented (AI.md "Project policy"); Turindrone,
Automitron and Macinti are reference behavior and stay unimplemented
unless the project decides otherwise.

## Design

- A planner reads one player's view and returns that player's orders as
  ordinary engine orders (AI.md §1). It never reads the true game state.
- Planners are not part of the engine's turn order. Running them before
  the year, in player order, on the game's random stream (AI.md §1
  "Random numbers") is the game loop's job.
- Clean per-player state (AI.md §1 "State leaking between computer
  players"): empty design slots read as empty and the armada parameters
  as 0 unless the planner set them in its own turn. This is
  INTENTIONALLY DIFFERENT from the original by project decision
  (2026-10-07). The legacy switch `legacy_ai_state_leak` is not
  implemented yet.
- No memory between years (AI.md §1, CONFIRMED AI-10).
- Random draws are made in the order the spec gives, so the draw count
  matches the spec. Elegy does not claim parity of draw-dependent orders:
  the original's computer players share one stream in player order, so a
  draw-dependent order matches only when every earlier computer player's
  turn ran too (AI.md §1, MEASURED AI-18).

## Implemented

| Rule | Spec | Evidence | Code | Tests |
|---|---|---|---|---|
| Research budget, field and next field; when the order is written | AI.md §4 | CONFIRMED AI-1 | `ai/research.go` | `TestResearch*` |
| Own-planet shuffle | AI.md §2 | BINARY-ONLY | `ai/shuffle.go` | `TestShufflePlanets` |
| AI part classes 0–44 | AI.md §5 | CONFIRMED with AI-2, AI-8, AI-19 | `ai/classes.go` | `TestPartClassNamesExist` |
| Starbase designs: missing slots, the year-50 family switch, counts per variant, picture, name | AI.md §5 | CONFIRMED AI-2 | `ai/starbase.go` | `TestStarbase*` |

| Rototill's turn: research, starbase designs, U, planet loop and colony-ship production, fleet passes 1 and 2 | rototill.md §1–§3 | MEASURED AI-14..AI-17 (branches marked not exercised there are BINARY-ONLY) | `ai/rototill.go` | `TestRototill*` |
| Hubs: starbase planets and rich developed planets, from year index 20 | AI.md §6 | BINARY-ONLY | `ai/hubs.go` | `TestHubs` |
| Planet automation: starbases for hubs, starbase upgrade, defenses, mines and factories fill | AI.md §7 | BINARY-ONLY (AI-7 not run) | `ai/automation.go`, `ai/economy.go` | `TestMinesAndFactories`, `TestStarbaseUpgrade`, `TestDefenses`, `TestDesignCostMatchesEngine` |
| Ship-design builder and store: hull and class lists, delete-then-create, picture, name; ageing | AI.md §10 | BINARY-ONLY (builder CONFIRMED through AI-8, AI-19) | `ai/designs.go` | `TestAgeGroup` |
| Robotoid's design ladder, steps 1–7, with its reproduced LEGACY BUGs | robotoid.md §2 | CONFIRMED AI-8 | `ai/robotoid_designs.go` | `TestRobotoid*` |
| Cybertron's design steps 1–8: Frigate, Destroyers, Privateers, warship groups, guards | cybertron.md §2 | CONFIRMED AI-19 | `ai/cybertron_designs.go` | `TestCybertron*` |
| A view of one player's report | AI.md §1 | CONFIRMED AI-12 (planet view) | `ai/view.go`, `ai/report.go` | `TestRototillPlaysAlone` |

The tests are unit tests of the rules as written; none is an oracle
capture comparison.

`TestRototillPlaysAlone` plays an expert Rototill against an idle human
for 40 years from a new game: Rototill plans first each year on the
game's random stream, every order it gives is accepted, and the same
seed replays to an identical game. It is a smoke test, not a parity
check.

## Assumptions

| Id | Choice |
|---|---|
| A1 | With no research plan (or every goal met) the original writes the research order only when the field changes. Elegy also writes it when only the budget changes (Rototill at year index 20), so the budget takes effect. AI.md §4 does not say how the budget changes without an order. |
| A2 | Starbase step 1 creates slot 4 together with slot 2 whether or not slot 4 holds a design, as AI.md §5 reads; a computer player's slot 4 is empty whenever slot 2 is. |
| A3 | "A slot with fewer than 4 that holds an orbital part or more than one item loses one" (variant 1) is read as: the chosen part is an orbital part, or the count is above 1. Orbital parts in the starbase classes are mass drivers, which only PP can build, and Cybertron is exempt, so the orbital clause never acts for the three personalities. A slot reduced to 0 is left empty. |
| A4 | A starbase design being replaced still counts for picture and name choice, as AI.md §10 says for ship designs. |
| A5 | After 20 failed name tries the name is `<name> <n>` with `n` from `Random(100)`; AI.md calls the original's form BINARY-ONLY. Elegy uses its own 13 starbase names (AI.md §5 allows it). |
| A6 | A planet report's current environment stands for the planet's original values in "habitability after terraforming" (ESTIMATES.md "Value and optimal value"); a player's report does not carry them. |
| A7 | An armed Meta Morph with 500 kT of cargo is not yet recognised as a transport (AI.md §11 "Fleet classes"); it needs the design-power formula. Rototill never has one. |
| A8 | Rototill's "known at level 3 or more" (rototill.md §1, §2; its meaning is an open experiment there) is an Elegy report at normal level or above, which carries environment and concentrations. "Desirability" is the planet's value for the race. |
| A9 | The scout's "wormhole within that distance" (rototill.md §3 step 4) is the nearest one, and `Random(100)` is drawn only when one exists. |
| A10 | The scout fallback's "best armada destination counted from the homeworld" is the highest §2 destination score, ties to the planet nearer the homeworld. |
| A11 | The turn's shuffled planet order (AI.md §2) is drawn right after the starbase designs. Hubs take own planets in that order, starbase planets first (robotoid.md §4 pass B; AI.md §6 does not order the rest). |
| A12 | The "full cost of the queue" (AI.md §7) counts auto items at their count like plain items, and a starbase item at its full build cost. |
| A13 | Starbase upgrade: the draw comes before the mineral test, and the upgrade is appended. |
| A14 | Defenses' "room" is the operable defenses less those installed and queued, and "population/8,000" is in colonists. |
| A15 | Mines and factories: "resources" is the planet's available resources; the mines' "resources left" is after the factories just queued; the alchemy is appended. Room counts queued plain items only. |
| A16 | Cybertron's list range `a..b`: `Random(m)` indexes the lists left in increasing order, and the list tried is removed (cybertron.md §2 checked ranges as sets of outcomes, AI-19). |
| A17 | Cybertron's warship group: when the big ships leave the target at the group's first slot, that slot gets the "first" Cruiser range (the last 9 lists); "slot g itself range 17..19" applies when the fill reaches it from above. |
| A18 | Robotoid's step 7 writes no delete for a slot 0 that is already empty (the original repeats it every year; Elegy's engine would reject it and it changes nothing). |

## Spec questions

- AI.md §11 "Nearest colonizable planet" says non-Robotoid personalities
  compute the "already targeted" marks once per turn, so a planet chosen
  earlier in the turn is not excluded; rototill.md §3 says Rototill marks
  the chosen planet so later fleets skip it. `ai/` follows rototill.md.
  Sent to the research lane for a one-line clarification.

## Decisions pending

- Wormhole distance overflow (AI.md §11, LEGACY BUG: wormholes about
  182 ly or more away count as near). The planner compares exact
  distances. It has no effect until the game supplies wormholes; before
  then the project chooses between reproducing it behind the legacy
  switch (the exact arithmetic would need a research answer) and the
  fixed rule (INTENTIONALLY DIFFERENT, Bobby's call).

## Inputs the planners need from outside `ai/`

- The game's random stream, in player order, before the year (AI.md §1
  "Random numbers"). The game loop's driver interface does not pass one
  yet.
- Each design's creation year and picture (AI.md §5, §10). The engine's
  designs carry neither; `ai.SlotDesign` takes them from the caller and
  `ai.NewDesign` returns them for the designs a planner stores.
- Every planet's id and position, seen or not (rototill.md §3 scouts,
  MEASURED AI-17; Robotoid colonizes planets it never scanned, AI.md §1).
- The player's planet history: for each planet ever reported, the latest
  report (AI.md §1 "What it sees", CONFIRMED AI-12; used by Rototill's
  scouts and colonizers, AI-16, AI-17).

## Not implemented yet

- Hub freighter assignment (AI.md §6 steps 3–4), automation steps 1
  (warp re-pick, AI.md §11 "Warp choice"), 4 (under attack) and 5
  (blocked queues); the shared fleet rules of AI.md §11.
- Robotoid's and Cybertron's turns: ship designs, ageing, splitting,
  merging, production, fleets, packets.
- Orders the engine does not accept yet: scrap and lay-mines tasks. A
  planner reports a scrap it cannot order in `Result.Unsupported`.
- Rototill's unreachable branches (rototill.md marks them not exercised):
  remote miners (slots 7–8), slots 13–14, transports (hub freighters) and
  armed scouts. Rototill never designs those ships; the code reports them
  as unsupported or leaves the fleet alone.
- Wormholes and other players' PRT in the view (the game loop's report
  does not carry them yet).
