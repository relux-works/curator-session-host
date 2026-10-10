package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"
)

type TestRef struct {
	Package string `json:"package"`
	Test    string `json:"test"`
}
type Guard struct {
	Package string   `json:"package"`
	Test    string   `json:"test"`
	Targets []string `json:"targets"`
}
type OptIn struct {
	Package   string `json:"package"`
	Test      string `json:"test"`
	Condition string `json:"condition"`
}
type GuardList struct {
	Version int     `json:"version"`
	Guards  []Guard `json:"guards"`
	OptIn   []OptIn `json:"opt_in"`
}
type Event struct {
	Action  string
	Package string
	Test    string
	Output  string
}
type TestResult struct {
	Name     string `json:"name"`
	File     string `json:"file,omitempty"`
	Tags     string `json:"tags,omitempty"`
	OptIn    string `json:"opt_in,omitempty"`
	Executed bool   `json:"executed"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}
type PackageInventory struct {
	Package string       `json:"package"`
	Status  string       `json:"status"`
	Tests   []TestResult `json:"tests"`
}
type GuardCoverage struct {
	Package  string   `json:"package"`
	Test     string   `json:"test"`
	Expected []string `json:"expected"`
	Reached  []string `json:"reached"`
	Ratio    string   `json:"ratio"`
}
type Inventory struct {
	Version     int                `json:"version"`
	Source      string             `json:"source"`
	Events      string             `json:"events"`
	Declaration string             `json:"declaration"`
	Packages    []PackageInventory `json:"packages"`
	Guards      []GuardCoverage    `json:"guards"`
	Ratio       string             `json:"guard_ratio"`
}

func eventStream(b []byte) ([]Event, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	events := []Event{}
	terminal := map[string]bool{}
	seen := map[string]bool{}
	runs := map[TestRef]bool{}
	end := map[TestRef]string{}
	failed := map[string]bool{}
	for {
		var ev Event
		e := d.Decode(&ev)
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if ev.Package == "" {
			return nil, errors.New("event without package")
		}
		switch ev.Action {
		case "start", "run", "pause", "cont", "pass", "fail", "skip", "output", "bench":
		default:
			return nil, fmt.Errorf("unknown event action %q", ev.Action)
		}
		if terminal[ev.Package] {
			return nil, errors.New("event after package terminal")
		}
		seen[ev.Package] = true
		if ev.Test != "" {
			ref := TestRef{ev.Package, ev.Test}
			switch ev.Action {
			case "run":
				if runs[ref] {
					return nil, errors.New("duplicate test execution; use count=1")
				}
				runs[ref] = true
			case "pass", "fail", "skip":
				if !runs[ref] || end[ref] != "" {
					return nil, errors.New("test terminal without unique execution")
				}
				end[ref] = ev.Action
				if ev.Action == "fail" {
					failed[ev.Package] = true
				}
			case "pause", "cont", "bench":
				if !runs[ref] || end[ref] != "" {
					return nil, errors.New("test activity outside execution")
				}
			case "output":
				// Go may emit diagnostic output outside a test execution.
			default:
				return nil, errors.New("package action attributed to test")
			}
		}
		if ev.Test == "" && (ev.Action == "pass" || ev.Action == "fail" || ev.Action == "skip") {
			for ref := range runs {
				if ref.Package == ev.Package && end[ref] == "" {
					return nil, errors.New("incomplete test execution before package terminal")
				}
			}
			if failed[ev.Package] && ev.Action != "fail" {
				return nil, errors.New("failed test without package failure")
			}
			terminal[ev.Package] = true
		}
		events = append(events, ev)
	}
	if len(events) == 0 {
		return nil, errors.New("empty event stream")
	}
	for p := range seen {
		if !terminal[p] {
			return nil, fmt.Errorf("incomplete package %s", p)
		}
	}
	return events, nil
}
func inventory(source, eventFile, guardFile, report string) error {
	files, _, e := tree(source)
	if e != nil {
		return e
	}
	mod, _, e := modIdentity(files["go.mod"])
	if e != nil {
		return e
	}
	var list GuardList
	if e = readJSON(guardFile, &list); e != nil {
		return e
	}
	if list.Version != 1 || list.Guards == nil || list.OptIn == nil {
		return errors.New("invalid guard declaration")
	}
	packages := map[string]map[string]*TestResult{}
	statuses := map[string]string{}
	for _, p := range keys(files) {
		if !strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, e := parser.ParseFile(token.NewFileSet(), p, files[p], parser.ParseComments)
		if e != nil {
			return e
		}
		pack := mod
		if path.Dir(p) != "." {
			pack += "/" + path.Dir(p)
		}
		if packages[pack] == nil {
			packages[pack] = map[string]*TestResult{}
		}
		tags := []string{}
		for _, c := range f.Comments {
			for _, l := range c.List {
				if strings.HasPrefix(l.Text, "//go:build ") {
					tags = append(tags, strings.TrimPrefix(l.Text, "//go:build "))
				}
			}
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			tail := strings.TrimPrefix(fn.Name.Name, "Test")
			if tail != "" && unicode.IsLower([]rune(tail)[0]) {
				continue
			}
			if _, ok := packages[pack][fn.Name.Name]; ok {
				return errors.New("duplicate declared test")
			}
			packages[pack][fn.Name.Name] = &TestResult{Name: fn.Name.Name, File: p, Tags: strings.Join(tags, " && "), Status: "not_observed"}
		}
	}
	lookup := func(pack, name string) (*TestResult, error) {
		r := packages[pack][name]
		if r == nil {
			return nil, fmt.Errorf("undeclared test %s/%s", pack, name)
		}
		return r, nil
	}
	optSeen := map[TestRef]bool{}
	for _, o := range list.OptIn {
		r, e := lookup(o.Package, o.Test)
		if e != nil {
			return e
		}
		ref := TestRef{o.Package, o.Test}
		if o.Condition == "" || optSeen[ref] {
			return errors.New("invalid opt-in declaration")
		}
		optSeen[ref] = true
		r.OptIn = o.Condition
	}
	reached := map[TestRef]map[string]bool{}
	outputs := map[TestRef]string{}
	end := map[TestRef]string{}
	b, e := os.ReadFile(eventFile)
	if e != nil {
		return e
	}
	events, e := eventStream(b)
	if e != nil {
		return e
	}
	for _, ev := range events {
		if ev.Test == "" {
			if packages[ev.Package] == nil {
				packages[ev.Package] = map[string]*TestResult{}
			}
			if ev.Action == "pass" || ev.Action == "fail" || ev.Action == "skip" {
				statuses[ev.Package] = ev.Action
			}
			continue
		}
		ref := TestRef{ev.Package, ev.Test}
		if packages[ev.Package] == nil {
			packages[ev.Package] = map[string]*TestResult{}
		}
		r := packages[ev.Package][ev.Test]
		if r == nil {
			if !strings.Contains(ev.Test, "/") {
				return errors.New("runtime test missing from source discovery")
			}
			r = &TestResult{Name: ev.Test, Status: "not_observed"}
			packages[ev.Package][ev.Test] = r
		}
		switch ev.Action {
		case "run":
			r.Executed = true
			r.Status = "running"
		case "pass", "fail", "skip":
			end[ref] = ev.Action
			r.Status = ev.Action
		case "output":
			outputs[ref] += ev.Output
			for _, line := range strings.Split(ev.Output, "\n") {
				line = strings.TrimSpace(line)
				i := strings.Index(line, "SH1_GUARD ")
				if i >= 0 {
					target := strings.TrimSpace(line[i+len("SH1_GUARD "):])
					if reached[ref] == nil {
						reached[ref] = map[string]bool{}
					}
					reached[ref][target] = true
				}
			}
		}
	}
	for ref, status := range end {
		if status != "skip" {
			continue
		}
		reason := []string{}
		for _, line := range strings.Split(outputs[ref], "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "===") && !strings.HasPrefix(line, "---") {
				reason = append(reason, line)
			}
		}
		packages[ref.Package][ref.Test].Reason = strings.Join(reason, "\n")
		if len(reason) == 0 {
			packages[ref.Package][ref.Test].Reason = "unknown"
		}
	}
	gb, e := os.ReadFile(guardFile)
	if e != nil {
		return e
	}
	r := Inventory{Version: 1, Events: digest(b), Declaration: digest(gb), Packages: []PackageInventory{}, Guards: []GuardCoverage{}}
	var sb bytes.Buffer
	for _, p := range keys(files) {
		fmt.Fprintf(&sb, "%s\x00%s\n", p, digest(files[p]))
	}
	r.Source = digest(sb.Bytes())
	for _, p := range keys(packages) {
		status := statuses[p]
		if status == "" {
			status = "not_observed"
		}
		pi := PackageInventory{p, status, []TestResult{}}
		for _, n := range keys(packages[p]) {
			pi.Tests = append(pi.Tests, *packages[p][n])
		}
		r.Packages = append(r.Packages, pi)
	}
	expectedCount, reachedCount := 0, 0
	guardSeen := map[TestRef]bool{}
	for _, g := range list.Guards {
		if _, e = lookup(g.Package, g.Test); e != nil {
			return e
		}
		ref := TestRef{g.Package, g.Test}
		if guardSeen[ref] || len(g.Targets) == 0 {
			return errors.New("empty or duplicate guard")
		}
		guardSeen[ref] = true
		expected := append([]string{}, g.Targets...)
		sort.Strings(expected)
		got := []string{}
		for i, p := range expected {
			if e = relative(p); e != nil {
				return e
			}
			if i > 0 && expected[i-1] == p {
				return errors.New("duplicate guard target")
			}
			prefix := strings.TrimPrefix(g.Package, mod)
			if prefix == g.Package {
				return errors.New("guard package outside source module")
			}
			file := strings.TrimPrefix(prefix+"/"+p, "/")
			if _, ok := files[file]; !ok {
				return errors.New("guard target absent from source")
			}
			expectedCount++
			if end[ref] == "pass" && reached[ref][p] {
				got = append(got, p)
				reachedCount++
			}
		}
		r.Guards = append(r.Guards, GuardCoverage{g.Package, g.Test, expected, got, fmt.Sprintf("%d/%d", len(got), len(expected))})
	}
	sort.Slice(r.Guards, func(i, j int) bool {
		a, b := r.Guards[i], r.Guards[j]
		return a.Package+"/"+a.Test < b.Package+"/"+b.Test
	})
	r.Ratio = fmt.Sprintf("%d/%d", reachedCount, expectedCount)
	if e = jsonWrite(report, r); e != nil {
		return e
	}
	if reachedCount != expectedCount {
		return errors.New("guard targets not reached")
	}
	return nil
}
