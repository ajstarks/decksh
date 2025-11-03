// Package decksh is a little language that generates deck markup
// KML processing
package decksh

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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
func parseKMLCoords(s string, g Geometry) ([]float64, []float64) {
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

// deckpolygon makes deck markup for a polygon given x, y coordinates slices
func deckpolygon(w io.Writer, x, y []float64, color string) {
	nc := len(x)
	if nc < 3 || nc != len(y) {
		return
	}
	fill, op := colorop(color)
	end := nc - 1
	fmt.Fprintf(w, "<polygon color=%s opacity=%q xc=\"%.5f", fill, op, x[0])
	for i := 1; i < nc; i++ {
		fmt.Fprintf(w, " %.5f", x[i])
	}
	fmt.Fprintf(w, " %.5f\" ", x[end])
	fmt.Fprintf(w, "yc=\"%.5f", y[0])
	for i := 1; i < nc; i++ {
		fmt.Fprintf(w, " %.5f", y[i])
	}
	fmt.Fprintf(w, " %.5f\"/>\n", y[end])
}

// deckline makes a line in deck markup
func deckline(w io.Writer, x1, y1, x2, y2, lw float64, fill, op string, g Geometry) {
	if x1 >= g.Xmin && x2 <= g.Xmax && y1 >= g.Ymin && y2 <= g.Ymax {
		fmt.Fprintf(w, decklinefmt, x1, y1, x2, y2, lw, fill, op)
	}
}

// Deckpolyline makes deck markup for a ployline given x, y coordinate slices,
// cycling back to the beginning
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

// deckpolyline makes deck markup for a ployline given x, y coordinate slices
func deckconnectline(w io.Writer, x, y []float64, lw float64, color string, op string, g Geometry) {
	lx := len(x)
	if lx < 2 {
		return
	}
	for i := 0; i < lx-1; i++ {
		deckline(w, x[i], y[i], x[i+1], y[i+1], lw, color, op, g)
	}
}

// deckshape draws the shapes (lines, polylines, polygons) in deck markup
func deckshape(w io.Writer, shape string, x, y []float64, shapesize float64, color string, g Geometry) {
	switch shape {
	case "line", "polyline":
		deckpolyline(w, x, y, shapesize, color, g)
	case "fill", "polygon":
		deckpolygon(w, x, y, color)
	}
}

// geoshape produces deck markup from KML data
func geoshape(w io.Writer, data Kml, m Geometry, linewidth float64, color, shape string) {
	// for every placemark, get the coordinates of the polygons
	for _, pms := range data.Document.Placemark {
		px, py := parseKMLCoords(pms.Polygon.OuterBoundaryIs.LinearRing.Coordinates, m) // single polygons
		deckshape(w, shape, px, py, linewidth, color, m)
		mpolys := pms.MultiGeometry.Polygon // multiple polygons
		for _, p := range mpolys {
			mx, my := parseKMLCoords(p.OuterBoundaryIs.LinearRing.Coordinates, m)
			deckshape(w, shape, mx, my, linewidth, color, m)
		}
	}
}

// ReadKML reads and parses KML files, writing deck markup
func readKML(w io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	var data Kml
	r, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer r.Close()
	err = xml.NewDecoder(r).Decode(&data)
	if err != nil {
		return err
	}
	var shape string
	if shapesize > 0 {
		shape = "line"
	} else {
		shape = "polygon"
	}
	geoshape(w, data, mapgeo, shapesize, color, shape)
	return nil
}
