// decksh: a little language that generates deck markup
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ajstarks/decksh"
)

// $ decksh                    # input from stdin, output to stdout
// $ decksh -o foo.xml         # input from stdin, output to foo.xml
// $ decksh foo.dsh            # input from foo.dsh output to stdout
// $ decksh -o foo.xml foo.dsh # input from foo.dsh output to foo.xml
// $ decksh -var  name=value   # make an assignment
// $ decksh -version           # show decksh version
func main() {
	var dest, cvar string
	var version, dump bool
	var input io.ReadCloser = os.Stdin
	var output io.WriteCloser = os.Stdout

	flag.StringVar(&dest, "o", "", "output destination")
	flag.StringVar(&cvar, "var", "", "assign name=value")
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&dump, "dump", false, "dump variables")
	flag.Parse()

	if len(cvar) > 2 {
		i := strings.Index(cvar, "=")
		if i > 0 {
			name := strings.TrimSpace(cvar[0:i])
			value := strings.TrimSpace(cvar[i+1:])
			decksh.Assign(name, value)
		}
	}

	if version {
		l := len(decksh.Version)
		fmt.Fprintln(os.Stderr, decksh.Version[1:l-1])
		os.Exit(1)
	}

	if len(flag.Args()) > 0 {
		var err error
		input, err = os.Open(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}

	if len(dest) > 0 {
		var err error
		output, err = os.Create(dest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
	}

	if err := decksh.Process(output, input); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(3)
	}

	if dump {
		decksh.Dump("")
	}

	input.Close()
	output.Close()
	os.Exit(0)
}
