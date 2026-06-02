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

// Snapshot is the serializable slice of game state worth persisting.
type Snapshot struct {
	Hero      *character.Character `json:"hero"`
	MapID     string               `json:"map_id"`
	PlayerPos world.Point          `json:"player_pos"`
	Clock     world.Clock          `json:"clock"`
	Spawns    []Spawn              `json:"spawns"`
	Pending   []PendingRespawn     `json:"pending"`
	Quests    []QuestProgress      `json:"quests"`
	Cleared   map[string]bool      `json:"cleared"`
	Won       bool                 `json:"won"`
}

// Spawn is a living enemy still standing on the map.
type Spawn struct {
	Pos   world.Point `json:"pos"`
	DefID string      `json:"def_id"`
}

// PendingRespawn is a defeated enemy queued to return at AtTick.
type PendingRespawn struct {
	Pos    world.Point `json:"pos"`
	DefID  string      `json:"def_id"`
	AtTick int         `json:"at_tick"`
}

// QuestProgress is per-quest objective counters plus completion.
type QuestProgress struct {
	ID     string `json:"id"`
	Counts []int  `json:"counts"`
	Done   bool   `json:"done"`
}

var (
	ErrBusy        = errors.New("app: not allowed during battle")
	ErrNotInBattle = errors.New("app: no battle in progress")
	ErrInvalidItem = errors.New("app: invalid item")
)
