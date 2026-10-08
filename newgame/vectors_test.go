package newgame

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/races"
)

// The universe-generation parity vectors (stars-elegy vectors corpus
// "ug", FORMAT.md "new_game"), copied into engine/testdata/vectors by the
// kernel lane. The engine's parity harness has no new-game path, so this
// test runs them through Generate itself.
//
// Player expectations (starting tech, ship design count) are exact. The
// samples are random outcomes of the original's stream; each is checked
// for the parts its constraint says the rules decide, over ugSeeds seeds
// (docs/UNIVERSE-STATUS.md "UG vectors").

const ugSeeds = 8

type ugVector struct {
	ID      string `json:"id"`
	NewGame struct {
		Settings struct {
			Size      string          `json:"size"`
			Density   string          `json:"density"`
			Positions int             `json:"player_positions"`
			Options   map[string]bool `json:"options"`
			Slower    bool            `json:"slower_tech"`
			Random    bool            `json:"random_events"`
			Public    bool            `json:"public_scores"`
			Victory   [][]int         `json:"victory_condition_lines"`
		} `json:"settings"`
		Races []ugRace `json:"races"`
	} `json:"new_game"`
	Cases []struct {
		ID     string `json:"id"`
		Tag    string `json:"tag"`
		Expect []struct {
			Kind     string          `json:"kind"`
			ID       int             `json:"id"`
			Check    string          `json:"check"`
			Sample   bool            `json:"sample"`
			Equals   json.RawMessage `json:"equals"`
			Observed json.RawMessage `json:"observed"`
		} `json:"expect"`
	} `json:"cases"`
}

type ugRace struct {
	Player   int `json:"player"`
	Computer *struct {
		Race  int `json:"race"`
		Level int `json:"level"`
		// Drawn is the type and level a random choice drew, recovered
		// from the recorded race (AI.md §3); absent in older vectors.
		Drawn *struct {
			Type  int `json:"type"`
			Level int `json:"level"`
		} `json:"drawn"`
	} `json:"computer"`
	Race struct {
		PRT    string            `json:"prt"`
		LRT    []string          `json:"lrt"`
		Growth int               `json:"growth_percent"`
		Hab    map[string]any    `json:"habitability"`
		Pop    int               `json:"colonists_per_resource"`
		Fact   map[string]int    `json:"factory"`
		Mine   map[string]int    `json:"mine"`
		Cost   map[string]string `json:"research_cost"`
		Spend  json.RawMessage   `json:"leftover_spend"`
		Stat15 int               `json:"stat_15"`
		High   bool              `json:"techs_start_high"`
		Less   bool              `json:"factories_cost_less"`
	} `json:"race"`
	Spend  *int `json:"leftover_spend_code"`
	Random bool `json:"random"`
}

var ugSpends = map[string]int{"surface_minerals": 0, "mineral_concentrations": 1, "mines": 2, "factories": 3, "defenses": 4}

var ugPRTs = map[string]engine.PRT{"HE": engine.PRTHyperExpansion, "SS": engine.PRTSuperStealth, "WM": engine.PRTWarMonger,
	"CA": engine.PRTClaimAdjuster, "IS": engine.PRTInnerStrength, "SD": engine.PRTSpaceDemolition, "PP": engine.PRTPacketPhysics,
	"IT": engine.PRTInterstellarTraveler, "AR": engine.PRTAlternateReality, "JOAT": engine.PRTJackOfAllTrades}

var ugFields = []string{"energy", "weapons", "propulsion", "construction", "electronics", "biotechnology"}

