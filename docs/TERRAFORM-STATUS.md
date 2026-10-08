# Terraforming and remote mining: implementation status

`terraform/` implements the planet-side rules of stars-elegy (as of
`main` at `0779d2f`) that change a planet's environment or deposits
outside the production loop's own items:

- `docs/KERNEL.md` "Terraforming" and "Remote mining";
- `docs/OBJECTS.md` "Impact", PP terraforming;
- `docs/COMPONENTS.md` "Remote mining".

Nothing here comes from the private archaeology repositories. The package
reads and changes the engine's exported state and exposes functions; the
turn engine decides when they run (KERNEL.md "Turn order"). The call
sites, all owned by the kernel lane, are under "Turn wiring" below.

## Terraforming

| Rule (KERNEL.md "Terraforming") | Function | Status |
|---|---|---|
| Reach per axis: largest available part, Total Terraform for every axis, immune axis 0 | `Reach` | CONFIRMED (KX-002 T1–T3, KX-005, KB-2C) |
| Limit: toward the centre within orig ± reach, 1–99, stopping at the centre | `Limit` | CONFIRMED (KX-002 T1, T3) |
| Capacity: sum of the distances to the limits, immune axes excluded | `Capacity` | CONFIRMED (KX-002 T1–T3, KB-2C) |
| Order clipped to the capacity with a message, removed at 0 | `ClipOrder` | CONFIRMED (KX-002 T1, T3, KB-2C) |
| Axis choice: score `trunc(|Δhab|·100/|Δ|) + 1` toward the limit, first axis on ties | `Improve` | CONFIRMED at one point (KX-002 T2: 101 against 67) and the tie (KX-005) |
| Unit cost 100, 70 with TT, halved for CA; no minerals | `UnitCost` | CONFIRMED (KX-002 T1, T2, KX-005); CA with TT is ASSUMPTION T2 |
| Auto Max / Auto Min Terraform | `AutoUnits` | CONFIRMED (KX-005) |
| Orbital Adjusters: one click per adjuster, fleet owner's reach, planet owner's habitat | `AdjusterClicks`, `Adjust` | CONFIRMED (KX-005 T0–T2) |
| Friendly adjusters improve, starbase or not | `Adjust` → `Improve` | CONFIRMED (KX-005) |
| Neutral or enemy adjusters: nothing at a starbase, else worsen toward the far end of orig ± reach | `Adjust` → `Worsen` | CONFIRMED (KX-005: 60/60/60 → 62/60/60) |
| Claim Adjuster drift and year-end step | engine `claimAdjusterYearEnd` | the kernel's (KERNEL-STATUS.md) |

`Reach` and `Limit` restate the engine's unexported `terraformReach` and
`terraformLimit` from the same spec text; the kernel may switch to these.

## PP packet terraforming

| Rule (OBJECTS.md "Impact") | Function | Status |
|---|---|---|
| Uncaught share `⌊m·(1000 − q)/1000⌋` per mineral | `Uncaught` | BINARY-ONLY |
| Ironium → gravity, boranium → temperature, germanium → radiation; toward the PP player's ideal; original value unchanged without a permanent success | `PacketTerraform` | CONFIRMED (OB-029-T1..T3) |
| Per 100 kT chunk `rand(200) < min(chunk, 100)`, permanent on `rand(10) == 0`; permanent moves the original value; successes move the current value within reach | `PacketTerraform` | BINARY-ONLY |
| On owned and unowned planets, before the damage | objects impact (`Impact.Terraform`) | CONFIRMED (OB-029-T1..T3) for owned and unowned; the order is the spec's step order |
| Catcher's starbase design known to the PP player | `Impact.DiscloseDesign` | BINARY-ONLY; not wired |

## Remote mining

| Rule (KERNEL.md "Remote mining") | Function | Status |
|---|---|---|
| Rate: Σ count × mining_rate, at most 4,000 | `MiningRate` | CONFIRMED (CS-003-B X1/X2, T-35, KB-1A) |
| Output `conc·m/100`, random +1, depletion, no homeworld floor, race output ignored | `RemoteMine` (engine `MineYear`, eff 10) | CONFIRMED (T-35, CS-003-B, KB-1A); the +1 draw BINARY-ONLY |
| Unowned planets only, except the owner's own miners at an Alternate Reality planet | `CanRemoteMine` | CONFIRMED (T-35, KB-1B) |
| AR: a separate mining step from the planet's own | `RemoteMine` | CONFIRMED (KB-1B); the order of the two steps BINARY-ONLY |
| A fleet that arrived or moved this year mines nothing | the caller | CONFIRMED (T-35) |

## Assumptions (spec gaps)

- **T1** A neutral or enemy adjuster never worsens an axis the planet
  owner is immune to.
- **T2** A Claim Adjuster with Total Terraforming pays ⌊70/2⌋ = 35 per
  unit.
- **T3** Adjusting fleets act one at a time in fleet order (owner, then
  number), each on the state the previous left.
- **P1** PP terraforming draws and applies mineral by mineral (ironium,
  boranium, germanium), each mineral's chunks in order.
- **P2** "The PP player's terraforming tech could improve the planet" is
  read per axis: the current value moves only when that axis's `Limit`
  (PP reach, new original value) differs from it, and at most to it.
- **P3** The extremes are 1 and 99; the nearer one is nearer to the value
  being moved, 1 at 50; half the count rounds down; the immune-axis move
  is not gated by P2.
- **R1** Another player's miners at an Alternate Reality planet
  (BINARY-ONLY) mine nothing.

## Turn wiring (kernel call sites)

| Turn step | Call | Notes |
|---|---|---|
| Production: Terraform Environment (planetary item 12), Auto Max / Auto Min Terraform | `UnitCost(owner race)` per unit, no minerals; when the queue reaches the item `ClipOrder(count, Capacity(p, owner race, Reach(owner race, levels before research)))`; each completed unit `Improve(p, owner race, reach)`; auto items build `AutoUnits(count, capacity, min, population change, Habitability)` | Needs the item kinds in `engine.ItemKind` and the harness's `pvQueue` (KX-002 T1–T3 skip on "planetary item 12") |
| 3, packet impacts | none: `hit` calls `PacketTerraform` itself | Messages for a terraformed planet and `Impact.DiscloseDesign` are the kernel's |
| 6c, remote mining | for each fleet in fleet order whose task is remote mining and that did not move this year: `RemoteMine(g, fi, rng)` | Needs a remote-mining task kind and the harness's `remote_mine` task (all 16 CS-003-B cases skip on it; X2 then checks `RemoteMine`) |
| 7.4, Orbital Adjusters | `Adjust(g)` after `claimAdjusterYearEnd`, after research | Fleet owner told per `Adjustment`; planet owner when `HabChanged` |

The parity corpus has no OB-029 or KX-005 vector yet; their rules are
tested from the spec's worked examples in `terraform/` tests.
