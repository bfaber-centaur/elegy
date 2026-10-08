package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// The component table: every J-RC3 hull, starbase hull, ship part and
// planetary item, from stars-elegy data/components.json (see
// data/README.md for the copy's source and hash). COMPONENTS.md explains
// the columns; this file turns rows into the engine's Part, Hull, Engine,
// PlanetScanner and DefenseType values.

//go:embed data/components.json
var componentsJSON []byte

// Component categories (COMPONENTS.md "Columns").
const (
	CatHull         = "hull"
	CatStarbaseHull = "starbase_hull"
	CatEngine       = "engine"
	CatScanner      = "scanner"
	CatShield       = "shield"
	CatArmor        = "armor"
	CatBeam         = "beam"
	CatTorpedo      = "torpedo"
	CatBomb         = "bomb"
	CatMiningRobot  = "mining_robot"
	CatMineLayer    = "mine_layer"
	CatOrbital      = "orbital"
	CatElectrical   = "electrical"
	CatMechanical   = "mechanical"
	CatPlanetary    = "planetary"
	CatTerraform    = "terraform"
)

// Restriction is a component's race rule (COMPONENTS.md "Who can build
// what", CONFIRMED CS-001), in the table's trait abbreviations.
type Restriction struct {
	PRTOnly      []string `json:"prt_only"`
	PRTNot       []string `json:"prt_not"`
	LRTRequired  []string `json:"lrt_required"`
	LRTForbidden []string `json:"lrt_forbidden"`
}

// Component is one row of the table.
type Component struct {
	Category      string
	Index         int
	Name          string
	TechReq       [NumFields]int
	Mass          int // 0 when the table has no mass (planetary, terraform)
	Cost          Cost
	Restriction   Restriction
	MysteryTrader bool
	// Stats are the row's category values, as in the table.
	Stats map[string]any
	// Status is CONFIRMED or BINARY-ONLY; BinaryOnly names the columns
	// read from the original program only.
	Status     string
	BinaryOnly []string
}

// Catalog is the parsed component table.
type Catalog struct {
	Components []Component
	byName     map[string]int
}

type rawComponent struct {
	Category string         `json:"category"`
	Index    int            `json:"index"`
	Name     string         `json:"name"`
	Tech     map[string]int `json:"tech"`
	Mass     *int           `json:"mass"`
	Cost     struct {
		Resources int `json:"resources"`
		Ironium   int `json:"ironium"`
		Boranium  int `json:"boranium"`
		Germanium int `json:"germanium"`
	} `json:"cost"`
	Restriction   Restriction    `json:"restriction"`
	MysteryTrader bool           `json:"mystery_trader"`
	Stats         map[string]any `json:"stats"`
	Status        string         `json:"status"`
	BinaryOnly    []string       `json:"binary_only"`
}

var fieldNames = [NumFields]string{"energy", "weapons", "propulsion", "construction", "electronics", "biotechnology"}

