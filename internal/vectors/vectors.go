package vectors

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/races"
	"github.com/bfaber-centaur/elegy/terraform"
)

// A case is skipped, not failed, when it needs something Elegy does not
// model yet (a task, target, object or expectation kind), or when its
// outcome is random. CONFIRMED and LEGACY BUG cases are exact-match
// targets; a MEASURED pass is a pass the caller may tally apart, and
// every failure, MEASURED or not, is a failure.

// Vector is one parity vector (FORMAT.md): a start, the orders and the
// cases to compare.
type Vector struct {
	ID           string   `json:"id"`
	Years        int      `json:"years"`
	Random       string   `json:"random"`
	InitialState pvState  `json:"initial_state"`
	Cases        []pvCase `json:"cases"`
	// NewGame is a new-game vector's settings (FORMAT.md "new_game"),
	// which newgame's tests generate; such a vector has no starting state.
	NewGame json.RawMessage `json:"new_game"`
	// Orders are the orders players submitted (FORMAT.md "orders").
	Orders []pvOrderBlock `json:"orders"`
	// RandomStart is where each stream's years started (streams.go;
	// a proposed field, not yet in FORMAT.md).
	RandomStart []pvRandomStart `json:"random_start"`
	path        string          // corpus/run
	corpus      string
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
	Bounds       []int  `json:"bounds"`
	RandomEvents bool   `json:"random_events"`
	SlowerTech   bool   `json:"slower_tech"`
	PublicScores bool   `json:"public_scores"`
	// Victory is the game record's victory conditions in game units, with
	// enabled for the seven that can be switched; absent when the
	// vector's builder did not read them.
	Victory map[string]struct {
		Value   int   `json:"value"`
		Enabled *bool `json:"enabled"`
	} `json:"victory_conditions"`
}

// size is the game's universe size. A missing or unknown size is an
// error, not tiny: the waypoint check clamps fleet positions into the
// galaxy box, 1000..1000+(size+1)·400 on each axis (ORDERS.md "Outside
// the galaxy"), so a wrong size moves fleets. Bounds, when the vector has
// them, must be that box.
func (gm pvGame) size() (int, error) {
	n, ok := map[string]int{"tiny": 0, "small": 1, "medium": 2, "large": 3, "huge": 4}[gm.Size]
	if !ok {
		return 0, fmt.Errorf("game: universe size %q is not tiny, small, medium, large or huge", gm.Size)
	}
	if gm.Bounds != nil {
		hi := 1000 + (n+1)*400
		if !slices.Equal(gm.Bounds, []int{1000, 1000, hi, hi}) {
			return 0, fmt.Errorf("game: bounds %v, want [1000 1000 %d %d] for a %s universe", gm.Bounds, hi, hi, gm.Size)
		}
	}
	return n, nil
}

// victory converts the vector's victory conditions to the new-game
// dialog's encoding (engine.Victory; KERNEL.md "Victory conditions").
func (pg pvGame) victory() (engine.Victory, error) {
	var v engine.Victory
	for _, c := range []struct {
		name     string
		to       *int
		div, off int // value = (v + off) · div
		cond     int // the condition it switches, or -1
	}{
		{"planets_owned_percent", &v.Planets, 5, 4, engine.VictoryPlanets},
		{"tech_level", &v.TechLevel, 1, 8, engine.VictoryTech},
		{"tech_fields", &v.TechFields, 1, 2, -1},
		{"score", &v.Score, 1000, 1, engine.VictoryScore},
		{"lead_over_second_percent", &v.Lead, 10, 2, engine.VictoryLead},
		{"resources_thousands", &v.Resources, 10, 1, engine.VictoryResources},
		{"capital_ships", &v.Capital, 10, 1, engine.VictoryCapital},
		{"highest_score_after_years", &v.Highest, 10, 3, engine.VictoryHighest},
		{"conditions_needed", &v.Needed, 1, 0, -1},
		{"minimum_years", &v.MinYears, 10, 3, -1},
	} {
		x, ok := pg.Victory[c.name]
		if !ok || x.Value%c.div != 0 || x.Value/c.div < c.off {
			return v, fmt.Errorf("victory condition %s %v", c.name, x)
		}
		*c.to = x.Value/c.div - c.off
		if c.cond >= 0 {
			if x.Enabled == nil {
				return v, fmt.Errorf("victory condition %s without enabled", c.name)
			}
			v.Enabled[c.cond] = *x.Enabled
		}
	}
	return v, nil
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
	Homeworld           *int              `json:"homeworld"`
	// DefaultQueue and DefaultLeftoverToResearch are the player's
	// default queue settings, which a planet it newly owns takes; both
	// absent when the vector's builder did not read them (vectors/README.md
	// "players").
	DefaultQueue              *[]json.RawMessage `json:"default_queue"`
	DefaultLeftoverToResearch *bool              `json:"default_leftover_to_research"`
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
	ID             int   `json:"id"`
	X              int   `json:"x"`
	Y              int   `json:"y"`
	Owner          int   `json:"owner"`
	Homeworld      bool  `json:"homeworld"` // the planet record's homeworld flag
	Artifact       bool  `json:"artifact"`  // an ancient artifact (start states only)
	Concentrations []int `json:"concentrations"`
	// each concentration's depletion fraction as the planet record
	// stores it (vectors README "planets"; absent: all 0)
	ConcFractions       []int `json:"concentration_fractions"`
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
	PacketDestination   *int  `json:"packet_destination"`
	PacketWarp          *int  `json:"packet_warp"`
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
	AttackWho                               int    `json:"attack_who"`
	DumpCargo                               bool   `json:"dump_cargo"`
	Name                                    string `json:"name"`
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
	// SeenBy names the player whose file held a score_record.
	SeenBy *int `json:"seen_by"`
	// Screen and Field name a client_estimate expectation's reading.
	Screen string `json:"screen"`
	Field  string `json:"field"`
	// Player, MessageID and Present are a message expectation's
	// (README.md "message").
	Player    *int  `json:"player"`
	MessageID *int  `json:"message_id"`
	Present   *bool `json:"present"`
	// before lists the present message expectations of the same player
	// and year listed before this one in its case (pvMessageOrder).
	before []pvMessagePrior
	// Messages is a message_sequence expectation's list (README.md
	// "Message sequences"), with Player.
	Messages []pvSeqEntry `json:"messages"`
}

// pvMaps maps vector ids to Elegy's: EndID a wormhole end id to its
// waypoint target ID, Planet a planet id to the Elegy planet ID.
type pvMaps struct {
	EndID, Planet map[int]int
}

// pvObjExpect is an expectation as the space-object checker sees it.
type pvObjExpect struct {
	Kind                       string
	Owner, ID                  *int
	Equals, Tolerance, Subject json.RawMessage
}

// pvRaceSettings is what the race check needs of a vector player beyond
// the engine's Race (vectors README.md "race").
type pvRaceSettings struct {
	ExpensiveAt3  bool
	Spend, Stat15 int
	Computer      bool
}

// pvRaces builds the game's race checker (KERNEL.md "Turn order" step
// 2a): each vector player's race with its wizard settings.
func pvRaces(g *engine.Game, settings []pvRaceSettings) engine.RaceChecker {
	r := &races.GameRaces{}
	for i, s := range settings {
		r.Designs = append(r.Designs, races.Design{Race: g.Players[i].Race, ExpensiveAt3: s.ExpensiveAt3, Spend: s.Spend, Stat15: s.Stat15})
		r.Computer = append(r.Computer, s.Computer)
	}
	return r
}

// pvRaceSettingsOf reads a player's wizard settings back from the game's
// race checker, after the year's check.
func pvRaceSettingsOf(c engine.RaceChecker, player int) (pvRaceSettings, bool) {
	r, ok := c.(*races.GameRaces)
	if !ok || player >= len(r.Designs) {
		return pvRaceSettings{}, false
	}
	d := r.Designs[player]
	return pvRaceSettings{ExpensiveAt3: d.ExpensiveAt3, Spend: d.Spend, Stat15: d.Stat15, Computer: r.Computer[player]}, true
}

// pvLeftoverSpend maps a race's leftover_spend to its wizard number.
func pvLeftoverSpend() map[string]int {
	return map[string]int{"surface_minerals": 0, "mineral_concentrations": 1, "mines": 2, "factories": 3, "defenses": 4}
}

