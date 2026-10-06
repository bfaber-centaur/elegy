package engine

import "testing"

func TestGenerateTurnAdvancesYear(t *testing.T) {
	game := Game{
		Year: 2400,
	}

	result, err := GenerateTurn(game, nil, nil)
	if err != nil {
		t.Fatalf("GenerateTurn() error = %v", err)
	}

	if got, want := result.Game.Year, 2401; got != want {
		t.Errorf("year = %d, want %d", got, want)
	}
}

func TestPopulationCapacity(t *testing.T) {
	tests := []struct {
		name string
		hab  int
		want int64
	}{
		{
			name: "perfect world",
			hab:  100,
			want: 1_000_000,
		},
		{
			name: "fifty percent world",
			hab:  50,
			want: 500_000,
		},
		{
			name: "minimum positive capacity",
			hab:  1,
			want: 50_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planet := Planet{
				Habitability: tt.hab,
			}

			got := PopulationCapacity(planet, Faction{}, nil)

			if got != tt.want {
				t.Errorf("PopulationCapacity() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPopulationGrowthBelowCrowdingThreshold(t *testing.T) {
	planet := Planet{
		Habitability: 100,
		Population:   100_000,
	}

	faction := Faction{
		GrowthRate: 10,
	}

	got := PopulationGrowth(planet, faction, JRC3Rules{})
	want := int64(10_000)

	if got != want {
		t.Errorf("PopulationGrowth() = %d, want %d", got, want)
	}
}
