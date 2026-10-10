//go:build sh1extra

package session

import "testing"

func TestTagged(t *testing.T) {
	if Accept("stranger") {
		t.Fatal("stranger admitted")
	}
}
