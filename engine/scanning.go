package engine

import (
	"math"
	"slices"
	"sort"
)

// Per-player knowledge (stars-elegy docs/SCANNING.md): what each player
// sees at the end of a year and what each sighting reveals. Views is a
// pure function of the post-turn game and the year's population
// estimates.

// PlanetScanner is one planetary scanner of the catalogue.
type PlanetScanner struct {
	Name        string
	Range       int
	Penetrating bool // P = Range/2
	TechReq     [NumFields]int
}

// DefenseType is one planetary defense of the catalogue. Coverage is per
// defense, in tenths of a percent.
type DefenseType struct {
	Name     string
	Coverage int
	TechReq  [NumFields]int
}

// ReportLevel is how much of a planet a player is told.
type ReportLevel int

const (
	ReportNone     ReportLevel = iota
	ReportPosition             // position, owner, homeworld, starbase
	ReportNormal
	ReportDetailed // normal plus surface minerals
	ReportOwn      // everything
)

// PlayerView is one player's knowledge at the end of a year.
type PlayerView struct {
	Player  int
	Fleets  []FleetSighting  // other players' fleets seen, by fleet index
	Planets []PlanetReport   // every planet with a report, by planet index
	Designs []DesignSighting // other players' designs seen, by design index
	Players []PlayerSighting // other known players, in player order
	Scores  []ScoreRecord    // score records this player may see (scores.go)
	Objects ObjectsSeen      // the space objects seen this year
}

// FleetSighting is another player's fleet as a player sees it.
type FleetSighting struct {
	Fleet  int // fleet id
	Owner  int
	Pos    Point
	Stacks []Stack // designs and counts; damage is not shown
	// Heading is the direction of this year's movement and Warp its warp,
	// both zero when the fleet did not move.
	Heading Point
	Warp    int
	Mass    int // ship mass plus ironium, boranium, germanium, colonists
	// Cargo is set only for a cargo scan (Pick Pocket, Robber Baron at
	// the same position).
	Cargo *Minerals
}

// PlanetReport is a planet as a player is told about it.
type PlanetReport struct {
	Planet int // planet id
	Level  ReportLevel
	Pos    Point
	// Every report.
	Owner          int
	Homeworld      bool
	Starbase       bool // false when the starbase is cloaked from the viewer
	StarbaseDesign int  // design index, or -1
	// Normal and above.
	Env            [3]int
	Concentrations [NumMinerals]int
	// PopEstimate is the year's population estimate in colonists for an
	// owned planet (0 for Alternate Reality and uninhabited planets).
	PopEstimate int
	// DefenseEstimate is the coverage estimate, 0 (none) or 1..15.
	DefenseEstimate int
	// Detailed.
	Surface Minerals
}

// DesignSighting is another player's design as a player knows it.
type DesignSighting struct {
	Design int // design index
	Full   bool
	Hull   string
	Mass   int
}

// PlayerSighting is another known player.
type PlayerSighting struct {
	Player int
	// Hab is the player's habitability, sent only to a Claim Adjuster
	// viewer (with every tech level shown as zero).
	Hab *[3]EnvRange
}

// legacyColocation reproduces the original's LEGACY BUG that every test
// passes at distance 0: a fleet at exactly an enemy's position sees it
// whatever its own scanner and the target's cloak (SCANNING.md
// "Co-location", CONFIRMED SC-002, SC-014). Set it to false to require a
// scanner and apply cloaking at distance 0 too.
const legacyColocation = true

func coLocated(d2 int) bool { return legacyColocation && d2 == 0 }

// tachyonFactor is T by Tachyon Detector count (capped at 17).
var tachyonFactor = [18]int{100, 95, 93, 91, 90, 89, 88, 87, 86, 86, 85, 84, 84, 83, 83, 82, 82, 81}

// scanner is one viewing object: a fleet or a planet of the viewer.
type scanner struct {
	pos        Point
	R, P       int
	tachyon    int // T, 100 for planets
	fleet      bool
	anyScanner bool // fleet: some scanner (Bat included)
	cargoScan  bool
	detailed   bool
}

