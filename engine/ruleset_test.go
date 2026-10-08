package engine

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// The Elegy ruleset is the behaviour every game had before rulesets: its
// engine switches equal the constants the engine still reads. (The
// objects and newgame packages check theirs.)
func TestElegyRulesMatchEngineSwitches(t *testing.T) {
	l := ElegyRules().Legacy
	for name, c := range map[string][2]bool{
		"FuelWrap":            {l.FuelWrap, legacyFuelWrap},
		"Colocation":          {l.Colocation, legacyColocation},
		"DropScan":            {l.DropScan, legacyDropScan},
		"StarbaseArmedClass":  {l.StarbaseArmedClass, legacyStarbaseArmedClass},
		"ObserverTechMask":    {l.ObserverTechMask, legacyObserverTechMask},
		"Plan0Recipient":      {l.Plan0Recipient, legacyPlan0},
		"CometAxes":           {l.CometAxes, legacyCometAxes},
		"MergeDilution":       {l.MergeDilution, legacyMergeDilution},
		"MergeOverflow":       {l.MergeOverflow, legacyMergeOverflow},
		"KeepUnentitledParts": {l.KeepUnentitledParts, legacyKeepUnentitledParts},
	} {
		if c[0] != c[1] {
			t.Errorf("ElegyRules %s = %v, engine switch %v", name, c[0], c[1])
		}
	}
}

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
	if _, err := GenerateTurn(pgHomeworld(), nil, highRand{}); !errors.Is(err, ErrNoRuleset) {
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
