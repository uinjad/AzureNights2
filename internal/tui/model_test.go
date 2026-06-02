package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
)

type noRepo struct{}

func (noRepo) Save(*app.Snapshot) error     { return nil }
func (noRepo) Load() (*app.Snapshot, error) { return nil, nil }
func (noRepo) Exists() bool                 { return false }

func newModel(t *testing.T) Model {
	t.Helper()
	reg, err := content.Load()
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	s := app.New(reg, noRepo{}, app.WithRoll(func() float64 { return 1.0 }))
	if err := s.NewGame("Aria"); err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return New(s)
}

func TestExplorationViewRenders(t *testing.T) {
	view := newModel(t).View()
	for _, want := range []string{"Aria", "Eldwood Forest", "HP"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestMovementUpdatesPosition(t *testing.T) {
	m := newModel(t)
	start := m.session.PlayerPos
	for _, key := range []tea.KeyType{tea.KeyRight, tea.KeyDown, tea.KeyUp, tea.KeyLeft} {
		updated, _ := m.Update(tea.KeyMsg{Type: key})
		if updated.(Model).session.PlayerPos != start {
			return
		}
		m = updated.(Model)
	}
	t.Errorf("hero should have moved from %+v in some direction", start)
}

func TestQuitAsksForConfirmationThenQuits(t *testing.T) {
	m := newModel(t)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if !m.confirmingQuit {
		t.Fatal("q should ask for confirmation, not quit immediately")
	}
	if cmd != nil {
		t.Error("q alone should not issue a command")
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y should confirm the quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("y should issue tea.Quit")
	}
}
