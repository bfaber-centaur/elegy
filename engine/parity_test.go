package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Parity harness: runs the stars-elegy parity vectors (testdata/vectors,
// format in its FORMAT.md) through GenerateTurn and compares each case's
// expectations. A case is skipped, not failed, when it needs something
// Elegy does not model yet (a task, target, object or expectation kind),
// or when its outcome is random. CONFIRMED and LEGACY BUG cases are
// exact-match targets; MEASURED ones are reported only.
//
// testdata/vectors/baseline.txt lists the cases that pass; the test fails
// when one of them fails. Run with -v to see the per-corpus report, and
// with PARITY_BASELINE=write to rewrite the baseline.

type pvVector struct {
	ID           string   `json:"id"`
	Years        int      `json:"years"`
	Random       string   `json:"random"`
	InitialState pvState  `json:"initial_state"`
	Cases        []pvCase `json:"cases"`
	// Orders are the orders players submitted (FORMAT.md "orders").
	Orders []pvOrderBlock `json:"orders"`
	path   string         // corpus/run
	corpus string
}

type pvState struct {
	Year            int            `json:"year"`
	Game            pvGame         `json:"game"`
	Players         []pvPlayer     `json:"players"`
	Planets         []pvPlanet     `json:"planets"`
	Designs         []pvDesign     `json:"designs"`
	StarbaseDesigns []pvDesign     `json:"starbase_designs"`
	BattlePlans     []pvBattlePlan `json:"battle_plans"`
	Fleets          []pvFleet      `json:"fleets"`
	Objects         []struct {
		Kind string `json:"kind"`
	} `json:"objects"`
	ProductionQueues []struct {
		Planet int               `json:"planet"`
		Items  []json.RawMessage `json:"items"`
	} `json:"production_queues"`
}

type pvGame struct {
	Size         string `json:"size"`
	RandomEvents bool   `json:"random_events"`
	SlowerTech   bool   `json:"slower_tech"`
}

type pvPlayer struct {
	ID                  int               `json:"id"`
	Tech                map[string]int    `json:"tech"`
	ResearchAccumulated map[string]int    `json:"research_accumulated"`
	ResearchPercent     int               `json:"research_percent"`
	ResearchField       string            `json:"research_field"`
	ResearchNextField   json.RawMessage   `json:"research_next_field"`
	Relations           map[string]string `json:"relations"`
	MysteryTraderItems  []string          `json:"mystery_trader_items"`
	Race                struct {
		PRT           json.RawMessage `json:"prt"`
		LRT           []string
		GrowthPercent int `json:"growth_percent"`
		Habitability  struct {
			Gravity     [3]int `json:"gravity"`
			Temperature [3]int `json:"temperature"`
			Radiation   [3]int `json:"radiation"`
		} `json:"habitability"`
		ColonistsPerResource int               `json:"colonists_per_resource"`
		Factory              pvEconomy         `json:"factory"`
		Mine                 pvEconomy         `json:"mine"`
		ResearchCost         map[string]string `json:"research_cost"`
		FactoriesCostLess    bool              `json:"factories_cost_less"`
	} `json:"race"`
}

type pvEconomy struct {
	Output, Cost int
	Per10k       int `json:"per_10k"`
}

type pvPlanet struct {
	ID                  int   `json:"id"`
	X                   int   `json:"x"`
	Y                   int   `json:"y"`
	Owner               int   `json:"owner"`
	Concentrations      []int `json:"concentrations"`
	Environment         []int `json:"environment"`
	OriginalEnvironment []int `json:"original_environment"`
	SurfaceMinerals     []int `json:"surface_minerals"`
	Population          *int  `json:"population"`
	Excess              *int  `json:"excess"`
	Mines               *int  `json:"mines"`
	Factories           *int  `json:"factories"`
	Defenses            *int  `json:"defenses"`
	PlanetaryScanner    *int  `json:"planetary_scanner"`
	LeftoverToResearch  *bool `json:"leftover_to_research"`
	Starbase            *struct {
		Design *int `json:"design"`
		Damage int  `json:"damage"`
	} `json:"starbase"`
}

type pvDesign struct {
	Owner int       `json:"owner"`
	Slot  int       `json:"slot"`
	Hull  string    `json:"hull"`
	Slots []*pvPart `json:"slots"`
	Mass  int       `json:"mass"`
}

type pvPart struct {
	Count int    `json:"count"`
	Part  string `json:"part"`
}

type pvBattlePlan struct {
	Owner, Slot, Tactic, Primary, Secondary int
	AttackWho                               int  `json:"attack_who"`
	DumpCargo                               bool `json:"dump_cargo"`
}

type pvFleet struct {
	Owner      int            `json:"owner"`
	ID         int            `json:"id"`
	X          int            `json:"x"`
	Y          int            `json:"y"`
	Ships      []pvShips      `json:"ships"`
	Cargo      map[string]int `json:"cargo"`
	Fuel       int            `json:"fuel"`
	BattlePlan int            `json:"battle_plan"`
	Waypoints  []pvWaypoint   `json:"waypoints"`
}

type pvShips struct {
	Design int `json:"design"`
	Count  int `json:"count"`
	Damage *struct {
		Units          int `json:"units"`
		PercentOfShips int `json:"percent_of_ships"`
	} `json:"damage"`
}

type pvWaypoint struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Warp   int `json:"warp"`
	Target struct {
		Kind  string `json:"kind"`
		ID    *int   `json:"id"`
		Owner *int   `json:"owner"`
	} `json:"target"`
	Task struct {
		Kind   string `json:"kind"`
		Orders map[string]struct {
			Action string `json:"action"`
			Value  int    `json:"value"`
		} `json:"orders"`
	} `json:"task"`
}

