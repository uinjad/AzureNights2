package combat

import "errors"

var ErrSkillUnavailable = errors.New("combat: skill unavailable (no MP or on cooldown)")

type DamageKind int

const (
	Physical DamageKind = iota
	Magical
)

type Skill struct {
	ID       string
	Name     string
	Emoji    string
	Kind     DamageKind
	MPCost   int
	Cooldown int
	Power    int
}

// CanUse reports whether the combatant has the MP and a ready cooldown.
func (c *Combatant) CanUse(s Skill) bool {
	return c.MP >= s.MPCost && c.cooldowns[s.ID] == 0
}

// tickCooldowns decrements running cooldowns; called at the start of a turn.
func (c *Combatant) tickCooldowns() {
	for id, left := range c.cooldowns {
		if left > 0 {
			c.cooldowns[id] = left - 1
		}
	}
}
