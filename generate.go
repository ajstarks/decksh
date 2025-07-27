// Package decksh is a little language that generates deck markup
// code generation
package decksh

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ajstarks/dchart"
)

const (
	locsep      = 0x09 // tab
	doublequote = 0x22
	stdnotch    = 0.75
	curvefmt    = "<curve xp1=\"%.2f\" yp1=\"%.2f\" xp2=\"%.2f\" yp2=\"%.2f\" xp3=\"%.2f\" yp3=\"%.2f\" %s/>\n"
	linefmt     = "<line xp1=\"%.2f\" yp1=\"%.2f\" xp2=\"%.2f\" yp2=\"%.2f\" %s/>\n"
	sqfmt       = "<rect xp=\"%.2f\" yp=\"%.2f\" wp=\"%.2f\" hp=\"%.2f\" hr=\"100\" %s/>\n"
)

// xmlmap defines the XML substitutions
var xmlmap = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;")

// xmlesc escapes XML
func xmlesc(s string) string {
	return xmlmap.Replace(s)
}

// validNumber checks that a string starts with a digit or sign or decimal point
func validNumber(s ...string) error {
	for i := 0; i < len(s); i++ {
		c := s[i][0]
		if !(('0' <= c && c <= '9') || c == '-' || c == '.') {
			return fmt.Errorf("'%v' is not a number (not defined?)", s[i])
		}
	}
	return nil
}

// elist ends a deck, slide, or list
func endtag(w io.Writer, s []string, linenumber int) error {
	tag := s[0]
	if len(tag) < 2 || tag[0:1] != "e" {
		return fmt.Errorf("line %d: edeck, edoc, eslide, epage or elist", linenumber)
	}
	// edoc and epage are the same as edeck and eslide
	switch tag {
	case "edoc":
		tag = "edeck"
	case "epage":
		tag = "eslide"
	}
	fmt.Fprintf(w, "</%s>\n", tag[1:])
	return nil
}

// colorstring returns the markup for a color.
// either a single color or two colors and a percentage used for gradients
func colorstring(prefix, s string) string {
	dc := prefix + s
	if !strings.Contains(s, "/") {
		return dc
	}
	if gc := strings.Split(s, "/"); len(gc) == 3 {
		return `gradcolor1=` + gc[0] + `" gradcolor2="` + gc[1] + `" gp="` + gc[2]
	}
	return dc
}

// fontColorOp generates markup for font, color, and opacity
func fontColorOp(s []string) string {
	switch len(s) {
	case 1:
		return fmt.Sprintf("font=%s", s[0])
	case 2:
		return fmt.Sprintf("font=%s color=%s", s[0], s[1])
	case 3:
		return fmt.Sprintf("font=%s color=%s opacity=%q", s[0], s[1], s[2])
	case 4:
		return fmt.Sprintf("font=%s color=%s opacity=%q link=%s", s[0], s[1], s[2], xmlesc(s[3]))
	default:
		return ""
	}
}

// fontColorOpLp generates markup for font, color, and opacity and linespacing
func fontColorOpLp(s []string) string {
	switch len(s) {
	case 1:
		return fmt.Sprintf("font=%s", s[0])
	case 2:
		return fmt.Sprintf("font=%s color=%s", s[0], s[1])
	case 3:
		return fmt.Sprintf("font=%s color=%s opacity=%q", s[0], s[1], s[2])
	case 4:
		return fmt.Sprintf("font=%s color=%s opacity=%q lp=%q", s[0], s[1], s[2], s[3])
	case 5:
		return fmt.Sprintf("font=%s color=%s opacity=%q lp=%q link=%s", s[0], s[1], s[2], s[3], xmlesc(s[4]))
	case 6:
		return fmt.Sprintf("font=%s color=%s opacity=%q lp=%q link=%s rotation=%q", s[0], s[1], s[2], s[3], xmlesc(s[4]), s[5])
	default:
		return ""
	}
}

// qesc remove quotes from a string, and XML escape it
func qesc(s string) string {
	if len(s) < 3 {
		return ""
	}
	return (xmlesc(s[1 : len(s)-1]))
}

// deck produces the "deck" element
func deck(w io.Writer, s []string, linenumber int) error {
	_, err := fmt.Fprintln(w, "<deck>")
	if err != nil {
		return fmt.Errorf("line %d: %s, unable to write", linenumber, s)
	}
	return err
}

// canvas produces the "canvas" element
// canvas width height
func canvas(w io.Writer, s []string, linenumber int) error {
	e := fmt.Errorf("line %d: %s width height", linenumber, s[0])
	if len(s) != 3 {
		return e
	}
	for i := 1; i < 3; i++ {
		s[i] = eval(s[i])
	}
	canvasWidth, e = strconv.ParseFloat(s[1], 64)
	if e != nil {
		return e
	}
	canvasHeight, e = strconv.ParseFloat(s[2], 64)
	if e != nil {
		return e
	}
	if canvasHeight == 0 || canvasWidth == 0 {
		canvasWidth, canvasHeight = 792.0, 612.0
	}
	fmt.Fprintf(w, "<canvas width=%q height=%q/>\n", s[1], s[2])
	return nil
}

// slide produces the "slide" element
// slide [bg] [fg]
func slide(w io.Writer, s []string, linenumber int) error {
	switch len(s) {
	case 1:
		fmt.Fprintln(w, "<slide>")
	case 2:
		fmt.Fprintf(w, "<slide %s>\n", colorstring("bg=", s[1]))
	case 3:
		fmt.Fprintf(w, "<slide %s fg=%s>\n", colorstring("bg=", s[1]), s[2])
	default:
		return fmt.Errorf("line %d: %s [bgcolor] [fgcolor]", linenumber, s[0])
	}
	return nil
}

