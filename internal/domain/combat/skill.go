package combat

import "errors"

// ErrSkillUnavailable means the skill is on cooldown or the caster lacks MP.
var ErrSkillUnavailable = errors.New("combat: skill unavailable (no MP or on cooldown)")

// DamageKind selects which attack/defense pair a skill uses.
type DamageKind int

const (
	Physical DamageKind = iota // PAtk vs PDef
	Magical                    // MAtk vs MDef
)

// Skill is an active ability. Power is added to the relevant attack stat; a
// Cooldown of 0 means it can be used every turn.
type Skill struct {
	ID       string
	Name     string
	Emoji    string
	Kind     DamageKind
	MPCost   int
	Cooldown int
	Power    int
}

// SkillResult records a cast for the log and UI.
type SkillResult struct {
	Caster   string
	Skill    string
	Target   string
	Damage   int
	Defeated bool
}

// CanUse reports whether the combatant has the MP and a ready cooldown.
func (c *Combatant) CanUse(s Skill) bool {
	return c.MP >= s.MPCost && c.cooldowns[s.ID] == 0
}

// UseSkill spends MP, deals damage of the skill's kind, and starts its cooldown.
func (c *Combatant) UseSkill(s Skill, target *Combatant) (SkillResult, error) {
	if !c.CanUse(s) {
		return SkillResult{}, ErrSkillUnavailable
	}
	c.MP -= s.MPCost

	var dmg int
	if s.Kind == Magical {
		dmg = skillDamage(c.Stats.MAtk, target.Stats.MDef, s.Power)
	} else {
		dmg = skillDamage(c.Stats.PAtk, target.Stats.PDef, s.Power)
	}
	target.HP -= dmg
	if target.HP < 0 {
		target.HP = 0
	}
	if s.Cooldown > 0 {
		c.cooldowns[s.ID] = s.Cooldown
	}
	return SkillResult{
		Caster:   c.Name,
		Skill:    s.Name,
		Target:   target.Name,
		Damage:   dmg,
		Defeated: !target.IsAlive(),
	}, nil
}

func skillDamage(atk, def, power int) int {
	if dmg := atk + power - def; dmg > 1 {
		return dmg
	}
	return 1
}

// tickCooldowns decrements every running cooldown by one, floored at zero. The
// battle calls this at the start of a combatant's turn, so "Cooldown: N" means
// "skip your next N turns".
func (c *Combatant) tickCooldowns() {
	for id, left := range c.cooldowns {
		if left > 0 {
			c.cooldowns[id] = left - 1
		}
	}
}
