// dshlint - linter for .dsh (decksh) files
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/scanner"
)

const (
	maxbufsize  = 256 * 1024 // the default 64k buffer is too small
	argerrfmt   = "line %d: %v should have at least %d arguments, you have %d %v\n"
	blockerrfmt = "The count of %s (%d) does not match the count of %s (%d)\n"
	listerrfmt  = "Number of lists (%d) does not match elist count (%d)\n"
)

const (
	statementType int = iota
	Blank
	Comment
	Keyword
	Var
	AssignOp
)

// keyword/argument counts
var kwargcount = map[string]int{
	// structure
	"deck":    0,
	"edeck":   0,
	"include": 1,
	"slide":   0,
	"eslide":  0,
	"canvas":  2,
	"content": 1,
	"dump":    0,
	"grid":    6,

	// special assignments
	"format": 2,
	"random": 2,
	"substr": 3,
	"vmap":   5,

	// graphics
	"acircle":  3,
	"arc":      6,
	"ellipse":  4,
	"circle":   3,
	"curve":    6,
	"hline":    3,
	"line":     4,
	"pill":     4,
	"polygon":  2,
	"polyline": 2,
	"rect":     4,
	"rrect":    5,
	"square":   3,
	"star":     5,
	"vline":    3,
	"area":     2,
	"legend":   6,
	"dchart":   1,

	// arrows
	"arrow":   4,
	"lcarrow": 6,
	"ucarrow": 6,
	"dcarrow": 6,
	"rcarrow": 6,

	// lists
	"blist": 3,
	"clist": 3,
	"list":  3,
	"nlist": 3,
	"li":    0,
	"elist": 0,

	// text
	"arctext":       7,
	"btext":         4,
	"ctext":         4,
	"etext":         4,
	"rtext":         5,
	"text":          4,
	"textblock":     5,
	"textblockfile": 5,
	"textcode":      5,
	"textfile":      4,

	// images
	"cimage": 6,
	"image":  5,

	// data
	"data":  1,
	"edata": 0,

	// definitions
	"def":    1,
	"edef":   0,
	"func":   2,
	"import": 1,

	// conditionals
	"if":   1,
	"else": 0,
	"eif":  0,

	// loop
	"for":  0,
	"efor": 0,

	// braces and brackets
	"dbrace":   5,
	"dbracket": 4,
	"lbrace":   5,
	"lbracket": 4,
	"ubrace":   5,
	"ubracket": 4,
	"rbrace":   5,
	"rbracket": 4,

	// polar coordinates
	"polar":  4,
	"polarx": 4,
	"polary": 4,

	// math functions
	"cosine":  1,
	"sine":    1,
	"sqrt":    1,
	"tangent": 1,
}
var kwcount = map[string]int{}

// kwcouter count keywords
func kwcounter(data [][]string) {
	for i := 0; i < len(data); i++ {
		line := data[i]
		if kind(line) == Keyword {
			kwcount[line[0]]++
		}
	}
}

// kwcheck checks for matching keywords
func kwcheck() int {
	issues := 0
	for _, s := range []string{"deck", "slide", "if", "for", "data", "def"} {
		if kwcount[s] != kwcount["e"+s] {
			fmt.Fprintf(os.Stderr, blockerrfmt, s, kwcount[s], "e"+s, kwcount["e"+s])
			issues++
		}
	}
	nlists := 0
	for _, s := range []string{"list", "clist", "nlist", "blist"} {
		nlists += kwcount[s]
	}
	if nlists != kwcount["elist"] {
		fmt.Fprintf(os.Stderr, listerrfmt, nlists, kwcount["elist"])
		issues++
	}
	return issues
}

// kind returns the type of statement
func kind(s []string) int {
	if len(s) == 0 {
		return Blank
	}
	if len(s) == 1 && s[0][0] == '/' && s[0][1] == '/' {
		return Comment
	}
	if len(s) > 2 && s[1] == "=" {
		return Var
	}
	if len(s) > 3 && s[2] == "=" && s[0] != "if" && s[0] != "for" {
		return AssignOp
	}
	return Keyword
}

// readDecks reads decksh lines from a io.Reader, parsing them into lines
// blank lines are preserved.
func readDecksh(r io.Reader) [][]string {
	var data [][]string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxbufsize), maxbufsize)
	for scanner.Scan() {
		tokens := parse(scanner.Text())
		data = append(data, tokens)
	}
	return data
}

// parse takes a line of input and returns a string slice containing the parsed tokens
func parse(src string) []string {
	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Mode = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanChars |
		scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments
	tokens := []string{}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		tokens = append(tokens, s.TokenText())
	}
	return tokens
}

// lint tests for proper keyword arguments
func lint(data [][]string) int {
	issues := 0
	for i := 0; i < len(data); i++ {
		if kind(data[i]) != Keyword {
			continue
		}
		keyword := data[i][0]
		kwn := kwargcount[keyword]
		ln := len(data[i]) - 1
		if ln < kwn {
			fmt.Fprintf(os.Stderr, argerrfmt, i+1, keyword, kwn, ln, data[i])
			issues++
		}
	}
	return issues
}

// process processes a io.Reader of decksh code
func process(r io.Reader) int {
	issues := 0
	data := readDecksh(r) // read the data
	kwcounter(data)       // count keywords and elements
	issues += lint(data)  // check argument counts
	issues += kwcheck()   // integrity check
	return issues
}

// lint named files or standard input if no file is specified.
func main() {
	args := os.Args
	la := len(args)

	if la < 2 {
		os.Exit(process(os.Stdin))
	}

	issues := 0
	for i := 1; i < la; i++ {
		input, err := os.Open(args[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		issues = process(input)
		if issues > 0 {
			fmt.Fprintf(os.Stderr, "%d issues for %v\n", issues, args[i])
		}
	}
	os.Exit(issues)
}