func (r ugRace) design(t *testing.T) races.Design {
	d := races.Design{Stat15: r.Race.Stat15, ExpensiveAt3: r.Race.High, Random: r.Random}
	switch {
	case r.Spend != nil:
		d.Spend = *r.Spend
	case len(r.Race.Spend) > 0:
		var name string
		if json.Unmarshal(r.Race.Spend, &name) == nil {
			code, ok := ugSpends[name]
			if !ok {
				t.Fatalf("leftover spend %q", name)
			}
			d.Spend = code
		} else if err := json.Unmarshal(r.Race.Spend, &d.Spend); err != nil {
			t.Fatalf("leftover spend %s", r.Race.Spend)
		}
	}
	x := &d.Race
	prt, ok := ugPRTs[r.Race.PRT]
	if !ok {
		t.Fatalf("PRT %q", r.Race.PRT)
	}
	x.PRT, x.GrowthRate, x.ColonistsPerResource = prt, r.Race.Growth, r.Race.Pop
	x.FactoryOutput, x.FactoryCost, x.FactoriesOperated = r.Race.Fact["output"], r.Race.Fact["cost"], r.Race.Fact["per_10k"]
	x.MineOutput, x.MineCost, x.MinesOperated = r.Race.Mine["output"], r.Race.Mine["cost"], r.Race.Mine["per_10k"]
	x.FactoryLessGermanium = r.Race.Less
	for _, code := range r.Race.LRT {
		found := false
		for i := range races.NumLRTs {
			if races.LRTCode(i) == code {
				d.SetLRT(i, true)
				found = true
			}
		}
		if !found {
			t.Fatalf("LRT %q", code)
		}
	}
	for a, name := range [3]string{"gravity", "temperature", "radiation"} {
		v, _ := r.Race.Hab[name].([]any)
		if len(v) != 3 {
			t.Fatalf("habitability %s: %v", name, r.Race.Hab[name])
		}
		c, lo, hi := int(v[0].(float64)), int(v[1].(float64)), int(v[2].(float64))
		// 255 is the stored immune marker (RACES.md "Repairs"); a race file
		// may hold it in one field only, which creation repairs (RW08).
		marker := func(v int) int {
			if v == 255 {
				return races.ImmuneMarker
			}
			return v
		}
		if c == 255 && lo == 255 && hi == 255 {
			x.Env[a] = engine.EnvRange{Immune: true}
		} else {
			x.Env[a] = engine.EnvRange{Center: marker(c), Low: marker(lo), High: marker(hi)}
		}
	}
	for f, name := range ugFields {
		switch r.Race.Cost[name] {
		case "expensive":
			x.ResearchCosts[f] = engine.ResearchExpensive
		case "cheap":
			x.ResearchCosts[f] = engine.ResearchCheap
		}
	}
	return d
}

func (v ugVector) settings(t *testing.T) Settings {
	st := v.NewGame.Settings
	sizes := map[string]Size{"tiny": Tiny, "small": Small, "medium": Medium, "large": Large, "huge": Huge}
	dens := map[string]Density{"sparse": Sparse, "normal": Normal, "dense": Dense, "packed": Packed}
	s := Settings{
		Size: sizes[st.Size], Density: dens[st.Density], Positions: Positions(st.Positions),
		MaxMinerals: st.Options["maximum_minerals"], SlowerTech: st.Slower, BBS: st.Options["accelerated_bbs"],
		NoRandomEvents: !st.Random, Clumping: st.Options["galaxy_clumping"], PublicScores: st.Public,
		ComputerAlliances: st.Options["flag_5"],
	}
	if vl := st.Victory; len(vl) == 8 {
		vic := &s.Victory
		for c := range engine.NumVictory {
			vic.Enabled[c] = vl[c][0] == 1
		}
		vic.Planets = vl[0][1]/5 - 4
		vic.TechLevel, vic.TechFields = vl[1][1]-8, vl[1][2]-2
		vic.Score = vl[2][1]/1000 - 1
		vic.Lead = vl[3][1]/10 - 2
		vic.Resources = vl[4][1]/10 - 1
		vic.Capital = vl[5][1]/10 - 1
		vic.Highest = vl[6][1]/10 - 3
		vic.Needed, vic.MinYears = vl[7][0], vl[7][1]/10-3
	} else {
		t.Fatalf("%s: victory_condition_lines %v", v.ID, vl)
	}
	ps := append([]ugRace(nil), v.NewGame.Races...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].Player < ps[j].Player })
	for _, r := range ps {
		p := PlayerSetup{Race: r.design(t)}
		if r.Computer != nil {
			// Levels 1..4 are easy..expert; 0 is a random level, given by
			// computer.drawn where the vector records it.
			l := r.Computer.Level
			if d := r.Computer.Drawn; l == 0 && d != nil {
				l = d.Level
			}
			p.Computer, p.Level = true, Level(max(0, l-1))
		}
		s.Players = append(s.Players, p)
	}
	return s
}