type pvCase struct {
	ID             string     `json:"id"`
	Rule           string     `json:"rule"`
	Tag            string     `json:"tag"`
	VariesByStream bool       `json:"varies_by_stream"`
	Expect         []pvExpect `json:"expect"`
}

type pvExpect struct {
	Year   int             `json:"year"`
	Kind   string          `json:"kind"`
	Stream string          `json:"stream"`
	Owner  *int            `json:"owner"`
	ID     *int            `json:"id"`
	Planet *int            `json:"planet"`
	Slot   *int            `json:"slot"`
	X      *int            `json:"x"`
	Y      *int            `json:"y"`
	Equals json.RawMessage `json:"equals"`
	// Tolerance is, as an object, the difference allowed per field.
	Tolerance json.RawMessage `json:"tolerance"`
}

func (p *pvBattlePlan) UnmarshalJSON(b []byte) error {
	var q struct {
		Owner     int  `json:"owner"`
		Slot      int  `json:"slot"`
		Tactic    int  `json:"tactic"`
		Primary   int  `json:"primary"`
		Secondary int  `json:"secondary"`
		AttackWho int  `json:"attack_who"`
		DumpCargo bool `json:"dump_cargo"`
	}
	if err := json.Unmarshal(b, &q); err != nil {
		return err
	}
	*p = pvBattlePlan{q.Owner, q.Slot, q.Tactic, q.Primary, q.Secondary, q.AttackWho, q.DumpCargo}
	return nil
}

func (e *pvEconomy) UnmarshalJSON(b []byte) error {
	var q struct {
		Output int `json:"output"`
		Cost   int `json:"cost"`
		Per10k int `json:"per_10k"`
	}
	if err := json.Unmarshal(b, &q); err != nil {
		return err
	}
	*e = pvEconomy{q.Output, q.Cost, q.Per10k}
	return nil
}

var pvFields = map[string]int{"energy": Energy, "weapons": Weapons, "propulsion": Propulsion,
	"construction": Construction, "electronics": Electronics, "biotechnology": Biotech}

var pvPRTs = map[string]PRT{"HE": PRTHyperExpansion, "JOAT": PRTJackOfAllTrades, "AR": PRTAlternateReality,
	"IS": PRTInnerStrength, "WM": PRTWarMonger, "IT": PRTInterstellarTraveler, "CA": PRTClaimAdjuster,
	"SS": PRTSuperStealth, "PP": PRTPacketPhysics, "SD": PRTSpaceDemolition}

var pvCargo = map[string]int{"ironium": Ironium, "boranium": Boranium, "germanium": Germanium, "colonists": CargoColonists}

// pvLoaded is a vector's start as an Elegy game, with the id maps the
// checks need.
type pvLoaded struct {
	g       Game
	fleetID map[[2]int]int // (owner, vector id) → Elegy fleet id
	design  map[[2]int]int // (owner, design slot) → Game.Designs index
	planet  map[int]int    // vector planet id → Elegy planet id
	// unsupported names, per fleet (owner, id), what Elegy does not model
	// in its orders; a case about such a fleet is skipped.
	unsupported map[[2]int]string
	start       map[[2]int]Point // each fleet's starting position
	// global is a reason every case of the vector is skipped, if any.
	global string
	// queued marks the players with a production queue the harness does
	// not load.
	queued map[int]bool
}

func pvFleetKey(owner, id int) [2]int { return [2]int{owner, id} }

// pvElegyFleetID keeps fleet order: by owner, then the vector's number.
func pvElegyFleetID(owner, id int) int { return owner*100000 + id + 1 }

