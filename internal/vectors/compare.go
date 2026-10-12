package vectors

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
)

// Field is one compared field of an expectation: a field of a fleet,
// planet or player expectation's equals, checked on its own, or a whole
// expectation of any other kind (Field "").
type Field struct {
	Case    string `json:"case"`
	Year    int    `json:"year"`
	Kind    string `json:"kind"`
	Subject string `json:"subject,omitempty"`
	Field   string `json:"field,omitempty"`
	// Status is pass, differ (a mismatch), sample (a sample expectation
	// that missed: one stream's random outcome, skipped as the harness
	// skips it) or skip (not compared; Why says why).
	Status string `json:"status"`
	// Sample marks a field whose mismatch is only a missed sample: a
	// random outcome compared without the oracle's draws.
	Sample bool   `json:"sample,omitempty"`
	Why    string `json:"why,omitempty"`
}

// Comparison is one run of a vector: each case's result, as Run gives
// it, and every compared field.
type Comparison struct {
	ID      string   `json:"id"`
	Years   int      `json:"years"`
	Variant int      `json:"variant"`
	Cases   []Result `json:"cases"`
	Fields  []Field  `json:"fields"`
}

// splitKinds are the expectation kinds whose equals fields are checked
// one at a time: each field of a fleet, planet or player is compared on
// its own (fleetEquals, planetEquals, playerEquals).
func splitKinds() map[string]bool {
	return map[string]bool{"fleet": true, "planet": true, "player": true}
}

// Compare loads v, generates its years with Elegy's own seeding for the
// seed variant and compares every case, as Run does, and every field of every
// expectation. A case's result is the harness's; the fields say which
// parts of it matched.
func Compare(v *Vector, variant int) Comparison {
	run := pvRun{seed: variant}
	out := Comparison{ID: v.ID, Years: v.Years, Variant: variant, Cases: []Result{}, Fields: []Field{}}
	l, games, genErr, stop := generate(v, run)
	out.Cases = checkCases(v, run, l, games, genErr, stop)
	for _, c := range v.Cases {
		for _, e := range c.Expect {
			out.Fields = append(out.Fields, l.fields(run, c, e, games, genErr, stop)...)
		}
	}
	return out
}

// fields compares one expectation field by field, as checkCases does
// for the whole expectation.
func (l *pvLoaded) fields(run pvRun, c pvCase, e pvExpect, games []engine.Game, genErr error, stop string) []Field {
	base := Field{Case: c.ID, Year: e.Year, Kind: e.Kind, Subject: subject(e), Sample: run.sample(e)}
	one := func(f Field, msg string) Field {
		switch {
		case strings.HasPrefix(msg, "skip: "):
			f.Status, f.Why = "skip", strings.TrimPrefix(msg, "skip: ")
		case msg != "" && run.sample(e):
			f.Status, f.Why = "sample", msg
		case msg != "":
			f.Status, f.Why = "differ", msg
		default:
			f.Status = "pass"
		}
		return f
	}
	switch {
	case !run.checks(e):
		return []Field{one(base, "skip: stream "+e.Stream)}
	case e.Year >= len(games) && stop != "":
		return []Field{one(base, "skip: "+stop)}
	case e.Year >= len(games):
		return []Field{one(base, fmt.Sprintf("year %d not generated: %v", e.Year, genErr))}
	}
	g := &games[e.Year]
	var eq map[string]json.RawMessage
	if !splitKinds()[e.Kind] || json.Unmarshal(e.Equals, &eq) != nil || len(eq) == 0 {
		return []Field{one(base, l.pvCheck(g, e))}
	}
	var out []Field
	for _, k := range pvKeys(eq) {
		ek := e
		ek.Equals, _ = json.Marshal(map[string]json.RawMessage{k: eq[k]})
		f := base
		f.Field = k
		out = append(out, one(f, l.pvCheck(g, ek)))
	}
	return out
}

// subject names an expectation's object from its ids.
func subject(e pvExpect) string {
	var parts []string
	for _, p := range []struct {
		name string
		v    *int
	}{{"owner", e.Owner}, {"id", e.ID}, {"planet", e.Planet}, {"slot", e.Slot}, {"x", e.X}, {"y", e.Y}, {"viewer", e.Viewer}} {
		if p.v != nil {
			parts = append(parts, fmt.Sprintf("%s %d", p.name, *p.v))
		}
	}
	if e.Screen != "" || e.Field != "" {
		parts = append(parts, strings.TrimSpace(e.Screen+" "+e.Field))
	}
	return strings.Join(parts, ", ")
}
