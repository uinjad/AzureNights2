// Package app is the application layer: it orchestrates the domain packages into
// playable use-cases and defines the ports (interfaces) through which outside
// adapters — storage today, an HTTP server tomorrow — plug in. It depends inward
// on content and the domain; nothing in the domain depends on it.
package app

import (
	"errors"

	"github.com/uinjad/AzureNights2/internal/domain/character"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

// Repository is the persistence port. The storage adapter implements it; the app
// knows only this interface, so swapping JSON files for a database is a drop-in.
type Repository interface {
	Save(snap *Snapshot) error
	Load() (*Snapshot, error)
	Exists() bool
}

// Snapshot is the serializable slice of game state worth persisting. The live
// Session also holds the content registry, the active battle, and the log — none
// of which belong in a save file.
type Snapshot struct {
	Hero      *character.Character `json:"hero"`
	MapID     string               `json:"map_id"`
	PlayerPos world.Point          `json:"player_pos"`
	Clock     world.Clock          `json:"clock"`
	Spawns    []Spawn              `json:"spawns"`
}

// Spawn is a living enemy still standing on the map.
type Spawn struct {
	Pos   world.Point `json:"pos"`
	DefID string      `json:"def_id"`
}

var (
	ErrBusy        = errors.New("app: not allowed during battle")
	ErrNotInBattle = errors.New("app: no battle in progress")
	ErrInvalidItem = errors.New("app: invalid inventory item")
)
