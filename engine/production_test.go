package engine

import (
	"reflect"
	"testing"
)

// PQ-001 (stars-elegy PARITY.md "Production Queues"): one PG001 planet,
// hab 100, max 10,000, excessPop 80. Common start: mines 0, factories 0,
// defenses 10, budget 0%, leftover-only off, minerals 500/500/500.
type pqCase struct {
	name                       string
	pop                        int
	mines, factories, defenses int
	minerals                   Minerals
	budget                     int
	leftoverOnly               bool
	queue                      []QueueItem

	wantMines, wantFactories, wantDefenses int
	wantMinerals                           Minerals
	wantQueue                              []QueueItem
	wantResearch                           int
	wantCompleted                          bool
	wantEvents                             []EventKind // must appear (in order)

	race func(*Race) // race changes from the PG race, if any
}

func q(kind ItemKind, count, pct int) QueueItem {
	return QueueItem{Kind: kind, Count: count, Percent: pct}
}

func runPQ(t *testing.T, tc pqCase) {
	t.Helper()
	c := pgColony()
	if tc.race != nil {
		tc.race(&c.Race)
	}
	p := &Planet{
		Owner:        0,
		Population:   tc.pop,
		Mines:        tc.mines,
		Factories:    tc.factories,
		Defenses:     tc.defenses,
		Surface:      tc.minerals,
		HasQueue:     true,
		Queue:        append([]QueueItem(nil), tc.queue...),
		LeftoverOnly: tc.leftoverOnly,
	}
	grown, _ := GrowPopulation(tc.pop, 80, c.MaxPop, 10, 100)
	research, events := RunProduction(p, ProductionInput{
		Colony:         c,
		Resources:      c.Resources(tc.pop, tc.factories),
		GrownPop:       grown,
		ResearchBudget: tc.budget,
		LeftoverOnly:   tc.leftoverOnly,
	})
	if p.Mines != tc.wantMines || p.Factories != tc.wantFactories || p.Defenses != tc.wantDefenses {
		t.Errorf("mines/factories/defenses %d/%d/%d, want %d/%d/%d",
			p.Mines, p.Factories, p.Defenses, tc.wantMines, tc.wantFactories, tc.wantDefenses)
	}
	if p.Surface != tc.wantMinerals {
		t.Errorf("minerals %v, want %v", p.Surface, tc.wantMinerals)
	}
	if len(p.Queue) != 0 || len(tc.wantQueue) != 0 {
		if !reflect.DeepEqual(p.Queue, tc.wantQueue) {
			t.Errorf("queue %v, want %v", p.Queue, tc.wantQueue)
		}
	}
	if p.HasQueue != (len(tc.wantQueue) > 0) {
		t.Errorf("HasQueue %v with queue %v", p.HasQueue, p.Queue)
	}
	if research != tc.wantResearch {
		t.Errorf("research %d, want %d", research, tc.wantResearch)
	}
	completed := false
	var kinds []EventKind
	for _, e := range events {
		if e.Kind == EventQueueCompleted {
			completed = true
		}
		kinds = append(kinds, e.Kind)
	}
	if completed != tc.wantCompleted {
		t.Errorf("completed message %v, want %v (events %v)", completed, tc.wantCompleted, events)
	}
	i := 0
	for _, k := range kinds {
		if i < len(tc.wantEvents) && k == tc.wantEvents[i] {
			i++
		}
	}
	if i != len(tc.wantEvents) {
		t.Errorf("events %v, want subsequence %v", kinds, tc.wantEvents)
	}
}

