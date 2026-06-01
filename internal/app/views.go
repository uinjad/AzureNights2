package app

import (
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

// HeroView is a read model of the hero for the UI — a flat, presentation-ready
// snapshot so the TUI never imports the domain's stat types.
type HeroView struct {
	Name      string
	ClassName string
	Level     int
	XP        int
	Gold      int
	HP, MaxHP int
	MP, MaxMP int
}

// HeroView builds the hero read model from live state.
func (s *Session) HeroView() HeroView {
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	c, _ := s.reg.Classes.Get(s.Hero.ClassID)
	return HeroView{
		Name: s.Hero.Name, ClassName: c.Name,
		Level: s.Hero.Level, XP: s.Hero.XP, Gold: s.Hero.Gold,
		HP: s.Hero.HP, MaxHP: d.MaxHP, MP: s.Hero.MP, MaxMP: d.MaxMP,
	}
}

// EnemyMarker is a renderable enemy standing on the map.
type EnemyMarker struct {
	Pos   world.Point
	Emoji string
}

// VisibleEnemies lists the enemies still on the map, for drawing.
func (s *Session) VisibleEnemies() []EnemyMarker {
	out := make([]EnemyMarker, 0, len(s.Spawns))
	for _, sp := range s.Spawns {
		out = append(out, EnemyMarker{Pos: sp.Pos, Emoji: s.reg.Enemies[sp.DefID].Emoji})
	}
	return out
}

// MapName returns the display name of the current map.
func (s *Session) MapName() string { return s.reg.Maps[s.MapID].Name }

// CombatantView is a presentation snapshot of one fighter.
type CombatantView struct {
	Name, Emoji          string
	HP, MaxHP, MP, MaxMP int
	Alive                bool
}

// BattleView is the read model the battle scene renders.
type BattleView struct {
	Player     CombatantView
	Enemies    []CombatantView
	PlayerTurn bool
	Log        []string
}

// BattleView builds the battle read model; ok is false when no fight is active.
func (s *Session) BattleView() (BattleView, bool) {
	if s.Battle == nil {
		return BattleView{}, false
	}
	view := func(c *combat.Combatant) CombatantView {
		return CombatantView{
			Name: c.Name, Emoji: c.Emoji,
			HP: c.HP, MaxHP: c.Stats.MaxHP, MP: c.MP, MaxMP: c.Stats.MaxMP,
			Alive: c.IsAlive(),
		}
	}
	bv := BattleView{Player: view(s.Battle.Player()), PlayerTurn: s.Battle.IsPlayerTurn(), Log: s.Battle.Log}
	for _, e := range s.Battle.Enemies() {
		bv.Enemies = append(bv.Enemies, view(e))
	}
	return bv, true
}

// SkillOption is a battle-menu entry for one of the hero's known skills.
type SkillOption struct {
	ID     string
	Name   string
	Emoji  string
	MPCost int
	Usable bool // enough MP and off cooldown right now
}

// BattleSkills lists the skills the hero has unlocked along its class path,
// flagged by whether they can be used this instant.
func (s *Session) BattleSkills() []SkillOption {
	chain, ok := s.reg.Classes.Path(s.Hero.ClassID)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []SkillOption
	for _, c := range chain {
		for _, id := range c.Skills {
			if seen[id] {
				continue
			}
			seen[id] = true
			sk, ok := s.reg.Skills[id]
			if !ok {
				continue
			}
			usable := s.Battle != nil && s.Battle.Player().CanUse(sk)
			out = append(out, SkillOption{ID: sk.ID, Name: sk.Name, Emoji: sk.Emoji, MPCost: sk.MPCost, Usable: usable})
		}
	}
	return out
}
