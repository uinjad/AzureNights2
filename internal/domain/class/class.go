 // Package class models the character advancement tree, in the spirit of
// Lineage 2 / Ragnarok Online: every character starts at a root class and
// advances along branches as it levels up, each branch granting attribute
// bonuses and unlocking skills.
//
// This package owns the *mechanism* (the tree and its rules); the concrete
// class data is supplied by the content layer later. It depends only on the
// stats package, keeping the dependency arrow pointing inward.
package class

import (
	"errors"
	"fmt"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

// ID uniquely identifies a class node within the tree.
type ID string

// Advance is an edge to a child class, gated by a minimum character level.
type Advance struct {
	To       ID
	MinLevel int
}

// Class is a single node in the advancement tree.
type Class struct {
	ID       ID
	Name     string
	Bonus    stats.Primary // attributes granted by reaching this class
	Skills   []string      // skill IDs unlocked here; resolved by the content layer
	Advances []Advance     // branches available from this class
}

// Sentinel errors callers can match with errors.Is.
var (
	ErrUnknownClass   = errors.New("class: unknown class id")
	ErrNotAdvanceable = errors.New("class: target is not a valid advancement")
)

// Tree is the immutable, validated set of classes plus the shared root every
// character starts from. Build it once at startup and treat it as read-only.
type Tree struct {
	root    ID
	classes map[ID]Class
}

// NewTree validates and assembles a tree. It fails if the root is missing or if
// any advancement points at a class that does not exist, so a malformed content
// file is caught at load time instead of mid-game.
func NewTree(root ID, classes ...Class) (*Tree, error) {
	index := make(map[ID]Class, len(classes))
	for _, c := range classes {
		index[c.ID] = c
	}
	if _, ok := index[root]; !ok {
		return nil, fmt.Errorf("%w: root %q", ErrUnknownClass, root)
	}
	for _, c := range classes {
		for _, a := range c.Advances {
			if _, ok := index[a.To]; !ok {
				return nil, fmt.Errorf("%w: %q advances to %q", ErrUnknownClass, c.ID, a.To)
			}
		}
	}
	return &Tree{root: root, classes: index}, nil
}

// Get returns a class by ID.
func (t *Tree) Get(id ID) (Class, bool) {
	c, ok := t.classes[id]
	return c, ok
}

// Root returns the starting class.
func (t *Tree) Root() Class { return t.classes[t.root] }

// Options lists the classes a character of the given level may advance into
// from its current class. An empty slice means no advancement is available yet.
func (t *Tree) Options(from ID, level int) []Class {
	current, ok := t.classes[from]
	if !ok {
		return nil
	}
	var out []Class
	for _, a := range current.Advances {
		if level >= a.MinLevel {
			out = append(out, t.classes[a.To])
		}
	}
	return out
}

// Advance moves a character from one class to a target branch, enforcing both
// that the edge exists and that the level requirement is met.
func (t *Tree) Advance(from, to ID, level int) (Class, error) {
	current, ok := t.classes[from]
	if !ok {
		return Class{}, fmt.Errorf("%w: %q", ErrUnknownClass, from)
	}
	for _, a := range current.Advances {
		if a.To == to && level >= a.MinLevel {
			return t.classes[to], nil
		}
	}
	return Class{}, fmt.Errorf("%w: %q -> %q at level %d", ErrNotAdvanceable, from, to, level)
}

// Path returns the chain of classes from the root down to the target,
// inclusive. It is the basis for any cumulative effect along progression.
func (t *Tree) Path(to ID) ([]Class, bool) {
	var walk func(id ID) ([]Class, bool)
	walk = func(id ID) ([]Class, bool) {
		c, ok := t.classes[id]
		if !ok {
			return nil, false
		}
		if id == to {
			return []Class{c}, true
		}
		for _, a := range c.Advances {
			if chain, found := walk(a.To); found {
				return append([]Class{c}, chain...), true
			}
		}
		return nil, false
	}
	return walk(t.root)
}

// CumulativePrimary sums the attribute bonuses of every class along the path
// from the root to the target. This is how the class tree feeds the stats
// pipeline: bonuses accumulate here, then stats.Derive turns them into combat
// values.
func (t *Tree) CumulativePrimary(to ID) (stats.Primary, bool) {
	chain, ok := t.Path(to)
	if !ok {
		return stats.Primary{}, false
	}
	var sum stats.Primary
	for _, c := range chain {
		sum.STR += c.Bonus.STR
		sum.DEX += c.Bonus.DEX
		sum.CON += c.Bonus.CON
		sum.INT += c.Bonus.INT
		sum.WIT += c.Bonus.WIT
		sum.MEN += c.Bonus.MEN
	}
	return sum, true
}