package engine

import (
	"errors"
	"fmt"
)

// Ruleset is a game's rules configuration: which compatibility behaviours
// the game follows. A game carries its ruleset (Game.Rules) from creation
// to its last year, and every rule that differs between rulesets reads it
// from there. There is no process-wide compatibility setting, so games
// with different rulesets can run side by side in one process.
//
// A Ruleset is plain data: it holds no pointers, slices, maps or funcs, so
// a copy is fully independent and == compares two rulesets exactly. The
// engine never changes a game's ruleset; GenerateTurn passes it on to the
// next year unchanged. It round-trips exactly through encoding/json.
//
// ID and Version name the ruleset; Legacy is its settings. The built-in
// rulesets (Rulesets) are fixed: a ruleset that claims a built-in ID and
// Version must carry exactly that ruleset's settings (Validate). Any other
// ID names a custom ruleset, whose settings are whatever it carries.
//
// docs/RULESET.md is the inventory of every switch.
type Ruleset struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Legacy  Legacy `json:"legacy"`
}

// Legacy is a ruleset's legacy-compatibility switches. Each one, when
// true, reproduces a behaviour of the original game that Elegy would
// otherwise do differently, usually a LEGACY BUG. The comment on each
// names the rule's place in the stars-elegy specification and its
// evidence; the comment at the switch's use site gives the detail.
//
// The zero value has every switch off. That is not the Elegy ruleset:
// most of the Elegy ruleset's switches are on (ElegyRules).
type Legacy struct {
	// engine

	// FuelWrap: the fuel term f·L·M keeps its low 32 bits and divides as
	// a signed 32-bit number (KERNEL.md "Designs without a full set of
	// engines", CONFIRMED FM-105). engine/movement.go.
	FuelWrap bool `json:"fuel_wrap"`
	// Colocation: every scanning test passes at distance 0 (SCANNING.md
	// "Co-location", CONFIRMED SC-002, SC-014). engine/scanning.go.
	Colocation bool `json:"colocation"`
	// DropScan: several players' drops pick the winner by the scan in
	// player order (TAKEOVER.md "Several players dropping at once",
	// CONFIRMED T-32). engine/takeover.go.
	DropScan bool `json:"drop_scan"`
	// StarbaseArmedClass: a starbase is always an armed target (COMBAT.md
	// "Starbases in battle", CONFIRMED CB-011..013 S4/S5).
	// engine/combat_token.go.
	StarbaseArmedClass bool `json:"starbase_armed_class"`
	// ObserverTechMask: a player outside a battle tests its number
	// against the observer bitmask (COMBAT.md "Tech from battle",
	// CONFIRMED CB-031-obs, CB-037). engine/combat.go.
	ObserverTechMask bool `json:"observer_tech_mask"`
	// Plan0Recipient: a starbase's plan 0 against "everyone" or a named
	// player is written to another player (COMBAT.md "LEGACY BUG: plan 0
	// ...", CONFIRMED CB-011..013, CB-022, CB-035). engine/combat_who.go.
	Plan0Recipient bool `json:"plan0_recipient"`
	// CometAxes: the comet strike message names the axes of a shuffled
	// order, not the ones that moved (KERNEL.md "Comet strike", CONFIRMED
	// KX-004 S2). Only the message differs. engine/randomevents.go.
	CometAxes bool `json:"comet_axes"`
	// MergeDilution: merging two damaged stacks divides the damage by
	// all the merged ships (ORDERS.md "Merge", CONFIRMED FO-01..07).
	// engine/fleetops.go.
	MergeDilution bool `json:"merge_dilution"`
	// MergeOverflow: the Merge with Fleet task applies no ship-count cap,
	// and a stack pushed to 32768 or more leaves the fleet with no ships
	// (ORDERS.md "Merge", CONFIRMED FO). engine/fleetops.go.
	MergeOverflow bool `json:"merge_overflow"`
	// KeepUnentitledParts: reading a design drops only parts above the
	// owner's tech, so a part the owner is not entitled to is kept
	// (ORDERS.md "Design legality (Mystery Trader parts kept)",
	// CONFIRMED). engine/fleetops.go.
	KeepUnentitledParts bool `json:"keep_unentitled_parts"`
	// ZeroMaxPopulationStop: an Alternate Reality planet with population,
	// habitability ≥ 0 and no starbase (maximum population 0) stops the
	// year, where the original stops with an integer divide by zero
	// (KERNEL.md "Maximum population", LEGACY BUG, CONFIRMED KX-001 Z1):
	// GenerateTurn returns a *ZeroMaxPopulationError and generates
	// nothing. Off, the year is generated, and the planet's growth takes
	// the crowding permille at its limit, the 12% overcrowding cap
	// (INTENTIONALLY DIFFERENT). engine/turn.go.
	ZeroMaxPopulationStop bool `json:"zero_max_population_stop"`

	// objects

	// FieldLimit511: a player's field number 511 is given only when no
	// other space object sorts after that player's minefields
	// (OBJECTS.md "Laying", MEASURED MF-13). objects/minefields.go.
	FieldLimit511 bool `json:"field_limit_511"`
	// EmptyFleetSalvage: a fleet with no minerals that loses ships to a
	// mine hit drops rand(10) kT of each mineral (OBJECTS.md "Hits on
	// moving fleets", LEGACY BUG candidate, MEASURED OB-024).
	// objects/minefields.go.
	EmptyFleetSalvage bool `json:"empty_fleet_salvage"`
	// DueNorthSouthCut: a leg with no east-west component takes the foot
	// of the perpendicular at the fleet's start (OBJECTS.md "Due-north
	// and due-south legs", MEASURED MF-15). objects/minefields.go.
	DueNorthSouthCut bool `json:"due_north_south_cut"`
	// MineSurvivorSalvage: a mine hit that destroys some ships drops
	// every mineral the survivors carry (OBJECTS.md "Cargo when ships are
	// destroyed", LEGACY BUG candidate, MEASURED MF-14).
	// objects/minefields.go.
	MineSurvivorSalvage bool `json:"mine_survivor_salvage"`
	// GateMixedFleetLoss: the original's mixed-fleet count for stargate
	// losses, which can delete the whole fleet (OBJECTS.md "Mixed
	// fleets", MEASURED GT-001 H2, GT-002, CONFIRMED GT-003 W0–W5).
	// objects/stargates.go.
	GateMixedFleetLoss bool `json:"gate_mixed_fleet_loss"`
	// TraderLastRedraw: a player whose 25th Mystery Trader part redraw
	// found an unowned part gets a ship instead (OBJECTS.md "Encounters",
	// BINARY-ONLY). objects/trader.go.
	TraderLastRedraw bool `json:"trader_last_redraw"`

	// newgame

	// SharedHomeworldMinerals: every homeworld starts with the same
	// surface minerals and planet 0's concentrations (UNIVERSE.md "Shared
	// starting minerals", CONFIRMED UG16–UG21). newgame/players.go.
	SharedHomeworldMinerals bool `json:"shared_homeworld_minerals"`
	// SecondPlanetFallback: a second planet whose 100 environment
	// redraws were used takes the homeworld's environment (UNIVERSE.md
	// "Second planet", CONFIRMED UG29, UG30). newgame/players.go.
	SecondPlanetFallback bool `json:"second_planet_fallback"`
}

