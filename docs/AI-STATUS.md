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
| Owned Mystery Trader parts and hulls count as buildable once tech meets their requirements, in ship and starbase designs | AI.md §5 "Mystery Trader items" | BINARY-ONLY | `ai/classes.go` `classPart`, `ai/designs.go`, `ai/starbase.go` (from `game.Report.TraderItems`) | `TestClassPartTrader`, `TestShipDesignTraderPart`, `TestStarbaseTraderPart`, `TestViewOfTraderItems` |
| Starbase designs: missing slots, the year-50 family switch, counts per variant, picture, name | AI.md §5 | CONFIRMED AI-2 | `ai/starbase.go` | `TestStarbase*` |
| Rototill's turn: research, starbase designs, U, planet loop and colony-ship production, fleet passes 1 and 2 | rototill.md §1–§3 | MEASURED AI-14..AI-17 (branches marked not exercised there are BINARY-ONLY) | `ai/rototill.go` | `TestRototill*` |
| Hubs: starbase planets and rich developed planets, from year index 20 | AI.md §6 | BINARY-ONLY | `ai/hubs.go` | `TestHubs` |
| Planet automation: starbases for hubs, starbase upgrade, defenses, mines and factories fill | AI.md §7 | BINARY-ONLY (AI-7 not run) | `ai/automation.go`, `ai/economy.go` | `TestMinesAndFactories`, `TestStarbaseUpgrade`, `TestDefenses`, `TestDesignCostMatchesEngine` |
| Under attack: quick defenses (and alchemy) at a planet a foreign bomber orbits, from year index (universe size + 2)·10 | AI.md §7 step 4 | BINARY-ONLY | `ai/automation.go` `underAttack` | `TestUnderAttack` |
| Blocked queues: mines or Auto Alchemy in front of a mineral-starved head, chosen by the head's completion estimate | AI.md §7 step 5; ESTIMATES.md "Production completion" (CONFIRMED, ES-001) for the estimate | BINARY-ONLY | `ai/automation.go` `blockedQueue`, `ai/estimate.go` | `TestBlockedQueue`, `TestBlockedQueueRun`, `TestBlockedQueueSkips`, `TestBlockedQueueLeftoverOnly`, `TestCompletion` |
| Ship-design builder and store: hull and class lists, delete-then-create, picture, name; ageing | AI.md §10 | BINARY-ONLY (builder CONFIRMED through AI-8, AI-19) | `ai/designs.go` | `TestAgeGroup` |
| Robotoid's design ladder, steps 1–7, with its reproduced LEGACY BUGs | robotoid.md §2 | CONFIRMED AI-8 | `ai/robotoid_designs.go` | `TestRobotoid*` |
| Cybertron's design steps 1–8: Frigate, Destroyers, Privateers, warship groups, guards | cybertron.md §2 | CONFIRMED AI-19 | `ai/cybertron_designs.go` | `TestCybertron*` |
| Cybertron's turn: merges by slot, parameters, ageing, splits, threat marks, fleet passes A and B (armada targeting, Destroyer attack targets, buddy joins, colony ships, freighters, slot-0 fleets), production, its starbase rule | cybertron.md §1, §3–§5, AI.md §10, §11 | Fleets MEASURED AI-21, starbases MEASURED AI-20; the rest BINARY-ONLY | `ai/cybertron.go`, `ai/automation.go` | `TestCybertron*` (the freighter invasion: `TestCybertronFreighterInvasion`, ASSUMPTION A55) |
| Cybertron's packets: supply, attack and the scanner shot, with packet marks | cybertron.md §6 | Warps, attack counts and scanner-shot destinations (as a band) MEASURED AI-24; supply not exercised; the rest BINARY-ONLY | `ai/packets.go` | `TestScannerShot*`, `TestAttackPacket*`, `TestViewOfForeignDesigns` |
| Scrap orders: waypoint 0's task set to scrap, route kept | AI.md §8; robotoid.md §4, rototill.md §3, cybertron.md §5 | Robotoid's MEASURED AI-3; a 2400 scrap by every expert but Rototill in the captured orders (AI.md §8); the rest BINARY-ONLY | `ai/fleet.go` `scrapOrder` | `TestDriversScrapStartingScouts` |
| Lay-mines tasks: Robotoid's idle scouts from year index 41, Cybertron's slot-0 fleets (here, or at a random nearby planet) | robotoid.md §4, cybertron.md §5 | Robotoid's duration MEASURED AI-25; the rest BINARY-ONLY (Cybertron's not exercised) | `ai/fleet.go` `layMines` | `TestDriversLayMines`, `TestCybertronMinelayer` |
| Robotoid's turn: merges, armada parameters, ageing, splits, threat marks, colonizer test, production, fleet passes A (transports, targets, colonizers, scouts), B (hub assignment and hub freighters) and C (obsolete fleets, armadas, join-up, attack targets) | robotoid.md §1, §3, §4; AI.md §6, §10, §11 | Fleets MEASURED AI-12, production order MEASURED AI-9; the rest BINARY-ONLY | `ai/robotoid.go`, `ai/turn.go` | `TestRobotoid*`, `TestAttackFleet`, `TestAssignHubs`, `TestHubLoad`, `TestHubFreighterLoad`, `TestArmadaDrop`, `TestArmadaInvades`, `TestDriversOrderSplitFleets` |
| Wormhole preference after a colonizable planet search, with the 16-bit distance wrap behind the `ai_wormhole_distance_wrap` switch (on in both built-ins) | AI.md §11 "Wormhole distance arithmetic" | LEGACY BUG, BINARY-ONLY | `ai/shared.go` `preferWormhole`, `wormholeD2` | `TestPreferWormhole`, `TestPreferWormholeBuiltins`, `TestPreferWormholeYear` |
| Warp re-pick: every fleet's first waypoint; inside another player's enlarged minefield 4, 5 or 6 (+1 SS), otherwise from 9 down while short of fuel, the raw cap and its exceptions, the slowest warp with the same years | AI.md §7 step 1, §11 "Warp choice" | CONFIRMED AI-11 (rule) | `ai/warp.go` (minefields from `game.Report.Objects`, MEASURED SC-038) | `TestWarpChoice*`, `TestMinefieldWarp`, `TestRototillScout`, `TestRototillEmptyColonyShipGoesHome` |
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
§6): it scraps its starting Scout (AI.md §8), a non-penetrating
planetary scanner reports no planets, and its colony ships take only planets it has seen. With the packets
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
| A32 | An armada launched by chance loads colonists as a normal launch does. |
| A33 | Armada launch target: planets chosen earlier this turn are drawn in planet-id order and the first that draws 0 wins. |
| A34 | Robotoid's colonizer step checks y > 4 first, and with no hubs the `Random(8 × hubs)` draw is not made. |
| A35 | "Population × max growth %" uses the population in hundreds; "the planet's resources" are this year's. |
| A36 | The universe size (colonizer test) is read from the map's extent. |
| A37 | Retired: the colonizer test's rule on b reads the ships ever built of the colony designs from the design slots (`engine.DesignSlot.Built`, save v7). |
| A38 | With no own fleet here that is not too weak, Robotoid's armada production step does nothing. |
| A39 | The D67 draw is made only when D67 holds a design. |
| A40 | Hub balancing takes the giving hub's last assigned fleet. |
| A41 | Hub freighters: the pickup marks (memory), small foreign colonies (values unpublished), salvage (not in the view) and the colonist rules are not built. |
| A42 | Join-up: the `Random(20)` draw is made only when the count test says join and the fleet holds 20 or more D1415 ships. |
| A43 | Player positions are never "close" (the view does not carry the setting). |
| A44 | A Robotoid scout's random nearby pick that lands on the planet it orbits means no move (robotoid.md: "other than its current one"), at its ideal warp otherwise. |
| A45 | A known wormhole end's reported stability stands for its movement class in the wormhole preference (AI.md §11), and counts as known. |
| A46 | A wormhole order targets the point in space where the end was last seen: the engine's waypoints cannot target a wormhole end yet. |
| A47 | Each attack-fleet item Cybertron's production queues adds one to its kind's fleet count only when its slot holds a design and the item is queued. |
| A48 | Packet supply: minerals tried in order ironium, boranium, germanium, the first with a target wins; "over 700 kT" and "at least 70 resources" are amounts left after the queue; the low notes read the target's surface minerals; a planet is not its own target; "up to 7" is as many as the amount left pays for, at most 7; the speed is w. |
| A49 | Attack packets (cybertron.md §6: the budget test BINARY-ONLY, the mass sent MEASURED AI-24): M sums the three minerals left after the queue and R is the available resources before it (§6 names the queue only for M); f is computed in floating point; the range test is inclusive. |
| A50 | An attack-packet target's catch warp c (the Dw + t of its starbase design known in full, OBJECTS.md "Impact") is not halved for an Interstellar Traveler owner: §6 names no halving, and the view holds no PRT. A target whose c is at least w is skipped, since a packet no faster than its catcher does no damage. |
| A51 | Retired: the scanner shot's mark on the planet one id higher and its w ≥ 14 distance overflow are the ruleset switches `CybertronPacketMarkNextID` and `CybertronScannerShotOverflow` (off in elegy, on in jrc3-faithful); with them off, the mark is on the destination. |
| A52 | A starbase with no mass driver (w = 3) leaves the packet speed unset: an order cannot carry warp 3. |
| A53 | Robotoid's lay-mines task lays indefinitely as robotoid.md §4 says (MEASURED AI-25); the assumption is only that the order's second field, written as 5 and UNRESOLVED there, is not modelled (the laying rules use only the duration), and that Cybertron's task, with no duration in cybertron.md, lays indefinitely too. |
| A54 | Robotoid does not clear an attack fleet's waypoint-0 marker task: robotoid.md §4 refers to it without defining it, and the only task Elegy's Robotoid sets that could be one, the scouts' lay mines, is never on an attack fleet. |
| A55 | Cybertron's freighter at another player's planet (cybertron.md §5, not exercised) unloads all its colonists there and moves to the pickup, or to the nearest own starbase planet when there is none: "move toward the nearest own starbase, and look for a pickup" read as one move. |
| A56 | Retired: a hub freighter's mineral orders away from the source follow AI.md §11 "Hub freighters" step 3 (MEASURED for Robotoid, AI-26): load all, load all of the scarce mineral only, or fill to 66 % and 33 % (`hubLoad`). |
| A57 | Retired: the warp re-pick's minefield rule is implemented from the minefields in the player's report (`game.Report.Objects`). |
| A58 | The raw-cap exception for an own planet at the waypoint reads the planet's starbase design index even when it has no starbase (§11: the slot is read). Such a planet holds its last starbase's design (the engine keeps it when a starbase is destroyed or scrapped), or 0 if it never had one. The index is looked up only among the player's starbase designs; any other index (a ship design, or a deleted starbase design) counts as no design, so the cap stays. |
| A59 | Robotoid's armada invasion at another player's planet (AI.md §11 "Armada (invasion) fleets", BINARY-ONLY) uses `need` (colonists) and the cargo `c` (kT) as plain numbers, with no conversion between them, as §11 reads; §11 marks the mix UNRESOLVED (a probable unit slip). The drop is a waypoint-0 unload-exactly task, with no move that turn. |
| A60 | Under attack (AI.md §7 step 4): a ship is a bomber when its design, known in full, carries a bomb part; a design known only by hull and mass does not count. A fleet orbits a planet when its sighting is at the planet's position. |
| A61 | Under attack: when n ≤ m (§7 names only n > m), the n defenses go to the front. "None queued" is no defense item in the queue; resources and minerals are the planet's available ones (§7 "available"), and the defense room is A14's. |
| A62 | Completion estimate (ESTIMATES.md "Production completion"): the walk stops at the head, leaving out the items behind it; the head finishes when its count reaches 0, or when an automatic head builds its whole count in a year; a starbase costs its full build cost (as A12); a packet head is built as a plain item. |
| A63 | Blocked queues (AI.md §7 step 5): "years" is the year the head's last unit finishes; "mines" and "terraforming" include their automatic items; the head's resource cost is what is left of it, counted as in `queueCost`; the planet's resources are its available resources (A15), with no research share on a planet that sends only leftover resources to research (as in the completion estimate); the mine room is that of the mines-and-factories fill. |
| A64 | Wormhole preference (AI.md §11): among ends with the same score and the same squared distance `w`, the first in the view's order wins; §11 gives only "ties: smaller w". §11's second distance test, `w ≤ 46,656` (216²), is applied in both modes: with the wrap on it always passes, as §11 says, and with it off it caps the exact distance at 216 ly. |

## Spec questions

None open.

## Decisions pending

None.

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

The driver keeps no state between years. Each design's creation year
and picture (AI.md §5, §10) are the game's: the design order stores
them on the engine's design slot, the save keeps them, and the report
gives them to the planner (`OwnDesign.Slot`). `TestDriversSaveReload`
plays the 50-year loop game under `elegy` and `jrc3-faithful`, saved and
reloaded into fresh drivers after 2425 and, separately, after every
year; every year's state hash matches the uninterrupted game
([#70](https://github.com/bfaber-centaur/elegy/issues/70), closed by this
test). It is a determinism check, not a parity check.

## Inputs the planners need from outside `ai/`

- Other players' PRT (the report withholds it; every other player counts
  as not Alternate Reality).

## Not implemented yet

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
