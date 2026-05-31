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
	if c.HP != 100 || c.MP != 40 {
		t.Errorf("pools not full: HP %d MP %d", c.HP, c.MP)
	}
	if !c.IsAlive() {
		t.Error("fresh combatant should be alive")
	}
}

func TestAttackAppliesDamage(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{PAtk: 20})
	goblin := NewCombatant("Goblin", "👹", SideEnemy, stats.Derived{MaxHP: 30, PDef: 8})

	res := hero.Attack(goblin)
	if res.Damage != 12 || goblin.HP != 18 {
		t.Fatalf("want 12 dmg leaving 18 HP, got %d dmg / %d HP", res.Damage, goblin.HP)
	}
	if res.Defeated {
		t.Error("goblin should survive")
	}
}

func TestAttackReportsDefeat(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{PAtk: 20})
	rat := NewCombatant("Rat", "🐀", SideEnemy, stats.Derived{MaxHP: 12, PDef: 8})

	res := hero.Attack(rat)
	if !res.Defeated || rat.HP != 0 {
		t.Fatalf("rat should be defeated at 0 HP, got defeated=%v HP=%d", res.Defeated, rat.HP)
	}
	if rat.IsAlive() {
		t.Error("defeated combatant must not be alive")
	}
}

func TestOrderByInitiativeStable(t *testing.T) {
	a := NewCombatant("A", "", SidePlayer, stats.Derived{Init: 5})
	b := NewCombatant("B", "", SideEnemy, stats.Derived{Init: 9})
	c := NewCombatant("C", "", SideEnemy, stats.Derived{Init: 9})
	d := NewCombatant("D", "", SidePlayer, stats.Derived{Init: 3})

	got := Order([]*Combatant{a, b, c, d})

	wantNames := []string{"B", "C", "A", "D"} // 9s keep input order, then 5, then 3
	for i, w := range wantNames {
		if got[i].Name != w {
			t.Errorf("position %d: want %s, got %s", i, w, got[i].Name)
		}
	}
}
