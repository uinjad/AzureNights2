package combat

import (
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func TestSkillConsumesMPAndCooldown(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxMP: 20, MAtk: 10})
	dummy := NewCombatant("Dummy", "", SideEnemy, stats.Derived{MaxHP: 9999, MDef: 5})
	fb := Skill{ID: "fb", Name: "Fireball", Kind: Magical, MPCost: 8, Cooldown: 2, Power: 15}
	b := NewBattle(hero, []*Combatant{dummy}, WithRNG(func() float64 { return 1.0 }))

	if !hero.CanUse(fb) {
		t.Fatal("should be usable initially")
	}
	b.resolveSkill(hero, fb, dummy)
	if hero.MP != 12 {
		t.Errorf("MP should drop to 12, got %d", hero.MP)
	}
	if hero.CanUse(fb) {
		t.Error("should be on cooldown right after use")
	}
	hero.tickCooldowns()
	if hero.CanUse(fb) {
		t.Error("still on cooldown after one tick")
	}
	hero.tickCooldowns()
	if !hero.CanUse(fb) {
		t.Error("ready after two ticks")
	}
}

func TestCanUseFalseWhenStarvedOfMP(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxMP: 5})
	if hero.CanUse(Skill{ID: "nuke", MPCost: 50}) {
		t.Error("not enough MP should make a skill unusable")
	}
}
