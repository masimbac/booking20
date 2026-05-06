package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "3" {
		t.Fatalf("Phase() = %q, want 3", got)
	}
}
