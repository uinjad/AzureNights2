package combat

import (
	"errors"
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func TestCanUseRespectsMPAndCooldown(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxMP: 20, MAtk: 10})
	dummy := NewCombatant("Dummy", "", SideEnemy, stats.Derived{MaxHP: 500, MDef: 5})
	fireball := Skill{ID: "fireball", Name: "Fireball", Kind: Magical, MPCost: 8, Cooldown: 2, Power: 15}

	if !hero.CanUse(fireball) {
		t.Fatal("should be usable initially")
	}
	res, err := hero.UseSkill(fireball, dummy)
	if err != nil {
		t.Fatalf("UseSkill: %v", err)
	}
	if res.Damage != 20 || hero.MP != 12 { // 10 MAtk + 15 power - 5 MDef
		t.Fatalf("want 20 dmg / 12 MP, got %d dmg / %d MP", res.Damage, hero.MP)
	}
	if hero.CanUse(fireball) {
		t.Error("should be on cooldown right after use")
	}

	hero.tickCooldowns() // 2 -> 1
	if hero.CanUse(fireball) {
		t.Error("still on cooldown after one tick")
	}
	hero.tickCooldowns() // 1 -> 0
	if !hero.CanUse(fireball) {
		t.Error("should be ready after two ticks")
	}
}

func TestUseSkillRejectsWhenStarvedOfMP(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxMP: 5})
	nuke := Skill{ID: "nuke", MPCost: 50}

	if hero.CanUse(nuke) {
		t.Fatal("not enough MP to use")
	}
	if _, err := hero.UseSkill(nuke, hero); !errors.Is(err, ErrSkillUnavailable) {
		t.Errorf("want ErrSkillUnavailable, got %v", err)
	}
}
