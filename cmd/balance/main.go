// Command balance is a headless tuning harness. It loads the real game content
// and runs thousands of duels through the actual combat engine, then prints
// win-rate tables: class-vs-class, class-vs-enemy, and the isolated faction
// triangle. Both sides cast their best usable skill each turn (falling back to a
// basic attack), so a Mage is judged by its Arcane Bolt, not its fists.
//
// The faction triangle uses neutral, identical stat blocks and swaps who strikes
// first across halves, cancelling initiative so only the multiplier shows.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/faction"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func main() {
	level := flag.Int("level", 10, "character level for the simulation")
	n := flag.Int("duels", 2000, "duels per matchup")
	seed := flag.Int64("seed", 1, "RNG seed (fixed for reproducible runs)")
	flag.Parse()

	reg, err := content.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load content:", err)
		os.Exit(1)
	}
	roll := rand.New(rand.NewSource(*seed)).Float64
	leaves := leafClasses(reg.Classes)

	fmt.Printf("AzureNights balance — level %d · %d duels/matchup · seed %d\n\n", *level, *n, *seed)
	printClassMatrix(reg, leaves, *level, *n, roll)
	fmt.Println()
	printMobTable(reg, leaves, *level, *n, roll)
	fmt.Println()
	printFactionTriangle(reg, *level, *n, roll)
}

func leafClasses(tree *class.Tree) []class.Class {
	var out []class.Class
	for _, c := range tree.All() {
		if len(c.Advances) == 0 {
			out = append(out, c)
		}
	}
	return out
}

func classCombatant(reg *content.Registry, c class.Class, level int, side combat.Side) *combat.Combatant {
	prim, _ := reg.Classes.CumulativePrimary(c.ID)
	cb := combat.NewCombatant(c.Name, "", side, stats.Derive(prim, level))
	cb.Faction = c.Faction
	return cb
}

func enemyCombatant(e content.EnemyDef, side combat.Side) *combat.Combatant {
	cb := combat.NewCombatant(e.Name, e.Emoji, side, e.Stats)
	cb.Faction = e.Faction
	return cb
}

// classSkills gathers a class's skills along its advancement path.
func classSkills(reg *content.Registry, c class.Class) []combat.Skill {
	chain, ok := reg.Classes.Path(c.ID)
	if !ok {
		return nil
	}
	var out []combat.Skill
	for _, node := range chain {
		for _, id := range node.Skills {
			if sk, ok := reg.Skills[id]; ok {
				out = append(out, sk)
			}
		}
	}
	return out
}

// duel runs one fight to the end. The player casts its best usable skill each
// turn; the enemy runs SkillAI. Reports whether the player-side combatant won.
func duel(player, enemy *combat.Combatant, playerSkills, enemySkills []combat.Skill, factions *faction.Table, roll func() float64) bool {
	b := combat.NewBattle(player, []*combat.Combatant{enemy},
		combat.WithRNG(roll), combat.WithFactions(factions))
	b.AI = combat.SkillAI(enemySkills)
	for b.Phase == combat.Ongoing {
		if b.IsPlayerTurn() {
			acted := false
			for _, sk := range playerSkills {
				if b.Player().CanUse(sk) {
					if b.PlayerUseSkill(sk, 0) == nil {
						acted = true
						break
					}
				}
			}
			if !acted {
				_ = b.PlayerAttack(0)
			}
		} else {
			_ = b.Step()
		}
	}
	return b.Phase == combat.PlayerWon
}

func classWinRate(reg *content.Registry, a, b class.Class, level, n int, roll func() float64) float64 {
	as, bs := classSkills(reg, a), classSkills(reg, b)
	wins := 0
	for i := 0; i < n; i++ {
		if duel(classCombatant(reg, a, level, combat.SidePlayer),
			classCombatant(reg, b, level, combat.SideEnemy), as, bs, reg.Factions, roll) {
			wins++
		}
	}
	return pct(wins, n)
}

