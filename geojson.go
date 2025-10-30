package decksh

import (
	"encoding/json"
	"io"
	"os"
)

// coordinates as polygon
type GeoJSON struct {
	Type     string     `json:"type"`
	Features []Features `json:"features"`
}

type Features struct {
	Type      string    `json:"type"`
	JGeometry JGeometry `json:"geometry"`
}

type JGeometry struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// coordinates as multipolygon
type MGeoJSON struct {
	Type     string      `json:"type"`
	Features []MFeatures `json:"features"`
}

type MFeatures struct {
	Type     string    `json:"type"`
	Geometry MGeometry `json:"geometry"`
}

type MGeometry struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

// Coordinates returns polygon lat long coordinates
func coordsJSON(dest io.Writer, data GeoJSON, g Geometry, color string, shapesize float64) {
	for _, f := range data.Features {
		if f.JGeometry.Type == "Polygon" {
			coords := f.JGeometry.Coordinates
			x := []float64{}
			y := []float64{}
			for _, c := range coords[0] {
				x = append(x, vmap(c[0], g.Longmin, g.Longmax, g.Xmin, g.Xmax))
				y = append(y, vmap(c[1], g.Latmin, g.Latmax, g.Ymin, g.Ymax))
			}
			if shapesize > 0 {
				shpdeckpolyline(dest, x, y, color, shapesize)
			} else {
				deckpolygon(dest, x, y, color)
			}
		}
	}
}

// readJSON opens a geoJSON file, parses its contents, and writes coordinates in deck format
func readJSON(dest io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	r, err := os.Open(filename)
	if err != nil {
		return err
	}
	data, err := parseJSON(r)
	if err != nil {
		return err
	}
	coordsJSON(dest, data, mapgeo, color, shapesize)
	return r.Close()
}

// parseJSON parses geoJSON
func parseJSON(r io.Reader) (GeoJSON, error) {
	var data GeoJSON
	err := json.NewDecoder(r).Decode(&data)
	return data, err
}
