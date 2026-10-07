package engine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Ground truth: every fleet of the FM-001..003 oracle corpus (testdata/fm).
// One year of movement from the spec must reproduce the observed position,
// fuel, remaining waypoints, next warp and events.

// fmEngines are the FM corpus engines by catalogue index, with the fuel
// tables of the component table (COMPONENTS.md "Fuel tables", CONFIRMED
// CS-002).
var fmEngines = func() map[int]Engine {
	out := map[int]Engine{}
	for _, c := range Components().Components {
		if c.Category == CatEngine {
			e, err := c.Engine()
			if err != nil {
				panic(err)
			}
			out[c.Index] = e
		}
	}
	return out
}()

// fmDesigns: design id → design (fmlib.py DES; cargo capacities are the
// game's documented hull values: Small Freighter 70 kT, Colony Ship 25 kT).
func fmDesigns() []Design {
	type d struct {
		name                   string
		mass, engine, fuel, cg int
	}
	table := map[int]d{
		0: {"Scout/QJ5", 18, 1, 300, 0},
		1: {"SmallFreighter/QJ5", 31, 1, 130, 70},
		2: {"ColonyShip/QJ5", 56, 1, 200, 25},
		3: {"Scout/LH6", 23, 3, 300, 0},
		4: {"Scout/DLL7", 27, 4, 300, 0},
		5: {"Scout/AD8", 31, 5, 300, 0},
		6: {"Scout/SD", 16, 0, 300, 0},
		7: {"Scout/FM", 20, 2, 300, 0},
		8: {"Scout/RHRS", 24, 10, 300, 0},
		9: {"SmallFreighter/LH6", 36, 3, 130, 70},
	}
	out := make([]Design, 10)
	for id, x := range table {
		out[id] = Design{Name: x.name, Mass: x.mass, Engine: fmEngines[x.engine], Engines: 1, CargoCapacity: x.cg, FuelCapacity: x.fuel}
	}
	return out
}

// fmPlanets is the PG001 universe (fmlib.py PLANETS).
const fmPlanets = "0 1019 1354;1 1031 1365;2 1045 1108;3 1047 1038;4 1057 1168;5 1085 1086;6 1101 1283;7 1113 1298;8 1130 1383;9 1133 1346;10 1133 1116;11 1146 1073;12 1153 1335;13 1156 1114;14 1159 1365;15 1159 1062;16 1164 1024;17 1221 1348;18 1237 1248;19 1244 1140;20 1246 1343;21 1269 1133;22 1270 1250;23 1301 1084;24 1305 1140;25 1320 1057;26 1322 1175;27 1328 1354;28 1337 1268;29 1339 1113;30 1378 1287;31 1386 1022"

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		t.Fatalf("atoi %q: %v", s, err)
	}
	return n
}

func loadFMSpec(t *testing.T, path string) Game {
	t.Helper()
	g := Game{Designs: fmDesigns()}
	for _, p := range strings.Split(fmPlanets, ";") {
		f := strings.Fields(p)
		g.Planets = append(g.Planets, Planet{ID: atoi(t, f[0]), Pos: Point{atoi(t, f[1]), atoi(t, f[2])}, Owner: NoOwner})
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		tok := strings.Fields(line)
		if len(tok) == 0 || tok[0] != "fleet" {
			continue
		}
		f := Fleet{ID: atoi(t, tok[1])}
		for i := 2; i < len(tok); {
			switch tok[i] {
			case "planet":
				i += 2
			case "at":
				f.Pos = Point{atoi(t, tok[i+1]), atoi(t, tok[i+2])}
				i += 3
			case "ships":
				for _, s := range strings.Split(tok[i+1], ",") {
					d, c, _ := strings.Cut(s, ":")
					f.Stacks = append(f.Stacks, Stack{Design: atoi(t, d), Count: atoi(t, c)})
				}
				i += 2
			case "fuel":
				f.Fuel = atoi(t, tok[i+1])
				i += 2
			case "cargo":
				f.Cargo = Cargo{Minerals: Minerals{atoi(t, tok[i+1]), atoi(t, tok[i+2]), atoi(t, tok[i+3])}, Colonists: atoi(t, tok[i+4])}
				i += 5
			case "wp":
				f.Waypoints = append(f.Waypoints, Waypoint{Pos: Point{atoi(t, tok[i+1]), atoi(t, tok[i+2])}, Warp: atoi(t, tok[i+3])})
				i += 4
			case "wpf", "wpp":
				kind := TargetFleet
				if tok[i] == "wpp" {
					kind = TargetPlanet
				}
				f.Waypoints = append(f.Waypoints, Waypoint{Target: kind, ID: atoi(t, tok[i+1]), Pos: Point{atoi(t, tok[i+2]), atoi(t, tok[i+3])}, Warp: atoi(t, tok[i+4])})
				i += 5
			default:
				t.Fatalf("%s: unknown token %q in %q", path, tok[i], line)
			}
		}
		g.Fleets = append(g.Fleets, f)
	}
	return g
}