func TestConfirmedProductionPQ001(t *testing.T) {
	m500 := Minerals{500, 500, 500}
	cases := []pqCase{
		{name: "C01", pop: 1050, defenses: 10, minerals: m500,
			queue:         []QueueItem{q(ItemFactory, 20, 0)},
			wantFactories: 10, wantDefenses: 10, wantMinerals: Minerals{500, 500, 458},
			wantQueue: []QueueItem{q(ItemFactory, 10, 59)}, wantResearch: 0},
		{name: "C02", pop: 2000, defenses: 10, minerals: Minerals{500, 500, 10},
			queue:         []QueueItem{q(ItemFactory, 5, 0), q(ItemMine, 5, 0)},
			wantFactories: 2, wantDefenses: 10, wantMinerals: Minerals{500, 500, 0},
			wantQueue: []QueueItem{q(ItemFactory, 3, 74), q(ItemMine, 5, 0)}, wantResearch: 173},
		{name: "C03", pop: 2000, defenses: 10, minerals: Minerals{500, 500, 2},
			queue:     []QueueItem{q(ItemAutoFactories, 100, 0), q(ItemMine, 5, 0)},
			wantMines: 5, wantDefenses: 10, wantMinerals: Minerals{500, 500, 2},
			wantQueue: []QueueItem{q(ItemAutoFactories, 100, 0)}, wantResearch: 175},
		{name: "C04", pop: 230, defenses: 10, minerals: m500,
			queue:     []QueueItem{q(ItemAutoMines, 100, 0)},
			wantMines: 4, wantDefenses: 10, wantMinerals: m500,
			wantQueue: []QueueItem{q(ItemMine, 1, 79), q(ItemAutoMines, 100, 0)}, wantResearch: 0},
		{name: "C05", pop: 2500, defenses: 10, minerals: Minerals{100, 100, 100},
			queue:        []QueueItem{q(ItemAutoAlchemy, 1, 0)},
			wantDefenses: 10, wantMinerals: Minerals{102, 102, 102},
			wantQueue: []QueueItem{q(ItemMineralAlchemy, 1, 50), q(ItemAutoAlchemy, 1, 0)}, wantResearch: 0,
			wantEvents: []EventKind{EventAlchemy}},
		{name: "C06", pop: 4000, defenses: 10, minerals: Minerals{100, 100, 1},
			queue:     []QueueItem{q(ItemAutoAlchemy, 1, 0), q(ItemFactory, 1, 0), q(ItemMine, 2, 0)},
			wantMines: 2, wantFactories: 1, wantDefenses: 10, wantMinerals: Minerals{103, 103, 0},
			wantResearch: 80, wantCompleted: true,
			wantEvents: []EventKind{EventAlchemy, EventBuilt, EventBuilt, EventQueueCompleted}},
		{name: "C07", pop: 2500, defenses: 10, minerals: Minerals{100, 100, 1},
			queue:        []QueueItem{q(ItemAutoAlchemy, 1, 0), q(ItemFactory, 1, 0), q(ItemMine, 2, 0)},
			wantDefenses: 10, wantMinerals: Minerals{102, 102, 2},
			wantQueue:    []QueueItem{q(ItemMineralAlchemy, 1, 46), q(ItemAutoAlchemy, 1, 0), q(ItemFactory, 1, 49), q(ItemMine, 2, 0)},
			wantResearch: 0, wantEvents: []EventKind{EventAlchemy}},
		{name: "C08a", pop: 1070, defenses: 10, minerals: m500, budget: 15,
			queue:         []QueueItem{q(ItemFactory, 20, 0)},
			wantFactories: 9, wantDefenses: 10, wantMinerals: Minerals{500, 500, 464},
			wantQueue: []QueueItem{q(ItemFactory, 11, 19)}, wantResearch: 16},
		{name: "C08b", pop: 1070, defenses: 10, minerals: m500, budget: 15, leftoverOnly: true,
			queue:         []QueueItem{q(ItemFactory, 20, 0)},
			wantFactories: 10, wantDefenses: 10, wantMinerals: Minerals{500, 500, 457},
			wantQueue: []QueueItem{q(ItemFactory, 10, 79)}, wantResearch: 0},
		{name: "C09", pop: 500, mines: 48, factories: 50, defenses: 10, minerals: m500,
			queue:     []QueueItem{q(ItemAutoMines, 100, 0), q(ItemAutoFactories, 3, 0)},
			wantMines: 55, wantFactories: 53, wantDefenses: 10, wantMinerals: Minerals{500, 500, 488},
			wantQueue: []QueueItem{q(ItemAutoMines, 100, 0), q(ItemAutoFactories, 3, 0)}, wantResearch: 35,
			wantCompleted: true},
		{name: "C10", pop: 1000, factories: 995, defenses: 10, minerals: m500,
			queue:         []QueueItem{q(ItemFactory, 10, 0)},
			wantFactories: 1000, wantDefenses: 10, wantMinerals: Minerals{500, 500, 480},
			wantResearch: 150, wantCompleted: true,
			wantEvents: []EventKind{EventOrderClipped, EventBuilt, EventQueueCompleted}},
		{name: "C11", pop: 50, defenses: 10, minerals: Minerals{100, 100, 2},
			queue:         []QueueItem{q(ItemFactory, 1, 59), q(ItemMine, 10, 0)},
			wantFactories: 1, wantDefenses: 10, wantMinerals: Minerals{100, 100, 0},
			wantQueue: []QueueItem{q(ItemMine, 10, 19)}, wantResearch: 0},
		{name: "C12", pop: 1000, defenses: 10, minerals: Minerals{3, 2, 100},
			queue:        []QueueItem{q(ItemDefenses, 5, 0), q(ItemMine, 2, 0)},
			wantDefenses: 10, wantMinerals: Minerals{1, 0, 98},
			wantQueue: []QueueItem{q(ItemDefenses, 5, 59), q(ItemMine, 2, 0)}, wantResearch: 92},
		{name: "C13", pop: 1010, defenses: 40, minerals: m500,
			queue:        []QueueItem{q(ItemAutoDefenses, 100, 0)},
			wantDefenses: 45, wantMinerals: Minerals{475, 475, 475},
			wantQueue: []QueueItem{q(ItemAutoDefenses, 100, 0)}, wantResearch: 26, wantCompleted: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runPQ(t, tc) })
	}
}

