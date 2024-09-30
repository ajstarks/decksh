// dshfmt - format .dsh (decksh) files
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/scanner"
)

const (
	maxbufsize = 256 * 1024 // the default 64k buffer is too small
)

var kwcount = map[string]int{}

const (
	statementType int = iota
	Blank
	Comment
	Keyword
	Var
	AssignOp
)

// kwcouter count keywords
func kwcounter(data [][]string) {
	for i := 0; i < len(data); i++ {
		line := data[i]
		if kind(line) == Keyword {
			kwcount[line[0]]++
		}
	}
}

// kwmatch checks for matching pairs
func kwmatch(s string) bool {
	end := "e" + s
	if kwcount[s] != kwcount[end] {
		return false
	}
	return true
}

// kwcheck checks for matching keywords
func kwcheck() int {
	issues := 0
	for _, s := range []string{"deck", "slide", "if", "for", "data", "def"} {
		if kwcount[s] != kwcount["e"+s] {
			fmt.Fprintf(os.Stderr, "The count of %s (%d) does not match the count of %s (%d)\n",
				s, kwcount[s], "e"+s, kwcount["e"+s])
			issues++
		}
	}
	nlists := 0
	for _, s := range []string{"list", "clist", "nlist", "blist"} {
		nlists += kwcount[s]
	}
	if nlists != kwcount["elist"] {
		fmt.Fprintf(os.Stderr, "Number of lists (%d) does not match elist count (%d)\n", nlists, kwcount["elist"])
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

// printlevel prints the leading spaces for the specified level
func printlevel(level int, spacer string) {
	for i := 0; i < level; i++ {
		fmt.Printf(spacer)
	}
}

// printargs prints arguments from the specified point to the last
func printargs(n int, s []string) {
	for i := n; i < len(s); i++ {
		fmt.Printf(" %s", s[i])
	}
	fmt.Printf("\n")
}

// dchart formats a dchart line
func dchart(level, max int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%-*s ", max, s[0])
	for i := 1; i < len(s)-1; i++ {
		if s[i] == "-" {
			fmt.Printf("-")
		} else {
			fmt.Printf("%s ", s[i])
		}
	}
	fmt.Printf("%s\n", s[len(s)-1])
}

// comment formats a comment
func comment(level int, spacer string, s []string) {
	printlevel(level, spacer)
	printargs(1, s)
}

// toplevel formats top level elements
func toplevel(level int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%s", s[0])
	printargs(1, s)
}

// stringarg formats a line with keyword followed by a string
func stringarg(level, kwmax, smax int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%-*s %-*s", kwmax, s[0], smax, s[1])
	printargs(2, s)
}

// keyword formats a general keyword
func keyword(level, varmax, kwmax int, spacer string, s []string) {
	if kind(s) == AssignOp {
		printlevel(level, spacer)
		fmt.Printf("%-*s %s%s", varmax, s[0], s[1], s[2])
		printargs(3, s)
		return
	}
	// assigments
	if kind(s) == Var {
		printlevel(level, spacer)
		fmt.Printf("%-*s %s", varmax, s[0], s[1])
		printargs(2, s)
		return
	}
	// keywords
	printlevel(level, spacer)
	fmt.Printf("%-*s", kwmax, s[0])
	printargs(1, s)
}

// listitem formats a list item
func listitem(level, max int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%-*s", max-len(spacer), s[0])
	printargs(1, s)
}

func conditional(level int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%s %s ", s[0], s[1])
	switch len(s) {
	case 4:
		fmt.Printf("%s %s\n", s[2], s[3])
	case 5:
		fmt.Printf("%s%s %s\n", s[2], s[3], s[4])
	case 6:
		fmt.Printf("%s%s %s %s\n", s[2], s[3], s[4], s[5])
	}
}

// format formats a series of decksh lines (each one is a parsed string slice)
func format(s [][]string, kwmax, strmax, assmax int, spacer string) {
	if kwmax > assmax {
		assmax = kwmax
	}
	level := 0
	for i := 0; i < len(s); i++ {
		line := s[i]
		if kind(line) == Blank {
			fmt.Printf("\n")
			continue
		}
		if kind(line) == Comment {
			printlevel(1, spacer)
			fmt.Printf("%s\n", line[0])
			continue
		}
		// process keywords
		switch line[0] {
		case "deck", "edeck", "def", "edef":
			level = 0
			toplevel(level, spacer, line)
			level++
		case "slide", "eslide", "import", "include":
			level = 1
			toplevel(level, spacer, line)
			level++
		case "text", "ctext", "etext", "btext", "rtext", "arctext", "image", "textblock":
			stringarg(level, kwmax, strmax, spacer, line)
		case "for", "clist", "list", "blist", "nlist", "else":
			level = 2
			keyword(level, assmax, kwmax, spacer, line)
			level++
		case "if":
			level = 2
			conditional(level, spacer, line)
			level++
		case "efor", "elist", "eif":
			level--
			keyword(level, assmax, kwmax, spacer, line)
		case "li":
			level = 3
			listitem(level, assmax, spacer, line)
		case "dchart", "chart":
			level = 2
			dchart(level, kwmax, spacer, line)
		default:
			keyword(level, assmax, kwmax, spacer, line)
		}
	}
}

// maxitem returns the maximum element within a collection of decksh lines
// between the <begin> and <end> elements
func maxitem(s [][]string, begin, end int) int {
	max := 0
	for i := 0; i < len(s); i++ {
		line := s[i]
		if len(line) <= 1 {
			continue
		}
		for j := begin; j < end; j++ {
			ll := len(line[j])
			if ll > max {
				max = ll
			}
		}
	}
	return max
}

// maxvar returns the maximum length element within an assignment line
func maxvar(s [][]string) int {
	max := 0
	for i := 0; i < len(s); i++ {
		line := s[i]
		for j := 0; j < len(line); j++ {
			if j == 1 && line[j] == "=" {
				ll := len(line[0])
				if ll > max {
					max = ll
				}
			}
		}
	}
	return max
}

// rd reads decksh lines from a io.Reader, parsing them into lines
func rd(r io.Reader) [][]string {
	var s scanner.Scanner
	s.Init(r)
	s.Mode ^= scanner.SkipComments
	var data [][]string
	var line []string
	var t string
	var index, next int

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		index = s.Position.Line - 1
		if index-next >= 1 { // new line
			data = append(data, line)
			line = []string{}
		}
		// build the line
		t = s.TokenText()
		line = append(line, t)
		next = index
	}
	data = append(data, line) // last line
	return data
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

// dump prints the parsed lines
func dump(data [][]string) {
	for i := 0; i < len(data); i++ {
		fmt.Fprintf(os.Stderr, "%d: %v (%d elements)\n", i+1, data[i], len(data[i]))
	}
}

// format a named file or standard input if no file is specified.
func main() {
	var spacer string
	var verbose bool

	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.StringVar(&spacer, "i", "\t", "indent")
	flag.Parse()

	input := os.Stdin
	if len(flag.Args()) == 1 {
		var err error
		input, err = os.Open(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}

	data := readDecksh(input)     // read the data
	kwcounter(data)               // count keywords and elements
	kwmax := maxitem(data, 0, 1)  // max keyword length
	strmax := maxitem(data, 1, 2) // max string argument length
	varmax := maxvar(data)        // max variable

	format(data, kwmax, strmax, varmax, spacer)

	if verbose {
		dump(data)
		for k, v := range kwcount {
			fmt.Fprintf(os.Stderr, "%-*s:%d\n", kwmax, k, v)
		}
		fmt.Fprintf(os.Stderr, "kwmax=%d strmax=%d varmax=%d spacer=%q\n",
			kwmax, strmax, varmax, spacer)
	}

	os.Exit(kwcheck()) // integrity check
}