type fmResult struct {
	id         int
	desc       string
	end        Point
	fuel       int
	waypoints  int // waypoints left after the current position
	nextTarget string
	nextWarp   int
	events     map[EventKind]bool
}

var pointRE = regexp.MustCompile(`\((\d+), ?(\d+)\)`)
var warpRE = regexp.MustCompile(`w(\d+)$`)

func loadFMResults(t *testing.T, path string) []fmResult {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []fmResult
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "id\t") {
			continue
		}
		c := strings.Split(line, "\t")
		end := pointRE.FindStringSubmatch(c[4])
		r := fmResult{
			id:         atoi(t, c[0]),
			desc:       c[1] + " " + c[2],
			end:        Point{atoi(t, end[1]), atoi(t, end[2])},
			fuel:       atoi(t, c[7]),
			waypoints:  atoi(t, c[9]) - 1,
			nextTarget: c[10],
			nextWarp:   atoi(t, warpRE.FindStringSubmatch(c[10])[1]),
			events:     map[EventKind]bool{},
		}
		for _, e := range strings.Fields(c[11]) {
			switch e {
			case "78":
				r.events[EventFleetArrived] = true
			case "139":
				r.events[EventOutOfFuel] = true
			case "243":
				r.events[EventRamScoopFuel] = true
			default:
				t.Fatalf("unknown event %s", e)
			}
		}
		out = append(out, r)
	}
	return out
}

func TestConfirmedFleetMovementCorpus(t *testing.T) {
	for _, name := range []string{"fm001", "fm002", "fm003"} {
		t.Run(name, func(t *testing.T) {
			g := loadFMSpec(t, filepath.Join("testdata", "fm", name+".spec"))
			results := loadFMResults(t, filepath.Join("testdata", "fm", name+".results.tsv"))
			events := moveFleets(&g)
			got := map[int]map[EventKind]bool{}
			for _, e := range events {
				if got[e.Fleet] == nil {
					got[e.Fleet] = map[EventKind]bool{}
				}
				got[e.Fleet][e.Kind] = true
			}
			if len(results) != len(g.Fleets) {
				t.Fatalf("%d results for %d fleets", len(results), len(g.Fleets))
			}
			for _, want := range results {
				f := &g.Fleets[g.fleetIndex(want.id)]
				var diffs []string
				if f.Pos != want.end {
					diffs = append(diffs, fmt.Sprintf("pos %v want %v", f.Pos, want.end))
				}
				if f.Fuel != want.fuel {
					diffs = append(diffs, fmt.Sprintf("fuel %d want %d", f.Fuel, want.fuel))
				}
				if len(f.Waypoints) != want.waypoints {
					diffs = append(diffs, fmt.Sprintf("waypoints %d want %d", len(f.Waypoints), want.waypoints))
				} else if len(f.Waypoints) > 0 {
					wp := f.Waypoints[0]
					if wp.Warp != want.nextWarp {
						diffs = append(diffs, fmt.Sprintf("next warp %d want %d", wp.Warp, want.nextWarp))
					}
					if wp.Target != TargetFleet {
						dest := g.destination(wp)
						if s := fmt.Sprintf("(%d,%d) w%d", dest.X, dest.Y, wp.Warp); s != want.nextTarget {
							diffs = append(diffs, fmt.Sprintf("next %s want %s", s, want.nextTarget))
						}
					}
				}
				if m := regexp.MustCompile(`obj=(\d+)`).FindStringSubmatch(want.nextTarget); m != nil && m[1] != "0" {
					if id, ok := g.OrbitedPlanet(f); !ok || strconv.Itoa(id) != m[1] {
						diffs = append(diffs, fmt.Sprintf("not orbiting planet %s", m[1]))
					}
				}
				for _, k := range []EventKind{EventFleetArrived, EventOutOfFuel, EventRamScoopFuel} {
					if got[f.ID][k] != want.events[k] {
						diffs = append(diffs, fmt.Sprintf("event %d = %v want %v", k, got[f.ID][k], want.events[k]))
					}
				}
				if len(diffs) > 0 {
					t.Errorf("fleet %d (%s): %s", want.id, want.desc, strings.Join(diffs, "; "))
				}
			}
		})
	}
}

