package engine

import "testing"

func TestConfirmedHomeworldMarkAfterCapture(t *testing.T) {
	// TAKEOVER.md "The homeworld mark after capture" (CONFIRMED T-41):
	// the marks are set from the players' records, whoever owns the
	// planet. Player 0's homeworld 3 was captured by player 1 and keeps
	// its mark; player 0's other planet 5 gets none; player 1 holds two
	// marked planets (3 and its own 9); dead player 2's unowned 11 keeps
	// its mark; player 3 has no record.
	g := Game{
		Players: []Player{
			{HasHomeworld: true, Homeworld: 3},
			{HasHomeworld: true, Homeworld: 9},
			{HasHomeworld: true, Homeworld: 11, Dead: true},
			{},
		},
		Planets: []Planet{
			{ID: 3, Owner: 1},
			{ID: 5, Owner: 0, Homeworld: true},
			{ID: 9, Owner: 1},
			{ID: 11, Owner: NoOwner, Homeworld: true},
			{ID: 0, Owner: 3, Homeworld: true},
		},
	}
	g.remarkHomeworlds()
	want := map[int]bool{3: true, 5: false, 9: true, 11: true, 0: false}
	for _, p := range g.Planets {
		if p.Homeworld != want[p.ID] {
			t.Errorf("planet %d homeworld %v, want %v", p.ID, p.Homeworld, want[p.ID])
		}
	}
}

func TestMeasuredWURSG2HomeworldRemarkInTurn(t *testing.T) {
	// WU-RSG2 (MEASURED, stars-elegy 52af513): the setup cleared the
	// homeworld's mark, the player's record named it, and after the year
	// the mark was back. Without a record the mark stays cleared.
	for _, recorded := range []bool{true, false} {
		g := pgHomeworld()
		g.Planets[0].Homeworld = false
		g.Players[0].HasHomeworld = recorded
		r, err := GenerateTurn(g, nil, &seqRand{})
		if err != nil {
			t.Fatal(err)
		}
		if got := r.Game.Planets[0].Homeworld; got != recorded {
			t.Errorf("recorded %v: homeworld mark %v after the year", recorded, got)
		}
	}
}