func (p *pvBattlePlan) UnmarshalJSON(b []byte) error {
	var q struct {
		Owner     int    `json:"owner"`
		Slot      int    `json:"slot"`
		Tactic    int    `json:"tactic"`
		Primary   int    `json:"primary"`
		Secondary int    `json:"secondary"`
		AttackWho int    `json:"attack_who"`
		DumpCargo bool   `json:"dump_cargo"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal(b, &q); err != nil {
		return err
	}
	*p = pvBattlePlan{q.Owner, q.Slot, q.Tactic, q.Primary, q.Secondary, q.AttackWho, q.DumpCargo, q.Name}
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
func (q pvRace) race() (engine.Race, error) {
	var r engine.Race
	// An out-of-range PRT is given as its stored number (FORMAT.md).
	var name string
	if json.Unmarshal(q.PRT, &name) != nil {
		var n int
		if err := json.Unmarshal(q.PRT, &n); err != nil {
			return r, fmt.Errorf("PRT %s", q.PRT)
		}
		r.PRT = engine.PRT(pvOutOfRangePRT + n)
	} else {
		prt, ok := pvPRTs()[name]
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
			r.Env[axis] = engine.EnvRange{Immune: true}
			continue
		}
		r.Env[axis] = engine.EnvRange{Center: pvByte(h[0]), Low: pvByte(h[1]), High: pvByte(h[2])}
	}
	r.ColonistsPerResource = q.ColonistsPerResource
	r.FactoryOutput, r.FactoryCost, r.FactoriesOperated = q.Factory.Output, q.Factory.Cost, q.Factory.Per10k
	r.MineOutput, r.MineCost, r.MinesOperated = q.Mine.Output, q.Mine.Cost, q.Mine.Per10k
	r.FactoryLessGermanium = q.FactoriesCostLess
	for name, c := range q.ResearchCost {
		r.ResearchCosts[pvFields()[name]] = map[string]engine.ResearchCost{"normal": engine.ResearchNormal, "expensive": engine.ResearchExpensive, "cheap": engine.ResearchCheap}[c]
	}
	return r, nil
}

func pvFields() map[string]int {
	return map[string]int{"energy": engine.Energy, "weapons": engine.Weapons, "propulsion": engine.Propulsion,
		"construction": engine.Construction, "electronics": engine.Electronics, "biotechnology": engine.Biotech}
}

func pvPRTs() map[string]engine.PRT {
	return map[string]engine.PRT{"HE": engine.PRTHyperExpansion, "JOAT": engine.PRTJackOfAllTrades, "AR": engine.PRTAlternateReality,
		"IS": engine.PRTInnerStrength, "WM": engine.PRTWarMonger, "IT": engine.PRTInterstellarTraveler, "CA": engine.PRTClaimAdjuster,
		"SS": engine.PRTSuperStealth, "PP": engine.PRTPacketPhysics, "SD": engine.PRTSpaceDemolition}
}

func pvCargo() map[string]int {
	return map[string]int{"ironium": engine.Ironium, "boranium": engine.Boranium, "germanium": engine.Germanium, "colonists": engine.CargoColonists}
}

// pvLoaded is a vector's start as an Elegy game, with the id maps the
// checks need.
type pvLoaded struct {
	g       engine.Game
	fleetID map[[2]int]int // (owner, vector id) → Elegy fleet id
	design  map[[2]int]int // (owner, design slot) → Game.Designs index
	planet  map[int]int    // vector planet id → Elegy planet id
	// unsupported names, per fleet (owner, id), what Elegy does not model
	// in its orders; a case about such a fleet is skipped.
	unsupported map[[2]int]string
	start       map[[2]int]engine.Point // each fleet's starting position
	// global is a reason every expectation of the vector is skipped, if
	// any.
	global string
	// queued marks the players with a production queue the harness does
	// not load.
	queued map[int]bool
	// defaults marks the players whose default queue settings the
	// vector carries.
	defaults map[int]bool
	// ordered marks the players the vector gives orders for.
	ordered map[int]bool
	// endID maps a vector wormhole end id to its waypoint target ID
	// (pvWormholeEnds).
	endID map[int]int
	// views is each generated year's player views, by year.
	views [][]engine.PlayerView
	// events is each generated year's events, by year.
	events [][]engine.Event
	// rejected lists the vector's orders Elegy refused, by year.
	rejected []string
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

func loadVector(v *Vector) (*pvLoaded, error) {
	s := v.InitialState
	cat := engine.Components()
	l := &pvLoaded{fleetID: map[[2]int]int{}, design: map[[2]int]int{}, planet: map[int]int{}, unsupported: map[[2]int]string{}, start: map[[2]int]engine.Point{}, queued: map[int]bool{}, defaults: map[int]bool{}, ordered: map[int]bool{}}
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
	g.PublicScores = s.Game.PublicScores
	if s.Game.Victory != nil {
		if g.Victory, err = s.Game.victory(); err != nil {
			return nil, err
		}
	}
	if v.NewGame == nil || len(s.Players) > 0 {
		if g.Size, err = s.Game.size(); err != nil {
			return nil, err
		}
	}
	g.PlanetScanners = cat.PlanetScanners()
	g.Defenses = cat.Defenses()

	g.Players = make([]engine.Player, len(s.Players))
	for _, p := range s.Players {
		pl := &g.Players[p.ID]
		pl.Computer = p.Computer
		// The planet the player's own record names as its homeworld,
		// re-marked each year (vectors/README.md "homeworld"; KERNEL.md
		// "Turn order" step 3.6); absent, the record names none.
		if p.Homeworld != nil {
			pl.HasHomeworld, pl.Homeworld = true, *p.Homeworld+1
		}
		race, err := p.Race.race()
		if err != nil {
			return nil, err
		}
		pl.Race = race
		if (p.DefaultQueue == nil) != (p.DefaultLeftoverToResearch == nil) {
			return nil, fmt.Errorf("player %d: default_queue and default_leftover_to_research go together", p.ID)
		}
		// A default queue holding an item Elegy does not build is left
		// unloaded, and what a newly owned planet takes stays skipped.
		if p.DefaultQueue != nil {
			if items, why := pvDefaultQueue(*p.DefaultQueue); why == "" {
				pl.DefaultQueue, pl.DefaultLeftoverOnly = items, *p.DefaultLeftoverToResearch
				l.defaults[p.ID] = true
			}
		}
		for name, lv := range p.Tech {
			pl.Research.Levels[pvFields()[name]] = lv
		}
		for name, a := range p.ResearchAccumulated {
			pl.Research.Accumulated[pvFields()[name]] = a
		}
		pl.Research.Current = pvFields()[p.ResearchField]
		pl.Research.Next = engine.NextSameField
		if len(p.ResearchNextField) > 0 {
			var next string
			if json.Unmarshal(p.ResearchNextField, &next) != nil {
				return nil, fmt.Errorf("research_next_field %s", p.ResearchNextField)
			}
			switch next {
			case "same":
			case "lowest":
				pl.Research.Next = engine.NextLowestField
			default:
				f, ok := pvFields()[next]
				if !ok {
					return nil, fmt.Errorf("research_next_field %q", next)
				}
				pl.Research.Next = f
			}
		}
		pl.ResearchBudget = p.ResearchPercent
		pl.Relations = make([]engine.Relation, len(s.Players))
		for i := range pl.Relations {
			pl.Relations[i] = engine.RelationNeutral
		}
		for other, rel := range p.Relations {
			var o int
			fmt.Sscan(other, &o)
			pl.Relations[o] = map[string]engine.Relation{"friend": engine.RelationFriend, "neutral": engine.RelationNeutral, "enemy": engine.RelationEnemy}[rel]
		}
	}
	if l.global == "" {
		settings := make([]pvRaceSettings, len(s.Players))
		for _, p := range s.Players {
			spend, ok := pvLeftoverSpend()[p.Race.LeftoverSpend]
			if !ok {
				return nil, fmt.Errorf("leftover_spend %q", p.Race.LeftoverSpend)
			}
			settings[p.ID] = pvRaceSettings{ExpensiveAt3: p.Race.TechsStartHigh, Spend: spend, Stat15: p.Race.Stat15, Computer: p.Computer}
		}
		g.Races = pvRaces(g, settings)
	}
	// The game's remote mining and Orbital Adjusters (KERNEL.md "Turn
	// order" steps 6c.2 and 7.4).
	g.Terraform = terraform.Rules{}

	addDesign := func(d pvDesign, starbase bool) error {
		var fills []engine.SlotFill
		engineSlot, engines := -1, 0
		for i, sl := range d.Slots {
			if sl == nil {
				continue
			}
			fills = append(fills, engine.SlotFill{Slot: i, Part: sl.Part, Count: sl.Count})
		}
		name := fmt.Sprintf("p%d d%d", d.Owner, d.Slot)
		ds, err := cat.NewDesign(name, d.Hull, fills)
		if err != nil && strings.Contains(err.Error(), "does not take") {
			// A design written directly into the oracle's state, with a
			// part in a slot the client would not allow, keeps it for the
			// year's cost (Catalog.NewDesignAnyKind; COMBAT.md "Parts the
			// game drops from a design").
			ds, err = cat.NewDesignAnyKind(name, d.Hull, fills)
		}
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
				if ds.Slots[i].Part.Kind == engine.PartEngine {
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
		g.DesignSlots = append(g.DesignSlots, engine.DesignSlot{Owner: d.Owner, Starbase: starbase, Slot: d.Slot, Design: len(g.Designs)})
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
			pl.Plans = append(pl.Plans, engine.BattlePlan{})
		}
		pl.Plans[bp.Slot] = pvPlan(bp.Name, bp.Tactic, bp.Primary, bp.Secondary, bp.AttackWho, bp.DumpCargo)
	}

	for _, p := range s.Planets {
		l.planet[p.ID] = p.ID + 1
		pl := engine.Planet{ID: p.ID + 1, Pos: engine.Point{X: p.X, Y: p.Y}, Owner: p.Owner, Homeworld: p.Homeworld, Artifact: p.Artifact, StarbaseDesign: -1}
		if p.Owner < 0 {
			pl.Owner = engine.NoOwner
		}
		for m, c := range p.Concentrations {
			pl.Deposits[m].Concentration = c
		}
		// The stored byte is 1/256ths of the current point remaining, 0
		// meaning a full 256 (KERNEL.md "Mining"), as Deposit.Fraction.
		for m, f := range p.ConcFractions {
			pl.Deposits[m].Fraction = f
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
		// The mass driver's settings (vectors/README.md: absent means no
		// destination and warp 4, a setting below 5 that counts as unset;
		// OBJECTS.md "Launch", OB-028-C, D).
		if p.PacketDestination != nil {
			pl.HasPacketDest, pl.PacketDest = true, *p.PacketDestination+1
		}
		set(&pl.PacketSpeed, p.PacketWarp)
		if pl.Owner != engine.NoOwner {
			// A planet the state lists no queue for has none: its
			// resources all go to research (KERNEL.md "Production").
			for _, q := range s.ProductionQueues {
				pl.HasQueue = pl.HasQueue || q.Planet == p.ID
			}
		}
		// The states write no starbase as design null with damage -1.
		// Any other record without a design is an error in the state,
		// not a planet without a starbase; so is a design the owner
		// (none at a planet nobody owns) does not have.
		sb := p.Starbase
		if sb != nil && sb.Design == nil && sb.Damage != -1 {
			return nil, fmt.Errorf("planet %d: starbase without a design", p.ID)
		}
		if sb != nil && sb.Design != nil {
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
	{
		// Every game holds space objects, so the Trader's appearance
		// draws run with random events on.
		obj, err := pvObjects{}.Load(g, s.Objects, pvMaps{l.endID, l.planet})
		if err != nil {
			l.global = "space objects: " + err.Error()
		}
		g.Objects = obj
		// Each player's Trader parts, the vector's player
		// mystery_trader_items (stars-elegy tools/vectors/build.py, the
		// part word by bit since #127), which a tech attempt never gives
		// again (COMBAT.md "Tech from battle", step 3).
		if obj != nil {
			for _, p := range s.Players {
				for _, it := range p.MysteryTraderItems {
					if b := obj.TraderBit(it); b >= 0 && p.ID >= 0 && p.ID < len(g.Players) {
						obj.GiveTraderBit(p.ID, b)
					}
				}
			}
		}
	}
	// Production queues (FORMAT.md "Queue items"). A queue holding an
	// item Elegy does not build (item 10, the Genesis Device, scanners)
	// is not loaded: checks its planet, its owner or
	// ships it may build could see are skipped.
	for _, q := range s.ProductionQueues {
		pi := -1
		for i := range g.Planets {
			if g.Planets[i].ID == l.planet[q.Planet] {
				pi = i
			}
		}
		if pi < 0 || g.Planets[pi].Owner == engine.NoOwner {
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
		l.start[pvFleetKey(f.Owner, f.ID)] = engine.Point{X: f.X, Y: f.Y}
	}
	for _, f := range s.Fleets {
		key := pvFleetKey(f.Owner, f.ID)
		ef := engine.Fleet{ID: l.fleetID[key], Number: f.ID + 1, Owner: f.Owner, Pos: engine.Point{X: f.X, Y: f.Y}, Fuel: f.Fuel, Plan: f.BattlePlan, Repeat: f.Repeat}
		for _, sh := range f.Ships {
			st := engine.Stack{Design: l.design[[2]int{f.Owner, sh.Design}], Count: sh.Count}
			if sh.Damage != nil {
				st.Damage = engine.Damage{Pct: sh.Damage.PercentOfShips, Units: sh.Damage.Units}
			}
			ef.Stacks = append(ef.Stacks, st)
		}
		for name, v := range f.Cargo {
			if c := pvCargo()[name]; c == engine.CargoColonists {
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
				// Waypoint 0 naming a packet or salvage (Fleet.Target0).
				if wp, why := l.waypoint(f.Owner, w); why == "" && wp.Target == engine.TargetPacketObject {
					ef.Target0 = &engine.ObjectRef{Owner: wp.Owner, Number: wp.ID}
				}
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
// items (vectors/README.md "Queue items"). The packet items 6 and 14–17
// launch with the planet's packet_destination and packet_warp (stars-elegy
// 4247afc); which mineral each of 14–17 launches is BINARY-ONLY there.
func pvItemKinds() map[int]engine.ItemKind {
	return map[int]engine.ItemKind{
		0: engine.ItemAutoMines, 1: engine.ItemAutoFactories, 2: engine.ItemAutoDefenses, 3: engine.ItemAutoAlchemy,
		4: engine.ItemAutoMinTerraform, 5: engine.ItemAutoMaxTerraform, 6: engine.ItemAutoPackets,
		7: engine.ItemFactory, 8: engine.ItemMine, 9: engine.ItemDefenses, 11: engine.ItemMineralAlchemy, 12: engine.ItemTerraform,
		14: engine.ItemIroniumPacket, 15: engine.ItemBoraniumPacket, 16: engine.ItemGermaniumPacket, 17: engine.ItemMixedPacket,
	}
}

// queueEquals checks a planet's production queue (FORMAT.md
// "production_queue").
func (l *pvLoaded) queueEquals(g *engine.Game, e pvExpect) string {
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
				if p0.ID == g.Planets[i].ID && p0.Owner != g.Planets[i].Owner && g.Planets[i].Owner != engine.NoOwner && !l.defaults[g.Planets[i].Owner] {
					// A new owner's planet takes that player's default
					// queue, which this vector does not carry. A planet
					// left with no owner takes no default.
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
func pvQueue(raw []json.RawMessage) ([]engine.QueueItem, string) {
	var items []engine.QueueItem
	for _, r := range raw {
		var it struct{ ID, Count, Percent, Kind int }
		if err := json.Unmarshal(r, &it); err != nil {
			return nil, "production queue item " + string(r)
		}
		if it.Kind == 2 {
			// A design item: id 0–15 is a ship design slot, 16–25 starbase
			// slot id − 16 (FORMAT.md "Queue items", CONFIRMED).
			k, slot := engine.ItemShip, it.ID
			if it.ID >= 16 {
				k, slot = engine.ItemStarbase, it.ID-16
			}
			items = append(items, engine.QueueItem{Kind: k, Count: it.Count, Percent: it.Percent, Slot: slot})
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
		k, ok := pvItemKinds()[it.ID]
		if !ok {
			return nil, fmt.Sprintf("production queue: planetary item %d", it.ID)
		}
		items = append(items, engine.QueueItem{Kind: k, Count: it.Count, Percent: it.Percent})
	}
	return items, ""
}

// pvDefaultQueue converts a player's default queue, which holds
// planetary items only (vectors/README.md "players"), or names what it
// cannot load.
func pvDefaultQueue(raw []json.RawMessage) ([]engine.QueueItem, string) {
	items, why := pvQueue(raw)
	if why != "" {
		return nil, "default " + why
	}
	for _, it := range items {
		if it.Kind == engine.ItemShip || it.Kind == engine.ItemStarbase {
			return nil, "default queue with a design item"
		}
	}
	return items, ""
}

// waypoint converts a waypoint's position, warp and target for a fleet of
// owner, or names what Elegy cannot model. Warp 11 is a stargate jump
// (movement runs it); a higher warp is not modelled.
func (l *pvLoaded) waypoint(owner int, w pvWaypoint) (engine.Waypoint, string) {
	wp := engine.Waypoint{Pos: engine.Point{X: w.X, Y: w.Y}, Warp: w.Warp}
	why := ""
	if w.Warp > engine.StargateWarp {
		why, wp.Warp = "warp "+strconv.Itoa(w.Warp), 0
	}
	switch w.Target.Kind {
	case "space":
	case "planet":
		wp.Target, wp.ID = engine.TargetPlanet, l.planet[*w.Target.ID]
	case "fleet":
		if w.Target.Owner != nil {
			owner = *w.Target.Owner
		}
		wp.Target, wp.ID = engine.TargetFleet, pvElegyFleetID(owner, *w.Target.ID)
	case "wormhole":
		id, ok := l.endID[*w.Target.ID]
		if !ok {
			why = "wormhole end " + strconv.Itoa(*w.Target.ID)
		}
		wp.Target, wp.ID = engine.TargetWormhole, id
	case "trader":
		wp.Target, wp.ID = engine.TargetTrader, *w.Target.ID
	case "packet_object":
		// A packet or salvage named by its owner and number as in objects
		// (vectors/README.md "fleets").
		if w.Target.Owner == nil || w.Target.ID == nil {
			why = "packet_object target without an owner and id"
			break
		}
		wp.Target, wp.Owner, wp.ID = engine.TargetPacketObject, *w.Target.Owner, *w.Target.ID
	default:
		why = "target " + w.Target.Kind
	}
	return wp, why
}

// task converts a waypoint task, or names what Elegy cannot model.
func (l *pvLoaded) task(w pvWaypoint) (engine.Task, string) {
	switch w.Task.Kind {
	case "none":
		return engine.Task{}, ""
	case "colonize":
		return engine.Task{Kind: engine.TaskColonize}, ""
	case "route":
		return engine.Task{Kind: engine.TaskRoute}, ""
	case "remote_mine":
		return engine.Task{Kind: engine.TaskRemoteMine}, ""
	case "scrap":
		return engine.Task{Kind: engine.TaskScrap}, ""
	case "transfer":
		return engine.Task{Kind: engine.TaskTransferFleet, Player: w.Task.ToPlayer}, ""
	case "patrol":
		return engine.Task{Kind: engine.TaskPatrol, Range: w.Task.Range}, ""
	case "lay_mines":
		// The years word w lays w + 1 years (PARITY.md "Lay-mines
		// duration", CONFIRMED OB-002-N, OB-019, OB-025).
		var word int
		if json.Unmarshal(w.Task.Years, &word) != nil {
			return engine.Task{Kind: engine.TaskLayMines, Years: engine.YearsIndefinitely}, ""
		}
		return engine.Task{Kind: engine.TaskLayMines, Years: word + 1}, ""
	case "merge":
		owner := 0
		if w.Target.Owner != nil {
			owner = *w.Target.Owner
		}
		if w.Target.Kind != "fleet" || w.Target.ID == nil {
			return engine.Task{}, "merge without a fleet target"
		}
		return engine.Task{Kind: engine.TaskMerge, Fleet: l.fleetID[pvFleetKey(owner, *w.Target.ID)]}, ""
	case "transport":
		if w.Target.Kind == "fleet" {
			return engine.Task{}, "fleet-to-fleet transport"
		}
		t := engine.Task{Kind: engine.TaskTransport}
		for _, cargo := range pvKeys(w.Task.Orders) {
			o := w.Task.Orders[cargo]
			c, ok := pvCargo()[cargo]
			if !ok {
				return engine.Task{}, "transport " + cargo
			}
			switch o.Action {
			case "unload_all":
				t.Transport[c] = engine.Transport{Action: engine.UnloadAll}
			case "unload_exactly":
				t.Transport[c] = engine.Transport{Action: engine.UnloadExactly, Amount: o.Value}
			case "load_all":
				t.Transport[c] = engine.Transport{Action: engine.LoadAll}
			case "load_exactly":
				t.Transport[c] = engine.Transport{Action: engine.LoadExactly, Amount: o.Value}
			case "fill_to_percent":
				t.Transport[c] = engine.Transport{Action: engine.FillTo, Amount: o.Value}
			case "wait_for_percent":
				t.Transport[c] = engine.Transport{Action: engine.WaitFor, Amount: o.Value}
			case "set_amount_to":
				t.Transport[c] = engine.Transport{Action: engine.SetAmount, Amount: o.Value}
			case "set_waypoint_to":
				t.Transport[c] = engine.Transport{Action: engine.SetWaypoint, Amount: o.Value}
			default:
				return engine.Task{}, "transport " + o.Action
			}
		}
		return t, ""
	}
	return engine.Task{}, "task " + w.Task.Kind
}

// Result is one case's outcome.
type Result struct {
	ID     string `json:"id"`
	Corpus string `json:"corpus"`
	Tag    string `json:"tag"`
	Status string `json:"status"` // pass, fail, skip, differs, random, sample-only
	Why    string `json:"why,omitempty"`
	Passes int    `json:"passes,omitempty"` // seed variants the case passes with
	// RefStatus is, for a random case, the reference seed's status.
	RefStatus string `json:"ref_status,omitempty"`
	// Skipped is the reasons of the case's skipped expectations, one per
	// expectation, whether or not the case compared any other.
	Skipped []string `json:"skipped,omitempty"`
}

// pvCheck compares one expectation with a generated game; "" passes, a
// leading "skip: " skips.
func (l *pvLoaded) pvCheck(g *engine.Game, e pvExpect) string {
	switch e.Kind {
	case "production_queue":
		return l.queueEquals(g, e)
	case "minefield", "wormhole", "trader", "packet", "object", "object_gone":
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
		return pvObjects{}.Check(g, pvObjExpect{Kind: e.Kind, Owner: e.Owner, ID: e.ID, Equals: e.Equals, Tolerance: e.Tolerance, Subject: e.Subject}, pvMaps{l.endID, l.planet})
	}
	var eq map[string]json.RawMessage
	if len(e.Equals) > 0 && json.Unmarshal(e.Equals, &eq) != nil {
		return "skip: " + e.Kind + " list"
	}
	// fleet finds a vector fleet: one from the start by its id, and one
	// made during the run (a gift) by its owner and number.
	fleet := func(owner, id int) *engine.Fleet { return l.fleetAt(g, owner, id) }
	switch e.Kind {
	case "view":
		return l.viewCheck(g, e, eq, fleet)
	case "client_estimate":
		return l.estimateCheck(g, e, eq, fleet)
	case "salvage_at":
		// One of the salvage objects at (x, y) holds exactly these
		// minerals (several when a battle's salvage overflowed, CB-040).
		if why := l.unsupportedAt(-1); why != "" {
			return "skip: " + why
		}
		var want struct{ Minerals []int }
		if json.Unmarshal(e.Equals, &want) != nil || len(want.Minerals) != engine.NumMinerals {
			return "skip: salvage_at " + string(e.Equals)
		}
		var got []engine.Minerals
		for _, s := range g.Salvage {
			if s.Pos == (engine.Point{X: *e.X, Y: *e.Y}) {
				if s.Minerals == (engine.Minerals{want.Minerals[0], want.Minerals[1], want.Minerals[2]}) {
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
			if l.start[k] == (engine.Point{X: *e.X, Y: *e.Y}) {
				return "skip: " + l.unsupported[k] + " nearby"
			}
		}
		var last string
		for i := range g.Fleets {
			f := &g.Fleets[i]
			if f.Owner == *e.Owner && f.Pos == (engine.Point{X: *e.X, Y: *e.Y}) {
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
			if f.Owner == *e.Owner && f.Pos == (engine.Point{X: *e.X, Y: *e.Y}) {
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
		if _, ok := eq["score_record"]; ok && len(eq) == 1 {
			return l.scoreRecordCheck(g, e, eq["score_record"])
		}
		return l.playerEquals(g, *e.ID, eq, l.playerUnsure(g, *e.ID))
	case "design", "starbase_design":
		if l.planned(*e.Owner) {
			return "skip: " + pvPlannedWhy
		}
		return designEquals(g, *e.Owner, *e.Slot, e.Kind == "starbase_design", eq)
	case "battle_plan":
		if l.planned(*e.Owner) {
			return "skip: " + pvPlannedWhy
		}
		return battlePlanEquals(g, *e.Owner, *e.Slot, eq)
	case "message":
		return l.messageCheck(g, e)
	case "message_sequence":
		return l.sequenceCheck(g, e)
	}
	return "skip: expectation " + e.Kind
}

// pvVictory names the victory conditions in flag order (ScoreRecord.Flags:
// 0x40 << c; stars-elegy tools/vectors/build_kx.py CONDITIONS).
func pvVictory() []string {
	return []string{"planets_owned_percent", "tech_level", "score", "lead_over_second_percent",
		"resources_thousands", "capital_ships", "highest_score_after_years"}
}

// scoreRecordCheck checks a player's score record as the seen_by player's
// file held it (vectors/README.md "player"): the record must be one that
// player sees this year (Views[seen_by].Scores). A record resting on what
// the harness cannot reproduce for its own player is skipped; rank and the
// victory conditions, which compare every player's scores, are skipped
// when any living player's are unsure.
func (l *pvLoaded) scoreRecordCheck(g *engine.Game, e pvExpect, raw json.RawMessage) string {
	id := *e.ID
	var want struct {
		Rank      int      `json:"rank"`
		Score     int      `json:"score"`
		Resources int      `json:"resources"`
		Planets   int      `json:"planets"`
		Starbases int      `json:"starbases"`
		Unarmed   int      `json:"unarmed_ships"`
		Escorts   int      `json:"escort_ships"`
		Capital   int      `json:"capital_ships"`
		TechSum   int      `json:"tech_level_sum"`
		Victory   []string `json:"victory_conditions_met"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		return "skip: score_record " + err.Error()
	}
	if why := pvSorted(l.playerUnsure(g, id)); len(why) > 0 {
		return "skip: score_record: " + why[0]
	}
	if e.SeenBy == nil || e.Year >= len(l.views) || *e.SeenBy >= len(l.views[e.Year]) {
		return "skip: score_record without its viewer"
	}
	var rec *engine.ScoreRecord
	for i, r := range l.views[e.Year][*e.SeenBy].Scores {
		if r.Player == id {
			rec = &l.views[e.Year][*e.SeenBy].Scores[i]
		}
	}
	if rec == nil {
		return fmt.Sprintf("score_record: not among player %d's scores", *e.SeenBy)
	}
	var errs []string
	for _, c := range []struct {
		name      string
		got, want int
	}{
		{"score", rec.Score, want.Score}, {"resources", rec.Resources, want.Resources},
		{"planets", rec.Planets, want.Planets}, {"starbases", rec.Starbases, want.Starbases},
		{"unarmed_ships", pvPackCount(rec.Unarmed), want.Unarmed},
		{"escort_ships", pvPackCount(rec.Escorts), want.Escorts},
		{"capital_ships", pvPackCount(rec.Capital), want.Capital}, {"tech_level_sum", rec.TechSum, want.TechSum},
	} {
		if c.got != c.want {
			errs = append(errs, pvMismatch("score_record."+c.name, c.got, c.want))
		}
	}
	skip := ""
	for j := range g.Players {
		if why := pvSorted(l.playerUnsure(g, j)); !g.Players[j].Dead && len(why) > 0 {
			skip = fmt.Sprintf("score_record rank and victory: player %d's %s", j, why[0])
			break
		}
	}
	if skip == "" {
		if rec.Rank != want.Rank {
			errs = append(errs, pvMismatch("score_record.rank", rec.Rank, want.Rank))
		}
		var met []string
		for c, name := range pvVictory() {
			if rec.Flags&(0x40<<c) != 0 {
				met = append(met, name)
			}
		}
		w := append([]string(nil), want.Victory...)
		sort.Strings(met)
		sort.Strings(w)
		if !slices.Equal(met, w) {
			errs = append(errs, pvMismatch("score_record.victory_conditions_met", met, w))
		}
	}
	return pvResult(errs, skip)
}

