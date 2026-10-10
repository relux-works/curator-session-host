package child

import "testing"

func TestParent(t *testing.T) {
	if !Allowed() {
		t.Fatal("parent gate denied member")
	}
}
