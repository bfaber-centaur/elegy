package races

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
)

// Computer-player types in definition-file order (stars-elegy AI.md, "#
// TYPE LEVEL": 1 HE, 2 SS, 3 IS, 4 CA, 5 PP, 6 AR).
var builtInTypes = []engine.PRT{
	engine.PRTHyperExpansion, engine.PRTSuperStealth, engine.PRTInnerStrength,
	engine.PRTClaimAdjuster, engine.PRTPacketPhysics, engine.PRTAlternateReality,
}

// builtInRows is AI.md "Built-in races" (CONFIRMED, AI-0, for 23 of 24;
// SS harder BINARY-ONLY), one row per type and level (easy, standard,
// harder, expert), in that table's columns: LRTs; growth; habitat
// gravity / temperature / radiation; colonists per resource; factories
// output/cost/operated; mines output/cost/operated; research costs
// energy..biotech (c cheap, n normal, x expensive); factories cost 1 less
// germanium; expensive fields start at 3.
var builtInRows = [6][4]string{
	{ // HE
		"IFE MA CE OBRM BET|5|imm/imm/imm|1000|12/10/16|10/5/10|nncnnx|no|no",
		"IFE MA CE OBRM|6|imm/imm/imm|900|13/9/16|10/4/11|nncnnx|no|no",
		"IFE UR MA OBRM|6|imm/imm/imm|800|13/9/18|10/4/12|ncccnx|yes|no",
		"IFE UR MA OBRM|7|imm/imm/imm|800|13/9/16|10/4/8|ncncnx|yes|no",
	},
	{ // SS
		"IFE ARM MA RS|14|27-89/7-63/35-95|1000|9/10/9|9/5/8|nxnnnx|no|no",
		"IFE ARM MA RS|14|32-92/6-60/26-96|1000|10/10/10|10/5/9|nnnnnn|yes|no",
		"IFE ARM MA RS|14|31-95/4-52/30-94|900|11/10/10|10/5/9|xnxnnn|yes|no",
		"IFE ARM MA RS|15|31-93/5-53/imm|800|15/10/25|10/5/9|xxxxxx|yes|yes",
	},
	{ // IS
		"GR CE OBRM NAS LSP|15|7-63/26-94/5-71|900|11/10/14|11/6/14|xxxxxx|no|yes",
		"GR CE OBRM NAS LSP|15|7-63/26-94/5-71|800|13/9/14|10/6/14|xxxxxx|yes|yes",
		"GR OBRM NAS LSP|15|7-63/26-94/5-71|800|14/9/15|14/5/15|xxxxxx|yes|yes",
		"GR OBRM NAS LSP|16|7-63/imm/0-100|800|14/9/14|14/5/14|xxxxxx|yes|yes",
	},
	{ // CA
		"TT CE OBRM NAS LSP BET|15|32-68/31-69/31-69|1000|10/10/10|10/5/10|xxxxxc|no|yes",
		"TT OBRM NAS LSP BET|15|32-68/31-69/31-69|800|12/10/12|14/5/12|xxxxxc|no|yes",
		"TT OBRM NAS LSP BET|15|23-77/24-76/25-75|800|12/10/12|14/5/12|xxxxxc|no|yes",
		"TT OBRM NAS LSP BET|15|imm/24-76/25-75|800|15/10/15|15/5/15|xxxxxc|no|yes",
	},
	{ // PP
		"IFE TT OBRM LSP|12|22-78/22-78/22-78|1000|9/18/9|9/10/8|nxxnxx|no|yes",
		"IFE TT OBRM NAS LSP|17|19-81/19-81/19-81|1000|10/13/19|10/10/7|nxxnnn|no|yes",
		"IFE TT MA OBRM NAS LSP|17|18-82/18-82/18-82|1000|14/10/20|10/10/6|nnxnnc|yes|yes",
		"IFE TT MA OBRM NAS LSP|19|17-83/17-83/17-83|1000|15/9/25|10/10/5|ccxcnn|yes|yes",
	},
	{ // AR
		"IFE TT ISB GR CE|10|20-80/20-80/20-80|1600|10/10/10|10/5/10|nnnnxn|no|no",
		"IFE TT ISB GR|14|15-85/15-85/15-85|1200|10/10/10|10/5/10|cnnnxn|no|no",
		"IFE TT ARM ISB GR UR MA|17|15-85/15-85/15-85|1000|10/10/10|10/5/10|cnnnnn|no|no",
		"IFE TT ARM ISB GR UR MA|20|15-85/15-85/15-85|1000|10/10/10|10/5/10|cnncnn|no|no",
	},
}