// pvPackCount is a ship count as the score record stores it (stars-elegy
// 56ebfcf, BINARY-ONLY): from 8192 up the count is divided by 4, truncating,
// once per step of the exponent, and the word is exp·8192 + rest.
func pvPackCount(n int) int {
	exp := 0
	for n >= 8192 {
		n /= 4
		exp++
	}
	return exp*8192 + n
}

// pvSorted lists a reason map's reasons by key, for a stable first one.
func pvSorted(m map[string]string) []string {
	var out []string
	for _, k := range pvKeys(m) {
		out = append(out, m[k])
	}
	return out
}

// playerUnsure names the fields of player id that rest on what the
// harness cannot reproduce, with the reason for each; they are skipped
// one by one, so the player's other fields are still compared.
func (l *pvLoaded) playerUnsure(g *engine.Game, id int) map[string]string {
	unsure := map[string]string{}
	research := func(why string) {
		for _, k := range []string{"tech", "research_accumulated", "research_field", "research_next_field"} {
			if unsure[k] == "" {
				unsure[k] = why
			}
		}
	}
	if l.queued[id] {
		// A queue the harness cannot load changes the resources
		// spent on research.
		research("production queue")
	}
	if l.planned(id) {
		// A computer player's research settings follow its planned
		// orders, as its fleets and queues do.
		research(pvPlannedWhy)
	}
	if g.Players[id].Race.PRT == engine.PRTSuperStealth && l.unsureSpender(g) {
		// A Super Stealth player steals from every player's research
		// spending (KERNEL.md "Super Stealth", stars-elegy fda4173;
		// engine stealResearch). When another player's spending rests
		// on a queue the harness cannot load or on planned orders, so
		// does the stolen research, which is stored and can raise a
		// level, and a level can move the current and next fields.
		for _, k := range []string{"tech", "research_accumulated", "research_field", "research_next_field"} {
			if unsure[k] == "" {
				unsure[k] = "Super Stealth research stealing"
			}
		}
	}
	return unsure
}