func loadUG(t *testing.T) []ugVector {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "engine", "testdata", "vectors", "ug", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no ug vectors: %v", err)
	}
	// The race-creation vectors (corpus "rw", RD-1..RD-7, RW08) have the
	// same form; their human races are race files, and case -R gives
	// each human player's race as created.
	rw, err := filepath.Glob(filepath.Join("..", "engine", "testdata", "vectors", "rw", "*.json"))
	if err != nil || len(rw) == 0 {
		t.Fatalf("no rw vectors: %v", err)
	}
	paths = append(paths, rw...)
	var out []ugVector
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var v ugVector
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out = append(out, v)
	}
	return out
}

// ugPlanet is a starting planet as the vectors' starting_planets sample
// gives it.
type ugPlanet struct {
	Population int    `json:"population"`
	Mines      int    `json:"mines"`
	Factories  int    `json:"factories"`
	Defenses   int    `json:"defenses"`
	Surface    [3]int `json:"surface_minerals"`
	Conc       [3]int `json:"concentrations"`
}

type ugStart struct {
	Homeworld []ugPlanet `json:"homeworld"`
	Other     []ugPlanet `json:"other_planets"`
	Fleets    int        `json:"fleets"`
}

func startOf(res Result, i int) ugStart {
	pl := func(pi int) ugPlanet {
		p := res.Game.Planets[pi]
		out := ugPlanet{Population: p.Population, Mines: p.Mines, Factories: p.Factories, Defenses: p.Defenses, Surface: p.Surface}
		for k := range out.Conc {
			out.Conc[k] = p.Deposits[k].Concentration
		}
		return out
	}
	ps := res.Players[i]
	s := ugStart{Homeworld: []ugPlanet{pl(ps.Homeworld)}, Fleets: len(ps.Fleets)}
	if ps.SecondPlanet >= 0 {
		s.Other = append(s.Other, pl(ps.SecondPlanet))
	}
	return s
}

// installs drops the random parts of a starting planet: surface
// minerals and concentrations come from the stream.
func installs(p ugPlanet) [4]int { return [4]int{p.Population, p.Mines, p.Factories, p.Defenses} }

func wormholeRange(s Size) (int, int) {
	r := [5][2]int{{0, 2}, {1, 3}, {1, 5}, {3, 6}, {4, 8}}[s]
	return r[0], r[1]
}

// nominal is UNIVERSE.md "Count" step 1's N.
func nominal(s Settings) int {
	w := s.Width()
	n := w * w / 5000
	n += n / 4 * (int(s.Density) - 1)
	if s.Density == Packed {
		n += n / 4
	}
	return min(n, 999)
}

// decodeVictory is engine.Victory in the vectors' game units.
func decodeVictory(v engine.Victory) map[string]any {
	val := func(x int, on bool) map[string]any { return map[string]any{"value": x, "enabled": on} }
	e := v.Enabled
	return map[string]any{
		"planets_owned_percent":     val((v.Planets+4)*5, e[engine.VictoryPlanets]),
		"tech_level":                val(v.TechLevel+8, e[engine.VictoryTech]),
		"tech_fields":               map[string]any{"value": v.TechFields + 2},
		"score":                     val((v.Score+1)*1000, e[engine.VictoryScore]),
		"lead_over_second_percent":  val((v.Lead+2)*10, e[engine.VictoryLead]),
		"resources_thousands":       val((v.Resources+1)*10, e[engine.VictoryResources]),
		"capital_ships":             val((v.Capital+1)*10, e[engine.VictoryCapital]),
		"highest_score_after_years": val((v.Highest+3)*10, e[engine.VictoryHighest]),
		"conditions_needed":         map[string]any{"value": v.Needed},
		"minimum_years":             map[string]any{"value": (v.MinYears + 3) * 10},
	}
}

