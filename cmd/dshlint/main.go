// dshlint - linter for .dsh (decksh) files
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
	maxbufsize  = 256 * 1024 // the default 64k buffer is too small
	kwfmt       = "%-75s %s\n"
	infofmt     = "%-15s %s\n"
	argerrfmt   = "line %d: %v -> %s should have at least %d arguments, you have %d\nUsage: %v\n"
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

// syntax defines the number of arguments and usage
type syntax struct {
	minargs int
	maxargs int
	desc    string
	usage   string
}

// keyword/argument counts, syntax
var kwInfo = map[string]syntax{
	"deck":          {minargs: 0, maxargs: 0, desc: "Begin a deck; end with \"edeck\"", usage: "deck"},
	"edeck":         {minargs: 0, maxargs: 0, desc: "End the deck", usage: "edeck"},
	"include":       {minargs: 1, maxargs: 1, desc: "Include the contents of a file", usage: "include \"file\""},
	"slide":         {minargs: 0, maxargs: 2, desc: "Begin a slide; end with \"eslide\"", usage: "slide [bgcolor] [fgcolor]"},
	"eslide":        {minargs: 0, maxargs: 0, desc: "End a slide", usage: "eslide (end slide)"},
	"canvas":        {minargs: 2, maxargs: 2, desc: "Define with dimensiond of the canvas", usage: "canvas width height"},
	"content":       {minargs: 1, maxargs: 1, desc: "Embed content", usage: "content \"scheme://file\""},
	"dump":          {minargs: 0, maxargs: 1, desc: "Dump varaibles", usage: "dump [name]"},
	"grid":          {minargs: 6, maxargs: 6, desc: "Define a content grid", usage: "grid \"file\" x y hspace vspace edge"},
	"area":          {minargs: 1, maxargs: 1, desc: "Assign an area", usage: "area expression"},
	"format":        {minargs: 2, maxargs: 7, desc: "Assign formatting to expressions", usage: "format \"format\" expr... (up to 5)"},
	"random":        {minargs: 2, maxargs: 2, desc: "Assign a random number between two values", usage: "random min max"},
	"substr":        {minargs: 3, maxargs: 3, desc: "Assign a substring", usage: "substr \"string\" begin end"},
	"vmap":          {minargs: 5, maxargs: 5, desc: "Assign a value mapped to two ranges", usage: "vmap data min1 max1 min2 max2"},
	"acircle":       {minargs: 3, maxargs: 5, desc: "Circle with sized based on area", usage: "arcicle x y w [color] [opacity]"},
	"arc":           {minargs: 6, maxargs: 9, desc: "Ellipical arc centered at (x,y), dimensions (w,h) between angles a1 and a2", usage: "arc x y w h a1 a2 [lw color opacity]"},
	"ellipse":       {minargs: 4, maxargs: 6, desc: "Ellipse centered at (x,y), dimension (w,h)", usage: "ellipse x y w h [color] [opacity]"},
	"circle":        {minargs: 3, maxargs: 5, desc: "Circle centered at (x,y), diameter w", usage: "circle x y w [color] [opacity]"},
	"curve":         {minargs: 6, maxargs: 9, desc: "Quadradic Bezier Curve begin (bx,by), control (cx, cy), end (ex,ey)", usage: "curve bx by cx cy ex ey [lw] [color] [opacity]"},
	"hline":         {minargs: 3, maxargs: 6, desc: "Horizontal line begin at (x,y), length w", usage: "hline x y w [lw] [color] [opacity]"},
	"line":          {minargs: 4, maxargs: 7, desc: "Line between (x1,y1) and (x2,y2)", usage: "line x1 y1 x2 y2 [lw] [color] [opacity]"},
	"pill":          {minargs: 4, maxargs: 5, desc: "Pill shape beginning at (x,y), dimensions (w,h)", usage: "pill x y w h [color]"},
	"polygon":       {minargs: 2, maxargs: 4, desc: "Polygon with specified x, y coordinates", usage: "polygon \"x1 x2 x3....\" \"y1 y2 y3...\" [color] [opacity]"},
	"polyline":      {minargs: 2, maxargs: 5, desc: "Polyline with specified x, y coordinates", usage: "polyline \"x1 x2 x3....\" \"y1 y2 y3...\" [lw] [color] [opacity]"},
	"rect":          {minargs: 4, maxargs: 6, desc: "Rectangle centered at (x,y), dimensions (w,h)", usage: "rect x y w h [color] [opacity]"},
	"rrect":         {minargs: 5, maxargs: 7, desc: "Rounded rectangle centered at (x,y), dimensions (w,h), corner radius r", usage: "rrect x y w h r [color] [opacity]"},
	"square":        {minargs: 3, maxargs: 5, desc: "Square centered at (x,y), size w", usage: "square x y w [color] [opacity]"},
	"star":          {minargs: 5, maxargs: 7, desc: "Star centered at (x,y), with sides, innner and outer sizes", usage: "star x y sides inner outer [color] [opacity]"},
	"vline":         {minargs: 3, maxargs: 5, desc: "Vertical line beginning at (x,y), h high", usage: "vline x y h [lw] [color] [opacity]"},
	"legend":        {minargs: 6, maxargs: 6, desc: "Chart legend", usage: "legend \"string\" x y fontsize font color"},
	"dchart":        {minargs: 1, maxargs: 9, desc: "Chart with specified options", usage: "dchart options..."},
	"arrow":         {minargs: 4, maxargs: 9, desc: "Arrow starting at (x1,y1), ending at (x2,y2), aw=arrow width, ah=arrow height", usage: "arrow x1 y1 x2 y2 [lw] [aw] [ah] [color] [opacity]"},
	"lcarrow":       {minargs: 6, maxargs: 11, desc: "Left curved arrow; curve specified by (bx,by), (cx,cy), (ex,ey)", usage: "lcarrow bx by bx xy ex ey [lw] [aw] [ah] [color] [opacity]"},
	"ucarrow":       {minargs: 6, maxargs: 11, desc: "Upward curved arrow; curve specified by (bx,by), (cx,cy), (ex,ey)", usage: "ucarrow bx by bx xy ex ey [lw] [aw] [ah] [color] [opacity]"},
	"dcarrow":       {minargs: 6, maxargs: 11, desc: "Downward curved arrow; curve specified by (bx,by), (cx,cy), (ex,ey)", usage: "dcarrow bx by bx xy ex ey [lw] [aw] [ah] [color] [opacity]"},
	"rcarrow":       {minargs: 6, maxargs: 11, desc: "Right curved arrow; curve specified by (bx,by), (cx,cy), (ex,ey)", usage: "rcarrow bx by bx xy ex ey [lw] [aw] [ah] [color] [opacity]"},
	"blist":         {minargs: 3, maxargs: 7, desc: "Bulleted list starting at (x,y), at fontsize", usage: "blist x y fontsize [font] [color] [opacity] [spacing]"},
	"clist":         {minargs: 3, maxargs: 7, desc: "Centered list starting at (x,y), at fontsize", usage: "clist x y fontsize [font] [color] [opacity] [spacing]"},
	"list":          {minargs: 3, maxargs: 7, desc: "List starting at (x,y), at fontsize", usage: "list x y fontsize [font] [color] [opacity] [spacing]"},
	"nlist":         {minargs: 3, maxargs: 7, desc: "Numbered list starting at (x,y), at fontsize", usage: "nlist x y fontsize [font] [color] [opacity] [spacing]"},
	"li":            {minargs: 0, maxargs: 4, desc: "List item", usage: "li \"item\" [font] [color] [opacity]"},
	"elist":         {minargs: 0, maxargs: 0, desc: "End the list", usage: "elist (end of list)"},
	"arctext":       {minargs: 7, maxargs: 11, desc: "Text on an arc, at fontsize, centered at (x,y), with radius r, between a1 and a2", usage: "arctext \"string\" x y radius a1 a2 fontsize [font] [color] [opacity] [link]"},
	"btext":         {minargs: 4, maxargs: 8, desc: "Text beginning at (x,y), at fontsize", usage: "btext \"string\" x y fontsize [font] [color] [opacity] [link]"},
	"ctext":         {minargs: 4, maxargs: 8, desc: "Centered text beginning at (x,y), at fontsize", usage: "ctext \"string\" x y fontsize [font] [color] [opacity] [link]"},
	"etext":         {minargs: 4, maxargs: 8, desc: "End-aligned text at (x,y), at fontsize", usage: "etext \"string\" x y fontsize [font] [color] [opacity] [link]"},
	"rtext":         {minargs: 5, maxargs: 9, desc: "Rotated text centered at x,y), at angle and fontsize", usage: "rtext \"string\" x y angle fontsize [font] [color] [opacity] [link]"},
	"text":          {minargs: 4, maxargs: 8, desc: "Text beginning at (x,y), at fontsize", usage: "text \"string\" x y fontsize [font] [color] [opacity] [link]"},
	"textblock":     {minargs: 5, maxargs: 9, desc: "Block of text beginning at (x,y), at fontsize, with width w", usage: "textblock \"string\" x y w fontsize [font] [color] [opacity] [link]"},
	"textblockfile": {minargs: 5, maxargs: 9, desc: "Block of text read for a file, beginning at (x,y), at fontsize, with width w", usage: "textblockfile \"file\" x y w fontsize [font] [color] [opacity] [link]"},
	"textcode":      {minargs: 5, maxargs: 7, desc: "Lines of code, read from a file, upper right corner at (x,y), margin at w", usage: "textcode \"file\" x y w fontsize [font] [color] [opacity]"},
	"textfile":      {minargs: 4, maxargs: 8, desc: "Contents of a text file pper right corner at (x,y)", usage: "textfile \"file\" x y fontsize [font] [color] [opacity] [spacing]"},
	"image":         {minargs: 5, maxargs: 7, desc: "Image center at (x,y), dimensions (w,h) (h=0, w is % of canvas width)", usage: "image \"file\" x y w h [scale] [link]"},
	"cimage":        {minargs: 6, maxargs: 8, desc: "Captioned image center at (x,y), dimensions (w,h) (h=0, w is % of canvas width)", usage: "cimage \"file\" \"caption\" x y w h [scale] [link] capsize"},
	"data":          {minargs: 1, maxargs: 1, desc: "Beginning at an embedded data file", usage: "data \"file\""},
	"edata":         {minargs: 0, maxargs: 0, desc: "End of data file", usage: "edata"},
	"def":           {minargs: 2, maxargs: 9, desc: "Define a function", usage: "def name args..."},
	"edef":          {minargs: 0, maxargs: 0, desc: "End a function", usage: "edef"},
	"import":        {minargs: 1, maxargs: 1, desc: "import function found in a file", usage: "import \"file\""},
	"if":            {minargs: 1, maxargs: 4, desc: "Conditional; one of: a==b,  a!=b,  a>b,  a<b,  a>=b, a<=b, a<>b c", usage: "if condition...[else]...eif"},
	"else":          {minargs: 0, maxargs: 0, desc: "Begin else section", usage: "else (followed by statements)"},
	"eif":           {minargs: 0, maxargs: 0, desc: "End condition", usage: "eif"},
	"for":           {minargs: 2, maxargs: 3, desc: "Begin loop; end with \"efor\"", usage: "for x=begin end [increment]"},
	"efor":          {minargs: 0, maxargs: 0, desc: "End loop", usage: "efor (end loop)"},
	"dbrace":        {minargs: 5, maxargs: 8, desc: "Downward pointing brace", usage: "dbrace x y w bw bh [lw] [color] [opacity]"},
	"ubrace":        {minargs: 5, maxargs: 8, desc: "Upward facing brace", usage: "ubrace x y w bw bh [lw] [color] [opacity]"},
	"lbrace":        {minargs: 5, maxargs: 8, desc: "Left pointing brace", usage: "lbrace x y h bw bh [lw] [color] [opacity]"},
	"rbrace":        {minargs: 5, maxargs: 8, desc: "Right pointing brace", usage: "rbrace x y h bw bh [lw] [color] [opacity]"},
	"dbracket":      {minargs: 4, maxargs: 7, desc: "Downward pointing bracket", usage: "dbracket x y w h [lw] [color] [opacity]"},
	"lbracket":      {minargs: 4, maxargs: 7, desc: "Left pointing bracket", usage: "lbracket x y w h [lw] [color] [opacity]"},
	"ubracket":      {minargs: 4, maxargs: 7, desc: "Upward facing bracket", usage: "ubracket x y w h [lw] [color] [opacity]"},
	"rbracket":      {minargs: 4, maxargs: 7, desc: "Right pointing bracket", usage: "rbracket x y w h [lw] [color] [opacity]"},
	"polar":         {minargs: 4, maxargs: 4, desc: "Assign polar coordinate centered at (x,y) at radius and angle (0-360)", usage: "polar x y radius angle"},
	"polarx":        {minargs: 4, maxargs: 4, desc: "Assign X-polar coordinate centered at (x,y) at radius and angle (0-360)", usage: "polarx x y radius angle"},
	"polary":        {minargs: 4, maxargs: 4, desc: "Assign Y-polar coordinate centered at (x,y) at radius and angle (0-360)", usage: "polary x y radius angle"},
	"cosine":        {minargs: 1, maxargs: 1, desc: "Assign the cosine of expression", usage: "cosine expression"},
	"sine":          {minargs: 1, maxargs: 1, desc: "sasign the sine of expression", usage: "sine expression"},
	"sqrt":          {minargs: 1, maxargs: 1, desc: "Assign the square root of expression", usage: "sqrt expression"},
	"tangent":       {minargs: 1, maxargs: 1, desc: "Assign the tangent of expression", usage: "tangent expression"},
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

func keyinfo(s string, kind string) {
	var keys []string
	if s == "all" {
		for k := range kwInfo {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	} else {
		keys = strings.Split(s, ",")
	}

	for _, k := range keys {
		switch kind {
		case "info":
			fmt.Printf(kwfmt, kwInfo[k].usage, kwInfo[k].desc)
		case "desc":
			fmt.Printf(infofmt, k, kwInfo[k].desc)
		case "usage":
			fmt.Printf("%s\n", kwInfo[k].usage)
		}
	}
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

func unquote(s string) string {
	l := len(s)
	if l > 2 && s[0] == '"' && s[l-1] == '"' {
		return s[1 : l-1]
	}
	return ""
}

// imagecheck checks if images can be opened
func imagecheck(data [][]string) int {
	errors := 0
	for _, d := range data {
		if len(d) > 2 && d[0] == "image" {
			imgfile := unquote(d[1])
			if len(imgfile) == 0 {
				continue
			}
			_, err := os.Open(imgfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				errors++
			}
		}
	}
	return errors
}

// lint tests for proper keyword arguments
func lint(data [][]string) int {
	issues := 0
	for i := 0; i < len(data); i++ {
		if kind(data[i]) != Keyword {
			continue
		}
		keyword := data[i][0]
		min := kwInfo[keyword].minargs
		ln := len(data[i]) - 1
		if ln < min {
			fmt.Fprintf(os.Stderr, argerrfmt, i+1, data[i], keyword, min, ln, kwInfo[keyword].usage)
			issues++
		}
	}
	return issues
}

// process processes an io.Reader of decksh code
func process(r io.Reader) int {
	issues := 0
	data := readDecksh(r) // read the data
	kwcounter(data)       // count keywords and elements
	issues += lint(data)  // check argument counts
	issues += kwcheck()   // integrity check
	issues += imagecheck(data)
	return issues
}

// lint named files or standard input if no file is specified.
func main() {
	var info string
	var desc string
	var usage string

	flag.StringVar(&info, "info", "", "show usage and description by keyword (\"all\" for every keyword, or a comma separated list. For example circle,rect)")
	flag.StringVar(&desc, "desc", "", "show description by keyword (\"all\" for every keyword, or a comma separated list. For example circle,rect))")
	flag.StringVar(&usage, "usage", "", "show usage by keyword (\"all\" for every keyword, or a comma separated list. For example circle,rect))")
	flag.Parse()

	if len(info) > 0 {
		keyinfo(info, "info")
		os.Exit(0)
	}

	if len(desc) > 0 {
		keyinfo(desc, "desc")
		os.Exit(0)
	}

	if len(usage) > 0 {
		keyinfo(usage, "usage")
		os.Exit(0)
	}

	args := flag.Args()
	la := len(args)

	if la == 0 {
		os.Exit(process(os.Stdin))
	}

	issues := 0
	for i := 0; i < la; i++ {
		input, err := os.Open(args[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		issues += process(input)
	}
	os.Exit(issues)
}
