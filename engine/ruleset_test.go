package engine

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// The faithful ruleset has every legacy switch on.
func TestFaithfulRulesAllOn(t *testing.T) {
	v := reflect.ValueOf(FaithfulRules().Legacy)
	for i := range v.NumField() {
		if !v.Field(i).Bool() {
			t.Errorf("FaithfulRules %s is off", v.Type().Field(i).Name)
		}
	}
}

func TestRulesetJSONRoundTrip(t *testing.T) {
	custom := ElegyRules()
	custom.ID, custom.Version = "elegy-test-variant", 3
	custom.Legacy.FuelWrap = false
	for _, r := range append(Rulesets(), custom) {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var back Ruleset
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back != r {
			t.Errorf("%s: round trip %+v, want %+v", b, back, r)
		}
		if err := back.Validate(); err != nil {
			t.Errorf("%s: %v", r.ID, err)
		}
	}
}

func TestRulesetValidate(t *testing.T) {
	if err := (Ruleset{}).Validate(); !errors.Is(err, ErrNoRuleset) {
		t.Errorf("zero ruleset: %v", err)
	}
	mislabelled := ElegyRules()
	mislabelled.Legacy.FieldLimit511 = true
	noVersion := ElegyRules()
	noVersion.ID = "custom"
	noVersion.Version = 0
	unreleased := FaithfulRules()
	unreleased.Version = 99
	for name, r := range map[string]Ruleset{
		"no ID":           {Version: 1, Legacy: Legacy{FuelWrap: true}},
		"no version":      noVersion,
		"mislabelled":     mislabelled,
		"unknown version": unreleased,
	} {
		if err := r.Validate(); !errors.Is(err, ErrRuleset) {
			t.Errorf("%s: %v", name, err)
		}
	}
	type key struct {
		id      string
		version int
	}
	ids := map[key]bool{}
	for _, r := range Rulesets() {
		key := key{r.ID, r.Version}
		if ids[key] {
			t.Errorf("built-in %s version %d listed twice", r.ID, r.Version)
		}
		ids[key] = true
	}
}

// docs/RULESET.md is the inventory: it names every switch by its saved
// name.
func TestRulesetInventoryDocumentsEverySwitch(t *testing.T) {
	doc, err := os.ReadFile("../docs/RULESET.md")
	if err != nil {
		t.Fatal(err)
	}
	typ := reflect.TypeOf(Legacy{})
	for i := range typ.NumField() {
		tag := typ.Field(i).Tag.Get("json")
		if !strings.Contains(string(doc), "`"+tag+"`") {
			t.Errorf("docs/RULESET.md does not list %s", tag)
		}
	}
}

func TestGenerateTurnKeepsRules(t *testing.T) {
	none := pgHomeworld()
	none.Rules = Ruleset{}
	if _, err := GenerateTurn(none, nil, highRand{}); !errors.Is(err, ErrNoRuleset) {
		t.Errorf("no ruleset: %v", err)
	}
	g := pgHomeworld()
	g.Rules = FaithfulRules()
	r, err := GenerateTurn(g, nil, highRand{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Game.Rules != FaithfulRules() {
		t.Errorf("rules after the turn %+v", r.Game.Rules)
	}
}

// Two games with different rulesets run side by side in one process, at
// the same time, and each follows its own. FM-105's under-engined Large
// Freighter with 200 mg at warp 5 moves 7 ly under FuelWrap (Elegy); a
// custom ruleset without it computes the exact fuel term, about 26 times
// larger, and the fleet gets less far.
func TestRulesetsCoexist(t *testing.T) {
	game := func(rules Ruleset) Game {
		d := Design{Name: "Large Freighter", Hull: Hull{Name: "Large Freighter", Slots: []HullSlot{{Kinds: []PartKind{PartEngine}, Max: 2}}},
			Mass: 134, Engines: 1, FuelCapacity: 2600}
		d.Engine.Fuel = [11]int{0, 0, 0, 0, 0, 0, 100, 100, 100, 100, 100}
		return Game{Rules: rules, Year: 2400, Players: make([]Player, 1), Designs: []Design{d},
			Fleets: []Fleet{{ID: 1, Stacks: []Stack{{Design: 0, Count: 1}}, Fuel: 200,
				Waypoints: []Waypoint{{Pos: Point{1000, 0}, Warp: 5}}}}}
	}
	exact := ElegyRules()
	exact.ID = "elegy-exact-fuel"
	exact.Legacy.FuelWrap = false
	if err := exact.Validate(); err != nil {
		t.Fatal(err)
	}
	rulesets := []Ruleset{ElegyRules(), exact}

	// Each ruleset's result run alone first.
	alone := make([]Game, len(rulesets))
	for i, r := range rulesets {
		res, err := GenerateTurn(game(r), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		alone[i] = res.Game
	}
	if got := alone[0].Fleets[0].Pos; got != (Point{7, 0}) {
		t.Fatalf("Elegy ruleset: fleet at %v, want (7,0)", got)
	}
	if a, b := alone[0].Fleets[0].Pos, alone[1].Fleets[0].Pos; a == b {
		t.Fatalf("both rulesets put the fleet at %v", a)
	}

	// Then many interleaved at once: every result matches its own
	// ruleset's, and every game keeps its ruleset.
	const runs = 50
	got := make([]Game, runs*len(rulesets))
	errs := make([]error, len(got))
	var wg sync.WaitGroup
	for k := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := GenerateTurn(game(rulesets[k%len(rulesets)]), nil, highRand{})
			got[k], errs[k] = res.Game, err
		}()
	}
	wg.Wait()
	for k, g := range got {
		i := k % len(rulesets)
		if errs[k] != nil {
			t.Fatalf("%s: %v", rulesets[i].ID, errs[k])
		}
		if g.Rules != rulesets[i] || g.Fleets[0].Pos != alone[i].Fleets[0].Pos || g.Fleets[0].Fuel != alone[i].Fleets[0].Fuel {
			t.Errorf("run %d (%s): rules %s, fleet at %v with %d mg; alone at %v with %d mg", k, rulesets[i].ID,
				g.Rules.ID, g.Fleets[0].Pos, g.Fleets[0].Fuel, alone[i].Fleets[0].Pos, alone[i].Fleets[0].Fuel)
		}
	}
}
