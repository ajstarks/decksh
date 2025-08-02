// Package decksh is a little language that generates deck markup
// geo data processing
package decksh

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	dotfmt      = "<ellipse xp=\"%.3f\" yp=\"%.3f\" wp=\"%.3f\" hr=\"100\" color=\"%s\" opacity=\"%v\"/>\n"
	decklinefmt = "<line xp1=\"%.5f\" yp1=\"%.5f\" xp2=\"%.5f\" yp2=\"%.5f\" sp=\"%.5f\" color=\"%s\" opacity=\"%s\"/>\n"
	textfmt     = "<text align=\"%s\" xp=\"%.3f\" yp=\"%.3f\" sp=\"%.3f\" %s>%s</text>\n"
)

// geometry defines the canvas and map boundaries
type Geometry struct {
	Xmin, Xmax       float64
	Ymin, Ymax       float64
	Latmin, Latmax   float64
	Longmin, Longmax float64
}

// locdata
type Locdata struct {
	X, Y []float64
	Name []string
}

// KML Structure
type Kml struct {
	XMLName  xml.Name `xml:"kml"`
	Text     string   `xml:",chardata"`
	Xmlns    string   `xml:"xmlns,attr"`
	Document struct {
		Text   string `xml:",chardata"`
		Schema struct {
			Text        string `xml:",chardata"`
			Name        string `xml:"name,attr"`
			ID          string `xml:"id,attr"`
			SimpleField []struct {
				Text        string `xml:",chardata"`
				Type        string `xml:"type,attr"`
				Name        string `xml:"name,attr"`
				DisplayName string `xml:"displayName"`
			} `xml:"SimpleField"`
		} `xml:"Schema"`
		Placemark []struct {
			Text    string `xml:",chardata"`
			Name    string `xml:"name"`
			Polygon struct {
				Text            string `xml:",chardata"`
				OuterBoundaryIs struct {
					Text       string `xml:",chardata"`
					LinearRing struct {
						Text        string `xml:",chardata"`
						Coordinates string `xml:"coordinates"`
					} `xml:"LinearRing"`
				} `xml:"outerBoundaryIs"`
				InnerBoundaryIs struct {
					Text       string `xml:",chardata"`
					LinearRing struct {
						Text        string `xml:",chardata"`
						Coordinates string `xml:"coordinates"`
					} `xml:"LinearRing"`
				} `xml:"innerBoundaryIs"`
			} `xml:"Polygon"`
			ExtendedData struct {
				Text       string `xml:",chardata"`
				SchemaData struct {
					Text       string `xml:",chardata"`
					SchemaUrl  string `xml:"schemaUrl,attr"`
					SimpleData []struct {
						Text string `xml:",chardata"`
						Name string `xml:"name,attr"`
					} `xml:"SimpleData"`
				} `xml:"SchemaData"`
			} `xml:"ExtendedData"`
			MultiGeometry struct {
				Text    string `xml:",chardata"`
				Polygon []struct {
					Text            string `xml:",chardata"`
					OuterBoundaryIs struct {
						Text       string `xml:",chardata"`
						LinearRing struct {
							Text        string `xml:",chardata"`
							Coordinates string `xml:"coordinates"`
						} `xml:"LinearRing"`
					} `xml:"outerBoundaryIs"`
					InnerBoundaryIs []struct {
						Text       string `xml:",chardata"`
						LinearRing struct {
							Text        string `xml:",chardata"`
							Coordinates string `xml:"coordinates"`
						} `xml:"LinearRing"`
					} `xml:"innerBoundaryIs"`
				} `xml:"Polygon"`
			} `xml:"MultiGeometry"`
		} `xml:"Placemark"`
	} `xml:"Document"`
}

// ParseCoords makes x, y slices from the string data contained in the kml coordinate element
// (lat,long,elevation separated by commas, each coordinate separated by spaces)
// The coordinates are mapped to a canvas bounding box in g.
func parseCoords(s string, g Geometry) ([]float64, []float64) {
	f := strings.Fields(s)
	n := len(f)
	x := make([]float64, n)
	y := make([]float64, n)
	var xp, yp float64
	for i, c := range f {
		coords := strings.Split(c, ",")
		xp, _ = strconv.ParseFloat(coords[0], 64)
		yp, _ = strconv.ParseFloat(coords[1], 64)
		x[i] = vmap(xp, g.Longmin, g.Longmax, g.Xmin, g.Xmax)
		y[i] = vmap(yp, g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	}
	return x, y
}

// evaluate "magic" reserved variables geoXmin, geoXmax, geoYmin, geoYmax
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

// set lat/long bounds, if errors occur use the defaults
func geolatlong(s []string) (float64, float64, float64, float64) {
	latmin, err := strconv.ParseFloat(eval(s[2]), 64)
	if err != nil {
		latmin = -90.0
	}
	latmax, err := strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		latmax = 90.0
	}
	longmin, err := strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		longmin = -180.0
	}
	longmax, err := strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		longmax = 180.0
	}
	return latmin, latmax, longmin, longmax

}

