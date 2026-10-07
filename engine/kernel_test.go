package engine

// Test naming follows stars-elegy docs/KERNEL.md statuses:
//
//   TestConfirmed*  — CONFIRMED vectors (white-box reading agrees with
//                     oracle observations). Ground truth: must never be
//                     changed to make code pass.
//   TestPrediction* — BINARY-ONLY vectors, worked from the rule with no
//                     oracle observation yet. They pin the implementation
//                     to the spec; an oracle result may later flip one.
//
// `go test -run Confirmed ./...` runs only the ground truth.

// pgRace is the PG-001..003 / PQ-001 oracle race: growth 10%, 1000
// colonists per resource, 10 factories produce 10 resources, factory 10
// resources + 4 kT germanium, mine 5 resources, mine output 10. Factories
// and mines operated are not given by KERNEL.md; 10 per 10,000 colonists is
// consistent with every PQ-001 cap.
func pgRace() Race {
	r := Race{
		GrowthRate:           10,
		ColonistsPerResource: 1000,
		FactoryOutput:        10,
		FactoriesOperated:    10,
		MinesOperated:        10,
		MineOutput:           10,
		FactoryCost:          Cost{Resources: 10, Minerals: Minerals{0, 0, 4}},
		MineCost:             Cost{Resources: 5},
	}
	for i := range r.Env {
		r.Env[i] = EnvRange{Center: 50, Low: 15, High: 85}
	}
	return r
}

// pgColony is a hab-100, max-10,000 planet of pgRace.
func pgColony() Colony {
	r := pgRace()
	return Colony{Race: r, Hab: 100, MaxPop: MaxPopulation(r, 100, 0)}
}

// seqRand returns a fixed sequence of draws (then zeros).
type seqRand struct{ draws []int }

func (s *seqRand) Intn(n int) int {
	if len(s.draws) == 0 {
		return 0
	}
	d := s.draws[0]
	s.draws = s.draws[1:]
	return d % n
}

// highRand never grants a probabilistic +1 (every draw is n−1).
type highRand struct{}

func (highRand) Intn(n int) int { return n - 1 }
