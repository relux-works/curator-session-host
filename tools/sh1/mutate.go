package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Mutation struct {
	Name        string `json:"name"`
	File        string `json:"file"`
	Anchor      string `json:"anchor"`
	Replacement string `json:"replacement"`
	Package     string `json:"package"`
	Test        string `json:"test"`
}
type MutationList struct {
	Version   int        `json:"version"`
	Mutations []Mutation `json:"mutations"`
}
type MutationResult struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Package string `json:"package"`
	Test    string `json:"test"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Exit    int    `json:"exit"`
	Status  string `json:"status"`
}
type MutationReport struct {
	Version      int              `json:"version"`
	Declaration  string           `json:"declaration"`
	Command      []string         `json:"command"`
	BaselineExit int              `json:"baseline_exit"`
	Results      []MutationResult `json:"results"`
	Ratio        string           `json:"kill_ratio"`
}

// eventStream validates a unique completed execution before this lookup.
func hasTest(events []Event, pack, test, status string) bool {
	for _, ev := range events {
		if ev.Package == pack && ev.Test == test && ev.Action == status {
			return true
		}
	}
	return false
}
func mutate(checkout, listFile, report string, command []string) error {
	var list MutationList
	if e := readJSON(listFile, &list); e != nil {
		return e
	}
	if list.Version != 1 || len(list.Mutations) == 0 {
		return errors.New("nonempty version 1 mutation list required")
	}
	originals := map[string][]byte{}
	modes := map[string]fs.FileMode{}
	names := map[string]bool{}
	for _, m := range list.Mutations {
		if e := relative(m.File); e != nil {
			return e
		}
		if names[m.Name] || m.Name == "" || m.Anchor == "" || m.Anchor == m.Replacement || m.Package == "" || m.Test == "" {
			return errors.New("invalid mutation declaration")
		}
		names[m.Name] = true
		p := filepath.Join(checkout, m.File)
		info, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return errors.New("mutation requires regular file")
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		if strings.Count(string(b), m.Anchor) != 1 {
			return fmt.Errorf("mutation %s needs exactly one anchor", m.Name)
		}
		originals[m.File] = b
		modes[m.File] = info.Mode().Perm()
	}
	// The complete tree read rejects symlink parents as well as leaf links.
	if _, _, e := tree(checkout); e != nil {
		return e
	}
	baseline, code, e := run(checkout, nil, command...)
	// Keep the caller's complete stream, including a refused baseline. Raw
	// output belongs in job logs, never in source-free report artifacts.
	fmt.Fprintf(os.Stderr, "SH1 mutation baseline exit %d\n", code)
	if _, outputErr := os.Stdout.Write(baseline); outputErr != nil {
		return outputErr
	}
	if e != nil {
		return e
	}
	events, e := eventStream(baseline)
	if e != nil {
		return fmt.Errorf("baseline evidence: %w", e)
	}
	if code != 0 {
		return fmt.Errorf("baseline failed with exit %d", code)
	}
	for _, m := range list.Mutations {
		if !hasTest(events, m.Package, m.Test, "pass") {
			return errors.New("baseline did not pass each named test")
		}
	}
	declaration, e := os.ReadFile(listFile)
	if e != nil {
		return e
	}
	r := MutationReport{1, digest(declaration), command, code, []MutationResult{}, ""}
	killed := 0
	for _, m := range list.Mutations {
		before := originals[m.File]
		after := []byte(strings.Replace(string(before), m.Anchor, m.Replacement, 1))
		p := filepath.Join(checkout, m.File)
		if e = write(p, after, modes[m.File]); e != nil {
			return e
		}
		output, exit, runErr := run(checkout, nil, command...)
		restoreErr := write(p, before, modes[m.File])
		if restoreErr != nil {
			return restoreErr
		}
		fmt.Fprintf(os.Stderr, "SH1 mutation %s exit %d\n", m.Name, exit)
		if _, outputErr := os.Stdout.Write(output); outputErr != nil {
			return outputErr
		}
		if runErr != nil {
			return runErr
		}
		ev, parseErr := eventStream(output)
		status := "survived"
		if parseErr == nil && exit != 0 && hasTest(ev, m.Package, m.Test, "fail") && hasTest(ev, m.Package, "", "fail") {
			status = "killed"
			killed++
		}
		if parseErr != nil {
			status = "unknown"
		}
		r.Results = append(r.Results, MutationResult{m.Name, m.File, m.Package, m.Test, digest(before), digest(after), exit, status})
	}
	r.Ratio = fmt.Sprintf("%d/%d", killed, len(r.Results))
	if e = jsonWrite(report, r); e != nil {
		return e
	}
	if killed != len(r.Results) {
		return errors.New("mutations survived or lacked valid failure evidence")
	}
	return nil
}
