package session

import (
	"os"
	"strings"
	"testing"

	"example.org/sh1-fixture/internal/session/internal/helper"
)

func TestAdmission(t *testing.T) {
	for _, id := range []string{helper.Identity(), "operator"} {
		if !Accept(id) {
			t.Fatalf("denied %s", id)
		}
	}
	for _, id := range []string{"", "stranger", "member-extra"} {
		if Accept(id) {
			t.Fatalf("admitted %q", id)
		}
	}
}

func TestGolden(t *testing.T) {
	b, err := os.ReadFile("testdata/identity.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != helper.Identity() {
		t.Fatal("golden mismatch")
	}
}

func TestSourceGuard(t *testing.T) {
	b, err := os.ReadFile("session.go")
	if err != nil {
		t.Fatal(err)
	}
	// An explicit marker makes runtime target coverage measurable.
	t.Log("SH1_GUARD session.go")
	if !strings.Contains(string(b), `identity == "member"`) {
		t.Fatal("member guard missing")
	}
}

func TestOptIn(t *testing.T) {
	if os.Getenv("SH1_FIXTURE_OPT_IN") != "1" {
		t.Skip("SH1_FIXTURE_OPT_IN=1 required")
	}
	if !Accept("operator") {
		t.Fatal("operator denied")
	}
}
