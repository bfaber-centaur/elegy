package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// Scanning tests follow stars-elegy docs/SCANNING.md statuses (see
// kernel_test.go). TestConfirmed* vectors come from the SC-001..SC-023
// records in PARITY.md "Scanning". Part ranges are the recorded ones
// (Rhino 50, Mole 100, Bat 0; penetrating Ferret 50, Chameleon 45,
// Dolphin 100, Robber Baron 120, Elephant 200). Where a test needs a
// normal range the record does not state, it uses the penetrating range.

func scanPart(name string, r, p int) Part {
	return Part{Name: name, Kind: PartElectrical, Scanner: true, ScanRange: r, PenRange: p}
}

var (
	sRhino    = scanPart("Rhino Scanner", 50, 0)
	sMole     = scanPart("Mole Scanner", 100, 0)
	sBat      = scanPart("Bat Scanner", 0, 0)
	sFerret   = scanPart("Ferret Scanner", 50, 50)
	sChamel   = scanPart("Chameleon Scanner", 45, 45)
	sDolphin  = scanPart("Dolphin Scanner", 100, 100)
	sRobber   = Part{Name: "Robber Baron Scanner", Kind: PartElectrical, Scanner: true, ScanRange: 120, PenRange: 120, CargoScan: true, DetailedPlanetScan: true}
	sElephant = scanPart("Elephant Scanner", 200, 200)
	sPossum   = scanPart("Possum Scanner", 150, 0) // 150 reproduces the recorded 214
	sPick     = Part{Name: "Pick Pocket Scanner", Kind: PartElectrical, Scanner: true, CargoScan: true}
	sStealth  = Part{Name: "Stealth Cloak", Kind: PartElectrical, CloakPoints: 70}
	sTachyon  = Part{Name: "Tachyon Detector", Kind: PartElectrical, Tachyon: true}
	hFreight  = Hull{Name: "Small Freighter", Mass: 25}
)

// scanLab is a two-player game (both at tech 26, no planet scanners):
// player 0 views, player 1 is the target. Designs are added by sd.
type scanLab struct{ g Game }

func newScanLab() *scanLab {
	l := &scanLab{g: Game{Rules: ElegyRules(), Players: make([]Player, 2)}}
	for p := range l.g.Players {
		for f := range NumFields {
			l.g.Players[p].Research.Levels[f] = MaxTechLevel
		}
	}
	return l
}

// design adds a design of mass m with the given slots and returns its index.
func (l *scanLab) design(h Hull, m int, slots ...Slot) int {
	l.g.Designs = append(l.g.Designs, Design{Name: h.Name, Hull: h, Mass: m, Slots: slots})
	return len(l.g.Designs) - 1
}

func (l *scanLab) fleet(owner int, pos Point, stacks ...Stack) int {
	id := len(l.g.Fleets) + 1
	l.g.Fleets = append(l.g.Fleets, Fleet{ID: id, Owner: owner, Pos: pos, Stacks: stacks})
	return id
}

func (l *scanLab) planet(owner int, pos Point) int {
	id := len(l.g.Planets) + 1
	l.g.Planets = append(l.g.Planets, Planet{ID: id, Owner: owner, Pos: pos})
	return id
}

func (l *scanLab) view(v int) PlayerView { return Views(l.g, nil)[v] }

func (l *scanLab) seesFleet(v, id int) bool {
	for _, f := range l.view(v).Fleets {
		if f.Fleet == id {
			return true
		}
	}
	return false
}

func (l *scanLab) report(v, id int) PlanetReport {
	for _, r := range l.view(v).Planets {
		if r.Planet == id {
			return r
		}
	}
	return PlanetReport{Planet: id}
}

func TestConfirmedScanRangeEdges(t *testing.T) {
	// SC-001, SC-008 (S-1): d² ≤ R². Rhino 50: 2500 seen, 2501 not; Mole
	// 100: 10000 / 10001; Bat 0: nothing at 1 ly.
	for _, c := range []struct {
		part Part
		at   Point
		seen bool
	}{
		{sRhino, Point{50, 0}, true}, {sRhino, Point{50, 1}, false},
		{sMole, Point{100, 0}, true}, {sMole, Point{100, 1}, false},
		{sBat, Point{1, 0}, false},
	} {
		l := newScanLab()
		v := l.design(hFreight, 25, Slot{c.part, 1})
		tg := l.design(hFreight, 25)
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		id := l.fleet(1, c.at, Stack{Design: tg, Count: 1})
		if got := l.seesFleet(0, id); got != c.seen {
			t.Errorf("%s at %v: seen %v, want %v", c.part.Name, c.at, got, c.seen)
		}
	}
}

