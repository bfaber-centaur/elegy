package ai

import (
	"fmt"

	"github.com/bfaber-centaur/elegy/engine"
)

// Result is a planner's turn: the player's orders, the designs it stored
// (with the picture and creation year the design order lacks), and
// diagnostics for rules it reached but could not express as orders.
type Result struct {
	Orders  []engine.Order
	Designs []NewDesign
	// Unsupported names each step the spec calls for that Elegy cannot
	// order yet (for example a scrap), so a game shows the gap instead of
	// hiding it.
	Unsupported []string
}

func (r *Result) unsupported(format string, args ...any) {
	r.Unsupported = append(r.Unsupported, fmt.Sprintf(format, args...))
}
