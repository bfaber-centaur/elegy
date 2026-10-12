package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCatalogFileHash(t *testing.T) {
	// data/README.md records the copy's source and hash.
	readme, err := os.ReadFile("data/README.md")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(componentsJSON)
	if h := hex.EncodeToString(sum[:]); !strings.Contains(string(readme), h) {
		t.Errorf("components.json hash %s is not the one data/README.md records", h)
	}
}

func TestConfirmedCatalogCounts(t *testing.T) {
	// COMPONENTS.md "Status": rows per category, 239 in all.
	want := map[string]int{
		CatHull: 32, CatStarbaseHull: 5, CatEngine: 16, CatScanner: 16, CatShield: 10, CatArmor: 12,
		CatBeam: 24, CatTorpedo: 12, CatBomb: 15, CatMiningRobot: 8, CatMineLayer: 10, CatOrbital: 16,
		CatElectrical: 17, CatMechanical: 11, CatPlanetary: 15, CatTerraform: 20,
	}
	got := map[string]int{}
	cat := Components()
	for _, c := range cat.Components {
		got[c.Category]++
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("%s: %d rows, want %d", k, got[k], n)
		}
	}
	if len(cat.Components) != 239 {
		t.Errorf("%d rows, want 239", len(cat.Components))
	}
}

func TestCatalogStatsAccountedFor(t *testing.T) {
	// Every column is either used or listed as deferred, and every row
	// converts.
	hullStats := map[string]bool{"armor": true, "initiative": true, "fuel_capacity": true, "cargo_capacity": true,
		"slots": true, "fuel_transport": true, "dock_capacity": true, "mine_layer_multiplier": true}
	otherStats := map[string]bool{"kind": true, "range": true, "penetrating_range": true, "coverage_tenths_pct": true,
		"axis": true, "amount": true}
	for _, c := range Components().Components {
		for k := range c.Stats {
			var ok bool
			switch c.Category {
			case CatHull, CatStarbaseHull:
				ok = hullStats[k]
			case CatPlanetary, CatTerraform:
				ok = otherStats[k]
			default:
				ok = partStats[k] || deferredStats[k]
			}
			if !ok {
				t.Errorf("%s: column %q is not handled", c.Name, k)
			}
		}
		var err error
		switch c.Category {
		case CatHull, CatStarbaseHull:
			_, err = c.Hull()
		case CatPlanetary, CatTerraform:
		case CatEngine:
			if _, err = c.Part(); err == nil {
				_, err = c.Engine()
			}
		default:
			_, err = c.Part()
		}
		if err != nil {
			t.Error(err)
		}
		for _, code := range append(append([]string{}, c.Restriction.LRTRequired...), c.Restriction.LRTForbidden...) {
			if _, err := lrtHas(LRTs{}, code); err != nil {
				t.Errorf("%s: %v", c.Name, err)
			}
		}
		for _, code := range append(append([]string{}, c.Restriction.PRTOnly...), c.Restriction.PRTNot...) {
			found := false
			for _, v := range prtCodes {
				found = found || v == code
			}
			if !found {
				t.Errorf("%s: unknown primary trait %q", c.Name, code)
			}
		}
	}
}

func TestCatalogBattleWarp(t *testing.T) {
	// COMBAT.md's battle-warp rule, applied to the fuel tables (CS-002),
	// gives the table's battle_warp column for every engine.
	for _, c := range Components().Components {
		if c.Category != CatEngine {
			continue
		}
		e, err := c.Engine()
		if err != nil {
			t.Fatal(err)
		}
		if got := battleWarp(e); got != c.num("battle_warp") {
			t.Errorf("%s: battle warp %d, table %d", c.Name, got, c.num("battle_warp"))
		}
	}
}