// BuiltIn is the original's built-in race for a computer player of
// definition-file type 1–6 (HE, SS, IS, CA, PP, AR) and level 0–3 (easy,
// standard, harder, expert), from stars-elegy AI.md "Built-in races". The
// name is left empty: the 24 built-in names are not in the spec.
//
// ASSUMPTION B1: AI.md does not list a built-in race's leftover spend
// (UNIVERSE.md shows computer players with both the minerals and the
// concentrations spend); BuiltIn leaves it at surface minerals for the
// caller to set.
func BuiltIn(typ, level int) (Design, error) {
	if typ < 1 || typ > len(builtInTypes) || level < 0 || level > 3 {
		return Design{}, fmt.Errorf("races: no built-in race for type %d level %d", typ, level)
	}
	d, err := parseBuiltIn(builtInRows[typ-1][level])
	if err != nil {
		return Design{}, err
	}
	d.Race.PRT = builtInTypes[typ-1]
	return d, nil
}

func parseBuiltIn(row string) (Design, error) {
	f := strings.Split(row, "|")
	if len(f) != 9 {
		return Design{}, fmt.Errorf("races: bad built-in row %q", row)
	}
	d := Design{}
	for _, code := range strings.Fields(f[0]) {
		i := lrtIndex(code)
		if i < 0 {
			return Design{}, fmt.Errorf("races: unknown LRT %q", code)
		}
		d.SetLRT(i, true)
	}
	var err error
	num := func(s string) int {
		v, e := strconv.Atoi(s)
		if e != nil && err == nil {
			err = e
		}
		return v
	}
	triple := func(s string) (int, int, int) {
		p := strings.Split(s, "/")
		if len(p) != 3 {
			err = fmt.Errorf("races: bad triple %q", s)
			return 0, 0, 0
		}
		return num(p[0]), num(p[1]), num(p[2])
	}
	r := &d.Race
	r.GrowthRate = num(f[1])
	for a, h := range strings.Split(f[2], "/") {
		if a > 2 {
			return Design{}, fmt.Errorf("races: bad habitat %q", f[2])
		}
		if h == "imm" {
			r.Env[a] = engine.EnvRange{Immune: true}
			continue
		}
		lo, hi, ok := strings.Cut(h, "-")
		if !ok {
			return Design{}, fmt.Errorf("races: bad range %q", h)
		}
		r.Env[a] = span(num(lo), num(hi))
	}
	r.ColonistsPerResource = num(f[3])
	r.FactoryOutput, r.FactoryCost, r.FactoriesOperated = triple(f[4])
	r.MineOutput, r.MineCost, r.MinesOperated = triple(f[5])
	if len(f[6]) != engine.NumFields {
		return Design{}, fmt.Errorf("races: bad research %q", f[6])
	}
	for i, c := range f[6] {
		l := strings.IndexRune("xnc", c)
		if l < 0 {
			return Design{}, fmt.Errorf("races: bad research %q", f[6])
		}
		r.ResearchCosts[i] = researchCost(l)
	}
	r.FactoryLessGermanium = f[7] == "yes"
	d.ExpensiveAt3 = f[8] == "yes"
	return d, err
}

func lrtIndex(code string) int {
	for i := range NumLRTs {
		if LRTCode(i) == code {
			return i
		}
	}
	return -1
}
