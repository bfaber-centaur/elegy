# Development direction

## Goal

Build a small deterministic simulation engine first, then grow it into a complete 4X game.

A J-RC3-compatible ruleset is a major target, but compatibility behavior should be based on measured/documented evidence rather than intuition. Explicitly fixed or modernized rulesets can coexist later.

The immediate goal is not UI parity, original-file compatibility, networking, AI opponents, or a Steam-ready application.

## Core architecture

Conceptually:

```text
GameState + PlayerOrders + Ruleset + explicit RNG state
                         ↓
                    GenerateTurn
                         ↓
                 TurnResult / reports
```

The simulation should not depend on:

- a graphical or terminal UI;
- wall-clock time;
- ambient filesystem state;
- networking;
- original Stars! save formats;
- hidden global randomness.

### Truth versus player knowledge

Authoritative universe state and player-visible information should remain separate concepts.

Eventually:

```text
GameState → ViewForPlayer(player) → PlayerView
```

A client should not need omniscient state and then be trusted to hide secrets.

### Rulesets

Compatibility differences should be explicit rather than scattered through unrelated conditionals. The exact interface can remain small until real variation requires it.

Likely long-term rulesets include:

- J-RC3-compatible;
- J-RC3 with explicitly chosen fixes;
- modern/experimental rules.

### Data first

Favor straightforward data structures and testable transformations. Split packages when real subsystem boundaries appear rather than pre-creating a large architecture.

## Current skeleton

The initial code was moved from the earlier `stars-elegy` prototype.

It currently demonstrates only:

- constructing a minimal game state;
- advancing a year;
- standard-race population capacity on positive-habitability worlds;
- simple uncrowded population-growth scaling.

That is scaffolding, not a claim of complete parity. In particular, the current growth function does not yet model the persistent J-RC3 growth remainder or crowding behavior already under investigation in `stars-elegy`.

## Near-term milestones

### 1. Exact population model

Promote settled population findings from `stars-elegy`:

- representation / fractional growth carry;
- positive-habitability scaling;
- crowding above 25% capacity;
- behavior at capacity and overcrowding;
- hostile-world deaths;
- relevant racial modifiers.

Every promoted rule should have evidence-backed tests.

### 2. Peaceful economy vertical slice

Once population is trustworthy, add only enough state for:

```text
race
→ planet environment
→ population
→ resources
→ factories / mines
→ production
→ research
```

A useful checkpoint is a deterministic homeworld scenario that can be simulated for decades with every yearly value explained.

### 3. Expansion

Add galaxy generation, ship designs, fleets, movement/fuel, cargo, and colonization.

### 4. Information model

Add scanning and stale player intel, then expose a safe `PlayerView`.

### 5. Combat and remaining systems

Build combat as a deterministic subsystem with tiny scenarios before integrating the long tail: starbases, minefields, terraforming, packets, gates, racial mechanics, bombing/invasion, diplomacy, and special events.

## Front ends

The engine should support more than one client.

Near-term experimentation may include a keyboard-first TUI. A bespoke graphical desktop client can become the mainstream product later. Neither should own simulation rules.

## Testing

Use several layers:

- formula/unit tests;
- subsystem interaction tests;
- complete turn tests;
- multi-turn deterministic scenarios;
- J-RC3 parity tests promoted from the research repository.

For uncertain compatibility behavior:

```text
hypothesis → experiment / evidence → specification → test → implementation
```

## Persistence

Do not reproduce the original binary formats as Elegy's native save model.

Early tests can construct state directly. When persistence becomes useful, choose a deterministic, versioned, inspectable format around Elegy's own domain model.

Original Stars! formats, if supported, are import/export compatibility.
