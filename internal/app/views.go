package app

import "github.com/uinjad/AzureNights2/internal/domain/world"

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
