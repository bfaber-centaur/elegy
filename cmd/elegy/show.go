package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/game"
	"github.com/bfaber-centaur/elegy/newgame"
)

// showSections are show's sections, in the order "all" prints them.
var showSections = []string{"summary", "planets", "galaxy", "fleets", "designs", "research", "messages"}

// cmdShow prints a player's report as text, one section after another.
// It reads only the player's Report, so it shows nothing the player does
// not know, plus the game's public settings (slower tech).
func cmdShow(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	path := fs.String("game", "game.json", "save file")
	player := fs.Int("player", 0, "player index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sections := fs.Args()
	if len(sections) == 0 {
		sections = []string{"summary"}
	}
	if len(sections) == 1 && sections[0] == "all" {
		sections = showSections
	}
	for _, s := range sections {
		if _, err := lookup("section", s, append(showSections, "all")); err != nil {
			return err
		}
	}
	g, err := readGame(*path)
	if err != nil {
		return err
	}
	r, err := g.Report(*player)
	if err != nil {
		return err
	}
	return show(out, newReportView(r, g.State.SlowerTech), sections)
}

func show(out io.Writer, v reportView, sections []string) error {
	for i, s := range sections {
		if i > 0 {
			fmt.Fprintln(out)
		}
		switch strings.ToLower(s) {
		case "summary":
			v.summary(out)
		case "planets":
			v.planets(out)
		case "galaxy":
			v.galaxy(out)
		case "fleets":
			v.fleets(out)
		case "designs":
			v.designs(out)
		case "research":
			v.research(out)
		case "messages":
			v.messages(out)
		default:
			return errors.New("show: section " + s)
		}
	}
	return nil
}

// reportView is a Report with the lookups the text needs.
type reportView struct {
	r          game.Report
	slowerTech bool
	names      map[int]string        // planet name by id
	planetAt   map[engine.Point]int  // planet id by position
	own        map[int]engine.Design // the player's own designs by design index
	known      map[int]game.KnownDesign
}

func newReportView(r game.Report, slowerTech bool) reportView {
	v := reportView{r: r, slowerTech: slowerTech, names: map[int]string{}, planetAt: map[engine.Point]int{},
		own: map[int]engine.Design{}, known: map[int]game.KnownDesign{}}
	for _, p := range r.Universe {
		v.names[p.ID] = newgame.PlanetName(p.NameIndex)
		v.planetAt[p.Pos] = p.ID
	}
	for _, d := range r.Designs {
		v.own[d.Index] = d.Design
	}
	for _, d := range r.KnownDesigns {
		v.known[d.Index] = d
	}
	return v
}

func (v reportView) planetName(id int) string {
	if n, ok := v.names[id]; ok {
		return fmt.Sprintf("%s (#%d)", n, id)
	}
	return fmt.Sprintf("planet #%d", id)
}

// where names a position: the planet there, or the point in space.
func (v reportView) where(p engine.Point) string {
	if id, ok := v.planetAt[p]; ok {
		return "at " + v.planetName(id)
	}
	return fmt.Sprintf("in space at (%d, %d)", p.X, p.Y)
}

// designName names a design index: the player's own design, or another
// player's as the player knows it.
func (v reportView) designName(i int) string {
	if d, ok := v.own[i]; ok {
		return d.Name
	}
	if d, ok := v.known[i]; ok {
		if d.Design != nil {
			return d.Design.Name
		}
		return "unknown " + d.Hull
	}
	return fmt.Sprintf("design #%d", i)
}

// slotDesign names the player's design in a ship or starbase slot.
func (v reportView) slotDesign(starbase bool, slot int) string {
	for _, d := range v.r.Designs {
		if d.Slot.Starbase == starbase && d.Slot.Slot == slot {
			return d.Design.Name
		}
	}
	return fmt.Sprintf("empty slot %d", slot)
}

// fleetName is the fleet's name, or its first design's name and its
// number, as the original's client names an unnamed fleet.
func (v reportView) fleetName(f engine.Fleet) string {
	if f.Name != "" {
		return f.Name
	}
	if len(f.Stacks) == 0 {
		return fmt.Sprintf("Fleet #%d", f.Number)
	}
	return fmt.Sprintf("%s #%d", v.designName(f.Stacks[0].Design), f.Number)
}

func (v reportView) homeworld() (engine.Point, bool) {
	if !v.r.Self.HasHomeworld {
		return engine.Point{}, false
	}
	for _, p := range v.r.Universe {
		if p.ID == v.r.Self.Homeworld {
			return p.Pos, true
		}
	}
	return engine.Point{}, false
}

func dist(a, b engine.Point) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}