func TestConfirmedCombinedScanners(t *testing.T) {
	// SC-001 (S-2, S-3): two Rhinos on one design reach 59 (d² 3481, not
	// 3482); two Rhino ships, or two Rhino designs, in one fleet still
	// scan 50 (a target at 55 ly is missed).
	l := newScanLab()
	two := l.design(hFreight, 25, Slot{sRhino, 2})
	one := l.design(hFreight, 25, Slot{sRhino, 1})
	one2 := l.design(hFreight, 25, Slot{sRhino, 1})
	tg := l.design(hFreight, 25)
	l.fleet(0, Point{}, Stack{Design: two, Count: 1})
	in := l.fleet(1, Point{59, 0}, Stack{Design: tg, Count: 1})
	out := l.fleet(1, Point{59, 1}, Stack{Design: tg, Count: 1})
	if !l.seesFleet(0, in) || l.seesFleet(0, out) {
		t.Error("two Rhinos: want d² 3481 seen, 3482 not")
	}
	for _, stacks := range [][]Stack{{{Design: one, Count: 2}}, {{Design: one, Count: 1}, {Design: one2, Count: 1}}} {
		l.g.Fleets = nil
		l.fleet(0, Point{}, stacks...)
		far := l.fleet(1, Point{55, 0}, Stack{Design: tg, Count: 1})
		if l.seesFleet(0, far) {
			t.Errorf("fleet %v saw a target at 55 ly", stacks)
		}
	}
}

func TestConfirmedJOATHullScanner(t *testing.T) {
	// SC-020, SC-022, SC-023 (S-10): a JOAT Scout scans 20·E / 10·E and
	// combines with scanner parts.
	scout := Hull{Name: "Scout", Mass: 8, JOATScanner: true}
	setup := func(e int, slots ...Slot) *scanLab {
		l := newScanLab()
		l.g.Players[0].Race.PRT = PRTJackOfAllTrades
		l.g.Players[0].Research.Levels[Electronics] = e
		d := l.design(scout, 8, slots...)
		l.fleet(0, Point{}, Stack{Design: d, Count: 1})
		l.design(hFreight, 25)
		return l
	}
	l := setup(10)
	in, out := l.fleet(1, Point{200, 0}, Stack{Design: 1, Count: 1}), l.fleet(1, Point{200, 1}, Stack{Design: 1, Count: 1})
	pin, pout := l.planet(NoOwner, Point{0, 100}), l.planet(NoOwner, Point{1, 100})
	if !l.seesFleet(0, in) || l.seesFleet(0, out) || l.report(0, pin).Level != ReportNormal || l.report(0, pout).Level != ReportNone {
		t.Error("E 10 Scout: want 200 / 100")
	}
	l = setup(10, Slot{sPossum, 1})
	in, out = l.fleet(1, Point{214, 0}, Stack{Design: 1, Count: 1}), l.fleet(1, Point{214, 1}, Stack{Design: 1, Count: 1})
	if !l.seesFleet(0, in) || l.seesFleet(0, out) {
		t.Error("E 10 Scout + Possum: want 214")
	}
	l = setup(16, Slot{sElephant, 1})
	pin, pout = l.planet(NoOwner, Point{217, 0}), l.planet(NoOwner, Point{217, 1})
	if l.report(0, pin).Level != ReportNormal || l.report(0, pout).Level != ReportNone {
		t.Error("E 16 Scout + Elephant: want penetration 217")
	}
}

func TestConfirmedNoAdvancedScanners(t *testing.T) {
	// SC-017..SC-019 (S-9): ship normal ranges double, penetration stays;
	// planets use the best non-penetrating scanner, doubled, with no
	// penetration.
	l := newScanLab()
	l.g.Players[0].Race.LRT.NoAdvancedScanners = true
	rh := l.design(hFreight, 25, Slot{sRhino, 1})
	tg := l.design(hFreight, 25)
	l.fleet(0, Point{}, Stack{Design: rh, Count: 1})
	in, out := l.fleet(1, Point{100, 0}, Stack{Design: tg, Count: 1}), l.fleet(1, Point{100, 1}, Stack{Design: tg, Count: 1})
	if !l.seesFleet(0, in) || l.seesFleet(0, out) {
		t.Error("NAS Rhino: want 100")
	}

	l = newScanLab()
	l.g.Players[0].Race.LRT.NoAdvancedScanners = true
	fe := l.design(hFreight, 25, Slot{sFerret, 1})
	tg = l.design(hFreight, 25)
	l.fleet(0, Point{}, Stack{Design: fe, Count: 1})
	p := l.planet(NoOwner, Point{50, 0})
	orb := l.fleet(1, Point{50, 0}, Stack{Design: tg, Count: 1})
	if l.report(0, p).Level != ReportNormal || !l.seesFleet(0, orb) {
		t.Error("NAS Ferret: want the planet at d² 2500 reported and its orbiting freighter seen")
	}

	l = newScanLab()
	l.g.Players[0].Race.LRT.NoAdvancedScanners = true
	l.g.Players[0].Research.Levels = [NumFields]int{3, 0, 0, 0, 10, 3}
	l.g.PlanetScanners = testPlanetScanners
	tg = l.design(hFreight, 25)
	home := l.planet(0, Point{})
	l.g.Planets[0].HasScanner = true
	_ = home
	deep := l.fleet(1, Point{560, 0}, Stack{Design: tg, Count: 1})
	far := l.planet(NoOwner, Point{0, 100})
	orbit := l.fleet(1, Point{0, 100}, Stack{Design: tg, Count: 1})
	if !l.seesFleet(0, deep) || l.seesFleet(0, orbit) || l.report(0, far).Level != ReportNone {
		t.Error("NAS planet: want 560 ly and no penetration")
	}
}

