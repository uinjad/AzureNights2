// Package world holds the exploration layer: the tile grid the hero walks on
// and the clock that drives the living world (time of day and weather). Like
// the rest of the domain it is pure — movement is validated arithmetic and the
// clock is a deterministic state machine fed by an injected source of randomness.
package world

import (
	"errors"
	"fmt"
)

// Point is a grid coordinate; X grows right, Y grows down (screen-friendly).
type Point struct{ X, Y int }

// Direction is a cardinal move.
type Direction int

const (
	North Direction = iota
	South
	East
	West
)

func (d Direction) delta() Point {
	switch d {
	case North:
		return Point{0, -1}
	case South:
		return Point{0, 1}
	case East:
		return Point{1, 0}
	case West:
		return Point{-1, 0}
	default:
		return Point{}
	}
}

// TileKind classifies terrain.
type TileKind int

const (
	Grass TileKind = iota
	Forest
	Water
	Mountain
	Floor
	Wall
)

// Tile is one cell of the map. Emoji is how it renders; Walkable gates movement.
type Tile struct {
	Kind     TileKind
	Emoji    string
	Walkable bool
}

// ErrBadDimensions is returned when a map's tile count doesn't match W*H.
var ErrBadDimensions = errors.New("world: tile count does not match width*height")

// TileMap is a rectangular grid stored in row-major order.
type TileMap struct {
	W, H  int
	tiles []Tile
}

// NewTileMap builds a map, validating that exactly W*H tiles were provided so a
// malformed map file fails at load instead of panicking mid-game.
func NewTileMap(w, h int, tiles []Tile) (*TileMap, error) {
	if w <= 0 || h <= 0 || len(tiles) != w*h {
		return nil, fmt.Errorf("%w: %dx%d needs %d, got %d", ErrBadDimensions, w, h, w*h, len(tiles))
	}
	return &TileMap{W: w, H: h, tiles: tiles}, nil
}

// InBounds reports whether a point lies on the map.
func (m *TileMap) InBounds(p Point) bool {
	return p.X >= 0 && p.X < m.W && p.Y >= 0 && p.Y < m.H
}

// At returns the tile at a point; ok is false when out of bounds.
func (m *TileMap) At(p Point) (Tile, bool) {
	if !m.InBounds(p) {
		return Tile{}, false
	}
	return m.tiles[p.Y*m.W+p.X], true
}

// Walkable reports whether a point is on the map and steppable.
func (m *TileMap) Walkable(p Point) bool {
	t, ok := m.At(p)
	return ok && t.Walkable
}

// Step computes the position after moving one tile. On an illegal move (off the
// map or into a blocked tile) it returns the original position and false.
func (m *TileMap) Step(from Point, dir Direction) (Point, bool) {
	d := dir.delta()
	next := Point{from.X + d.X, from.Y + d.Y}
	if !m.Walkable(next) {
		return from, false
	}
	return next, true
}