func minerals(m engine.Minerals) string {
	return fmt.Sprintf("ironium %d, boranium %d, germanium %d", m[engine.Ironium], m[engine.Boranium], m[engine.Germanium])
}

func (v reportView) itemText(it engine.QueueItem) string {
	s := itemName(it.Kind)
	switch it.Kind {
	case engine.ItemShip:
		s += " " + v.slotDesign(false, it.Slot)
	case engine.ItemStarbase:
		s += " " + v.slotDesign(true, it.Slot)
	}
	s += fmt.Sprintf(" ×%d", it.Count)
	if it.Percent > 0 {
		s += fmt.Sprintf(" (%d%% done)", it.Percent)
	}
	return s
}

func (v reportView) summary(out io.Writer) {
	r := v.r
	fmt.Fprintf(out, "Game %d, year %d, player %d, ruleset %s v%d\n", r.GameID, r.Year, r.Player, r.Rules.ID, r.Rules.Version)
	if r.Self.Dead {
		fmt.Fprintln(out, "This player has been eliminated.")
	}
	pop := 0
	for _, p := range r.Planets {
		pop += p.Population
	}
	fmt.Fprintf(out, "Planets: %d, population %d\n", len(r.Planets), pop*100)
	if r.Self.HasHomeworld {
		fmt.Fprintf(out, "Homeworld: %s\n", v.planetName(r.Self.Homeworld))
	}
	ships := 0
	for _, f := range r.Fleets {
		for _, s := range f.Stacks {
			ships += s.Count
		}
	}
	fmt.Fprintf(out, "Fleets: %d, ships %d\n", len(r.Fleets), ships)
	rs := r.Self.Research
	fmt.Fprintf(out, "Research: %d%% of resources to %s (level %d), then %s\n", r.Self.ResearchBudget, fieldName(rs.Current), rs.Levels[rs.Current], nextName(rs.Next))
	rejected := 0
	for _, o := range r.Orders {
		if o.Message != "" {
			rejected++
		}
	}
	fmt.Fprintf(out, "Messages: %d; last year's orders: %d, %d rejected\n", len(r.Events), len(r.Orders), rejected)
}

func (v reportView) planets(out io.Writer) {
	r := v.r
	fmt.Fprintf(out, "Planets (%d)\n", len(r.Planets))
	for _, p := range r.Planets {
		head := fmt.Sprintf("%s at (%d, %d)", v.planetName(p.ID), p.Pos.X, p.Pos.Y)
		if p.Homeworld {
			head += ", homeworld"
		}
		fmt.Fprintln(out, head)
		fmt.Fprintf(out, "  population %d, mines %d, factories %d, defenses %d\n", p.Population*100, p.Mines, p.Factories, p.Defenses)
		if p.HasStarbase {
			fmt.Fprintf(out, "  starbase %s\n", v.designName(p.StarbaseDesign))
		}
		fmt.Fprintf(out, "  surface minerals: %s kT\n", minerals(p.Surface))
		fmt.Fprintf(out, "  concentrations: %d, %d, %d\n", p.Deposits[engine.Ironium].Concentration, p.Deposits[engine.Boranium].Concentration, p.Deposits[engine.Germanium].Concentration)
		fmt.Fprintf(out, "  environment %d/%d/%d, habitability %d%%\n", p.Env[0], p.Env[1], p.Env[2], engine.Habitability(r.Self.Race, p.Env))
		if len(p.Queue) == 0 {
			fmt.Fprintln(out, "  queue: empty")
		} else {
			fmt.Fprintln(out, "  queue:")
			for i, it := range p.Queue {
				fmt.Fprintf(out, "    %d. %s\n", i+1, v.itemText(it))
			}
		}
		if p.LeftoverOnly {
			fmt.Fprintln(out, "  only leftover resources go to research")
		}
		if p.HasRoute {
			fmt.Fprintf(out, "  route to %s\n", v.planetName(p.RouteTo))
		}
	}
}

