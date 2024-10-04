// dshfmt - format .dsh (decksh) files
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"text/scanner"
)

const (
	maxbufsize = 256 * 1024 // the default 64k buffer is too small
)

const (
	matchfmt    = "The count of %s (%d) does not match the count of %s (%d). Check lines: "
	diffmt      = "%d unmatched list(s).\n"
	dumpfmt     = "%-4d: (%d) %v\n"
	dumpheadfmt = "%-4s %4s TOKENS\n"
	sortheadfmt = "%-*s: %s %s\n"
	maxfmt      = "kwmax=%d strmax=%d varmax=%d spacer=%q\n"
)

const (
	statementType int = iota
	Blank
	Comment
	Keyword
	Var
	AssignOp
)

type keywordInfo map[string][]int

// kwcouter count keywords occurances
func kwcounter(data [][]string) keywordInfo {
	var ki = keywordInfo{}
	count := 0
	for i := 0; i < len(data); i++ {
		line := data[i]
		if kind(line) == Keyword {
			count++
			if count > 0 {
				ki[line[0]] = append(ki[line[0]], i+1)
			}
		}
	}
	return ki
}

// matcherr displays unmatched keyword errors
func matcherr(ki keywordInfo, s string, c1, c2 int) {
	fmt.Fprintf(os.Stderr, matchfmt, s, c1, "e"+s, c2)
	list := ki[s]
	ll := len(list) - 1
	for i := 0; i < ll; i++ {
		fmt.Fprintf(os.Stderr, "%d, ", list[i])
	}
	fmt.Fprintf(os.Stderr, "%d\n", list[ll])
}

// listerr displays unmatched list errors
func listerr(ki keywordInfo, diff int) {
	fmt.Fprintf(os.Stderr, diffmt, diff)
	for _, s := range []string{"list", "blist", "clist", "nlist"} {
		fmt.Fprintf(os.Stderr, "    %-5s found in lines :%v\n", s, ki[s])
	}
}

// check checks the structural integrity by analyzing keywords
func intcheck(ki keywordInfo) int {
	issues := 0
	for _, s := range []string{"deck", "slide", "if", "for", "data", "def"} {
		c1 := len(ki[s])
		c2 := len(ki["e"+s])
		if c1 != c2 {
			matcherr(ki, s, c1, c2)
			issues++
		}
	}
	nlists := 0
	for _, s := range []string{"list", "blist", "clist", "nlist"} {
		nlists += len(ki[s])
	}
	diff := nlists - len(ki["elist"])
	if diff > 0 {
		listerr(ki, diff)
		issues++
	}
	return issues
}

// kind returns the type of statement
func kind(s []string) int {
	if len(s) == 0 {
		return Blank
	}
	if len(s) == 1 && (s[0][0] == '/' && s[0][1] == '/') || s[0][0] == '#' {
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
	if s[0] == "eslide" || s[0] == "deck" {
		fmt.Printf("%s\n", s[0])
	} else {
		fmt.Printf("%s", s[0])
	}
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
	if s[0] == "efor" || s[0] == "elist" || s[0] == "eif" {
		fmt.Printf("%-*s\n", kwmax, s[0])
	} else {
		fmt.Printf("%-*s", kwmax, s[0])
	}
	printargs(1, s)
}

// listitem formats a list item
func listitem(level, max int, spacer string, s []string) {
	printlevel(level, spacer)
	fmt.Printf("%-*s", max-len(spacer), s[0])
	printargs(1, s)
}

// conditional formats if statements
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
func format(s [][]string, ki keywordInfo, kwmax, strmax, assmax int, spacer string) {
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

// dump prints the parsed lines
func dump(data [][]string) {
	fmt.Fprintf(os.Stderr, dumpheadfmt, "LINE", "LEN")
	for i := 0; i < len(data); i++ {
		fmt.Fprintf(os.Stderr, dumpfmt, i+1, len(data[i]), data[i])
	}
	fmt.Fprintln(os.Stderr)
}

// sort keywords by occurance
func sortkwfreq(ki keywordInfo, kwlen int) {
	type kv struct {
		Key   string
		Value []int
	}

	var ss []kv
	for k, v := range ki {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return len(ss[i].Value) > len(ss[j].Value)
	})

	fmt.Fprintf(os.Stderr, sortheadfmt, kwlen, "CMD", "FRQ", "LINES")
	for _, kv := range ss {
		fmt.Fprintf(os.Stderr, "%-*s: (%d) %v\n", kwlen, kv.Key, len(kv.Value), kv.Value)
	}
	fmt.Fprintln(os.Stderr)
}

// sort keywords by name
func sortkw(ki keywordInfo, kwlen int) {
	keys := make([]string, 0, len(ki))
	for k := range ki {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintf(os.Stderr, sortheadfmt, kwlen, "CMD", "FRQ", "LINES")
	for _, k := range keys {
		fmt.Fprintf(os.Stderr, "%-*s: (%d) %v\n", kwlen, k, len(ki[k]), ki[k])
	}
	fmt.Fprintln(os.Stderr)
}

// dshread reads data from stdin or named file
func dshread(args []string) [][]string {
	var input io.Reader
	var err error
	input = os.Stdin
	if len(args) > 0 {
		input, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
	return rd(input)
}

func diag(data [][]string, kwlen int, ki keywordInfo, all, rawdump, ksort, fsort bool) {

	if all {
		rawdump, ksort, fsort = true, true, true
	}

	if rawdump {
		dump(data)
	}
	if ksort {
		sortkw(ki, kwlen)
	}
	if fsort {
		sortkwfreq(ki, kwlen)
	}
}

// format a named file or standard input if no file is specified.
func main() {
	verbose := flag.Bool("v", false, "verbose diagnostics")
	rawdump := flag.Bool("dump", false, "dump raw parsed data")
	kwsort := flag.Bool("ksort", false, "show keyword counts")
	freqsort := flag.Bool("fsort", false, "show keyword frequencies")
	spacer := flag.String("i", "\t", "indent")
	flag.Parse()

	data := dshread(flag.Args())                                      // read data
	kwinfo := kwcounter(data)                                         // count keywords and elements
	kwmax := maxitem(data, 0, 1)                                      // max keyword length
	strmax := maxitem(data, 1, 2)                                     // max string argument length
	varmax := maxvar(data)                                            // max variable
	diag(data, kwmax, kwinfo, *verbose, *rawdump, *kwsort, *freqsort) // if specified, show various diagnostics
	issues := intcheck(kwinfo)                                        // check for issues
	if issues == 0 {                                                  // if no issues, format
		format(data, kwinfo, kwmax, strmax, varmax, *spacer)
	}
	os.Exit(issues)
}
