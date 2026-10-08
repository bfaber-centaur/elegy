package engine

import "testing"

// salvageObjects gives the engine the objects' salvage load and room
// rules (OBJECTS.md "Salvage", "Loading"); the objects package tests them
// and its adapter (objects.SalvageLoad, SalvageRoom).
type salvageObjects struct{ SpaceObjects }

func (salvageObjects) SalvageLoad(sv Salvage, k, want int) int { return min(want, sv.Minerals[k]) }

func (salvageObjects) SalvageRoom(sv Salvage) int {
	return max(0, 10*sv.Steps-sv.Minerals[0]-sv.Minerals[1]-sv.Minerals[2])
}

// salvageLab is a Medium Freighter (210 kT) of player 0 at (500,500),
// away from any planet, on a salvage object of player 1 holding sv.
func salvageLab(t *testing.T, sv Salvage, task Task) *tkLab {
	l := newTKLab(t, 3)
	medium := l.design("Medium Freighter", SlotFill{0, "Quick Jump 5", 1})
	at := Point{500, 500}
	sv.Pos, sv.Owner = at, 1
	l.g.Objects = salvageObjects{}
	l.g.Salvage = []Salvage{sv}
	l.g.Fleets = []Fleet{{ID: 1, Owner: 0, Pos: at, Stacks: []Stack{{Design: medium, Count: 1}}, Task: task}}
	return l
}

// Loading from salvage (OBJECTS.md "Salvage", "Loading"; ASSUMPTION T5,
// T6): the usual amounts capped by what it holds, whoever owns it;
// minerals in order from the space left; a colonist load waits.
func TestLoadFromSalvage(t *testing.T) {
	task := Task{Kind: TaskTransport}
	task.Transport[0] = Transport{Action: LoadAll}
	task.Transport[1] = Transport{Action: LoadExactly, Amount: 150}
	task.Transport[2] = Transport{Action: LoadAll}
	task.Transport[CargoColonists] = Transport{Action: LoadAll}
	l := salvageLab(t, Salvage{Minerals: Minerals{120, 500, 40}, Steps: 66}, task)
	l.g.loadPass(false)
	f, sv := l.g.Fleets[0], l.g.Salvage[0]
	// Ironium: all 120 (the hold has 210). Boranium: 150 asked, 90 of
	// space left, so 90 and 60 still to load. Germanium: no space left.
	if f.Cargo.Minerals != (Minerals{120, 90, 0}) || sv.Minerals != (Minerals{0, 410, 40}) {
		t.Errorf("fleet %v, salvage %v; want [120 90 0] and [0 410 40]", f.Cargo.Minerals, sv.Minerals)
	}
	want := [NumCargo]Transport{1: {Action: LoadExactly, Amount: 60}, CargoColonists: {Action: LoadAll}}
	if f.Task.Kind != TaskTransport || f.Task.Transport != want {
		t.Errorf("task %+v, want %+v", f.Task, want)
	}
}

// Unloading into salvage (OBJECTS.md "Salvage", "Loading": only up to
// the stored size; ASSUMPTION T7: the rest stays aboard and colonists
// are refused). The salvage holds 37 kT in 6 steps: 23 kT of room.
func TestUnloadIntoSalvage(t *testing.T) {
	task := Task{Kind: TaskTransport}
	task.Transport[0] = Transport{Action: UnloadAll}
	task.Transport[2] = Transport{Action: UnloadExactly, Amount: 15}
	task.Transport[CargoColonists] = Transport{Action: UnloadAll}
	l := salvageLab(t, Salvage{Minerals: Minerals{30, 0, 7}, Steps: 6}, task)
	l.g.Fleets[0].Cargo = Cargo{Minerals: Minerals{10, 5, 40}, Colonists: 20}
	_, ev := l.g.unloadPhase([]bool{})
	f, sv := l.g.Fleets[0], l.g.Salvage[0]
	// Ironium 10 of 23; germanium 15 asked, 13 of room left.
	if sv.Minerals != (Minerals{40, 0, 20}) || f.Cargo.Minerals != (Minerals{0, 5, 27}) || f.Cargo.Colonists != 20 {
		t.Errorf("salvage %v, fleet %+v; want [40 0 20] and [0 5 27] with 20 colonists", sv.Minerals, f.Cargo)
	}
	if sv.Steps != 6 || sv.Fresh {
		t.Errorf("salvage %+v: an unload refreshes nothing", sv)
	}
	if len(ev) != 1 || ev[0].Kind != EventDropRefused || ev[0].Count != 20 {
		t.Errorf("events %+v, want one refused drop of 20", ev)
	}
	if f.Task.Kind != TaskNone {
		t.Errorf("task %+v, want none", f.Task)
	}
}
