// decksh: a little language that generates deck markup
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ajstarks/decksh"
)

// $ decksh                    # input from stdin, output to stdout
// $ decksh -o foo.xml         # input from stdin, output to foo.xml
// $ decksh foo.dsh            # input from foo.dsh output to stdout
// $ decksh -o foo.xml foo.dsh # input from foo.dsh output to foo.xml
// $ decksh -version           # show decksh version
func main() {
	dest := flag.String("o", "", "output destination")
	version := flag.Bool("version", false, "show version")
	var input io.ReadCloser = os.Stdin
	var output io.WriteCloser = os.Stdout
	var rerr, werr error

	flag.Parse()

	if *version {
		fmt.Fprintln(os.Stderr, decksh.Version)
		os.Exit(1)
	}

	if len(flag.Args()) > 0 {
		input, rerr = os.Open(flag.Args()[0])
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", rerr)
			os.Exit(1)
		}
	}

	if len(*dest) > 0 {
		output, werr = os.Create(*dest)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", werr)
			os.Exit(2)
		}
	}

	err := decksh.Process(output, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(3)
	}

	input.Close()
	output.Close()
	os.Exit(0)
}
