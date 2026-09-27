// Command repohygiene checks the repository's own material for two things
// the project does not accept: attribution of commits or content to a
// coding tool, and Conventional Commit subjects ("feat:", "fix(api):", ...).
//
// It checks the tracked files (git ls-files) and, with -commits, the author,
// committer and message of every commit in a revision range:
//
//	go run ./cmd/repohygiene                               # tracked files
//	go run ./cmd/repohygiene -commits origin/master..HEAD  # plus those commits
//
// It exits 1 when it finds something and prints file and line or commit and
// field, never more of the matched text than the line itself.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// attribution matches tool attribution. Some letters are written as
// escapes so that this file does not match itself.
var attribution = regexp.MustCompile(`(?i)\x63\x6c\x61\x75\x64\x65|\x61\x6e\x74\x68\x72\x6f\x70\x69\x63|\x63o-\x61uthored-by:|gener\x61ted (by|with) (\x61i|\x61n \x61i)|\b\x61i-(gener\x61ted|\x61ssisted)\b`)

// conventional matches a Conventional Commit subject, scoped or not.
var conventional = regexp.MustCompile(`^(feat|fix|docs|refactor|test|tests|chore|ci|build|perf|style|revert)(\([^)]*\))?!?:`)

// allowedCommitters are service identities that legitimately commit on the
// author's behalf (GitHub's web merge).
var allowedCommitters = map[string]bool{"GitHub <noreply@github.com>": true}

// Finding is one problem.
type Finding struct {
	Where string
	What  string
}

func main() {
	commits := flag.String("commits", "", "also check the commits of this revision range, for example origin/master..HEAD")
	author := flag.String("author", "Andreas Dier", "required author name of the checked commits (empty: any)")
	flag.Parse()

	var findings []Finding
	files, err := trackedFiles()
	if err != nil {
		fail(err)
	}
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // G304: tracked files of this repository
		if err != nil {
			continue // deleted in the working tree
		}
		findings = append(findings, checkContent(f, data)...)
	}
	if *commits != "" {
		log, err := commitLog(*commits)
		if err != nil {
			fail(err)
		}
		findings = append(findings, checkCommits(log, *author)...)
	}
	for _, f := range findings {
		fmt.Printf("%s: %s\n", f.Where, f.What)
	}
	if len(findings) > 0 {
		fmt.Printf("\n%d finding(s). See CONTRIBUTING.md, \"Git requirements\".\n", len(findings))
		os.Exit(1)
	}
	fmt.Println("repository hygiene: no findings")
}

func trackedFiles() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// checkContent reports every line of a text file that matches the
// attribution pattern. Binary files are skipped.
func checkContent(name string, data []byte) []Finding {
	if bytes.IndexByte(data, 0) >= 0 {
		return nil
	}
	var out []Finding
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for n := 1; sc.Scan(); n++ {
		if attribution.MatchString(sc.Text()) {
			out = append(out, Finding{Where: fmt.Sprintf("%s:%d", name, n), What: "tool attribution: " + strings.TrimSpace(sc.Text())})
		}
	}
	return out
}

// commit record separators for commitLog.
const (
	recordSep = "\x1e"
	fieldSep  = "\x1f"
)

func commitLog(rng string) (string, error) {
	format := "%H" + fieldSep + "%an <%ae>" + fieldSep + "%cn <%ce>" + fieldSep + "%B" + recordSep
	out, err := exec.Command("git", "log", "--format="+format, rng).Output() //nolint:gosec // G204: revision range from the command line
	if err != nil {
		return "", fmt.Errorf("git log %s: %w", rng, err)
	}
	return string(out), nil
}

// checkCommits checks a log produced by commitLog. Merge commits made by
// GitHub carry GitHub as committer; that is allowed.
func checkCommits(log, author string) []Finding {
	var out []Finding
	for _, rec := range strings.Split(log, recordSep) {
		rec = strings.TrimLeft(rec, "\n")
		if rec == "" {
			continue
		}
		f := strings.SplitN(rec, fieldSep, 4)
		if len(f) != 4 {
			continue
		}
		sha, authorID, committerID, message := f[0], f[1], f[2], f[3]
		short := sha
		if len(short) > 12 {
			short = short[:12]
		}
		if attribution.MatchString(authorID) {
			out = append(out, Finding{short + " author", authorID})
		}
		if attribution.MatchString(committerID) {
			out = append(out, Finding{short + " committer", committerID})
		}
		if author != "" && !strings.HasPrefix(authorID, author+" <") && !isBot(authorID) {
			out = append(out, Finding{short + " author", fmt.Sprintf("%s, expected %s", authorID, author)})
		}
		if author != "" && !strings.HasPrefix(committerID, author+" <") && !allowedCommitters[committerID] && !isBot(committerID) {
			out = append(out, Finding{short + " committer", fmt.Sprintf("%s, expected %s", committerID, author)})
		}
		subject, _, _ := strings.Cut(message, "\n")
		if conventional.MatchString(subject) && !isBot(authorID) {
			out = append(out, Finding{short + " subject", "Conventional Commit prefix: " + subject})
		}
		for i, line := range strings.Split(message, "\n") {
			if attribution.MatchString(line) {
				out = append(out, Finding{fmt.Sprintf("%s message line %d", short, i+1), strings.TrimSpace(line)})
			}
		}
	}
	return out
}

// isBot reports dependency bots, whose commits follow their own format.
func isBot(id string) bool {
	return strings.Contains(id, "[bot]")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "repohygiene:", err)
	os.Exit(2)
}
