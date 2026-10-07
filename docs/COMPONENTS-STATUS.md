# Components status

`engine/catalog.go` loads the component table: every J-RC3 hull, starbase
hull, ship part and planetary item. The table is a verbatim copy of
stars-elegy `data/components.json` (`engine/data/README.md` gives the
source commit and hash). The rules follow stars-elegy `docs/COMPONENTS.md`
on `main` at `96fd2e7`. Nothing else was used.

Each row keeps its status (CONFIRMED or BINARY-ONLY, with the
BINARY-ONLY columns) from the table.

## What the engine takes from the table

| Rule (COMPONENTS.md section) | Code | Status | Test |
|---|---|---|---|
| Rows and categories | `ParseCatalog`, `Components` | CONFIRMED (239 rows) | `TestConfirmedCatalogCounts`, `TestCatalogFileHash` |
| Part values (category values, values carried outside the category) | `Component.Part` | per row | `TestConfirmedPartValues`, `TestCatalogStatsAccountedFor` |
| Token values from catalogue parts (COMBAT.md) | `tokenValues` | CONFIRMED (CB-002, CB-007) | `TestConfirmedCatalogTokenValues` |
| Engine fuel tables, rated engines | `Component.Engine` | CONFIRMED (CS-002); `warp10_rated`, `battle_warp` BINARY-ONLY | `TestCatalogBattleWarp`, the FM movement corpus |
| Hull and starbase hull values, slots, docks | `Component.Hull` | CONFIRMED | `TestNewDesign` |
| Designs from a hull and filled slots | `Catalog.NewDesign` | see below | `TestNewDesign` |
| Who can build what, Mystery Trader items | `Component.Allowed`, `Buildable` | CONFIRMED (CS-001) | `TestConfirmedWhoCanBuild` |
| Cost for an owner, terraform/planetary exemption, CA | `Component.OwnerCost`, `itemCost` | CONFIRMED (CS-001) | `TestConfirmedOwnerCostExemptions`, `TestConfirmedStarbaseBuildCost` |
| Starbase build cost (ISB/AR, halved) | `StarbaseBuildCost` | CONFIRMED (CS-001 designer) | `TestConfirmedStarbaseBuildCost` |
| Planetary scanners and defenses | `Catalog.PlanetScanners`, `Defenses` | CONFIRMED | `TestConfirmedPlanetaryCatalogue` |
| Defense coverage `1 − (1 − c/1000)^n` | `DefenseCoverage` | CONFIRMED (CS-001) | `TestConfirmedPlanetaryCatalogue` |

Values that come from other specs, not from a table column:

- JOAT built-in scanners on the Scout, Frigate and Destroyer hulls
  (SCANNING.md "JOAT hulls").
- The repair bonus `f` of the Fuel Transport (25) and Super-Fuel Xport
  (50) hulls (COMBAT.md "Repair").
- The jammer factor `f = 100 − jammer %` and the deflector's `× 90/100`
  (COMBAT.md "Token values").

## Readings of the table

These follow from what the columns mean rather than from a separate rule:

- A design's mass is the hull's mass plus every part's mass. This is
  consistent with COMBAT.md's Energy Dampener note: the battle record's
  mass matched the part-table sum.
  Its fuel and cargo capacity are the hull's plus those of its parts
  (fuel tanks, the Anti-matter Generator, cargo pods).
- A hull slot holds one part type, at most its `max`. A ship hull's
  engine slot must hold exactly `max` engines ("needs N").

## Not used yet

The table carries columns for rules Elegy does not have yet. They are
listed in `deferredStats` and kept in `Component.Stats`:

- mine sweeping, laying and fields;
- remote mining and the Orbital Adjuster;
- bombs and colonizing modules (TAKEOVER.md);
- stargates, mass drivers and jump gates;
- the Anti-matter Generator's fuel per year;
- the mine-layer hull multiplier.

`GenerateTurn` still takes designs, planetary scanners and defenses as
data. A game set up from the catalogue gets them from `NewDesign`,
`PlanetScanners` and `Defenses`.
