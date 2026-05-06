package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "8" {
		t.Fatalf("Phase() = %q, want 8", got)
	}
}
