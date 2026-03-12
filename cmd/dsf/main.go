package main

import (
	"fmt"
	"os"

	"github.com/ajstarks/decksh"
)

func main() {
	if err := decksh.Process(os.Stdout, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
