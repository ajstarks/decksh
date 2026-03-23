// dshfmt - format .dsh (decksh) files
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
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

var kwcount = map[string]int{}

const (
	statementType int = iota
	Blank
	Comment
	Keyword
	Var
	AssignOp
)

type options struct {
	verbose, writeback, rawdump, kwsort, freqsort bool
	spacer                                        string
}

type keywordInfo map[string][]int

// kwcouter count keywords occurances
func kwcounter(data [][]string) keywordInfo {
	var ki = keywordInfo{}
	count := 0
	for i := range data {
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

// matcherr displays unmatched keyword errors
func matcherr(ki keywordInfo, s string, c1, c2 int) {
	fmt.Fprintf(os.Stderr, matchfmt, s, c1, "e"+s, c2)
	list := ki[s]
	ll := len(list) - 1
	for i := range ll {
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
	if diff < 0 {
		issues++
	}
	return issues
}

// kind returns the type of statement
func kind(s []string) int {
	if len(s) == 0 {
		return Blank
	}
	if len(s) == 1 && len(s[0]) > 1 && s[0][0] == '/' && s[0][1] == '/' {
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
func printlevel(w io.Writer, level int, spacer string) {
	for range level {
		fmt.Fprintf(w, spacer)
	}
}

// printargs prints arguments from the specified point to the last
func printargs(w io.Writer, n int, s []string) {
	for i := n; i < len(s); i++ {
		fmt.Fprintf(w, " %s", s[i])
	}
	fmt.Fprintf(w, "\n")
}

// dchart formats a dchart line
func dchart(w io.Writer, level, max int, spacer string, s []string) {
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%-*s ", max, s[0])
	for i := 1; i < len(s)-1; i++ {
		if s[i] == "-" {
			fmt.Fprintf(w, "-")
		} else {
			fmt.Fprintf(w, "%s ", s[i])
		}
	}
	fmt.Fprintf(w, "%s\n", s[len(s)-1])
}

// comment formats a comment
func comment(w io.Writer, level int, spacer string, s []string) {
	printlevel(w, level, spacer)
	printargs(w, 1, s)
}

// toplevel formats top level elements
func toplevel(w io.Writer, level int, spacer string, s []string) {
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%s", s[0])
	printargs(w, 1, s)
}

// stringarg formats a line with keyword followed by a string
func stringarg(w io.Writer, level, kwmax, smax int, spacer string, s []string) {
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%-*s %-*s", kwmax, s[0], smax, s[1])
	printargs(w, 2, s)
}

// keyword formats a general keyword
func keyword(w io.Writer, level, varmax, kwmax int, spacer string, s []string) {
	if kind(s) == AssignOp {
		printlevel(w, level, spacer)
		fmt.Fprintf(w, "%-*s %s%s", varmax, s[0], s[1], s[2])
		printargs(w, 3, s)
		return
	}
	// assigments
	if kind(s) == Var {
		printlevel(w, level, spacer)
		fmt.Fprintf(w, "%-*s %s", varmax, s[0], s[1])
		printargs(w, 2, s)
		return
	}
	// keywords
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%-*s", kwmax, s[0])
	printargs(w, 1, s)
}

// listitem formats a list item
func listitem(w io.Writer, level, max int, spacer string, s []string) {
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%-*s", max-len(spacer), s[0])
	printargs(w, 1, s)
}

// conditional processes if...eif statements
func conditional(w io.Writer, level int, spacer string, s []string) {
	printlevel(w, level, spacer)
	fmt.Fprintf(w, "%s %s ", s[0], s[1])
	switch len(s) {
	case 4:
		fmt.Fprintf(w, "%s %s\n", s[2], s[3])
	case 5:
		fmt.Fprintf(w, "%s%s %s\n", s[2], s[3], s[4])
	case 6:
		fmt.Fprintf(w, "%s%s %s %s\n", s[2], s[3], s[4], s[5])
	default:
		fmt.Fprintf(os.Stderr, "unexpected length: %d\n", len(s))
	}
}

// format formats a series of decksh lines (each one is a parsed string slice)
func format(w io.Writer, s [][]string, kwmax, strmax, assmax int, spacer string) {
	if kwmax > assmax {
		assmax = kwmax
	}
	level := 0
	for i := range s {
		line := s[i]
		if kind(line) == Blank {
			fmt.Fprintf(w, "\n")
			continue
		}
		if kind(line) == Comment {
			printlevel(w, 1, spacer)
			fmt.Fprintf(w, "%s\n", line[0])
			continue
		}
		// process keywords
		switch line[0] {
		case "deck", "edeck", "def", "edef":
			level = 0
			toplevel(w, level, spacer, line)
			level++
		case "slide", "eslide", "import":
			level = 1
			toplevel(w, level, spacer, line)
			level++
		case "text", "ctext", "etext", "btext", "rtext", "arctext", "image", "textblock":
			stringarg(w, level, kwmax, strmax, spacer, line)
		case "for", "clist", "list", "blist", "nlist", "else":
			level = 2
			keyword(w, level, assmax, kwmax, spacer, line)
			level++
		case "if":
			level = 2
			conditional(w, level, spacer, line)
			level++
		case "efor", "elist", "eif", "edata":
			level--
			if level < 0 {
				level = 0
			}
			keyword(w, level, assmax, kwmax, spacer, line)
		case "li":
			level = 3
			listitem(w, level, assmax, spacer, line)
		case "dchart", "chart":
			level = 2
			dchart(w, level, kwmax, spacer, line)
		default:
			keyword(w, level, assmax, kwmax, spacer, line)
		}
	}
}

// maxitem returns the maximum element within a collection of decksh lines
// between the <begin> and <end> elements
func maxitem(s [][]string, begin, end int) int {
	max := 0
	for i := range s {
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
	for i := range s {
		line := s[i]
		for j := range line {
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

// readDecks reads decksh lines from a io.Reader, parsing them into lines
// blank lines are preserved.
func readDecksh(r io.Reader) ([][]string, error) {
	var data [][]string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxbufsize), maxbufsize)
	for scanner.Scan() {
		tokens := parse(scanner.Text())
		data = append(data, tokens)
	}
	return data, scanner.Err()
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

// diag shows specified diagnostics
func diag(data [][]string, kwlen int, ki keywordInfo, verbose, rawdump, ksort, fsort bool) {
	if verbose {
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

// dump prints the parsed lines
func dump(data [][]string) {
	for i := range data {
		fmt.Fprintf(os.Stderr, "%d: %v (%d elements)\n", i+1, data[i], len(data[i]))
	}
}

// make a temp file from a name
func tempfile(s string) string {
	return s + "!!!"
}

// setIO sets up the input and output
func setIO(args []string, writeback bool) (io.Writer, io.Reader, error) {
	var input io.Reader = os.Stdin
	var output io.Writer = os.Stdout
	var err error

	if len(args) == 1 {
		input, err = os.Open(args[0])
		if err != nil {
			return nil, nil, err
		}
	}
	if writeback && len(args) == 1 {
		output, err = os.Create(tempfile(args[0]))
		if err != nil {
			return nil, nil, err
		}
	}
	return output, input, nil
}

// rename the tempfile to the original
func writecopy(s string) {
	if len(s) > 0 {
		err := os.Rename(tempfile(s), s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

// process the input, make output
func process(args []string, opts options) error {
	wb := opts.writeback
	output, input, err := setIO(args, wb)
	if err != nil {
		return err
	}
	data, err := readDecksh(input) // read the data
	if err != nil {
		return err
	}
	kwinfo := kwcounter(data)     // count keywords and elements
	kwmax := maxitem(data, 0, 1)  // max keyword length
	strmax := maxitem(data, 1, 2) // max string argument length
	varmax := maxvar(data)        // max variable
	diag(data, kwmax, kwinfo, opts.verbose, opts.rawdump, opts.kwsort, opts.freqsort)
	issues := intcheck(kwinfo)
	if issues == 0 {
		format(output, data, kwmax, strmax, varmax, opts.spacer)
	}
	if wb && len(args) == 1 {
		writecopy(args[0])
	}
	if opts.verbose {
		dump(data)
		for k, v := range kwcount {
			fmt.Fprintf(os.Stderr, "%-*s:%d\n", kwmax, k, v)
		}
		fmt.Fprintf(os.Stderr, "kwmax=%d strmax=%d varmax=%d spacer=%q\n",
			kwmax, strmax, varmax, opts.spacer)
	}
	return nil
}

// format a named file or standard input if no file is specified.
func main() {
	var opts options
	flag.BoolVar(&opts.verbose, "v", false, "verbose")
	flag.BoolVar(&opts.writeback, "w", false, "write changes")
	flag.StringVar(&opts.spacer, "i", "\t", "indent")
	flag.BoolVar(&opts.rawdump, "dump", false, "dump raw parsed data")
	flag.BoolVar(&opts.kwsort, "ksort", false, "show keyword counts")
	flag.BoolVar(&opts.freqsort, "fsort", false, "show keyword frequencies")
	flag.Parse()

	if err := process(flag.Args(), opts); err != nil { // process
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	os.Exit(kwcheck()) // integrity check
}
