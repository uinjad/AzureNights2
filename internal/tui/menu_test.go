package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
)

func openMenu(m Model) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	return updated.(Model)
}

func TestMenuShowsInventory(t *testing.T) {
	m := openMenu(newModel(t))
	if m.mode != modeMenu {
		t.Fatal("'c' should open the character menu")
	}
	if view := m.View(); !strings.Contains(view, "Iron Sword") {
		t.Errorf("menu should list a starter item:\n%s", view)
	}
}

func TestEquipThroughMenu(t *testing.T) {
	reg, _ := content.Load()
	s := app.New(reg, noRepo{}, app.WithRoll(func() float64 { return 1.0 }))
	_ = s.NewGame("Aria")
	before, _ := s.Hero.EffectiveStats(reg.Classes)
	invBefore := len(s.InventoryView())

	m := openMenu(New(s)) // no advancement at lvl 1 → first row is an item
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated.(Model)

	after, _ := s.Hero.EffectiveStats(reg.Classes)
	if after.PAtk == before.PAtk && after.PDef == before.PDef {
		t.Error("equipping should change stats")
	}
	if len(s.InventoryView()) >= invBefore {
		t.Error("equipping should remove the item from the bag")
	}
}

func TestAdvanceThroughMenu(t *testing.T) {
	reg, _ := content.Load()
	s := app.New(reg, noRepo{}, app.WithRoll(func() float64 { return 1.0 }))
	_ = s.NewGame("Aria")
	if _, err := s.Hero.AddXP(reg.Classes, 1000); err != nil { // reach level 5
		t.Fatalf("AddXP: %v", err)
	}

	m := openMenu(New(s))
	if view := m.View(); !strings.Contains(view, "Advance to Solar Initiate") {
		t.Fatalf("menu should offer advancement:\n%s", view)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // first branch = Solar Initiate
	_ = updated.(Model)
	if s.Hero.ClassID != "solar_initiate" {
		t.Errorf("expected solar_initiate, got %q", s.Hero.ClassID)
	}
}
