// Package decksh is a little language that generates deck markup
// parsing
package decksh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
)

// eval evaluates an id string
func eval(s string) string {
	v, ok := emap[s]
	if ok {
		return v
	}
	return s
}

// isaop tests for assignment operators
func isaop(s []string) bool {
	if len(s) < 4 {
		return false
	}
	op := s[1]
	if (op == "+" || op == "-" || op == "*" || op == "/") && s[2] == "=" {
		return true
	}
	return false
}

// parse takes a line of input and returns a string slice containing the parsed tokens
func parse(src string) []string {
	var s scanner.Scanner
	s.Init(strings.NewReader(src))

	tokens := []string{}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		tokens = append(tokens, s.TokenText())
	}
	for i := 1; i < len(tokens); i++ {
		tokens[i] = eval(tokens[i])
	}
	return tokens
}

// filequote gets a name from a quoted string
func filequote(s string, linenumber int) (string, error) {
	end := len(s) - 1
	if len(s) < 3 || s[0] != doublequote && s[end] != doublequote {
		return "", fmt.Errorf("line %d: %v is not a valid filename", linenumber, s)
	}
	return s[1:end], nil
}

// include inserts the contents of a file
// include "file"
func include(w io.Writer, s []string, linenumber int) error {
	if len(s) != 2 {
		return fmt.Errorf("line %d: include \"file\"", linenumber)
	}

	filearg, err := filequote(s[1], linenumber)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	r, err := os.Open(filearg)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	defer r.Close()
	return Process(w, r)
}

// loadata creates a file using the  data keyword
// data "file"...edata
func loadata(s []string, linenumber int, scanner *bufio.Scanner) error {
	if len(s) != 2 {
		return fmt.Errorf("line %d: data \"file\"...edata", linenumber)
	}
	filearg, err := filequote(s[1], linenumber)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	dataw, err := os.Create(filearg)
	if err != nil {
		return fmt.Errorf("line %d: %v (%v)", linenumber, s, err)
	}
	for scanner.Scan() {
		t := scanner.Text()
		if strings.TrimSpace(t) == "edata" {
			break
		}
		f := strings.Fields(t)
		if len(f) != 2 {
			continue
		}
		fmt.Fprintf(dataw, "%v\t%v\n", f[0], f[1])
	}
	err = dataw.Close()
	return err
}