// galaxy lists every planet the player does not own, nearest to its
// homeworld first, with what its latest report showed. Every player knows
// where every planet is (game.Report.Universe).
func (v reportView) galaxy(out io.Writer) {
	r := v.r
	home, hasHome := v.homeworld()
	own := map[int]bool{}
	for _, p := range r.Planets {
		own[p.ID] = true
	}
	seen := map[int]game.PlanetRecord{}
	for _, h := range r.History {
		seen[h.Report.Planet] = h
	}
	var others []game.UniversePlanet
	for _, p := range r.Universe {
		if !own[p.ID] {
			others = append(others, p)
		}
	}
	if hasHome {
		sort.SliceStable(others, func(a, b int) bool {
			return dist(others[a].Pos, home) < dist(others[b].Pos, home)
		})
	}
	fmt.Fprintf(out, "Other planets (%d)\n", len(others))
	for _, u := range others {
		line := fmt.Sprintf("%s at (%d, %d)", v.planetName(u.ID), u.Pos.X, u.Pos.Y)
		if hasHome {
			line += fmt.Sprintf(", %.1f ly", dist(u.Pos, home))
		}
		h, ok := seen[u.ID]
		if !ok {
			fmt.Fprintln(out, line+", never seen")
			continue
		}
		p := h.Report
		line += fmt.Sprintf(", seen %d", h.Year)
		if p.Owner == engine.NoOwner {
			line += ", unowned"
		} else {
			line += fmt.Sprintf(", owned by player %d", p.Owner)
		}
		if p.Level >= engine.ReportNormal {
			line += fmt.Sprintf(", habitability %d%%, concentrations %d/%d/%d", engine.Habitability(r.Self.Race, p.Env),
				p.Concentrations[engine.Ironium], p.Concentrations[engine.Boranium], p.Concentrations[engine.Germanium])
		}
		fmt.Fprintln(out, line)
	}
	if len(r.View.Fleets) > 0 {
		fmt.Fprintf(out, "Other players' fleets seen (%d)\n", len(r.View.Fleets))
		for _, f := range r.View.Fleets {
			var ships []string
			for _, s := range f.Stacks {
				ships = append(ships, fmt.Sprintf("%d %s", s.Count, v.designName(s.Design)))
			}
			fmt.Fprintf(out, "player %d fleet %s: %s\n", f.Owner, v.where(f.Pos), strings.Join(ships, ", "))
		}
	}
}

func (v reportView) fleets(out io.Writer) {
	r := v.r
	fmt.Fprintf(out, "Fleets (%d)\n", len(r.Fleets))
	for _, f := range r.Fleets {
		fmt.Fprintf(out, "#%d %s, %s\n", f.ID, v.fleetName(f), v.where(f.Pos))
		var ships []string
		for _, s := range f.Stacks {
			ships = append(ships, fmt.Sprintf("%d %s", s.Count, v.designName(s.Design)))
		}
		if len(ships) == 0 {
			ships = []string{"none"}
		}
		fmt.Fprintf(out, "  ships: %s\n", strings.Join(ships, ", "))
		fmt.Fprintf(out, "  fuel %d mg; cargo %s kT, colonists %d\n", f.Fuel, minerals(f.Cargo.Minerals), f.Cargo.Colonists*100)
		if f.Task.Kind != engine.TaskNone {
			fmt.Fprintf(out, "  task here: %s\n", taskName(f.Task.Kind))
		}
		from := f.Pos
		for i, w := range f.Waypoints {
			to := fmt.Sprintf("(%d, %d)", w.Pos.X, w.Pos.Y)
			switch w.Target {
			case engine.TargetPlanet:
				to = v.planetName(w.ID)
			case engine.TargetFleet:
				to = fmt.Sprintf("fleet #%d", w.ID)
			}
			line := fmt.Sprintf("  waypoint %d: %s, %.1f ly, warp %d", i+1, to, dist(from, w.Pos), w.Warp)
			if w.Task.Kind != engine.TaskNone {
				line += ", then " + taskName(w.Task.Kind)
			}
			fmt.Fprintln(out, line)
			from = w.Pos
		}
		if f.Repeat {
			fmt.Fprintln(out, "  repeats its orders")
		}
	}
}