// unsureSpender reports whether a player still in the game has research
// spending the harness cannot reproduce.
func (l *pvLoaded) unsureSpender(g *engine.Game) bool {
	for j, pl := range g.Players {
		if !pl.Dead && (l.queued[j] || l.planned(j)) {
			return true
		}
	}
	return false
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

// pvAlsoSkipped separates a check's mismatches from the reason one of its
// fields could not be compared (pvResult).
const pvAlsoSkipped = "\nalso skipped: "

// pvResult is a check's result from its mismatches and the reason the
// first field it could not compare was skipped ("" for none). A mismatch
// is always reported, whatever else the expectation holds; the skip
// reason rides along after pvAlsoSkipped so the tally still counts it.
// With no mismatch the result is the skip.
func pvResult(errs []string, skip string) string {
	switch {
	case len(errs) == 0 && skip == "":
		return ""
	case len(errs) == 0:
		return "skip: " + skip
	case skip == "":
		return strings.Join(errs, "; ")
	}
	return strings.Join(errs, "; ") + pvAlsoSkipped + skip
}

func pvMismatch(field string, got, want any) string {
	return fmt.Sprintf("%s = %v, want %v", field, got, want)
}

// fleetEquals compares a fleet with an expectation's fields. A field the
// harness cannot compare makes the check a skip only when every field it
// did compare matched; a mismatch is always reported (pvResult).
func (l *pvLoaded) fleetEquals(g *engine.Game, f *engine.Fleet, eq map[string]json.RawMessage) string {
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
					got = f.Cargo.Minerals[pvCargo()[c]]
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
				// The slot's design now: one a design order or a gift put
				// there during the run included.
				d, ok := l.designIn(g, f.Owner, s.Design)
				if !ok {
					d = -1 - s.Design
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
				di, _ := l.designIn(g, f.Owner, d.Design)
				found := false
				for _, s := range f.Stacks {
					if s.Design == di {
						found = true
						if s.Damage != (engine.Damage{Pct: d.PercentOfShips, Units: d.Units}) {
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
			msg, why := l.waypointsEqual(f, want)
			if msg != "" {
				errs = append(errs, msg)
			}
			if skip == "" {
				skip = why
			}
		case "first_waypoint_task":
			var want string
			json.Unmarshal(raw, &want)
			got := map[engine.TaskKind]string{engine.TaskNone: "none", engine.TaskTransport: "transport", engine.TaskColonize: "colonize", engine.TaskMerge: "merge", engine.TaskRoute: "route", engine.TaskPatrol: "patrol", engine.TaskTransferFleet: "transfer", engine.TaskLayMines: "lay_mines", engine.TaskRemoteMine: "remote_mine", engine.TaskScrap: "scrap"}[f.Task.Kind]
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		default:
			if skip == "" {
				skip = "fleet field " + k
			}
		}
	}
	return pvResult(errs, skip)
}

// waypointsEqual compares a fleet's waypoint list with the vector's, where
// waypoint 0 is the fleet's own location. Elegy keeps only waypoint 0's
// task (Fleet.Task), so its warp and target are not compared. It returns
// the mismatches and the reason the first waypoint it could not compare
// was skipped; the other waypoints are still compared.
func (l *pvLoaded) waypointsEqual(f *engine.Fleet, want []pvWaypoint) (mismatch, skip string) {
	if len(want) == 0 {
		return "", "no waypoint 0"
	}
	var errs []string
	skipped := func(why string) {
		if skip == "" {
			skip = "waypoint " + why
		}
	}
	if p := (engine.Point{X: want[0].X, Y: want[0].Y}); p != f.Pos {
		errs = append(errs, pvMismatch("waypoints[0] position", f.Pos, p))
	}
	if task, why := l.task(want[0]); why != "" {
		skipped(why)
	} else if !reflect.DeepEqual(f.Task, task) {
		errs = append(errs, pvMismatch("waypoints[0] task", f.Task, task))
	}
	if len(f.Waypoints) != len(want)-1 {
		got := make([]engine.Point, len(f.Waypoints))
		for i, wp := range f.Waypoints {
			got[i] = wp.Pos
		}
		return strings.Join(append(errs, fmt.Sprintf("%d waypoints after waypoint 0 %v, want %d", len(f.Waypoints), got, len(want)-1)), "; "), skip
	}
	for i, w := range want[1:] {
		wp, why := l.waypoint(f.Owner, w)
		if why == "" {
			wp.Task, why = l.task(w)
		}
		if why != "" {
			skipped(why)
			continue
		}
		if got := f.Waypoints[i]; !reflect.DeepEqual(got, wp) {
			errs = append(errs, pvMismatch(fmt.Sprintf("waypoints[%d]", i+1), got, wp))
		}
	}
	return strings.Join(errs, "; "), skip
}

func (l *pvLoaded) planetEquals(g *engine.Game, p *engine.Planet, eq map[string]json.RawMessage, tol map[string]int) string {
	var errs []string
	skip := ""
	skipped := func(why string) {
		if skip == "" {
			skip = why
		}
	}
	for _, k := range pvKeys(eq) {
		raw := eq[k]
		if string(raw) == "null" && string(eq["population"]) != "null" {
			// stars-elegy ORACLE.md (main 1d4c97b, the `dump` notes):
			// "The game omits a planet's 8-byte installations block when
			// growth carry, mines, factories, defenses and the
			// leftover-only box are all 0 and there is no scanner." The
			// vectors then write null for those fields where the start
			// state had them. Reading a missing block as those zero
			// values is the converse of that rule (ASSUMPTION: the block
			// is written whenever any of them is non-zero). One field
			// read this way misses: RG-001-2435 y6 planet 41 excess, on a
			// population that already differs; the y6 restart, from the
			// oracle's year-5 state, matches it (docs/PARITY-STATUS.md).
			// The ORACLE.md note that pins this
			// to populated planets ("Populations above 0 with no mines,
			// factories, defenses or carry are written without an
			// installations section") says nothing about a record whose
			// population is null too, so that record stays skipped.
			switch k {
			case "excess", "mines", "factories", "defenses":
				raw = json.RawMessage("0")
			case "leftover_to_research":
				raw = json.RawMessage("false")
			case "planetary_scanner":
				// null is no planetary scanner, here or as the start
				// state's scanner index 31 (loadVector).
				if p.HasScanner {
					errs = append(errs, pvMismatch(k, "a scanner", "none"))
				}
				continue
			}
		}
		if string(raw) == "null" && (k == "packet_destination" || k == "packet_warp") {
			// vectors/README.md "initial_state": a planet whose
			// mass-driver settings are the default has no
			// packet_destination or packet_warp, and absent means no
			// destination and warp 4. The vector diff writes null where
			// the start state had them. Every such record so far is a
			// planet whose starbase was lost in the year, and whether the
			// original cleared the settings or only does not write them
			// for a planet with no starbase is UNRESOLVED, so a planet
			// left with no starbase is not compared.
			if !p.HasStarbase {
				skipped("planet field " + k + " null with no starbase (cleared or not written: UNRESOLVED)")
				continue
			}
			raw = json.RawMessage("-1")
			if k == "packet_warp" {
				raw = json.RawMessage("4")
			}
		}
		if string(raw) == "null" && k != "environment" && k != "original_environment" && k != "starbase" {
			// The vectors write null for a value the original's file
			// does not hold, such as an unowned planet's population
			// and growth carry: it is not a 0 to compare.
			skipped("planet field " + k + " null")
			continue
		}
		switch k {
		case "owner":
			var want int
			json.Unmarshal(raw, &want)
			got := p.Owner
			if got == engine.NoOwner {
				got = -1
			}
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "homeworld":
			var want bool
			json.Unmarshal(raw, &want)
			if p.Homeworld != want {
				errs = append(errs, pvMismatch(k, p.Homeworld, want))
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
		case "starbase":
			// The planet's starbase: its owner's starbase design slot and
			// its damage, or null for none (vectors/README.md "planets").
			// null, or design null with damage -1 as the states write
			// it, is no starbase. Any other value must name a design:
			// malformed expected data fails the check rather than
			// reading as no starbase.
			var want *struct {
				Design *int `json:"design"`
				Damage int  `json:"damage"`
			}
			if err := json.Unmarshal(raw, &want); err != nil || want != nil && want.Design == nil && want.Damage != -1 {
				errs = append(errs, fmt.Sprintf("starbase: malformed expectation %s", raw))
				continue
			}
			if want == nil || want.Design == nil {
				if p.HasStarbase {
					errs = append(errs, pvMismatch(k, fmt.Sprintf("design %d", p.StarbaseDesign), "none"))
				}
				continue
			}
			d, ok := g.PlayerDesign(p.Owner, true, *want.Design)
			if !ok {
				d, ok = l.design[[2]int{p.Owner, -1 - *want.Design}]
			}
			if !ok {
				skipped(fmt.Sprintf("starbase design %d", *want.Design))
				continue
			}
			if !p.HasStarbase || p.StarbaseDesign != d || p.StarbaseDamage != max(0, want.Damage) {
				got := "none"
				if p.HasStarbase {
					got = fmt.Sprintf("design %d damage %d", p.StarbaseDesign, p.StarbaseDamage)
				}
				errs = append(errs, pvMismatch(k, got, fmt.Sprintf("design %d damage %d", d, max(0, want.Damage))))
			}
		case "concentrations":
			// The stored concentration bytes (KERNEL.md "Mining":
			// depletion uses the mining before its random +1).
			var want [3]int
			json.Unmarshal(raw, &want)
			got := [3]int{p.Deposits[0].Concentration, p.Deposits[1].Concentration, p.Deposits[2].Concentration}
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "environment", "original_environment":
			var want *[3]int
			json.Unmarshal(raw, &want)
			got := p.Env
			if k == "original_environment" {
				got = p.OrigEnv
			}
			switch {
			case want == nil && k == "original_environment":
				// The game stores no separate original values when
				// they equal the current ones (TAKEOVER.md "Capture",
				// CONFIRMED TK-116), and the vectors write null.
				want = &p.Env
			case want == nil:
				skipped("environment null")
				continue
			}
			if got != *want {
				errs = append(errs, pvMismatch(k, got, *want))
			}
		case "packet_destination":
			// The mass driver's target planet, -1 for none (vectors/README.md
			// "planet_settings"). A null, where the original's file holds no
			// value, is not compared (above): ELEGY CHOICE. Malformed
			// expected data fails the check, never reads as absent.
			var want int
			if json.Unmarshal(raw, &want) != nil {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, raw))
				continue
			}
			got := -1
			if p.HasPacketDest {
				got = p.PacketDest - 1
			}
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "packet_warp":
			// The packet speed setting, 4..19; a setting below 5 counts as
			// unset (vectors/README.md "initial_state", OBJECTS.md "Launch"),
			// which Elegy keeps as 0.
			var want int
			if json.Unmarshal(raw, &want) != nil {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, raw))
				continue
			}
			got := p.PacketSpeed
			if got < 5 && want < 5 {
				got = want
			}
			if got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "leftover_to_research":
			// A newly owned planet takes its new owner's default setting,
			// which not every vector carries. A planet left with no owner
			// takes none.
			changed := true
			for _, p0 := range l.g.Planets {
				if p0.ID == p.ID {
					changed = p0.Owner != p.Owner
				}
			}
			var want bool
			switch {
			case changed && p.Owner != engine.NoOwner && !l.defaults[p.Owner]:
				skipped("player default queue settings")
			case json.Unmarshal(raw, &want) != nil:
				skipped("leftover_to_research " + string(raw))
			case p.LeftoverOnly != want:
				errs = append(errs, pvMismatch(k, p.LeftoverOnly, want))
			}
		default:
			skipped("planet field " + k)
		}
	}
	return pvResult(errs, skip)
}