// pqGame is a one-planet game for the multi-year PQ-001 cases. The planet
// is not a homeworld and has no mineral concentration, so nothing is mined.
func pqGame(pop int, queue []QueueItem) Game {
	race := pgRace()
	p := Planet{
		ID: 7, Owner: 0, Env: [3]int{50, 50, 50},
		Population: pop, GrowthCarry: 80, Defenses: 10,
		Surface: Minerals{500, 500, 500}, HasQueue: true, Queue: queue,
	}
	return Game{Rules: ElegyRules(), Year: 2407, Players: []Player{{Race: race}}, Planets: []Planet{p}}
}

func TestConfirmedProductionPQ001TwoYears(t *testing.T) {
	t.Run("C01 year 2", func(t *testing.T) {
		g := pqGame(1050, []QueueItem{q(ItemFactory, 20, 0)})
		for range 2 {
			r, err := GenerateTurn(withRules(g), nil, highRand{})
			if err != nil {
				t.Fatal(err)
			}
			g = r.Game
		}
		p := g.Planets[0]
		if p.Factories != 20 || p.Surface[Germanium] != 420 || p.HasQueue {
			t.Errorf("factories %d Ge %d queue %v %v", p.Factories, p.Surface[Germanium], p.HasQueue, p.Queue)
		}
		// 2409 research 30; energy accumulates it (2408 sent 0).
		if got := g.Players[0].Research.Accumulated[Energy]; got != 30 {
			t.Errorf("research %d, want 30", got)
		}
	})
	t.Run("C14", func(t *testing.T) {
		g := pqGame(230, []QueueItem{q(ItemAutoMines, 100, 0)})
		r, _ := GenerateTurn(withRules(g), nil, highRand{})
		r, _ = GenerateTurn(withRules(r.Game), nil, highRand{})
		p := r.Game.Planets[0]
		want := []QueueItem{q(ItemMine, 1, 79), q(ItemAutoMines, 100, 0)}
		if p.Mines != 9 || !reflect.DeepEqual(p.Queue, want) {
			t.Errorf("mines %d queue %v, want 9 %v", p.Mines, p.Queue, want)
		}
		if got := r.Game.Players[0].Research.Accumulated[Energy]; got != 0 {
			t.Errorf("research %d, want 0", got)
		}
		built := 0
		for _, e := range r.Events {
			if e.Kind == EventBuilt && e.Item == ItemMine {
				built += e.Count
			}
		}
		if built != 5 {
			t.Errorf("2409 built %d mines, want 5", built)
		}
	})
}

func TestPredictionProductionEdgeCases(t *testing.T) {
	c := pgColony()
	in := ProductionInput{Colony: c, Resources: 100, GrownPop: 1100}

	empty := &Planet{HasQueue: true}
	if r, ev := RunProduction(empty, in); r != 0 || len(ev) != 0 {
		t.Errorf("empty queue: research %d events %v, want 0 and none", r, ev)
	}

	none := &Planet{}
	if r, _ := RunProduction(none, in); r != 100 {
		t.Errorf("no queue: research %d, want 100", r)
	}

	zero := &Planet{HasQueue: true, Queue: []QueueItem{q(ItemMine, 1, 0)}}
	in0 := in
	in0.Resources = 0
	if r, ev := RunProduction(zero, in0); r != 0 || len(ev) != 0 || zero.Queue[0].Percent != 0 {
		t.Errorf("zero resources: research %d events %v queue %v", r, ev, zero.Queue)
	}
}
