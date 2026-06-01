// Command balance is a headless tuning harness. It loads the real game content
// and runs thousands of duels through the actual combat engine, then prints
// win-rate tables: class-vs-class, class-vs-enemy, and the isolated faction
// triangle. Because it consumes the same JSON and the same engine the game does,
// re-balancing a fork is a feedback loop, not guesswork: edit classes.json or
// factions.json, run `make balance`, read the matrix.
//
// To keep matchups fair, every duel is fought with basic attacks only (no skill
// AI), so the tables measure base-stat and faction balance. The faction triangle
// additionally swaps who strikes first across halves, cancelling initiative so
// only the multiplier shows through.
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

// leafClasses returns the terminal classes — the playable archetypes. A class is
// terminal when nothing advances from it.
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

// duel runs one fight to the end with both sides auto-attacking. It reports
// whether the player-side combatant won.
func duel(player, enemy *combat.Combatant, factions *faction.Table, roll func() float64) bool {
	b := combat.NewBattle(player, []*combat.Combatant{enemy},
		combat.WithRNG(roll), combat.WithFactions(factions))
	for b.Phase == combat.Ongoing {
		if b.IsPlayerTurn() {
			_ = b.PlayerAttack(0)
		} else {
			_ = b.Step()
		}
	}
	return b.Phase == combat.PlayerWon
}

// classWinRate is the row class's win % as the acting side against the column.
func classWinRate(reg *content.Registry, a, b class.Class, level, n int, roll func() float64) float64 {
	wins := 0
	for i := 0; i < n; i++ {
		if duel(classCombatant(reg, a, level, combat.SidePlayer),
			classCombatant(reg, b, level, combat.SideEnemy), reg.Factions, roll) {
			wins++
		}
	}
	return pct(wins, n)
}

func printClassMatrix(reg *content.Registry, leaves []class.Class, level, n int, roll func() float64) {
	fmt.Println("Class vs class — row's win % vs column (row acts on initiative ties):")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(w, " ")
	for _, c := range leaves {
		fmt.Fprintf(w, "\t%s", short(c.Name))
	}
	fmt.Fprintln(w)
	for _, a := range leaves {
		fmt.Fprintf(w, "%s", short(a.Name))
		for _, b := range leaves {
			if a.ID == b.ID {
				fmt.Fprint(w, "\t  —")
				continue
			}
			fmt.Fprintf(w, "\t%3.0f", classWinRate(reg, a, b, level, n, roll))
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

func printMobTable(reg *content.Registry, leaves []class.Class, level, n int, roll func() float64) {
	fmt.Println("Class vs enemy — win %:")
	enemies := sortedEnemies(reg)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(w, " ")
	for _, e := range enemies {
		fmt.Fprintf(w, "\t%s", e.Name)
	}
	fmt.Fprintln(w)
	for _, a := range leaves {
		fmt.Fprintf(w, "%s", short(a.Name))
		for _, e := range enemies {
			wins := 0
			for i := 0; i < n; i++ {
				if duel(classCombatant(reg, a, level, combat.SidePlayer),
					enemyCombatant(e, combat.SideEnemy), reg.Factions, roll) {
					wins++
				}
			}
			fmt.Fprintf(w, "\t%3.0f", pct(wins, n))
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// printFactionTriangle isolates the faction multiplier: identical stat blocks,
// only allegiance differs, and each pair is fought twice with the strike order
// swapped so initiative cancels out.
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
			if duel(ap, be, reg.Factions, roll) {
				wins++
			}
			// swap strike order: b acts first; a (now the enemy) wins if b loses.
			bp := combat.NewCombatant("b", "", combat.SidePlayer, base)
			bp.Faction = p.b
			ae := combat.NewCombatant("a", "", combat.SideEnemy, base)
			ae.Faction = p.a
			if !duel(bp, ae, reg.Factions, roll) {
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

// short abbreviates "Solar Warrior" -> "Sol-War" to keep the matrix narrow.
func short(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 2 && len(parts[0]) >= 3 && len(parts[1]) >= 3 {
		return parts[0][:3] + "-" + parts[1][:3]
	}
	return name
}