// playerEquals compares a player with an expectation's fields; a field
// named in unsure is skipped with its reason.
func (l *pvLoaded) playerEquals(g *engine.Game, id int, eq map[string]json.RawMessage, unsure map[string]string) string {
	var errs []string
	skip := ""
	skipped := func(why string) {
		if skip == "" {
			skip = why
		}
	}
	for _, k := range pvKeys(eq) {
		if why := unsure[k]; why != "" {
			skipped(why)
			continue
		}
		switch k {
		case "research_percent":
			// Malformed expected data, null included, fails the check
			// instead of reading as absent or as 0.
			var want int
			if string(eq[k]) == "null" || json.Unmarshal(eq[k], &want) != nil {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, eq[k]))
				continue
			}
			if got := g.Players[id].ResearchBudget; got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "tech", "research_accumulated":
			var want map[string]int
			json.Unmarshal(eq[k], &want)
			for _, f := range pvKeys(want) {
				got := g.Players[id].Research.Levels[pvFields()[f]]
				if k == "research_accumulated" {
					got = g.Players[id].Research.Accumulated[pvFields()[f]]
				}
				if got != want[f] {
					errs = append(errs, pvMismatch(k+"."+f, got, want[f]))
				}
			}
		case "research_field":
			var want string
			json.Unmarshal(eq[k], &want)
			f, ok := pvFields()[want]
			if !ok {
				skipped("research_field " + string(eq[k]))
				continue
			}
			if got := g.Players[id].Research.Current; got != f {
				errs = append(errs, pvMismatch(k, got, f))
			}
		case "research_next_field":
			var want string
			json.Unmarshal(eq[k], &want)
			f, ok := map[string]int{"same": engine.NextSameField, "lowest": engine.NextLowestField}[want]
			if !ok {
				f, ok = pvFields()[want]
			}
			if !ok {
				skipped("research_next_field " + string(eq[k]))
				continue
			}
			if got := g.Players[id].Research.Next; got != f {
				errs = append(errs, pvMismatch(k, got, f))
			}
		case "default_queue":
			var raw []json.RawMessage
			if err := json.Unmarshal(eq[k], &raw); err != nil {
				skipped("default_queue " + err.Error())
				continue
			}
			want, why := pvDefaultQueue(raw)
			if why != "" {
				skipped(why)
				continue
			}
			if got := g.Players[id].DefaultQueue; !reflect.DeepEqual(got, want) {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "default_leftover_to_research":
			var want bool
			if err := json.Unmarshal(eq[k], &want); err != nil {
				skipped("default_leftover_to_research " + string(eq[k]))
				continue
			}
			if got := g.Players[id].DefaultLeftoverOnly; got != want {
				errs = append(errs, pvMismatch(k, got, want))
			}
		case "counts":
			// The player's numbers of ship and starbase designs.
			var want map[string]int
			if json.Unmarshal(eq[k], &want) != nil || len(want) == 0 {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, eq[k]))
				continue
			}
			got := map[string]int{"ship_designs": 0, "starbase_designs": 0}
			for _, s := range g.DesignSlots {
				switch {
				case s.Owner != id:
				case s.Starbase:
					got["starbase_designs"]++
				default:
					got["ship_designs"]++
				}
			}
			for _, c := range pvKeys(want) {
				n, ok := got[c]
				if !ok {
					skipped("player counts " + c)
					continue
				}
				if n != want[c] {
					errs = append(errs, pvMismatch(k+" "+c, n, want[c]))
				}
			}
		case "race":
			var q pvRace
			if err := json.Unmarshal(eq[k], &q); err != nil {
				skipped("race: " + err.Error())
				continue
			}
			want, err := q.race()
			if err != nil {
				skipped("race: " + err.Error())
				continue
			}
			if got := g.Players[id].Race; got != want {
				errs = append(errs, pvMismatch("race", fmt.Sprintf("%+v", got), fmt.Sprintf("%+v", want)))
			}
			if g.Races == nil {
				skipped("race settings without a race check")
				continue
			}
			st, ok := pvRaceSettingsOf(g.Races, id)
			if !ok {
				skipped("race settings without a race check")
				continue
			}
			spend := pvLeftoverSpend()[q.LeftoverSpend]
			if st.Spend != spend || st.Stat15 != q.Stat15 || st.ExpensiveAt3 != q.TechsStartHigh {
				errs = append(errs, pvMismatch("race settings", fmt.Sprintf("spend %d, stat 15 %d, techs start high %v", st.Spend, st.Stat15, st.ExpensiveAt3),
					fmt.Sprintf("spend %d, stat 15 %d, techs start high %v", spend, q.Stat15, q.TechsStartHigh)))
			}
		default:
			skipped("player field " + k)
		}
	}
	return pvResult(errs, skip)
}

