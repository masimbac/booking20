package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "5" {
		t.Fatalf("Phase() = %q, want 5", got)
	}
}
