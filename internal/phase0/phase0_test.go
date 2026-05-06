package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "1" {
		t.Fatalf("Phase() = %q, want 1", got)
	}
}