// Built-in ruleset identities.
const (
	// ElegyRulesID is Elegy's own ruleset: the original's behaviour with
	// the fixes Elegy has chosen (ElegyRules).
	ElegyRulesID = "elegy"
	// FaithfulRulesID is the original's behaviour with every legacy
	// switch on (FaithfulRules).
	FaithfulRulesID = "jrc3-faithful"
)

// ElegyRules is Elegy's default ruleset, its latest version: version 2.
// Four switches are off, where Elegy has chosen its own rule over the
// original's: FieldLimit511, MergeOverflow, KeepUnentitledParts and
// ZeroMaxPopulationStop.
//
// Version 2 differs from version 1 only in ZeroMaxPopulationStop: an
// Alternate Reality planet with population and no starbase no longer
// stops the year (Bobby's decision, 2026-10-08, "Keep going").
func ElegyRules() Ruleset {
	r := elegyRulesV1()
	r.Version = 2
	r.Legacy.ZeroMaxPopulationStop = false
	return r
}

// elegyRulesV1 is the Elegy ruleset, version 1: the behaviour every game
// had before rulesets existed.
func elegyRulesV1() Ruleset {
	return Ruleset{ID: ElegyRulesID, Version: 1, Legacy: Legacy{
		FuelWrap:                true,
		Colocation:              true,
		DropScan:                true,
		StarbaseArmedClass:      true,
		ObserverTechMask:        true,
		Plan0Recipient:          true,
		CometAxes:               true,
		MergeDilution:           true,
		MergeOverflow:           false,
		KeepUnentitledParts:     false,
		ZeroMaxPopulationStop:   true,
		FieldLimit511:           false,
		EmptyFleetSalvage:       true,
		DueNorthSouthCut:        true,
		MineSurvivorSalvage:     true,
		GateMixedFleetLoss:      true,
		TraderLastRedraw:        true,
		SharedHomeworldMinerals: true,
		SecondPlanetFallback:    true,
	}}
}

// FaithfulRules is the faithful J-RC3 ruleset, version 1: every legacy
// switch on. It differs from ElegyRules only in FieldLimit511,
// MergeOverflow, KeepUnentitledParts and ZeroMaxPopulationStop. It is only
// as faithful as the switches go: behaviour Elegy has not modelled, or
// models differently without a switch, is the same in both.
func FaithfulRules() Ruleset {
	r := elegyRulesV1()
	r.ID, r.Version = FaithfulRulesID, 1
	r.Legacy.MergeOverflow = true
	r.Legacy.KeepUnentitledParts = true
	r.Legacy.FieldLimit511 = true
	return r
}

// Rulesets is every built-in ruleset, every version. A built-in ruleset's
// settings never change once released: a change is a new version, and
// the old version stays here so games saved under it still load and
// replay.
func Rulesets() []Ruleset {
	return []Ruleset{elegyRulesV1(), ElegyRules(), FaithfulRules()}
}

// ErrNoRuleset is returned for a game with no ruleset (the zero Ruleset).
var ErrNoRuleset = errors.New("engine: the game has no ruleset")

// ErrRuleset is returned for a ruleset that is not well formed.
var ErrRuleset = errors.New("engine: invalid ruleset")

// Validate reports whether r is a ruleset a game can run under: it has an
// ID and a version of at least 1, and if it claims a built-in ruleset's
// ID and version it carries exactly that ruleset's settings.
func (r Ruleset) Validate() error {
	if r == (Ruleset{}) {
		return ErrNoRuleset
	}
	if r.ID == "" {
		return fmt.Errorf("%w: no ID", ErrRuleset)
	}
	if r.Version < 1 {
		return fmt.Errorf("%w: %s version %d", ErrRuleset, r.ID, r.Version)
	}
	builtin := false
	for _, b := range Rulesets() {
		if b.ID != r.ID {
			continue
		}
		builtin = true
		if b.Version == r.Version {
			if b != r {
				return fmt.Errorf("%w: %s version %d does not carry its settings", ErrRuleset, r.ID, r.Version)
			}
			return nil
		}
	}
	if builtin {
		return fmt.Errorf("%w: %s has no version %d", ErrRuleset, r.ID, r.Version)
	}
	return nil
}
