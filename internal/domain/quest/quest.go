// Package quest models data-driven objectives: a quest is a list of objectives
// (defeat N of an enemy, reach a map) plus a reward. The package owns the rules
// — matching an event to an objective, capping progress, deciding completion —
// while the concrete quest data comes from the content layer and the live
// progress lives in the session. It depends on nothing else in the domain.
package quest

import (
	"errors"
	"fmt"
)

// ObjectiveKind is what an objective tracks.
type ObjectiveKind int

const (
	DefeatEnemy ObjectiveKind = iota // Target = enemy id, Count = how many
	ReachMap                         // Target = map id
)

func (k ObjectiveKind) String() string {
	switch k {
	case DefeatEnemy:
		return "defeat"
	case ReachMap:
		return "reach"
	default:
		return "unknown"
	}
}

// Objective is one goal within a quest.
type Objective struct {
	Kind   ObjectiveKind
	Target string
	Count  int
	Desc   string
}

// Required is how many times the objective's event must fire to satisfy it.
func (o Objective) Required() int {
	if o.Kind == ReachMap {
		return 1
	}
	if o.Count < 1 {
		return 1
	}
	return o.Count
}

// Reward is granted once when every objective is satisfied.
type Reward struct{ XP, Gold int }

// Quest is a named bundle of objectives.
type Quest struct {
	ID         string
	Name       string
	Summary    string
	Objectives []Objective
	Reward     Reward
}

// Event is something quests react to.
type Event struct {
	Kind   ObjectiveKind
	Target string
}

// Apply advances counts (one int per objective) for an event, returning whether
// anything changed. counts must be len(q.Objectives).
func (q Quest) Apply(counts []int, e Event) bool {
	changed := false
	for i, o := range q.Objectives {
		if o.Kind == e.Kind && o.Target == e.Target && counts[i] < o.Required() {
			counts[i]++
			changed = true
		}
	}
	return changed
}

// Complete reports whether every objective is satisfied.
func (q Quest) Complete(counts []int) bool {
	for i, o := range q.Objectives {
		if counts[i] < o.Required() {
			return false
		}
	}
	return true
}

// Set is a validated, immutable collection of quests.
type Set struct {
	quests []Quest
	byID   map[string]Quest
}

// NewSet validates and assembles a set: ids are unique and every quest has at
// least one objective, so a malformed quests file fails at load time.
func NewSet(quests ...Quest) (*Set, error) {
	byID := make(map[string]Quest, len(quests))
	for _, q := range quests {
		if q.ID == "" {
			return nil, errors.New("quest: empty quest id")
		}
		if _, dup := byID[q.ID]; dup {
			return nil, fmt.Errorf("quest: duplicate id %q", q.ID)
		}
		if len(q.Objectives) == 0 {
			return nil, fmt.Errorf("quest: %q has no objectives", q.ID)
		}
		byID[q.ID] = q
	}
	return &Set{quests: quests, byID: byID}, nil
}

func (s *Set) All() []Quest                { return s.quests }
func (s *Set) Get(id string) (Quest, bool) { q, ok := s.byID[id]; return q, ok }