// testPlanetScanners is a planetary scanner catalogue consistent with the
// SC-011..SC-013 records (electronics 5 → 150, 6 → 220, electronics 10
// with energy and biotech 3 → 320 penetrating, best non-penetrating there
// 280). The tech requirements are chosen to reproduce those records.
var testPlanetScanners = []PlanetScanner{
	{Name: "Viewer 50", Range: 50},
	{Name: "Scoper 150", Range: 150, TechReq: [NumFields]int{Electronics: 3}},
	{Name: "Scoper 220", Range: 220, TechReq: [NumFields]int{Electronics: 6}},
	{Name: "Scoper 280", Range: 280, TechReq: [NumFields]int{Electronics: 8}},
	{Name: "Snooper 320X", Range: 320, Penetrating: true, TechReq: [NumFields]int{Energy: 3, Electronics: 10, Biotech: 3}},
}

func TestConfirmedPlanetScanners(t *testing.T) {
	// SC-011..SC-013 (S-7, S-8): the best scanner the owner's current tech
	// allows, whatever was built; penetrating 320 / 160.
	for _, c := range []struct {
		levels [NumFields]int
		R, P   int
	}{
		{[NumFields]int{Electronics: 5}, 150, 0},
		{[NumFields]int{Electronics: 6}, 220, 0},
		{[NumFields]int{Energy: 3, Electronics: 10, Biotech: 3}, 320, 160},
	} {
		g := Game{Rules: ElegyRules(), Players: []Player{{Research: ResearchState{Levels: c.levels}}}, PlanetScanners: testPlanetScanners}
		p := Planet{Owner: 0, HasScanner: true}
		if R, P, ok := g.planetScan(&p); !ok || R != c.R || P != c.P {
			t.Errorf("levels %v: %d/%d, want %d/%d", c.levels, R, P, c.R, c.P)
		}
	}
	g := Game{Rules: ElegyRules(), Players: []Player{{}}, PlanetScanners: testPlanetScanners}
	if _, _, ok := g.planetScan(&Planet{Owner: 0}); ok {
		t.Error("a planet without a scanner scans")
	}
}

func TestConfirmedPenetrationAndOrbit(t *testing.T) {
	// SC-003..SC-008 (S-5, S-6): only penetrating range reports planets;
	// an orbiting fleet needs both ranges.
	l := newScanLab()
	mole := l.design(hFreight, 25, Slot{sMole, 1})
	tg := l.design(hFreight, 25)
	l.fleet(0, Point{}, Stack{Design: mole, Count: 1})
	p := l.planet(NoOwner, Point{30, 0})
	orb := l.fleet(1, Point{30, 0}, Stack{Design: tg, Count: 1})
	if l.report(0, p).Level != ReportNone || l.seesFleet(0, orb) {
		t.Error("Mole at 30 ly: want no planet report and the orbiting freighter unseen")
	}
	for _, sc := range []Part{sFerret, sChamel, sDolphin, sRobber, sElephant} {
		l := newScanLab()
		v := l.design(hFreight, 25, Slot{sc, 1})
		tg := l.design(hFreight, 25)
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		P := sc.PenRange
		in, out := l.planet(NoOwner, Point{P, 0}), l.planet(NoOwner, Point{P, 1})
		oin, oout := l.fleet(1, Point{P, 0}, Stack{Design: tg, Count: 1}), l.fleet(1, Point{P, 1}, Stack{Design: tg, Count: 1})
		if l.report(0, in).Level != ReportNormal || l.report(0, out).Level != ReportNone || !l.seesFleet(0, oin) || l.seesFleet(0, oout) {
			t.Errorf("%s: want the edge planet and its orbiter seen at d² = P², not past it", sc.Name)
		}
	}
	// The cloak test uses P for an orbiting fleet: a Stealth (35%)
	// freighter orbiting at P is unseen, one in deep space at the same
	// distance is seen (viewer normal 300, penetrating 100).
	l = newScanLab()
	v := l.design(hFreight, 25, Slot{scanPart("Wide", 300, 100), 1})
	st := l.design(hFreight, 31, Slot{sStealth, 1})
	l.fleet(0, Point{}, Stack{Design: v, Count: 1})
	l.planet(NoOwner, Point{100, 0})
	orb = l.fleet(1, Point{100, 0}, Stack{Design: st, Count: 1})
	deep := l.fleet(1, Point{0, 100}, Stack{Design: st, Count: 1})
	if l.seesFleet(0, orb) || !l.seesFleet(0, deep) {
		t.Error("Stealth at P: want the orbiter unseen and the deep-space one seen")
	}
}