func catalogPart(t *testing.T, name string) Part {
	t.Helper()
	c, ok := Components().Lookup(name)
	if !ok {
		t.Fatalf("no component %q", name)
	}
	p, err := c.Part()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfirmedPartValues(t *testing.T) {
	// COMBAT.md "Token values" factors and COMPONENTS.md values.
	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"Jammer 20 factor", catalogPart(t, "Jammer 20").Jammer, 80},
		{"Langston Shell factor", catalogPart(t, "Langston Shell").Jammer, 95},
		{"Mega Poly Shell factor", catalogPart(t, "Mega Poly Shell").Jammer, 80},
		{"Alien Miner factor", catalogPart(t, "Alien Miner").Jammer, 70},
		{"Multi Function Pod factor", catalogPart(t, "Multi Function Pod").Jammer, 90},
		{"Battle Nexus computer", catalogPart(t, "Battle Nexus").Computer, 50},
		{"Battle Nexus initiative", catalogPart(t, "Battle Nexus").Initiative, 3},
		{"Multi Contained Munition computer", catalogPart(t, "Multi Contained Munition").Computer, 10},
		{"Flux Capacitor", catalogPart(t, "Flux Capacitor").Capacitor, 20},
		{"Croby Sharmor armor", catalogPart(t, "Croby Sharmor").Armor, 65},
		{"Multi Cargo Pod armor", catalogPart(t, "Multi Cargo Pod").Armor, 50},
		{"Fielded Kelarium shield", catalogPart(t, "Fielded Kelarium").Shield, 50},
		{"Overthruster", catalogPart(t, "Overthruster").Thrust, 2},
		{"Rhino range", catalogPart(t, "Rhino Scanner").ScanRange, 50},
		{"Mega Poly Shell scanner", catalogPart(t, "Mega Poly Shell").ScanRange, 80},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if p := catalogPart(t, "Energy Dampener"); !p.Dampener || p.Thrust != 0 {
		t.Errorf("Energy Dampener %+v", p)
	}
	if p := catalogPart(t, "Robber Baron Scanner"); !p.CargoScan || !p.DetailedPlanetScan {
		t.Errorf("Robber Baron %+v", p)
	}
	if p := catalogPart(t, "Pick Pocket Scanner"); !p.CargoScan || p.DetailedPlanetScan {
		t.Errorf("Pick Pocket %+v", p)
	}
	if p := catalogPart(t, "Alien Miner"); !p.HalfThrust {
		t.Errorf("Alien Miner %+v", p)
	}
	if p := catalogPart(t, "Beam Deflector"); !p.Deflector {
		t.Errorf("Beam Deflector %+v", p)
	}
	if p := catalogPart(t, "Mini Gun"); !p.Gatling {
		t.Errorf("Mini Gun %+v", p)
	}
	if p := catalogPart(t, "Pulsed Sapper"); !p.Sapper {
		t.Errorf("Pulsed Sapper %+v", p)
	}
	if p := catalogPart(t, "Jihad Missile"); !p.Missile || p.Accuracy != 20 {
		t.Errorf("Jihad Missile %+v", p)
	}
	e, _ := Components().Lookup("Enigma Pulsar")
	if eng, _ := e.Engine(); !eng.EnigmaPulsar || !eng.BattleWarp10 {
		t.Errorf("Enigma Pulsar %+v", eng)
	}
	if p := catalogPart(t, "Enigma Pulsar"); p.CloakPoints != 20 || p.HalfThrust {
		t.Errorf("Enigma Pulsar part %+v", p)
	}
}

func TestConfirmedCatalogTokenValues(t *testing.T) {
	// COMBAT.md "Token values", CONFIRMED (CB-007, a Regenerating Shields
	// race): a Destroyer with 2 Tritanium has armor 250 (armor parts count
	// half); 2 Mole-skin give 70 shields.
	// CB-002 C4: one Flux and one Energy Capacitor give 132%.
	cat := Components()
	d, err := cat.NewDesign("D", "Destroyer", []SlotFill{{0, "Quick Jump 5", 1}, {4, "Tritanium", 2}})
	if err != nil {
		t.Fatal(err)
	}
	rs := Race{}
	rs.LRT.RegeneratingShields = true
	if tv := tokenValues(d, rs, false, Cost{}); tv.armor != 250 {
		t.Errorf("Destroyer + 2 Tritanium under RS: armor %d, want 250", tv.armor)
	}
	d, err = cat.NewDesign("M", "Small Freighter", []SlotFill{{0, "Quick Jump 5", 1}, {2, "Mole-skin Shield", 1}})
	if err != nil {
		t.Fatal(err)
	}
	d.Slots[1].Count = 2 // two Mole-skin, as CB-007 (shields only)
	if tv := tokenValues(d, rs, false, Cost{}); tv.shield != 70 {
		t.Errorf("2 Mole-skin under RS: shields %d, want 70", tv.shield)
	}
	d, err = cat.NewDesign("C", "Destroyer", []SlotFill{{0, "Quick Jump 5", 1}, {1, "Laser", 1},
		{3, "Flux Capacitor", 1}, {6, "Energy Capacitor", 1}})
	if err != nil {
		t.Fatal(err)
	}
	if tv := tokenValues(d, Race{}, false, Cost{}); tv.capacitor != 132 {
		t.Errorf("capacitor %d%%, want 132", tv.capacitor)
	}
}

