package engine_test

import (
	"github.com/bfaber-centaur/elegy/engine"
	"github.com/bfaber-centaur/elegy/terraform"
)

// The parity harness's remote mining and Orbital Adjusters (KERNEL.md
// "Turn order" steps 6c.2 and 7.4). This file is in package engine_test
// because package terraform imports the engine.

func init() { engine.SetParityTerraform(terraform.Rules{}) }
