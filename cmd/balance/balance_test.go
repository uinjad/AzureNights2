package main

import (
	"testing"

	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
)

func TestLeafClassesAreTheNineArchetypes(t *testing.T) {
	reg, err := content.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	leaves := leafClasses(reg.Classes)
	if len(leaves) != 9 {
		t.Fatalf("want 9 terminal archetypes, got %d", len(leaves))
	}
	for _, c := range leaves {
		if len(c.Advances) != 0 {
			t.Errorf("%s should be terminal", c.ID)
		}
	}
}

func TestDuelsAreReproducibleUnderAFixedRoll(t *testing.T) {
	reg, _ := content.Load()
	roll := func() float64 { return 0.5 }
	leaves := leafClasses(reg.Classes)

	r1 := classWinRate(reg, leaves[0], leaves[1], 10, 100, roll)
	r2 := classWinRate(reg, leaves[0], leaves[1], 10, 100, roll)
	if r1 != r2 {
		t.Errorf("a fixed roll must give a reproducible win rate: %v vs %v", r1, r2)
	}
}

func TestDuelTerminates(t *testing.T) {
	reg, _ := content.Load()
	leaves := leafClasses(reg.Classes)
	a, b := leaves[0], leaves[1]
	_ = duel(
		classCombatant(reg, a, 10, combat.SidePlayer),
		classCombatant(reg, b, 10, combat.SideEnemy),
		classSkills(reg, a), classSkills(reg, b),
		reg.Factions, func() float64 { return 1.0 },
	)
}
