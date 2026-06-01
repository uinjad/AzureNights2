package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/uinjad/AzureNights2/internal/domain/world"
)

func (m Model) View() string {
	switch m.mode {
	case modeMenu:
		return m.viewMenu()
	case modeBattle:
		return m.viewBattle()
	case modeGameOver:
		return m.viewGameOver()
	default:
		return m.viewExploration()
	}
}

func (m Model) viewExploration() string {
	tint, timeLabel := timeStyle(m.session.Clock.TimeOfDay)

	header := lipgloss.NewStyle().Foreground(tint).Bold(true).
		Render(fmt.Sprintf(" %s  ·  %s · %s ",
			m.session.MapName(), timeLabel, weatherLabel(m.session.Clock.Weather)))

	mapBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(tint).Padding(0, 1).
		Render(m.renderMap())

	top := lipgloss.JoinHorizontal(lipgloss.Top, mapBox, "  ", panelStyle().Render(m.renderStatus()))
	logBox := panelStyle().Width(lipgloss.Width(top) - 2).Render(m.renderLog())
	footer := dimStyle().Render(" arrows/wasd move · c character · ctrl+s save · q quit ")

	return lipgloss.JoinVertical(lipgloss.Left, header, top, logBox, footer)
}

func (m Model) renderMap() string {
	tm := m.session.Map()
	hero := m.session.PlayerPos
	enemies := make(map[world.Point]string)
	for _, e := range m.session.VisibleEnemies() {
		enemies[e.Pos] = e.Emoji
	}

	var b strings.Builder
	for y := 0; y < tm.H; y++ {
		for x := 0; x < tm.W; x++ {
			p := world.Point{X: x, Y: y}
			switch {
			case p == hero:
				b.WriteString("🧝")
			case enemies[p] != "":
				b.WriteString(enemies[p])
			default:
				t, _ := tm.At(p)
				b.WriteString(t.Emoji)
			}
		}
		if y < tm.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderStatus() string {
	h := m.session.HeroView()
	return strings.Join([]string{
		fmt.Sprintf("%s — %s  Lv%d", h.Name, h.ClassName, h.Level),
		fmt.Sprintf("HP %s %d/%d", bar(h.HP, h.MaxHP, 10), h.HP, h.MaxHP),
		fmt.Sprintf("MP %s %d/%d", bar(h.MP, h.MaxMP, 10), h.MP, h.MaxMP),
		fmt.Sprintf("💰 %d   ✨ %d XP", h.Gold, h.XP),
	}, "\n")
}

func (m Model) renderLog() string {
	log := m.session.Log
	const n = 5
	if len(log) > n {
		log = log[len(log)-n:]
	}
	return strings.Join(log, "\n")
}

func (m Model) viewGameOver() string {
	return alertStyle().Render("💀  You have fallen.\n\nPress q to quit.")
}

// --- styles & helpers ---

func panelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).Padding(0, 1)
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
}

func alertStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).Padding(1, 2)
}

// timeStyle maps the day phase to a tint and label — the living-world palette.
func timeStyle(t world.TimeOfDay) (lipgloss.Color, string) {
	switch t {
	case world.Dawn:
		return lipgloss.Color("217"), "🌅 dawn"
	case world.Day:
		return lipgloss.Color("220"), "☀️  day"
	case world.Dusk:
		return lipgloss.Color("208"), "🌆 dusk"
	default:
		return lipgloss.Color("63"), "🌙 night"
	}
}

func weatherLabel(w world.Weather) string {
	switch w {
	case world.Rain:
		return "🌧 rain"
	case world.Fog:
		return "🌫 fog"
	default:
		return "✨ clear"
	}
}

func bar(cur, max, width int) string {
	if max <= 0 {
		max = 1
	}
	filled := cur * width / max
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func (m Model) viewBattle() string {
	bv, ok := m.session.BattleView()
	if !ok {
		return ""
	}

	var lines []string
	for _, e := range bv.Enemies {
		lines = append(lines, fmt.Sprintf("%s %-12s HP %s %d/%d",
			e.Emoji, e.Name, bar(e.HP, e.MaxHP, 12), e.HP, e.MaxHP))
	}
	p := bv.Player
	lines = append(lines, "",
		fmt.Sprintf("🧝 %-12s HP %s %d/%d", p.Name, bar(p.HP, p.MaxHP, 12), p.HP, p.MaxHP),
		fmt.Sprintf("   %-12s MP %s %d/%d", "", bar(p.MP, p.MaxMP, 12), p.MP, p.MaxMP),
		"")

	lines = append(lines, m.menuItem(0, "⚔ Attack", true))
	for i, sk := range m.session.BattleSkills() {
		lines = append(lines, m.menuItem(i+1, fmt.Sprintf("%s %s (%d MP)", sk.Emoji, sk.Name, sk.MPCost), sk.Usable))
	}

	battleBox := alertStyle().Render(strings.Join(lines, "\n"))

	log := bv.Log
	const n = 4
	if len(log) > n {
		log = log[len(log)-n:]
	}
	logBox := panelStyle().Render(strings.Join(log, "\n"))
	hint := dimStyle().Render(" ↑/↓ choose · enter act · q quit ")

	return lipgloss.JoinVertical(lipgloss.Left, battleBox, logBox, hint)
}

// menuItem renders one action row: a cursor when selected, greyed when unusable.
func (m Model) menuItem(idx int, label string, usable bool) string {
	cursor := "  "
	if idx == m.bMenu {
		cursor = "▸ "
	}
	style := lipgloss.NewStyle()
	switch {
	case !usable:
		style = style.Foreground(lipgloss.Color("240"))
	case idx == m.bMenu:
		style = style.Foreground(lipgloss.Color("220")).Bold(true)
	}
	return style.Render(cursor + label)
}

func (m Model) viewMenu() string {
	h := m.session.HeroView()
	eq := m.session.EquippedView()

	stats := strings.Join([]string{
		fmt.Sprintf("%s — %s  Lv%d", h.Name, h.ClassName, h.Level),
		fmt.Sprintf("HP %d/%d   MP %d/%d", h.HP, h.MaxHP, h.MP, h.MaxMP),
		fmt.Sprintf("Weapon: %s", eq.Weapon),
		fmt.Sprintf("Armor:  %s", eq.Armor),
	}, "\n")

	actions := m.menuActions()
	var rows []string
	if len(actions) == 0 {
		rows = append(rows, dimStyle().Render("  (nothing to do right now)"))
	}
	for i, a := range actions {
		cursor, style := "  ", lipgloss.NewStyle()
		if i == m.mMenu {
			cursor = "▸ "
			style = style.Foreground(lipgloss.Color("220")).Bold(true)
		}
		rows = append(rows, style.Render(cursor+a.label))
	}

	box := panelStyle().Render(stats + "\n\n" + strings.Join(rows, "\n"))
	hint := dimStyle().Render(" ↑/↓ choose · enter select · c/esc close · q quit ")
	return lipgloss.JoinVertical(lipgloss.Left, box, hint)
}
