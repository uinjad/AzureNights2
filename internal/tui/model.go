// Package tui is the terminal adapter: a Bubble Tea program that drives an
// app.Session. It follows the Elm architecture — Update mutates state in response
// to messages, View renders it — and switches scenes with a small mode state
// machine. It depends only on the app's use-cases and view models.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

type mode int

const (
	modeExploration mode = iota
	modeBattle
	modeGameOver
)

// Model is the Bubble Tea adapter over a game session.
type Model struct {
	session *app.Session
	mode    mode
}

// New wraps a started session in a TUI model.
func New(session *app.Session) Model {
	return Model{session: session, mode: modeFor(session)}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(key)
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "ctrl+s":
		_ = m.session.Save()
		return m, nil
	}
	if m.mode == modeExploration {
		if dir, ok := dirFromKey(key.String()); ok {
			_ = m.session.Move(dir)
			m.mode = modeFor(m.session) // a move may have started a battle
		}
	}
	return m, nil
}

func modeFor(s *app.Session) mode {
	switch {
	case s.GameOver():
		return modeGameOver
	case s.InBattle():
		return modeBattle
	default:
		return modeExploration
	}
}

func dirFromKey(k string) (world.Direction, bool) {
	switch k {
	case "up", "w":
		return world.North, true
	case "down", "s":
		return world.South, true
	case "left", "a":
		return world.West, true
	case "right", "d":
		return world.East, true
	default:
		return 0, false
	}
}
