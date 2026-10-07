# Orders status

Fleet operations and the design read in `engine/fleetops.go` follow the
public specification stars-elegy `docs/ORDERS.md` on `main` at `5a6621a`
(PR #40 fleet operations, with the corrections and the design-legality
rule merged in #48, and the answers to Elegy's questions merged in #51), and the task placement in `docs/TAKEOVER.md` ("Where
each task happens", "Other waypoint tasks"). Nothing else was used.

Elegy does not apply order files yet (KERNEL-STATUS.md). The merge order
and the design read are engine functions an order layer will call:
`Game.MergeFleets` and `Catalog.ReadDesign`.

## Rules and tests

| Rule (ORDERS.md section) | Code | Status | Test |
|---|---|---|---|
| Merge with Fleet task: ships add per design, cargo and fuel into the target, which keeps its id | `loadPass`, `mergeTask`, `absorb` | CONFIRMED (FO-01..07) | `TestConfirmedMergeTask` |
| A target elsewhere refuses the task and clears it | `mergeTask` | CONFIRMED (FO) | `TestConfirmedMergeTask` |
| Where the task runs: both load passes, after the drops (before movement) and after the second research check (after movement), fleet order | `GenerateTurn`, `loadPass` | BINARY-ONLY (TAKEOVER.md) | `TestConfirmedMergeTask` |
| Damage on a merge (`ceil(100·ΣD/n)`; one damaged stack keeps its units; two divide by all ships) | `mergeDamage` (switch `legacyMergeDilution`, on) | LEGACY BUG, CONFIRMED (FO, seven cases) | `TestConfirmedMergeDamageDilution` (both settings) |
| No ship-count cap on the task; 32768 or more leaves no ships | `absorb` (switch `legacyMergeOverflow`, off) | LEGACY BUG, CONFIRMED (FO) | `TestConfirmedMergeTaskOverflow` (both settings) |
| Elegy's chosen rule: the task takes the merge order's cap | `absorb` | Elegy decision, below | `TestConfirmedMergeTaskOverflow` |
| Merge order: co-located fleets of the owner; a stack passing 32767 becomes 32766, the rest lost, the order not refused | `MergeFleets`, `absorb` | BINARY-ONLY (#51) | `TestPredictionMergeOrder` |
| Merge order damage: percentage over all ships, units over the damaged ships only (no dilution) | `MergeFleets`, `mergeDamage` | BINARY-ONLY (#51) | `TestPredictionMergeOrderDamage` |
| Task target gone, merged away or another player's: refused | `mergeTask` | BINARY-ONLY (#51) | `TestPredictionMergeTaskForeignTarget` |
| Ownership checked on every order (chosen rule) | `MergeFleets` | chosen rule (ORDERS.md "Ownership") | `TestPredictionMergeOrder` |
| Design read: parts above research tech dropped; mass and capacities from the parts kept | `ReadDesign` | BINARY-ONLY | `TestPredictionDesignTechStrip` |
| Hull not entitled: the original keeps it; Elegy's chosen rule rejects the design | `readDesign` | BINARY-ONLY, chosen rule (#51) | `TestPredictionDesignHullAndEngine` |
| An emptied engine slot is back-filled with the basic engine (Quick Jump 5) at the slot's capacity | `readDesign` | BINARY-ONLY (#51) | `TestPredictionDesignHullAndEngine` |
| Design read keeps parts the owner is not entitled to (Mystery Trader, race) | `readDesign` (switch `legacyKeepUnentitledParts`, off) | LEGACY BUG, CONFIRMED | `TestConfirmedDesignLegality` (both settings) |
| Elegy's chosen rule: drop every part the owner is not entitled to (tech, race, Mystery Trader items) | `ReadDesign` | chosen rule (ORDERS.md "Design legality") | `TestConfirmedDesignLegality` |

## Elegy decisions

- **Merge overflow.** The Merge with Fleet task's overflow empties a fleet
  of ships while keeping its cargo. Elegy gives the task the merge order's
  cap instead (32767 kept, anything above becomes 32766);
  `legacyMergeOverflow` reproduces the original.
- **Merge damage.** The dilution is reproduced (`legacyMergeDilution` is
  on); switched off, the units are divided by the damaged ships.
- **Design read.** ORDERS.md's chosen rule; `legacyKeepUnentitledParts`
  reproduces the original's keep-behavior.

## Implementation assumptions

Elegy's own choices where ORDERS.md is silent. They are **not**
established Stars! behavior; each is marked `ASSUMPTION On` in the code.

None open. O1 to O5 were answered by stars-elegy #51 and are cited rules
above.

## Not modelled

- order files and their acceptance, per-order clamps other than the
  design read, and conflicts between players;
- loads (direct or by task), cargo transfers between the player's own
  fleets, the "transfer fleet" task, cargo given to other players'
  fleets, and splits;
- Mystery Trader items: `ReadDesign` takes the owned items by part name,
  and no game state holds them yet. Designs in `Game` still have no
  owner, so the design read is not applied to them.