// content generates content of arbitrary type
// content "scheme://file" x y size
func content(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n != 5 {
		return fmt.Errorf("line %d: %s \"scheme://file\" x y size", linenumber, s[0])
	}
	uri := unquote(s[1])
	i := strings.Index(uri, "://")
	var scheme, file string
	if i > 0 && len(uri) > 4 {
		scheme = uri[0:i]
		file = uri[i+3:]
	}
	if err := validNumber(s[2], s[3], s[4]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	fmt.Fprintf(w, "<text type=%q file=%q xp=%q yp=%q sp=%q/>\n", scheme, file, s[2], s[3], s[4])
	return nil
}

// text generates markup for text
// text x y size [font] [color] [opacity] [link]
func text(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 5 {
		return fmt.Errorf("line %d: %s \"text\" x y size [font] [color] [opacity] [link]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	fco := fontColorOp(s[5:])
	switch s[0] {
	case "text", "btext":
		fmt.Fprintf(w, "<text xp=%q yp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], fco, qesc(s[1]))
	case "ctext":
		fmt.Fprintf(w, "<text align=\"c\" xp=%q yp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], fco, qesc(s[1]))
	case "etext":
		fmt.Fprintf(w, "<text align=\"e\" xp=%q yp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], fco, qesc(s[1]))
	case "textfile":
		fmt.Fprintf(w, "<text file=%s xp=%q yp=%q sp=%q %s/>\n", s[1], s[2], s[3], s[4], fontColorOpLp(s[5:]))
	}
	return nil
}

// arctext places text on an arc
// arctext cx cy radius begin-angle end-angle size [font] [color] [opacity] [link]
func arctext(w io.Writer, s []string, linenumber int) error {
	if len(s) < 8 {
		return fmt.Errorf("line %d: %s \"arctext\" cx cy radius begin-angle end-angle size [font] [color] [opacity] [link]", linenumber, s[0])
	}
	cx, err := strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a valid beginning center x", linenumber, s[2])
	}
	cy, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a valid beginning center y", linenumber, s[3])
	}
	radius, err := strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a valid beginning radius", linenumber, s[4])
	}
	begin, err := strconv.ParseFloat(s[5], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a valid beginning angle", linenumber, s[5])
	}
	end, err := strconv.ParseFloat(s[6], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a valid ending angle", linenumber, s[6])
	}
	if len(s[1]) < 3 {
		return fmt.Errorf("line %d: %v is not valid text", linenumber, s[1])
	}
	text := s[1][1 : len(s[1])-1]
	interval := (end - begin) / float64(len(text)-1)

	angle := begin
	var textangle float64
	for i := 0; i < len(text); i++ {
		px, py := polar(cx, cy, radius, angle*(math.Pi/180))
		if end >= begin {
			textangle = angle + 90
		} else {
			textangle = angle + 270
		}
		fmt.Fprintf(w, "<text xp=\"%.2f\" yp=\"%.2f\" rotation=\"%.2f\" sp=%q %s>%s</text>\n", px, py, textangle, s[7], fontColorOp(s[8:]), xmlesc(text[i:i+1]))
		angle += interval
	}
	return nil
}

// rtext generates markup for rotated text
// rtext x y angle size [font] [color] [opacity] [link]
func rtext(w io.Writer, s []string, linenumber int) error {
	if len(s) < 6 {
		return fmt.Errorf("line %d: %s \"text\" x y angle size [font] [color] [opacity] [link]", linenumber, s[0])
	}
	angle, err := strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d %v is not a valid rotation angle", linenumber, angle)
	}
	if err := validNumber(s[2], s[3], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	fmt.Fprintf(w, "<text xp=%q yp=%q rotation=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], s[5], fontColorOp(s[6:]), qesc(s[1]))
	return nil
}

// textblock generates markup for a block of text
// textblock x y width size [font] [color] [opacity] [link]
func textblock(w io.Writer, s []string, linenumber int) error {
	if len(s) < 6 {
		return fmt.Errorf("line %d: %s \"textblock\" x y width size [font] [color] [opacity] [link]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	fmt.Fprintf(w, "<text type=\"block\" xp=%q yp=%q wp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], s[5], fontColorOp(s[6:]), qesc(s[1]))
	return nil
}

// textblockfile generates markup for a block of text read from a file
func textblockfile(w io.Writer, s []string, linenumber int) error {
	if len(s) < 6 {
		return fmt.Errorf("line %d: %s \"filename\" x y width size [font] [color] [opacity] [link]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	filename := unquote(s[1])
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	data := xmlesc(string(content))
	fmt.Fprintf(w, "<text type=\"block\" xp=%q yp=%q wp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], s[5], fontColorOp(s[6:]), data)
	return nil
}

// textcode generates markup for a block of code
// textcode x y width size [color]
func textcode(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s \"filename\" x y width size [color]", linenumber, s[0])

	if n < 6 {
		return e
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	switch n {
	case 6:
		fmt.Fprintf(w, "<text type=\"code\" file=%s xp=%q yp=%q wp=%q sp=%q/>\n", s[1], s[2], s[3], s[4], s[5])
	case 7:
		fmt.Fprintf(w, "<text type=\"code\" file=%s xp=%q yp=%q wp=%q sp=%q color=%s/>\n", s[1], s[2], s[3], s[4], s[5], s[6])
	default:
		return e
	}
	return nil
}

// image generates markup for images (plain and captioned)
// image "file" x y w h [scale] [link]
func image(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s \"image-file\" x y w h [scale] [link]", linenumber, s[0])
	if n < 6 {
		return e
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}

	switch n {
	case 6:
		fmt.Fprintf(w, "<image name=%s xp=%q yp=%q width=%q height=%q/>\n", s[1], s[2], s[3], s[4], s[5])
	case 7:
		fmt.Fprintf(w, "<image name=%s xp=%q yp=%q width=%q height=%q scale=%q/>\n", s[1], s[2], s[3], s[4], s[5], s[6])
	case 8:
		fmt.Fprintf(w, "<image name=%s xp=%q yp=%q width=%q height=%q scale=%q link=%s/>\n", s[1], s[2], s[3], s[4], s[5], s[6], s[7])
	default:
		return e
	}
	return nil
}

// cimage makes a captioned image
// cimage "file" "caption" x y w h [scale] [link]
func cimage(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s \"image-file\" \"caption\" x y w h [scale] [link] [caption size]", linenumber, s[0])
	if n < 7 {
		return e
	}
	if err := validNumber(s[3], s[4], s[5], s[6]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	caption := xmlesc(s[2])
	switch n {
	case 7:
		fmt.Fprintf(w, "<image name=%s caption=%s xp=%q yp=%q width=%q height=%q/>\n", s[1], caption, s[3], s[4], s[5], s[6])
	case 8:
		fmt.Fprintf(w, "<image name=%s caption=%s xp=%q yp=%q width=%q height=%q scale=%q/>\n", s[1], caption, s[3], s[4], s[5], s[6], s[7])
	case 9:
		fmt.Fprintf(w, "<image name=%s caption=%s xp=%q yp=%q width=%q height=%q scale=%q link=%s/>\n", s[1], caption, s[3], s[4], s[5], s[6], s[7], s[8])
	case 10:
		fmt.Fprintf(w, "<image name=%s caption=%s xp=%q yp=%q width=%q height=%q scale=%q link=%s sp=%q/>\n", s[1], caption, s[3], s[4], s[5], s[6], s[7], s[8], s[9])
	default:
		return e
	}
	return nil
}

// list generates markup for lists
// list x y size [font] [color] [opacity] [lp] [link]
func list(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 4 {
		return fmt.Errorf("line %d: %s x y size [font] [color] [opacity] [lp] [link]", linenumber, s[0])
	}
	var fco string
	if n > 4 {
		fco = fontColorOpLp(s[4:])
	}
	if err := validNumber(s[1], s[1], s[3]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	switch s[0] {
	case "list":
		fmt.Fprintf(w, "<list xp=%q yp=%q sp=%q %s>\n", s[1], s[2], s[3], fco)
	case "blist":
		fmt.Fprintf(w, "<list type=\"bullet\" xp=%q yp=%q sp=%q %s>\n", s[1], s[2], s[3], fco)
	case "nlist":
		fmt.Fprintf(w, "<list type=\"number\" xp=%q yp=%q sp=%q %s>\n", s[1], s[2], s[3], fco)
	case "clist":
		fmt.Fprintf(w, "<list align=\"center\" xp=%q yp=%q sp=%q %s>\n", s[1], s[2], s[3], fco)
	}
	return nil
}

// listitem generates list items
func listitem(w io.Writer, s []string, linenumber int) error {
	ls := len(s)
	switch {
	case ls == 1:
		fmt.Fprintln(w, "<li/>")
	case ls == 2:
		fmt.Fprintf(w, "<li>%s</li>\n", qesc(s[1]))
	case ls > 2:
		fmt.Fprintf(w, "<li %s>%s</li>\n", fontColorOp(s[2:]), qesc(s[1]))
	default:
		return fmt.Errorf("line %d: %s, listitem should have 1, 2, or more arguments", linenumber, s)
	}
	return nil
}

// shapes generates markup for rectangle and ellipse
func shapes(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s x y w h [color] [opacity]", linenumber, s[0])
	if n < 5 {
		return e
	}
	if err := validNumber(s[1], s[2], s[3], s[4]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	dim := fmt.Sprintf("xp=%q yp=%q wp=%q hp=%q", s[1], s[2], s[3], s[4])
	switch n {
	case 5:
		fmt.Fprintf(w, "<%s %s/>\n", s[0], dim)
	case 6:
		fmt.Fprintf(w, "<%s %s %s/>\n", s[0], dim, colorstring("color=", s[5]))
	case 7:
		fmt.Fprintf(w, "<%s %s color=%s opacity=%q/>\n", s[0], dim, s[5], s[6])
	default:
		return e
	}
	return nil
}

// regshapes generates markup for square and circle
func regshapes(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s x y w [color] [opacity]", linenumber, s[0])
	if n < 4 {
		return e
	}
	if err := validNumber(s[1], s[2], s[3]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	switch s[0] {
	case "square":
		s[0] = "rect"
	case "circle":
		s[0] = "ellipse"
	case "acircle":
		s[0] = "ellipse"
		v, err := strconv.ParseFloat(s[3], 64)
		if err != nil {
			return err
		}
		s[3] = ftoa(area(v))
	}
	dim := fmt.Sprintf("xp=%q yp=%q wp=%q hr=\"100\"", s[1], s[2], s[3])
	switch n {
	case 4:
		fmt.Fprintf(w, "<%s %s/>\n", s[0], dim)
	case 5:
		fmt.Fprintf(w, "<%s %s %s/>\n", s[0], dim, colorstring("color=", s[4]))
	case 6:
		fmt.Fprintf(w, "<%s %s color=%s opacity=%q/>\n", s[0], dim, s[4], s[5])
	default:
		return e
	}
	return nil
}

// roundrect makes a rounded rectangle centered at (x,y) with dimensions (w, h), r is the corner radius
// rrect x y w h r [color]
func roundrect(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s x y w h r [color]", linenumber, s[0])
	}
	x, err := strconv.ParseFloat(eval(s[1]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	y, err := strconv.ParseFloat(eval(s[2]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	width, err := strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	height, err := strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	radius, err := strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	var endtag string
	if n > 6 {
		endtag = `color=` + s[6]
	}

	// Adjust coordinates so that the reference point is the middle of the rectangle
	rx := x
	ry := y
	x -= (width / 2)
	y += (height / 2)
	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x, y, radius, endtag)
	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x+width, y, radius, endtag)
	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x, y-height, radius, endtag)
	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x+width, y-height, radius, endtag)
	fmt.Fprintf(w, "<line xp1=\"%v\" yp1=\"%v\" xp2=\"%v\" yp2=\"%v\" sp=\"%v\" %s/>\n", x, y, x+width, y, radius, endtag)
	fmt.Fprintf(w, "<line xp1=\"%v\" yp1=\"%v\" xp2=\"%v\" yp2=\"%v\" sp=\"%v\" %s/>\n", x, y-height, x+width, y-height, radius, endtag)
	fmt.Fprintf(w, "<rect xp=\"%v\" yp=\"%v\" wp=\"%v\" hp=\"%v\" %s/>\n", rx, ry, width+radius, height, endtag)
	return nil
}

// pill makes a horizontal  pill shape
// pill x y w h [color]
func pill(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 5 {
		return fmt.Errorf("line %d: %s x y w h [color]", linenumber, s[0])
	}
	x, err := strconv.ParseFloat(eval(s[1]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	y, err := strconv.ParseFloat(eval(s[2]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	width, err := strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	height, err := strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	var endtag string
	if n > 5 {
		endtag = `color=` + s[5]
	}

	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x, y, height, endtag)
	fmt.Fprintf(w, "<line xp1=\"%v\" yp1=\"%v\" xp2=\"%v\" yp2=\"%v\" sp=\"%v\" hp=\"100\" %s/>\n", x, y, x+width, y, height, endtag)
	fmt.Fprintf(w, "<ellipse xp=\"%v\" yp=\"%v\" wp=\"%v\" hr=\"100\" %s/>\n", x+width, y, height, endtag)
	return nil
}

// star makes a n-sided star
// star x y nsides inner outer [color] [op]
func star(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s x y nsides inner outer [color] [op]", linenumber, s[0])
	if n < 6 {
		return e
	}
	// get the parameters
	x, err := strconv.ParseFloat(eval(s[1]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	y, err := strconv.ParseFloat(eval(s[2]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	nsides, err := strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	inner, err := strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	outer, err := strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	// compute the polygon coordinates
	ns2 := int(nsides) * 2
	xp, yp := make([]float64, ns2), make([]float64, ns2)
	a := 90.0
	ai := 360.0 / nsides
	for i := 0; i < ns2; i++ {
		if i%2 == 0 {
			xp[i], yp[i] = polar(x, y, outer, a*(math.Pi/180))
		} else {
			xp[i], yp[i] = polar(x, y, inner, a*(math.Pi/180))
		}
		a += ai
	}
	fmt.Fprintf(w, "<polygon xc=\"")
	// x coords
	for i := 0; i < ns2-1; i++ {
		fmt.Fprintf(w, "%.2f ", xp[i])
	}
	// y coords
	fmt.Fprintf(w, "%.2f\" yc=\"", xp[ns2-1])
	for i := 0; i < ns2-1; i++ {
		fmt.Fprintf(w, "%.2f ", yp[i])
	}
	fmt.Fprintf(w, "%.2f\"", yp[ns2-1])
	switch n {
	case 6:
		fmt.Fprintf(w, "/>\n")
	case 7:
		fmt.Fprintf(w, " color=%s/>\n", eval(s[6]))
	case 8:
		fmt.Fprintf(w, " color=%s opacity=%q/>\n", eval(s[6]), eval(s[7]))
	default:
		return e
	}
	return nil
}

// geopoly makes polygons from geometric data
func geopoly(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s \"file\" latmin latmax longmin longmax [color]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return err
	}
	kml, err := readKMLData(unquote(s[1]))
	if err != nil {
		return err
	}
	m, err := makegeometry(s)
	if err != nil {
		return err
	}
	color := "gray"
	if n > 6 {
		color = s[6]
	}
	geoshape(w, kml, m, 0, unquote(color), "polygon")
	return nil
}

// geoline makes lines from geometric data
func geoline(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s \"file\" latmin latmax longmin longmax [size] [color]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	kml, err := readKMLData(unquote(s[1]))
	if err != nil {
		return err
	}
	m, err := makegeometry(s)
	if err != nil {
		return err
	}
	size := 0.2
	color := "gray"
	if n > 6 {
		var serr error
		size, serr = strconv.ParseFloat(eval(s[6]), 64)
		if serr != nil {
			return serr
		}
	}
	if n > 7 {
		color = eval(s[7])
	}
	geoshape(w, kml, m, size, unquote(color), "polyline")
	return nil
}

// geoloc makes dot and label with alighnment
func geoloc(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s \"file\" latmin latmax longmin longmax [align] [size] [font] [color]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	r, err := os.Open(unquote(s[1]))
	if err != nil {
		return err
	}
	m, err := makegeometry(s)
	if err != nil {
		return err
	}
	loc, err := readLoc(r, locsep)
	defer r.Close()
	x := loc.X
	y := loc.Y
	x, y = mapData(x, y, m)
	size := 1.0
	align := "c"
	if n > 6 {
		align = eval(s[6])
	}
	if n > 7 {
		var serr error
		size, serr = strconv.ParseFloat(eval(s[7]), 64)
		if serr != nil {
			return serr
		}
	}
	var fco string
	if n > 8 {
		fco = fontColorOpLp(s[8:])
	}
	color := "gray"
	if n > 9 {
		color = unquote(eval(s[9]))
	}
	geodot(w, x, y, size, color)
	geotext(w, x, y, loc.Name, unquote(align), size, fco)
	return nil
}

// geolabel makes labels from geometric data (lat/long pairs)
func geolabel(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s \"file\" latmin latmax longmin longmax [size] [font] [color] [op]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	r, err := os.Open(unquote(s[1]))
	if err != nil {
		return err
	}
	m, err := makegeometry(s)
	if err != nil {
		return err
	}
	loc, err := readLoc(r, locsep)
	defer r.Close()
	x := loc.X
	y := loc.Y
	x, y = mapData(x, y, m)
	size := 1.0
	if n > 6 {
		var serr error
		size, serr = strconv.ParseFloat(eval(s[6]), 64)
		if serr != nil {
			return serr
		}
	}
	var fco string
	if n > 7 {
		fco = fontColorOpLp(s[7:])
	}
	geotext(w, x, y, loc.Name, "c", size, fco)
	return nil
}

// geopoint makes points from geometric data (lat/long pairs)
func geopoint(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 6 {
		return fmt.Errorf("line %d: %s \"file\" latmin latmax longmin longmax [size] [color]", linenumber, s[0])
	}
	if err := validNumber(s[2], s[3], s[4], s[5]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	r, err := os.Open(unquote(s[1]))
	if err != nil {
		return err
	}
	m, err := makegeometry(s)
	if err != nil {
		return err
	}
	loc, err := readLoc(r, locsep)
	defer r.Close()
	x := loc.X
	y := loc.Y
	x, y = mapData(x, y, m)
	size := 1.0
	color := "black"
	if n > 6 {
		var serr error
		size, serr = strconv.ParseFloat(eval(s[6]), 64)
		if serr != nil {
			return serr
		}
	}
	if n > 7 {
		color = eval(s[7])
	}
	geodot(w, x, y, size, unquote(color))
	return nil
}

// unquote removes quotes from a string
func unquote(s string) string {
	la := len(s)
	if la > 2 && s[0] == doublequote && s[la-1] == doublequote {
		s = s[1 : la-1]
	}
	return s
}

// polygon generates markup for polygons
// polygon "xcoord" "ycoord" [color] [opacity]
func polygon(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s \"xcoord\" \"ycoord\" [color] [opacity]", linenumber, s[0])
	if n < 3 {
		return e
	}

	// get the coordinates from the quoted strings
	xc := strings.Fields(unquote(s[1]))
	yc := strings.Fields(unquote(s[2]))

	lx := len(xc)
	ly := len(yc)
	if lx != ly {
		return fmt.Errorf("line %d: %s coordinates are not the same length (x=%d, y=%d)", linenumber, s[0], lx, ly)
	}

	if lx < 3 || ly < 3 {
		return fmt.Errorf("line %d: %s needs at least 3 coordinates", linenumber, s[0])
	}

	// generate, eval x coordinates
	fmt.Fprintf(w, "<%s xc=\"", s[0])
	for i := 0; i < lx-1; i++ {
		fmt.Fprintf(w, "%s ", eval(xc[i]))
	}
	// generate, eval y coordinates
	fmt.Fprintf(w, "%s\" yc=\"", eval(xc[lx-1]))
	for i := 0; i < ly-1; i++ {
		fmt.Fprintf(w, "%s ", eval(yc[i]))
	}
	fmt.Fprintf(w, "%s\"", eval(yc[ly-1]))
	switch n {
	case 3:
		fmt.Fprintf(w, "/>\n")
	case 4:
		fmt.Fprintf(w, " color=%s/>\n", s[3])
	case 5:
		fmt.Fprintf(w, " color=%s opacity=%q/>\n", s[3], s[4])
	default:
		return e
	}
	return nil
}

// polyline makes a series of lines given a set of coordinantes
// polyline "xcoord" "ycoord" [lw] [color] [opacity]
func polyline(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s \"xcoord\" \"ycoord\" [lw] [color] [opacity]", linenumber, s[0])
	if n < 3 {
		return e
	}

	// get the coordinates from the quoted strings
	xc := strings.Fields(unquote(s[1]))
	yc := strings.Fields(unquote(s[2]))

	lx := len(xc)
	ly := len(yc)
	if lx != ly {
		return fmt.Errorf("line %d: %s coordinates are not the same length (x=%d, y=%d)", linenumber, s[0], lx, ly)
	}

	if lx < 3 || ly < 3 {
		return fmt.Errorf("line %d: %s needs at least 3 coordinates", linenumber, s[0])
	}

	var attr string
	switch n {
	case 3:
		attr = "/>"
	case 4:
		attr = fmt.Sprintf("sp=%q/>", s[3])
	case 5:
		attr = fmt.Sprintf("sp=%q color=%s/>", s[3], s[4])
	case 6:
		attr = fmt.Sprintf("sp=%q color=%s opacity=%q/>", s[3], s[4], s[5])
	default:
		return e
	}

	// generate, eval coordinates
	for i := 0; i < lx-1; i++ {
		fmt.Fprintf(w, "<line xp1=%q yp1=%q xp2=%q yp2=%q %s\n", eval(xc[i]), eval(yc[i]), eval(xc[i+1]), eval(yc[i+1]), attr)
	}
	fmt.Fprintf(w, "<line xp1=%q yp1=%q xp2=%q yp2=%q %s\n", eval(xc[0]), eval(yc[0]), eval(xc[lx-1]), eval(yc[lx-1]), attr)
	return nil
}

// line generates markup for lines
// line x1 y1 x2 y2 [size] [color] [opacity]
func line(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s x1 y1 x2 y2 [size] [color] [opacity]", linenumber, s[0])
	if n < 5 {
		return e
	}
	if err := validNumber(s[1], s[2], s[3], s[4]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	lc := fmt.Sprintf("xp1=%q yp1=%q xp2=%q yp2=%q", s[1], s[2], s[3], s[4])
	switch n {
	case 5:
		fmt.Fprintf(w, "<%s %s/>\n", s[0], lc)
	case 6:
		fmt.Fprintf(w, "<%s %s sp=%q/>\n", s[0], lc, s[5])
	case 7:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s/>\n", s[0], lc, s[5], s[6])
	case 8:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s opacity=%q/>\n", s[0], lc, s[5], s[6], s[7])
	default:
		return e
	}
	return nil
}

// hline makes a horizontal line
// hline x y length [size] [color] [opacity]
func hline(w io.Writer, s []string, linenumber int) error {
	e := fmt.Errorf("line %d: %s x y length [size] [color] [opacity]", linenumber, s[0])
	n := len(s)
	if n < 4 {
		return e
	}

	x1, err := strconv.ParseFloat(s[1], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v: (%v) %v", linenumber, err, s, x1)
	}

	l, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v: %v %v", linenumber, err, s, l)
	}
	if err := validNumber(s[2]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}

	if n > 4 {
		if err := validNumber(s[4]); err != nil {
			return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
		}
	}

	lc := fmt.Sprintf("xp1=%q yp1=%q xp2=\"%v\" yp2=%q", s[1], s[2], x1+l, s[2])
	switch n {
	case 4:
		fmt.Fprintf(w, "<line %s/>\n", lc)
	case 5:
		fmt.Fprintf(w, "<line %s sp=%q/>\n", lc, s[4])
	case 6:
		fmt.Fprintf(w, "<line %s sp=%q color=%s/>\n", lc, s[4], s[5])
	case 7:
		fmt.Fprintf(w, "<line %s sp=%q color=%s opacity=%q/>\n", lc, s[4], s[5], s[6])
	default:
		return e
	}
	return nil
}

// vline makes a vertical line
// vline x y length [size] [color] [opacity]
func vline(w io.Writer, s []string, linenumber int) error {
	e := fmt.Errorf("line %d: %s x y length [size] [color] [opacity]", linenumber, s[0])
	n := len(s)
	if n < 4 {
		return e
	}

	y1, err := strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v: (%v) %v", linenumber, err, s, y1)
	}
	l, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v: (%v) %v", linenumber, err, s, l)
	}

	if err := validNumber(s[1]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}

	if n > 4 {
		if err := validNumber(s[4]); err != nil {
			return fmt.Errorf("line %d: %v: %v", linenumber, s, n)
		}
	}

	lc := fmt.Sprintf("xp1=%q yp1=%q xp2=%q yp2=\"%v\"", s[1], s[2], s[1], y1+l)
	switch n {
	case 4:
		fmt.Fprintf(w, "<line %s/>\n", lc)
	case 5:
		fmt.Fprintf(w, "<line %s sp=%q/>\n", lc, s[4])
	case 6:
		fmt.Fprintf(w, "<line %s sp=%q color=%s/>\n", lc, s[4], s[5])
	case 7:
		fmt.Fprintf(w, "<line %s sp=%q color=%s opacity=%q/>\n", lc, s[4], s[5], s[6])
	default:
		return e
	}
	return nil
}

// arc makes the markup for arc
// arc cx cy w h a1 a2 [size] [color] [opacity]
func arc(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s cx cy w h a1 a2 [size] [color] [opacity]", linenumber, s[0])
	if n < 7 {
		return e
	}
	if err := validNumber(s[1], s[2], s[3], s[4], s[5], s[6]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	ac := fmt.Sprintf("xp=%q yp=%q wp=%q hp=%q a1=%q a2=%q", s[1], s[2], s[3], s[4], s[5], s[6])
	switch n {
	case 7:
		fmt.Fprintf(w, "<%s %s/>\n", s[0], ac)
	case 8:
		fmt.Fprintf(w, "<%s %s sp=%q/>\n", s[0], ac, s[7])
	case 9:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s/>\n", s[0], ac, s[7], s[8])
	case 10:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s opacity=%q/>\n", s[0], ac, s[7], s[8], s[9])
	default:
		return e
	}
	return nil
}

// curve make quadratic Bezier curve
// curve x1 y1 x2 y2 x3 y3 [size] [color] [opacity]
func curve(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	e := fmt.Errorf("line %d: %s x1 y1 x2 y2 x3 y3 [size] [color] [opacity]", linenumber, s[0])
	if n < 7 {
		return e
	}
	if err := validNumber(s[1], s[2], s[3], s[4], s[5], s[6]); err != nil {
		return fmt.Errorf("line %d: %v: %v", linenumber, err, s)
	}
	ac := fmt.Sprintf("xp1=%q yp1=%q xp2=%q yp2=%q xp3=%q yp3=%q", s[1], s[2], s[3], s[4], s[5], s[6])
	switch n {
	case 7:
		fmt.Fprintf(w, "<%s %s/>\n", s[0], ac)
	case 8:
		fmt.Fprintf(w, "<%s %s sp=%q/>\n", s[0], ac, s[7])
	case 9:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s/>\n", s[0], ac, s[7], s[8])
	case 10:
		fmt.Fprintf(w, "<%s %s sp=%q color=%s opacity=%q/>\n", s[0], ac, s[7], s[8], s[9])
	default:
		return e
	}
	return nil
}

// legend makes the markup for the legend keyword
// legend "text" x y size font color
func legend(w io.Writer, s []string, linenumber int) error {
	n := len(s)
	if n < 7 {
		return fmt.Errorf("line %d: legend \"text\" x y size font color", linenumber)
	}

	tx, err := strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	cy, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	ts, err := strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	tshalf := ts / 2
	fmt.Fprintf(w, "<text xp=%q yp=%q sp=%q %s>%s</text>\n", s[2], s[3], s[4], fontColorOp(s[5:]), qesc(s[1]))
	fmt.Fprintf(w, "<line xp1=%q yp1=%q xp2=%q yp2=%q sp=%q color=%s/>\n", ftoa(tx-tshalf), s[3], ftoa(tx-tshalf), ftoa(cy+ts), ftoa(tshalf), s[6])

	return nil
}

// bracket makes various kinds of braces: (left, right, up, down)
// lbracket x y width height [linewidth] [color] [opacity]
// rbracket x y width height [linewidth] [color] [opacity]
// ubracket x y width height [linewidth] [color] [opacity]
// dbracket x y width height [linewidth] [color] [opacity]
func bracket(w io.Writer, s []string, linenumber int) error {
	if len(s) < 5 {
		return fmt.Errorf("line %d: [l,r,u,d]bracket x y w h [linewidth] [color] [opacity]", linenumber)
	}
	var x, y, width, height, linewidth float64
	var err error

	x, err = strconv.ParseFloat(s[1], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[1])
	}
	y, err = strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[2])
	}
	width, err = strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[3])
	}
	height, err = strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[4])
	}

	linewidth = 0.2 // default
	// replace optional args, checking for validity
	attr := ""
	if len(s) >= 6 {
		lw, nerr := strconv.ParseFloat(s[5], 64)
		if nerr != nil {
			return fmt.Errorf("line %d: %s is not a number", linenumber, s[5])
		}
		linewidth = lw
		attr += "sp=\"" + s[5] + "\""
	}
	if len(s) >= 7 {
		attr += " color=" + s[6]
	}
	if len(s) >= 8 {
		if _, nerr := strconv.ParseFloat(s[7], 64); nerr != nil {
			return fmt.Errorf("line %d: %s is not a number", linenumber, s[7])
		}
		attr += " opacity=\"" + s[7] + "\""
	}
	switch s[0] {
	case "lbracket":
		lbracket(w, x, y, width, height, linewidth, attr)
	case "rbracket":
		rbracket(w, x, y, width, height, linewidth, attr)
	case "ubracket":
		ubracket(w, x, y, width, height, linewidth, attr)
	case "dbracket":
		dbracket(w, x, y, width, height, linewidth, attr)
	default:
		return fmt.Errorf("line %d: use lbracket (left), rbracket (right), ubracket (up), dbracket (down)", linenumber)
	}
	return nil
}

// lbracket makes a left-facing bracket
func lbracket(w io.Writer, x, y, width, height, lw float64, attr string) {
	hh := (height / 2)
	x2 := x + (width / 2)
	xw := x + width
	fmt.Fprintf(w, linefmt, x, y, x2, y, attr)
	fmt.Fprintf(w, linefmt, x2, y+hh, x2, y-hh, attr)
	fmt.Fprintf(w, sqfmt, x2, y+hh, lw, lw, attr) // join segment (top)
	fmt.Fprintf(w, sqfmt, x2, y-hh, lw, lw, attr) // join segment (bottom)
	fmt.Fprintf(w, linefmt, x2, y+hh, xw, y+hh, attr)
	fmt.Fprintf(w, linefmt, x2, y-hh, xw, y-hh, attr)
}

// rbracket makes a right-facing bracket
func rbracket(w io.Writer, x, y, width, height, lw float64, attr string) {
	hh := height / 2
	x2 := x - (width / 2)
	xw := x - width
	fmt.Fprintf(w, linefmt, x, y, x2, y, attr)
	fmt.Fprintf(w, linefmt, x2, y+hh, x2, y-hh, attr)
	fmt.Fprintf(w, sqfmt, x2, y+hh, lw, lw, attr) // join segment (top)
	fmt.Fprintf(w, sqfmt, x2, y-hh, lw, lw, attr) // join segment (bottom)
	fmt.Fprintf(w, linefmt, x2, y+hh, xw, y+hh, attr)
	fmt.Fprintf(w, linefmt, x2, y-hh, xw, y-hh, attr)
}

// ubracket makes a up-facing bracket
func ubracket(w io.Writer, x, y, width, height, lw float64, attr string) {
	miter := lw / 2 // join the segments
	hh := height / 2
	wh := width / 2
	yh := y - hh
	fmt.Fprintf(w, linefmt, x, y, x, yh, attr)
	fmt.Fprintf(w, linefmt, x-wh-miter, yh, x+wh+miter, yh, attr) // join
	fmt.Fprintf(w, linefmt, x-wh, yh, x-wh, y-height, attr)
	fmt.Fprintf(w, linefmt, x+wh, yh, x+wh, y-height, attr)
}

// dbracket makes a down-facing bracket
func dbracket(w io.Writer, x, y, width, height, lw float64, attr string) {
	miter := lw / 2 // join the segments
	hh := height / 2
	wh := width / 2
	yh := y + hh
	fmt.Fprintf(w, linefmt, x, y, x, yh, attr)
	fmt.Fprintf(w, linefmt, x-wh-miter, yh, x+wh+miter, yh, attr) // join
	fmt.Fprintf(w, linefmt, x-wh, yh, x-wh, y+height, attr)
	fmt.Fprintf(w, linefmt, x+wh, yh, x+wh, y+height, attr)
}

// brace makes various kinds of braces: (left, right, up, down)
// lbrace x y size aw ah [linewidth] [color] [opacity]
// rbrace x y size aw ah [linewidth] [color] [opacity]
// ubrace x y size aw ah [linewidth] [color] [opacity]
// dbrace x y size aw ah [linewidth] [color] [opacity]
func brace(w io.Writer, s []string, linenumber int) error {
	if len(s) < 6 {
		return fmt.Errorf("line %d: [l,r,u,d]brace x y size aw ah [linewidth] [color] [opacity]", linenumber)
	}
	var x, y, size, aw, ah float64
	var err error

	x, err = strconv.ParseFloat(s[1], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[1])
	}
	y, err = strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[2])
	}
	size, err = strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[3])
	}
	aw, err = strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[5])
	}
	ah, err = strconv.ParseFloat(s[5], 64)
	if err != nil {
		return fmt.Errorf("line %d: %s is not a number", linenumber, s[6])
	}

	// replace optional args, checking for validity
	attr := ""
	if len(s) >= 7 {
		if _, nerr := strconv.ParseFloat(s[6], 64); nerr != nil {
			return fmt.Errorf("line %d: %s is not a number", linenumber, s[6])
		}
		attr += "sp=\"" + s[6] + "\""
	}
	if len(s) >= 8 {
		attr += " color=" + s[7]
	}
	if len(s) >= 9 {
		if _, nerr := strconv.ParseFloat(s[8], 64); nerr != nil {
			return fmt.Errorf("line %d: %s is not a number", linenumber, s[8])
		}
		attr += " opacity=\"" + s[8] + "\""
	}
	switch s[0] {
	case "lbrace":
		lbrace(w, x, y, size, aw, ah, attr)
	case "rbrace":
		rbrace(w, x, y, size, aw, ah, attr)
	case "ubrace":
		ubrace(w, x, y, size, aw, ah, attr)
	case "dbrace":
		dbrace(w, x, y, size, aw, ah, attr)
	default:
		return fmt.Errorf("line %d: use lbrace (left), rbrace (right), ubrace (up), dbrace (down)", linenumber)
	}
	return nil
}

