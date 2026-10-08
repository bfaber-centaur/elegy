package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Parity harness: runs the stars-elegy parity vectors (testdata/vectors,
// format in its FORMAT.md) through GenerateTurn and compares each case's
// expectations. A case is skipped, not failed, when it needs something
// Elegy does not model yet (a task, target, object or expectation kind),
// or when its outcome is random. CONFIRMED and LEGACY BUG cases are
// exact-match targets; MEASURED passes are tallied apart, and every
// failure, MEASURED or not, counts as a failure.
//
// testdata/vectors/baseline.txt lists the cases that pass, MEASURED ones
// included, and the "random" cases with how many seed variants they pass
// with (pvSeeds); the test fails when a listed case no longer passes, or
// a random one passes with fewer seeds. Run with -v to see the per-corpus
// report, and with PARITY_BASELINE=write to rewrite the baseline.

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
	Year             int               `json:"year"`
	Game             pvGame            `json:"game"`
	Players          []pvPlayer        `json:"players"`
	Planets          []pvPlanet        `json:"planets"`
	Designs          []pvDesign        `json:"designs"`
	StarbaseDesigns  []pvDesign        `json:"starbase_designs"`
	BattlePlans      []pvBattlePlan    `json:"battle_plans"`
	Fleets           []pvFleet         `json:"fleets"`
	Objects          []json.RawMessage `json:"objects"`
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
	Computer            bool              `json:"computer"`
	ResearchField       string            `json:"research_field"`
	ResearchNextField   json.RawMessage   `json:"research_next_field"`
	Relations           map[string]string `json:"relations"`
	MysteryTraderItems  []string          `json:"mystery_trader_items"`
	Race                pvRace            `json:"race"`
}

