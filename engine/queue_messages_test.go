package engine

import (
	"reflect"
	"testing"
)

// TestNoQueueMessage checks the no-queue message (MESSAGES.md 0x03f at
// stars-elegy d89b8a3, CONFIRMED): every year, an owned planet with no
// production queue is told so, once, and keeps all its resources for
// research; a planet with a queue is not.
func TestNoQueueMessage(t *testing.T) {
	for _, c := range []struct {
		name  string
		queue []QueueItem
		has   bool
		want  []Event
	}{
		{"no queue", nil, false, []Event{{Kind: EventNoQueue, Player: 0, Planet: 7, Fleet: -1}}},
		{"a queue", []QueueItem{{Kind: ItemFactory, Count: 1}}, true, nil},
	} {
		g := pgHomeworld()
		g.Planets[0].HasQueue, g.Planets[0].Queue = c.has, c.queue
		r, err := GenerateTurn(withRules(g), nil, highRand{})
		if err != nil {
			t.Fatal(err)
		}
		var got []Event
		for _, ev := range r.Events {
			if ev.Kind == EventNoQueue {
				got = append(got, ev)
			}
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}