func (v reportView) designs(out io.Writer) {
	r := v.r
	fmt.Fprintf(out, "Designs (%d)\n", len(r.Designs))
	for _, d := range r.Designs {
		kind := "ship"
		if d.Slot.Starbase {
			kind = "starbase"
		}
		x := d.Design
		fmt.Fprintf(out, "%s slot %d: %s (%s hull), %d kT", kind, d.Slot.Slot, x.Name, x.Hull.Name, x.Mass)
		if x.Engines > 0 {
			fmt.Fprintf(out, ", %d %s", x.Engines, x.Engine.Name)
		}
		if x.FuelCapacity > 0 {
			fmt.Fprintf(out, ", fuel %d mg", x.FuelCapacity)
		}
		if x.CargoCapacity > 0 {
			fmt.Fprintf(out, ", cargo %d kT", x.CargoCapacity)
		}
		fmt.Fprintln(out)
		// Parts are counted by name, in the order they first appear.
		var names []string
		count := map[string]int{}
		for _, s := range x.Parts() {
			if count[s.Part.Name] == 0 {
				names = append(names, s.Part.Name)
			}
			count[s.Part.Name] += s.Count
		}
		var parts []string
		for _, n := range names {
			parts = append(parts, fmt.Sprintf("%d %s", count[n], n))
		}
		if len(parts) > 0 {
			fmt.Fprintf(out, "  parts: %s\n", strings.Join(parts, ", "))
		}
	}
	if len(r.KnownDesigns) > 0 {
		fmt.Fprintf(out, "Other players' designs known (%d)\n", len(r.KnownDesigns))
		for _, d := range r.KnownDesigns {
			fmt.Fprintf(out, "design #%d: %s, %s hull, %d kT, seen %d\n", d.Index, v.designName(d.Index), d.Hull, d.Mass, d.Year)
		}
	}
}

// research prints each field's level and progress. The cost of the next
// level is ResearchLevelCost; under slower tech the progress is stored at
// half scale (engine.AddResearch), so both are shown at that scale.
func (v reportView) research(out io.Writer) {
	self := v.r.Self
	rs := self.Research
	sum := 0
	for _, l := range rs.Levels {
		sum += l
	}
	fmt.Fprintf(out, "Research: %d%% of resources, current field %s, then %s\n", self.ResearchBudget, fieldName(rs.Current), nextName(rs.Next))
	for f := range engine.NumFields {
		line := fmt.Sprintf("  %-13s level %2d", fieldName(f), rs.Levels[f])
		maxLevel := rs.MaxLevel
		if maxLevel == 0 {
			maxLevel = engine.MaxTechLevel
		}
		if rs.Levels[f] < maxLevel {
			cost := engine.ResearchLevelCost(rs.Levels[f]+1, sum, self.Race.ResearchCosts[f], v.slowerTech)
			acc := rs.Accumulated[f]
			if v.slowerTech {
				acc *= 2
			}
			line += fmt.Sprintf(", %d of %d toward level %d", acc, cost, rs.Levels[f]+1)
		}
		if f == rs.Current {
			line += "  <- researching"
		}
		fmt.Fprintln(out, line)
	}
}

func (v reportView) messages(out io.Writer) {
	r := v.r
	fmt.Fprintf(out, "Messages for %d (%d)\n", r.Year-1, len(r.Events))
	for _, e := range r.Events {
		line := eventName(e.Kind)
		if e.Kind == engine.EventBuilt {
			line += " " + itemName(e.Item)
		}
		if e.Planet >= 0 {
			line += ": " + v.planetName(e.Planet)
		}
		if e.Fleet >= 0 {
			line += fmt.Sprintf(", fleet #%d", e.Fleet)
		}
		if e.Count != 0 {
			line += fmt.Sprintf(", count %d", e.Count)
		}
		fmt.Fprintln(out, line)
	}
	for _, o := range r.Orders {
		if o.Message == "" {
			continue
		}
		if o.Index < 0 {
			fmt.Fprintf(out, "order file refused: %s\n", o.Message)
		} else {
			fmt.Fprintf(out, "order %d rejected: %s\n", o.Index, o.Message)
		}
	}
}
