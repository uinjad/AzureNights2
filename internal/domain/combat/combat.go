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

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

// Side tells the engine who fights for whom.
type Side int

const (
	SidePlayer Side = iota
	SideEnemy
)

// Combatant is one participant in a battle. Stats is a snapshot taken at battle
// start; HP and MP are the pools that change as the fight unfolds.
type Combatant struct {
	Name  string
	Emoji string
	Side  Side
	Stats stats.Derived
	HP    int
	MP    int

	cooldowns map[string]int // skill ID -> turns remaining
}

func NewCombatant(name, emoji string, side Side, st stats.Derived) *Combatant {
	return &Combatant{
		Name:      name,
		Emoji:     emoji,
		Side:      side,
		Stats:     st,
		HP:        st.MaxHP,
		MP:        st.MaxMP,
		cooldowns: make(map[string]int),
	}
}

// IsAlive reports whether the combatant can still act.
func (c *Combatant) IsAlive() bool { return c.HP > 0 }

// PhysicalDamage is the core melee formula: attack minus defense, floored at 1
// so every blow chips at least a little. Kept simple and linear on purpose —
// it is a balancing knob.
func PhysicalDamage(atk, def int) int {
	if dmg := atk - def; dmg > 1 {
		return dmg
	}
	return 1
}

// AttackResult records the outcome of a single physical attack, for the battle
// log and the UI.
type AttackResult struct {
	Attacker string
	Target   string
	Damage   int
	Defeated bool
}

// Attack performs a basic physical strike, applying damage to the target and
// reporting what happened.
func (a *Combatant) Attack(target *Combatant) AttackResult {
	dmg := PhysicalDamage(a.Stats.PAtk, target.Stats.PDef)
	target.HP -= dmg
	if target.HP < 0 {
		target.HP = 0
	}
	return AttackResult{
		Attacker: a.Name,
		Target:   target.Name,
		Damage:   dmg,
		Defeated: !target.IsAlive(),
	}
}

// Order returns the combatants sorted by initiative, highest first. The sort is
// stable, so ties resolve by input order — letting callers control tie-breaking
// simply by how they pass the slice in.
func Order(combatants []*Combatant) []*Combatant {
	out := make([]*Combatant, len(combatants))
	copy(out, combatants)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Stats.Init > out[j].Stats.Init
	})
	return out
}
