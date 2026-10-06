package engine

type Game struct {
	Year int
}

// TODO: move to planet.go or similar when the model earns that split.
type Planet struct {
	Habitability int // -100..100; eventually derived from environment + faction
	Population   int64
}

type Faction struct {
	GrowthRate int // percent per year
}

// Stubs: keep these small until real rules/orders require shape.
type PlayerOrders struct{}
type Ruleset interface{}
type JRC3Rules struct{}

func Jrc3() Ruleset {
	return JRC3Rules{}
}

type TurnResult struct {
	Game Game
}

func GenerateTurn(
	game Game,
	orders []PlayerOrders,
	rules Ruleset,
) (TurnResult, error) {
	game.Year++

	return TurnResult{
		Game: game,
	}, nil
}

// PopulationCapacity currently covers the standard positive-habitability
// capacity rule only. Racial modifiers belong in explicit rules once promoted
// from parity evidence.
func PopulationCapacity(
	planet Planet,
	faction Faction,
	rules Ruleset,
) int64 {
	hab := planet.Habitability

	if hab <= 0 {
		return 0
	}

	if hab < 5 {
		hab = 5
	}

	return 1_000_000 * int64(hab) / 100
}

// PopulationGrowth is intentionally only the early uncrowded skeleton.
// It does not yet represent J-RC3's persistent fractional growth carry or
// post-25%-capacity crowding. Those should be promoted from stars-elegy with
// evidence-backed tests rather than guessed here.
func PopulationGrowth(
	planet Planet,
	faction Faction,
	rules Ruleset,
) int64 {
	if planet.Habitability <= 0 {
		return 0 // hostile-world deaths are not implemented yet
	}

	return planet.Population *
		int64(faction.GrowthRate) *
		int64(planet.Habitability) /
		10_000
}