// designEquals checks the design in owner's ship (or starbase) design
// slot at the end of the year (README.md "design"): its hull, its mass
// per ship, and its slots in hull-slot order, null for an empty one.
// Malformed expected data fails the check.
func designEquals(g *engine.Game, owner, slot int, starbase bool, eq map[string]json.RawMessage) string {
	var d *engine.Design
	for _, s := range g.DesignSlots {
		if s.Owner == owner && s.Slot == slot && s.Starbase == starbase {
			d = &g.Designs[s.Design]
		}
	}
	if d == nil {
		return fmt.Sprintf("no design in slot %d of player %d", slot, owner)
	}
	var errs []string
	skip := ""
	for _, k := range pvKeys(eq) {
		switch k {
		case "hull":
			var want string
			if json.Unmarshal(eq[k], &want) != nil {
				errs = append(errs, fmt.Sprintf("hull: malformed expectation %s", eq[k]))
			} else if d.Hull.Name != want {
				errs = append(errs, pvMismatch(k, d.Hull.Name, want))
			}
		case "mass":
			var want int
			if string(eq[k]) == "null" || json.Unmarshal(eq[k], &want) != nil {
				errs = append(errs, fmt.Sprintf("mass: malformed expectation %s", eq[k]))
			} else if d.Mass != want {
				errs = append(errs, pvMismatch(k, d.Mass, want))
			}
		case "slots":
			var want []*pvPart
			if json.Unmarshal(eq[k], &want) != nil {
				errs = append(errs, fmt.Sprintf("slots: malformed expectation %s", eq[k]))
				continue
			}
			if len(d.SlotPos) != len(d.Slots) {
				if skip == "" {
					skip = "design slots without hull-slot positions"
				}
				continue
			}
			got := make([]*pvPart, len(want))
			for i, s := range d.Slots {
				p := d.SlotPos[i]
				if p >= len(got) {
					got = append(got, make([]*pvPart, p+1-len(got))...)
				}
				got[p] = &pvPart{Count: s.Count, Part: s.Part.Name}
			}
			if g, w := pvSlotsText(got), pvSlotsText(want); g != w {
				errs = append(errs, pvMismatch(k, g, w))
			}
		default:
			if skip == "" {
				skip = "design field " + k
			}
		}
	}
	return pvResult(errs, skip)
}