func TestConfirmedFleetCloak(t *testing.T) {
	// SC-001, SC-010 (S-12..S-14).
	l := newScanLab()
	st := l.design(hFreight, 31, Slot{sStealth, 1})
	for _, c := range []struct {
		cargo Cargo
		fuel  int
		want  int
	}{
		{Cargo{}, 0, 35},
		{Cargo{Minerals: Minerals{31, 0, 0}}, 0, 17},
		{Cargo{Minerals: Minerals{70, 0, 0}}, 0, 10},
		{Cargo{}, 130, 35},
	} {
		f := Fleet{Owner: 1, Stacks: []Stack{{Design: st, Count: 1}}, Cargo: c.cargo, Fuel: c.fuel}
		if got := l.g.fleetCloak(&f); got != c.want {
			t.Errorf("cargo %+v fuel %d: cloak %d%%, want %d%%", c.cargo, c.fuel, got, c.want)
		}
	}
	// Points to percent: Super-Stealth 140 → 55, Transport Cloaking 300 →
	// 75, Ultra-Stealth 540 → 85.
	for pts, want := range map[int]int{70: 35, 140: 55, 300: 75, 540: 85} {
		if got := cloakPercent(pts); got != want {
			t.Errorf("cloakPercent(%d) = %d, want %d", pts, got, want)
		}
	}
	// Detection bounds (R 50: 35% → 1056, 17% → 1722, 10% → 2025; R 59:
	// 26% → 1905).
	for _, c := range []struct{ R, c, want int }{{50, 35, 1056}, {50, 17, 1722}, {50, 10, 2025}, {59, 26, 1905}} {
		if got := cloakBound(c.R, c.c); got != c.want {
			t.Errorf("cloakBound(%d, %d) = %d, want %d", c.R, c.c, got, c.want)
		}
	}
}

func TestConfirmedTachyonDetectors(t *testing.T) {
	// SC-010: against Transport Cloaking (75%) a Mole sees to d² 625 with
	// no detector, 841 with one (71%), 961 with two (69%).
	for n, edge := range []int{625, 841, 961} {
		l := newScanLab()
		v := l.design(hFreight, 25, Slot{sMole, 1}, Slot{sTachyon, n})
		tc := l.design(hFreight, 25, Slot{Part{CloakPoints: 300}, 1})
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		// Points on the axes at d² = edge and edge+1 (both are sums of
		// two squares here: 625 = 25², 626 = 25²+1; 841 = 29², 842 =
		// 29²+1; 961 = 31², 962 = 31²+1).
		r := isqrt(edge)
		in, out := l.fleet(1, Point{r, 0}, Stack{Design: tc, Count: 1}), l.fleet(1, Point{r, 1}, Stack{Design: tc, Count: 1})
		if !l.seesFleet(0, in) || l.seesFleet(0, out) {
			t.Errorf("%d detectors: want d² %d seen, %d not", n, edge, edge+1)
		}
	}
}

func TestConfirmedColocation(t *testing.T) {
	// SC-002, SC-014 (S-4), LEGACY BUG: at the same position a blind
	// freighter sees a 98% fleet and is seen back; one ly away neither a
	// blind nor a Bat viewer sees anything.
	l := newScanLab()
	blind := l.design(hFreight, 25)
	cloaked := l.design(hFreight, 25, Slot{Part{CloakPoints: 2000}, 1})
	bat := l.design(hFreight, 25, Slot{sBat, 1})
	a := l.fleet(0, Point{10, 10}, Stack{Design: blind, Count: 1})
	b := l.fleet(1, Point{10, 10}, Stack{Design: cloaked, Count: 1})
	if !l.seesFleet(0, b) || !l.seesFleet(1, a) {
		t.Error("co-located fleets do not see each other")
	}
	l.g.Fleets = nil
	l.fleet(0, Point{0, 0}, Stack{Design: blind, Count: 1})
	l.fleet(0, Point{100, 100}, Stack{Design: bat, Count: 1})
	t1 := l.fleet(1, Point{1, 0}, Stack{Design: blind, Count: 1})
	t2 := l.fleet(1, Point{101, 100}, Stack{Design: blind, Count: 1})
	if l.seesFleet(0, t1) || l.seesFleet(0, t2) {
		t.Error("a blind or Bat viewer saw a fleet 1 ly away")
	}
}

func TestConfirmedOrbitReports(t *testing.T) {
	// SC-002, SC-014 (S-15): an orbited planet is reported at position
	// level by a scannerless fleet, normal by any scanner (Bat included),
	// detailed by a Robber Baron.
	for _, c := range []struct {
		slots []Slot
		want  ReportLevel
	}{{nil, ReportPosition}, {[]Slot{{sBat, 1}}, ReportNormal}, {[]Slot{{sRobber, 1}}, ReportDetailed}} {
		l := newScanLab()
		v := l.design(hFreight, 25, c.slots...)
		l.fleet(0, Point{40, 40}, Stack{Design: v, Count: 1})
		p := l.planet(1, Point{40, 40})
		l.g.Planets[0].Surface = Minerals{1, 2, 3}
		r := l.report(0, p)
		if r.Level != c.want {
			t.Errorf("%v: level %d, want %d", c.slots, r.Level, c.want)
		}
		if (r.Surface != Minerals{}) != (c.want == ReportDetailed) {
			t.Errorf("%v: surface %v", c.slots, r.Surface)
		}
	}
}

