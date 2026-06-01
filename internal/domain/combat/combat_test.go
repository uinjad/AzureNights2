package combat

import (
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func TestPhysicalDamageFloorsAtOne(t *testing.T) {
	if got := PhysicalDamage(5, 10); got != 1 {
		t.Errorf("weak attack should floor at 1, got %d", got)
	}
	if got := PhysicalDamage(20, 8); got != 12 {
		t.Errorf("want 12, got %d", got)
	}
}

func TestNewCombatantStartsFull(t *testing.T) {
	c := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxHP: 100, MaxMP: 40})
	if c.HP != 100 || c.MP != 40 || !c.IsAlive() {
		t.Errorf("fresh combatant should start full and alive")
	}
}

func TestHitAppliesVarianceNoCrit(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{PAtk: 20})
	goblin := NewCombatant("Goblin", "👹", SideEnemy, stats.Derived{MaxHP: 100, PDef: 8})
	b := NewBattle(hero, []*Combatant{goblin}, WithRNG(func() float64 { return 1.0 }))

	b.hit(hero, goblin, Physical, 0, "") // base 12 × 1.15 = 13, rng 1.0 -> no crit
	if goblin.HP != 87 {
		t.Errorf("want 87 HP after a 13 hit, got %d", goblin.HP)
	}
}

func TestHitCritsOnLowRoll(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{PAtk: 20, Crit: 50})
	goblin := NewCombatant("Goblin", "👹", SideEnemy, stats.Derived{MaxHP: 200, PDef: 8})
	b := NewBattle(hero, []*Combatant{goblin}, WithRNG(func() float64 { return 0.0 }))

	b.hit(hero, goblin, Physical, 0, "") // 0.0 -> variance 0.85, crit fires ×2; 12×0.85×2=20
	if goblin.HP != 180 {
		t.Errorf("crit should deal 20, leaving 180, got %d", goblin.HP)
	}
}

func TestOrderByInitiativeStable(t *testing.T) {
	a := NewCombatant("A", "", SidePlayer, stats.Derived{Init: 5})
	bb := NewCombatant("B", "", SideEnemy, stats.Derived{Init: 9})
	c := NewCombatant("C", "", SideEnemy, stats.Derived{Init: 9})
	d := NewCombatant("D", "", SidePlayer, stats.Derived{Init: 3})

	got := Order([]*Combatant{a, bb, c, d})
	for i, w := range []string{"B", "C", "A", "D"} {
		if got[i].Name != w {
			t.Errorf("pos %d: want %s, got %s", i, w, got[i].Name)
		}
	}
}
