package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

type mode int

const (
	modeExploration mode = iota
	modeBattle
	modeMenu
	modeGameOver
)

type tickMsg time.Time

// tickEvery schedules the next world tick on Bubble Tea's own goroutine; the
// result returns as a message, keeping the game loop single-threaded.
func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Model is the Bubble Tea adapter over a game session.
type Model struct {
	session *app.Session
	mode    mode
	bMenu   int // cursor in the battle action menu
	mMenu   int // cursor in the character menu
}

func New(session *app.Session) Model {
	return Model{session: session, mode: modeFor(session)}
}

func (m Model) Init() tea.Cmd { return tickEvery() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.mode == modeExploration { // world pauses in menus and battle
			m.session.Tick()
		}
		return m, tickEvery()
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
	case modeMenu:
		return m.updateMenu(key)
	}
	return m, nil
}

func (m Model) updateExploration(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "c" {
		m.mode, m.mMenu = modeMenu, 0
		return m, nil
	}
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
	options := 1 + len(skills)
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
			_ = m.session.Attack(0)
		} else if opt := skills[m.bMenu-1]; opt.Usable {
			_ = m.session.UseSkill(opt.ID, 0)
		} else {
			return m, nil
		}
		m.mode = modeFor(m.session)
		m.bMenu = 0
	}
	return m, nil
}

func (m Model) updateMenu(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := m.menuActions()
	switch key.String() {
	case "c", "esc":
		m.mode = modeExploration
	case "up", "w":
		if m.mMenu > 0 {
			m.mMenu--
		}
	case "down", "s":
		if m.mMenu < len(actions)-1 {
			m.mMenu++
		}
	case "enter", " ":
		if m.mMenu < len(actions) {
			actions[m.mMenu].do()
			if n := len(m.menuActions()); n == 0 {
				m.mMenu = 0
			} else if m.mMenu >= n {
				m.mMenu = n - 1
			}
		}
	}
	return m, nil
}

// menuAction is one selectable row on the character screen.
type menuAction struct {
	label string
	do    func()
}

func (m Model) menuActions() []menuAction {
	var out []menuAction
	for _, opt := range m.session.AdvancementView() {
		id := opt.ID
		out = append(out, menuAction{
			label: "⬆ Advance to " + opt.Name,
			do:    func() { _ = m.session.AdvanceTo(id) },
		})
	}
	for i, it := range m.session.InventoryView() {
		idx := i
		out = append(out, menuAction{
			label: fmt.Sprintf("🎒 Equip %s %s (%s)", it.Emoji, it.Name, it.Slot),
			do:    func() { _ = m.session.EquipFromInventory(idx) },
		})
	}
	return out
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
