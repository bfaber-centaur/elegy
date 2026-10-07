package engine

import "testing"

// pgDeposits is PG002, 2407..2411: concentration and stored fraction for
// ironium, boranium and germanium, and surface minerals.
var pgDeposits = []struct {
	year    int
	conc    [3]int
	frac    [3]int
	surface [3]int
}{
	{2407, [3]int{30, 113, 84}, [3]int{242, 86, 157}, [3]int{550, 713, 545}},
	{2408, [3]int{30, 113, 84}, [3]int{240, 61, 143}, [3]int{553, 724, 553}},
	{2409, [3]int{30, 113, 84}, [3]int{238, 36, 129}, [3]int{556, 735, 562}},
	{2410, [3]int{30, 113, 84}, [3]int{236, 12, 114}, [3]int{559, 747, 570}},
	{2411, [3]int{30, 112, 84}, [3]int{234, 243, 100}, [3]int{562, 758, 579}},
}

func TestConfirmedMiningPG(t *testing.T) {
	for i := 0; i+1 < len(pgDeposits); i++ {
		from, to := pgDeposits[i], pgDeposits[i+1]
		for m := range NumMinerals {
			d := Deposit{Concentration: from.conc[m], Fraction: from.frac[m]}
			gain, out := MineYear(d, 10, 10, true, highRand{})
			if out.Concentration != to.conc[m] || out.Fraction != to.frac[m] {
				t.Errorf("%d mineral %d: deposit %+v, want conc %d frac %d", from.year, m, out, to.conc[m], to.frac[m])
			}
			// The +1 is random; the observed gain is the floor or one more.
			if delta := to.surface[m] - from.surface[m]; delta != gain && delta != gain+1 {
				t.Errorf("%d mineral %d: surface delta %d, want %d or %d", from.year, m, delta, gain, gain+1)
			}
		}
	}
}

func TestPredictionMiningRandomPlusOne(t *testing.T) {
	// Boranium 113·10 = 1130: gain 11, +1 when rand(100) < 30.
	d := Deposit{Concentration: 113, Fraction: 86}
	if gain, _ := MineYear(d, 10, 10, true, &seqRand{draws: []int{29}}); gain != 12 {
		t.Errorf("draw 29: gain %d, want 12", gain)
	}
	if gain, _ := MineYear(d, 10, 10, true, &seqRand{draws: []int{30}}); gain != 11 {
		t.Errorf("draw 30: gain %d, want 11", gain)
	}
	// No draw when the remainder is 0.
	r := &seqRand{draws: []int{0}}
	MineYear(Deposit{Concentration: 100, Fraction: 0}, 10, 10, false, r)
	if len(r.draws) != 1 {
		t.Error("a draw was taken for a zero remainder")
	}
}

func TestPredictionMiningHomeworldFloor(t *testing.T) {
	d := Deposit{Concentration: 20, Fraction: 0}
	if gain, _ := MineYear(d, 10, 10, true, highRand{}); gain != 3 {
		t.Errorf("homeworld gain %d, want 3 (floor 30)", gain)
	}
	if gain, _ := MineYear(d, 10, 10, false, highRand{}); gain != 2 {
		t.Errorf("non-homeworld gain %d, want 2", gain)
	}
}