// lbrace makes a left-facing brace
func lbrace(w io.Writer, x, y, size, aw, ah float64, attr string) {
	aw2 := aw / 2
	h2 := size / 2
	linelen := h2 - (ah / 2)
	xshift := x + aw2
	fmt.Fprintf(w, curvefmt, x, y, xshift, y, xshift, y+ah, attr)
	fmt.Fprintf(w, curvefmt, x, y, xshift, y, xshift, y-ah, attr)
	fmt.Fprintf(w, curvefmt, xshift, y+linelen, xshift, y+h2, x+aw, y+h2, attr)
	fmt.Fprintf(w, curvefmt, xshift, y-linelen, xshift, y-h2, x+aw, y-h2, attr)
	fmt.Fprintf(w, linefmt, xshift, y+ah, xshift, y+linelen, attr)
	fmt.Fprintf(w, linefmt, xshift, y-ah, xshift, y-linelen, attr)
}

// rbrace makes a right-facing brace
func rbrace(w io.Writer, x, y, size, aw, ah float64, attr string) {
	aw2 := aw / 2
	h2 := size / 2
	linelen := h2 - (ah / 2)
	xshift := x - aw2
	fmt.Fprintf(w, curvefmt, x, y, xshift, y, xshift, y+ah, attr)
	fmt.Fprintf(w, curvefmt, x, y, xshift, y, xshift, y-ah, attr)
	fmt.Fprintf(w, curvefmt, xshift, y+linelen, xshift, y+h2, x-aw, y+h2, attr)
	fmt.Fprintf(w, curvefmt, xshift, y-linelen, xshift, y-h2, x-aw, y-h2, attr)
	fmt.Fprintf(w, linefmt, xshift, y+ah, xshift, y+linelen, attr)
	fmt.Fprintf(w, linefmt, xshift, y-ah, xshift, y-linelen, attr)
}