func TestConfirmedStarbaseCloak(t *testing.T) {
	// SC-009 (S-16): Dolphin (P 100) vs a Space Station with one Stealth
	// Cloak (35%, unweighted): shown at d² 4225, hidden at 4226; with two
	// (55%): 2025 / 2026. The planet itself is still reported.
	for _, c := range []struct {
		cloaks int
		r      int // 4225 = 65², 2025 = 45²
	}{{1, 65}, {2, 45}} {
		l := newScanLab()
		v := l.design(hFreight, 25, Slot{sDolphin, 1})
		sb := l.design(Hull{Name: "Space Station", Starbase: true}, 0, Slot{sStealth, c.cloaks})
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		in, out := l.planet(1, Point{c.r, 0}), l.planet(1, Point{c.r, 1})
		for i := range l.g.Planets {
			l.g.Planets[i].HasStarbase, l.g.Planets[i].StarbaseDesign = true, sb
		}
		ri, ro := l.report(0, in), l.report(0, out)
		if !ri.Starbase || ri.StarbaseDesign != sb || ro.Starbase || ro.StarbaseDesign != -1 || ro.Level != ReportNormal {
			t.Errorf("%d cloaks: inside %+v, outside %+v", c.cloaks, ri, ro)
		}
	}
}

func TestConfirmedDisclosure(t *testing.T) {
	// SC-001..SC-023 (S-20): seen designs are partial; a War Monger viewer
	// gets them in full. A Claim Adjuster viewer gets known players'
	// habitability, nothing without contact. Pick Pocket and Robber Baron
	// see cargo at their exact position only.
	setup := func(prt PRT, at Point, scan Part) (*scanLab, int) {
		l := newScanLab()
		l.g.Players[0].Race.PRT = prt
		l.g.Players[1].Race.Env[0] = EnvRange{Center: 40, Low: 20, High: 60}
		v := l.design(hFreight, 25, Slot{scan, 1})
		tg := l.design(hFreight, 25)
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		id := l.fleet(1, at, Stack{Design: tg, Count: 2})
		l.g.Fleets[1].Cargo = Cargo{Minerals: Minerals{5, 6, 7}, Colonists: 3}
		return l, id
	}
	l, _ := setup(PRTOther, Point{0, 0}, sRhino)
	v := l.view(0)
	if len(v.Designs) != 1 || v.Designs[0].Full || v.Designs[0].Mass != 25 || v.Fleets[0].Cargo != nil || v.Fleets[0].Mass != 2*25+5+6+7+3 {
		t.Errorf("ordinary view %+v", v)
	}
	l, _ = setup(PRTWarMonger, Point{0, 0}, sRhino)
	if v := l.view(0); !v.Designs[0].Full {
		t.Error("War Monger: want full designs")
	}
	l, _ = setup(PRTClaimAdjuster, Point{0, 0}, sRhino)
	if v := l.view(0); len(v.Players) != 1 || v.Players[0].Hab == nil || v.Players[0].Hab[0].Center != 40 {
		t.Errorf("Claim Adjuster players %+v", v.Players)
	}
	l, _ = setup(PRTClaimAdjuster, Point{500, 0}, sRhino)
	if v := l.view(0); len(v.Players) != 0 {
		t.Errorf("Claim Adjuster without contact: %+v", v.Players)
	}
	for _, c := range []struct {
		scan  Part
		at    Point
		cargo bool
	}{{sPick, Point{0, 0}, true}, {sRobber, Point{0, 0}, true}, {sRobber, Point{30, 0}, false}} {
		l, _ := setup(PRTOther, c.at, c.scan)
		v := l.view(0)
		if len(v.Fleets) != 1 || (v.Fleets[0].Cargo != nil) != c.cargo {
			t.Errorf("%s at %v: fleets %+v, want cargo %v", c.scan.Name, c.at, v.Fleets, c.cargo)
		} else if c.cargo && *v.Fleets[0].Cargo != (Minerals{5, 6, 7}) {
			t.Errorf("cargo %v", *v.Fleets[0].Cargo)
		}
	}
}

func TestConfirmedAlliesDoNotShare(t *testing.T) {
	// SC-001F (S-23): views with mutual friends equal those with enemies.
	build := func(rel Relation) []PlayerView {
		l := newScanLab()
		v := l.design(hFreight, 25, Slot{sRhino, 1})
		l.fleet(0, Point{}, Stack{Design: v, Count: 1})
		l.fleet(1, Point{40, 0}, Stack{Design: v, Count: 1})
		l.fleet(1, Point{200, 0}, Stack{Design: v, Count: 1})
		l.g.Players[0].Relations = []Relation{RelationFriend, rel}
		l.g.Players[1].Relations = []Relation{rel, RelationFriend}
		return Views(l.g, nil)
	}
	if !reflect.DeepEqual(build(RelationFriend), build(RelationEnemy)) {
		t.Error("friend and enemy views differ")
	}
}

func TestPredictionPopulationEstimate(t *testing.T) {
	// 400 × max(1, min(4090, (u + rand(u/4) − u/8)/4)), in planet id order;
	// rand(0) still consumes a draw; AR planets 0 with no draw.
	g := Game{Rules: ElegyRules(), Players: make([]Player, 2), Planets: []Planet{
		{ID: 2, Owner: 0, Population: 100},
		{ID: 1, Owner: 0, Population: 2},
		{ID: 3, Owner: 1, Population: 500},
		{ID: 4, Owner: NoOwner},
	}}
	g.Players[1].Race.PRT = PRTAlternateReality
	est := PopulationEstimates(g, &seqRand{draws: []int{7, 10}})
	// id 1: u 2, rand(0) consumes the 7 and gives 0: (2 + 0 − 0)/4 = 0 →
	// 1 → 400. id 2: (100 + 10 − 12)/4 = 24 → 9600.
	if want := map[int]int{1: 400, 2: 9600, 3: 0}; !reflect.DeepEqual(est, want) {
		t.Errorf("estimates %v, want %v", est, want)
	}
}