func loadVector(v *pvVector) (*pvLoaded, error) {
	s := v.InitialState
	cat := Components()
	l := &pvLoaded{fleetID: map[[2]int]int{}, design: map[[2]int]int{}, planet: map[int]int{}, unsupported: map[[2]int]string{}, start: map[[2]int]Point{}, queued: map[int]bool{}}
	g := &l.g
	g.Year = s.Year
	g.RandomEvents = s.Game.RandomEvents
	g.SlowerTech = s.Game.SlowerTech
	g.Size = map[string]int{"tiny": 0, "small": 1, "medium": 2, "large": 3, "huge": 4}[s.Game.Size]
	g.PlanetScanners = cat.PlanetScanners()
	g.Defenses = cat.Defenses()

	g.Players = make([]Player, len(s.Players))
	for _, p := range s.Players {
		pl := &g.Players[p.ID]
		r := &pl.Race
		// An out-of-range PRT is given as its stored number (FORMAT.md).
		var name string
		if json.Unmarshal(p.Race.PRT, &name) != nil {
			l.global = "out-of-range PRT " + string(p.Race.PRT)
			continue
		}
		prt, ok := pvPRTs[name]
		if !ok {
			return nil, fmt.Errorf("PRT %q", name)
		}
		r.PRT = prt
		for _, t := range p.Race.LRT {
			switch t {
			case "IFE":
				r.LRT.ImprovedFuelEfficiency = true
			case "TT":
				r.LRT.TotalTerraforming = true
			case "ARM":
				r.LRT.AdvancedRemoteMining = true
			case "ISB":
				r.LRT.ImprovedStarbases = true
			case "GR":
				r.LRT.GeneralizedResearch = true
			case "UR":
				r.LRT.UltimateRecycling = true
			case "MA":
				r.LRT.MineralAlchemy = true
			case "NRSE":
				r.LRT.NoRamScoopEngines = true
			case "CE":
				r.LRT.CheapEngines = true
			case "OBRM":
				r.LRT.OnlyBasicRemoteMining = true
			case "NAS":
				r.LRT.NoAdvancedScanners = true
			case "LSP":
				r.LRT.LowStartingPopulation = true
			case "BET":
				r.LRT.BleedingEdgeTech = true
			case "RS":
				r.LRT.RegeneratingShields = true
			default:
				return nil, fmt.Errorf("LRT %q", t)
			}
		}
		r.GrowthRate = p.Race.GrowthPercent
		for axis, h := range [3][3]int{p.Race.Habitability.Gravity, p.Race.Habitability.Temperature, p.Race.Habitability.Radiation} {
			r.Env[axis] = EnvRange{Center: h[0], Low: h[1], High: h[2], Immune: h[0] == 255}
		}
		r.ColonistsPerResource = p.Race.ColonistsPerResource
		r.FactoryOutput, r.FactoryCost, r.FactoriesOperated = p.Race.Factory.Output, p.Race.Factory.Cost, p.Race.Factory.Per10k
		r.MineOutput, r.MineCost, r.MinesOperated = p.Race.Mine.Output, p.Race.Mine.Cost, p.Race.Mine.Per10k
		r.FactoryLessGermanium = p.Race.FactoriesCostLess
		for name, c := range p.Race.ResearchCost {
			r.ResearchCosts[pvFields[name]] = map[string]ResearchCost{"normal": ResearchNormal, "expensive": ResearchExpensive, "cheap": ResearchCheap}[c]
		}
		for name, lv := range p.Tech {
			pl.Research.Levels[pvFields[name]] = lv
		}
		for name, a := range p.ResearchAccumulated {
			pl.Research.Accumulated[pvFields[name]] = a
		}
		pl.Research.Current = pvFields[p.ResearchField]
		pl.Research.Next = NextSameField
		if len(p.ResearchNextField) > 0 {
			var next string
			if json.Unmarshal(p.ResearchNextField, &next) != nil {
				return nil, fmt.Errorf("research_next_field %s", p.ResearchNextField)
			}
			switch next {
			case "same":
			case "lowest":
				pl.Research.Next = NextLowestField
			default:
				f, ok := pvFields[next]
				if !ok {
					return nil, fmt.Errorf("research_next_field %q", next)
				}
				pl.Research.Next = f
			}
		}
		pl.ResearchBudget = p.ResearchPercent
		pl.Relations = make([]Relation, len(s.Players))
		for i := range pl.Relations {
			pl.Relations[i] = RelationNeutral
		}
		for other, rel := range p.Relations {
			var o int
			fmt.Sscan(other, &o)
			pl.Relations[o] = map[string]Relation{"friend": RelationFriend, "neutral": RelationNeutral, "enemy": RelationEnemy}[rel]
		}
	}

	addDesign := func(d pvDesign, starbase bool) error {
		var fills []SlotFill
		engineSlot, engines := -1, 0
		for i, sl := range d.Slots {
			if sl == nil {
				continue
			}
			fills = append(fills, SlotFill{Slot: i, Part: sl.Part, Count: sl.Count})
		}
		name := fmt.Sprintf("p%d d%d", d.Owner, d.Slot)
		ds, err := cat.NewDesign(name, d.Hull, fills)
		if err != nil && !starbase {
			// A design without a full set of engines (FM-105): build it
			// with the slot filled, then take the missing engines off.
			hc, ok := cat.Lookup(d.Hull)
			if !ok {
				return err
			}
			h, herr := hc.Hull()
			if herr != nil || len(h.Slots) == 0 {
				return err
			}
			engineSlot = 0
			for i := range fills {
				if fills[i].Slot == 0 {
					engines = fills[i].Count
					fills[i].Count = h.Slots[0].Max
				}
			}
			ds, err = cat.NewDesign(name, d.Hull, fills)
			if err != nil || engines == 0 {
				return fmt.Errorf("design %s: %v", name, err)
			}
			ec, _ := cat.Lookup(ds.Engine.Name)
			ep, _ := ec.Part()
			ds.Mass -= (h.Slots[0].Max - engines) * ep.Mass
			ds.Engines = engines
			for i := range ds.Slots {
				if ds.Slots[i].Part.Kind == PartEngine {
					ds.Slots[i].Count = engines
				}
			}
		} else if err != nil {
			return err
		}
		_ = engineSlot
		key := [2]int{d.Owner, d.Slot}
		if starbase {
			key[1] = -1 - d.Slot
		}
		l.design[key] = len(g.Designs)
		g.Designs = append(g.Designs, ds)
		return nil
	}
	for _, d := range s.Designs {
		if err := addDesign(d, false); err != nil {
			return nil, err
		}
	}
	for _, d := range s.StarbaseDesigns {
		if err := addDesign(d, true); err != nil {
			return nil, err
		}
	}

	for _, bp := range s.BattlePlans {
		pl := &g.Players[bp.Owner]
		for len(pl.Plans) <= bp.Slot {
			pl.Plans = append(pl.Plans, BattlePlan{})
		}
		plan := BattlePlan{Tactic: Tactic(bp.Tactic), Primary: TargetType(bp.Primary), Secondary: TargetType(bp.Secondary), DumpCargo: bp.DumpCargo}
		if bp.AttackWho >= int(AttackPlayer) {
			plan.Attack, plan.Player = AttackPlayer, bp.AttackWho-int(AttackPlayer)
		} else {
			plan.Attack = AttackWho(bp.AttackWho)
		}
		pl.Plans[bp.Slot] = plan
	}

	for _, p := range s.Planets {
		l.planet[p.ID] = p.ID + 1
		pl := Planet{ID: p.ID + 1, Pos: Point{p.X, p.Y}, Owner: p.Owner, StarbaseDesign: -1}
		if p.Owner < 0 {
			pl.Owner = NoOwner
		}
		for m, c := range p.Concentrations {
			pl.Deposits[m].Concentration = c
		}
		copy(pl.Env[:], p.Environment)
		copy(pl.OrigEnv[:], p.Environment)
		if p.OriginalEnvironment != nil {
			copy(pl.OrigEnv[:], p.OriginalEnvironment)
		}
		copy(pl.Surface[:], p.SurfaceMinerals)
		set := func(dst *int, v *int) {
			if v != nil {
				*dst = *v
			}
		}
		set(&pl.Population, p.Population)
		set(&pl.GrowthCarry, p.Excess)
		set(&pl.Mines, p.Mines)
		set(&pl.Factories, p.Factories)
		set(&pl.Defenses, p.Defenses)
		pl.HasScanner = p.PlanetaryScanner != nil
		if p.LeftoverToResearch != nil {
			pl.LeftoverOnly = *p.LeftoverToResearch
		}
		if pl.Owner != NoOwner {
			// A planet the state lists no queue for has none: its
			// resources all go to research (KERNEL.md "Production").
			for _, q := range s.ProductionQueues {
				pl.HasQueue = pl.HasQueue || q.Planet == p.ID
			}
			pl.Homeworld = false
		}
		if sb := p.Starbase; sb != nil && sb.Design != nil && pl.Owner != NoOwner {
			di, ok := l.design[[2]int{p.Owner, -1 - *sb.Design}]
			if !ok {
				return nil, fmt.Errorf("planet %d: starbase design %d", p.ID, *sb.Design)
			}
			d := g.Designs[di]
			pl.HasStarbase, pl.StarbaseDesign, pl.StarbaseDamage = true, di, max(0, sb.Damage)
			pl.StarbaseHull = d.Hull.StarbaseNumber
			pl.StarbaseDock = d.Hull.StarbaseNumber >= 2
		}
		g.Planets = append(g.Planets, pl)
	}
	if len(s.Objects) > 0 {
		l.global = "space objects (" + s.Objects[0].Kind + ")"
	}
	// Production queues (FORMAT.md "Queue items"). A queue holding an
	// item Elegy does not build (designs, terraforming, packets, scanners,
	// the Genesis Device) is not loaded: checks its planet, its owner or
	// ships it may build could see are skipped.
	for _, q := range s.ProductionQueues {
		pi := -1
		for i := range g.Planets {
			if g.Planets[i].ID == l.planet[q.Planet] {
				pi = i
			}
		}
		if pi < 0 || g.Planets[pi].Owner == NoOwner {
			continue
		}
		p := &g.Planets[pi]
		items, why := pvQueue(q.Items)
		if why == "" {
			p.HasQueue, p.Queue = true, items
			continue
		}
		key := [2]int{p.Owner, -1 - q.Planet}
		l.unsupported[key] = why
		l.start[key] = p.Pos
		l.queued[p.Owner] = true
	}
	for _, f := range s.Fleets {
		l.fleetID[pvFleetKey(f.Owner, f.ID)] = pvElegyFleetID(f.Owner, f.ID)
		l.start[pvFleetKey(f.Owner, f.ID)] = Point{f.X, f.Y}
	}
	for _, f := range s.Fleets {
		key := pvFleetKey(f.Owner, f.ID)
		ef := Fleet{ID: l.fleetID[key], Number: f.ID, Owner: f.Owner, Pos: Point{f.X, f.Y}, Fuel: f.Fuel, Plan: f.BattlePlan}
		for _, sh := range f.Ships {
			st := Stack{Design: l.design[[2]int{f.Owner, sh.Design}], Count: sh.Count}
			if sh.Damage != nil {
				st.Damage = Damage{Pct: sh.Damage.PercentOfShips, Units: sh.Damage.Units}
			}
			ef.Stacks = append(ef.Stacks, st)
		}
		for name, v := range f.Cargo {
			if c := pvCargo[name]; c == CargoColonists {
				ef.Cargo.Colonists = v
			} else {
				ef.Cargo.Minerals[c] = v
			}
		}
		for i, w := range f.Waypoints {
			task, why := l.task(w)
			if why != "" {
				l.unsupported[key] = why
				if w.Target.Kind == "fleet" && w.Target.ID != nil {
					owner := f.Owner
					if w.Target.Owner != nil {
						owner = *w.Target.Owner
					}
					l.unsupported[pvFleetKey(owner, *w.Target.ID)] = why + " (target)"
				}
			}
			if i == 0 {
				ef.Task = task
				continue
			}
			wp, why := l.waypoint(f.Owner, w)
			if why != "" {
				l.unsupported[key] = why
			}
			wp.Task = task
			ef.Waypoints = append(ef.Waypoints, wp)
		}
		g.Fleets = append(g.Fleets, ef)
	}
	return l, nil
}