// pvRace is a vector race (FORMAT.md "race").
type pvRace struct {
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
	LeftoverSpend        string            `json:"leftover_spend"`
	Stat15               int               `json:"stat_15"`
	TechsStartHigh       bool              `json:"techs_start_high"`
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
	RouteTo             *int  `json:"route_to"`
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
	Repeat     bool           `json:"repeat_orders"`
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
		Kind     string          `json:"kind"`
		Range    int             `json:"range"`
		ToPlayer int             `json:"to_player"`
		Years    json.RawMessage `json:"years"`
		Orders   map[string]struct {
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
	// Subject names an object_gone or view expectation's object.
	Subject json.RawMessage `json:"subject"`
	// Sample marks one stream's random outcome (README.md "sample"):
	// a match counts, a mismatch is skipped as a sample.
	Sample bool `json:"sample"`
	// Viewer is a view expectation's viewing player.
	Viewer *int `json:"viewer"`
}

// pvSpace loads a vector's space objects and checks the expectations
// about them. Package objects imports the engine, so the external test
// package supplies it (objects_parity_test.go); without it every vector
// with space objects is skipped.
type pvSpace interface {
	// Load returns the objects of initial_state.objects.
	Load(g *Game, objects []json.RawMessage, m PVMaps) (SpaceObjects, error)
	// Check checks an expectation of kind minefield, wormhole, trader,
	// packet, object or object_gone as pvCheck does ("" passes, a
	// leading "skip: " skips).
	Check(g *Game, e PVExpect, m PVMaps) string
}

// PVMaps maps vector ids to Elegy's: EndID a wormhole end id to its
// waypoint target ID, Planet a planet id to the Elegy planet ID.
type PVMaps struct {
	EndID, Planet map[int]int
}

// PVExpect is an expectation as pvSpace sees it.
type PVExpect struct {
	Kind                       string
	Owner, ID                  *int
	Equals, Tolerance, Subject json.RawMessage
}

var pvSpaceObjects pvSpace

// PVRaceSettings is what the race check needs of a vector player beyond
// the engine's Race (vectors README.md "race").
type PVRaceSettings struct {
	ExpensiveAt3  bool
	Spend, Stat15 int
	Computer      bool
}

// pvRaces builds the game's race checker (KERNEL.md step 2a) from the
// loaded players and their settings. Package races imports the engine,
// so the external test package supplies it (races_parity_test.go).
var pvRaces func(g *Game, settings []PVRaceSettings) RaceChecker

// pvRaceSettingsOf reads a player's wizard settings back from the game's
// race checker, after the year's check (races_parity_test.go).
var pvRaceSettingsOf func(r RaceChecker, player int) (PVRaceSettings, bool)

// pvTerraform is the game's remote mining and Orbital Adjusters
// (terraformer.go). Package terraform imports the engine, so the external
// test package supplies it (terraform_parity_test.go).
var pvTerraform Terraformer

// pvLeftoverSpend maps a race's leftover_spend to its wizard number.
var pvLeftoverSpend = map[string]int{"surface_minerals": 0, "mineral_concentrations": 1, "mines": 2, "factories": 3, "defenses": 4}

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

// pvOutOfRangePRT offsets an out-of-range stored PRT number (FORMAT.md
// "race") into values no PRT constant takes, so the race check's clamp
// sees an invalid PRT (RACES.md "In a running game" step 1, RD-P7).
const pvOutOfRangePRT = 1000

// pvByte reads a stored race setting as the signed byte the original
// keeps (RACES.md "Repairs": a stored 251 is −5, the immune marker 255 is
// −1).
func pvByte(v int) int {
	if v > 127 {
		return v - 256
	}
	return v
}

// race is the vector race as an Elegy Race. An axis stored as 255 on all
// three values is immune; any other habitat value is read as a signed
// byte, so a low equal to the immune marker reaches the race check as
// races.ImmuneMarker and is repaired there.
func (q pvRace) race() (Race, error) {
	var r Race
	// An out-of-range PRT is given as its stored number (FORMAT.md).
	var name string
	if json.Unmarshal(q.PRT, &name) != nil {
		var n int
		if err := json.Unmarshal(q.PRT, &n); err != nil {
			return r, fmt.Errorf("PRT %s", q.PRT)
		}
		r.PRT = PRT(pvOutOfRangePRT + n)
	} else {
		prt, ok := pvPRTs[name]
		if !ok {
			return r, fmt.Errorf("PRT %q", name)
		}
		r.PRT = prt
	}
	for _, t := range q.LRT {
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
			return r, fmt.Errorf("LRT %q", t)
		}
	}
	r.GrowthRate = q.GrowthPercent
	for axis, h := range [3][3]int{q.Habitability.Gravity, q.Habitability.Temperature, q.Habitability.Radiation} {
		if h == [3]int{255, 255, 255} {
			r.Env[axis] = EnvRange{Immune: true}
			continue
		}
		r.Env[axis] = EnvRange{Center: pvByte(h[0]), Low: pvByte(h[1]), High: pvByte(h[2])}
	}
	r.ColonistsPerResource = q.ColonistsPerResource
	r.FactoryOutput, r.FactoryCost, r.FactoriesOperated = q.Factory.Output, q.Factory.Cost, q.Factory.Per10k
	r.MineOutput, r.MineCost, r.MinesOperated = q.Mine.Output, q.Mine.Cost, q.Mine.Per10k
	r.FactoryLessGermanium = q.FactoriesCostLess
	for name, c := range q.ResearchCost {
		r.ResearchCosts[pvFields[name]] = map[string]ResearchCost{"normal": ResearchNormal, "expensive": ResearchExpensive, "cheap": ResearchCheap}[c]
	}
	return r, nil
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
	// ordered marks the players the vector gives orders for.
	ordered map[int]bool
	// endID maps a vector wormhole end id to its waypoint target ID
	// (pvWormholeEnds).
	endID map[int]int
	// views is each generated year's player views, by year.
	views [][]PlayerView
}

// pvWormholeEnds pairs the vector's wormhole ends: in id order, an end not
// yet paired starts the next wormhole as end 0 and its partner is end 1.
// The target ID is 2 × wormhole + end (objects.WormholeEndID).
func pvWormholeEnds(objects []json.RawMessage) (map[int]int, error) {
	type end struct {
		Kind    string `json:"kind"`
		ID      int    `json:"id"`
		Partner int    `json:"partner"`
	}
	var ends []end
	for _, raw := range objects {
		var e end
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		if e.Kind == "wormhole" {
			ends = append(ends, e)
		}
	}
	sort.Slice(ends, func(i, j int) bool { return ends[i].ID < ends[j].ID })
	m := map[int]int{}
	n := 0
	for _, e := range ends {
		if _, ok := m[e.ID]; ok {
			continue
		}
		m[e.ID], m[e.Partner] = 2*n, 2*n+1
		n++
	}
	return m, nil
}

func pvFleetKey(owner, id int) [2]int { return [2]int{owner, id} }

// pvElegyFleetID keeps fleet order: by owner, then the vector's number.
func pvElegyFleetID(owner, id int) int { return owner*100000 + id + 1 }

