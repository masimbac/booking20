package phase0

import "testing"

func TestPhase(t *testing.T) {
	if got := Phase(); got != "9" {
		t.Fatalf("Phase() = %q, want 9", got)
	}
}
