package engine

// SetParitySpace installs the parity harness's space-object loader and
// checker (objects_parity_test.go, in package engine_test, which may
// import package objects).
func SetParitySpace(s pvSpace) { pvSpaceObjects = s }

// SetParityRaces installs the parity harness's race-check builder
// (races_parity_test.go).
func SetParityRaces(f func(g *Game, settings []PVRaceSettings) RaceChecker) { pvRaces = f }

// SetParityRaceSettings installs the reader of a race checker's wizard
// settings, for the harness's race expectations.
func SetParityRaceSettings(f func(r RaceChecker, player int) (PVRaceSettings, bool)) {
	pvRaceSettingsOf = f
}

// SetParityTerraform installs the parity harness's remote mining and
// Orbital Adjusters (terraform_parity_test.go).
func SetParityTerraform(t Terraformer) { pvTerraform = t }
