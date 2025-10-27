package decksh

import (
	"fmt"
	"io"

	"github.com/jonas-p/go-shp"
)

const (
	shpdotfmt = "<ellipse xp=\"%.7f\" yp=\"%.7f\" hr=\"100\" color=%q opacity=%q wp=\"%.3f\"/>\n"
)

func readSHP(dest io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	shape, err := shp.Open(filename)
	if err != nil {
		return err
	}
	// for each feature...
	for shape.Next() {
		_, s := shape.Shape()
		// process each type of shape
		if p, ok := s.(*shp.Polygon); ok {
			if shapesize > 0 {
				shpPolygonCoords(dest, "line", p, mapgeo, color, shapesize)
			} else {
				shpPolygonCoords(dest, "polygon", p, mapgeo, color, shapesize)
			}
		}
		if p, ok := s.(*shp.PolyLine); ok {
			shpPolylineCoords(dest, "line", p, mapgeo, color, shapesize)
		}
		if p, ok := s.(*shp.MultiPoint); ok {
			shpMultipointCoords(dest, p, mapgeo, color, shapesize)
		}
		if p, ok := s.(*shp.Point); ok {
			shpPointCoords(dest, p, mapgeo, color, shapesize)
		}
	}
	shape.Close()
	return nil
}

// deckpolyline makes a series of lines in deck markup from a set of (x,y) coordinates
func shpdeckpolyline(w io.Writer, x, y []float64, color string, size float64) {
	fill, op := colorop(color)
	lx := len(x)
	for i := 0; i < lx-1; i++ {
		fmt.Fprintf(w, decklinefmt, x[i], y[i], x[i+1], y[i+1], size, fill, op)
	}
	fmt.Fprintf(w, decklinefmt, x[0], y[0], x[lx-1], y[lx-1], size, fill, op)
}

// deckdot makes a series of circles in deck markup from a set of (x,y) coordinates
func shpdeckdot(w io.Writer, x, y []float64, color string, size float64) {
	fill, op := colorop(color)
	for i := range len(x) {
		fmt.Fprintf(w, shpdotfmt, x[i], y[i], fill, op, size)
	}
}

// mapshape writes markup to the destination according to the specified shape
func mapshape(w io.Writer, x, y []float64, shape string, color string, size float64) {
	switch shape {
	case "p", "poly", "region", "polygon":
		deckpolygon(w, x, y, color)
	case "l", "line", "border":
		shpdeckpolyline(w, x, y, color, size)
	case "d", "dot", "circle":
		shpdeckdot(w, x, y, color, size)
	}
}

// polygonCoords converts a set of coordinates and makes polygons
// the polygons are mapped from geographical coordinates to screen bounding box
// the coordinates are processed in the order specified by a vector that contains
// the coordinate indicies.
func shpPolygonCoords(dest io.Writer, maptype string, poly *shp.Polygon, g Geometry, color string, shapesize float64) {
	// for every part...
	last := poly.NumParts - 1
	for i := range last {
		// index into each part, reading coordinates, and map to map geometries
		x := []float64{}
		y := []float64{}
		for j := poly.Parts[i]; j < poly.Parts[i+1]; j++ {
			x = append(x, vmap(poly.Points[j].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
			y = append(y, vmap(poly.Points[j].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
		}
		mapshape(dest, x, y, maptype, color, shapesize)
	}
	// process the last part
	x := []float64{}
	y := []float64{}
	for k := poly.Parts[last]; k < poly.NumPoints; k++ {
		x = append(x, vmap(poly.Points[k].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
		y = append(y, vmap(poly.Points[k].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
	}
	mapshape(dest, x, y, maptype, color, shapesize)
}

// polygonCoords converts a set of coordinates and makes polylines
// the polylines are mapped from geographical coordinates to screen bounding box
// the coordinates are processed in the order specified by a vector that contains
// the coordinate indicies.
func shpPolylineCoords(dest io.Writer, maptype string, poly *shp.PolyLine, g Geometry, color string, shapesize float64) {
	// for every part...
	last := poly.NumParts - 1
	for i := range last {
		// index into each part, reading coordinates, and map to map geometries
		x := []float64{}
		y := []float64{}
		for j := poly.Parts[i]; j < poly.Parts[i+1]; j++ {
			x = append(x, vmap(poly.Points[j].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
			y = append(y, vmap(poly.Points[j].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
		}
		mapshape(dest, x, y, maptype, color, shapesize)
	}
	// process the last part
	x := []float64{}
	y := []float64{}
	for k := poly.Parts[last]; k < poly.NumPoints; k++ {
		x = append(x, vmap(poly.Points[k].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
		y = append(y, vmap(poly.Points[k].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
	}
	mapshape(dest, x, y, maptype, color, shapesize)
}

// multipointCoords converts a set of coordinates and makes circles for each coordinate.
// the coordinates are mapped from geographical coordinates to screen bounding box
func shpMultipointCoords(dest io.Writer, mp *shp.MultiPoint, g Geometry, color string, shapesize float64) {
	x := []float64{}
	y := []float64{}
	for i := int32(0); i < mp.NumPoints; i++ {
		x = append(x, vmap(mp.Points[i].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
		y = append(y, vmap(mp.Points[i].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
	}
	mapshape(dest, x, y, "dot", color, shapesize)
}

// pointCoords places a circle at a coordinate.
// the coordinates are mapped from geographical coordinates to screen bounding box.
func shpPointCoords(dest io.Writer, p *shp.Point, g Geometry, color string, shapesize float64) {
	x := vmap(p.X, g.Longmin, g.Longmax, g.Xmin, g.Xmax)
	y := vmap(p.Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	fill, op := colorop(color)
	fmt.Fprintf(dest, shpdotfmt, x, y, fill, op, shapesize)
}