// pvItemKinds maps the planetary queue item ids Elegy builds to its
// items (FORMAT.md "Queue items").
var pvItemKinds = map[int]ItemKind{
	0: ItemAutoMines, 1: ItemAutoFactories, 2: ItemAutoDefenses, 3: ItemAutoAlchemy,
	7: ItemFactory, 8: ItemMine, 9: ItemDefenses, 11: ItemMineralAlchemy,
}

// queueEquals checks a planet's production queue (FORMAT.md
// "production_queue").
func (l *pvLoaded) queueEquals(g *Game, e pvExpect) string {
	if e.Planet == nil {
		return "skip: production_queue without a planet"
	}
	if why := l.unsupportedAt(-1); why != "" {
		return "skip: " + why
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(e.Equals, &raw); err != nil {
		return "skip: production_queue " + err.Error()
	}
	want, why := pvQueue(raw)
	if why != "" {
		return "skip: " + why
	}
	for i := range g.Planets {
		if g.Planets[i].ID == l.planet[*e.Planet] {
			got := g.Planets[i].Queue
			if len(got) == 0 && len(want) == 0 || reflect.DeepEqual(got, want) {
				return ""
			}
			return pvMismatch("queue", got, want)
		}
	}
	return "no planet"
}

// pvQueue converts a production queue, or names an item Elegy does not
// build.
func pvQueue(raw []json.RawMessage) ([]QueueItem, string) {
	var items []QueueItem
	for _, r := range raw {
		var it struct{ ID, Count, Percent, Kind int }
		if err := json.Unmarshal(r, &it); err != nil {
			return nil, "production queue item " + string(r)
		}
		if it.Kind != 1 {
			return nil, "production queue: a design"
		}
		k, ok := pvItemKinds[it.ID]
		if !ok {
			return nil, fmt.Sprintf("production queue: planetary item %d", it.ID)
		}
		items = append(items, QueueItem{Kind: k, Count: it.Count, Percent: it.Percent})
	}
	return items, ""
}

// waypoint converts a waypoint's position, warp and target for a fleet of
// owner, or names what Elegy cannot model. A stargate jump (warp 11)
// holds the fleet.
func (l *pvLoaded) waypoint(owner int, w pvWaypoint) (Waypoint, string) {
	wp := Waypoint{Pos: Point{w.X, w.Y}, Warp: w.Warp}
	why := ""
	if w.Warp > 10 {
		why, wp.Warp = "stargate", 0
	}
	switch w.Target.Kind {
	case "space":
	case "planet":
		wp.Target, wp.ID = TargetPlanet, l.planet[*w.Target.ID]
	case "fleet":
		if w.Target.Owner != nil {
			owner = *w.Target.Owner
		}
		wp.Target, wp.ID = TargetFleet, l.fleetID[pvFleetKey(owner, *w.Target.ID)]
	default:
		why = "target " + w.Target.Kind
	}
	return wp, why
}

// task converts a waypoint task, or names what Elegy cannot model.
func (l *pvLoaded) task(w pvWaypoint) (Task, string) {
	switch w.Task.Kind {
	case "none":
		return Task{}, ""
	case "colonize":
		return Task{Kind: TaskColonize}, ""
	case "merge":
		owner := 0
		if w.Target.Owner != nil {
			owner = *w.Target.Owner
		}
		if w.Target.Kind != "fleet" || w.Target.ID == nil {
			return Task{}, "merge without a fleet target"
		}
		return Task{Kind: TaskMerge, Fleet: l.fleetID[pvFleetKey(owner, *w.Target.ID)]}, ""
	case "transport":
		if w.Target.Kind == "fleet" {
			return Task{}, "fleet-to-fleet transport"
		}
		t := Task{Kind: TaskTransport}
		for cargo, o := range w.Task.Orders {
			c, ok := pvCargo[cargo]
			if !ok {
				return Task{}, "transport " + cargo
			}
			switch o.Action {
			case "unload_all":
				t.Transport[c] = Transport{Action: UnloadAll}
			case "unload_exactly":
				t.Transport[c] = Transport{Action: UnloadExactly, Amount: o.Value}
			default:
				return Task{}, "transport " + o.Action
			}
		}
		return t, ""
	}
	return Task{}, "task " + w.Task.Kind
}

func loadVectors(t *testing.T) []*pvVector {
	paths, err := filepath.Glob(filepath.Join("testdata", "vectors", "*", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no vectors: %v", err)
	}
	sort.Strings(paths)
	var out []*pvVector
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var v pvVector
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		v.corpus = filepath.Base(filepath.Dir(p))
		v.path = v.corpus + "/" + filepath.Base(p)
		out = append(out, &v)
	}
	return out
}

// pvResult is one case's outcome.
type pvResult struct {
	id, corpus, tag string
	status          string // pass, fail, skip
	why             string
}

// pvCheck compares one expectation with a generated game; "" passes, a
// leading "skip: " skips.
func (l *pvLoaded) pvCheck(g *Game, e pvExpect) string {
	if e.Kind == "production_queue" {
		return l.queueEquals(g, e)
	}
	var eq map[string]json.RawMessage
	if len(e.Equals) > 0 && json.Unmarshal(e.Equals, &eq) != nil {
		return "skip: " + e.Kind + " list"
	}
	fleet := func(owner, id int) *Fleet {
		fid := l.fleetID[pvFleetKey(owner, id)]
		for i := range g.Fleets {
			if g.Fleets[i].ID == fid && fid != 0 {
				return &g.Fleets[i]
			}
		}
		return nil
	}
	switch e.Kind {
	case "fleet", "fleet_gone":
		if _, ok := l.fleetID[pvFleetKey(*e.Owner, *e.ID)]; !ok && l.queued[*e.Owner] {
			return "skip: fleet built from a production queue"
		}
	}
	switch e.Kind {
	case "fleet":
		if why := l.unsupportedNear(pvFleetKey(*e.Owner, *e.ID)); why != "" {
			return "skip: " + why
		}
		f := fleet(*e.Owner, *e.ID)
		if f == nil {
			return "fleet gone"
		}
		return l.fleetEquals(f, eq)
	case "fleet_at":
		for _, k := range pvKeys2(l.unsupported) {
			if l.start[k] == (Point{*e.X, *e.Y}) {
				return "skip: " + l.unsupported[k] + " nearby"
			}
		}
		var last string
		for i := range g.Fleets {
			f := &g.Fleets[i]
			if f.Owner == *e.Owner && f.Pos == (Point{*e.X, *e.Y}) {
				if last = l.fleetEquals(f, eq); last == "" {
					return ""
				}
			}
		}
		if last == "" {
			last = "no fleet there"
		}
		return last
	case "no_fleet_at":
		for _, f := range g.Fleets {
			if f.Owner == *e.Owner && f.Pos == (Point{*e.X, *e.Y}) {
				return fmt.Sprintf("fleet %d at (%d,%d)", f.ID, *e.X, *e.Y)
			}
		}
		return ""
	case "fleet_gone":
		if why := l.unsupportedNear(pvFleetKey(*e.Owner, *e.ID)); why != "" {
			return "skip: " + why
		}
		if fleet(*e.Owner, *e.ID) != nil {
			return "fleet still exists"
		}
		return ""
	case "planet":
		id := *e.ID
		if e.Planet != nil {
			id = *e.Planet
		}
		if why := l.unsupportedAt(-1); why != "" {
			return "skip: " + why
		}
		for i := range g.Planets {
			if g.Planets[i].ID == l.planet[id] {
				var tol map[string]int
				json.Unmarshal(e.Tolerance, &tol)
				return l.planetEquals(g, &g.Planets[i], eq, tol)
			}
		}
		return "no planet"
	case "player":
		if *e.ID >= len(g.Players) {
			return "skip: player not in the state"
		}
		if l.queued[*e.ID] {
			return "skip: production queue"
		}
		if g.Players[*e.ID].Race.PRT == PRTSuperStealth && len(g.Players) > 1 {
			if _, ok := eq["research_accumulated"]; ok {
				return "skip: Super Stealth research stealing"
			}
		}
		return l.playerEquals(g, *e.ID, eq)
	case "design":
		return l.designEquals(g, *e.Owner, *e.Slot, eq)
	}
	return "skip: expectation " + e.Kind
}

// unsupportedAt is a reason to skip a check that any of owner's fleets
// (or any fleet, for owner −1) could affect.
func (l *pvLoaded) unsupportedAt(owner int) string {
	for _, k := range pvKeys2(l.unsupported) {
		if owner < 0 || k[0] == owner {
			return l.unsupported[k]
		}
	}
	return ""
}

// unsupportedNear is a reason to skip a check on fleet key: it, or a fleet
// that started at its position, has an order Elegy does not model.
func (l *pvLoaded) unsupportedNear(key [2]int) string {
	if why := l.unsupported[key]; why != "" {
		return why
	}
	for _, k := range pvKeys2(l.unsupported) {
		if l.start[k] == l.start[key] {
			return l.unsupported[k] + " nearby"
		}
	}
	return ""
}

func pvKeys2(m map[[2]int]string) [][2]int {
	keys := make([][2]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		return keys[a][0] < keys[b][0] || keys[a][0] == keys[b][0] && keys[a][1] < keys[b][1]
	})
	return keys
}

