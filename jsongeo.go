// Package decksh is a little language that generates deck markup
// geoJSON processing
package decksh

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// GeoJSON structure definitions
type GeoJSON struct {
	Type      string     `json:"type"`
	Features  []Feature  `json:"features,omitempty"`
	JGeometry *JGeometry `json:"geometry,omitempty"`
}

type Feature struct {
	Type       string         `json:"type"`
	JGeometry  JGeometry      `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

type JGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// readJSON opens a geoJSON file, parses its contents, and writes coordinates in deck format
func readJSON(dest io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	// Parse and Print, continue on error
	err = parseJSON(dest, data, mapgeo, color, shapesize)
	if err != nil {
		return err
	}
	return nil
}

// parseJSONData parses raw JSON data
func parseJSON(w io.Writer, data []byte, g Geometry, color string, shapesize float64) error {
	var geoJSON GeoJSON
	err := json.Unmarshal(data, &geoJSON)
	if err != nil {
		return err
	}
	// for every feature found in the collection, print coordinates according to type:
	// (MultPolygon, Polygon, MultiLineString, LineString, MultiPoint and Point)
	for _, feature := range geoJSON.Features {
		JSONCoords(w, g, color, shapesize, feature.JGeometry)
	}
	return nil
}

// JSONCoordinates extracts coordinates from a JGeometry object, according to type
func JSONCoords(w io.Writer, g Geometry, color string, shapesize float64, jg JGeometry) {
	switch jg.Type {
	case "Point":
		JSONPoint(w, g, color, shapesize, jg.Coordinates)
	case "LineString", "MultiPoint":
		JSONLineString(w, g, color, shapesize, jg.Coordinates)
	case "Polygon", "MultiLineString":
		JSONPolygon(w, g, color, shapesize, jg.Coordinates)
	case "MultiPolygon":
		JSONMultiPolygon(w, g, color, shapesize, jg.Coordinates)
	}
}

// JSONPoint parses a single coordinate pair [longitude, latitude]
func JSONPoint(w io.Writer, g Geometry, color string, shapesize float64, raw json.RawMessage) {
	var point []float64
	fill, op := colorop(color)
	json.Unmarshal(raw, &point)

	xp := vmap(point[0], g.Longmin, g.Longmax, g.Xmin, g.Xmax)
	yp := vmap(point[1], g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	fmt.Fprintf(w, shpdotfmt, xp, yp, fill, op, shapesize)
}

// JSONLineString parses an array of coordinate pairs
func JSONLineString(w io.Writer, g Geometry, color string, shapesize float64, raw json.RawMessage) {
	var line [][]float64
	json.Unmarshal(raw, &line)

	x := []float64{}
	y := []float64{}
	for _, ll := range line {
		x = append(x, vmap(ll[0], g.Longmin, g.Longmax, g.Xmin, g.Xmax))
		y = append(y, vmap(ll[1], g.Latmin, g.Latmax, g.Ymin, g.Ymax))
	}
	if shapesize > 0 {
		shpdeckpolyline(w, x, y, color, shapesize)
	} else {
		deckpolygon(w, x, y, color)
	}
}

// JSONPolygon parses an array of linear rings (array of coordinate pairs)
func JSONPolygon(w io.Writer, g Geometry, color string, shapesize float64, raw json.RawMessage) {
	var polygon [][][]float64
	json.Unmarshal(raw, &polygon)

	for _, ring := range polygon {
		x := []float64{}
		y := []float64{}
		for _, ll := range ring {
			x = append(x, vmap(ll[0], g.Longmin, g.Longmax, g.Xmin, g.Xmax))
			y = append(y, vmap(ll[1], g.Latmin, g.Latmax, g.Ymin, g.Ymax))
		}
		if shapesize > 0 {
			shpdeckpolyline(w, x, y, color, shapesize)
		} else {
			deckpolygon(w, x, y, color)
		}
	}
}

// JSONMultiPolygon parses an array of polygons
func JSONMultiPolygon(w io.Writer, g Geometry, color string, shapesize float64, raw json.RawMessage) {
	var multiPoly [][][][]float64
	json.Unmarshal(raw, &multiPoly)

	for _, polygon := range multiPoly {
		for _, ring := range polygon {
			x := []float64{}
			y := []float64{}
			for _, ll := range ring {
				x = append(x, vmap(ll[0], g.Longmin, g.Longmax, g.Xmin, g.Xmax))
				y = append(y, vmap(ll[1], g.Latmin, g.Latmax, g.Ymin, g.Ymax))
			}
			if shapesize > 0 {
				shpdeckpolyline(w, x, y, color, shapesize)
			} else {
				deckpolygon(w, x, y, color)
			}
		}
	}
}
