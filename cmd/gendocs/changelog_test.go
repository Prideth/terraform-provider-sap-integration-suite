package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	changelogUnreleased = regexp.MustCompile(`^## Unreleased( \(planned as \d+\.\d+\.\d+\))?$`)
	changelogRelease    = regexp.MustCompile(`^## (\d+\.\d+\.\d+) — (\d{4}-\d{2}-\d{2})$`)
)

// TestChangelog_ReleasesAreFinished keeps CHANGELOG.md readable at every
// tag: a second-level heading is either the single Unreleased section at
// the top or a released version with its date, and every released version
// has the hand-written notes of its GitHub release page. The release
// workflow checks the same heading at the tagged commit.
func TestChangelog_ReleasesAreFinished(t *testing.T) {
	root := filepath.Join("..", "..")
	src, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	var releases []string
	for i, line := range strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		switch {
		case changelogUnreleased.MatchString(line):
			if len(releases) > 0 {
				t.Errorf("CHANGELOG.md:%d: the Unreleased section must come before every release", i+1)
			}
		case changelogRelease.MatchString(line):
			releases = append(releases, changelogRelease.FindStringSubmatch(line)[1])
		default:
			t.Errorf("CHANGELOG.md:%d: %q is neither \"## Unreleased\" nor \"## <version> — <YYYY-MM-DD>\"", i+1, line)
		}
	}
	if len(releases) == 0 {
		t.Fatal("CHANGELOG.md has no released version")
	}
	for _, v := range releases {
		notes := filepath.Join(root, ".github", "release-notes", "v"+v+".md")
		if info, err := os.Stat(notes); err != nil || info.Size() == 0 {
			t.Errorf("release %s has no release notes in %s", v, notes)
		}
	}
}
