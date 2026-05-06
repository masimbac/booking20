package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "2" {
		t.Fatalf("Phase() = %q, want 2", got)
	}
}
