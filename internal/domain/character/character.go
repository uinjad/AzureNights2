// Package character defines the single playable hero and binds the domain
// together: it turns a class (from the advancement tree) plus a level plus
// equipped gear into the concrete stats combat will use.
//
// A Character is plain, serializable data — it stores its class by ID, not a
// pointer to the shared class tree. Anything that needs the tree's rules takes
// it as a parameter. That keeps the hero trivially saveable (Stage 9) while the
// tree stays a single shared piece of content.
package character

import (
	"errors"
	"fmt"

	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/item"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

// ErrNegativeXP is returned when AddXP is called with a negative amount.
var ErrNegativeXP = errors.New("character: xp amount must be non-negative")

// Character is the player-controlled hero. HP and MP are the mutable pools that
// change during play; their ceilings come from EffectiveStats.
type Character struct {
	Name    string
	ClassID class.ID
	Level   int
	XP      int // progress toward the next level; resets on level up
	Gold    int

	HP int
	MP int

	Equipment map[item.Slot]item.Item
}

// New creates a fresh hero at the root class, level 1, with full pools.
func New(name string, tree *class.Tree) (*Character, error) {
	c := &Character{
		Name:      name,
		ClassID:   tree.Root().ID,
		Level:     1,
		Equipment: make(map[item.Slot]item.Item),
	}
	if err := c.restore(tree); err != nil {
		return nil, err
	}
	return c, nil
}

// XPForNext returns the experience needed to go from the given level to the
// next. A gentle linear curve keeps early progression brisk; it is a balancing
// knob, not a constant.
func XPForNext(level int) int { return 100 * level }

// EffectiveStats is the payoff of the whole domain pipeline: cumulative class
// attributes -> derived combat values -> equipment bonuses.
func (c *Character) EffectiveStats(tree *class.Tree) (stats.Derived, error) {
	primary, ok := tree.CumulativePrimary(c.ClassID)
	if !ok {
		return stats.Derived{}, fmt.Errorf("%w: %q", class.ErrUnknownClass, c.ClassID)
	}
	d := stats.Derive(primary, c.Level)
	for _, it := range c.Equipment {
		d = addDerived(d, it.Bonus)
	}
	return d, nil
}

// AddXP grants experience, leveling up as many times as the total allows. Each
// level gained fully restores HP and MP, JRPG-style. It returns how many levels
// were gained.
func (c *Character) AddXP(tree *class.Tree, amount int) (int, error) {
	if amount < 0 {
		return 0, ErrNegativeXP
	}
	c.XP += amount
	gained := 0
	for c.XP >= XPForNext(c.Level) {
		c.XP -= XPForNext(c.Level)
		c.Level++
		gained++
	}
	if gained > 0 {
		if err := c.restore(tree); err != nil {
			return gained, err
		}
	}
	return gained, nil
}

// Advance switches the hero onto a new class branch, enforcing the tree's level
// gate. Stats rise immediately because the new class adds its attribute bonus.
func (c *Character) Advance(tree *class.Tree, to class.ID) error {
	next, err := tree.Advance(c.ClassID, to, c.Level)
	if err != nil {
		return err
	}
	c.ClassID = next.ID
	return c.clampPools(tree)
}

// Equip puts an item into its slot, replacing whatever was there.
func (c *Character) Equip(tree *class.Tree, it item.Item) error {
	if c.Equipment == nil {
		c.Equipment = make(map[item.Slot]item.Item)
	}
	c.Equipment[it.Slot] = it
	return c.clampPools(tree)
}

// Unequip clears a slot.
func (c *Character) Unequip(tree *class.Tree, slot item.Slot) error {
	delete(c.Equipment, slot)
	return c.clampPools(tree)
}

// restore tops both pools up to their current maximums.
func (c *Character) restore(tree *class.Tree) error {
	d, err := c.EffectiveStats(tree)
	if err != nil {
		return err
	}
	c.HP, c.MP = d.MaxHP, d.MaxMP
	return nil
}

// clampPools keeps current HP/MP within [0, max] after anything that can change
// the maximums (equipping, unequipping, advancing).
func (c *Character) clampPools(tree *class.Tree) error {
	d, err := c.EffectiveStats(tree)
	if err != nil {
		return err
	}
	c.HP = clamp(c.HP, 0, d.MaxHP)
	c.MP = clamp(c.MP, 0, d.MaxMP)
	return nil
}

func clamp(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

func addDerived(a, b stats.Derived) stats.Derived {
	return stats.Derived{
		MaxHP: a.MaxHP + b.MaxHP,
		MaxMP: a.MaxMP + b.MaxMP,
		PAtk:  a.PAtk + b.PAtk,
		MAtk:  a.MAtk + b.MAtk,
		PDef:  a.PDef + b.PDef,
		MDef:  a.MDef + b.MDef,
		Init:  a.Init + b.Init,
	}
}