func pvMismatch(field string, got, want any) string {
	return fmt.Sprintf("%s = %v, want %v", field, got, want)
}

func (l *pvLoaded) fleetEquals(f *Fleet, eq map[string]json.RawMessage) string {
	var errs []string
	for _, k := range pvKeys(eq) {
		raw := eq[k]
		switch k {
		case "x", "y", "fuel":
			var want int
			json.Unmarshal(raw, &want)
			got := map[string]int{"x": f.Pos.X, "y": f.Pos.Y, "fuel": f.Fuel}[k]
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "next_waypoint_warp":
			var want *int
			json.Unmarshal(raw, &want)
			var got *int
			if len(f.Waypoints) > 0 {
				got = &f.Waypoints[0].Warp
			}
			if (got == nil) != (want == nil) || got != nil && *got != *want {
				errs = append(errs, pvMismatch(k, pvPtr(got), pvPtr(want)))
			}
		case "cargo":
			var want map[string]int
			json.Unmarshal(raw, &want)
			for _, c := range pvKeys(want) {
				got := 0
				switch c {
				case "colonists":
					got = f.Cargo.Colonists
				case "fuel":
					got = f.Fuel
				default:
					got = f.Cargo.Minerals[pvCargo[c]]
				}
				if got != want[c] {
					errs = append(errs, pvMismatch("cargo."+c, got, want[c]))
				}
			}
		case "ships":
			var want []struct{ Design, Count int }
			json.Unmarshal(raw, &want)
			got := map[int]int{}
			for _, s := range f.Stacks {
				got[s.Design] += s.Count
			}
			w := map[int]int{}
			for _, s := range want {
				w[l.design[[2]int{f.Owner, s.Design}]] += s.Count
			}
			if !reflect.DeepEqual(got, w) {
				errs = append(errs, pvMismatch("ships", got, w))
			}
		case "damage":
			var want []struct {
				Design         int `json:"design"`
				Units          int `json:"units"`
				PercentOfShips int `json:"percent_of_ships"`
			}
			json.Unmarshal(raw, &want)
			for _, d := range want {
				di := l.design[[2]int{f.Owner, d.Design}]
				found := false
				for _, s := range f.Stacks {
					if s.Design == di {
						found = true
						if s.Damage != (Damage{Pct: d.PercentOfShips, Units: d.Units}) {
							errs = append(errs, pvMismatch(fmt.Sprintf("damage[%d]", d.Design), s.Damage, d))
						}
					}
				}
				if !found {
					errs = append(errs, fmt.Sprintf("damage: no stack of design %d", d.Design))
				}
			}
		case "first_waypoint_task":
			var want string
			json.Unmarshal(raw, &want)
			got := map[TaskKind]string{TaskNone: "none", TaskTransport: "transport", TaskColonize: "colonize", TaskMerge: "merge"}[f.Task.Kind]
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		default:
			return "skip: fleet field " + k
		}
	}
	return strings.Join(errs, "; ")
}

