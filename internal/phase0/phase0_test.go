package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "0" {
		t.Fatalf("Phase() = %q, want 0", got)
	}
}
