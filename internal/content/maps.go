package content

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/uinjad/AzureNights2/internal/domain/world"
)

type tileDTO struct {
	Kind     string `json:"kind"`
	Emoji    string `json:"emoji"`
	Walkable bool   `json:"walkable"`
}

type pointDTO struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type mapDTO struct {
	Name    string              `json:"name"`
	Spawn   pointDTO            `json:"spawn"`
	Legend  map[string]tileDTO  `json:"legend"`
	Rows    []string            `json:"rows"`
	Enemies []enemyPlacementDTO `json:"enemies"`
}

type enemyPlacementDTO struct {
	X  int    `json:"x"`
	Y  int    `json:"y"`
	ID string `json:"id"`
}

func loadMaps(enemies map[string]EnemyDef) (map[string]MapDef, error) {
	entries, err := fs.ReadDir(dataFS, "data/maps")
	if err != nil {
		return nil, fmt.Errorf("content: listing maps: %w", err)
	}
	out := make(map[string]MapDef)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := dataFS.ReadFile("data/maps/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("content: reading map %s: %w", e.Name(), err)
		}
		def, err := parseMap(b)
		if err != nil {
			return nil, fmt.Errorf("content: map %s: %w", e.Name(), err)
		}
		for _, p := range def.Enemies {
			if _, ok := enemies[p.DefID]; !ok {
				return nil, fmt.Errorf("content: map %s places unknown enemy %q", e.Name(), p.DefID)
			}
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = def
	}
	return out, nil
}

// parseMap expands a map file's legend over its ASCII rows. Kept pure (bytes in,
// value out) so it can be tested directly with inline fixtures, including
// malformed ones.
func parseMap(b []byte) (MapDef, error) {
	dto, err := unmarshal[mapDTO](b)
	if err != nil {
		return MapDef{}, err
	}
	if len(dto.Rows) == 0 {
		return MapDef{}, fmt.Errorf("map has no rows")
	}
	h := len(dto.Rows)
	w := len([]rune(dto.Rows[0]))
	tiles := make([]world.Tile, 0, w*h)
	for y, row := range dto.Rows {
		runes := []rune(row)
		if len(runes) != w {
			return MapDef{}, fmt.Errorf("row %d has width %d, want %d", y, len(runes), w)
		}
		for _, r := range runes {
			td, ok := dto.Legend[string(r)]
			if !ok {
				return MapDef{}, fmt.Errorf("no legend entry for %q", string(r))
			}
			kind, err := parseTileKind(td.Kind)
			if err != nil {
				return MapDef{}, err
			}
			tiles = append(tiles, world.Tile{Kind: kind, Emoji: td.Emoji, Walkable: td.Walkable})
		}
	}
	tm, err := world.NewTileMap(w, h, tiles)
	if err != nil {
		return MapDef{}, err
	}
	placements := make([]EnemyPlacement, 0, len(dto.Enemies))
	for _, e := range dto.Enemies {
		placements = append(placements, EnemyPlacement{Pos: world.Point{X: e.X, Y: e.Y}, DefID: e.ID})
	}
	return MapDef{Name: dto.Name, Map: tm, Spawn: world.Point{X: dto.Spawn.X, Y: dto.Spawn.Y}, Enemies: placements}, nil
}
