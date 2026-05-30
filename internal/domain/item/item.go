// Package item defines equippable gear and the stat bonuses it grants while
// worn. The MVP keeps exactly two slots: a weapon and a piece of armor.
package item

import "github.com/uinjad/AzureNights2/internal/domain/stats"

// Slot is an equipment slot.
type Slot int

const (
	Weapon Slot = iota
	Armor
)

// Item is a piece of equippable gear. Its Bonus is added on top of a
// character's derived stats while equipped.
type Item struct {
	ID    string
	Name  string
	Emoji string
	Slot  Slot
	Bonus stats.Derived
}