func (l *pvLoaded) planetEquals(g *Game, p *Planet, eq map[string]json.RawMessage, tol map[string]int) string {
	var errs []string
	for _, k := range pvKeys(eq) {
		raw := eq[k]
		switch k {
		case "owner":
			var want int
			json.Unmarshal(raw, &want)
			got := p.Owner
			if got == NoOwner {
				got = -1
			}
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "population", "defenses", "mines", "factories", "excess":
			var want int
			json.Unmarshal(raw, &want)
			got := map[string]int{"population": p.Population, "defenses": p.Defenses,
				"mines": p.Mines, "factories": p.Factories, "excess": p.GrowthCarry}[k]
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "surface_minerals":
			var want [3]int
			json.Unmarshal(raw, &want)
			for m, w := range want {
				if d := p.Surface[m] - w; d > tol[k] || -d > tol[k] {
					errs = append(errs, pvMismatch(k, p.Surface, want))
					break
				}
			}
		case "environment", "original_environment":
			var want *[3]int
			json.Unmarshal(raw, &want)
			got := p.Env
			if k == "original_environment" {
				got = p.OrigEnv
			}
			if want == nil {
				return "skip: original environment null"
			}
			if got != *want {
				errs = append(errs, pvMismatch(k, got, *want))
			}
		case "leftover_to_research":
			// A captured planet takes its new owner's default setting,
			// which the vectors do not carry.
			return "skip: player default queue settings"
		default:
			return "skip: planet field " + k
		}
	}
	return strings.Join(errs, "; ")
}