// grid places objects read from a file in a grid
// grid "file" x y xint yint xlimit
func grid(w io.Writer, s []string, linenumber int) error {
	if len(s) < 7 {
		return fmt.Errorf("line %d: %s \"file\" x y xint yint xlimit", linenumber, s[0])
	}
	x, err := strconv.ParseFloat(eval(s[2]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	y, err := strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	xint, err := strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	yint, err := strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	limit, err := strconv.ParseFloat(eval(s[6]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	filearg, err := filequote(s[1], linenumber)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	r, err := os.Open(filearg)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	xp, yp := x, y
	for scanner.Scan() {
		if xp > limit {
			xp = x
			yp -= yint
		}
		t := scanner.Text()
		if len(t) == 0 {
			continue
		}
		keyparse(w, subxy(t, xp, yp), t, linenumber)
		xp += xint
	}
	return scanner.Err()
}

// subxy replaces the "x" and "y" arguments with the named values
func subxy(s string, x, y float64) []string {
	args := parse(s)
	if len(args) < 3 {
		return nil
	}
	for i := range args {
		if args[i] == "x" || args[i] == "X" {
			args[i] = ftoa(x)
		}
		if args[i] == "y" || args[i] == "Y" {
			args[i] = ftoa(y)
		}
	}
	return args
}

var funcmap = map[string]string{}

// import loads the definition of a function from a file
// import "file"
func importfunc(s []string, linenumber int) error {
	if len(s) < 2 {
		return fmt.Errorf("line %d: import \"file\"", linenumber)
	}
	filearg, err := filequote(s[1], linenumber)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	_, err = funcbody(filearg)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	return nil
}

// funcbody caches the body of function defintions
// avoiding reapeted open/read/close
func funcbody(s string) (string, error) {
	_, loaded := funcmap[s]
	var name string
	// if the name is not loaded, read from the file
	if !loaded {
		data, err := os.ReadFile(s)
		if err != nil {
			return "", err
		}
		fs := string(data)
		f := strings.Fields(fs)
		if len(f) > 2 && f[0] == "def" {
			name = f[1]
			funcmap[name] = fs
		}
	}
	return name, nil
}

// def reads and processes the function defintion
// def name arg1 arg2 ... argn
// ...
// edef
func def(scanner *bufio.Scanner, w io.Writer, s []string, filearg string, argoffset int, linenumber int) error {
	n := 0
	for scanner.Scan() {
		t := scanner.Text()
		if len(t) == 0 {
			continue
		}
		if t == "edef" {
			break
		}
		// the first line defines the arguments
		// def name arg1 arg2 ... argn
		if n == 0 {
			fargs := strings.Fields(t)
			if fargs[0] != "def" || len(fargs) < 3 {
				return fmt.Errorf("line %d: %q, begin function definition with 'def name args...'", linenumber, filearg)
			}
			fargs = fargs[2:]
			if len(fargs) != len(s)-argoffset {
				return fmt.Errorf("line %d: the number of arguments do not match: (%v=%d and %v=%d)", linenumber, fargs, len(fargs), s[argoffset:], len(s)-argoffset)
			}
			// copy the callers arguments
			for i := 0; i < len(fargs); i++ {
				emap[fargs[i]] = eval(s[i+argoffset])
			}
			keyparse(w, fargs, t, linenumber)
		} else {
			tokens := parse(t)
			if len(tokens) < 1 || t[0] == '#' {
				continue
			}
			if tokens[0] == "for" {
				parsefor(w, tokens, n, scanner)
				continue
			}
			if tokens[0] == "if" {
				parseif(w, t, n, scanner)
				continue
			}
			keyparse(w, tokens, t, linenumber)
		}
		n++
	}
	return scanner.Err()
}

// subfunc handles argument substitution in a function
// func "file" arg1 [arg2] [argn]
func subfunc(w io.Writer, s []string, linenumber int) error {
	if len(s) < 3 {
		return fmt.Errorf("line %d: %s \"file\" arg1 arg2...argn", linenumber, s[0])
	}
	filearg, err := filequote(s[1], linenumber)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	name, ferr := funcbody(filearg)
	if ferr != nil {
		return fmt.Errorf("line %d: %v", linenumber, ferr)
	}
	scanner := bufio.NewScanner(strings.NewReader(funcmap[name]))
	return def(scanner, w, s, filearg, 2, linenumber)
}

// directfunc calls a previously imported function
// function args...
func directfunc(w io.Writer, s []string, linenumber int) error {
	//fmt.Fprintf(os.Stderr, "line %d, directfunc=%v\n", linenumber, s)
	if len(s) < 2 {
		return fmt.Errorf("line %d: %v, need at least one argument for a function", linenumber, s)
	}
	scanner := bufio.NewScanner(strings.NewReader(funcmap[s[0]]))
	return def(scanner, w, s, s[0], 1, linenumber)
}

// evalif evaluates an if statement
func evalif(t string, n int) (bool, error) {
	tk := strings.Fields(t)
	r, err := condition(tk, n)
	return r, err
}

// parseblock parses a block of code
func parseblock(w io.Writer, data []string, linenumber int) error {
	for _, t := range data {
		err := keyparse(w, parse(t), t, linenumber)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
		linenumber++
	}
	return nil
}

// parseif parses if -- else -- eif
func parseif(w io.Writer, t string, linenumber int, scanner *bufio.Scanner) error {
	r, everr := evalif(t, linenumber)
	if everr != nil {
		return everr
	}
	ifblock, elseblock, blkerr := ifelse(scanner)
	if blkerr != nil {
		return blkerr
	}
	if r {
		return parseblock(w, ifblock, linenumber)
	} else {
		return parseblock(w, elseblock, linenumber)
	}
}

// ifelse returns if and (possibly empty) else blocks
func ifelse(scanner *bufio.Scanner) ([]string, []string, error) {
	inelse := false
	elseblock := []string{}
	ifblock := []string{}
	for scanner.Scan() {
		t := scanner.Text()
		s := strings.TrimSpace(t)
		if s == "eif" {
			break
		}
		if s == "else" {
			inelse = true
			continue
		}
		if inelse {
			elseblock = append(elseblock, t)
		} else {
			ifblock = append(ifblock, t)
		}
	}
	//fmt.Fprintf(os.Stderr, "ifblock=%v\telseblock=%v\n", ifblock, elseblock)
	return ifblock, elseblock, scanner.Err()
}

// keyparse parses keywords and executes
func keyparse(w io.Writer, tokens []string, t string, n int) error {
	// fmt.Fprintf(os.Stderr, "%v\n", len(tokens))
	if len(tokens) < 1 {
		return nil
	}

	if len(tokens) > 1 && tokens[1] == "=" {
		return assign(tokens, n)
	}
	if isaop(tokens) {
		return assignop(tokens, n)
	}

	switch tokens[0] {
	case "deck", "doc":
		return deck(w, tokens, n)

	case "dump":
		return Dump(t)

	case "canvas":
		return canvas(w, tokens, n)

	case "ruler":
		return ruler(w, tokens, n)

	case "include":
		return include(w, tokens, n)

	case "import":
		return importfunc(tokens, n)

	case "call", "func", "callfunc":
		return subfunc(w, tokens, n)

	case "slide", "page":
		return slide(w, tokens, n)

	case "grid":
		return grid(w, tokens, n)

	case "content":
		return content(w, tokens, n)

	case "text", "btext", "ctext", "etext", "textfile":
		return text(w, tokens, n)

	case "arctext":
		return arctext(w, tokens, n)

	case "rtext":
		return rtext(w, tokens, n)

	case "textblock", "textbox":
		return textblock(w, tokens, n)

	case "textblockfile", "textboxfile":
		return textblockfile(w, tokens, n)

	case "textcode":
		return textcode(w, tokens, n)

	case "image":
		return image(w, tokens, n)

	case "cimage":
		return cimage(w, tokens, n)

	case "list", "blist", "nlist", "clist":
		return list(w, tokens, n)

	case "elist", "eslide", "edeck", "edoc", "epage":
		return endtag(w, tokens, n)

	case "li":
		return listitem(w, tokens, n)

	case "ellipse", "rect":
		return shapes(w, tokens, n)

	case "circle", "square", "acircle":
		return regshapes(w, tokens, n)

	case "rrect", "roundrect":
		return roundrect(w, tokens, n)

	case "pill":
		return pill(w, tokens, n)

	case "star":
		return star(w, tokens, n)

	case "polygon", "poly":
		return polygon(w, tokens, n)

	case "polyline":
		return polyline(w, tokens, n)

	case "line":
		return line(w, tokens, n)

	case "arc":
		return arc(w, tokens, n)

	case "curve":
		return curve(w, tokens, n)

	case "legend":
		return legend(w, tokens, n)

	case "arrow":
		return arrow(w, tokens, n)

	case "lbrace", "rbrace", "ubrace", "dbrace":
		return brace(w, tokens, n)

	case "lbracket", "rbracket", "ubracket", "dbracket":
		return bracket(w, tokens, n)

	case "lcarrow", "rcarrow", "ucarrow", "dcarrow":
		return carrow(w, tokens, n)

	case "vline":
		return vline(w, tokens, n)

	case "hline":
		return hline(w, tokens, n)

	case "dchart", "chart":
		return chart(w, t, n)

	case "barchart", "linechart", "scatterchart", "hbarchart", "wbarchart", "dotchart", "areachart", "area", "dot", "wbar", "hbar", "scatter":
		return stdchart(w, tokens, n)

	case "pmap", "piechart", "donutchart", "pie", "donut", "lego", "pgrid", "waffle":
		return pchart(w, tokens, n)

	case "fanchart", "fan", "bow", "bowtie":
		return fanchart(w, tokens, n)

	case "georegion", "geopoly":
		return geopoly(w, tokens, n)

	case "geoborder", "geoline":
		return geoline(w, tokens, n)

	case "geolabel":
		return geolabel(w, tokens, n)

	case "geoloc":
		return geoloc(w, tokens, n)

	case "geomark", "geopoint":
		return geopoint(w, tokens, n)

	case "geopathfile":
		return geopathfile(w, tokens, n)

	case "geopath":
		return geopath(w, tokens, n)

	case "geoarc":
		return geoarc(w, tokens, n)

	case "geoimage":
		return geoimage(w, tokens, n)

	default: // not a keyword, direct function calls
		return directfunc(w, tokens, n)
	}
}
