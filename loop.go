// Package decksh is a little language that generates deck markup
// loops
package decksh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

// types of for loops
const (
	noloop = iota
	numloop
	fileloop
	vectloop
)

// fortype returns the type of for loop; either:
// for v = begin end incr
// for v = ["abc" "123"]
// for v = "file"
func fortype(s []string) int {
	n := len(s)
	// for x = ...
	if n < 4 || s[2] != "=" {
		return noloop
	}
	// for x = [...]
	if s[3] == "[" && s[len(s)-1] == "]" {
		return vectloop
	}
	// for x = "foo.d"
	if n == 4 && len(s[3]) > 3 && s[3][0] == doublequote && s[3][len(s[3])-1] == doublequote {
		return fileloop
	}
	// for x = begin end [increment]
	if n == 5 || n == 6 {
		return numloop
	}
	return noloop
}

// forvector returns the elements between "[" and "]"
func forvector(s []string) ([]string, error) {
	n := len(s)
	if n < 5 {
		return nil, fmt.Errorf("incomplete for: %v", s)
	}
	elements := make([]string, n-5)
	for i := 4; i < n-1; i++ {
		elements[i-4] = s[i]
	}
	return elements, nil
}

// forfile reads and returns the contents of the file in for x = "file"
func forfile(s []string) ([]string, error) {
	var contents []string
	fname := s[3][1 : len(s[3])-1] // remove quotes
	r, err := os.Open(fname)
	if err != nil {
		return contents, err
	}
	fs := bufio.NewScanner(r)
	for fs.Scan() {
		contents = append(contents, fs.Text())
	}
	return contents, fs.Err()
}

// fornum returns the arguments for for x=begin end [incr]
func fornum(s []string, linenumber int) (float64, float64, float64, error) {
	var incr float64
	if len(s) < 5 {
		return 0, -1, 0, fmt.Errorf("line %d: for begin end [incr] ... efor", linenumber)
	}

	begin, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return 0, -1, 0, err
	}
	end, err := strconv.ParseFloat(s[4], 64)
	if err != nil {
		return 0, -1, 0, err
	}

	incr = 1.0
	if len(s) > 5 {
		var ierr error
		incr, ierr = strconv.ParseFloat(s[5], 64)
		if ierr != nil {
			return 0, -1, 0, ierr
		}
	}
	return begin, end, incr, nil
}

// forbody collects items within a for loop body
func forbody(scanner *bufio.Scanner) [][]string {
	elements := [][]string{}
	for scanner.Scan() {
		p := parse(scanner.Text())
		if len(p) < 1 {
			continue
		}
		if p[0] == "efor" {
			break
		}
		elements = append(elements, p)
	}
	return elements
}

// parsefor collects and evaluates a loop body
func parsefor(w io.Writer, s []string, linenumber int, scanner *bufio.Scanner) error {
	forvar := s[1]
	body := forbody(scanner)
	// determine the type of loop
	switch fortype(s) {
	case numloop:
		begin, end, incr, err := fornum(s, linenumber)
		if err != nil {
			return err
		}
		for v := begin; v <= end; v += incr {
			for _, fb := range body {
				evaloop(w, forvar, "%s", ftoa(v), fb, linenumber)
			}
		}
		return err
	case vectloop:
		vl, err := forvector(s)
		if err != nil {
			return err
		}
		for _, v := range vl {
			for _, fb := range body {
				evaloop(w, forvar, "\"%s\"", v, fb, linenumber)
			}
		}
		return err
	case fileloop:
		fl, err := forfile(s)
		if err != nil {
			return err
		}
		for _, v := range fl {
			for _, fb := range body {
				evaloop(w, forvar, "\"%s\"", v, fb, linenumber)
			}
		}
		return err
	default:
		return fmt.Errorf("line %d: incorrect for loop: %v", linenumber, s)
	}
}

// evaloop evaluates items in a loop body
func evaloop(w io.Writer, forvar string, format string, v string, s []string, linenumber int) {
	e := make([]string, len(s))
	copy(e, s)
	for i := 0; i < len(s); i++ {
		if s[i] == forvar {
			e[i] = fmt.Sprintf(format, v)
		}
	}
	keyparse(w, e, "", linenumber)
}
