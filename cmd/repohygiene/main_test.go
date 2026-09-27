package main

import (
	"strings"
	"testing"
)

// The names are assembled at run time so that this file stays clean.
var (
	toolName    = string([]byte{0x43, 0x6c, 0x61, 0x75, 0x64, 0x65})
	companyName = string([]byte{0x41, 0x6e, 0x74, 0x68, 0x72, 0x6f, 0x70, 0x69, 0x63})
)

func TestCheckContent(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"Written by " + toolName + " Code", true},
		{"see https://" + strings.ToLower(companyName) + ".com", true},
		{"Co-" + "Authored-By: someone <x@example.com>", true},
		{"This text was AI-" + "generated.", true},
		{"Generated with " + "AI tooling", true},
		{"A claudication of pain", false}, // not the name
		{"SAP's assistant drafts OpenAPI specifications", false},
		{"Andreas Dier <dierandreas@gmail.com>", false},
	} {
		got := len(checkContent("f.md", []byte("intro\n"+tc.line+"\n"))) > 0
		if got != tc.want {
			t.Errorf("%q: finding = %v, want %v", tc.line, got, tc.want)
		}
	}
	if len(checkContent("bin", []byte{0, 1, 2})) != 0 {
		t.Error("binary content must be skipped")
	}
}

func record(sha, author, committer, message string) string {
	return sha + fieldSep + author + fieldSep + committer + fieldSep + message + recordSep
}

func TestCheckCommits(t *testing.T) {
	me := "Andreas Dier <dierandreas@gmail.com>"
	for _, tc := range []struct {
		name string
		log  string
		want string // substring of a finding; empty = no finding
	}{
		{"clean", record("a1", me, me, "Add recursive metadata discovery\n\nBody.\n"), ""},
		{"github merge", record("a2", "Andreas Dier <1+x@users.noreply.github.com>", "GitHub <noreply@github.com>", "Merge pull request #1\n"), ""},
		{"bot", record("a3", "dependabot[bot] <x@github.com>", "GitHub <noreply@github.com>", "chore(deps): bump x\n"), ""},
		{"trailer", record("a4", me, me, "Add a thing\n\nCo-"+"Authored-By: "+toolName+" <noreply@example.com>\n"), "message line 3"},
		{"tool author", record("a5", toolName+" <noreply@example.com>", me, "Add a thing\n"), "author"},
		{"other author", record("a6", "Someone Else <s@example.com>", me, "Add a thing\n"), "expected Andreas Dier"},
		{"conventional", record("a7", me, me, "feat(api): add a thing\n"), "Conventional Commit prefix"},
		{"conventional plain", record("a8", me, me, "docs: update readme\n"), "Conventional Commit prefix"},
		{"natural colon", record("a9", me, me, "Edge Integration Cell: enable targeting\n"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := checkCommits(tc.log, "Andreas Dier")
			if tc.want == "" {
				if len(findings) != 0 {
					t.Fatalf("unexpected findings: %+v", findings)
				}
				return
			}
			for _, f := range findings {
				if strings.Contains(f.Where+" "+f.What, tc.want) {
					return
				}
			}
			t.Fatalf("no finding containing %q in %+v", tc.want, findings)
		})
	}
}
