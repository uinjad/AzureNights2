package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

// engageFirstEnemy walks the hero onto the first map enemy to start a battle.
func engageFirstEnemy(t *testing.T, s *app.Session) {
	t.Helper()
	target := s.Spawns[0].Pos
	tm := s.Map()
	tries := []struct {
		dir  world.Direction
		from world.Point
	}{
		{world.North, world.Point{X: target.X, Y: target.Y + 1}},
		{world.South, world.Point{X: target.X, Y: target.Y - 1}},
		{world.East, world.Point{X: target.X - 1, Y: target.Y}},
		{world.West, world.Point{X: target.X + 1, Y: target.Y}},
	}
	for _, tr := range tries {
		if tm.Walkable(tr.from) {
			s.PlayerPos = tr.from
			_ = s.Move(tr.dir)
			if s.InBattle() {
				return
			}
		}
	}
	t.Fatal("could not engage the enemy")
}

func TestTickAdvancesTheWorld(t *testing.T) {
	m := newModel(t) // exploration model from model_test.go
	before := m.session.Clock.Tick

	updated, cmd := m.Update(tickMsg(time.Now()))
	if updated.(Model).session.Clock.Tick != before+1 {
		t.Error("a tick should advance the world clock")
	}
	if cmd == nil {
		t.Error("a tick should reschedule itself")
	}
}

func TestAttackInBattleDamagesEnemy(t *testing.T) {
	reg, _ := content.Load()
	s := app.New(reg, noRepo{}, app.WithRoll(func() float64 { return 1.0 }))
	if err := s.NewGame("Aria"); err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	engageFirstEnemy(t, s)

	m := New(s)
	if m.mode != modeBattle {
		t.Fatal("expected battle mode")
	}
	before, _ := s.BattleView()
	hpBefore := before.Enemies[0].HP

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm "Attack"
	m = updated.(Model)

	if after, ok := s.BattleView(); ok && after.Enemies[0].HP >= hpBefore {
		t.Errorf("attacking should hurt the enemy: %d -> %d", hpBefore, after.Enemies[0].HP)
	}
}

func TestBattleMenuListsSkillsAfterAdvancing(t *testing.T) {
	reg, _ := content.Load()
	s := app.New(reg, noRepo{}, app.WithRoll(func() float64 { return 1.0 }))
	_ = s.NewGame("Aria")
	if _, err := s.Hero.AddXP(reg.Classes, 1000); err != nil { // reach level 5
		t.Fatalf("AddXP: %v", err)
	}
	if err := s.AdvanceClass("fighter"); err != nil {
		t.Fatalf("AdvanceClass: %v", err)
	}
	engageFirstEnemy(t, s)

	skills := s.BattleSkills()
	if len(skills) == 0 || skills[0].ID != "power_strike" {
		t.Fatalf("fighter should know power_strike, got %+v", skills)
	}
	if view := New(s).View(); !strings.Contains(view, "Power Strike") {
		t.Errorf("battle menu should list the skill:\n%s", view)
	}
}
