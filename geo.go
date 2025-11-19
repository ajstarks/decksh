// Package decksh is a little language that generates deck markup
// geo data processing
package decksh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	DataFormat int = iota
	UNK
	KML
	SHP
	JSON
)

const (
	dotfmt      = "<ellipse xp=\"%.3f\" yp=\"%.3f\" wp=\"%.3f\" hr=\"100\" color=%s opacity=\"%v\"/>\n"
	decklinefmt = "<line xp1=\"%.5f\" yp1=\"%.5f\" xp2=\"%.5f\" yp2=\"%.5f\" sp=\"%.5f\" color=%s opacity=%q/>\n"
	textfmt     = "<text align=%s xp=\"%.3f\" yp=\"%.3f\" sp=\"%.3f\" %s>%s</text>\n"
	geoimgfmt   = "<image name=\"%s\" xp=\"%.3f\" yp=\"%.3f\" width=%q height=%q/>\n"
	shpdotfmt   = "<ellipse xp=\"%.7f\" yp=\"%.7f\" hr=\"100\" color=%q opacity=%q wp=\"%.3f\"/>\n"
)

// geometry defines the canvas and map boundaries
type Geometry struct {
	Xmin, Xmax       float64
	Ymin, Ymax       float64
	Latmin, Latmax   float64
	Longmin, Longmax float64
}

// locdata is location data: coordinates and a label
type Locdata struct {
	X, Y []float64
	Name []string
}

// evaluate reserved variables for canvas bounds (geoXmin, geoXmax, geoYmin, geoYmax)
// if errors occur use the defaults
func geocanvas() (float64, float64, float64, float64) {
	xmin, err := strconv.ParseFloat(eval("geoXmin"), 64)
	if err != nil {
		xmin = 0.0
	}
	xmax, err := strconv.ParseFloat(eval("geoXmax"), 64)
	if err != nil {
		xmax = 100.0
	}
	ymin, err := strconv.ParseFloat(eval("geoYmin"), 64)
	if err != nil {
		ymin = 0.0
	}
	ymax, err := strconv.ParseFloat(eval("geoYmax"), 64)
	if err != nil {
		ymax = 100.0
	}
	return xmin, xmax, ymin, ymax
}

// evaluate reserved variables for set lat/long bounds,
// if errors occur use the defaults
func geolatlong() (float64, float64, float64, float64) {
	latmin, err := strconv.ParseFloat(eval("geoLatMin"), 64)
	if err != nil {
		latmin = -90.0
	}
	latmax, err := strconv.ParseFloat(eval("geoLatMax"), 64)
	if err != nil {
		latmax = 90.0
	}
	longmin, err := strconv.ParseFloat(eval("geoLongMin"), 64)
	if err != nil {
		longmin = -180.0
	}
	longmax, err := strconv.ParseFloat(eval("geoLongMax"), 64)
	if err != nil {
		longmax = 180.0
	}
	return latmin, latmax, longmin, longmax
}

// makegeometry fills in the geometry from arguments
func makegeometry() Geometry {
	var m Geometry
	m.Xmin, m.Xmax, m.Ymin, m.Ymax = geocanvas()
	m.Latmin, m.Latmax, m.Longmin, m.Longmax = geolatlong()
	return m
}

// mapData maps raw lat/long coordinates to canvas coordinates
func mapData(x, y []float64, g Geometry) ([]float64, []float64) {
	for i := range x {
		x[i] = vmap(x[i], g.Longmin, g.Longmax, g.Xmin, g.Xmax)
		y[i] = vmap(y[i], g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	}
	return x, y
}

// geoDataFormat looks at a filename and returns the format type
// based on the suffix
func geoDataFormat(s string) int {
	i := strings.LastIndex(s, ".")
	if i < 0 {
		return UNK
	}
	switch strings.ToLower(s[i:]) {
	case ".shp":
		return SHP
	case ".kml":
		return KML
	case ".json", ".geojson":
		return JSON
	default:
		return UNK
	}
}

// readLoc reads lat/long pairs and an optional name from a file
func readLoc(r io.Reader, sep byte) (Locdata, error) {
	var data Locdata
	s := bufio.NewScanner(r)
	ff := func(c rune) bool { return c == rune(sep) }
	for s.Scan() {
		t := s.Text()
		if strings.HasPrefix(t, "geo:") { // convert geom: URI to tab-separated list
			t = strings.Replace(t[4:], ",", "\t", 1)
		}
		f := strings.FieldsFunc(t, ff)
		if len(f) < 2 { // not enough fields
			continue
		}
		xp, err := strconv.ParseFloat(f[1], 64) // long
		if err != nil {
			continue
		}
		yp, err := strconv.ParseFloat(f[0], 64) // lat
		if err != nil {
			continue
		}
		data.X = append(data.X, xp)
		data.Y = append(data.Y, yp)
		if len(f) > 2 { // if name is present
			data.Name = append(data.Name, f[2])
		} else {
			data.Name = append(data.Name, "")
		}
	}
	return data, s.Err()
}

// colorop makes a color and optional opacity in the form of name:op
func colorop(color string) (string, string) {
	ci := strings.Index(color, ":")
	op := "100"
	if ci > 0 && ci < len(color) {
		op = color[ci+1:]
		color = color[0:ci]
	}
	return color, op
}

// geodot places dots at geometric coordinates
func geodot(w io.Writer, x, y []float64, size float64, color string, op string) {
	nc := len(x)
	if nc != len(y) {
		return
	}
	for i := range nc {
		fmt.Fprintf(w, dotfmt, x[i], y[i], size, color, op)
	}
}

// deckgeoimg places images at geometric coordinates
func deckgeoimg(w io.Writer, loc Locdata, width, height string) {
	nc := len(loc.X)
	for i := range nc {
		if len(loc.Name[i]) < 0 {
			fmt.Fprintf(os.Stderr, "missing image file")
			continue
		}
		fmt.Fprintf(w, geoimgfmt,
			loc.Name[i], loc.X[i], loc.Y[i], width, height)
	}
}

// textadj adjusts alignment of text
func textadj(align string, size float64) (float64, float64) {
	var xdiff, ydiff float64
	size /= 2
	switch align {
	case "c", "ctext", "a":
		ydiff = size
	case "u":
		ydiff = -size * 2
	case "b", "btext", "text":
		xdiff = size * 0.75
		ydiff = -size * 0.6
	case "e", "etext":
		xdiff = -size * 0.75
		ydiff = -size * 0.6
	default:
		ydiff = size
	}
	return xdiff, ydiff
}

// wordstack makes a vertical stack of string in s
func wordstack(w io.Writer, x, y float64, s []string, align string, size float64, fco string) {
	ls := size * 1.8
	for i := range s {
		fmt.Fprintf(w, textfmt, align, x, y, size, fco, xmlesc(s[i]))
		y -= ls
	}
}

// geotext places text at geographic coordinates
func geotext(w io.Writer, x, y []float64, names []string, align string, size float64, fco string) {
	xdiff, ydiff := textadj(align, size)
	if align == "u" || align == "a" { // above and under are centered
		align = "c"
	}
	for i := range x {
		parts := strings.Split(names[i], "\\n")
		wordstack(w, x[i]+xdiff, y[i]+ydiff, parts, align, size, fco)
	}
}

func geopl(w io.Writer, x, y []float64, color string, shapesize float64) {
	if shapesize > 0 {
		shpdeckpolyline(w, x, y, color, shapesize)
	} else {
		deckpolygon(w, x, y, color)
	}
}
