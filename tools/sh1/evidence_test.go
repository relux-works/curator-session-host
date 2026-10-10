package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func encodeEvents(events []Event) []byte {
	var b bytes.Buffer
	for _, event := range events {
		data, _ := json.Marshal(event)
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// Drive the real mutate CLI with the caller-command interface. The command
// emits controlled evidence so malformed lifecycles are independent of Go's
// own well-formed output. This does not claim to authenticate caller evidence.
func TestMutationRejectsInvalidLifecycleEvidence(t *testing.T) {
	pack := "example.org/sh1-fixture/internal/session"
	test := "TestAdmission"
	runEvent := Event{Action: "run", Package: pack, Test: test}
	pass := Event{Action: "pass", Package: pack, Test: test}
	fail := Event{Action: "fail", Package: pack, Test: test}
	packagePass := Event{Action: "pass", Package: pack}
	packageFail := Event{Action: "fail", Package: pack}
	goodBaseline := []Event{runEvent, pass, packagePass}
	goodMutant := []Event{runEvent, fail, packageFail}
	cases := []struct {
		name             string
		baseline, mutant []Event
	}{
		{"terminal-only-baseline", []Event{pass, packagePass}, goodMutant},
		{"terminal-only-mutant", goodBaseline, []Event{fail, packageFail}},
		{"conflicting-baseline", []Event{runEvent, pass, fail, packagePass}, goodMutant},
		{"conflicting-mutant", goodBaseline, []Event{runEvent, fail, pass, packageFail}},
		{"duplicate-terminal-baseline", []Event{runEvent, pass, pass, packagePass}, goodMutant},
		{"duplicate-terminal-mutant", goodBaseline, []Event{runEvent, fail, fail, packageFail}},
		{"unfinished-baseline", []Event{runEvent, pass, {Action: "run", Package: pack, Test: "TestGolden"}, packagePass}, goodMutant},
		{"unfinished-mutant", goodBaseline, []Event{runEvent, fail, {Action: "run", Package: pack, Test: "TestGolden"}, packageFail}},
		{"unfinished-named-baseline", []Event{runEvent, packagePass}, goodMutant},
		{"unfinished-named-mutant", goodBaseline, []Event{runEvent, packageFail}},
		{"duplicate-run-baseline", []Event{runEvent, runEvent, pass, packagePass}, goodMutant},
		{"duplicate-run-mutant", goodBaseline, []Event{runEvent, runEvent, fail, packageFail}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := fixture(t)
			dir := t.TempDir()
			decl := filepath.Join(dir, "mutations.json")
			report := filepath.Join(dir, "report.json")
			list := MutationList{Version: 1, Mutations: []Mutation{{"lifecycle", "internal/session/session.go", `identity == "member" || identity == "operator"`, `identity == "member"`, pack, test}}}
			if e := jsonWrite(decl, list); e != nil {
				t.Fatal(e)
			}
			original, e := os.ReadFile(filepath.Join(src, "internal/session/session.go"))
			if e != nil {
				t.Fatal(e)
			}
			save(t, filepath.Join(src, "evidence-original"), original)
			save(t, filepath.Join(src, "evidence-baseline"), encodeEvents(c.baseline))
			save(t, filepath.Join(src, "evidence-mutant"), encodeEvents(c.mutant))
			script := filepath.Join(dir, "command.sh")
			save(t, script, []byte("if cmp -s internal/session/session.go evidence-original; then\ncat evidence-baseline\nexit 0\nelse\ncat evidence-mutant\nexit 1\nfi\n"))
			reject(t, "mutate", "-checkout", src, "-mutations", decl, "-report", report, "--", "sh", script)
			b, e := os.ReadFile(report)
			if e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
			if bytes.Contains(b, []byte(`"killed"`)) {
				t.Fatal("invalid lifecycle reported as killed", string(b))
			}
			if c.name == "terminal-only-baseline" || strings.HasSuffix(c.name, "baseline") {
				if e == nil {
					t.Fatal("invalid baseline admitted")
				}
			} else {
				var r MutationReport
				if e := readJSON(report, &r); e != nil {
					t.Fatal(e)
				}
				if r.Ratio != "0/1" || len(r.Results) != 1 || r.Results[0].Status != "unknown" || r.Results[0].Exit != 1 {
					t.Fatal("invalid mutant not classified unknown", r)
				}
			}
			restored, e := os.ReadFile(filepath.Join(src, "internal/session/session.go"))
			if e != nil || !bytes.Equal(restored, original) {
				t.Fatal("mutation not restored")
			}

			// The same malformed stream must also refuse through inventory.
			bad := c.mutant
			if strings.HasSuffix(c.name, "baseline") {
				bad = c.baseline
			}
			eventFile := filepath.Join(dir, "events.jsonl")
			guards := filepath.Join(dir, "guards.json")
			save(t, eventFile, encodeEvents(bad))
			save(t, guards, []byte(`{"version":1,"guards":[],"opt_in":[]}`))
			reject(t, "inventory", "-source", src, "-events", eventFile, "-guards", guards, "-report", filepath.Join(dir, "inventory.json"))
		})
	}
}

func TestInventorySkippedReasonUnknown(t *testing.T) {
	for _, output := range []string{"", "=== RUN TestOptIn\n--- SKIP: TestOptIn (0.00s)\n", "    opt-in disabled\n"} {
		src := fixture(t)
		dir := t.TempDir()
		pack := "example.org/sh1-fixture/internal/session"
		events := []Event{{Action: "run", Package: pack, Test: "TestOptIn"}}
		if output != "" {
			events = append(events, Event{Action: "output", Package: pack, Test: "TestOptIn", Output: output})
		}
		events = append(events, Event{Action: "skip", Package: pack, Test: "TestOptIn"}, Event{Action: "pass", Package: pack})
		eventFile, guards, report := filepath.Join(dir, "events"), filepath.Join(dir, "guards"), filepath.Join(dir, "report")
		save(t, eventFile, encodeEvents(events))
		save(t, guards, []byte(`{"version":1,"guards":[],"opt_in":[]}`))
		call(t, "inventory", "-source", src, "-events", eventFile, "-guards", guards, "-report", report)
		var r Inventory
		if e := readJSON(report, &r); e != nil {
			t.Fatal(e)
		}
		found := false
		for _, p := range r.Packages {
			for _, test := range p.Tests {
				if test.Name == "TestOptIn" {
					want := "unknown"
					if output == "    opt-in disabled\n" {
						want = "opt-in disabled"
					}
					if test.Status != "skip" || !test.Executed || test.Reason != want {
						t.Fatal(test)
					}
					found = true
				}
			}
		}
		if !found {
			t.Fatal("skipped test absent")
		}
	}
}

func TestConsumerPreservesHistoricalChecksums(t *testing.T) {
	src, cand, _ := candidateFixture(t)
	if e := os.RemoveAll(filepath.Join(src, "internal")); e != nil {
		t.Fatal(e)
	}
	version := "v0.0.0-20261010000000-0123456789ab"
	historical := "example.org/sh1-host v0.0.0-20261009000000-0123456789ab h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n" +
		"example.org/sh1-host v0.0.0-20261009000000-0123456789ab/go.mod h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n" +
		"example.org/unrelated v1.0.0/go.mod h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n"
	save(t, filepath.Join(src, "go.sum"), []byte(historical))
	dir := t.TempDir()
	report := filepath.Join(dir, "report")
	call(t, "consumer", "-candidate", cand, "-checkout", src, "-version", version, "-proxy", filepath.Join(dir, "proxy"), "-report", report)
	var r ConsumerReport
	if e := readJSON(report, &r); e != nil {
		t.Fatal(e)
	}
	if r.SelectedVersion != version || len(r.Sum) != 2 {
		t.Fatal(r)
	}
	for i, suffix := range []string{"", "/go.mod"} {
		fields := strings.Fields(r.Sum[i])
		if len(fields) != 3 || fields[0] != "example.org/sh1-host" || fields[1] != version+suffix {
			t.Fatal(r.Sum)
		}
	}
	sum, e := os.ReadFile(filepath.Join(src, "go.sum"))
	if e != nil {
		t.Fatal(e)
	}
	for _, line := range strings.Split(strings.TrimSpace(historical), "\n") {
		if !strings.Contains(string(sum), line+"\n") {
			t.Fatal("historical sum removed", line)
		}
	}
}

func TestConsumerRejectsDifferentSelectedVersion(t *testing.T) {
	src, cand, _ := candidateFixture(t)
	if e := os.RemoveAll(filepath.Join(src, "internal")); e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	// A local unrelated dependency raises the candidate's MVS minimum. It is
	// caller-owned; no replacement for the candidate is introduced.
	dep := filepath.Join(dir, "dependency")
	newer := "v0.0.0-20261011000000-0123456789ab"
	save(t, filepath.Join(dep, "go.mod"), []byte("module example.org/dependency\n\ngo 1.25.5\nrequire example.org/sh1-host "+newer+"\n"))
	save(t, filepath.Join(src, "go.mod"), []byte("module example.org/sh1-fixture\n\ngo 1.25.5\nrequire example.org/dependency v0.0.0\nreplace example.org/dependency => "+filepath.ToSlash(dep)+"\n"))
	// The newer module is resolvable, so rejection is the selection gate,
	// rather than a missing proxy version.
	proxy := filepath.Join(dir, "proxy")
	mod, e := os.ReadFile(filepath.Join(cand, "go.mod"))
	if e != nil {
		t.Fatal(e)
	}
	save(t, filepath.Join(proxy, "example.org/sh1-host/@v/"+newer+".mod"), mod)
	save(t, filepath.Join(proxy, "example.org/sh1-host/@v/"+newer+".info"), []byte(`{"Version":"`+newer+`","Time":"2026-10-11T00:00:00Z"}`))
	report := filepath.Join(dir, "report")
	b, code, e := run(".", nil, tool, "consumer", "-candidate", cand, "-checkout", src, "-version", "v0.0.0-20261010000000-0123456789ab", "-proxy", proxy, "-report", report)
	t.Logf("different selected version CLI exit %d", code)
	if e != nil || code != 1 || !bytes.Contains(b, []byte("effective candidate selection differs")) {
		t.Fatalf("wrong refusal: exit %d, %v, %s", code, e, b)
	}
	if _, e := os.Stat(report); !os.IsNotExist(e) {
		t.Fatal("wrong-version success report written", e)
	}
}

func TestLifecycleNarrowingMutant(t *testing.T) {
	root := t.TempDir()
	copyTree(t, ".", filepath.Join(root, "tools/sh1"))
	copyTree(t, "../../testdata/sh1-fixture", filepath.Join(root, "testdata/sh1-fixture"))
	mod, e := os.ReadFile("../../go.mod")
	if e != nil {
		t.Fatal(e)
	}
	save(t, filepath.Join(root, "go.mod"), mod)
	p := filepath.Join(root, "tools/sh1/inventory.go")
	original, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	anchor := `if !runs[ref] || end[ref] != "" {`
	// Narrow lifecycle enforcement to source guards; admission evidence now
	// escapes it. This preserves the check instead of deleting the gate.
	narrowed := `if (!runs[ref] || end[ref] != "") && ev.Test == "TestSourceGuard" {`
	if strings.Count(string(original), anchor) != 2 {
		t.Fatal("lifecycle mutation anchor changed")
	}
	save(t, p, []byte(strings.Replace(string(original), anchor, narrowed, 1)))
	args := []string{"go", "test", "-json", "-count=1", "-run=^TestMutationRejectsInvalidLifecycleEvidence$", "./tools/sh1/..."}
	output, code, e := run(root, []string{"SH1_ARTIFACT_DIR="}, args...)
	t.Logf("narrowed lifecycle regression go test exit %d (expected red)", code)
	if e != nil || code != 1 {
		t.Fatalf("narrowing did not fail: exit %d, %v\n%s", code, e, output)
	}
	events, e := eventStream(output)
	pack := "github.com/relux-works/curator-session-host/tools/sh1"
	if e != nil || !hasTest(events, pack, "TestMutationRejectsInvalidLifecycleEvidence", "fail") || !hasTest(events, pack, "", "fail") {
		t.Fatalf("missing named regression failure: %v\n%s", e, output)
	}
	if artifacts := os.Getenv("SH1_ARTIFACT_DIR"); artifacts != "" {
		save(t, filepath.Join(artifacts, "lifecycle-narrowing.jsonl"), output)
		if e := jsonWrite(filepath.Join(artifacts, "lifecycle-narrowing.json"), struct {
			Name   string `json:"name"`
			Test   string `json:"test"`
			Before string `json:"before"`
			After  string `json:"after"`
			Exit   int    `json:"exit"`
			Ratio  string `json:"kill_ratio"`
		}{"lifecycle-only-source-guards", "TestMutationRejectsInvalidLifecycleEvidence", digest(original), digest([]byte(strings.Replace(string(original), anchor, narrowed, 1))), code, "1/1"}); e != nil {
			t.Fatal(e)
		}
	}
}
