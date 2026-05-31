package world

import (
	"errors"
	"testing"
)

func tinyMap(t *testing.T) *TileMap {
	t.Helper()
	g := Tile{Kind: Grass, Emoji: "🌿", Walkable: true}
	w := Tile{Kind: Water, Emoji: "🌊", Walkable: false}
	// 3x2:  g g w
	//       g w g
	m, err := NewTileMap(3, 2, []Tile{g, g, w, g, w, g})
	if err != nil {
		t.Fatalf("NewTileMap: %v", err)
	}
	return m
}

func TestNewTileMapRejectsBadDimensions(t *testing.T) {
	if _, err := NewTileMap(3, 2, []Tile{{}, {}, {}}); !errors.Is(err, ErrBadDimensions) {
		t.Fatalf("want ErrBadDimensions, got %v", err)
	}
}

func TestWalkableAndBounds(t *testing.T) {
	m := tinyMap(t)
	if !m.Walkable(Point{0, 0}) {
		t.Error("(0,0) grass should be walkable")
	}
	if m.Walkable(Point{2, 0}) {
		t.Error("(2,0) water should block")
	}
	if m.Walkable(Point{5, 5}) {
		t.Error("out of bounds must not be walkable")
	}
}

func TestStep(t *testing.T) {
	m := tinyMap(t)

	if got, ok := m.Step(Point{0, 0}, East); !ok || got != (Point{1, 0}) {
		t.Errorf("east onto grass: got %v ok=%v", got, ok)
	}
	if got, ok := m.Step(Point{1, 0}, East); ok || got != (Point{1, 0}) {
		t.Errorf("east into water should stay put: got %v ok=%v", got, ok)
	}
	if _, ok := m.Step(Point{0, 0}, North); ok {
		t.Error("stepping off the map must fail")
	}
}
