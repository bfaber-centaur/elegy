package oraclerng

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestCycles(t *testing.T) {
	for _, c := range []struct {
		label string
		want  int
		ok    bool
	}{
		{"cycles 20000", 20000, true},
		{"cycles 880", 880, true},
		{"cycles", 0, false},
		{"cycles 0", 0, false},
		{"cycles -5", 0, false},
		{"cycles 20000x", 0, false},
		{"20000", 0, false},
		{"", 0, false},
	} {
		got, ok := Cycles(c.label)
		if got != c.want || ok != c.ok {
			t.Errorf("Cycles(%q) = %d, %v; want %d, %v", c.label, got, ok, c.want, c.ok)
		}
	}
}

// Every tick in the table is trunc(k·54.925) for an integer k (ORACLE.md
// "Which stream a cycles value reaches"), and k is close to 70000/cycles
// above cycles 1200, where the mapping is monotonic.
func TestTicksAreWholeTimerPeriods(t *testing.T) {
	for cycles, tick := range ticks {
		k := int(math.Round(float64(tick) / 54.925))
		if int(float64(k)*54.925) != tick {
			t.Errorf("cycles %d: tick %d is not trunc(k·54.925)", cycles, tick)
		}
		if cycles >= 1200 {
			want := 70000.0 / float64(cycles)
			if math.Abs(float64(k)-want) > 1+want*0.1 {
				t.Errorf("cycles %d: k %d, about 70000/cycles = %.1f expected", cycles, k, want)
			}
		}
	}
}

func TestStreamTick(t *testing.T) {
	if tick, ok := StreamTick("cycles 35000"); !ok || tick != 109 {
		t.Errorf("cycles 35000: %d, %v; want 109", tick, ok)
	}
	if tick, ok := StreamTick("cycles 20000"); !ok || tick != 164 {
		t.Errorf("cycles 20000: %d, %v; want 164", tick, ok)
	}
	if _, ok := StreamTick("cycles 25000"); ok {
		t.Error("cycles 25000 has no measured tick")
	}
	if _, ok := StreamTick("seed 1"); ok {
		t.Error("a label without cycles has no tick")
	}
}

type countRand struct{ next int }

func (c *countRand) Intn(n int) int { c.next++; return c.next % n }

func TestRecorder(t *testing.T) {
	r := &Recorder{R: &countRand{}}
	got := []int{r.Intn(2), r.Intn(1), r.Intn(100)}
	if !reflect.DeepEqual(got, []int{1, 0, 3}) || !reflect.DeepEqual(r.Bounds, []int{2, 1, 100}) {
		t.Errorf("draws %v bounds %v", got, r.Bounds)
	}
}

func TestStartRecorded(t *testing.T) {
	for _, c := range []struct {
		s    Start
		want bool
	}{
		{Start{}, false},
		{Start{State: json.RawMessage("null")}, false},
		{Start{Tick: 0, HasTick: true}, true},
		{Start{State: json.RawMessage(`"0a0b"`)}, true},
	} {
		if got := c.s.Recorded(); got != c.want {
			t.Errorf("%+v: Recorded %v, want %v", c.s, got, c.want)
		}
	}
}
