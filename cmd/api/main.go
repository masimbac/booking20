package main

import (
	"fmt"
	"os"

	"github.com/parama/booking/internal/phase0"
)

func main() {
	fmt.Fprintf(os.Stdout, "booking api bootstrap (%s)\n", phase0.Phase())
	os.Exit(0)
}
