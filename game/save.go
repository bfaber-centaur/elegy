package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"

	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/newgame"
	"github.com/bfaber-centaur/elegy/objects"
	"github.com/bfaber-centaur/elegy/races"
	"github.com/bfaber-centaur/elegy/terraform"
)

// The save format. ELEGY CHOICE: a saved game is Elegy's own versioned
// JSON document, not any of the original's file formats (stars-elegy
// ORDERS.md "Scope and vocabulary" makes the same choice for orders).
// See docs/GAME-LOOP.md.
const (
	SaveFormat  = "elegy-save"
	SaveVersion = 4
)

// saveFile is the saved document. Field order is the encoding order.
type saveFile struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	Seed    uint64 `json:"seed"`
	// NameIndex is each planet's name index, by planet index.
	NameIndex []int `json:"nameIndex"`
	// RNG is the game generator's state (newgame.Source.State).
	RNG uint64 `json:"rng"`
	// Game is the engine state with its Objects, Races and Terraform
	// left out; they are the three fields below.
	Game      engine.Game      `json:"game"`
	Objects   *objects.Space   `json:"objects"`
	Races     *races.GameRaces `json:"races"`
	Terraform bool             `json:"terraform"`
	// Levels is each player's computer-player level (Game.ComputerLevel).
	Levels []newgame.Level `json:"levels"`
	Last   savedYear       `json:"last"`
}

// savedYear is what the last year told the players.
type savedYear struct {
	Views   []engine.PlayerView `json:"views"`
	Events  []engine.Event      `json:"events"`
	Results []savedResult       `json:"results"`
	// History is each player's planet history, in planet id order.
	History [][]PlanetRecord `json:"history"`
	// Wormholes is each player's wormhole sightings, in end order.
	Wormholes [][]wormholeRecord `json:"wormholes"`
	// Designs is each player's knowledge of other players' designs, in
	// design index order.
	Designs [][]designRecord `json:"designs"`
}

type savedResult struct {
	Player int    `json:"player"`
	Index  int    `json:"index"`
	Err    string `json:"err,omitempty"`
	// Sentinel indexes orderSentinels, −1 for none.
	Sentinel int `json:"sentinel,omitempty"`
}

// ErrSave is wrapped by every error loading a save.
var ErrSave = errors.New("game: bad save")

func (g *Game) document() (saveFile, error) {
	st := g.State
	f := saveFile{
		Format:    SaveFormat,
		Version:   SaveVersion,
		Seed:      g.Seed,
		NameIndex: g.nameIndex,
		RNG:       g.rng.State(),
		Terraform: st.Terraform != nil,
		Levels:    g.levels,
		Last:      savedYear{Views: g.views, Events: g.events},
	}
	if st.Objects != nil {
		sp, ok := st.Objects.(*objects.Space)
		if !ok {
			return saveFile{}, fmt.Errorf("game: cannot save space objects of type %T", st.Objects)
		}
		f.Objects = sp
	}
	if st.Races != nil {
		r, ok := st.Races.(*races.GameRaces)
		if !ok {
			return saveFile{}, fmt.Errorf("game: cannot save a race checker of type %T", st.Races)
		}
		f.Races = r
	}
	if st.Terraform != nil {
		if _, ok := st.Terraform.(terraform.Rules); !ok {
			return saveFile{}, fmt.Errorf("game: cannot save terraform rules of type %T", st.Terraform)
		}
	}
	for p := range st.Players {
		f.Last.History = append(f.Last.History, g.history.list(p))
		f.Last.Wormholes = append(f.Last.Wormholes, g.wormholes.list(p))
		f.Last.Designs = append(f.Last.Designs, g.designs.list(p))
	}
	st.Objects, st.Races, st.Terraform = nil, nil, nil
	f.Game = st
	for _, r := range g.results {
		s := savedResult{Player: r.Player, Index: r.Index, Sentinel: -1}
		if r.Err != nil {
			s.Err = r.Err.Error()
			s.Sentinel = sentinelIndex(r.Err)
		}
		f.Last.Results = append(f.Last.Results, s)
	}
	return f, nil
}

// Encode is the game's save document: canonical, indented JSON. The same
// game always encodes to the same bytes.
func (g *Game) Encode() ([]byte, error) {
	f, err := g.document()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(f, "", " ")
}