func (l *pvLoaded) playerEquals(g *Game, id int, eq map[string]json.RawMessage) string {
	var errs []string
	for _, k := range pvKeys(eq) {
		switch k {
		case "tech", "research_accumulated":
			var want map[string]int
			json.Unmarshal(eq[k], &want)
			for _, f := range pvKeys(want) {
				got := g.Players[id].Research.Levels[pvFields[f]]
				if k == "research_accumulated" {
					got = g.Players[id].Research.Accumulated[pvFields[f]]
				}
				if got != want[f] {
					errs = append(errs, pvMismatch(k+"."+f, got, want[f]))
				}
			}
		default:
			return "skip: player field " + k
		}
	}
	return strings.Join(errs, "; ")
}

// designEquals checks a design after the design read (Catalog.ReadDesign)
// with the owner's race and levels.
func (l *pvLoaded) designEquals(g *Game, owner, slot int, eq map[string]json.RawMessage) string {
	return "skip: design read needs the order layer's design source"
}

func pvPtr(p *int) any {
	if p == nil {
		return "none"
	}
	return *p
}

func pvKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// runVector generates the vector's years once and checks every case.
func runVector(v *pvVector) []pvResult {
	var out []pvResult
	res := func(c pvCase, status, why string) {
		out = append(out, pvResult{id: c.ID, corpus: v.corpus, tag: c.Tag, status: status, why: why})
	}
	l, err := loadVector(v)
	if err != nil {
		for _, c := range v.Cases {
			res(c, "skip", "load: "+err.Error())
		}
		return out
	}
	if v.corpus == "rp" {
		// Every rp case tests the race penalty.
		l.global = "the turn-time race check (RACES.md \"In a running game\")"
	}
	if l.global != "" {
		for _, c := range v.Cases {
			res(c, "skip", l.global)
		}
		return out
	}
	// Generate year by year, keeping each year's game.
	games := []Game{l.g}
	g := l.g
	var genErr error
	for y := 1; y <= v.Years; y++ {
		files, why := l.orders(v.Orders, y, &g)
		if why != "" {
			for _, c := range v.Cases {
				res(c, "skip", why)
			}
			return out
		}
		r, err := GenerateTurn(g, files, Jrc3(), rand.New(rand.NewSource(int64(y))))
		if err != nil {
			genErr = err
			break
		}
		g = r.Game
		games = append(games, g)
	}
	for _, c := range v.Cases {
		if l.global != "" {
			res(c, "skip", l.global)
			continue
		}
		if why := pvNotModelled[c.ID]; why != "" {
			res(c, "skip", why)
			continue
		}
		if c.VariesByStream {
			res(c, "skip", "random outcome")
			continue
		}
		var fails, skips []string
		checked := 0
		for _, e := range c.Expect {
			if e.Stream != "" {
				skips = append(skips, "stream "+e.Stream)
				continue
			}
			if e.Year >= len(games) {
				fails = append(fails, fmt.Sprintf("year %d not generated: %v", e.Year, genErr))
				continue
			}
			msg := l.pvCheck(&games[e.Year], e)
			switch {
			case strings.HasPrefix(msg, "skip: "):
				skips = append(skips, strings.TrimPrefix(msg, "skip: "))
			case msg != "":
				checked++
				fails = append(fails, fmt.Sprintf("y%d %s: %s", e.Year, e.Kind, msg))
			default:
				checked++
			}
		}
		switch {
		case len(fails) > 0 && pvLegacyOff[c.ID] != "":
			res(c, "differs", pvLegacyOff[c.ID]+": "+strings.Join(fails, " | "))
		case len(fails) > 0:
			res(c, "fail", strings.Join(fails, " | "))
		case checked == 0:
			res(c, "skip", strings.Join(pvUnique(skips), ", "))
		default:
			res(c, "pass", "")
		}
	}
	return out
}