func TestNewDesign(t *testing.T) {
	cat := Components()
	d, err := cat.NewDesign("Hauler", "Small Freighter", []SlotFill{
		{0, "Quick Jump 5", 1}, {1, "Fuel Tank", 1}, {2, "Tritanium", 1}})
	if err != nil {
		t.Fatal(err)
	}
	qj, _ := cat.Lookup("Quick Jump 5")
	ft, _ := cat.Lookup("Fuel Tank")
	tr, _ := cat.Lookup("Tritanium")
	if want := 25 + qj.Mass + ft.Mass + tr.Mass; d.Mass != want {
		t.Errorf("mass %d, want %d", d.Mass, want)
	}
	if d.FuelCapacity != 130+250 || d.CargoCapacity != 70 {
		t.Errorf("fuel %d cargo %d", d.FuelCapacity, d.CargoCapacity)
	}
	if d.Engines != 1 || d.Engine.Name != "Quick Jump 5" || d.Engine.Fuel[6] != 180 {
		t.Errorf("engine %+v × %d", d.Engine, d.Engines)
	}
	for _, c := range []struct {
		hull  string
		fills []SlotFill
	}{
		{"Small Freighter", []SlotFill{{1, "Fuel Tank", 1}}},                            // no engine
		{"Small Freighter", []SlotFill{{0, "Quick Jump 5", 1}, {1, "Laser", 1}}},        // wrong kind
		{"Small Freighter", []SlotFill{{0, "Quick Jump 5", 1}, {2, "Tritanium", 2}}},    // over max
		{"Small Freighter", []SlotFill{{0, "Quick Jump 5", 1}, {0, "Quick Jump 5", 1}}}, // twice
		{"Large Freighter", []SlotFill{{0, "Quick Jump 5", 1}}},                         // needs 2
		{"Small Freighter", []SlotFill{{0, "Quick Jump 5", 1}, {3, "Tritanium", 1}}},    // no slot 3
		{"Nonesuch", nil},
	} {
		if _, err := cat.NewDesign("x", c.hull, c.fills); err == nil {
			t.Errorf("%s %v: no error", c.hull, c.fills)
		}
	}
	sb, err := cat.NewDesign("Fort", "Orbital Fort", []SlotFill{{1, "Laser", 2}})
	if err != nil || !sb.Hull.Starbase || sb.Hull.StarbaseNumber != 1 || sb.Hull.Dock != 0 {
		t.Errorf("Orbital Fort %+v %v", sb.Hull, err)
	}
	if h, _ := cat.Lookup("Space Station"); func() bool { hh, _ := h.Hull(); return hh.Dock != DockUnlimited }() {
		t.Errorf("Space Station dock should be unlimited")
	}
	if h, _ := cat.Lookup("Frigate"); func() bool { hh, _ := h.Hull(); return !hh.JOATScanner }() {
		t.Errorf("Frigate should carry the JOAT scanner")
	}
}

