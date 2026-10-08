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
| Cybertron's turn: merges by slot, parameters, ageing, splits, threat marks, fleet passes A and B (armada targeting, Destroyer attack targets, buddy joins, colony ships, freighters, slot-0 fleets), production, its starbase rule | cybertron.md §1, §3–§5, AI.md §10, §11 | Fleets MEASURED AI-21, starbases MEASURED AI-20; the rest BINARY-ONLY | `ai/cybertron.go`, `ai/automation.go` | `TestCybertron*` |
| Cybertron's packets: supply, attack and the scanner shot, with packet marks | cybertron.md §6 | Warps, attack counts and scanner-shot destinations (as a band) MEASURED AI-24; supply not exercised; the rest BINARY-ONLY | `ai/packets.go` | `TestScannerShot*`, `TestAttackPacket` |
| Robotoid's turn: merges, armada parameters, ageing, splits, threat marks, colonizer test, production, fleet passes A (transports, targets, colonizers, scouts), B (hub assignment and hub freighters) and C (obsolete fleets, armadas, join-up, attack targets) | robotoid.md §1, §3, §4; AI.md §6, §10, §11 | Fleets MEASURED AI-12, production order MEASURED AI-9; the rest BINARY-ONLY | `ai/robotoid.go`, `ai/turn.go` | `TestRobotoid*`, `TestAttackFleet`, `TestAssignHubs` |
| A view of one player's report | AI.md §1 | CONFIRMED AI-12 (planet view) | `ai/view.go`, `ai/report.go` | `TestRototillPlaysAlone` |

The tests are unit tests of the rules as written; none is an oracle
capture comparison.

`TestRototillPlaysAlone`, `TestCybertronPlaysAlone` and
`TestRobotoidPlaysAlone` play one expert computer player against an idle
human from a new game, for 40, 60 and 60 years: the computer player plans first each year on the game's random
stream, every order it gives is accepted, and the same seed replays to
an identical game. `TestAllThreeTogether` plays Robotoid, Rototill and
Cybertron in one game for 60 years, each planning in player order on the
one random stream with its own report and history, with the same checks.
They are smoke tests, not parity checks. The same three also play
through the game loop (`ai.Driver`, below).