const pvBaseline = "testdata/vectors/baseline.txt"

// pvLegacyOff names the LEGACY BUG cases whose switch is off by default,
// where Elegy's chosen rule intentionally differs from the original.
var pvLegacyOff = map[string]string{
	"FO-03-E": "legacyMergeOverflow off",
	"FO-06-G": "legacyMergeOverflow off",
}

// pvNotModelled skips cases that need state the vector does not carry or
// a rule Elegy does not model yet.
var pvNotModelled = map[string]string{
	"KX-002-R3": "the next research field choice is not in the vector",
	"KX-002-R4": "the next research field choice is not in the vector",
}

func pvUnique(xs []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func TestParityVectors(t *testing.T) {
	var all []pvResult
	for _, v := range loadVectors(t) {
		all = append(all, runVector(v)...)
	}
	type tally struct{ pass, fail, skip, measured, differs int }
	per := map[string]*tally{}
	var corpora []string
	for _, r := range all {
		tl := per[r.corpus]
		if tl == nil {
			tl = &tally{}
			per[r.corpus] = tl
			corpora = append(corpora, r.corpus)
		}
		switch {
		case r.status == "skip":
			tl.skip++
		case r.status == "differs":
			tl.differs++
		case r.tag == "MEASURED":
			tl.measured++
		case r.status == "pass":
			tl.pass++
		default:
			tl.fail++
		}
	}
	for _, c := range corpora {
		tl := per[c]
		t.Logf("%-4s pass %3d  fail %3d  skip %3d  differs %3d  measured %3d", c, tl.pass, tl.fail, tl.skip, tl.differs, tl.measured)
	}
	for _, r := range all {
		if r.status != "pass" {
			t.Logf("%s %-10s %-4s %s", r.status, r.tag, r.id, r.why)
		}
	}

	if os.Getenv("PARITY_BASELINE") == "write" {
		f, err := os.Create(pvBaseline)
		if err != nil {
			t.Fatal(err)
		}
		w := bufio.NewWriter(f)
		fmt.Fprintln(w, "# Parity cases Elegy passes (TestParityVectors). One case id per line.")
		for _, r := range all {
			if r.status == "pass" && r.tag != "MEASURED" {
				fmt.Fprintln(w, r.id)
			}
		}
		w.Flush()
		f.Close()
		return
	}
	b, err := os.ReadFile(pvBaseline)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]pvResult{}
	for _, r := range all {
		status[r.id] = r
	}
	for _, line := range strings.Split(string(b), "\n") {
		id := strings.TrimSpace(line)
		if id == "" || strings.HasPrefix(id, "#") {
			continue
		}
		if r := status[id]; r.status != "pass" {
			t.Errorf("baseline case %s now %s: %s", id, r.status, r.why)
		}
	}
}