// ParseCatalog parses a component table in the stars-elegy format.
func ParseCatalog(data []byte) (*Catalog, error) {
	var raw struct {
		Schema string         `json:"schema"`
		Items  []rawComponent `json:"items"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Schema != "stars-elegy components v1" {
		return nil, fmt.Errorf("component table: unknown schema %q", raw.Schema)
	}
	c := &Catalog{byName: map[string]int{}}
	for _, r := range raw.Items {
		comp := Component{
			Category: r.Category, Index: r.Index, Name: r.Name,
			Cost:          Cost{Resources: r.Cost.Resources, Minerals: Minerals{r.Cost.Ironium, r.Cost.Boranium, r.Cost.Germanium}},
			Restriction:   r.Restriction,
			MysteryTrader: r.MysteryTrader,
			Stats:         r.Stats,
			Status:        r.Status,
			BinaryOnly:    r.BinaryOnly,
		}
		for f, name := range fieldNames {
			comp.TechReq[f] = r.Tech[name]
		}
		if r.Mass != nil {
			comp.Mass = *r.Mass
		}
		if _, dup := c.byName[r.Name]; dup {
			return nil, fmt.Errorf("component table: duplicate name %q", r.Name)
		}
		c.byName[r.Name] = len(c.Components)
		c.Components = append(c.Components, comp)
	}
	return c, nil
}

var (
	defaultCatalog     *Catalog
	defaultCatalogOnce sync.Once
)

// Components returns the built-in J-RC3 component table. The table is
// embedded and checked by tests, so a parse error is a build defect.
func Components() *Catalog {
	defaultCatalogOnce.Do(func() {
		c, err := ParseCatalog(componentsJSON)
		if err != nil {
			panic(err)
		}
		defaultCatalog = c
	})
	return defaultCatalog
}

// Lookup returns the component with this name.
func (c *Catalog) Lookup(name string) (Component, bool) {
	i, ok := c.byName[name]
	if !ok {
		return Component{}, false
	}
	return c.Components[i], true
}

func (c Component) num(key string) int {
	v, _ := c.Stats[key].(float64)
	return int(v)
}

func (c Component) flag(key string) bool {
	v, _ := c.Stats[key].(bool)
	return v
}

func (c Component) str(key string) string {
	v, _ := c.Stats[key].(string)
	return v
}

func (c Component) has(key string) bool {
	_, ok := c.Stats[key]
	return ok
}

// Kind is the component's PartKind.
func (c Component) Kind() PartKind {
	switch c.Category {
	case CatEngine:
		return PartEngine
	case CatScanner:
		return PartScanner
	case CatShield:
		return PartShield
	case CatArmor:
		return PartArmor
	case CatBeam:
		return PartBeam
	case CatTorpedo:
		return PartTorpedo
	case CatBomb:
		return PartBomb
	case CatMiningRobot:
		return PartMiningRobot
	case CatMineLayer:
		return PartMineLayer
	case CatOrbital:
		if c.str("kind") == "stargate" {
			return PartStargate
		}
		return PartMassDriver
	case CatElectrical:
		return PartElectrical
	case CatMechanical:
		return PartMechanical
	case CatPlanetary:
		return PartPlanetary
	case CatTerraform:
		return PartTerraform
	}
	return PartOther
}

// slotKind maps a slot's accepted kind (COMPONENTS.md "Hull") to the
// part kinds it admits.
func slotKinds(kind string) ([]PartKind, error) {
	switch kind {
	case "engine":
		return []PartKind{PartEngine}, nil
	case "scanner":
		return []PartKind{PartScanner}, nil
	case "shield":
		return []PartKind{PartShield}, nil
	case "armor":
		return []PartKind{PartArmor}, nil
	case "beam":
		return []PartKind{PartBeam}, nil
	case "torpedo":
		return []PartKind{PartTorpedo}, nil
	case "bomb":
		return []PartKind{PartBomb}, nil
	case "mining_robot":
		return []PartKind{PartMiningRobot}, nil
	case "mine_layer":
		return []PartKind{PartMineLayer}, nil
	case "orbital":
		return []PartKind{PartStargate, PartMassDriver}, nil
	case "electrical":
		return []PartKind{PartElectrical}, nil
	case "mechanical":
		return []PartKind{PartMechanical}, nil
	}
	return nil, fmt.Errorf("unknown slot kind %q", kind)
}

// partStats are the stats Part takes over; deferredStats are table
// columns no Elegy rule uses yet. Every column of a part row is in one of
// the two (TestCatalogStatsAccountedFor).
var partStats = map[string]bool{
	"damage": true, "range": true, "initiative": true, "accuracy": true, "kind": true,
	"armor": true, "shield": true, "cloak_points": true, "jammer_pct": true,
	"scanner_range": true, "penetrating_range": true, "steals_cargo": true,
	"accuracy_pct": true, "computer_pct": true, "beam_damage_pct": true,
	"beam_deflection_pct": true, "battle_speed": true, "battle_speed_half_steps": true,
	"tachyon": true, "fuel_capacity": true, "cargo_capacity": true, "fuel_per_year": true,
	// Bombs and colonizers (TAKEOVER.md).
	"kill_tenths_pct": true, "installations": true, "min_kill": true,
	"bomb_kill_tenths_pct": true, "bomb_installations": true, "bomb_min_kill": true,
	"colonizes": true,
	// Engines: fuel_table, warp10_rated and battle_warp go to Engine.
	"fuel_table": true, "warp10_rated": true, "battle_warp": true, "free_warps": true,
}

var deferredStats = map[string]bool{
	// Mine sweeping, laying and fields; remote mining; terraforming by
	// ship; bombs; colonizing; stargates and mass drivers; fuel
	// generation; jump gates. Elegy has none of these rules yet. The
	// Orbital Construction Module's own flag: Elegy reads its colonizing
	// and bomb values, which are columns of their own.
	"mines_swept": true, "mines_laid": true, "mines_per_year": true, "field": true,
	"mining_rate": true, "terraform_pct": true,
	"orbital_construction": true,
	"safe_mass":            true, "safe_range": true, "warp": true,
	"jump_gate": true,
}

// Part converts a ship or starbase part row to a Part.
func (c Component) Part() (Part, error) {
	switch c.Category {
	case CatHull, CatStarbaseHull, CatPlanetary, CatTerraform:
		return Part{}, fmt.Errorf("%s: a %s is not a part", c.Name, c.Category)
	}
	p := Part{Name: c.Name, Kind: c.Kind(), Mass: c.Mass, Cost: c.Cost, TechReq: c.TechReq}
	switch c.Category {
	case CatBeam:
		p.Damage, p.Range, p.Initiative = c.num("damage"), c.num("range"), c.num("initiative")
		switch c.str("kind") {
		case "gatling":
			p.Gatling = true
		case "sapper":
			p.Sapper = true
		case "beam":
		default:
			return Part{}, fmt.Errorf("%s: unknown beam kind %q", c.Name, c.str("kind"))
		}
	case CatTorpedo:
		p.Damage, p.Range, p.Initiative = c.num("damage"), c.num("range"), c.num("initiative")
		p.Accuracy = c.num("accuracy")
		switch c.str("kind") {
		case "missile":
			p.Missile = true
		case "torpedo":
		default:
			return Part{}, fmt.Errorf("%s: unknown torpedo kind %q", c.Name, c.str("kind"))
		}
	case CatScanner:
		p.Scanner = true
		p.ScanRange, p.PenRange = c.num("range"), c.num("penetrating_range")
	case CatElectrical:
		// Battle computers: initiative added to the ship, accuracy %.
		p.Initiative = c.num("initiative")
	}
	// Values any category can carry (COMPONENTS.md "Category values").
	p.Armor, p.Shield = c.num("armor"), c.num("shield")
	p.CloakPoints = c.num("cloak_points")
	if c.has("jammer_pct") {
		// COMBAT.md "Jammer %": the factor f is 100 − the part's %.
		p.Jammer = 100 - c.num("jammer_pct")
	}
	if c.has("scanner_range") {
		p.Scanner = true
		p.ScanRange, p.PenRange = c.num("scanner_range"), c.num("penetrating_range")
	}
	switch c.str("steals_cargo") {
	case "fleets":
		p.CargoScan = true
	case "fleets_and_planets":
		p.CargoScan, p.DetailedPlanetScan = true, true
	}
	p.Computer = c.num("accuracy_pct") + c.num("computer_pct")
	p.Capacitor = c.num("beam_damage_pct")
	switch v := c.num("beam_deflection_pct"); v {
	case 0:
	case 10: // COMBAT.md "Deflector %": × 90/100 per Beam Deflector.
		p.Deflector = true
	default:
		return Part{}, fmt.Errorf("%s: beam deflection %d%%", c.Name, v)
	}
	switch v := c.num("battle_speed"); {
	case v == -4: // the Energy Dampener (COMPONENTS.md "Electrical")
		p.Dampener = true
	case v >= 0:
		p.Thrust = v
	default:
		return Part{}, fmt.Errorf("%s: battle speed %d", c.Name, v)
	}
	// Half-step speed: Alien Miner as a part; the Enigma Pulsar counts
	// through Engine.EnigmaPulsar instead.
	p.HalfThrust = c.flagOrOne("battle_speed_half_steps") && c.Category != CatEngine
	p.Tachyon = c.flag("tachyon")
	p.FuelCapacity, p.CargoCapacity = c.num("fuel_capacity"), c.num("cargo_capacity")
	p.FuelPerYear = c.num("fuel_per_year")
	// Bombing (TAKEOVER.md "Bomb totals"): bomb rows, and the bomb values
	// of the Multi Contained Munition and Orbital Construction Module,
	// which count as normal bombs.
	switch {
	case c.Category == CatBomb:
		switch c.str("kind") {
		case "normal":
			p.Bomb = BombNormal
		case "smart":
			p.Bomb = BombSmart
		case "retro":
			p.Bomb = BombRetro
		default:
			return Part{}, fmt.Errorf("%s: unknown bomb kind %q", c.Name, c.str("kind"))
		}
		p.KillRate, p.InstallKill, p.MinKill = c.num("kill_tenths_pct"), c.num("installations"), c.num("min_kill")
	case c.has("bomb_kill_tenths_pct") || c.has("bomb_installations") || c.has("bomb_min_kill"):
		p.Bomb = BombNormal
		p.KillRate, p.InstallKill, p.MinKill = c.num("bomb_kill_tenths_pct"), c.num("bomb_installations"), c.num("bomb_min_kill")
	}
	p.Colonizer = c.flag("colonizes")
	return p, nil
}

func (c Component) flagOrOne(key string) bool { return c.flag(key) || c.num(key) == 1 }

// Engine converts an engine row to an Engine (COMPONENTS.md "Engine").
func (c Component) Engine() (Engine, error) {
	if c.Category != CatEngine {
		return Engine{}, fmt.Errorf("%s: not an engine", c.Name)
	}
	e := Engine{Name: c.Name, BattleWarp10: c.flag("warp10_rated"), EnigmaPulsar: c.flagOrOne("battle_speed_half_steps")}
	table, _ := c.Stats["fuel_table"].([]any)
	if len(table) != len(e.Fuel) {
		return Engine{}, fmt.Errorf("%s: fuel table has %d entries", c.Name, len(table))
	}
	for w, v := range table {
		f, _ := v.(float64)
		e.Fuel[w] = int(f)
	}
	return e, nil
}

// joatHulls have a built-in scanner for a Jack of All Trades owner
// (SCANNING.md "JOAT hulls", CONFIRMED SC-020, SC-022, SC-023).
var joatHulls = map[string]bool{"Scout": true, "Frigate": true, "Destroyer": true}

// repairBonus is f in the repair rule (COMBAT.md "Repair").
var repairBonus = map[string]int{"Fuel Transport": 25, "Super-Fuel Xport": 50}

// Hull converts a hull or starbase hull row to a Hull.
func (c Component) Hull() (Hull, error) {
	h := Hull{Name: c.Name, Mass: c.Mass, Armor: c.num("armor"), Initiative: c.num("initiative"),
		Cost: c.Cost, TechReq: c.TechReq}
	switch c.Category {
	case CatHull:
		h.FuelCapacity, h.CargoCapacity = c.num("fuel_capacity"), c.num("cargo_capacity")
		h.FuelTransport = c.flag("fuel_transport")
		h.RepairBonus = repairBonus[c.Name]
		h.JOATScanner = joatHulls[c.Name]
	case CatStarbaseHull:
		h.Starbase = true
		h.StarbaseNumber = c.Index + 1
		if c.Stats["dock_capacity"] == nil {
			h.Dock = DockUnlimited
		} else {
			h.Dock = c.num("dock_capacity")
		}
	default:
		return Hull{}, fmt.Errorf("%s: not a hull", c.Name)
	}
	slots, _ := c.Stats["slots"].([]any)
	for i, s := range slots {
		m, _ := s.(map[string]any)
		kinds, _ := m["kinds"].([]any)
		mx, _ := m["max"].(float64)
		hs := HullSlot{Max: int(mx)}
		for _, k := range kinds {
			ks, _ := k.(string)
			pk, err := slotKinds(ks)
			if err != nil {
				return Hull{}, fmt.Errorf("%s slot %d: %w", c.Name, i, err)
			}
			hs.Kinds = append(hs.Kinds, pk...)
		}
		h.Slots = append(h.Slots, hs)
	}
	return h, nil
}

// PlanetScanners returns the planetary scanners, in table order, for
// Game.PlanetScanners.
func (c *Catalog) PlanetScanners() []PlanetScanner {
	var out []PlanetScanner
	for _, comp := range c.Components {
		if comp.Category == CatPlanetary && comp.str("kind") == "scanner" {
			out = append(out, PlanetScanner{Name: comp.Name, Range: comp.num("range"),
				Penetrating: comp.num("penetrating_range") > 0, TechReq: comp.TechReq})
		}
	}
	return out
}

// Defenses returns the planetary defense types, in table order, for
// Game.Defenses.
func (c *Catalog) Defenses() []DefenseType {
	var out []DefenseType
	for _, comp := range c.Components {
		if comp.Category == CatPlanetary && comp.str("kind") == "defense" {
			out = append(out, DefenseType{Name: comp.Name, Coverage: comp.num("coverage_tenths_pct"), TechReq: comp.TechReq})
		}
	}
	return out
}

// prtCodes and lrtHas give a race's traits in the table's abbreviations.
var prtCodes = map[PRT]string{
	PRTHyperExpansion: "HE", PRTSuperStealth: "SS", PRTWarMonger: "WM", PRTClaimAdjuster: "CA",
	PRTInnerStrength: "IS", PRTSpaceDemolition: "SD", PRTPacketPhysics: "PP",
	PRTInterstellarTraveler: "IT", PRTAlternateReality: "AR", PRTJackOfAllTrades: "JOAT",
}

func lrtHas(l LRTs, code string) (bool, error) {
	switch code {
	case "IFE":
		return l.ImprovedFuelEfficiency, nil
	case "TT":
		return l.TotalTerraforming, nil
	case "ARM":
		return l.AdvancedRemoteMining, nil
	case "ISB":
		return l.ImprovedStarbases, nil
	case "GR":
		return l.GeneralizedResearch, nil
	case "UR":
		return l.UltimateRecycling, nil
	case "MA":
		return l.MineralAlchemy, nil
	case "NRSE":
		return l.NoRamScoopEngines, nil
	case "CE":
		return l.CheapEngines, nil
	case "OBRM":
		return l.OnlyBasicRemoteMining, nil
	case "NAS":
		return l.NoAdvancedScanners, nil
	case "LSP":
		return l.LowStartingPopulation, nil
	case "BET":
		return l.BleedingEdgeTech, nil
	case "RS":
		return l.RegeneratingShields, nil
	}
	return false, fmt.Errorf("unknown lesser racial trait %q", code)
}

// Allowed reports whether a race may ever build the component
// (COMPONENTS.md "Who can build what", CONFIRMED CS-001). ownsTrader says
// whether the player owns it, for a Mystery Trader item.
func (c Component) Allowed(race Race, ownsTrader bool) (bool, error) {
	if c.MysteryTrader && !ownsTrader {
		return false, nil
	}
	prt := prtCodes[race.PRT]
	if len(c.Restriction.PRTOnly) > 0 && !contains(c.Restriction.PRTOnly, prt) {
		return false, nil
	}
	if contains(c.Restriction.PRTNot, prt) {
		return false, nil
	}
	for _, code := range c.Restriction.LRTRequired {
		ok, err := lrtHas(race.LRT, code)
		if err != nil || !ok {
			return false, err
		}
	}
	for _, code := range c.Restriction.LRTForbidden {
		ok, err := lrtHas(race.LRT, code)
		if err != nil || ok {
			return false, err
		}
	}
	return true, nil
}

// Buildable reports whether a race at these tech levels may build the
// component now: it is allowed and every level meets the requirement.
func (c Component) Buildable(race Race, levels [NumFields]int, ownsTrader bool) (bool, error) {
	ok, err := c.Allowed(race, ownsTrader)
	if !ok || err != nil {
		return false, err
	}
	for f := range NumFields {
		if levels[f] < c.TechReq[f] {
			return false, nil
		}
	}
	return true, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// OwnerCost is one unit's cost for an owner (COMPONENTS.md "Cost for an
// owner", CONFIRMED CS-001).
func (c Component) OwnerCost(race Race, levels [NumFields]int) Cost {
	return itemCost(c.Cost, c.TechReq, c.Kind(), race, levels)
}

// SlotFill puts Count of the part named Part into hull slot Slot.
type SlotFill struct {
	Slot  int
	Part  string
	Count int
}

// NewDesign builds a design from the table: a hull and its filled slots.
// Each slot holds one part type, of a kind the slot accepts, at most its
// Max; a ship hull's engine slot must hold exactly Max engines
// (COMPONENTS.md "Hull"). Mass is the hull's plus every part's; fuel and
// cargo capacity are the hull's plus those of its parts.
func (c *Catalog) NewDesign(name, hull string, fills []SlotFill) (Design, error) {
	return c.newDesign(name, hull, fills, true)
}

// newDesign is NewDesign; with kinds false a part of a kind its slot does
// not take is kept. Some oracle states were written directly with such
// designs (CB-009, CB-010: a Beam Deflector in a shield-or-armor slot);
// the parity harness loads those.
//
// ASSUMPTION K10: the original keeps and uses such a part. TAKEOVER.md
// "Design parts dropped when the year is generated" says parity is "keep
// and use" for parts beyond the owner's tech or race; extending it to a
// part in a slot of the wrong kind is inferred, supported by CB-010
// passing with the part kept.
func (c *Catalog) newDesign(name, hull string, fills []SlotFill, kinds bool) (Design, error) {
	hc, ok := c.Lookup(hull)
	if !ok {
		return Design{}, fmt.Errorf("design %q: no hull %q", name, hull)
	}
	h, err := hc.Hull()
	if err != nil {
		return Design{}, fmt.Errorf("design %q: %w", name, err)
	}
	d := Design{Name: name, Hull: h, Mass: h.Mass, FuelCapacity: h.FuelCapacity, CargoCapacity: h.CargoCapacity}
	used := map[int]bool{}
	for _, f := range fills {
		if f.Slot < 0 || f.Slot >= len(h.Slots) {
			return Design{}, fmt.Errorf("design %q: %s has no slot %d", name, hull, f.Slot)
		}
		if used[f.Slot] {
			return Design{}, fmt.Errorf("design %q: slot %d filled twice", name, f.Slot)
		}
		used[f.Slot] = true
		pc, ok := c.Lookup(f.Part)
		if !ok {
			return Design{}, fmt.Errorf("design %q: no part %q", name, f.Part)
		}
		p, err := pc.Part()
		if err != nil {
			return Design{}, fmt.Errorf("design %q: %w", name, err)
		}
		hs := h.Slots[f.Slot]
		if kinds && !kindIn(p.Kind, hs.Kinds) {
			return Design{}, fmt.Errorf("design %q: slot %d does not take %s", name, f.Slot, f.Part)
		}
		if f.Count < 1 || f.Count > hs.Max {
			return Design{}, fmt.Errorf("design %q: slot %d holds 1..%d, not %d", name, f.Slot, hs.Max, f.Count)
		}
		if p.Kind == PartEngine {
			if d.Engines > 0 {
				return Design{}, fmt.Errorf("design %q: more than one engine slot", name)
			}
			e, err := pc.Engine()
			if err != nil {
				return Design{}, fmt.Errorf("design %q: %w", name, err)
			}
			d.Engine, d.Engines = e, f.Count
		}
		d.Mass += f.Count * p.Mass
		d.FuelCapacity += f.Count * p.FuelCapacity
		d.CargoCapacity += f.Count * p.CargoCapacity
		d.Slots = append(d.Slots, Slot{Part: p, Count: f.Count})
		d.SlotPos = append(d.SlotPos, f.Slot)
	}
	if !h.Starbase && len(h.Slots) > 0 && d.Engines != h.Slots[0].Max {
		return Design{}, fmt.Errorf("design %q: %s needs %d engines", name, hull, h.Slots[0].Max)
	}
	return d, nil
}

func kindIn(k PartKind, ks []PartKind) bool {
	for _, v := range ks {
		if v == k {
			return true
		}
	}
	return false
}
