package regress

import (
	"errors"

	"github.com/bfaber-centaur/elegy/engine"
)

// yearChecks are the checks every generated year must pass beyond
// game.Check, which game.Advance already runs: the engine's structural
// invariants (engine.CheckYear).
func yearChecks() []Check {
	return []Check{{Name: "engine.CheckYear", Check: checkYear}}
}

// checkYear is engine.CheckYear's problems.
func checkYear(prev engine.Game, r engine.TurnResult) []string {
	err := engine.CheckYear(prev, r)
	var ie *engine.InvariantError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ie):
		return ie.Problems
	}
	return []string{err.Error()}
}
