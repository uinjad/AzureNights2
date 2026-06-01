package tui

import (
	"time"

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

// tickMsg is delivered on the world clock's interval.
type tickMsg time.Time

// tickEvery schedules the next world tick. Bubble Tea runs the timer on its own
// goroutine and delivers the result as a message, so the game loop stays
// single-threaded and lock-free — the framework is the concurrency boundary.
func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Model is the Bubble Tea adapter over a game session.
type Model struct {
	session *app.Session
	mode    mode
	bMenu   int // cursor in the battle action menu
}

// New wraps a started session in a TUI model.
func New(session *app.Session) Model {
	return Model{session: session, mode: modeFor(session)}
}

// Init starts the world clock ticking.
func (m Model) Init() tea.Cmd { return tickEvery() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.session.Tick() // advances the living world; a no-op during battle
		m.mode = modeFor(m.session)
		return m, tickEvery() // keep the clock running
	case tea.KeyMsg:
		return m.handleKey(msg)
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
	switch m.mode {
	case modeExploration:
		return m.updateExploration(key)
	case modeBattle:
		return m.updateBattle(key)
	}
	return m, nil
}

func (m Model) updateExploration(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if dir, ok := dirFromKey(key.String()); ok {
		_ = m.session.Move(dir)
		m.mode = modeFor(m.session)
		if m.mode == modeBattle {
			m.bMenu = 0
		}
	}
	return m, nil
}

func (m Model) updateBattle(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	skills := m.session.BattleSkills()
	options := 1 + len(skills) // Attack + each known skill
	switch key.String() {
	case "up", "w":
		if m.bMenu > 0 {
			m.bMenu--
		}
	case "down", "s":
		if m.bMenu < options-1 {
			m.bMenu++
		}
	case "enter", " ":
		if m.bMenu == 0 {
			_ = m.session.Attack(0) // single enemy in MVP -> target 0
		} else if opt := skills[m.bMenu-1]; opt.Usable {
			_ = m.session.UseSkill(opt.ID, 0)
		} else {
			return m, nil // unusable choice; ignore
		}
		m.mode = modeFor(m.session)
		m.bMenu = 0
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
