package ai

// Personality is a computer player's personality (AI.md table; the
// definition file's TYPE). Only the approved three are defined.
type Personality int

const (
	Robotoid  Personality = iota // HE, definition type 1
	Rototill                     // CA, definition type 4
	Cybertron                    // PP, definition type 5
)

func (p Personality) String() string {
	switch p {
	case Robotoid:
		return "Robotoid"
	case Rototill:
		return "Rototill"
	case Cybertron:
		return "Cybertron"
	}
	return "unknown"
}

// Level is a computer player's level (AI.md §3; the definition file's
// LEVEL minus 1).
type Level int

const (
	Easy Level = iota
	Standard
	Harder
	Expert
)

// FirstYear is the calendar year of year index 0 (AI.md "Status": "Year
// index" is the year minus 2400).
const FirstYear = 2400
