// Package faction models the rock-paper-scissors allegiances that flavor combat.
// Each faction beats exactly one other and is beaten by one other, forming a
// cycle (Solar > Illumite > Lawful > Solar). The multipliers that make the cycle
// bite are data, loaded from content, so the balance is moddable without code.
package faction

import "fmt"

// ID identifies a faction; the empty ID means "no allegiance" (neutral).
type ID string

const Neutral ID = ""

// Relation describes how an attacker's faction fares against a defender's.
type Relation int

const (
	Even Relation = iota
	Advantage
	Disadvantage
)

// Table holds the cycle and the damage multipliers.
type Table struct {
	beats   map[ID]ID
	names   map[ID]string
	advMult float64
	disMult float64
}

// NewTable assembles a table, validating that every "beats" target is a known
// faction so a malformed factions file fails at load instead of mid-fight.
func NewTable(advMult, disMult float64, names map[ID]string, beats map[ID]ID) (*Table, error) {
	for from, to := range beats {
		if _, ok := names[from]; !ok {
			return nil, fmt.Errorf("faction: unknown faction %q", from)
		}
		if _, ok := names[to]; !ok {
			return nil, fmt.Errorf("faction: %q beats unknown faction %q", from, to)
		}
	}
	return &Table{beats: beats, names: names, advMult: advMult, disMult: disMult}, nil
}

// Relation reports how the attacker fares against the defender.
func (t *Table) Relation(attacker, defender ID) Relation {
	if attacker == Neutral || defender == Neutral {
		return Even
	}
	switch {
	case t.beats[attacker] == defender:
		return Advantage
	case t.beats[defender] == attacker:
		return Disadvantage
	default:
		return Even
	}
}

// DamageMultiplier is the factor an attacker's damage is scaled by.
func (t *Table) DamageMultiplier(attacker, defender ID) float64 {
	switch t.Relation(attacker, defender) {
	case Advantage:
		return t.advMult
	case Disadvantage:
		return t.disMult
	default:
		return 1.0
	}
}

// Name returns a faction's display name ("" for neutral).
func (t *Table) Name(id ID) string { return t.names[id] }