// battlePlanEquals compares player owner's battle plan in slot with an
// expectation's fields at the end of the year (vectors/README.md
// "battle_plan": `name`, `tactic`, `primary`, `secondary`, `attack_who`,
// `dump_cargo`, numbered as the state's battle_plans). Malformed expected
// data fails the check.
func battlePlanEquals(g *engine.Game, owner, slot int, eq map[string]json.RawMessage) string {
	if owner < 0 || owner >= len(g.Players) {
		return "skip: player not in the state"
	}
	plans := g.Players[owner].Plans
	if slot < 0 || slot >= len(plans) {
		return fmt.Sprintf("no battle plan in slot %d of player %d", slot, owner)
	}
	bp := plans[slot]
	attack := int(bp.Attack)
	if bp.Attack == engine.AttackPlayer {
		attack += bp.Player
	}
	ints := map[string]int{"tactic": int(bp.Tactic), "primary": int(bp.Primary), "secondary": int(bp.Secondary), "attack_who": attack}
	var errs []string
	skip := ""
	for _, k := range pvKeys(eq) {
		switch k {
		case "tactic", "primary", "secondary", "attack_who":
			var want int
			if string(eq[k]) == "null" || json.Unmarshal(eq[k], &want) != nil {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, eq[k]))
			} else if ints[k] != want {
				errs = append(errs, pvMismatch(k, ints[k], want))
			}
		case "dump_cargo":
			var want bool
			if json.Unmarshal(eq[k], &want) != nil || string(eq[k]) == "null" {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, eq[k]))
			} else if bp.DumpCargo != want {
				errs = append(errs, pvMismatch(k, bp.DumpCargo, want))
			}
		case "name":
			var want string
			if json.Unmarshal(eq[k], &want) != nil || string(eq[k]) == "null" {
				errs = append(errs, fmt.Sprintf("%s: malformed expectation %s", k, eq[k]))
			} else if bp.Name != want {
				errs = append(errs, pvMismatch(k, fmt.Sprintf("%q", bp.Name), fmt.Sprintf("%q", want)))
			}
		default:
			if skip == "" {
				skip = "battle plan field " + k
			}
		}
	}
	return pvResult(errs, skip)
}