// ubrace makes a upwards-facing brace
func ubrace(w io.Writer, x, y, size, aw, ah float64, attr string) {
	linelen := (size / 2) - aw
	yshift := y - (ah / 2)
	fmt.Fprintf(w, curvefmt, x, y, x, yshift, x+aw, yshift, attr)
	fmt.Fprintf(w, curvefmt, x, y, x, yshift, x-aw, yshift, attr)
	fmt.Fprintf(w, linefmt, x+aw, yshift, x+linelen, yshift, attr)
	fmt.Fprintf(w, linefmt, x-aw, yshift, x-linelen, yshift, attr)
	fmt.Fprintf(w, curvefmt, x+linelen, yshift, x+linelen+aw, yshift, x+linelen+aw, y-(ah), attr)
	fmt.Fprintf(w, curvefmt, x-linelen, yshift, x-linelen-aw, yshift, x-linelen-aw, y-(ah), attr)
}

// lbrace makes a downward-facing brace
func dbrace(w io.Writer, x, y, size, aw, ah float64, attr string) {
	linelen := (size / 2) - aw
	yshift := y + (ah / 2)
	fmt.Fprintf(w, curvefmt, x, y, x, yshift, x+aw, yshift, attr)
	fmt.Fprintf(w, curvefmt, x, y, x, yshift, x-aw, yshift, attr)
	fmt.Fprintf(w, linefmt, x+aw, yshift, x+linelen, yshift, attr)
	fmt.Fprintf(w, linefmt, x-aw, yshift, x-linelen, yshift, attr)
	fmt.Fprintf(w, curvefmt, x+linelen, yshift, x+linelen+aw, yshift, x+linelen+aw, y+(ah), attr)
	fmt.Fprintf(w, curvefmt, x-linelen, yshift, x-linelen-aw, yshift, x-linelen-aw, y+(ah), attr)
}

