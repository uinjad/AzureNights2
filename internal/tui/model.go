package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

type mode int

const (
	modeName mode = iota
	modeExploration
	modeBattle
	modeMenu
	modeGameOver
)

type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Model is the Bubble Tea adapter over a game session.
type Model struct {
	session        *app.Session
	mode           mode
	bMenu          int    // battle menu cursor
	mMenu          int    // character menu cursor
	nameInput      string // hero name being typed on the title screen
	confirmingQuit bool   // showing the quit confirmation
}

func New(session *app.Session) Model {
	m := Model{session: session}
	if session.Started() {
		m.mode = modeFor(session)
	} else {
		m.mode = modeName
	}
	return m
}

func (m Model) Init() tea.Cmd { return tickEvery() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.mode == modeExploration && !m.confirmingQuit { // world pauses elsewhere
			m.session.Tick()
		}
		return m, tickEvery()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := key.String()

	if m.confirmingQuit {
		if s == "y" || s == "Y" {
			return m, tea.Quit
		}
		m.confirmingQuit = false
		return m, nil
	}
	if s == "ctrl+c" { // hard escape hatch
		return m, tea.Quit
	}
	if m.mode == modeName {
		return m.updateName(key)
	}

	switch s {
	case "q":
		m.confirmingQuit = true
		return m, nil
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

func (m Model) updateName(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "ctrl+l" {
		if m.session.HasSave() {
			if err := m.session.LoadGame(); err == nil {
				m.mode = modeFor(m.session)
			}
		}
		return m, nil
	}
	switch key.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput)
		if name == "" {
			name = "Aria"
		}
		_ = m.session.NewGame(name)
		m.mode = modeExploration
	case tea.KeyBackspace, tea.KeyDelete:
		if r := []rune(m.nameInput); len(r) > 0 {
			m.nameInput = string(r[:len(r)-1])
		}
	case tea.KeySpace:
		if len([]rune(m.nameInput)) < 16 {
			m.nameInput += " "
		}
	case tea.KeyRunes:
		if len([]rune(m.nameInput)) < 16 {
			m.nameInput += string(key.Runes)
		}
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
		if it.Kind == "potion" {
			out = append(out, menuAction{
				label: "Use " + it.Name,
				do:    func() { _ = m.session.UsePotion(idx) },
			})
		} else {
			out = append(out, menuAction{
				label: fmt.Sprintf("Equip %s (%s)", it.Name, it.Slot),
				do:    func() { _ = m.session.EquipFromInventory(idx) },
			})
		}
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
