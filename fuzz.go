// +build gofuzz

package decksh

import (
	"bytes"
	"os"
)

func Fuzz(data []byte) int {
	err := Process(os.Stdout, bytes.NewReader(data))
	if err != nil {
		return 0
	}
	return 1
}