// Package content turns human-editable JSON into validated domain objects. All
// data lives under content/data and is embedded with go:embed, so the game ships
// as a single self-contained binary. The JSON uses readable string enums
// ("magical", "weapon", "grass"); this package translates them into the domain's
// integer enums, keeping the wire format friendly and the domain model tight.
package content

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/faction"
	"github.com/uinjad/AzureNights2/internal/domain/item"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

func Load() (*Registry, error) {
	factions, err := loadFactions()
	if err != nil {
		return nil, err
	}
	skills, err := loadSkills()
	if err != nil {
		return nil, err
	}
	classes, err := loadClasses(skills)
	if err != nil {
		return nil, err
	}
	items, err := loadItems()
	if err != nil {
		return nil, err
	}
	enemies, err := loadEnemies()
	if err != nil {
		return nil, err
	}
	maps, err := loadMaps(enemies)
	if err != nil {
		return nil, err
	}
	return &Registry{
		Factions: factions, Classes: classes, Skills: skills,
		Items: items, Enemies: enemies, Maps: maps,
	}, nil
}

//go:embed data
var dataFS embed.FS

// EnemyPlacement marks where an enemy stands on a map.
type EnemyPlacement struct {
	Pos   world.Point
	DefID string
}

// MapDef now also carries enemy placements:
type MapDef struct {
	Name    string
	Map     *world.TileMap
	Spawn   world.Point
	Enemies []EnemyPlacement
}

// Registry is the loaded, validated game content, ready for the app layer.
type Registry struct {
	Factions *faction.Table
	Classes  *class.Tree
	Skills   map[string]combat.Skill
	Items    map[string]item.Item
	Enemies  map[string]EnemyDef
	Maps     map[string]MapDef
}

// EnemyDef is a template a battle turns into a combat.Combatant.
type EnemyDef struct {
	ID         string
	Faction    faction.ID
	Name       string
	Emoji      string
	Stats      stats.Derived
	XPReward   int
	GoldReward int
}

func readJSON[T any](name string) (T, error) {
	b, err := dataFS.ReadFile(name)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("content: reading %s: %w", name, err)
	}
	return unmarshal[T](b)
}

func unmarshal[T any](b []byte) (T, error) {
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("content: parsing data: %w", err)
	}
	return v, nil
}

// --- shared value DTOs ---

type primaryDTO struct {
	STR int `json:"str"`
	DEX int `json:"dex"`
	CON int `json:"con"`
	INT int `json:"int"`
	WIT int `json:"wit"`
	MEN int `json:"men"`
}

func (d primaryDTO) toDomain() stats.Primary {
	return stats.Primary{STR: d.STR, DEX: d.DEX, CON: d.CON, INT: d.INT, WIT: d.WIT, MEN: d.MEN}
}

type derivedDTO struct {
	MaxHP int `json:"max_hp"`
	MaxMP int `json:"max_mp"`
	PAtk  int `json:"p_atk"`
	MAtk  int `json:"m_atk"`
	PDef  int `json:"p_def"`
	MDef  int `json:"m_def"`
	Init  int `json:"init"`
	Crit  int `json:"crit"`
}

func (d derivedDTO) toDomain() stats.Derived {
	return stats.Derived{
		MaxHP: d.MaxHP, MaxMP: d.MaxMP, PAtk: d.PAtk, MAtk: d.MAtk,
		PDef: d.PDef, MDef: d.MDef, Init: d.Init, Crit: d.Crit,
	}
}

// --- string enum parsers ---

func parseDamageKind(s string) (combat.DamageKind, error) {
	switch s {
	case "physical":
		return combat.Physical, nil
	case "magical":
		return combat.Magical, nil
	default:
		return 0, fmt.Errorf("content: unknown damage kind %q", s)
	}
}

func parseSlot(s string) (item.Slot, error) {
	switch s {
	case "weapon":
		return item.Weapon, nil
	case "armor":
		return item.Armor, nil
	default:
		return 0, fmt.Errorf("content: unknown equipment slot %q", s)
	}
}

func parseTileKind(s string) (world.TileKind, error) {
	switch s {
	case "grass":
		return world.Grass, nil
	case "forest":
		return world.Forest, nil
	case "water":
		return world.Water, nil
	case "mountain":
		return world.Mountain, nil
	case "floor":
		return world.Floor, nil
	case "wall":
		return world.Wall, nil
	default:
		return 0, fmt.Errorf("content: unknown tile kind %q", s)
	}
}