func d2(a, b Point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

// root4 is ⌊⁴√x⌋.
func root4(x int) int {
	r := 0
	for (r+1)*(r+1)*(r+1)*(r+1) <= x {
		r++
	}
	return r
}

// designScan is a design's normal and penetrating range for an owner,
// and whether it has any scanner (SCANNING.md "Ship designs").
func (g *Game) designScan(d Design, owner int) (R, P int, any bool) {
	sumR, sumP := 0, 0
	for _, s := range d.Slots {
		if s.Count <= 0 || !s.Part.Scanner {
			continue
		}
		any = true
		r, p := s.Part.ScanRange, s.Part.PenRange
		sumR += s.Count * r * r * r * r
		sumP += s.Count * p * p * p * p
	}
	pl := &g.Players[owner]
	if d.Hull.JOATScanner && pl.Race.PRT == PRTJackOfAllTrades {
		any = true
		e := pl.Research.Levels[Electronics]
		r, p := 20*e, 10*e
		sumR += r * r * r * r
		sumP += p * p * p * p
	}
	R, P = root4(sumR), root4(sumP)
	if pl.Race.LRT.NoAdvancedScanners {
		R *= 2
	}
	return R, P, any
}

func tachyons(d Design) int {
	n := 0
	for _, s := range d.Slots {
		if s.Part.Tachyon {
			n += s.Count
		}
	}
	return min(n, 17)
}

// planetScan is a planet's scanner ranges (SCANNING.md "Planets").
func (g *Game) planetScan(p *Planet) (R, P int, ok bool) {
	pl := &g.Players[p.Owner]
	nas := pl.Race.LRT.NoAdvancedScanners
	if pl.Race.PRT == PRTAlternateReality {
		// BINARY-ONLY: from population (colonists).
		R = isqrt(p.Population * 100 / 10)
		if nas {
			return R * 1412 / 1000, 0, true
		}
		if p.StarbaseHull >= 4 { // Ultra Station or Death Star
			P = R / 2
		}
		return R, P, true
	}
	if !p.HasScanner {
		return 0, 0, false
	}
	best := -1
	for i, s := range g.PlanetScanners {
		if nas && s.Penetrating {
			continue
		}
		allowed := true
		for f := range NumFields {
			if pl.Research.Levels[f] < s.TechReq[f] {
				allowed = false
			}
		}
		if allowed && (best < 0 || s.Range > g.PlanetScanners[best].Range) {
			best = i
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	s := g.PlanetScanners[best]
	R = s.Range
	if s.Penetrating {
		P = R / 2
	}
	if nas {
		R *= 2
	}
	return R, P, true
}

func isqrt(x int) int {
	r := 0
	for (r+1)*(r+1) <= x {
		r++
	}
	return r
}

// cloakPercent turns cloak points into a percentage.
func cloakPercent(u int) int {
	switch {
	case u <= 100:
		return u / 2
	case u <= 300:
		return 50 + (u-100)/8
	case u <= 612:
		return 75 + (u-300)/24
	case u <= 1124:
		return 88 + (u-612)/64
	case u < 1380:
		return 96
	case u < 1612:
		return 97
	}
	return 98
}

func designCloakPoints(d Design) int {
	n := 0
	for _, s := range d.Slots {
		n += s.Part.CloakPoints * s.Count
	}
	return n
}

// fleetCloak is a fleet's cloak percent (SCANNING.md "Fleet cloak").
func (g *Game) fleetCloak(f *Fleet) int {
	ss := g.Players[f.Owner].Race.PRT == PRTSuperStealth
	num, den := 0, 0
	for _, s := range f.Stacks {
		if s.Count <= 0 {
			continue
		}
		d := g.Designs[s.Design]
		pts := designCloakPoints(d)
		if ss {
			pts += 300
		}
		m := d.Mass * s.Count
		num += pts * m
		den += m
	}
	if !ss {
		den += f.Cargo.mass()
	}
	if den == 0 {
		return 0
	}
	return cloakPercent(num / den)
}

// starbaseCloak is a starbase's cloak percent: design points without
// mass weighting, +40 for Improved Starbases, +300 for Super Stealth;
// above 25,000 points counts as 0.
func (g *Game) starbaseCloak(p *Planet) int {
	pts := designCloakPoints(g.Designs[p.StarbaseDesign])
	race := g.Players[p.Owner].Race
	if race.LRT.ImprovedStarbases {
		pts += 40
	}
	if race.PRT == PRTSuperStealth {
		pts += 300
	}
	if pts > 25000 {
		pts = 0
	}
	return cloakPercent(pts)
}

// cloakBound is ⌊⌊(100−c)·X²/100⌋·(100−c)/100⌋.
func cloakBound(x, c int) int { return (100 - c) * x * x / 100 * (100 - c) / 100 }

// scanners lists player v's viewing objects.
func (g *Game) scanners(v int) []scanner {
	var out []scanner
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if f.Owner != v || fleetShips(f) == 0 {
			continue
		}
		s := scanner{pos: f.Pos, tachyon: 100, fleet: true}
		for _, st := range f.Stacks {
			if st.Count <= 0 {
				continue
			}
			d := g.Designs[st.Design]
			R, P, any := g.designScan(d, v)
			s.R, s.P = max(s.R, R), max(s.P, P)
			s.anyScanner = s.anyScanner || any
			s.tachyon = min(s.tachyon, tachyonFactor[tachyons(d)])
			for _, sl := range d.Slots {
				if sl.Count > 0 && sl.Part.CargoScan {
					s.cargoScan = true
				}
				if sl.Count > 0 && sl.Part.DetailedPlanetScan {
					s.detailed = true
				}
			}
		}
		out = append(out, s)
	}
	for i := range g.Planets {
		p := &g.Planets[i]
		if p.Owner != v {
			continue
		}
		if R, P, ok := g.planetScan(p); ok {
			out = append(out, scanner{pos: p.Pos, R: R, P: P, tachyon: 100})
		}
	}
	return out
}

// seesFleet reports whether scanner s sees fleet f (SCANNING.md "Seeing
// fleets"). orbit is whether f orbits a planet; cloak its cloak percent.
func seesFleet(s scanner, pos Point, orbit bool, cloak int) bool {
	dd := d2(s.pos, pos)
	if coLocated(dd) {
		return true
	}
	if dd > s.R*s.R || (orbit && dd > s.P*s.P) {
		return false
	}
	if c := cloak * s.tachyon / 100; c > 0 {
		if dd > cloakBound(s.R, c) || (orbit && dd > cloakBound(s.P, c)) {
			return false
		}
	}
	return true
}

// PopulationEstimates draws the year's population estimate (colonists)
// for every owned planet, in planet id order, shared by every viewer
// (SCANNING.md "Population estimate": the range CONFIRMED SC-024..SC-033,
// the draw order BINARY-ONLY). Alternate Reality
// planets get 0 without a draw. rand(0) for a population under 4 units
// still consumes one draw and gives 0.
func PopulationEstimates(g Game, rng Rand) map[int]int {
	order := make([]int, len(g.Planets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return g.Planets[order[a]].ID < g.Planets[order[b]].ID })
	est := map[int]int{}
	for _, i := range order {
		p := &g.Planets[i]
		if p.Owner == NoOwner || p.Population <= 0 {
			continue
		}
		if g.Players[p.Owner].Race.PRT == PRTAlternateReality {
			est[p.ID] = 0
			continue
		}
		u := p.Population
		r := rng.Intn(max(1, u/4)) // rand(0) consumes a draw and yields 0
		est[p.ID] = 400 * max(1, min(4090, (u+r-u/8)/4))
	}
	return est
}

// Views computes every player's view of the post-turn game (SCANNING.md),
// with the year's population estimates.
func Views(g Game, estimates map[int]int) []PlayerView {
	return views(g, estimates, nil, g.bombChecks())
}

// views is Views with the year's battles, which give their players
// planet reports and designs, and the bombing checks made at the bombing
// step (bombs[v] holds the planet indexes viewer v would bomb).
func views(g Game, estimates map[int]int, battles []battleSeen, bombs []map[int]bool) []PlayerView {
	return viewsWith(g, estimates, battles, bombs, nil)
}

// viewsWith is views with what the space objects add to each player's
// sight (ObjectSight, by player; nil for none).
func viewsWith(g Game, estimates map[int]int, battles []battleSeen, bombs []map[int]bool, sights []ObjectSight) []PlayerView {
	out := make([]PlayerView, len(g.Players))
	for v := range g.Players {
		var sight ObjectSight
		if v < len(sights) {
			sight = sights[v]
		}
		out[v] = g.view(v, estimates, battles, bombs[v], sight)
	}
	return out
}

func (g *Game) view(v int, estimates map[int]int, battles []battleSeen, bombs map[int]bool, sight ObjectSight) PlayerView {
	view := PlayerView{Player: v, Objects: sight.Seen}
	scs := g.scanners(v)
	// A PP player's packets scan like penetrating planet scanners
	// (SCANNING.md "PP packet scanners").
	for _, s := range sight.Scanners {
		scs = append(scs, scanner{pos: s.Pos, R: s.R, P: s.P, tachyon: 100})
	}
	sdSeen := map[int]bool{}
	for _, fi := range sight.Fleets {
		sdSeen[fi] = true
	}
	known := map[int]bool{}
	for _, p := range sight.Owners {
		known[p] = true
	}
	designs := map[int]bool{}
	// Designs disclosed in full: every other player's design that fought
	// the viewer in a battle, even when all its ships were destroyed
	// (SCANNING.md "Designs", CONFIRMED SC-031). battlePlanets get at
	// least a position-only report (SCANNING.md "Battles", BINARY-ONLY);
	// leftOutPlanets, where a viewer's fleet in orbit was left out by the
	// battle's size limit, a normal report (SCANNING.md "Left out of a
	// battle or hit by mines", BINARY-ONLY).
	//
	// "Fought it" is every other player in the battle's player list P,
	// allies included, and each becomes a known player (SCANNING.md
	// "Designs" and "Players", CONFIRMED SC-036).
	fullDesigns := map[int]bool{}
	battlePlanets := map[int]bool{}
	leftOutPlanets := map[int]bool{}
	for _, b := range battles {
		if b.planet >= 0 && slices.Contains(b.leftOut, v) {
			leftOutPlanets[b.planet] = true
		}
		if !slices.Contains(b.players, v) {
			continue
		}
		if b.planet >= 0 {
			battlePlanets[b.planet] = true
		}
		for p, ds := range b.designs {
			if p == v {
				continue
			}
			known[p] = true
			for _, d := range ds {
				designs[d] = true
				fullDesigns[d] = true
			}
		}
	}
	for _, d := range sight.Designs {
		designs[d] = true
		fullDesigns[d] = true
	}
	planetOwner := map[Point]int{}
	for i := range g.Planets {
		planetOwner[g.Planets[i].Pos] = g.Planets[i].Owner
	}

	// Fleets.
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if f.Owner == v || fleetShips(f) == 0 {
			continue
		}
		owner, orbit := planetOwner[f.Pos]
		cloak := g.fleetCloak(f)
		// Every enemy fleet orbiting the viewer's planet is seen
		// (SCANNING.md "Fleets at your planets", CONFIRMED SC-024, SC-026).
		seen := orbit && owner == v || sdSeen[i]
		cargo := false
		for _, s := range scs {
			if seesFleet(s, f.Pos, orbit, cloak) {
				seen = true
				if s.fleet && s.cargoScan && s.pos == f.Pos {
					cargo = true
				}
			}
		}
		if !seen {
			continue
		}
		known[f.Owner] = true
		fs := FleetSighting{Fleet: f.ID, Owner: f.Owner, Pos: f.Pos, Mass: f.Cargo.mass()}
		for _, st := range f.Stacks {
			if st.Count <= 0 {
				continue
			}
			fs.Stacks = append(fs.Stacks, Stack{Design: st.Design, Count: st.Count})
			fs.Mass += g.Designs[st.Design].Mass * st.Count
			designs[st.Design] = true
		}
		fs.Heading, fs.Warp = f.Heading, f.HeadingWarp
		if cargo {
			m := f.Cargo.Minerals
			fs.Cargo = &m
		}
		view.Fleets = append(view.Fleets, fs)
	}

	// Planets. An Interstellar Traveler gets a normal report of every
	// planet whose starbase has a stargate within range of one of its own
	// planets' gates (SCANNING.md "Interstellar Traveler through gates",
	// CONFIRMED OB-013).
	type itGate struct {
		pos Point
		r   int // -1 for any range
	}
	var itGates []itGate
	if g.Players[v].Race.PRT == PRTInterstellarTraveler {
		for i := range g.Planets {
			if p := &g.Planets[i]; p.Owner == v {
				if r, ok := g.gateRange(p); ok {
					itGates = append(itGates, itGate{p.Pos, r})
				}
			}
		}
	}
	for i := range g.Planets {
		p := &g.Planets[i]
		if p.Owner == v {
			view.Planets = append(view.Planets, g.report(p, ReportOwn, true, estimates))
			continue
		}
		level := ReportNone
		sbShown := false
		sbCloak := 0
		if p.HasStarbase && p.Owner != NoOwner {
			sbCloak = g.starbaseCloak(p)
		}
		if battlePlanets[i] {
			level = max(level, ReportPosition)
			sbShown = true
		}
		if leftOutPlanets[i] || bombs[i] {
			level = max(level, ReportNormal)
		}
		// An IT viewer's gates report the gated planets in their range.
		for _, gt := range itGates {
			if _, ok := g.gateRange(p); ok {
				dd := d2(gt.pos, p.Pos)
				if gt.r < 0 || dd <= gt.r*gt.r {
					level = max(level, ReportNormal)
					// ASSUMPTION S5: the starbase cloak bound uses the
					// gate's range as P (BINARY-ONLY in SCANNING.md).
					if gt.r < 0 || dd <= (100-sbCloak)*(100-sbCloak)*gt.r*gt.r/10000 {
						sbShown = true
					}
				}
			}
		}
		for _, s := range scs {
			dd := d2(s.pos, p.Pos)
			if s.P > 0 && dd <= s.P*s.P {
				level = max(level, ReportNormal)
				if dd <= (100-sbCloak)*(100-sbCloak)*s.P*s.P/10000 {
					sbShown = true
				}
			}
			if s.fleet && dd == 0 {
				switch {
				case s.detailed:
					level = max(level, ReportDetailed)
				case s.anyScanner:
					level = max(level, ReportNormal)
				default:
					level = max(level, ReportPosition)
				}
				sbShown = true // d² = 0 is inside every starbase cloak bound
			}
		}
		if level == ReportNone {
			continue
		}
		r := g.report(p, level, sbShown, estimates)
		if p.Owner != NoOwner {
			known[p.Owner] = true
		}
		if r.Starbase {
			designs[p.StarbaseDesign] = true
		}
		view.Planets = append(view.Planets, r)
	}

	// Designs: partial (hull and mass), or full for a War Monger viewer.
	full := g.Players[v].Race.PRT == PRTWarMonger
	var ds []int
	for d := range designs {
		ds = append(ds, d)
	}
	sort.Ints(ds)
	for _, d := range ds {
		view.Designs = append(view.Designs, DesignSighting{Design: d, Full: full || fullDesigns[d], Hull: g.Designs[d].Hull.Name, Mass: g.Designs[d].Mass})
	}

	// Players: name only, plus habitability for a Claim Adjuster viewer.
	ca := g.Players[v].Race.PRT == PRTClaimAdjuster
	for p := range g.Players {
		if p == v || !known[p] {
			continue
		}
		ps := PlayerSighting{Player: p}
		if ca {
			hab := g.Players[p].Race.Env
			ps.Hab = &hab
		}
		view.Players = append(view.Players, ps)
	}
	return view
}

// bombChecks is, per viewer, the planets it would bomb (bombCheck), taken
// at the bombing step right after battles. The report itself is written
// from the end-of-year state, so a planet bombing empties still gets its
// normal report (SCANNING.md "Bombing", CONFIRMED SC-035).
func (g *Game) bombChecks() []map[int]bool {
	out := make([]map[int]bool, len(g.Players))
	for v := range g.Players {
		out[v] = map[int]bool{}
		for i := range g.Planets {
			if g.bombCheck(v, i) {
				out[v][i] = true
			}
		}
	}
	return out
}

// bombCheck reports whether viewer v would bomb planet p by TAKEOVER.md
// "Who bombs": another player owns it, it has no starbase, and one of v's
// fleets in orbit has a battle plan that attacks the owner. Such a planet
// gets a normal report whatever v's scanners and whether or not any fleet
// carries bombs (SCANNING.md "Bombing check", CONFIRMED for scannerless
// fleets without bombs, SC-024, SC-031, SC-032; mechanism BINARY-ONLY).
func (g *Game) bombCheck(v, pi int) bool {
	p := &g.Planets[pi]
	if p.Owner == NoOwner || p.Owner == v || g.hasStarbase(pi) {
		return false
	}
	for i := range g.Fleets {
		f := &g.Fleets[i]
		if f.Owner == v && f.Pos == p.Pos && fleetShips(f) > 0 && g.bombsOwner(v, g.plan(v, f.Plan), p.Owner) {
			return true
		}
	}
	return false
}

// report builds a planet report at a level.
func (g *Game) report(p *Planet, level ReportLevel, starbase bool, estimates map[int]int) PlanetReport {
	r := PlanetReport{Planet: p.ID, Level: level, Pos: p.Pos, Owner: p.Owner, Homeworld: p.Homeworld, StarbaseDesign: -1}
	if p.HasStarbase && starbase {
		r.Starbase, r.StarbaseDesign = true, p.StarbaseDesign
	}
	if level < ReportNormal {
		return r
	}
	r.Env = p.Env
	for m := range NumMinerals {
		r.Concentrations[m] = p.Deposits[m].Concentration
	}
	if p.Owner != NoOwner {
		r.PopEstimate = estimates[p.ID]
		if level == ReportOwn {
			r.PopEstimate = p.Population * 100
		}
		r.DefenseEstimate = g.defenseEstimate(p)
	}
	if level >= ReportDetailed {
		r.Surface = p.Surface
	}
	return r
}

// scanHeading halves both components, truncating toward zero, while
// either has magnitude 128 or more (SCANNING.md "Heading", CONFIRMED
// SC-027).
func scanHeading(dx, dy int) Point {
	for abs(dx) >= 128 || abs(dy) >= 128 {
		dx, dy = dx/2, dy/2
	}
	return Point{dx, dy}
}

// DefenseCoverage is a planet's defense coverage, 0..1, for n defenses of
// coverage c tenths of a percent each: 1 − (1 − c/1000)^n (COMPONENTS.md
// "Defense coverage", CONFIRMED CS-001).
func DefenseCoverage(n, c int) float64 {
	return 1 - math.Pow(1-float64(c)/1000, float64(n))
}

// defenseEstimate is the planet's defense coverage estimate (SCANNING.md,
// CONFIRMED SC-024..SC-033): 0 without defenses, else 1..15 from the share of a bomb's
// kill that gets through the best defense the owner's tech allows.
func (g *Game) defenseEstimate(p *Planet) int {
	if p.Defenses <= 0 || p.Owner == NoOwner {
		return 0
	}
	pl := &g.Players[p.Owner]
	s := 1.0
	if v, ok := g.bestDefense(p.Owner); ok {
		n := min(p.Defenses, NewColony(p, pl).OperableDefenses(p.Population))
		s = math.Pow(1-float64(v)/1000, float64(n)) // 1 − DefenseCoverage(n, v)
	}
	k := int(100*s + 0.5)
	return max(1, min(15, (104-k)/6))
}

// gateRange is the safe range in ly of the stargate on planet p's
// starbase, -1 for any range (OBJECTS.md "What makes a gate": the
// component table's safe_range, null meaning any).
func (g *Game) gateRange(p *Planet) (int, bool) {
	if p.Owner == NoOwner || !p.HasStarbase || p.StarbaseDesign < 0 || p.StarbaseDesign >= len(g.Designs) {
		return 0, false
	}
	for _, sl := range g.Designs[p.StarbaseDesign].Slots {
		if sl.Count <= 0 || sl.Part.Kind != PartStargate {
			continue
		}
		c, ok := Components().Lookup(sl.Part.Name)
		if !ok {
			continue
		}
		if r, ok := c.Stats["safe_range"].(float64); ok {
			return int(r), true
		}
		return -1, true
	}
	return 0, false
}