// printMatrix renders a tabwriter-aligned table: a header row of column
// labels, then one row per row label with cell(i, j) supplying each entry.
// It is the shared shape behind both win-rate tables below.
func printMatrix(title string, rows, cols []string, cell func(i, j int) string) {
	fmt.Println(title)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(w, " ")
	for _, c := range cols {
		fmt.Fprintf(w, "\t%s", c)
	}
	fmt.Fprintln(w)
	for i, r := range rows {
		fmt.Fprintf(w, "%s", r)
		for j := range cols {
			fmt.Fprintf(w, "\t%s", cell(i, j))
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

func printClassMatrix(reg *content.Registry, leaves []class.Class, level, n int, roll func() float64) {
	names := make([]string, len(leaves))
	for i, c := range leaves {
		names[i] = short(c.Name)
	}
	printMatrix("Class vs class — row's win % vs column (both cast skills):", names, names,
		func(i, j int) string {
			if leaves[i].ID == leaves[j].ID {
				return "  —"
			}
			return fmt.Sprintf("%3.0f", classWinRate(reg, leaves[i], leaves[j], level, n, roll))
		})
}

func printMobTable(reg *content.Registry, leaves []class.Class, level, n int, roll func() float64) {
	enemies := sortedEnemies(reg)
	rows := make([]string, len(leaves))
	skills := make([][]combat.Skill, len(leaves))
	cols := make([]string, len(enemies))
	for i, c := range leaves {
		rows[i], skills[i] = short(c.Name), classSkills(reg, c)
	}
	for j, e := range enemies {
		cols[j] = e.Name
	}
	printMatrix("Class vs enemy — win % (raw enemy tier, no level scaling):", rows, cols,
		func(i, j int) string {
			wins := 0
			for k := 0; k < n; k++ {
				if duel(classCombatant(reg, leaves[i], level, combat.SidePlayer),
					enemyCombatant(enemies[j], combat.SideEnemy), skills[i], nil, reg.Factions, roll) {
					wins++
				}
			}
			return fmt.Sprintf("%3.0f", pct(wins, n))
		})
}

func printFactionTriangle(reg *content.Registry, level, n int, roll func() float64) {
	fmt.Println("Faction triangle — identical stats, allegiance only (attacker win %):")
	base := stats.Derive(stats.Primary{STR: 8, DEX: 8, CON: 8, INT: 8, WIT: 8, MEN: 8}, level)
	pairs := []struct{ a, b faction.ID }{
		{"solar", "illumite"}, {"illumite", "lawful"}, {"lawful", "solar"},
	}
	for _, p := range pairs {
		wins := 0
		for i := 0; i < n; i++ {
			ap := combat.NewCombatant("a", "", combat.SidePlayer, base)
			ap.Faction = p.a
			be := combat.NewCombatant("b", "", combat.SideEnemy, base)
			be.Faction = p.b
			if duel(ap, be, nil, nil, reg.Factions, roll) {
				wins++
			}
			bp := combat.NewCombatant("b", "", combat.SidePlayer, base)
			bp.Faction = p.b
			ae := combat.NewCombatant("a", "", combat.SideEnemy, base)
			ae.Faction = p.a
			if !duel(bp, ae, nil, nil, reg.Factions, roll) {
				wins++
			}
		}
		fmt.Printf("  %-15s > %-15s : %4.1f%%\n",
			reg.Factions.Name(p.a), reg.Factions.Name(p.b), pct(wins, 2*n))
	}
}

func sortedEnemies(reg *content.Registry) []content.EnemyDef {
	out := make([]content.EnemyDef, 0, len(reg.Enemies))
	for _, e := range reg.Enemies {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func pct(part, total int) float64 { return float64(part) / float64(total) * 100 }

func short(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 2 && len(parts[0]) >= 3 && len(parts[1]) >= 3 {
		return parts[0][:3] + "-" + parts[1][:3]
	}
	return name
}