// makegeometry fills in the geometry from arguments
func makegeometry(s []string) (Geometry, error) {
	var m Geometry
	m.Xmin, m.Xmax, m.Ymin, m.Ymax = geocanvas()
	m.Latmin, m.Latmax, m.Longmin, m.Longmax = geolatlong(s)
	return m, nil
}

// mapData maps raw lat/long coordinates to canvas coordinates
func mapData(x, y []float64, g Geometry) ([]float64, []float64) {
	for i := 0; i < len(x); i++ {
		x[i] = vmap(x[i], g.Longmin, g.Longmax, g.Xmin, g.Xmax)
		y[i] = vmap(y[i], g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	}
	return x, y
}

// readData loads the KML structure from a file
func readKMLData(filename string) (Kml, error) {
	var data Kml
	r, err := os.Open(filename)
	if err != nil {
		return data, err
	}
	err = xml.NewDecoder(r).Decode(&data)
	r.Close()
	return data, err
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
	for i := 0; i < nc; i++ {
		fmt.Fprintf(w, dotfmt, x[i], y[i], size, color, op)
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

func wordstack(w io.Writer, x, y float64, s []string, align string, size float64, fco string) {
	ls := size * 1.2
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(w, textfmt, align, x, y, size, fco, xmlesc(s[i]))
		y -= ls
	}
}

func geotext(w io.Writer, x, y []float64, names []string, align string, size float64, fco string) {
	xdiff, ydiff := textadj(align, size)
	if align == "u" || align == "a" { // above and under are centered
		align = "c"
	}
	for i := 0; i < len(x); i++ {
		parts := strings.Split(names[i], "\\n")
		wordstack(w, x[i]+xdiff, y[i]+ydiff, parts, align, size, fco)
	}
}

// deckpolygon makes deck markup for a polygon given x, y coordinates slices
func deckpolygon(w io.Writer, x, y []float64, color string) {
	nc := len(x)
	if nc < 3 || nc != len(y) {
		return
	}
	fill, op := colorop(color)
	end := nc - 1
	fmt.Fprintf(w, "<polygon color=\"%s\" opacity=\"%s\" xc=\"%.3f", fill, op, x[0])
	for i := 1; i < nc; i++ {
		fmt.Fprintf(w, " %.3f", x[i])
	}
	fmt.Fprintf(w, " %.3f\" ", x[end])
	fmt.Fprintf(w, "yc=\"%.3f", y[0])
	for i := 1; i < nc; i++ {
		fmt.Fprintf(w, " %.3f", y[i])
	}
	fmt.Fprintf(w, " %.3f\"/>\n", y[end])
}

// deckline makes a line in deck markup
func deckline(w io.Writer, x1, y1, x2, y2, lw float64, fill, op string, g Geometry) {
	if x1 >= g.Xmin && x2 <= g.Xmax && y1 >= g.Ymin && y2 <= g.Ymax {
		fmt.Fprintf(w, decklinefmt, x1, y1, x2, y2, lw, fill, op)
	}
}

// Deckpolyline makes deck markup for a ployline given x, y coordinate slices
func deckpolyline(w io.Writer, x, y []float64, lw float64, color string, g Geometry) {
	lx := len(x)
	if lx < 2 {
		return
	}
	op := "100"
	for i := 0; i < lx-1; i++ {
		deckline(w, x[i], y[i], x[i+1], y[i+1], lw, color, op, g)
	}
	deckline(w, x[0], y[0], x[lx-1], y[lx-1], lw, color, op, g)
}

func deckshape(w io.Writer, shape string, x, y []float64, shapesize float64, color string, g Geometry) {
	switch shape {
	case "line", "polyline":
		deckpolyline(w, x, y, shapesize, color, g)
	case "fill", "polygon":
		deckpolygon(w, x, y, color)
	}
}

func geoshape(w io.Writer, data Kml, m Geometry, linewidth float64, color, shape string) {
	// for every placemark, get the coordinates of the polygons
	for _, pms := range data.Document.Placemark {
		px, py := parseCoords(pms.Polygon.OuterBoundaryIs.LinearRing.Coordinates, m) // single polygons
		deckshape(w, shape, px, py, linewidth, color, m)
		mpolys := pms.MultiGeometry.Polygon // multiple polygons
		for _, p := range mpolys {
			mx, my := parseCoords(p.OuterBoundaryIs.LinearRing.Coordinates, m)
			deckshape(w, shape, mx, my, linewidth, color, m)
		}
	}
}
