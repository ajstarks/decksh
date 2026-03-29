package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// command line options
type options struct {
	items       int
	dump        bool
	top         float64
	left        float64
	size        float64
	linespacing float64
	colwidth    float64
	title       string
	color       string
	font        string
	outfile     string
	inputdir    string
}

const (
	doublequote = '"'
	tocmarker   = "// TOC: "
)

// unquote removes quotes from a string
func unquote(s string) string {
	la := len(s)
	if la > 2 && s[0] == doublequote && s[la-1] == doublequote {
		s = s[1 : la-1]
	}
	return s
}

// include reads the contents of included decksh source, counting pages
func include(name string, dir string) int {
	name = unquote(name)
	pages := 0
	r, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to look inside %s\n", name)
		return pages

	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		// count pages
		if len(args) > 0 && (args[0] == "slide" || args[0] == "page") {
			pages++
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	r.Close()
	return pages
}

// title writes a title to w
func title(w io.Writer, opts options) {
	if len(opts.title) > 0 {
		fmt.Fprintf(w, "text %q %v %v %v\n",
			opts.title, opts.left,
			opts.top+(opts.size*3), opts.size*1.5)
	}
}

// newlist writes the beginning of a list to w
func newlist(w io.Writer, opts options) {
	fmt.Fprintf(w, "list %v %v %v %q %q %v %v\n",
		opts.left, opts.top, opts.size,
		opts.font, opts.color, "100", opts.linespacing)
}

// nameitem writes a list item (name) to w
func nameitem(w io.Writer, name string) {
	fmt.Fprintf(w, "li %q\n", name)
}

// pageitem writes a page number item to w
func pageitem(w io.Writer, page int) {
	fmt.Fprintf(w, "li \"%d\"\n", page)
}

// endlist writes an ending list to w
func endlist(w io.Writer) {
	fmt.Fprintln(w, "elist")
}

// beginslide writes new slide to w
func beginslide(w io.Writer) {
	fmt.Fprintln(w, "slide")
}

// endslide writes a ending slide markup to w
func endslide(w io.Writer) {
	fmt.Fprintln(w, "eslide")
}

// mktoc makes a table of contents reading from r, writing to w
func mktoc(w io.Writer, r io.Reader, opts options) {
	page := 1
	scanner := bufio.NewScanner(r)

	// read each line, counting slides
	// if as special TOC comment is found tag it as an item in the TOC
	var toc = map[int]string{} // map page numbers to names
	for scanner.Scan() {
		t := scanner.Text()
		args := strings.Fields(t)

		// count pages
		if len(args) > 0 && (args[0] == "slide" || args[0] == "page") {
			page++
		}

		// count pages in included files
		// (assume includes are relative to the input directory)
		if len(args) > 1 && args[0] == "include" {
			page += include(args[1], opts.inputdir)
		}

		// special TOC comment found, tag it with the current page
		if i := strings.Index(t, tocmarker); i >= 0 {
			toc[page] = t[i+len(tocmarker):]
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}

	// process items, ordered by the page

	// sort pages
	pages := make([]int, 0, len(toc))
	for k := range toc {
		pages = append(pages, k)
	}
	slices.Sort(pages)

	// top matter
	beginslide(w)
	title(w, opts)

	// TOC names
	initleft := opts.left
	newlist(w, opts)
	for i, p := range pages {
		if i > 0 && i%opts.items == 0 { // every opt.items, make a new column
			endlist(w)
			opts.left += opts.colwidth
			newlist(w, opts)
		}
		nameitem(w, toc[p])
	}
	endlist(w)

	// TOC page numbers
	pageopts := opts
	pageopts.left = initleft + (opts.colwidth * 0.75)
	pageopts.color = "gray"
	newlist(w, pageopts)
	for i, p := range pages {
		if i > 0 && i%opts.items == 0 { // every opt.items, make a new column
			endlist(w)
			pageopts.left += pageopts.colwidth
			newlist(w, pageopts)
		}
		pageitem(w, p)
	}
	endlist(w)

	endslide(w)

	// dump the TOC if specified
	if opts.dump {
		for _, p := range pages {
			fmt.Fprintf(os.Stderr, "%-35s %d\n", toc[p], p)
		}
	}
}

// setoptions sets command line options and input files
func setoptions() ([]string, options) {
	var opts options
	flag.IntVar(&opts.items, "items", 20, "items per column")
	flag.BoolVar(&opts.dump, "dump", false, "dump TOC")
	flag.Float64Var(&opts.top, "top", 85, "top of the page %")
	flag.Float64Var(&opts.left, "left", 5, "left margin %")
	flag.Float64Var(&opts.size, "size", 1.5, "font size %")
	flag.Float64Var(&opts.linespacing, "ls", 1.2, "line spacing %")
	flag.Float64Var(&opts.colwidth, "cw", 30, "name column width %")
	flag.StringVar(&opts.font, "font", "sans", "font")
	flag.StringVar(&opts.color, "color", "", "text color")
	flag.StringVar(&opts.outfile, "o", "", "output file")
	flag.StringVar(&opts.inputdir, "dir", ".", "input directory")
	flag.StringVar(&opts.title, "title", "", "TOC Title")
	flag.Parse()
	return flag.Args(), opts
}

// deckshTOC makes a table of contents from decksh source
func deckshTOC(args []string, opts options) {
	var output io.Writer = os.Stdout
	var input io.Reader = os.Stdin

	if opts.items <= 0 {
		opts.items = 20
	}

	// if no args, use stdout, stdin
	if len(args) < 1 {
		mktoc(output, input, opts)
		return
	}

	// set an output file, if specified, fallback to stdout on error
	var err error
	if len(opts.outfile) > 0 {
		output, err = os.Create(opts.outfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v (falling back to standard output)\n", err)
			output = os.Stdout
		}
	}

	// for every file specified, make a TOC
	for _, f := range args {
		r, err := os.Open(filepath.Join(opts.inputdir, f))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		mktoc(output, r, opts)
		r.Close()
	}
}

func main() {
	args, opts := setoptions()
	deckshTOC(args, opts)
}