func TestPredictionFleetsAtOwnPlanets(t *testing.T) {
	// An enemy fleet orbiting the viewer's planet is seen, whatever its
	// cloak and whether or not the planet has a scanner.
	l := newScanLab()
	cl := l.design(hFreight, 25, Slot{Part{CloakPoints: 2000}, 1})
	l.planet(0, Point{70, 70})
	id := l.fleet(1, Point{70, 70}, Stack{Design: cl, Count: 1})
	if !l.seesFleet(0, id) {
		t.Error("enemy fleet at the viewer's planet unseen")
	}
}

func TestPredictionAlternateRealityPlanetScanner(t *testing.T) {
	// R = √(pop/10) colonists: 250,000 → 158; P = R/2 with an Ultra
	// Station; ×1.412 and no P under NAS.
	g := Game{Rules: ElegyRules(), Players: []Player{{Race: Race{PRT: PRTAlternateReality}}}}
	p := Planet{Owner: 0, Population: 2500}
	if R, P, _ := g.planetScan(&p); R != 158 || P != 0 {
		t.Errorf("AR: %d/%d", R, P)
	}
	p.StarbaseHull = 4
	if R, P, _ := g.planetScan(&p); R != 158 || P != 79 {
		t.Errorf("AR Ultra Station: %d/%d", R, P)
	}
	g.Players[0].Race.LRT.NoAdvancedScanners = true
	if R, P, _ := g.planetScan(&p); R != 223 || P != 0 {
		t.Errorf("AR NAS: %d/%d", R, P)
	}
}

func TestPredictionSuperStealthAndStarbaseBonus(t *testing.T) {
	// SS adds 300 points per design and leaves cargo out; Improved
	// Starbases adds 40 to a starbase; above 25,000 points counts as 0.
	l := newScanLab()
	d := l.design(hFreight, 25)
	l.g.Players[1].Race.PRT = PRTSuperStealth
	f := Fleet{Owner: 1, Stacks: []Stack{{Design: d, Count: 1}}, Cargo: Cargo{Minerals: Minerals{100, 0, 0}}}
	if got := l.g.fleetCloak(&f); got != 75 {
		t.Errorf("SS cloak %d%%, want 75%%", got)
	}
	sb := l.design(Hull{Starbase: true}, 0, Slot{sStealth, 1})
	l.g.Players[1].Race = Race{}
	l.g.Players[1].Race.LRT.ImprovedStarbases = true
	p := Planet{Owner: 1, HasStarbase: true, StarbaseDesign: sb}
	if got := l.g.starbaseCloak(&p); got != cloakPercent(110) {
		t.Errorf("ISB starbase cloak %d%%", got)
	}
	huge := l.design(Hull{Starbase: true}, 0, Slot{Part{CloakPoints: 30000}, 1})
	p.StarbaseDesign = huge
	l.g.Players[1].Race = Race{}
	if got := l.g.starbaseCloak(&p); got != 0 {
		t.Errorf("over 25,000 points: %d%%", got)
	}
}

func TestPredictionTurnViews(t *testing.T) {
	// GenerateTurn returns each player's view of the final state.
	prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
	res, err := GenerateTurn(withRules(combatLabGame(prey, 6, 3)), nil, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Views) != 2 || res.Views[0].Player != 0 || res.Views[1].Player != 1 {
		t.Fatalf("views %+v", res.Views)
	}
}

func TestPredictionHeading(t *testing.T) {
	// The vector of this year's movement, halved while a component is
	// 128 or more: (1000,1000) toward (1300,1100) shows (75,25). A fleet
	// that did not move shows none.
	if got := scanHeading(300, 100); got != (Point{75, 25}) {
		t.Errorf("scanHeading(300, 100) = %v", got)
	}
	if got := scanHeading(-255, 3); got != (Point{-127, 1}) {
		t.Errorf("scanHeading(-255, 3) = %v", got)
	}
	l := newScanLab()
	eng := Engine{Name: "Test"}
	d := Design{Name: "Scout", Mass: 8, Engine: eng, Engines: 1, FuelCapacity: 1000, Slots: []Slot{{sRhino, 1}}}
	l.g.Designs = []Design{d}
	l.fleet(0, Point{1000, 1000}, Stack{Design: 0, Count: 1})
	mover := l.fleet(1, Point{1010, 1000}, Stack{Design: 0, Count: 1})
	still := l.fleet(1, Point{1000, 1010}, Stack{Design: 0, Count: 1})
	l.g.Fleets[1].Fuel = 1000
	l.g.Fleets[1].Waypoints = []Waypoint{{Pos: Point{1310, 1100}, Warp: 5}}
	l.g.Fleets[2].Heading, l.g.Fleets[2].HeadingWarp = Point{9, 9}, 9 // stale, cleared by movement
	moveFleets(&l.g)
	for _, f := range l.view(0).Fleets {
		switch f.Fleet {
		case mover:
			if f.Heading != (Point{75, 25}) || f.Warp != 5 {
				t.Errorf("mover heading %v warp %d", f.Heading, f.Warp)
			}
		case still:
			if f.Heading != (Point{}) || f.Warp != 0 {
				t.Errorf("still fleet heading %v warp %d", f.Heading, f.Warp)
			}
		}
	}
}

