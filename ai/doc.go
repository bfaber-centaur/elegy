// Package ai holds Elegy's computer-player planners.
//
// The specification is stars-elegy docs/AI.md (the shared core) and the
// personality files under docs/ai/. Elegy implements only the three
// personalities the project approved for faithful implementation
// (AI.md "Project policy"): Robotoid (HE), Rototill (CA) and Cybertron
// (PP). Turindrone, Automitron and Macinti are reference behavior and are
// not implemented here.
//
// A planner reads one player's view of the game and returns that player's
// orders for the year, as ordinary engine orders (AI.md §1: a computer
// player acts only through the orders a human could give). It never reads
// the true game state or another player's hidden state. The planners do
// not decide when they run: running them before the year, in player
// order, on the game's random stream (AI.md §1 "Random numbers") is the
// game loop's job.
//
// Per-player state is clean (AI.md §1 "State leaking between computer
// players"): every planner reads empty design slots as empty and the
// armada parameters as 0 unless it set them itself. The original's
// cross-player leaks are INTENTIONALLY DIFFERENT by project decision; a
// legacy switch for them (legacy_ai_state_leak, off by default) is not
// implemented yet. The planners keep no memory between years (AI.md §1,
// CONFIRMED AI-10).
//
// Each rule cites its spec section and evidence label. Rules where the
// spec leaves a choice to Elegy are marked ASSUMPTION An and listed in
// docs/AI-STATUS.md.
package ai
