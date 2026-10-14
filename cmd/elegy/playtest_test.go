package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// playtestScript is every command line in the sh blocks of
// docs/PLAYTEST.md's walkthrough section, in order.
func playtestScript(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "PLAYTEST.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	start := strings.Index(doc, "\n## The first turns, step by step\n")
	if start < 0 {
		t.Fatal("PLAYTEST.md has no walkthrough section")
	}
	section := doc[start+1:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	var lines []string
	in := false
	sc := bufio.NewScanner(strings.NewReader(section))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "```sh":
			in = true
		case strings.HasPrefix(line, "```"):
			in = false
		case in && line != "":
			lines = append(lines, line)
		}
	}
	return lines
}

// shellWords splits a command line into words as sh would for the
// lines the walkthrough uses: spaces separate words, double quotes group.
func shellWords(line string) ([]string, error) {
	var words []string
	var w strings.Builder
	inWord, quoted := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, inWord = !quoted, true
		case r == ' ' && !quoted:
			if inWord {
				words = append(words, w.String())
				w.Reset()
				inWord = false
			}
		default:
			w.WriteRune(r)
			inWord = true
		}
	}
	if quoted {
		return nil, fmt.Errorf("unterminated quote in %q", line)
	}
	if inWord {
		words = append(words, w.String())
	}
	return words, nil
}

// TestPlaytestScript runs docs/PLAYTEST.md's literal walkthrough: a new
// game, inspection, research, a new design queued in the same order file,
// exploration, colonization, a battle with the computer player, and a save
// copied and continued. Every command must succeed, the walkthrough must
// reach what the document says it does, and the copy continued from the
// save must reach the same state as the original.
func TestPlaytestScript(t *testing.T) {
	lines := playtestScript(t)
	if len(lines) < 20 {
		t.Fatalf("walkthrough has %d lines", len(lines))
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	var all strings.Builder
	var hashes []string
	for _, line := range lines {
		words, err := shellWords(line)
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case line == "mkdir -p playtest && cd playtest":
			// The test plays in its own temporary directory.
		case words[0] == "cp" && len(words) == 3:
			b, err := os.ReadFile(words[1])
			if err != nil {
				t.Fatalf("%s: %v", line, err)
			}
			if err := os.WriteFile(words[2], b, 0o644); err != nil {
				t.Fatal(err)
			}
		case words[0] == "../elegy":
			var out bytes.Buffer
			if err := run(words[1:], &out); err != nil {
				t.Fatalf("%s: %v\n%s", line, err, out.String())
			}
			all.WriteString(out.String())
			if len(words) > 1 && words[1] == "hash" {
				hashes = append(hashes, out.String())
			}
		default:
			t.Fatalf("walkthrough line %q: not a command the test runs", line)
		}
	}
	for _, want := range []string{
		"orders0-2400.json: order 2 added (queue)",
		"6 orders checked, 0 would be rejected",
		"Planet 264 (#18) at (1210, 1036), 50.2 ly, seen 2403, unowned, habitability 20%",
		"ships built: Planet 482 (#14), fleet #10",
		"colonized: Planet 264 (#18)",
		"#10 Gunboat #7",
		"battle summary: Planet 867 (#21); ",
		"Planet 867 (#21) at (1237, 1233), 171.2 ly, seen 2407, owned by player 1",
	} {
		if !strings.Contains(all.String(), want) {
			t.Errorf("the walkthrough never printed %q", want)
		}
	}
	if len(hashes) != 2 || hashes[0] != hashes[1] || !strings.Contains(hashes[0], "year 2408") {
		t.Fatalf("hashes after the save copy: %q", hashes)
	}
}