func TestPredictionPositionReportStarbase(t *testing.T) {
	// A scannerless fleet orbiting an owned planet with a starbase gets a
	// position-only report with the owner, homeworld flag and starbase,
	// and learns the owner and the starbase design (partially).
	l := newScanLab()
	blind := l.design(hFreight, 25)
	sb := l.design(Hull{Name: "Space Station", Starbase: true}, 0)
	l.fleet(0, Point{40, 40}, Stack{Design: blind, Count: 1})
	p := l.planet(1, Point{40, 40})
	l.g.Planets[0].Homeworld, l.g.Planets[0].HasStarbase, l.g.Planets[0].StarbaseDesign = true, true, sb
	l.g.Planets[0].Population = 100
	v := l.view(0)
	r := l.report(0, p)
	if r.Level != ReportPosition || r.Owner != 1 || !r.Homeworld || !r.Starbase || r.StarbaseDesign != sb || r.PopEstimate != 0 {
		t.Errorf("report %+v", r)
	}
	if len(v.Players) != 1 || len(v.Designs) != 1 || v.Designs[0].Design != sb || v.Designs[0].Full {
		t.Errorf("players %+v designs %+v", v.Players, v.Designs)
	}
}

func TestPredictionDefenseEstimate(t *testing.T) {
	// 0 without defenses; else max(1, min(15, (104 − k)/6)) with k =
	// trunc(100·(1 − v/1000)ⁿ + 0.5) for the best allowed defense.
	g := Game{Rules: ElegyRules(), Players: []Player{{Race: pgRace()}}, Defenses: []DefenseType{
		{Name: "SDI", Coverage: 99},
		{Name: "Missile Battery", Coverage: 199, TechReq: [NumFields]int{Energy: 5}},
	}}
	p := Planet{Owner: 0, Population: 10000, Env: [3]int{50, 50, 50}}
	if got := g.defenseEstimate(&p); got != 0 {
		t.Errorf("no defenses: %d", got)
	}
	// SDI 9.9%: 10 defenses → 0.901^10 = 0.3527 → k 35 → (104−35)/6 = 11.
	p.Defenses = 10
	if got := g.defenseEstimate(&p); got != 11 {
		t.Errorf("10 SDI: %d, want 11", got)
	}
	// Missile Battery 19.9% at energy 5: 0.801^10 = 0.1088 → k 11 → 15.
	g.Players[0].Research.Levels[Energy] = 5
	if got := g.defenseEstimate(&p); got != 15 {
		t.Errorf("10 Missile Batteries: %d, want 15", got)
	}
	// No defense type available: estimate 1.
	g.Defenses = nil
	if got := g.defenseEstimate(&p); got != 1 {
		t.Errorf("no type: %d, want 1", got)
	}
}

func TestConfirmedBombingCheck(t *testing.T) {
	// SC-024, SC-031, SC-032: a blind freighter whose plan attacks the
	// owner gets a normal report of an enemy colony without a starbase
	// (estimates, no surface minerals), but position only at a colony with
	// a starbase and at an unowned planet. A plan attacking nobody gives
	// only the orbit report (BINARY-ONLY).
	for _, c := range []struct {
		name     string
		owner    int
		starbase bool
		attack   AttackWho
		want     ReportLevel
	}{
		{"enemy colony", 1, false, AttackNeutralsAndEnemies, ReportNormal},
		{"enemy starbase", 1, true, AttackNeutralsAndEnemies, ReportPosition},
		{"unowned", NoOwner, false, AttackNeutralsAndEnemies, ReportPosition},
		{"plan nobody", 1, false, AttackNobody, ReportPosition},
		{"enemies only, neutral owner", 1, false, AttackEnemies, ReportPosition},
	} {
		l := newScanLab()
		l.g.Players[0].Plans = []BattlePlan{{Attack: c.attack}}
		blind := l.design(hFreight, 25)
		sb := l.design(Hull{Name: "Space Station", Starbase: true}, 0)
		l.fleet(0, Point{40, 40}, Stack{Design: blind, Count: 1})
		p := l.planet(c.owner, Point{40, 40})
		pl := &l.g.Planets[0]
		pl.Surface, pl.Population = Minerals{1, 2, 3}, 100
		pl.HasStarbase, pl.StarbaseDesign = c.starbase, sb
		r := l.report(0, p)
		if r.Level != c.want || r.Surface != (Minerals{}) {
			t.Errorf("%s: level %d surface %v, want level %d", c.name, r.Level, r.Surface, c.want)
		}
	}
}

