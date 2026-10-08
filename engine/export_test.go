package engine

// SetParitySpace installs the parity harness's space-object loader and
// checker (objects_parity_test.go, in package engine_test, which may
// import package objects).
func SetParitySpace(s pvSpace) { pvSpaceObjects = s }

// SetParityRaces installs the parity harness's race-check builder
// (races_parity_test.go).
func SetParityRaces(f func(g *Game, settings []PVRaceSettings) RaceChecker) { pvRaces = f }

// SetParityTerraform installs the parity harness's remote mining and
// Orbital Adjusters (terraform_parity_test.go).
func SetParityTerraform(t Terraformer) { pvTerraform = t }

// withRules is g under the Elegy ruleset.
func withRules(g Game) Game {
	g.Rules = ElegyRules()
	return g
}
