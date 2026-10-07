package races

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

var prtByCode = map[string]engine.PRT{
	"HE": engine.PRTHyperExpansion, "SS": engine.PRTSuperStealth, "WM": engine.PRTWarMonger,
	"CA": engine.PRTClaimAdjuster, "IS": engine.PRTInnerStrength, "SD": engine.PRTSpaceDemolition,
	"PP": engine.PRTPacketPhysics, "IT": engine.PRTInterstellarTraveler,
	"AR": engine.PRTAlternateReality, "JOAT": engine.PRTJackOfAllTrades,
}

// parseRace reads the race column of stars-elegy experiments/rd/races.tsv:
// "PRT LRTs gN hab[g t r] colN facOUT/COST/OPER mineOUT/COST/OPER res......
// [f-germ] [tech3]", where a habitat axis is "imm" or "lo-hi" and each
// research character is "-" (75% extra), "n" (normal) or "+" (50% less).
func parseRace(s string) (Design, error) {
	s = strings.Replace(s, "hab[", "hab ", 1)
	s = strings.Replace(s, "]", "", 1)
	f := strings.Fields(s)
	var d Design
	prt, ok := prtByCode[f[0]]
	if !ok {
		return d, fmt.Errorf("PRT %q", f[0])
	}
	d.Race.PRT = prt
	if f[1] != "-" {
		for _, code := range strings.Split(f[1], ",") {
			found := false
			for i := range NumLRTs {
				if LRTCode(i) == code {
					d.SetLRT(i, true)
					found = true
				}
			}
			if !found {
				return d, fmt.Errorf("LRT %q", code)
			}
		}
	}
	num := func(v string) int {
		n, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		return n
	}
	triple := func(v string) (int, int, int) {
		p := strings.Split(v, "/")
		return num(p[0]), num(p[1]), num(p[2])
	}
	for i := 2; i < len(f); i++ {
		w := f[i]
		switch {
		case strings.HasPrefix(w, "g"):
			d.Race.GrowthRate = num(w[1:])
		case w == "hab":
			for a := range 3 {
				v := f[i+1+a]
				if v == "imm" {
					d.Race.Env[a] = engine.EnvRange{Immune: true}
					continue
				}
				lohi := strings.Split(v, "-")
				lo, hi := num(lohi[0]), num(lohi[1])
				d.Race.Env[a] = engine.EnvRange{Low: lo, High: hi, Center: lo + (hi-lo)/2}
			}
			i += 3
		case strings.HasPrefix(w, "col"):
			d.Race.ColonistsPerResource = num(w[3:])
		case strings.HasPrefix(w, "fac"):
			d.Race.FactoryOutput, d.Race.FactoryCost, d.Race.FactoriesOperated = triple(w[3:])
		case strings.HasPrefix(w, "mine"):
			d.Race.MineOutput, d.Race.MineCost, d.Race.MinesOperated = triple(w[4:])
		case strings.HasPrefix(w, "res"):
			for k, c := range w[3:] {
				d.Race.ResearchCosts[k] = map[rune]engine.ResearchCost{'-': engine.ResearchExpensive, 'n': engine.ResearchNormal, '+': engine.ResearchCheap}[c]
			}
		case w == "f-germ":
			d.Race.FactoryLessGermanium = true
		case w == "tech3":
			d.ExpensiveAt3 = true
		default:
			return d, fmt.Errorf("field %q", w)
		}
	}
	return d, nil
}

type rdCase struct {
	id, label string
	points    int
	race      Design
}

