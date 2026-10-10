package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type ConsumerReport struct {
	Version       int             `json:"version"`
	Module        string          `json:"module"`
	PseudoVersion string          `json:"pseudo_version"`
	Candidate     string          `json:"candidate"`
	Proxy         []FileRecord    `json:"proxy"`
	Consumer      []FileRecord    `json:"consumer"`
	Sum           []string        `json:"go_sum"`
	Graph         []string        `json:"module_graph"`
	Commands      []CommandResult `json:"commands"`
}
type CommandResult struct {
	Command []string `json:"command"`
	Exit    int      `json:"exit"`
}

func immutableWrite(p string, b []byte) error {
	existing, e := os.ReadFile(p)
	if e == nil {
		if !bytes.Equal(existing, b) {
			return errors.New("proxy version already contains different bytes")
		}
		return nil
	}
	if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return write(p, b, 0644)
}
func consumer(root, checkout, version, proxy, report string) error {
	m, files, e := candidate(root)
	if e != nil {
		return e
	}
	if e = modulePath(m.New); e != nil {
		return e
	}
	// v0 candidates only; immutable release tags belong to later SH1 lanes.
	re := regexp.MustCompile(`^v0\.0\.0-([0-9]{14})-([0-9a-f]{12})$`)
	parts := re.FindStringSubmatch(version)
	if parts == nil {
		return errors.New("version must be a v0 pseudo-version")
	}
	timestamp, e := time.Parse("20060102150405", parts[1])
	if e != nil {
		return e
	}
	name, _, e := modIdentity(files["go.mod"])
	if e != nil {
		return e
	}
	if name != m.New {
		return errors.New("candidate module identity mismatch")
	}
	before, _, e := tree(checkout)
	if e != nil {
		return e
	}
	if _, _, e = modIdentity(before["go.mod"]); e != nil {
		return e
	}
	// A replacement of this module would bypass the proxy. Inspect via Go's parser.
	out, exit, e := run(checkout, []string{"GOWORK=off"}, "go", "mod", "edit", "-json")
	if e != nil {
		return e
	}
	if exit != 0 {
		return fmt.Errorf("go mod edit -json exit %d", exit)
	}
	var mod struct {
		Replace []struct {
			Old struct{ Path string }
			New struct{ Path string }
		}
	}
	if e = json.Unmarshal(out, &mod); e != nil {
		return e
	}
	for _, r := range mod.Replace {
		if r.Old.Path == m.New {
			return errors.New("candidate replace directive refused")
		}
	}
	var z bytes.Buffer
	zw := zip.NewWriter(&z)
	for _, p := range keys(files) {
		h := &zip.FileHeader{Name: m.New + "@" + version + "/" + p, Method: zip.Deflate}
		h.SetMode(0644)
		w, e := zw.CreateHeader(h)
		if e != nil {
			return e
		}
		if _, e = w.Write(files[p]); e != nil {
			return e
		}
	}
	if e = zw.Close(); e != nil {
		return e
	}
	info, e := json.Marshal(struct {
		Version string
		Time    time.Time
	}{version, timestamp.UTC()})
	if e != nil {
		return e
	}
	payload := map[string][]byte{version + ".mod": files["go.mod"], version + ".info": append(info, '\n'), version + ".zip": z.Bytes(), "list": []byte(version + "\n")}
	proxyRoot, e := filepath.Abs(proxy)
	if e != nil {
		return e
	}
	base := filepath.Join(proxyRoot, filepath.FromSlash(m.New), "@v")
	mb, e := os.ReadFile(filepath.Join(root, manifestName))
	if e != nil {
		return e
	}
	r := ConsumerReport{1, m.New, version, digest(mb), []FileRecord{}, []FileRecord{}, []string{}, []string{}, []CommandResult{}}
	for _, p := range keys(payload) {
		if e = immutableWrite(filepath.Join(base, p), payload[p]); e != nil {
			return e
		}
		r.Proxy = append(r.Proxy, FileRecord{Path: m.New + "/@v/" + p, After: digest(payload[p]), Mode: 0644})
	}
	proxyURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxyRoot)}).String()
	// Fresh caches guarantee that evidence comes from this proxy, even on repeated runs.
	cache, e := os.MkdirTemp("", "sh1-consumer-cache-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(cache)
	env := []string{"GOPROXY=" + proxyURL, "GOSUMDB=off", "GOPRIVATE=", "GONOPROXY=none", "GONOSUMDB=none", "GOWORK=off", "GOMODCACHE=" + cache, "GOFLAGS="}
	invoke := func(args ...string) ([]byte, error) {
		b, code, e := run(checkout, env, args...)
		r.Commands = append(r.Commands, CommandResult{args, code})
		if e != nil {
			return nil, e
		}
		if code != 0 {
			return nil, fmt.Errorf("consumer %s exited %d", strings.Join(args, " "), code)
		}
		return b, nil
	}
	if _, e = invoke("go", "mod", "edit", "-require="+m.New+"@"+version); e != nil {
		return e
	}
	for _, p := range keys(before) {
		if strings.HasSuffix(p, ".go") {
			next, e := rewrite(before[p], m.Old, m.New)
			if e != nil {
				return e
			}
			if !bytes.Equal(next, before[p]) {
				if e = write(filepath.Join(checkout, p), next, 0644); e != nil {
					return e
				}
			}
		}
	}
	if _, e = invoke("go", "mod", "download", m.New); e != nil {
		return e
	}
	graph, e := invoke("go", "mod", "graph")
	if e != nil {
		return e
	}
	for _, line := range strings.Split(strings.TrimSpace(string(graph)), "\n") {
		if line != "" {
			r.Graph = append(r.Graph, line)
		}
	}
	sort.Strings(r.Graph)
	sum, e := os.ReadFile(filepath.Join(checkout, "go.sum"))
	if e != nil {
		return e
	}
	for _, line := range strings.Split(string(sum), "\n") {
		if strings.HasPrefix(line, m.New+" ") {
			r.Sum = append(r.Sum, line)
		}
	}
	sort.Strings(r.Sum)
	if len(r.Sum) != 2 {
		return errors.New("expected module and go.mod checksum lines")
	}
	after, _, e := tree(checkout)
	if e != nil {
		return e
	}
	for _, p := range keys(after) {
		r.Consumer = append(r.Consumer, FileRecord{Path: p, Before: hashIfPresent(before, p), After: digest(after[p]), Mode: 0644, Protected: protected(p)})
	}
	return jsonWrite(report, r)
}
func hashIfPresent(files map[string][]byte, p string) string {
	if b, ok := files[p]; ok {
		return digest(b)
	}
	return ""
}
