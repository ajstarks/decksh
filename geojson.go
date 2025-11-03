package decksh

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// readJSON opens a geoJSON file, parses its contents, and writes coordinates in deck format
func readJSON(dest io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	// Parse and Print, continue on error
	err = parseData(dest, data, mapgeo, color, shapesize)
	if err != nil {
		return err
	}
	return nil
}

// parseData parses raw JSON data
// for every feature found in the collection, print coordinates according to type:
// (MultPolygon, Polygon, MultiLineString, LineString, MultiPoint and Point)
func parseData(w io.Writer, data []byte, g Geometry, color string, shapesize float64) error {
	fc := geojson.NewFeatureCollection()
	err := json.Unmarshal(data, &fc)
	if err != nil {
		return err
	}

	lf := len(fc.Features)

	if lf < 1 {
		return fmt.Errorf("empty collection")
	}

	for i := range lf { // for every feature...
		switch fc.Features[i].Geometry.GeoJSONType() { // process according to type

		case "MultiPolygon":
			for _, poly := range fc.Features[i].Geometry.(orb.MultiPolygon) {
				for _, c := range poly {
					x := []float64{}
					y := []float64{}
					for _, ll := range c {
						x = append(x, vmap(ll.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax))
						y = append(y, vmap(ll.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax))
					}
					if shapesize > 0 {
						shpdeckpolyline(w, x, y, color, shapesize)
					} else {
						deckpolygon(w, x, y, color)
					}
				}
			}

		case "Polygon":
			for _, c := range fc.Features[i].Geometry.(orb.Polygon) {
				x := []float64{}
				y := []float64{}
				for _, ll := range c {
					x = append(x, vmap(ll.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax))
					y = append(y, vmap(ll.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax))
				}
				if shapesize > 0 {
					shpdeckpolyline(w, x, y, color, shapesize)
				} else {
					deckpolygon(w, x, y, color)
				}
			}

		case "MultiLineString":
			for _, ml := range fc.Features[i].Geometry.(orb.MultiLineString) {
				x := []float64{}
				y := []float64{}
				for _, ll := range ml {
					x = append(x, vmap(ll.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax))
					y = append(y, vmap(ll.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax))
				}
				shpdeckpolyline(w, x, y, color, shapesize)
			}

		case "LineString":
			x := []float64{}
			y := []float64{}
			for _, ll := range fc.Features[i].Geometry.(orb.LineString) {
				x = append(x, vmap(ll.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax))
				y = append(y, vmap(ll.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax))
			}
			shpdeckpolyline(w, x, y, color, shapesize)

		case "MultiPoint":
			x := []float64{}
			y := []float64{}
			for _, p := range fc.Features[i].Geometry.(orb.MultiPoint) {
				x = append(x, vmap(p.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax))
				y = append(y, vmap(p.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax))
			}
			shpdeckdot(w, x, y, color, shapesize)

		case "Point":
			p := fc.Features[i].Geometry.(orb.Point)
			fill, op := colorop(color)
			xp := vmap(p.Lon(), g.Longmin, g.Longmax, g.Xmin, g.Xmax)
			yp := vmap(p.Lat(), g.Latmin, g.Latmax, g.Ymin, g.Ymax)
			fmt.Fprintf(w, shpdotfmt, xp, yp, fill, op, shapesize)
		}
	}
	return nil
}
