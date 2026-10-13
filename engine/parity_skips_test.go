package engine_test

import (
	"slices"
	"strings"
	"testing"
)

func TestSkipCensusDiff(t *testing.T) {
	want := map[string]int{"message 0x002 has no event mapping": 3, "expectation battle": 183}
	for _, tc := range []struct {
		name string
		got  map[string]int
		diff []string
	}{
		{"same", map[string]int{"message 0x002 has no event mapping": 3, "expectation battle": 183}, nil},
		{"rise", map[string]int{"message 0x002 has no event mapping": 4, "expectation battle": 183},
			[]string{`skipped expectations "message 0x002 has no event mapping": 4, was 3`}},
		{"fall", map[string]int{"message 0x002 has no event mapping": 3, "expectation battle": 182},
			[]string{`skipped expectations "expectation battle": 182, was 183`}},
		{"new reason", map[string]int{"message 0x002 has no event mapping": 3, "expectation battle": 183, "message 0x026 has no event mapping": 2},
			[]string{`skipped expectations "message 0x026 has no event mapping": 2, was 0`}},
		{"reason gone", map[string]int{"expectation battle": 183},
			[]string{`skipped expectations "message 0x002 has no event mapping": 0, was 3`}},
	} {
		if d := skipCensusDiff(want, tc.got); !slices.Equal(d, tc.diff) {
			t.Errorf("%s: diff %q, want %q", tc.name, d, tc.diff)
		}
	}
}

func TestSkipCensusRoundTrip(t *testing.T) {
	census := map[string]int{"client estimate status_tile population": 10, "expectation battle": 183, "Super Stealth research stealing": 3}
	got, err := parseSkipCensus(formatSkipCensus(census))
	if err != nil {
		t.Fatal(err)
	}
	if d := skipCensusDiff(census, got); d != nil {
		t.Errorf("round trip differs: %q", d)
	}
	want := "   3 Super Stealth research stealing\n  10 client estimate status_tile population\n 183 expectation battle\n"
	if s := formatSkipCensus(census); !strings.HasSuffix(s, want) {
		t.Errorf("census lines:\n%s\nwant them sorted by reason:\n%s", s, want)
	}
}

func TestParseSkipCensusRejects(t *testing.T) {
	for _, s := range []string{
		"3\n",
		"x reason\n",
		"0 reason\n",
		"-2 reason\n",
		"3 reason\n4 reason\n",
	} {
		if _, err := parseSkipCensus(s); err == nil {
			t.Errorf("parseSkipCensus(%q) accepted it", s)
		}
	}
}