func TestConfirmedBattleDisclosure(t *testing.T) {
	// SC-031: a design that fought the viewer is disclosed in full even
	// when every ship of it was destroyed. SC-032: the battle planet gets
	// at least a position-only report (BINARY-ONLY for a viewer with no
	// fleet left there).
	l := newScanLab()
	mine := l.design(hFreight, 25)
	theirs := l.design(hFreight, 30)
	far := l.design(hFreight, 35)
	l.fleet(0, Point{500, 500}, Stack{Design: mine, Count: 1})
	l.fleet(1, Point{900, 900}, Stack{Design: far, Count: 1})
	p := l.planet(NoOwner, Point{40, 40})
	b := []battleSeen{{planet: 0, players: []int{0, 1}, designs: map[int][]int{0: {mine}, 1: {theirs}}}}
	v := views(l.g, nil, b, l.g.bombChecks())[0]
	if len(v.Designs) != 1 || v.Designs[0].Design != theirs || !v.Designs[0].Full {
		t.Errorf("designs %+v, want %d in full", v.Designs, theirs)
	}
	if len(v.Players) != 1 || v.Players[0].Player != 1 {
		t.Errorf("players %+v", v.Players)
	}
	found := false
	for _, r := range v.Planets {
		if r.Planet == p {
			found = r.Level == ReportPosition
		}
	}
	if !found {
		t.Errorf("battle planet: planets %+v", v.Planets)
	}
	// A player not in the battle learns nothing from it.
	if v := views(l.g, nil, []battleSeen{{planet: 0, players: []int{1}, designs: map[int][]int{1: {theirs}}}}, l.g.bombChecks())[0]; len(v.Designs) != 0 || len(v.Planets) != 0 {
		t.Errorf("outsider %+v", v)
	}
}

func TestPredictionLeftOutOfBattle(t *testing.T) {
	// A fleet in orbit left out of a battle by the size limit gives a
	// normal report of the planet it orbits.
	l := newScanLab()
	blind := l.design(hFreight, 25)
	l.fleet(0, Point{40, 40}, Stack{Design: blind, Count: 1})
	p := l.planet(1, Point{40, 40})
	b := []battleSeen{{planet: 0, players: []int{0, 1}, designs: map[int][]int{}, leftOut: []int{0}}}
	level := ReportNone
	for _, r := range views(l.g, nil, b, l.g.bombChecks())[0].Planets {
		if r.Planet == p {
			level = r.Level
		}
	}
	if level != ReportNormal {
		t.Errorf("level %d, want normal", level)
	}
}

func TestPredictionBattleRecords(t *testing.T) {
	// battles records each battle's planet, players and the designs on the
	// board, destroyed ones included, for the views.
	prey := testDesign(tFrigate, 20, Slot{tLaser, 1})
	g := combatLabGame(prey, 6, 3)
	res := battles(&g, rand.New(rand.NewSource(3)), map[int]bool{})
	if len(res.seen) != 1 {
		t.Fatalf("seen %+v", res.seen)
	}
	s := res.seen[0]
	if len(s.players) != 2 || len(s.designs[0]) == 0 || len(s.designs[1]) == 0 {
		t.Errorf("seen %+v", s)
	}
}

func TestConfirmedDefenseEstimateVectors(t *testing.T) {
	// SCANNING.md: Neutron Shields (v 38): 1, 3, 5, 10 defenses → 1, 2, 3,
	// 6; 40 defenses with 10 operable → 6; 100 defenses with population
	// 104,400 (42 operable) → 14.
	g := Game{Rules: ElegyRules(), Players: []Player{{Race: pgRace()}}, Defenses: []DefenseType{{Name: "Neutron Shield", Coverage: 38}}}
	for _, c := range []struct{ defenses, pop, want int }{
		{1, 10000, 1}, {3, 10000, 2}, {5, 10000, 3}, {10, 10000, 6}, {40, 250, 6}, {100, 1044, 14},
	} {
		p := Planet{Owner: 0, Population: c.pop, Defenses: c.defenses, Env: [3]int{50, 50, 50}}
		if got := g.defenseEstimate(&p); got != c.want {
			t.Errorf("%d defenses, pop %d: %d, want %d", c.defenses, c.pop, got, c.want)
		}
	}
}

func TestConfirmedHeadingVectors(t *testing.T) {
	// SC-027: halved toward zero while a component is 128 or more.
	for _, c := range []struct{ in, want Point }{
		{Point{300, 100}, Point{75, 25}}, {Point{50, -120}, Point{50, -120}}, {Point{-128, 3}, Point{-64, 1}},
		{Point{300, -7}, Point{75, -1}}, {Point{-1, -395}, Point{0, -98}},
	} {
		if got := scanHeading(c.in.X, c.in.Y); got != c.want {
			t.Errorf("scanHeading(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPredictionBombingCheckAtBombingStep(t *testing.T) {
	// SCANNING.md, CONFIRMED SC-035: the check is taken at the bombing step and
	// the report written from the end-of-year state, so a planet bombing
	// empties is still reported at the normal level, now unowned.
	l := newScanLab()
	l.g.Players[0].Plans = []BattlePlan{{Attack: AttackNeutralsAndEnemies}}
	blind := l.design(hFreight, 25)
	l.fleet(0, Point{40, 40}, Stack{Design: blind, Count: 1})
	p := l.planet(1, Point{40, 40})
	bombs := l.g.bombChecks()
	l.g.Planets[0].Owner = NoOwner // emptied by bombing
	level := ReportNone
	for _, r := range views(l.g, nil, nil, bombs)[0].Planets {
		if r.Planet == p {
			level = r.Level
			if r.Owner != NoOwner {
				t.Errorf("owner %d, want none", r.Owner)
			}
		}
	}
	if level != ReportNormal {
		t.Errorf("level %d, want normal", level)
	}
}