// rdCases are the RD corpus races with a predicted (and confirmed)
// points value.
func rdCases(t *testing.T) []rdCase {
	t.Helper()
	fh, err := os.Open("testdata/races.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	var out []rdCase
	sc := bufio.NewScanner(fh)
	sc.Scan() // header
	for sc.Scan() {
		cols := strings.Split(sc.Text(), "\t")
		if len(cols) < 5 || cols[2] == "" {
			continue
		}
		d, err := parseRace(cols[4])
		if err != nil {
			t.Fatalf("%s: %v", cols[0], err)
		}
		pts, err := strconv.Atoi(cols[2])
		if err != nil {
			t.Fatalf("%s: %v", cols[0], err)
		}
		out = append(out, rdCase{cols[0], cols[1], pts, d})
	}
	return out
}

// Advantage points (RACES.md "Advantage points", CONFIRMED RD-1..RD-4):
// every race of the RD corpus with a points value. rw04d's centre and
// rw04e's stat 15 are not part of the race column, so their points are
// those of the repaired race, as RACES.md scores them.
func TestConfirmedRDPoints(t *testing.T) {
	cases := rdCases(t)
	if len(cases) < 55 {
		t.Fatalf("only %d cases", len(cases))
	}
	for _, c := range cases {
		if got := Points(c.race); got != c.points {
			t.Errorf("%s (%s): %d points, want %d", c.id, c.label, got, c.points)
		}
	}
}

// The default race scores 25 (RACES.md "Race settings").
func TestConfirmedDefaultRace(t *testing.T) {
	if got := Points(Default()); got != 25 {
		t.Errorf("default race: %d points, want 25", got)
	}
}

// pg001 is the RD-P base race: the PG001 game's player race (SS, no
// LRTs, growth 10, 15–85 centre 50 on every axis, the standard economy
// and research), worth 245 points (stars-elegy experiments/rd README).
func pg001() Design {
	d := Default()
	d.Race.PRT = engine.PRTSuperStealth
	d.Race.GrowthRate = 10
	return d
}

// The turn-time penalty cases RD-P1..RD-P10 (RACES.md "In a running
// game", CONFIRMED; PARITY.md "Turn-time penalty").
func TestConfirmedYearlyCheck(t *testing.T) {
	if got := Points(pg001()); got != 245 {
		t.Fatalf("PG001 base race: %d points, want 245", got)
	}
	type want struct {
		points   int // after the edit
		punished bool
		col      int
		growth   int
		after    int // points after the check, 0 to skip
	}
	cases := []struct {
		id   string
		edit func(d *Design)
		want want
	}{
		{"P1", func(d *Design) { d.Race.FactoryCost, d.Race.MineCost = 5, 2 }, want{-444, true, 2500, 7, 1042}},
		{"P2", func(d *Design) {
			r := &d.Race
			r.FactoryOutput, r.FactoryCost, r.FactoriesOperated, r.MineOutput, r.MineCost, r.MinesOperated = 15, 5, 25, 25, 2, 25
		}, want{-2092, true, 2500, 4, 1602}},
		{"P3", func(d *Design) {
			r := &d.Race
			r.FactoryOutput, r.FactoryCost, r.FactoriesOperated, r.MineOutput, r.MineCost, r.MinesOperated = 15, 5, 25, 25, 2, 25
			for f := range r.ResearchCosts {
				r.ResearchCosts[f] = engine.ResearchCheap
			}
			// lrt=a000006d: bits 0, 2, 3, 5, 6 (IFE, ARM, ISB, UR,
			// MA), 29 (expensive fields start at 3) and 31 (the
			// germanium option); RACES.md "Race file" confirms bit
			// 29.
			d.ExpensiveAt3 = true
			for _, code := range []string{"IFE", "ARM", "ISB", "UR", "MA"} {
				for i := range NumLRTs {
					if LRTCode(i) == code {
						d.SetLRT(i, true)
					}
				}
			}
			r.FactoryLessGermanium = true
		}, want{-3667, true, 2500, 3, 1457}},
		{"P4", func(d *Design) { d.Race.Env[engine.Gravity].Center = 51 }, want{245, true, 1700, 10, 525}},
		{"P5", func(d *Design) { d.Stat15 = 1 }, want{245, false, 1000, 10, 245}},
		{"P6", func(d *Design) { d.Race.ColonistsPerResource = 2600 }, want{845, false, 2500, 10, 845}},
		{"P7", func(d *Design) { d.Race.PRT = engine.PRT(99) }, want{299, false, 1000, 10, 299}},
		{"P8", func(d *Design) { d.Race.Env[engine.Gravity].Low, d.Race.Env[engine.Gravity].Center = -5, 40 }, want{212, true, 1800, 10, 532}},
		{"P9", func(d *Design) { d.Race.ColonistsPerResource, d.Race.FactoryOutput, d.Race.MineOutput = 900, 8, 12 }, want{0, false, 900, 10, 0}},
		{"P10", func(d *Design) { d.Race.ColonistsPerResource, d.Race.FactoryOutput, d.Race.MineOutput = 1400, 7, 19 }, want{-1, true, 2500, 9, 559}},
	}
	for _, c := range cases {
		d := pg001()
		c.edit(&d)
		if got := Points(d); got != c.want.points {
			t.Errorf("%s: %d points after the edit, want %d", c.id, got, c.want.points)
		}
		budget := 15
		res := YearlyCheck(&d, &budget, false)
		if res.Punished != c.want.punished || d.Tampered != c.want.punished {
			t.Errorf("%s: punished %v tampered %v, want %v", c.id, res.Punished, d.Tampered, c.want.punished)
		}
		if d.Race.ColonistsPerResource != c.want.col || d.Race.GrowthRate != c.want.growth {
			t.Errorf("%s: colonists %d growth %d, want %d, %d", c.id, d.Race.ColonistsPerResource, d.Race.GrowthRate, c.want.col, c.want.growth)
		}
		if got := Points(d); got != c.want.after {
			t.Errorf("%s: %d points after the check, want %d", c.id, got, c.want.after)
		}
		// A second year changes nothing (BINARY-ONLY).
		before := d
		if res := YearlyCheck(&d, &budget, false); res.Punished || d != before {
			t.Errorf("%s: second year changed the race", c.id)
		}
	}
	// P8's repaired gravity axis: 0–85, centre 42.
	d := pg001()
	d.Race.Env[engine.Gravity].Low, d.Race.Env[engine.Gravity].Center = -5, 40
	YearlyCheck(&d, nil, false)
	if e := d.Race.Env[engine.Gravity]; e.Low != 0 || e.High != 85 || e.Center != 42 {
		t.Errorf("P8 gravity %+v", e)
	}
	// Silent research-share clamp (BINARY-ONLY) and computer races.
	d = pg001()
	d.Race.FactoryCost, d.Race.MineCost = 5, 2
	budget := 140
	if res := YearlyCheck(&d, &budget, true); res.Punished || !res.Clamped || budget != 15 || d.Tampered {
		t.Errorf("computer: %+v budget %d tampered %v", res, budget, d.Tampered)
	}
	// A computer race gets the scoring repair and the tampered flag, but
	// no penalty (RACES.md "In a running game" step 5, BINARY-ONLY).
	d = pg001()
	d.Race.Env[engine.Gravity].Low, d.Race.Env[engine.Gravity].Center = -5, 40
	before := d
	if res := YearlyCheck(&d, nil, true); res.Punished {
		t.Errorf("computer punished: %+v", res)
	}
	if e := d.Race.Env[engine.Gravity]; !d.Tampered || e.Low != 0 || e.Center != 42 ||
		d.Race.GrowthRate != before.Race.GrowthRate || d.Race.ColonistsPerResource != before.Race.ColonistsPerResource {
		t.Errorf("computer repair: tampered %v gravity %+v", d.Tampered, e)
	}
}

// Repairs (RACES.md "Repairs", BINARY-ONLY): the immune marker in the low
// makes an axis immune; research costs outside the settings go to the
// nearest end.
func TestPredictionRepairs(t *testing.T) {
	d := Default()
	d.Race.Env[engine.Radiation] = engine.EnvRange{Low: ImmuneMarker, Center: 50, High: 80}
	if !scoringRepair(&d) || d.Race.Env[engine.Radiation] != (engine.EnvRange{Immune: true}) {
		t.Errorf("immune marker: %+v", d.Race.Env[engine.Radiation])
	}
	d = Default()
	d.Race.Env[engine.Radiation] = engine.EnvRange{Immune: true, Center: 3}
	if !scoringRepair(&d) || d.Race.Env[engine.Radiation] != (engine.EnvRange{Immune: true}) {
		t.Errorf("immune centre: %+v", d.Race.Env[engine.Radiation])
	}
	d = Default()
	d.Race.ResearchCosts[0], d.Race.ResearchCosts[1] = -100, 1000
	clampSettings(&d)
	if d.Race.ResearchCosts[0] != engine.ResearchExpensive || d.Race.ResearchCosts[1] != engine.ResearchCheap {
		t.Errorf("research clamp: %v", d.Race.ResearchCosts)
	}
}

// Game creation (RACES.md "At game creation", CONFIRMED RD-4).
func TestConfirmedAtCreation(t *testing.T) {
	byID := map[string]Design{}
	for _, c := range rdCases(t) {
		byID[c.id] = c.race
	}
	// rw04b (−1) and rw04c (−1433): replaced by the default race, L 25.
	for _, id := range []string{"rw04b", "rw04c"} {
		r := AtCreation(byID[id], false)
		def := Default()
		def.Tampered = true
		if !r.Replaced || r.Race != def || Leftover(r.Points) != 25 {
			t.Errorf("%s: %+v", id, r)
		}
		// Computer players are not checked.
		if r := AtCreation(byID[id], true); r.Replaced || r.Race != byID[id] {
			t.Errorf("%s as a computer race: %+v", id, r)
		}
	}
	// rw04d: a centre one off is repaired and the race kept, tampered.
	d := byID["rw04d"]
	d.Race.Env[engine.Temperature].Center++
	if r := AtCreation(d, false); r.Replaced || !r.Race.Tampered || r.Race.Race.Env != byID["rw04d"].Race.Env || r.Points != 34 {
		t.Errorf("rw04d: %+v", r)
	}
	// rw04e: stat 15 = 1 is reset and marks the race.
	d = byID["rw04e"]
	d.Stat15 = 1
	if r := AtCreation(d, false); !r.Race.Tampered || r.Race.Stat15 != 0 || r.Points != 32 {
		t.Errorf("rw04e: %+v", r)
	}
	// rw04f: growth 0 becomes 1 (5851 points, L 50).
	if r := AtCreation(byID["rw04f"], false); !r.Race.Tampered || r.Race.Race.GrowthRate != 1 || r.Points != 5851 || Leftover(r.Points) != 50 {
		t.Errorf("rw04f: %+v", r)
	}
	// An untouched legal race is kept as is.
	if r := AtCreation(byID["rw04a"], false); r.Race.Tampered || r.Points != 0 || Leftover(r.Points) != 0 {
		t.Errorf("rw04a: %+v", r)
	}
	if Leftover(51) != 50 || Leftover(50) != 50 {
		t.Error("leftover cap")
	}
	if EffectiveSpend(5) != 0 || EffectiveSpend(6) != 0 || EffectiveSpend(3) != 3 {
		t.Error("spends 5 and 6")
	}
}

type splitmix struct{ s uint64 }

func (r *splitmix) Intn(n int) int {
	r.s += 0x9e3779b97f4a7c15
	z := r.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return int(z % uint64(n))
}

// Random races score 0..50 (RACES.md "Random race", CONFIRMED in outcome:
// 14 generated races, 3..44 points) and keep their name.
func TestConfirmedRandomRaces(t *testing.T) {
	rng := &splitmix{s: 1}
	fallback := 0
	prts := map[engine.PRT]int{}
	for i := range 500 {
		d := Generate("Zorgon", rng)
		p := Points(d)
		if p < 0 || p > 50 {
			t.Fatalf("race %d: %d points", i, p)
		}
		if d.Name != "Zorgon" || d.Random {
			t.Fatalf("race %d: name %q random %v", i, d.Name, d.Random)
		}
		if r, changed := Repaired(d); changed && r.Race.Env != d.Race.Env {
			t.Fatalf("race %d: generated a malformed habitat %+v", i, d.Race.Env)
		}
		if d == func() Design { x := Default(); x.Name = "Zorgon"; return x }() {
			fallback++
		}
		prts[d.Race.PRT]++
	}
	if len(prts) != 10 {
		t.Errorf("PRTs seen %v", prts)
	}
	if fallback > 25 {
		t.Errorf("%d of 500 races fell back to the default race", fallback)
	}
	if tpl := RandomTemplate("Random"); tpl.Race.PRT != engine.PRTHyperExpansion || !tpl.Random || tpl.Race.Env[0].Low != 17 {
		t.Errorf("template %+v", tpl)
	}
}