// TestUGVectors runs the universe-generation vectors. Exact (player)
// expectations fail the test; samples log what differs and are tallied.
func TestUGVectors(t *testing.T) {
	type tally struct{ pass, fail int }
	byCheck := map[string]*tally{}
	note := func(check string, ok bool, format string, args ...any) {
		if byCheck[check] == nil {
			byCheck[check] = &tally{}
		}
		if ok {
			byCheck[check].pass++
			return
		}
		byCheck[check].fail++
		t.Logf("differs "+format, args...)
	}
	for _, v := range loadUG(t) {
		s := v.settings(t)
		// A Random race is generated from the stream, so its created race
		// (case -R, samples) cannot match Elegy's. Play the recorded race
		// in its place so the rules downstream of the race are still
		// checked; Generate's own Random races are tested in races/.
		created := map[int]ugRace{}
		for _, c := range v.Cases {
			for _, e := range c.Expect {
				var eq struct {
					Race *json.RawMessage `json:"race"`
				}
				if e.Kind == "player" && json.Unmarshal(e.Equals, &eq) == nil && eq.Race != nil {
					var r ugRace
					if err := json.Unmarshal([]byte(`{"race":`+string(*eq.Race)+`}`), &r); err != nil {
						t.Fatalf("%s: %v", c.ID, err)
					}
					created[e.ID] = r
				}
			}
		}
		for i := range s.Players {
			if s.Players[i].Race.Random {
				r, ok := created[i]
				if !ok {
					t.Fatalf("%s: Random race of player %d not recorded", v.ID, i)
				}
				s.Players[i].Race = r.design(t)
			}
		}
		gen := func(s Settings) []Result {
			var out []Result
			for seed := range uint64(ugSeeds) {
				res, err := Generate(s, NewRand(seed+1))
				if err != nil {
					t.Errorf("%s seed %d: %v", v.ID, seed+1, err)
					continue
				}
				out = append(out, res)
			}
			return out
		}
		results := gen(s)
		if len(results) == 0 {
			continue
		}
		res := results[0]
		for _, c := range v.Cases {
			for _, e := range c.Expect {
				switch {
				case e.Kind == "player" && created[e.ID].Race.PRT != "" && strings.HasSuffix(c.ID, "-R"):
					if v.NewGame.Races[e.ID].Random {
						continue // a sample: played as recorded above
					}
					want := created[e.ID].design(t)
					got := res.Players[e.ID].Race
					got.Name, got.Tampered, got.Random = "", false, false
					if got != want {
						t.Errorf("%s player %d race as created:\n got  %+v\n want %+v", c.ID, e.ID, got, want)
					}
				case e.Kind == "player":
					var eq struct {
						Tech    map[string]int `json:"tech"`
						Designs *int           `json:"ship_design_count"`
					}
					if err := json.Unmarshal(e.Equals, &eq); err != nil {
						t.Fatalf("%s: %v", c.ID, err)
					}
					lv := res.Game.Players[e.ID].Research.Levels
					for f, name := range ugFields {
						if want, ok := eq.Tech[name]; ok && lv[f] != want {
							t.Errorf("%s player %d: %s %d, want %d", c.ID, e.ID, name, lv[f], want)
						}
					}
					if eq.Designs != nil && len(res.Players[e.ID].ShipDesigns) != *eq.Designs {
						t.Errorf("%s player %d: %d ship designs, want %d", c.ID, e.ID, len(res.Players[e.ID].ShipDesigns), *eq.Designs)
					}
				case e.Check == "planet_count":
					// Seed-dependent (UNIVERSE.md "Count" step 4): the
					// count is N unless the spacing pass removed more
					// than M − N. Check both sides stay at or below N and
					// that Elegy reaches N when the original did.
					var want int
					json.Unmarshal(e.Observed, &want)
					n := nominal(s)
					var got []int
					ok, hitN := want <= n, false
					for _, r := range results {
						got = append(got, len(r.Game.Planets))
						ok = ok && len(r.Game.Planets) <= n
						hitN = hitN || len(r.Game.Planets) == n
					}
					note("planet_count", ok && (want < n || hitN), "%s planet count %d, N %d, Elegy %v", c.ID, want, n, got)
				case e.Check == "wormholes":
					// The vectors count wormhole ends: every observed
					// count is even, and as pairs it falls in OBJECTS.md's
					// ranges.
					var ends int
					json.Unmarshal(e.Observed, &ends)
					lo, hi := wormholeRange(s.Size)
					if s.NoRandomEvents {
						lo, hi = 0, 0
					}
					ok := ends%2 == 0 && ends/2 >= lo && ends/2 <= hi
					var got []int
					for _, r := range results {
						got = append(got, len(r.Wormholes))
						ok = ok && len(r.Wormholes) >= lo && len(r.Wormholes) <= hi
					}
					note("wormholes", ok, "%s wormholes: observed %d ends, Elegy pairs %v, range %d..%d pairs", c.ID, ends, got, lo, hi)
				case e.Check == "starting_planets":
					var obs map[string]ugStart
					if err := json.Unmarshal(e.Observed, &obs); err != nil {
						t.Fatalf("%s: %v", c.ID, err)
					}
					for i := range res.Players {
						want, ok := obs[fmt.Sprint(i)]
						if !ok {
							continue
						}
						diffs := startDiffs(startOf(res, i), want)
						level := ""
						if r := v.NewGame.Races[i]; diffs != nil && r.Computer != nil && r.Computer.Level == 0 && r.Computer.Drawn == nil {
							// A random level the vector does not record:
							// try each level for this player.
							for l := Easy; l <= Expert && diffs != nil; l++ {
								s2 := s
								s2.Players = append([]PlayerSetup(nil), s.Players...)
								s2.Players[i].Level = l
								if rs := gen(s2); len(rs) > 0 && startDiffs(startOf(rs[0], i), want) == nil {
									diffs, level = nil, fmt.Sprintf(" (random level: matches level %d)", l+1)
								}
							}
						}
						if level != "" {
							t.Logf("%s player %d%s", c.ID, i, level)
						}
						note("starting_planets", diffs == nil, "%s player %d (%s): %s", c.ID, i, v.NewGame.Races[i].Race.PRT, strings.Join(diffs, "; "))
					}
				case e.Check == "stored_victory_conditions":
					var obs map[string]any
					json.Unmarshal(e.Observed, &obs)
					got := decodeVictory(res.Game.Victory)
					gb, _ := json.Marshal(got)
					var gotNorm map[string]any
					json.Unmarshal(gb, &gotNorm)
					note("stored_victory_conditions", fmt.Sprint(gotNorm) == fmt.Sprint(obs), "%s victory: %v, want %v", c.ID, gotNorm, obs)
				}
			}
		}
	}
	keys := make([]string, 0, len(byCheck))
	for k := range byCheck {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%-26s pass %3d  differ %3d", k, byCheck[k].pass, byCheck[k].fail)
		if byCheck[k].fail > 0 {
			t.Errorf("%s: %d samples differ (run with -v)", k, byCheck[k].fail)
		}
	}
}

// startDiffs compares the parts of a starting_planets sample the rules
// decide: population and installations of the homeworld and the second
// planet, and the fleet count. Surface minerals and concentrations are
// draws.
func startDiffs(got, want ugStart) []string {
	var diffs []string
	if installs(got.Homeworld[0]) != installs(want.Homeworld[0]) {
		diffs = append(diffs, fmt.Sprintf("homeworld pop/mines/factories/defenses %v, want %v", installs(got.Homeworld[0]), installs(want.Homeworld[0])))
	}
	if len(got.Other) != len(want.Other) {
		diffs = append(diffs, fmt.Sprintf("%d other planets, want %d", len(got.Other), len(want.Other)))
	} else if len(got.Other) == 1 && installs(got.Other[0]) != installs(want.Other[0]) {
		diffs = append(diffs, fmt.Sprintf("second planet %v, want %v", installs(got.Other[0]), installs(want.Other[0])))
	}
	if got.Fleets != want.Fleets {
		diffs = append(diffs, fmt.Sprintf("%d fleets, want %d", got.Fleets, want.Fleets))
	}
	return diffs
}
