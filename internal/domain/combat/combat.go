// Package combat is the turn-based battle engine. It operates on Combatant
// snapshots — fixed copies of derived stats plus mutable HP/MP pools — and is
// intentionally decoupled from how those stats were produced. The app layer
// translates the hero (via character.EffectiveStats) and enemy definitions into
// Combatants when a battle starts, then writes the outcome back afterward.
//
// Everything here is deterministic: no randomness, so tests assert exact
// numbers. Variance and criticals can be layered in later behind an injected
// source without reshaping the engine.
package combat

import (
	"sort"

	"github.com/uinjad/AzureNights2/internal/domain/faction"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

type Side int

const (
	SidePlayer Side = iota
	SideEnemy
)

// Combatant is one participant in a battle.
type Combatant struct {
	Name      string
	Emoji     string
	Side      Side
	Faction   faction.ID
	Stats     stats.Derived
	HP        int
	MP        int
	cooldowns map[string]int
}

func NewCombatant(name, emoji string, side Side, st stats.Derived) *Combatant {
	return &Combatant{
		Name: name, Emoji: emoji, Side: side, Stats: st,
		HP: st.MaxHP, MP: st.MaxMP, cooldowns: make(map[string]int),
	}
}

func (c *Combatant) IsAlive() bool { return c.HP > 0 }

// PhysicalDamage is the subtractive base, floored at 1. hit() layers variance,
// crit, and faction on top of it.
func PhysicalDamage(atk, def int) int {
	if dmg := atk - def; dmg > 1 {
		return dmg
	}
	return 1
}

// Order returns the combatants sorted by initiative, highest first (stable).
func Order(combatants []*Combatant) []*Combatant {
	out := make([]*Combatant, len(combatants))
	copy(out, combatants)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Stats.Init > out[j].Stats.Init
	})
	return out
}