Cybertron explores only with its scanner-shot packets (cybertron.md
§6): it scraps its starting Scout (Elegy reports the scrap as
unsupported), a non-penetrating planetary scanner reports no planets,
and its colony ships take only planets it has seen. With the packets
built, `TestCybertronPlaysAlone` requires it to own more than its
homeworld after 60 years (it logged 21 planets on this branch); before
them it stayed on its homeworld.

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
| A19 | Merging: a later pass (after more than 32 places) skips the places an earlier pass tracked. |
| A20 | Cybertron's `GR` on equal creation years of slots 6 and 10 is group 6–9. |
| A21 | robotoid.md §1 does not name the threat mark's population unit; Robotoid uses Cybertron's (cybertron.md §4.1: the estimate in units of 400 colonists). |
| A22 | Cybertron's own-planet shuffle is drawn after the starbase designs, as Rototill's (A11). |
| A23 | Cybertron's weak-armada branches (stay, retreat, hard-level draws) are not built: under clean per-player state the armada parameters are 0, so no armada is ever weak. They come with the legacy switch. |
| A24 | An armada in deep space moves with no task at warp 4. |
| A25 | Attack target: "can reach an own starbase" is not tested (the engine's waypoint fuel estimate is not exported); with the planner's empty memory no planet is marked visited. Computer alliances are off. |
| A26 | The colonizer test's `Random(2)` passes when it draws 0. |
| A27 | Attack fleet production: the guard draw is made only when `GG` holds a design and fewer than 40 guard fleets exist; the third and fourth group draws are made whether or not those slots hold designs. |
| A28 | Retired: cybertron.md §5 now gives pass A's fleet classes. |
| A29 | "Power > 0" in the fleet classes is tested as "some beam, torpedo or bomb term > 0" (KERNEL.md "Power of a design"): capacitors and the speed factor never take a positive beam term to zero. |
| A30 | Robotoid's "merge the slots 2–7 and 9–10" is one merge over those slots. |
| A31 | An armada not at a planet with no other player's planet within 150 ly goes to the nearest planet ("the nearest object of interest"). |
| A32 | An armada launched by chance loads colonists as a normal launch does; the invasion step at another player's planet is reported as unsupported (no defense-percentage estimate yet). |
| A33 | Armada launch target: planets chosen earlier this turn are drawn in planet-id order and the first that draws 0 wins. |
| A34 | Robotoid's colonizer step checks y > 4 first, and with no hubs the `Random(8 × hubs)` draw is not made. |
| A35 | "Population × max growth %" uses the population in hundreds; "the planet's resources" are this year's. |
| A36 | The universe size (colonizer test) is read from the map's extent. |
| A37 | The colonizer test's rule on ships ever built of the colony designs (b) is skipped: Elegy does not track that count. |
| A38 | With no own fleet here that is not too weak, Robotoid's armada production step does nothing. |
| A39 | The D67 draw is made only when D67 holds a design. |
| A40 | Hub balancing takes the giving hub's last assigned fleet. |
| A41 | Hub freighters: the pickup marks (memory), small foreign colonies (values unpublished), salvage (not in the view) and the colonist rules are not built. Loads at a target are reported, since Elegy's transport task cannot load. |
| A42 | Join-up: the `Random(20)` draw is made only when the count test says join and the fleet holds 20 or more D1415 ships. |
| A43 | Player positions are never "close" (the view does not carry the setting). |
| A44 | A Robotoid scout's random nearby pick that lands on the planet it orbits means no move (robotoid.md: "other than its current one"), at its ideal warp otherwise. |
| A45 | A known wormhole end's reported stability stands for its movement class in the wormhole preference (AI.md §11), and counts as known. |
| A46 | A wormhole order targets the point in space where the end was last seen: the engine's waypoints cannot target a wormhole end yet. |
| A47 | Each attack-fleet item Cybertron's production queues adds one to its kind's fleet count only when its slot holds a design and the item is queued. |
| A48 | Packet supply: minerals tried in order ironium, boranium, germanium, the first with a target wins; "over 700 kT" and "at least 70 resources" are amounts left after the queue; the low notes read the target's surface minerals; a planet is not its own target; "up to 7" is as many as the amount left pays for, at most 7; the speed is w. |
| A49 | Attack packets: M sums the three minerals left; "the budget can kill" is a kill mass of at most M; the kill mass is divided by q^(distance/w²) (what a packet keeps over its flight, so a farther target needs more), in floating point with the distance in ly; the count is ⌈kill mass / 70⌉; ranges are inclusive. |
| A50 | The report does not carry the parts of another player's starbase design, so its catch warp cannot be read: only planets with no starbase in view are attack-packet targets. |
| A51 | Retired: the scanner shot's mark on the planet one id higher and its w ≥ 14 distance overflow are the ruleset switches `CybertronPacketMarkNextID` and `CybertronScannerShotOverflow` (off in elegy, on in jrc3-faithful); with them off, the mark is on the destination. |
| A52 | A starbase with no mass driver (w = 3) leaves the packet speed unset: an order cannot carry warp 3. |

## Spec questions

- cybertron.md §6 attack: whether the kill mass is multiplied or divided
  by `q^(distance/w²)` (A49).

## Decisions pending

- Wormhole distance overflow (AI.md §11, LEGACY BUG: wormholes about
  182 ly or more away count as near). The planner compares exact
  distances. It has no effect until the game supplies wormholes; before
  then the project chooses between reproducing it behind the legacy
  switch (the exact arithmetic would need a research answer) and the
  fixed rule (INTENTIONALLY DIFFERENT, Bobby's call).

## Game-loop driver

`ai.Driver` implements `game.Driver` for Robotoid, Rototill and
Cybertron. Each year it builds the player's View from the
`game.Report` alone (universe map, planet history, known wormhole
ends), and plays the turn on the report's random stream. The game loop
asks the players in player order, so the computer players draw from one
stream as AI.md §1 says. `TestDriversInGameLoop` plays an idle human and
the three personalities for 50 years through `game.Game.Advance`: every
order is accepted, each keeps a planet, and two runs from the same seed
give the same state hash every year. It is a smoke test, not a parity
check.

The driver's only state is the creation year and picture of the designs
it stored (AI.md §5, §10), which the engine's designs do not carry. It
does not survive a save and load (`game.Driver`'s ELEGY CHOICE); after a
load those designs count as created in the first year with picture 0.

## Inputs the planners need from outside `ai/`

- Each design's creation year and picture, in the game state, so that
  the planners' ageing and picture rules survive a save and load.
- Other players' PRT (the report withholds it; every other player counts
  as not Alternate Reality).
- The parts of another player's fully known starbase design, for the
  attack packets' catch warp (A50).

## Not implemented yet

- Automation steps 1 (warp re-pick, AI.md §11 "Warp choice"), 4 (under
  attack) and 5 (blocked queues).
- Orders to a fleet split off in the same turn: the engine gives the new
  fleet an id the order file cannot name yet (engine request open); such
  orders go to `Result.Unsupported`.
- Orders the engine does not accept yet: scrap and load tasks, and
  unloading colonists at another player's planet (freighter invasion). A
  planner reports a scrap it cannot order in `Result.Unsupported`.
- The lay-mines steps: the engine now has a lay-mines task, but the
  planners do not order it yet and report the step as unsupported.
- Battle plan 4 is ordered now that every new player starts with the
  five plans (newgame, COMBAT.md "Starting plans"); a player with fewer
  plans reports it as unsupported.
- Rototill's unreachable branches (rototill.md marks them not exercised):
  remote miners (slots 7–8), slots 13–14, transports (hub freighters) and
  armed scouts. Rototill never designs those ships; the code reports them
  as unsupported or leaves the fleet alone.
- Waypoints to a wormhole end: the engine's waypoints cannot target one
  yet, so the planners send the fleet to the point where the end was last
  seen (A46).