// FM-004 has its own results format (id, end, fuel0, fuel1, warp, orbited
// planet, waypoints left, message ids) and adds starbase refuelling: planet
// 7 is player 0's homeworld with a dock-capable starbase.
func TestConfirmedFleetMovementCorpusFM004(t *testing.T) {
	g := loadFMSpec(t, filepath.Join("testdata", "fm", "fm004.spec"))
	for i := range g.Planets {
		if g.Planets[i].ID == 7 {
			g.Planets[i].Owner = 0
			g.Planets[i].StarbaseDock = true
		}
	}
	events := moveFleets(&g)
	refuelFleets(&g)
	got := map[int]map[EventKind]bool{}
	for _, e := range events {
		if got[e.Fleet] == nil {
			got[e.Fleet] = map[EventKind]bool{}
		}
		got[e.Fleet][e.Kind] = true
	}

	data, err := os.ReadFile(filepath.Join("testdata", "fm", "fm004.results.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "id\t") {
			continue
		}
		c := strings.Split(line, "\t")
		for len(c) < 8 {
			c = append(c, "")
		}
		n++
		id := atoi(t, c[0])
		x, y, _ := strings.Cut(c[1], ",")
		want := struct {
			end             Point
			fuel, warp, orb int
			waypoints       int
			events          map[EventKind]bool
		}{Point{atoi(t, x), atoi(t, y)}, atoi(t, c[3]), atoi(t, c[4]), atoi(t, c[5]), atoi(t, c[6]) - 1, map[EventKind]bool{}}
		for _, e := range strings.Split(c[7], ",") {
			switch e {
			case "":
			case "78":
				want.events[EventFleetArrived] = true
			case "139":
				want.events[EventOutOfFuel] = true
			case "243":
				want.events[EventRamScoopFuel] = true
			default:
				t.Fatalf("unknown event %s", e)
			}
		}

		f := &g.Fleets[g.fleetIndex(id)]
		var diffs []string
		if f.Pos != want.end {
			diffs = append(diffs, fmt.Sprintf("pos %v want %v", f.Pos, want.end))
		}
		if f.Fuel != want.fuel {
			diffs = append(diffs, fmt.Sprintf("fuel %d want %d", f.Fuel, want.fuel))
		}
		if len(f.Waypoints) != want.waypoints {
			diffs = append(diffs, fmt.Sprintf("waypoints %d want %d", len(f.Waypoints), want.waypoints))
		} else if len(f.Waypoints) > 0 && f.Waypoints[0].Warp != want.warp {
			diffs = append(diffs, fmt.Sprintf("warp %d want %d", f.Waypoints[0].Warp, want.warp))
		}
		orb := -1
		if p, ok := g.OrbitedPlanet(f); ok {
			orb = p
		}
		if orb != want.orb {
			diffs = append(diffs, fmt.Sprintf("orbit %d want %d", orb, want.orb))
		}
		for _, k := range []EventKind{EventFleetArrived, EventOutOfFuel, EventRamScoopFuel} {
			if got[id][k] != want.events[k] {
				diffs = append(diffs, fmt.Sprintf("event %d = %v want %v", k, got[id][k], want.events[k]))
			}
		}
		if len(diffs) > 0 {
			t.Errorf("fleet %d: %s", id, strings.Join(diffs, "; "))
		}
	}
	if n != len(g.Fleets) {
		t.Fatalf("%d results for %d fleets", n, len(g.Fleets))
	}
}
