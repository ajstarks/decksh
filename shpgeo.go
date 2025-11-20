// Package decksh is a little language that generates deck markup
// Shapefile processing
package decksh

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// Shapefile constants
const (
	ShapeFileCode     = 9994
	ShapeTypeNull     = 0
	ShapeTypePoint    = 1
	ShapeTypePolyLine = 3
	ShapeTypePolygon  = 5
)

// shpBox is the coordinates bounding box
type shpBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// shpPoint is a single coordinate
type shpPoint struct {
	X, Y float64
}

// shpPolyline is shape polygon
type shpPolyLine struct {
	shpBox
	NumParts  int32
	NumPoints int32
	Parts     []int32
	Points    []shpPoint
}

// readSHP opens and parses a Shapefile, generating deck markup
func readSHP(w io.Writer, filename string, mapgeo Geometry, color string, shapesize float64) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read file header (100 bytes)
	header := make([]byte, 100)
	if _, err := io.ReadFull(file, header); err != nil {
		return err
	}
	// Verify file code (should be 9994)
	fileCode := binary.BigEndian.Uint32(header[0:4])
	if fileCode != ShapeFileCode {
		return fmt.Errorf("invalid shapefile: wrong file code %d", fileCode)
	}
	// Read records
	for {
		// Read record header (8 bytes)
		recordHeader := make([]byte, 8)
		nr, err := io.ReadFull(file, recordHeader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if nr != 8 {
			return fmt.Errorf("invalid shapefile record size (%d)", nr)
		}
		// Content length in 16-bit words (big endian)
		contentLength := binary.BigEndian.Uint32(recordHeader[4:8]) * 2

		// Read record content
		recordContent := make([]byte, contentLength)
		if _, err := io.ReadFull(file, recordContent); err != nil {
			return err
		}

		// Parse shape type (little endian)
		shapeType := binary.LittleEndian.Uint32(recordContent[0:4])
		parseShape(shapeType, recordContent[4:], w, mapgeo, color, shapesize)
	}
	return nil
}

// ParseShape retrieves coordinate data from shape records,
// producing deck markup
func parseShape(shapeType uint32, data []byte, w io.Writer, g Geometry, color string, shapesize float64) {
	switch shapeType {
	case ShapeTypePoint:
		shpdeckpoint(w, g, color, shapesize, parsePoint(data))

	case ShapeTypePolyLine, ShapeTypePolygon:
		shpdeckpoly(w, g, color, shapesize, parsePolyLine(data))

	default:
		fmt.Fprintf(os.Stderr, "unsupported shape type (%d)\n", shapeType)
	}
}

// readFloat64 turns raw data into floating point
func readFloat64(b []byte) float64 {
	bits := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(bits)
}

// parsePoint return coordinates from the shape record
func parsePoint(data []byte) shpPoint {
	var p shpPoint
	if len(data) < 16 {
		return p
	}
	p.X = readFloat64(data[0:8])
	p.Y = readFloat64(data[8:16])
	return p
}

// parsePolyLine returns the poly[gon|line] struct from the shape record
func parsePolyLine(data []byte) shpPolyLine {
	p := shpPolyLine{}
	if len(data) < 40 {
		return p
	}
	file := bytes.NewReader(data)
	binary.Read(file, binary.LittleEndian, &p.shpBox)
	binary.Read(file, binary.LittleEndian, &p.NumParts)
	binary.Read(file, binary.LittleEndian, &p.NumPoints)
	p.Parts = make([]int32, p.NumParts)
	p.Points = make([]shpPoint, p.NumPoints)
	binary.Read(file, binary.LittleEndian, &p.Parts)
	binary.Read(file, binary.LittleEndian, &p.Points)
	return p
}

// shpdeckpoly makes deck markup from poly data
func shpdeckpoly(w io.Writer, g Geometry, color string, shapesize float64, poly shpPolyLine) {
	last := poly.NumParts - 1
	for i := range last {
		// index into each part, reading coordinates, and map to geographic and canvas geometries
		x := []float64{}
		y := []float64{}
		for j := poly.Parts[i]; j < poly.Parts[i+1]; j++ {
			x = append(x, vmap(poly.Points[j].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
			y = append(y, vmap(poly.Points[j].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
		}
		geopl(w, x, y, color, shapesize)

	}
	// process the last part
	x := []float64{}
	y := []float64{}
	for k := poly.Parts[last]; k < poly.NumPoints; k++ {
		x = append(x, vmap(poly.Points[k].X, g.Longmin, g.Longmax, g.Xmin, g.Xmax))
		y = append(y, vmap(poly.Points[k].Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax))
	}
	geopl(w, x, y, color, shapesize)
}

// shpdeckpolyline makes a series of lines in deck markup from a set of (x,y) coordinates
func shpdeckpolyline(w io.Writer, x, y []float64, color string, size float64) {
	fill, op := colorop(color)
	lx := len(x)
	for i := 0; i < lx-1; i++ {
		fmt.Fprintf(w, decklinefmt, x[i], y[i], x[i+1], y[i+1], size, fill, op)
	}
	fmt.Fprintf(w, decklinefmt, x[0], y[0], x[lx-1], y[lx-1], size, fill, op)
}

// shpdeckpoint places a circle at a coordinate.
// the coordinates are mapped from geographical coordinates to screen bounding box.
func shpdeckpoint(dest io.Writer, g Geometry, color string, shapesize float64, p shpPoint) {
	x := vmap(p.X, g.Longmin, g.Longmax, g.Xmin, g.Xmax)
	y := vmap(p.Y, g.Latmin, g.Latmax, g.Ymin, g.Ymax)
	fill, op := colorop(color)
	fmt.Fprintf(dest, shpdotfmt, x, y, fill, op, shapesize)
}