// angle computes the angle formed by a line, compensating for aspect ratio
func angle(x1, y1, x2, y2 float64) float64 {
	aspect := (canvasHeight / canvasWidth)
	return math.Atan2((aspect*y2)-(aspect*y1), x2-x1)
}

// rt returns the distance and angle of a line, compensating for aspect ratio
func rt(x1, y1, x2, y2 float64) (float64, float64) {
	aspect := (canvasHeight / canvasWidth)
	dx := x2 - x1
	dy := (aspect * y2) - (aspect * y1)
	return math.Sqrt((dx * dx) + (dy * dy)), math.Atan2(dy, dx)
}

// polar converts polar to Cartesian coordinates, compensating for canvas aspect ratio
func polar(cx, cy, r, t float64) (float64, float64) {
	ry := r * (canvasWidth / canvasHeight)
	return ((r * math.Cos(t)) + (cx)), ((ry * math.Sin(t)) + (cy))
}

// wpolar converts polar to Cartesian coordinates
//func wpolar(cx, cy, r, t float64) (float64, float64) {
//	return ((r * math.Cos(t)) + (cx)), ((r * math.Sin(t)) + (cy))
//}

// rotatexy computes the coordinates rotated around t
//func rotatexy(cx, cy, x, y, t float64) (float64, float64) {
//	return cx + (x * math.Cos(t)) - (y * math.Sin(t)), cy + (x * math.Sin(t)) - (y * math.Cos(t))
//}