// Save writes the game's save document to w.
func (g *Game) Save(w io.Writer) error {
	b, err := g.Encode()
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// Hash is the SHA-256 of the save document, in hex: two games with the
// same hash continue identically under the same orders.
func (g *Game) Hash() (string, error) {
	b, err := g.Encode()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Load reads a save document written by Save.
func Load(r io.Reader) (*Game, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var f saveFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSave, err)
	}
	if f.Format != SaveFormat {
		return nil, fmt.Errorf("%w: format %q, want %q", ErrSave, f.Format, SaveFormat)
	}
	if f.Version != SaveVersion {
		return nil, fmt.Errorf("%w: version %d; this build reads version %d", ErrSave, f.Version, SaveVersion)
	}
	st := f.Game
	if err := st.Rules.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSave, err)
	}
	if f.Objects != nil {
		st.Objects = f.Objects
	}
	if f.Races != nil {
		st.Races = f.Races
	}
	if f.Terraform {
		st.Terraform = terraform.Rules{}
	}
	if probs := Check(st); len(probs) > 0 {
		return nil, fmt.Errorf("%w: %d problems, first: %s", ErrSave, len(probs), probs[0])
	}
	if len(f.Last.Views) != len(st.Players) {
		return nil, fmt.Errorf("%w: %d views for %d players", ErrSave, len(f.Last.Views), len(st.Players))
	}
	if len(f.Levels) != len(st.Players) {
		return nil, fmt.Errorf("%w: %d levels for %d players", ErrSave, len(f.Levels), len(st.Players))
	}
	for p, l := range f.Levels {
		if l < newgame.Easy || l > newgame.Expert {
			return nil, fmt.Errorf("%w: player %d level %d", ErrSave, p, l)
		}
	}
	g := &Game{
		State:     st,
		Seed:      f.Seed,
		nameIndex: f.NameIndex,
		levels:    f.Levels,
		rng:       newgame.NewRand(f.RNG),
		views:     f.Last.Views,
		events:    f.Last.Events,
	}
	if len(f.Last.History) != len(st.Players) {
		return nil, fmt.Errorf("%w: planet histories for %d of %d players", ErrSave, len(f.Last.History), len(st.Players))
	}
	g.history = make(history, len(f.Last.History))
	for p, recs := range f.Last.History {
		g.history[p] = map[int]PlanetRecord{}
		for _, r := range recs {
			g.history[p][r.Report.Planet] = r
		}
	}
	if len(f.Last.Wormholes) != len(st.Players) {
		return nil, fmt.Errorf("%w: wormhole sightings for %d of %d players", ErrSave, len(f.Last.Wormholes), len(st.Players))
	}
	g.wormholes = make(wormholeHistory, len(f.Last.Wormholes))
	for p, recs := range f.Last.Wormholes {
		g.wormholes[p] = map[int]wormholeRecord{}
		for _, r := range recs {
			g.wormholes[p][r.End] = r
		}
	}
	if len(f.Last.Designs) != len(st.Players) {
		return nil, fmt.Errorf("%w: design knowledge for %d of %d players", ErrSave, len(f.Last.Designs), len(st.Players))
	}
	g.designs = make(designHistory, len(f.Last.Designs))
	for p, recs := range f.Last.Designs {
		g.designs[p] = map[int]designRecord{}
		for _, r := range recs {
			if r.Design < 0 || r.Design >= len(st.Designs) {
				return nil, fmt.Errorf("%w: player %d knows design %d of %d", ErrSave, p, r.Design, len(st.Designs))
			}
			g.designs[p][r.Design] = r
		}
	}
	for _, s := range f.Last.Results {
		r := engine.OrderResult{Player: s.Player, Index: s.Index}
		if s.Err != "" {
			r.Err = newOrderError(s.Err, s.Sentinel)
		}
		g.results = append(g.results, r)
	}
	return g, nil
}

// Diff lists where two games' save documents differ, as JSON paths with
// both values, at most max entries (all when max ≤ 0). It is the
// diagnostic for a determinism failure: the first paths name the objects
// that diverged.
func Diff(a, b *Game, max int) ([]string, error) {
	var docs [2]any
	for i, g := range []*Game{a, b} {
		enc, err := g.Encode()
		if err != nil {
			return nil, err
		}
		d := json.NewDecoder(bytes.NewReader(enc))
		d.UseNumber()
		if err := d.Decode(&docs[i]); err != nil {
			return nil, err
		}
	}
	var out []string
	diffJSON("$", docs[0], docs[1], &out, max)
	return out, nil
}

func diffJSON(path string, a, b any, out *[]string, max int) {
	if max > 0 && len(*out) >= max {
		return
	}
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			break
		}
		keys := map[string]bool{}
		for k := range x {
			keys[k] = true
		}
		for k := range y {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			diffJSON(path+"."+k, x[k], y[k], out, max)
		}
		return
	case []any:
		y, ok := b.([]any)
		if !ok {
			break
		}
		for i := range min(len(x), len(y)) {
			diffJSON(path+"["+strconv.Itoa(i)+"]", x[i], y[i], out, max)
		}
		if len(x) != len(y) {
			*out = append(*out, fmt.Sprintf("%s: length %d vs %d", path, len(x), len(y)))
		}
		return
	}
	if !reflect.DeepEqual(a, b) {
		*out = append(*out, fmt.Sprintf("%s: %s vs %s", path, short(a), short(b)))
	}
}

func short(v any) string {
	b, _ := json.Marshal(v)
	if len(b) > 80 {
		return string(b[:77]) + "..."
	}
	return string(b)
}
