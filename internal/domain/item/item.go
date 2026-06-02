// Package item defines gear and consumables the hero can carry. Gear occupies a
// slot and grants stat bonuses while worn; potions are consumed for a one-shot
// restore. The Kind field keeps the two apart without separate collections.
package item

import "github.com/uinjad/AzureNights2/internal/domain/stats"

type Slot int

const (
	Weapon Slot = iota
	Armor
)

type Kind int

const (
	Gear Kind = iota
	Potion
)

// Item is a piece of carried equipment or a consumable.
type Item struct {
	ID    string
	Name  string
	Emoji string
	Kind  Kind

	// Gear:
	Slot  Slot
	Bonus stats.Derived

	// Potion:
	Heal int // HP restored
	Mana int // MP restored
}