// genarrow returns the components of an arrow
func genarrow(x1, y1, x2, y2, aw, ah float64) (float64, float64, float64, float64, float64, float64, float64, float64) {
	r, t := rt(x1, y1, x2, y2)
	n := r - (aw * stdnotch)
	nt := angle(x1, y1, x1+n, (y1 + (ah / 2)))
	ax1, ay1 := polar(x1, y1, r, t)
	ax2, ay2 := polar(x1, y1, r-aw, t+nt)
	ax3, ay3 := polar(x1, y1, n, t)
	ax4, ay4 := polar(x1, y1, r-aw, t-nt)

	return ax1, ay1, ax2, ay2, ax3, ay3, ax4, ay4
}

// arrow draws a general arrow given two points.
// The rotation of the arrowhead is computed.
// arrow x1 y1 x2 y2 [linewidth] [arrowidth] [arrowheight] [color] [opacity]
func arrow(w io.Writer, s []string, linenumber int) error {
	ls := len(s)
	e := fmt.Errorf("line: %d arrow x1 y1 x2 y2 [linewidth] [arrowidth] [arrowheight] [color] [opacity]", linenumber)
	if ls < 5 {
		return e
	}
	aw := 3.0
	ah := 3.0
	lw := "0.2"
	color := `"gray"`
	opacity := "100"

	x1, err := strconv.ParseFloat(s[1], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	y1, err := strconv.ParseFloat(s[2], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	x2, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	y2, err := strconv.ParseFloat(s[4], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	if ls >= 6 {
		lw = s[5] // linewidth
	}
	if ls >= 7 {
		aw, err = strconv.ParseFloat(s[6], 64)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
	}
	if ls >= 8 {
		ah, err = strconv.ParseFloat(s[7], 64)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
	}
	if ls >= 9 {
		color = s[8] // color
	}
	if ls == 10 {
		opacity = s[9] // opacity
	}
	ax1, ay1, ax2, ay2, ax3, ay3, ax4, ay4 := genarrow(x1, y1, x2, y2, aw, ah)
	fmt.Fprintf(w, "<line xp1=%q yp1=%q xp2=\"%v\" yp2=\"%v\" sp=%q color=%s opacity=%q/>\n", s[1], s[2], ax3, ay3, lw, color, opacity)
	fmt.Fprintf(w, "<polygon xc=\"%v %v %v %v\" yc=\"%v %v %v %v\" color=%s opacity=%q/>\n", ax1, ax2, ax3, ax4, ay1, ay2, ay3, ay4, color, opacity)
	return nil
}

// arrowhead returns the coordinates for left, right, up, down arrowheads.
// x, y is the point of the arrow, aw, ah are width, height
func arrowhead(x, y, ah, aw, notch float64, arrowtype byte) (float64, float64, float64, float64, float64, float64, float64, float64) {
	var ax1, ax2, ax3, ax4, ay1, ay2, ay3, ay4 float64
	switch arrowtype {
	case 'r':
		ax1 = x
		ax2 = ax1 - aw
		ax3 = x - (aw * notch)
		ax4 = ax2

		ay1 = y
		ay2 = y + (ah / 2)
		ay3 = y
		ay4 = y - (ah / 2)
	case 'l':
		ax1 = x
		ax2 = ax1 + aw
		ax3 = x + (aw * notch)
		ax4 = ax2

		ay1 = y
		ay2 = y + (ah / 2)
		ay3 = y
		ay4 = y - (ah / 2)
	case 'u':
		ax1 = x
		ax2 = x + (aw / 2)
		ax3 = x
		ax4 = x - (aw / 2)

		ay1 = y
		ay2 = ay1 - ah
		ay3 = ay1 - (ah * notch)
		ay4 = ay2
	case 'd':
		ax1 = x
		ax2 = x + (aw / 2)
		ax3 = x
		ax4 = x - (aw / 2)

		ay1 = y
		ay2 = ay1 + ah
		ay3 = ay1 + (ah * notch)
		ay4 = ay2
	}
	return ax1, ax2, ax3, ax4, ay1, ay2, ay3, ay4
}

// carrow makes a arrow with a curved line
// lcarrow x1 y1 x2 y2 x3 y3 [linewidth] [arrowidth] [arrowheight] [color] [opacity]
// rcarrow x1 y1 x2 y2 x3 y3 [linewidth] [arrowidth] [arrowheight] [color] [opacity]
// ucarrow x1 y1 x2 y2 x3 y3 [linewidth] [arrowidth] [arrowheight] [color] [opacity]
// dcarrow x1 y1 x2 y2 x3 y3 [linewidth] [arrowidth] [arrowheight] [color] [opacity]
func carrow(w io.Writer, s []string, linenumber int) error {
	ls := len(s)
	e := fmt.Errorf("line: %d [l|r|u|d]carrow x1 y1 x2 y2 x3 y3 [linewidth] [arrowidth] [arrowheight] [color] [opacity]", linenumber)
	if len(s[0]) < 7 {
		return e
	}
	if ls < 7 {
		return e
	}
	aw := 3.0
	ah := 3.0

	color := `"gray"`
	opacity := "100"

	// copy the curve portion
	curvestring := make([]string, 10)
	curvestring[0] = "curve"
	for i := 1; i < 7; i++ {
		curvestring[i] = s[i]
	}
	// set defaults for linewidth, color, and opacity
	curvestring[7] = "0.2"
	curvestring[8] = color
	curvestring[9] = opacity

	// override settings for  linewidth, color, and opacity
	if ls >= 8 {
		curvestring[7] = s[7] // linewidth
	}
	if ls >= 11 {
		color = s[10]
		curvestring[8] = color // color
	}
	if ls == 12 {
		opacity = s[11]
		curvestring[9] = opacity // opacity
	}

	// end point of the curve is the point of the arrow
	x, err := strconv.ParseFloat(s[5], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	y, err := strconv.ParseFloat(s[6], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}

	// override width and height of the arrow
	if ls >= 9 {
		aw, err = strconv.ParseFloat(s[8], 64)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
	}
	if ls >= 10 {
		ah, err = strconv.ParseFloat(s[9], 64)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
	}
	// compute the coordinates for the arrowhead
	ax1, ax2, ax3, ax4, ay1, ay2, ay3, ay4 := arrowhead(x, y, ah, aw, stdnotch, s[0][0])
	// adjust the end point of the curve to be the notch point
	curvestring[5] = ftoa(ax3)
	curvestring[6] = ftoa(ay3)

	curve(w, curvestring, linenumber)
	fmt.Fprintf(w, "<polygon xc=\"%v %v %v %v\" yc=\"%v %v %v %v\" color=%s opacity=%q/>\n", ax1, ax2, ax3, ax4, ay1, ay2, ay3, ay4, color, opacity)
	return nil
}

// chartflags sets the flag for the dchart keyword
func chartflags(s []string) dchart.Settings {
	var chart dchart.Settings

	fs := flag.NewFlagSet(s[0], flag.ContinueOnError)

	// Measures
	fs.Float64Var(&chart.TextSize, "textsize", 1.5, "text size")
	fs.Float64Var(&chart.CanvasWidth, "cw", 792, "canvas width")
	fs.Float64Var(&chart.CanvasHeight, "ch", 612, "canvas height")
	fs.Float64Var(&chart.Left, "left", -1, "left margin")
	fs.Float64Var(&chart.Right, "right", 90.0, "right margin")
	fs.Float64Var(&chart.Top, "top", 80.0, "top of the plot")
	fs.Float64Var(&chart.Bottom, "bottom", 30.0, "bottom of the plot")
	fs.Float64Var(&chart.LineSpacing, "ls", 2.4, "ls")
	fs.Float64Var(&chart.BarWidth, "barwidth", 0, "barwidth")
	fs.Float64Var(&chart.UserMin, "min", -1, "minimum")
	fs.Float64Var(&chart.UserMax, "max", -1, "maximum")
	fs.Float64Var(&chart.PSize, "psize", 40.0, "size of the donut")
	fs.Float64Var(&chart.PWidth, "pwidth", chart.Measures.TextSize*3, "width of the pmap/donut/radial")
	fs.Float64Var(&chart.LineWidth, "linewidth", 0.2, "width of line for line charts")
	fs.Float64Var(&chart.VolumeOpacity, "volop", 50, "volume opacity")
	fs.Float64Var(&chart.XLabelRotation, "xlabrot", 0, "xlabel rotation (degrees)")
	fs.IntVar(&chart.XLabelInterval, "xlabel", 1, "x axis label interval (show every n labels, 0 to show no labels)")
	fs.IntVar(&chart.PMapLength, "pmlen", 20, "pmap label length")
	fs.StringVar(&chart.Boundary, "bounds", "", "chart boundary: left,right,top,bottom")

	// Flags (On/Off)
	fs.BoolVar(&chart.ShowBar, "bar", true, "show a bar chart")
	fs.BoolVar(&chart.ShowDot, "dot", false, "show a dot chart")
	fs.BoolVar(&chart.ShowVolume, "vol", false, "show a volume chart")
	fs.BoolVar(&chart.ShowDonut, "donut", false, "show a donut chart")
	fs.BoolVar(&chart.ShowBowtie, "bowtie", false, "show a bowtie chart")
	fs.BoolVar(&chart.ShowFan, "fan", false, "show a fan chart")
	fs.BoolVar(&chart.ShowPMap, "pmap", false, "show a proportional map")
	fs.BoolVar(&chart.ShowLine, "line", false, "show a line chart")
	fs.BoolVar(&chart.ShowHBar, "hbar", false, "show a horizontal bar chart")
	fs.BoolVar(&chart.ShowValues, "val", true, "show data values")
	fs.BoolVar(&chart.ShowAxis, "yaxis", false, "show y axis")
	fs.BoolVar(&chart.ShowSlope, "slope", false, "show a slope graph")
	fs.BoolVar(&chart.ShowTitle, "title", true, "show title")
	fs.BoolVar(&chart.ShowGrid, "grid", false, "show y axis grid")
	fs.BoolVar(&chart.ShowScatter, "scatter", false, "show scatter chart")
	fs.BoolVar(&chart.ShowRadial, "radial", false, "show a radial chart")
	fs.BoolVar(&chart.ShowSpokes, "spokes", false, "show spokes on radial charts")
	fs.BoolVar(&chart.ShowPGrid, "pgrid", false, "show proportional grid")
	fs.BoolVar(&chart.ShowLego, "lego", false, "show lego chart")
	fs.BoolVar(&chart.ShowNote, "note", true, "show annotations")
	fs.BoolVar(&chart.ShowFrame, "frame", false, "show frame")
	fs.BoolVar(&chart.ShowRegressionLine, "rline", false, "show regression line")
	fs.BoolVar(&chart.ShowXLast, "xlast", false, "show the last label")
	fs.BoolVar(&chart.ShowXstagger, "xstagger", false, "stagger x axis labels")
	fs.BoolVar(&chart.FullDeck, "fulldeck", false, "generate full markup")
	fs.BoolVar(&chart.DataMinimum, "dmin", false, "zero minimum")
	fs.BoolVar(&chart.ReadCSV, "csv", false, "read CSV data")
	fs.BoolVar(&chart.ShowWBar, "wbar", false, "show word bar chart")
	fs.BoolVar(&chart.ShowPercentage, "pct", false, "show computed percentages with values")
	fs.BoolVar(&chart.SolidPMap, "solidpmap", false, "solid pmap colors")

	// Attributes
	fs.StringVar(&chart.ChartTitle, "chartitle", "", "specify the title (overiding title in the data)")
	fs.StringVar(&chart.CSVCols, "csvcol", "", "label,value from the CSV header")
	fs.StringVar(&chart.ValuePosition, "valpos", "t", "value position (t=top, b=bottom, m=middle)")
	fs.StringVar(&chart.LabelColor, "lcolor", "rgb(75,75,75)", "label color")
	fs.StringVar(&chart.DataColor, "color", "lightsteelblue", "data color")
	fs.StringVar(&chart.ValueColor, "vcolor", "rgb(127,0,0)", "value color")
	fs.StringVar(&chart.RegressionLineColor, "rlcolor", "rgb(127,0,0)", "regression line color")
	fs.StringVar(&chart.FrameColor, "framecolor", "rgb(127,127,127)", "framecolor")
	fs.StringVar(&chart.BackgroundColor, "bgcolor", "white", "background color")
	fs.StringVar(&chart.DataFmt, "datafmt", dchart.Defaultfmt, "data format")
	fs.StringVar(&chart.YAxisR, "yrange", "", "y-axis range (min,max,step)")
	fs.StringVar(&chart.HLine, "hline", "", "horizontal line value,label")
	fs.StringVar(&chart.NoteLocation, "noteloc", "c", "note location (c-center, r-right aligned, l-left aligned)")
	fs.StringVar(&chart.DataCondition, "datacond", "", "data condition: low,high,color")
	fs.Parse(s[1:])
	if len(chart.Boundary) > 0 {
		chart.Left, chart.Right, chart.Top, chart.Bottom = dchart.Parsebounds(chart.Boundary)
	}
	return chart
}

// chart uses the dchart API to make charts
// dchart [args]
func chart(w io.Writer, s string, linenumber int) error {
	// copy the command line into fields, evaluating as we go
	args := strings.Fields(s)
	if len(args) < 2 {
		return fmt.Errorf("line %d: length of args = %d", linenumber, len(args))
	}
	for i := 1; i < len(args); i++ {
		args[i] = eval(args[i])
		args[i] = unquote(args[i])
	}
	// fmt.Fprintf(os.Stderr, "line %d - chart args=%v\n", linenumber, args)
	// glue the arguments back into a single string
	s = args[0]
	for i := 1; i < len(args); i++ {
		s = s + " " + args[i]
	}
	// separate again
	args = strings.Fields(s)
	chartsettings := chartflags(args)
	r, err := os.Open(args[len(args)-1]) // last arg is the filename
	if err != nil {
		return fmt.Errorf("line %d: %v", linenumber, err)
	}
	chartsettings.Write(w, r)
	return nil
}