// pvSlotsText writes design slots as "count part" per hull slot, "-" for
// an empty one.
func pvSlotsText(slots []*pvPart) string {
	out := make([]string, len(slots))
	for i, s := range slots {
		out[i] = "-"
		if s != nil {
			out[i] = fmt.Sprintf("%d %s", s.Count, s.Part)
		}
	}
	return "[" + strings.Join(out, ", ") + "]"
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

// runVector generates v's years with the seed variant k (0 is the
// harness's reference seeding) and checks its cases.
func runVector(v *Vector, k int) []Result {
	return runVectorWith(v, pvRun{seed: k})
}

// runVectorWith generates v's years as run says and checks its cases.
func runVectorWith(v *Vector, run pvRun) []Result {
	l, games, genErr, stop := generate(v, run)
	return checkCases(v, run, l, games, genErr, stop)
}

// generate loads v and generates its years as run says. games holds the
// start and every year generated. stop is why the years from len(games)
// on were not generated: the vector could not be loaded, or the orders of
// a year could not be applied; genErr is the engine error that stopped
// generation early instead.
func generate(v *Vector, run pvRun) (l *pvLoaded, games []engine.Game, genErr error, stop string) {
	l, err := loadVector(v)
	switch {
	case err != nil:
		return nil, nil, nil, "load: " + err.Error()
	case l.global != "":
		return l, nil, nil, l.global
	}
	// Generate year by year, keeping each year's game.
	games = []engine.Game{l.g}
	l.views = [][]engine.PlayerView{nil}
	l.events = [][]engine.Event{nil}
	g := l.g
	for y := 1; y <= v.Years; y++ {
		files, why := l.orders(v.Orders, y, &g)
		if why != "" {
			return l, games, nil, why
		}
		gy := g
		gy.Rules = engine.ElegyRules()
		turn := engine.GenerateTurn
		if run.turn != nil {
			turn = run.turn
		}
		r, err := turn(gy, files, run.rand(y))
		if err != nil {
			return l, games, err, ""
		}
		for _, o := range r.Orders {
			if o.Err == nil {
				continue
			}
			kind := "file"
			for _, f := range files {
				if f.Player == o.Player && o.Index >= 0 && o.Index < len(f.Orders) {
					kind = fmt.Sprintf("%T", f.Orders[o.Index])
				}
			}
			l.rejected = append(l.rejected, fmt.Sprintf("year %d player %d order %d (%s): %v", y, o.Player, o.Index, strings.TrimPrefix(kind, "engine."), o.Err))
		}
		g = r.Game
		games = append(games, g)
		l.views = append(l.views, r.Views)
		l.events = append(l.events, r.Events)
	}
	return l, games, nil, ""
}

// checkCases checks every case of v against the generated games. Each
// expectation of a year not generated is skipped with stop, one by one,
// so the years before it are still compared.
func checkCases(v *Vector, run pvRun, l *pvLoaded, games []engine.Game, genErr error, stop string) []Result {
	var out []Result
	res := func(c pvCase, status, why string) {
		out = append(out, Result{ID: c.ID, Corpus: v.corpus, Tag: c.Tag, Status: status, Why: why})
	}
	for _, c := range v.Cases {
		var fails, skips []string
		// checked counts the compared expectations; exact, those of them
		// that are not samples.
		checked, exact := 0, 0
		for i, e := range c.Expect {
			if !run.checks(e) {
				skips = append(skips, "stream "+e.Stream)
				run.observe1(c, i, "skip: stream "+e.Stream)
				continue
			}
			if e.Year >= len(games) && stop != "" {
				skips = append(skips, stop)
				run.observe1(c, i, "skip: "+stop)
				continue
			}
			if e.Year >= len(games) {
				fails = append(fails, fmt.Sprintf("year %d not generated: %v", e.Year, genErr))
				run.observe1(c, i, "year not generated")
				continue
			}
			msg, also, _ := strings.Cut(l.pvCheck(&games[e.Year], e), pvAlsoSkipped)
			if also != "" {
				skips = append(skips, also)
			}
			run.observe1(c, i, msg)
			switch {
			case strings.HasPrefix(msg, "skip: "):
				skips = append(skips, strings.TrimPrefix(msg, "skip: "))
			case msg != "" && run.sample(e):
				skips = append(skips, "sample: one stream's random outcome")
			case msg != "":
				checked++
				exact++
				fails = append(fails, fmt.Sprintf("y%d %s: %s", e.Year, e.Kind, msg))
			default:
				checked++
				if !run.sample(e) {
					exact++
				}
			}
		}
		switch {
		case len(fails) > 0 && pvLegacyOff()[c.ID] != "":
			res(c, "differs", pvLegacyOff()[c.ID]+": "+strings.Join(fails, " | "))
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
		// Every skipped expectation is tallied, a wholly skipped case's
		// too, so the census names each reason a case was not compared.
		out[len(out)-1].Skipped = skips
	}
	return out
}

// estimateCheck checks a client_estimate expectation (ESTIMATES.md) on
// the fleet estimates Elegy computes: "Est. range" and "Est. fuel usage".
// Other readings are skipped by screen and field.
func (l *pvLoaded) estimateCheck(g *engine.Game, e pvExpect, eq map[string]json.RawMessage, fleet func(owner, id int) *engine.Fleet) string {
	var sub struct {
		Kind     string `json:"kind"`
		Owner    int    `json:"owner"`
		ID       int    `json:"id"`
		Waypoint int    `json:"waypoint"`
	}
	json.Unmarshal(e.Subject, &sub)
	reading := e.Screen + " " + e.Field
	if sub.Kind != "fleet" || (reading != "fleet_composition range" && reading != "fleet_waypoints fuel_usage") {
		return "skip: client estimate " + reading
	}
	if l.planned(sub.Owner) {
		return "skip: " + pvPlannedWhy
	}
	f := fleet(sub.Owner, sub.ID)
	if f == nil {
		return "fleet gone"
	}
	got := map[string]any{}
	if e.Field == "range" {
		if r, unlimited := g.EstRange(f); unlimited {
			got["state"] = "infinite"
		} else {
			got["light_years"] = r
		}
	} else {
		mg := g.EstFuelUsage(f, sub.Waypoint)
		got["mg"] = mg
		if mg > f.Fuel {
			got["warning"] = true
		}
	}
	gb, _ := json.Marshal(got)
	var gm, wm map[string]any
	json.Unmarshal(gb, &gm)
	wb, _ := json.Marshal(eq)
	json.Unmarshal(wb, &wm)
	if !reflect.DeepEqual(gm, wm) {
		return fmt.Sprintf("%s: %s, want %s", reading, gb, wb)
	}
	return ""
}

// pvPlanetLevels is the vectors' planet report level for each of
// Elegy's (SCANNING.md "What a planet report contains"; PARITY.md
// "Co-location and orbit reports": 1 position, 3 normal, 4 detailed).
func pvPlanetLevels() map[engine.ReportLevel]int {
	return map[engine.ReportLevel]int{engine.ReportNone: 0, engine.ReportPosition: 1, engine.ReportNormal: 3, engine.ReportDetailed: 4}
}

// viewCheck checks a view expectation: what player viewer knows of a
// planet, a fleet or a space object at the end of the year (vectors
// README.md "view").
func (l *pvLoaded) viewCheck(g *engine.Game, e pvExpect, eq map[string]json.RawMessage, fleet func(owner, id int) *engine.Fleet) string {
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
		level, sb := engine.ReportNone, false
		for _, r := range v.Planets {
			if r.Planet == id {
				level, sb = r.Level, r.Starbase
			}
		}
		if level == engine.ReportOwn {
			return "skip: view of an own planet"
		}
		got["level"], got["starbase_visible"] = pvPlanetLevels()[level], sb
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
	return pvCompare(sub.Kind, got, eq, 0, nil)
}

// pvLegacyOff names the LEGACY BUG cases whose switch is off by default,
// where Elegy's chosen rule intentionally differs from the original.
func pvLegacyOff() map[string]string {
	return map[string]string{
		"FO-03-E": "Legacy.MergeOverflow off",
		"FO-06-G": "Legacy.MergeOverflow off",
	}
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

// Seeds is how many seed variants each vector runs with. The harness's
// random stream is not the original's (the oracle's seeds are not in the
// vectors), so a case whose outcome depends on the draws matches only by
// chance. Such a case passes with some seeds and fails with others; it is
// reported as "random" with its pass count and the reference seed's
// comparison kept. The tally counts random cases whose reference seed
// fails apart, and the baseline lists each random case with its pass
// count.
const Seeds = 8

// pvStreamCheck folds one vector's seed variants into one result per case.
func pvStreamCheck(runs [][]Result) []Result {
	out := runs[0]
	for i, r := range out {
		passes, samples, same, onlySamples := 0, 0, true, true
		for _, run := range runs {
			if run[i].ID != r.ID {
				panic("parity: seed variants disagree on case order")
			}
			switch run[i].Status {
			case "pass":
				passes++
			case "sample-only":
				samples++
			}
			same = same && run[i].Status == r.Status
			onlySamples = onlySamples && (run[i].Status == "sample-only" || run[i].Status == "skip")
		}
		switch {
		case !same && onlySamples:
			// Each seed either matched only samples or had every sample
			// miss: sample-only on the seeds that matched.
			out[i].Status = "sample-only"
			out[i].Passes = samples
			out[i].Why = fmt.Sprintf("sample-only with %d of %d seeds; every sample missed on the others", samples, len(runs))
		case !same:
			why := r.Why
			if why == "" {
				why = "matches"
			}
			out[i].Status = "random"
			out[i].Passes = passes
			out[i].RefStatus = r.Status
			out[i].Why = fmt.Sprintf("passes with %d of %d seeds; reference seed %s: %s", passes, len(runs), r.Status, why)
		case r.Status == "sample-only":
			out[i].Passes = len(runs)
		}
	}
	return out
}