func loadVector(v *pvVector) (*pvLoaded, error) {
	s := v.InitialState
	cat := Components()
	l := &pvLoaded{fleetID: map[[2]int]int{}, design: map[[2]int]int{}, planet: map[int]int{}, unsupported: map[[2]int]string{}, start: map[[2]int]Point{}, queued: map[int]bool{}, ordered: map[int]bool{}}
	for _, b := range v.Orders {
		l.ordered[b.Player] = true
	}
	var err error
	if l.endID, err = pvWormholeEnds(s.Objects); err != nil {
		return nil, err
	}
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
		pl.Computer = p.Computer
		race, err := p.Race.race()
		if err != nil {
			return nil, err
		}
		pl.Race = race
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
	if pvRaces != nil && l.global == "" {
		settings := make([]PVRaceSettings, len(s.Players))
		for _, p := range s.Players {
			spend, ok := pvLeftoverSpend[p.Race.LeftoverSpend]
			if !ok {
				return nil, fmt.Errorf("leftover_spend %q", p.Race.LeftoverSpend)
			}
			settings[p.ID] = PVRaceSettings{ExpensiveAt3: p.Race.TechsStartHigh, Spend: spend, Stat15: p.Race.Stat15, Computer: p.Computer}
		}
		g.Races = pvRaces(g, settings)
	}
	g.Terraform = pvTerraform

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
		g.DesignSlots = append(g.DesignSlots, DesignSlot{Owner: d.Owner, Starbase: starbase, Slot: d.Slot, Design: len(g.Designs)})
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
		if p.RouteTo != nil {
			pl.HasRoute, pl.RouteTo = true, *p.RouteTo+1
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
	switch {
	case pvSpaceObjects != nil:
		// Every game holds space objects, so the Trader's appearance
		// draws run with random events on.
		obj, err := pvSpaceObjects.Load(g, s.Objects, PVMaps{l.endID, l.planet})
		if err != nil {
			l.global = "space objects: " + err.Error()
		}
		g.Objects = obj
	case len(s.Objects) > 0:
		var o struct {
			Kind string `json:"kind"`
		}
		json.Unmarshal(s.Objects[0], &o)
		l.global = "space objects (" + o.Kind + ")"
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
		ef := Fleet{ID: l.fleetID[key], Number: f.ID + 1, Owner: f.Owner, Pos: Point{f.X, f.Y}, Fuel: f.Fuel, Plan: f.BattlePlan, Repeat: f.Repeat}
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
// items (FORMAT.md "Queue items"). The packet items (6, 14–17) stay unmapped: a
// vector's planet carries no packet destination or speed, which they need.
var pvItemKinds = map[int]ItemKind{
	0: ItemAutoMines, 1: ItemAutoFactories, 2: ItemAutoDefenses, 3: ItemAutoAlchemy,
	4: ItemAutoMinTerraform, 5: ItemAutoMaxTerraform,
	7: ItemFactory, 8: ItemMine, 9: ItemDefenses, 11: ItemMineralAlchemy, 12: ItemTerraform,
}

// queueEquals checks a planet's production queue (FORMAT.md
// "production_queue").
func (l *pvLoaded) queueEquals(g *Game, e pvExpect) string {
	if e.Planet == nil {
		return "skip: production_queue without a planet"
	}
	if l.planned(l.startOwner(*e.Planet)) {
		return "skip: " + pvPlannedWhy
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
			for _, p0 := range l.g.Planets {
				if p0.ID == g.Planets[i].ID && p0.Owner != g.Planets[i].Owner {
					// A new owner's planet takes that player's default
					// queue, which the vectors do not carry.
					return "skip: player default queue settings"
				}
			}
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
		if it.Kind == 2 {
			// A design item: id 0–15 is a ship design slot, 16–25 starbase
			// slot id − 16 (FORMAT.md "Queue items", CONFIRMED).
			k, slot := ItemShip, it.ID
			if it.ID >= 16 {
				k, slot = ItemStarbase, it.ID-16
			}
			items = append(items, QueueItem{Kind: k, Count: it.Count, Percent: it.Percent, Slot: slot})
			continue
		}
		if it.Kind == 0 {
			// FORMAT.md lists production_queue expectation items as
			// {id, count, percent}: without a kind the id is ambiguous.
			return nil, "production queue item without a kind"
		}
		if it.Kind != 1 {
			return nil, "production queue: item kind " + strconv.Itoa(it.Kind)
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
// owner, or names what Elegy cannot model. Warp 11 is a stargate jump
// (movement runs it); a higher warp is not modelled.
func (l *pvLoaded) waypoint(owner int, w pvWaypoint) (Waypoint, string) {
	wp := Waypoint{Pos: Point{w.X, w.Y}, Warp: w.Warp}
	why := ""
	if w.Warp > StargateWarp {
		why, wp.Warp = "warp "+strconv.Itoa(w.Warp), 0
	}
	switch w.Target.Kind {
	case "space":
	case "planet":
		wp.Target, wp.ID = TargetPlanet, l.planet[*w.Target.ID]
	case "fleet":
		if w.Target.Owner != nil {
			owner = *w.Target.Owner
		}
		wp.Target, wp.ID = TargetFleet, pvElegyFleetID(owner, *w.Target.ID)
	case "wormhole":
		id, ok := l.endID[*w.Target.ID]
		if !ok {
			why = "wormhole end " + strconv.Itoa(*w.Target.ID)
		}
		wp.Target, wp.ID = TargetWormhole, id
	case "trader":
		wp.Target, wp.ID = TargetTrader, *w.Target.ID
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
	case "route":
		return Task{Kind: TaskRoute}, ""
	case "remote_mine":
		return Task{Kind: TaskRemoteMine}, ""
	case "scrap":
		return Task{Kind: TaskScrap}, ""
	case "transfer":
		return Task{Kind: TaskTransferFleet, Player: w.Task.ToPlayer}, ""
	case "patrol":
		return Task{Kind: TaskPatrol, Range: w.Task.Range}, ""
	case "lay_mines":
		// The years word w lays w + 1 years (PARITY.md "Lay-mines
		// duration", CONFIRMED OB-002-N, OB-019, OB-025).
		var word int
		if json.Unmarshal(w.Task.Years, &word) != nil {
			return Task{Kind: TaskLayMines, Years: YearsIndefinitely}, ""
		}
		return Task{Kind: TaskLayMines, Years: word + 1}, ""
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
	status          string // pass, fail, skip, differs, random
	why             string
	passes          int    // seed variants the case passes with
	refStatus       string // for a random case, the reference seed's status
	// skipped is the reasons of the expectations skipped inside a case
	// that was otherwise checked, one per expectation.
	skipped []string
}

// pvCheck compares one expectation with a generated game; "" passes, a
// leading "skip: " skips.
func (l *pvLoaded) pvCheck(g *Game, e pvExpect) string {
	switch e.Kind {
	case "production_queue":
		return l.queueEquals(g, e)
	case "minefield", "wormhole", "trader", "packet", "object", "object_gone":
		if pvSpaceObjects == nil {
			return "skip: space objects"
		}
		// A packet of a player whose queue was not loaded may come from
		// a packet item.
		owner := e.Owner
		if e.Kind == "object" {
			var o struct {
				Kind  string `json:"kind"`
				Owner int    `json:"owner"`
			}
			json.Unmarshal(e.Equals, &o)
			if o.Kind == "packet" {
				owner = &o.Owner
			}
		}
		if (e.Kind == "packet" || e.Kind == "object") && owner != nil && l.queued[*owner] {
			return "skip: a packet from a production queue"
		}
		return pvSpaceObjects.Check(g, PVExpect{Kind: e.Kind, Owner: e.Owner, ID: e.ID, Equals: e.Equals, Tolerance: e.Tolerance, Subject: e.Subject}, PVMaps{l.endID, l.planet})
	}
	var eq map[string]json.RawMessage
	if len(e.Equals) > 0 && json.Unmarshal(e.Equals, &eq) != nil {
		return "skip: " + e.Kind + " list"
	}
	// fleet finds a vector fleet: one from the start by its id, and one
	// made during the run (a gift) by its owner and number.
	fleet := func(owner, id int) *Fleet {
		fid, ok := l.fleetID[pvFleetKey(owner, id)]
		started := map[int]bool{}
		for _, v := range l.fleetID {
			started[v] = true
		}
		for i := range g.Fleets {
			f := &g.Fleets[i]
			if ok && f.ID == fid {
				return f
			}
			// A fleet made during the run (built or given) is known by its
			// fleet number: the vector stores it from 0, the client shows
			// it from 1 (PRODUCTION-LAUNCH.md "The new fleet").
			if !ok && !started[f.ID] && f.Owner == owner && f.Number == id+1 {
				return f
			}
		}
		return nil
	}
	switch e.Kind {
	case "view":
		return l.viewCheck(g, e, eq, fleet)
	case "salvage_at":
		// One of the salvage objects at (x, y) holds exactly these
		// minerals (several when a battle's salvage overflowed, CB-040).
		if why := l.unsupportedAt(-1); why != "" {
			return "skip: " + why
		}
		var want struct{ Minerals []int }
		if json.Unmarshal(e.Equals, &want) != nil || len(want.Minerals) != NumMinerals {
			return "skip: salvage_at " + string(e.Equals)
		}
		var got []Minerals
		for _, s := range g.Salvage {
			if s.Pos == (Point{*e.X, *e.Y}) {
				if s.Minerals == (Minerals{want.Minerals[0], want.Minerals[1], want.Minerals[2]}) {
					return ""
				}
				got = append(got, s.Minerals)
			}
		}
		return pvMismatch("salvage", got, want.Minerals)
	}
	switch e.Kind {
	case "fleet", "fleet_gone", "fleet_at", "no_fleet_at":
		if l.planned(*e.Owner) {
			return "skip: " + pvPlannedWhy
		}
	case "planet":
		id := *e.ID
		if e.Planet != nil {
			id = *e.Planet
		}
		if l.planned(l.startOwner(id)) {
			return "skip: " + pvPlannedWhy
		}
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
		return l.fleetEquals(g, f, eq)
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
				if last = l.fleetEquals(g, f, eq); last == "" {
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

// pvPlannedWhy is the reason a computer player's fleets, planets and
// queues are not compared when the vector carries no orders for it: the
// host plans a computer player's orders each year (FORMAT.md "players"),
// and Elegy's harness runs no planner.
const pvPlannedWhy = "a computer player's planned orders (not in the vector)"

// planned reports whether owner is a computer player the vector gives no
// orders for.
func (l *pvLoaded) planned(owner int) bool {
	return owner >= 0 && owner < len(l.g.Players) && l.g.Players[owner].Computer && !l.ordered[owner]
}

// startOwner is the owner of the vector's planet id at the start, or −1.
func (l *pvLoaded) startOwner(id int) int {
	for _, p := range l.g.Planets {
		if p.ID == l.planet[id] {
			return p.Owner
		}
	}
	return -1
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

// fleetEquals compares a fleet with an expectation's fields. A field the
// harness cannot compare makes the check a skip only when every field it
// did compare matched; a mismatch is always reported.
func (l *pvLoaded) fleetEquals(g *Game, f *Fleet, eq map[string]json.RawMessage) string {
	var errs []string
	skip := ""
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
				d, ok := l.design[[2]int{f.Owner, s.Design}]
				if !ok {
					// A design the player gained during the run (a gift).
					if d, ok = g.PlayerDesign(f.Owner, false, s.Design); !ok {
						d = -1 - s.Design
					}
				}
				w[d] += s.Count
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
		case "orbiting":
			var want *int
			json.Unmarshal(raw, &want)
			id, ok := l.g.OrbitedPlanet(f)
			if w := pvPtr(want); !ok && want != nil || ok && (want == nil || l.planet[*want] != id) {
				got := "null"
				for vid, eid := range l.planet {
					if ok && eid == id {
						got = fmt.Sprint(vid)
					}
				}
				errs = append(errs, pvMismatch(k, got, w))
			}
		case "battle_plan":
			var want int
			json.Unmarshal(raw, &want)
			if f.Plan != want {
				errs = append(errs, pvMismatch(k, f.Plan, want))
			}
		case "repeat_orders":
			var want bool
			json.Unmarshal(raw, &want)
			if f.Repeat != want {
				errs = append(errs, pvMismatch(k, f.Repeat, want))
			}
		case "waypoints":
			var want []pvWaypoint
			json.Unmarshal(raw, &want)
			msg := l.waypointsEqual(f, want)
			if strings.HasPrefix(msg, "skip: ") {
				skip = msg
				continue
			}
			if msg != "" {
				errs = append(errs, msg)
			}
		case "first_waypoint_task":
			var want string
			json.Unmarshal(raw, &want)
			got := map[TaskKind]string{TaskNone: "none", TaskTransport: "transport", TaskColonize: "colonize", TaskMerge: "merge", TaskRoute: "route", TaskPatrol: "patrol", TaskTransferFleet: "transfer", TaskLayMines: "lay_mines", TaskRemoteMine: "remote_mine", TaskScrap: "scrap"}[f.Task.Kind]
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		default:
			skip = "skip: fleet field " + k
		}
	}
	if len(errs) == 0 {
		return skip
	}
	return strings.Join(errs, "; ")
}

// waypointsEqual compares a fleet's waypoint list with the vector's, where
// waypoint 0 is the fleet's own location. Elegy keeps only waypoint 0's
// task (Fleet.Task), so its warp and target are not compared.
func (l *pvLoaded) waypointsEqual(f *Fleet, want []pvWaypoint) string {
	if len(want) == 0 {
		return "skip: no waypoint 0"
	}
	var errs []string
	if p := (Point{want[0].X, want[0].Y}); p != f.Pos {
		errs = append(errs, pvMismatch("waypoints[0] position", f.Pos, p))
	}
	task, why := l.task(want[0])
	if why != "" {
		return "skip: waypoint " + why
	}
	if !reflect.DeepEqual(f.Task, task) {
		errs = append(errs, pvMismatch("waypoints[0] task", f.Task, task))
	}
	if len(f.Waypoints) != len(want)-1 {
		got := make([]Point, len(f.Waypoints))
		for i, wp := range f.Waypoints {
			got[i] = wp.Pos
		}
		return strings.Join(append(errs, fmt.Sprintf("%d waypoints after waypoint 0 %v, want %d", len(f.Waypoints), got, len(want)-1)), "; ")
	}
	for i, w := range want[1:] {
		wp, why := l.waypoint(f.Owner, w)
		if why != "" {
			return "skip: waypoint " + why
		}
		if wp.Task, why = l.task(w); why != "" {
			return "skip: waypoint " + why
		}
		if got := f.Waypoints[i]; !reflect.DeepEqual(got, wp) {
			errs = append(errs, pvMismatch(fmt.Sprintf("waypoints[%d]", i+1), got, wp))
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
		case "race":
			var q pvRace
			if err := json.Unmarshal(eq[k], &q); err != nil {
				return "skip: race: " + err.Error()
			}
			want, err := q.race()
			if err != nil {
				return "skip: race: " + err.Error()
			}
			if got := g.Players[id].Race; got != want {
				errs = append(errs, pvMismatch("race", fmt.Sprintf("%+v", got), fmt.Sprintf("%+v", want)))
			}
			if pvRaceSettingsOf == nil || g.Races == nil {
				return "skip: race settings without a race check"
			}
			st, ok := pvRaceSettingsOf(g.Races, id)
			if !ok {
				return "skip: race settings without a race check"
			}
			spend := pvLeftoverSpend[q.LeftoverSpend]
			if st.Spend != spend || st.Stat15 != q.Stat15 || st.ExpensiveAt3 != q.TechsStartHigh {
				errs = append(errs, pvMismatch("race settings", fmt.Sprintf("spend %d, stat 15 %d, techs start high %v", st.Spend, st.Stat15, st.ExpensiveAt3),
					fmt.Sprintf("spend %d, stat 15 %d, techs start high %v", spend, q.Stat15, q.TechsStartHigh)))
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
// runVector generates v's years with the seed variant k (0 is the
// harness's reference seeding) and checks its cases.
func runVector(v *pvVector, k int) []pvResult {
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
	if l.global != "" {
		for _, c := range v.Cases {
			res(c, "skip", l.global)
		}
		return out
	}
	// Generate year by year, keeping each year's game.
	games := []Game{l.g}
	l.views = [][]PlayerView{nil}
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
		r, err := GenerateTurn(withRules(g), files, rand.New(rand.NewSource(int64(y)+int64(k)<<32)))
		if err != nil {
			genErr = err
			break
		}
		g = r.Game
		games = append(games, g)
		l.views = append(l.views, r.Views)
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
		var fails, skips []string
		// checked counts the compared expectations; exact, those of them
		// that are not samples.
		checked, exact := 0, 0
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
			case msg != "" && e.Sample:
				skips = append(skips, "sample: one stream's random outcome")
			case msg != "":
				checked++
				exact++
				fails = append(fails, fmt.Sprintf("y%d %s: %s", e.Year, e.Kind, msg))
			default:
				checked++
				if !e.Sample {
					exact++
				}
			}
		}
		switch {
		case len(fails) > 0 && pvLegacyOff[c.ID] != "":
			res(c, "differs", pvLegacyOff[c.ID]+": "+strings.Join(fails, " | "))
		case len(fails) > 0:
			res(c, "fail", strings.Join(fails, " | "))
		case checked == 0:
			res(c, "skip", strings.Join(pvUnique(skips), ", "))
		case exact == 0:
			// Only samples matched: evidence of the rule, never of an
			// exact value.
			res(c, "sample-only", "every compared expectation is a sample")
		default:
			res(c, "pass", "")
		}
		if checked > 0 {
			out[len(out)-1].skipped = skips
		}
	}
	return out
}

// pvPlanetLevels is the vectors' planet report level for each of
// Elegy's (SCANNING.md "What a planet report contains"; PARITY.md
// "Co-location and orbit reports": 1 position, 3 normal, 4 detailed).
var pvPlanetLevels = map[ReportLevel]int{ReportNone: 0, ReportPosition: 1, ReportNormal: 3, ReportDetailed: 4}

// viewCheck checks a view expectation: what player viewer knows of a
// planet, a fleet or a space object at the end of the year (vectors
// README.md "view").
func (l *pvLoaded) viewCheck(g *Game, e pvExpect, eq map[string]json.RawMessage, fleet func(owner, id int) *Fleet) string {
	if e.Viewer == nil || e.Year >= len(l.views) || *e.Viewer >= len(l.views[e.Year]) {
		return "skip: view without its viewer"
	}
	v := l.views[e.Year][*e.Viewer]
	var sub struct {
		Kind  string `json:"kind"`
		Owner int    `json:"owner"`
		ID    int    `json:"id"`
	}
	json.Unmarshal(e.Subject, &sub)
	got := map[string]any{}
	switch sub.Kind {
	case "planet":
		id := l.planet[sub.ID]
		level, sb := ReportNone, false
		for _, r := range v.Planets {
			if r.Planet == id {
				level, sb = r.Level, r.Starbase
			}
		}
		if level == ReportOwn {
			return "skip: view of an own planet"
		}
		got["level"], got["starbase_visible"] = pvPlanetLevels[level], sb
	case "fleet":
		if sub.Owner == *e.Viewer {
			return "skip: view of an own fleet"
		}
		f := fleet(sub.Owner, sub.ID)
		level := 0
		if f != nil {
			for _, s := range v.Fleets {
				if s.Fleet == f.ID {
					// Seen (3), or seen with its cargo (4: PARITY.md
					// "Co-location and orbit reports").
					level = 3
					if s.Cargo != nil {
						level = 4
					}
				}
			}
		}
		got["level"], got["known"] = level, level > 0
		if level != 4 {
			// Cargo not shown reads as all zero (SC027).
			got["cargo_shown"] = []int{0, 0, 0, 0}
		}
	case "minefield", "packet":
		// "known" is whether the object is in the viewer's file this
		// year: seen (SCANNING.md "Space objects").
		list := v.Objects.Minefields
		if sub.Kind == "packet" {
			list = v.Objects.Packets
		}
		got["known"] = slices.Contains(list, [2]int{sub.Owner, sub.ID})
	case "wormhole":
		id, ok := l.endID[sub.ID]
		got["known"] = ok && slices.Contains(v.Objects.Wormholes, id)
	case "trader":
		got["known"] = slices.Contains(v.Objects.Traders, sub.ID)
	default:
		return "skip: view of a " + sub.Kind
	}
	for k, w := range eq {
		gv, ok := got[k]
		if !ok {
			return "skip: view field " + k
		}
		var want any
		json.Unmarshal(w, &want)
		gb, _ := json.Marshal(gv)
		var gn any
		json.Unmarshal(gb, &gn)
		if fmt.Sprint(gn) != fmt.Sprint(want) {
			return pvMismatch(sub.Kind+" "+k, gv, want)
		}
	}
	return ""
}

const pvBaseline = "testdata/vectors/baseline.txt"

// pvLegacyOff names the LEGACY BUG cases whose switch is off by default,
// where Elegy's chosen rule intentionally differs from the original.
var pvLegacyOff = map[string]string{
	"FO-03-E": "Legacy.MergeOverflow off",
	"FO-06-G": "Legacy.MergeOverflow off",
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

// pvSeeds is how many seed variants each vector runs with. The harness's
// random stream is not the original's (the oracle's seeds are not in the
// vectors), so a case whose outcome depends on the draws matches only by
// chance. Such a case passes with some seeds and fails with others; it is
// reported as "random" with its pass count and the reference seed's
// comparison kept. The tally counts random cases whose reference seed
// fails apart, and the baseline lists each random case with its pass
// count.
const pvSeeds = 8

// pvStreamCheck folds one vector's seed variants into one result per case.
func pvStreamCheck(runs [][]pvResult) []pvResult {
	out := runs[0]
	for i, r := range out {
		passes, samples, same, onlySamples := 0, 0, true, true
		for _, run := range runs {
			if run[i].id != r.id {
				panic("parity: seed variants disagree on case order")
			}
			switch run[i].status {
			case "pass":
				passes++
			case "sample-only":
				samples++
			}
			same = same && run[i].status == r.status
			onlySamples = onlySamples && (run[i].status == "sample-only" || run[i].status == "skip")
		}
		switch {
		case !same && onlySamples:
			// Each seed either matched only samples or had every sample
			// miss: sample-only on the seeds that matched.
			out[i].status = "sample-only"
			out[i].passes = samples
			out[i].why = fmt.Sprintf("sample-only with %d of %d seeds; every sample missed on the others", samples, len(runs))
		case !same:
			why := r.why
			if why == "" {
				why = "matches"
			}
			out[i].status = "random"
			out[i].passes = passes
			out[i].refStatus = r.status
			out[i].why = fmt.Sprintf("passes with %d of %d seeds; reference seed %s: %s", passes, len(runs), r.status, why)
		case r.status == "sample-only":
			out[i].passes = len(runs)
		}
	}
	return out
}

func TestParityVectors(t *testing.T) {
	var all []pvResult
	// Expectations skipped inside checked cases, by reason, from the
	// reference seed.
	partial := map[string]int{}
	for _, v := range loadVectors(t) {
		runs := make([][]pvResult, pvSeeds)
		for k := range runs {
			runs[k] = runVector(v, k)
		}
		for _, r := range runs[0] {
			for _, why := range r.skipped {
				if strings.HasPrefix(why, "stream ") {
					why = "stream (another oracle random stream)"
				}
				partial[why]++
			}
		}
		all = append(all, pvStreamCheck(runs)...)
	}
	type tally struct{ pass, fail, skip, measured, differs, random, randomFail, sampleOnly int }
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
		case r.status == "sample-only":
			tl.sampleOnly++
		case r.status == "random" && r.refStatus == "fail":
			tl.randomFail++
		case r.status == "random":
			tl.random++
		case r.status == "pass" && r.tag == "MEASURED":
			tl.measured++
		case r.status == "pass":
			tl.pass++
		default:
			tl.fail++
		}
	}
	for _, c := range corpora {
		tl := per[c]
		t.Logf("%-5s pass %3d  measured pass %3d  sample-only %3d  fail %3d  skip %3d  differs %3d  random %3d  random, reference seed fails %3d", c, tl.pass, tl.measured, tl.sampleOnly, tl.fail, tl.skip, tl.differs, tl.random, tl.randomFail)
	}
	for _, r := range all {
		if r.status != "pass" && r.status != "sample-only" {
			t.Logf("%s %-10s %-4s %s", r.status, r.tag, r.id, r.why)
		}
	}
	for _, why := range pvKeys(partial) {
		t.Logf("expectations skipped in checked cases: %4d %s", partial[why], why)
	}

	if os.Getenv("PARITY_BASELINE") == "write" {
		f, err := os.Create(pvBaseline)
		if err != nil {
			t.Fatal(err)
		}
		w := bufio.NewWriter(f)
		fmt.Fprintln(w, "# Parity cases Elegy passes (TestParityVectors). One case id per line;")
		fmt.Fprintf(w, "# \"random k\" after an id: the case passes with k of the %d seed variants.\n", pvSeeds)
		fmt.Fprintln(w, "# \"sample-only\" after an id: every expectation the case compares is a sample")
		fmt.Fprintln(w, "# that matched, so it is evidence of the rule, not of an exact value; \"sample-only k\":")
		fmt.Fprintln(w, "# so on k seeds, with every sample missing on the others.")
		for _, r := range all {
			switch r.status {
			case "pass":
				fmt.Fprintln(w, r.id)
			case "sample-only":
				if r.passes < pvSeeds {
					fmt.Fprintf(w, "%s sample-only %d\n", r.id, r.passes)
				} else {
					fmt.Fprintln(w, r.id, "sample-only")
				}
			case "random":
				fmt.Fprintf(w, "%s random %d\n", r.id, r.passes)
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
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		r := status[fields[0]]
		if len(fields) >= 2 && fields[1] == "sample-only" {
			want := pvSeeds
			if len(fields) == 3 {
				want, _ = strconv.Atoi(fields[2])
			}
			if r.status != "pass" && (r.status != "sample-only" || r.passes < want) {
				t.Errorf("baseline case %s was sample-only with %d of %d seeds, now %s: %s", fields[0], want, pvSeeds, r.status, r.why)
			}
			continue
		}
		want := pvSeeds
		if len(fields) == 3 && fields[1] == "random" {
			want, _ = strconv.Atoi(fields[2])
		}
		got := r.passes
		if r.status == "pass" {
			got = pvSeeds
		}
		if got < want {
			t.Errorf("baseline case %s passes with %d of %d seeds, was %d: %s %s", fields[0], got, pvSeeds, want, r.status, r.why)
		}
	}
}