func TestConfirmedStarbaseBuildCost(t *testing.T) {
	// COMPONENTS.md "Starbases" vectors at tech 26 (CONFIRMED CS-001).
	var tech26 [NumFields]int
	for f := range tech26 {
		tech26[f] = 26
	}
	ar := Race{PRT: PRTAlternateReality}
	ss := Race{PRT: PRTSuperStealth}
	ss.LRT.ImprovedStarbases = true
	he := Race{PRT: PRTHyperExpansion}
	cat := Components()
	for _, c := range []struct {
		race         Race
		hull         string
		owner, shown Cost
	}{
		{ar, "Death Star", Cost{960, Minerals{154, 102, 448}}, Cost{384, Minerals{62, 41, 180}}},
		{ss, "Space Station", Cost{300, Minerals{60, 40, 125}}, Cost{120, Minerals{24, 16, 50}}},
		{ss, "Space Dock", Cost{50, Minerals{10, 2, 12}}, Cost{20, Minerals{4, 1, 5}}},
		{he, "Space Station", Cost{300, Minerals{60, 40, 125}}, Cost{150, Minerals{30, 20, 63}}},
	} {
		h, _ := cat.Lookup(c.hull)
		if got := h.OwnerCost(c.race, tech26); got != c.owner {
			t.Errorf("%s owner cost %+v, want %+v", c.hull, got, c.owner)
		}
		d, err := cat.NewDesign(c.hull, c.hull, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := StarbaseBuildCost(d, c.race, tech26); got != c.shown {
			t.Errorf("%s build cost %+v, want %+v", c.hull, got, c.shown)
		}
	}
}

func TestConfirmedOwnerCostExemptions(t *testing.T) {
	// COMPONENTS.md "Cost for an owner": terraform and planetary items skip
	// miniaturization and are never doubled by BET, except the Genesis
	// Device; Claim Adjuster pays half the resources for terraform items.
	cat := Components()
	bet := Race{}
	bet.LRT.BleedingEdgeTech = true
	var tech0, tech26 [NumFields]int
	for f := range tech26 {
		tech26[f] = 26
	}
	for _, name := range []string{"Snooper 620X", "Neutron Shield"} {
		c, _ := cat.Lookup(name)
		if got := c.OwnerCost(bet, c.TechReq); got != c.Cost {
			t.Errorf("BET %s at its requirement: %+v, want base %+v", name, got, c.Cost)
		}
		if got := c.OwnerCost(Race{}, tech26); got != c.Cost {
			t.Errorf("%s at tech 26: %+v, want base %+v (no miniaturization)", name, got, c.Cost)
		}
	}
	for _, c := range cat.Components {
		if c.Category != CatTerraform {
			continue
		}
		got := c.OwnerCost(Race{PRT: PRTClaimAdjuster}, tech0)
		want := c.Cost
		want.Resources /= 2
		if got != want {
			t.Errorf("CA %s: %+v, want %+v", c.Name, got, want)
		}
	}
	// The Genesis Device is miniaturized and doubled like a ship part
	// (CS-001 readouts, stars-elegy experiments/cs001: c00 5000 at tech 0,
	// c01 3800 at tech 26, c05 3500 with BET at tech 26, c11 10000 with BET
	// at tech 0). GAP-001 G2–G4 bound BET at tech 26 to 3,445–3,567.
	genesis, _ := cat.Lookup("Genesis Device")
	for _, c := range []struct {
		race  Race
		tech  [NumFields]int
		want  int
		label string
	}{
		{Race{}, tech0, 5000, "c00"},
		{Race{}, tech26, 3800, "c01"},
		{bet, tech26, 3500, "c05"},
		{bet, tech0, 10000, "c11"},
	} {
		if got := genesis.OwnerCost(c.race, c.tech); got != (Cost{Resources: c.want}) {
			t.Errorf("%s Genesis Device %+v, want %d resources", c.label, got, c.want)
		}
	}
	// A ship part with a requirement, at that requirement, is doubled
	// under BET.
	xray, _ := cat.Lookup("X-Ray Laser")
	if got := xray.OwnerCost(bet, xray.TechReq); got.Resources != 2*xray.Cost.Resources {
		t.Errorf("BET X-Ray Laser %+v, base %+v", got, xray.Cost)
	}
}

func TestConfirmedWhoCanBuild(t *testing.T) {
	// COMPONENTS.md "Who can build what" and "Oddities" (CONFIRMED CS-001).
	cat := Components()
	allowed := func(name string, r Race, owns bool) bool {
		c, ok := cat.Lookup(name)
		if !ok {
			t.Fatalf("no %q", name)
		}
		ok, err := c.Allowed(r, owns)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	is := Race{PRT: PRTInnerStrength}
	nas := Race{}
	nas.LRT.NoAdvancedScanners = true
	isb := Race{}
	isb.LRT.ImprovedStarbases = true
	for _, c := range []struct {
		name  string
		race  Race
		owns  bool
		allow bool
	}{
		{"Super Freighter", is, false, true},
		{"Super Freighter", Race{}, false, false},
		{"Smart Bomb", is, false, false},
		{"Smart Bomb", Race{}, false, true},
		{"Ferret Scanner", nas, false, false},
		{"Snooper 320X", nas, false, false},
		{"Scoper 280", nas, false, true},
		{"Space Dock", isb, false, true},
		{"Space Dock", Race{}, false, false},
		{"Multi Cargo Pod", Race{}, false, false},
		{"Multi Cargo Pod", Race{}, true, true},
	} {
		if got := allowed(c.name, c.race, c.owns); got != c.allow {
			t.Errorf("%s for %+v (trader item owned %v): %v, want %v", c.name, c.race.PRT, c.owns, got, c.allow)
		}
	}
	snooper, _ := cat.Lookup("Snooper 320X")
	low := snooper.TechReq
	low[Electronics]--
	if ok, _ := snooper.Buildable(Race{}, low, false); ok {
		t.Errorf("Snooper 320X buildable below its requirement")
	}
	if ok, _ := snooper.Buildable(Race{}, snooper.TechReq, false); !ok {
		t.Errorf("Snooper 320X not buildable at its requirement")
	}
}

func TestConfirmedPlanetaryCatalogue(t *testing.T) {
	// Penetrating planetary scanners have P = R/2, as SCANNING.md's planet
	// rule uses; defense coverage vectors, 10 defenses (CS-001).
	cat := Components()
	scanners := cat.PlanetScanners()
	if len(scanners) != 9 {
		t.Errorf("%d planetary scanners, want 9", len(scanners))
	}
	for _, s := range scanners {
		c, _ := cat.Lookup(s.Name)
		if p := c.num("penetrating_range"); s.Penetrating != (p > 0) || (p > 0 && p != s.Range/2) {
			t.Errorf("%s: range %d, penetrating %d", s.Name, s.Range, p)
		}
	}
	want := map[string]float64{"SDI": 9.56, "Missile Battery": 18.29, "Laser Battery": 21.56,
		"Planetary Shield": 26.25, "Neutron Shield": 32.11}
	defs := cat.Defenses()
	if len(defs) != len(want) {
		t.Errorf("%d defense types", len(defs))
	}
	for _, d := range defs {
		shown := math.Floor(DefenseCoverage(10, d.Coverage)*10000) / 100
		if math.Abs(shown-want[d.Name]) > 1e-9 {
			t.Errorf("%s: %.2f%%, want %.2f%%", d.Name, shown, want[d.Name])
		}
	}
}

// TestMisslottedPartThisYear: a part in a hull slot that does not take
// its kind counts in the design's cost and in the battle token's armor in
// the year that reads the design, and in nothing else computed from the
// parts; at the end of the year it leaves the stored design (COMBAT.md
// "Parts the game drops from a design"; armor BINARY-ONLY).
func TestMisslottedPartThisYear(t *testing.T) {
	cat := Components()
	fills := []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 2, Part: "Laser", Count: 1}}
	bare, err := cat.NewDesign("bare", "Frigate", fills)
	if err != nil {
		t.Fatal(err)
	}
	d, err := cat.NewDesignAnyKind("armored", "Frigate", append(fills, SlotFill{Slot: 1, Part: "Tritanium", Count: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if d.Mass != bare.Mass || d.FuelCapacity != bare.FuelCapacity || d.CargoCapacity != bare.CargoCapacity {
		t.Errorf("mass, fuel, cargo %d %d %d, want the design without the part's %d %d %d", d.Mass, d.FuelCapacity, d.CargoCapacity, bare.Mass, bare.FuelCapacity, bare.CargoCapacity)
	}
	var lv [NumFields]int
	if designCost(d, Race{}, lv) == designCost(bare, Race{}, lv) {
		t.Error("this year's cost leaves the part out")
	}
	plan := []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies}}
	g := &Game{
		Rules: ElegyRules(),
		Players: []Player{
			{Plans: plan, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Plans: plan, Relations: []Relation{RelationEnemy, RelationFriend}},
		},
		Designs: []Design{d, bare},
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 2, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 1}}},
		},
	}
	loc := g.locations()[0]
	sets, inv, n := g.whoFights(loc, locationHistory{})
	b := &battle{g: g, rng: &seqRand{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
	b.setup(map[int]bool{})
	want, without := tokenValues(d, Race{}, false, Cost{}).armor, tokenValues(bare, Race{}, false, Cost{}).armor
	if want <= without {
		t.Fatalf("Tritanium adds no armor: %d, %d", want, without)
	}
	for _, tk := range b.tokens {
		if tk.player == 0 && (tk.armor != want || tk.mass != bare.Mass) {
			t.Errorf("token armor %d, mass %d; want armor %d with the Tritanium, mass %d without", tk.armor, tk.mass, want, bare.Mass)
		}
	}
	g.dropMisslottedParts()
	if got := g.Designs[0]; len(got.Slots) != 2 || len(got.SlotPos) != 2 || designCost(got, Race{}, lv) != designCost(bare, Race{}, lv) || got.Mass != bare.Mass {
		t.Errorf("after the year: %d slots, cost %+v; want the design without the part, cost %+v", len(got.Slots), designCost(got, Race{}, lv), designCost(bare, Race{}, lv))
	}
	if got := g.Designs[1]; len(got.Slots) != len(bare.Slots) {
		t.Errorf("a valid design lost a part: %+v", got)
	}
}

// TestMisslottedPartWorksNowhere: in the year that reads the design, a
// misslotted part counts in the cost and the armor but adds nothing to
// any other use of the parts: scanning, cloaking, weapons, score power
// and tech requirements read Parts, which leaves it out (COMBAT.md
// "Parts the game drops from a design", BINARY-ONLY).
func TestMisslottedPartWorksNowhere(t *testing.T) {
	cat := Components()
	fills := []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}}
	bare, err := cat.NewDesign("bare", "Frigate", fills)
	if err != nil {
		t.Fatal(err)
	}
	// The Frigate's slot 1 takes scanners, slot 3 shields and armor.
	d, err := cat.NewDesignAnyKind("odd", "Frigate", append(slices.Clone(fills),
		SlotFill{Slot: 1, Part: "Laser", Count: 1},
		SlotFill{Slot: 3, Part: "Bat Scanner", Count: 1},
	))
	if err != nil {
		t.Fatal(err)
	}
	cloak, err := cat.NewDesignAnyKind("cloak", "Frigate", append(slices.Clone(fills), SlotFill{Slot: 1, Part: "Transport Cloaking", Count: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if !d.misslotted(1) || !d.misslotted(2) || d.misslotted(0) || !cloak.misslotted(1) {
		t.Fatalf("misslotted parts: %+v at %v", d.Slots, d.SlotPos)
	}
	if designCloakPoints(cloak) != 0 {
		t.Errorf("cloak points %d, want 0", designCloakPoints(cloak))
	}
	if got := d.Parts(); !reflect.DeepEqual(got, bare.Slots) {
		t.Errorf("Parts %+v, want %+v", got, bare.Slots)
	}
	if got := bare.Parts(); !reflect.DeepEqual(got, bare.Slots) {
		t.Errorf("a valid design's Parts %+v, want its Slots", got)
	}
	g := &Game{Players: []Player{{}}}
	if _, _, any := g.designScan(d, 0); any {
		t.Error("the misslotted scanner scans")
	}
	if d.armed() {
		t.Error("the misslotted Laser arms the design")
	}
	if shipPower(d) != shipPower(bare) {
		t.Errorf("power %d, want %d", shipPower(d), shipPower(bare))
	}
	if d.techReq() != bare.techReq() {
		t.Errorf("tech %v, want %v", d.techReq(), bare.techReq())
	}
	var lv [NumFields]int
	if designCost(d, Race{}, lv) == designCost(bare, Race{}, lv) {
		t.Error("the cost leaves the parts out")
	}
}

// TestConfirmedDroppedPartTargetChoice: target choice uses the design
// cost, which counts a misslotted part in the year that reads the design
// and leaves it out from the next (COMBAT.md "Parts the game drops from a
// design": CONFIRMED for salvage, CB-054, and for target choice, CB-055,
// stars-elegy b1e38ef, 18 of 20 battles). Written as a prediction before
// CB-055 ran (#160, e35c871); CB-055 used Colloidal Phaser frigates and
// 15-ship stacks, with the same freighter costs. A
// Small Freighter with a Beam Deflector in its shield-or-armor slot costs
// 8 (resources + boranium) that year and 6 after; one with a Fuel Tank
// costs 7, with the same armor and no shield. A Laser frigate fires at the
// deflector freighter first in the first year and at the Fuel Tank
// freighter first in the next.
func TestConfirmedDroppedPartTargetChoice(t *testing.T) {
	cat := Components()
	qj := []SlotFill{{Slot: 0, Part: "Quick Jump 5", Count: 1}}
	odd, err := cat.NewDesignAnyKind("deflector", "Small Freighter", append(slices.Clone(qj), SlotFill{Slot: 2, Part: "Beam Deflector", Count: 1}))
	if err != nil {
		t.Fatal(err)
	}
	tank, err := cat.NewDesign("tank", "Small Freighter", append(slices.Clone(qj), SlotFill{Slot: 1, Part: "Fuel Tank", Count: 1}))
	if err != nil {
		t.Fatal(err)
	}
	gun, err := cat.NewDesign("gun", "Frigate", []SlotFill{{Slot: 0, Part: "Long Hump 6", Count: 1}, {Slot: 2, Part: "Laser", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	var lv [NumFields]int
	for f := range lv {
		lv[f] = 26
	}
	plan := []BattlePlan{{Tactic: TacticMaximizeDamage, Primary: TargetAny, Attack: AttackEnemies}}
	g := &Game{
		Rules: ElegyRules(),
		Players: []Player{
			{Plans: plan, Relations: []Relation{RelationFriend, RelationEnemy}},
			{Plans: plan, Relations: []Relation{RelationEnemy, RelationFriend}, Research: ResearchState{Levels: lv}},
		},
		Designs: []Design{gun, odd, tank},
		Fleets: []Fleet{
			{ID: 1, Owner: 0, Pos: Point{5, 5}, Stacks: []Stack{{Design: 0, Count: 1}}},
			{ID: 2, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 1, Count: 1}}},
			{ID: 3, Owner: 1, Pos: Point{5, 5}, Stacks: []Stack{{Design: 2, Count: 1}}},
		},
	}
	first := func() (int, [3]int) {
		gg := *g
		gg.Designs = slices.Clone(g.Designs)
		gg.Fleets = slices.Clone(g.Fleets)
		for i := range gg.Fleets {
			gg.Fleets[i].Stacks = slices.Clone(gg.Fleets[i].Stacks)
		}
		loc := gg.locations()[0]
		sets, inv, n := gg.whoFights(loc, locationHistory{})
		b := &battle{g: &gg, rng: pvNoDraws{}, loc: loc, sets: sets, players: inv, involved: n, killed: map[int]bool{}}
		b.setup(map[int]bool{})
		var cost [3]int
		for _, tk := range b.tokens {
			cost[tk.design] = tk.cost
			tk.x, tk.y = 0, 0 // all in range
		}
		b.fire()
		if len(b.hits) == 0 {
			t.Fatal("no hit")
		}
		return b.tokens[b.hits[0].Target].design, cost
	}
	if d, c := first(); d != 1 || c[1] != 8 || c[2] != 7 {
		t.Errorf("first year: first target design %d, costs %v; want the deflector freighter (1) at 8 over the tank at 7", d, c)
	}
	g.dropMisslottedParts()
	if d, c := first(); d != 2 || c[1] != 6 || c[2] != 7 {
		t.Errorf("next year: first target design %d, costs %v; want the tank freighter (2) at 7 over the deflector freighter at 6", d, c)
	}
}
