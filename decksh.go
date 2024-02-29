// Package decksh is a little language that generates deck markup
package decksh

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const (
	maxbufsize = 256 * 1024 // the default 64k buffer is too small
)

// emap is the id=expression map
var emap = map[string]string{"deckshVersion": `"2024-02-28-1.0.0"`}

var (
	canvasWidth  = 792.0
	canvasHeight = 612.0
)

// Process reads input, parses, dispatches functions for code generation
func Process(w io.Writer, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxbufsize), maxbufsize) // the default 64k buffer is too small
	errors := []error{}

	// For every line in the input, parse into tokens,
	// call the appropriate function, collecting errors as we go.
	// If any errors occurred, print them at the end, and return the latest
	for n := 1; scanner.Scan(); n++ {
		t := scanner.Text()
		tokens := parse(t)
		if len(tokens) < 1 || t[0] == '#' {
			continue
		}
		if tokens[0] == "for" {
			errors = append(errors, parsefor(w, tokens, n, scanner))
			continue
		}
		if tokens[0] == "data" {
			errors = append(errors, loadata(tokens, n, scanner))
			continue
		}
		if tokens[0] == "if" {
			errors = append(errors, parseif(w, t, n, scanner))
			continue
		}
		errors = append(errors, keyparse(w, tokens, t, n))
	}
	// report any collected errors
	nerrs := 0
	for _, e := range errors {
		if e != nil {
			nerrs++
			fmt.Fprintf(os.Stderr, "%v\n", e)
		}
	}

	// handle read errors from scanning
	if err := scanner.Err(); err != nil {
		return err
	}

	// return the latest error
	if nerrs > 0 {
		return errors[nerrs-1]
	}

	// all is well, no errors
	return nil
}
