package combat

import (
	"errors"
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func drive(b *Battle) {
	for b.Phase == Ongoing {
		if b.IsPlayerTurn() {
			_ = b.PlayerAttack(0)
		} else {
			_ = b.Step()
		}
	}
}

func TestFasterCombatantActsFirst(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxHP: 100, Init: 5})
	swift := NewCombatant("Wolf", "🐺", SideEnemy, stats.Derived{MaxHP: 30, Init: 9})
	if NewBattle(hero, []*Combatant{swift}).IsPlayerTurn() {
		t.Error("faster enemy should act first")
	}
}

func TestPlayerActionRejectedOnEnemyTurn(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxHP: 100, Init: 1})
	swift := NewCombatant("Wolf", "🐺", SideEnemy, stats.Derived{MaxHP: 30, Init: 9})
	b := NewBattle(hero, []*Combatant{swift})
	if err := b.PlayerAttack(0); !errors.Is(err, ErrNotPlayerTurn) {
		t.Errorf("want ErrNotPlayerTurn, got %v", err)
	}
}

func TestPlayerWins(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxHP: 100, PAtk: 25, PDef: 10, Init: 7})
	rat := NewCombatant("Rat", "🐀", SideEnemy, stats.Derived{MaxHP: 12, PAtk: 4, PDef: 2, Init: 3})
	b := NewBattle(hero, []*Combatant{rat}, WithRNG(func() float64 { return 1.0 }))
	drive(b)
	if b.Phase != PlayerWon {
		t.Fatalf("want PlayerWon, got %v", b.Phase)
	}
}

func TestPlayerLoses(t *testing.T) {
	hero := NewCombatant("Hero", "🧝", SidePlayer, stats.Derived{MaxHP: 10, PAtk: 2, PDef: 1, Init: 1})
	ogre := NewCombatant("Ogre", "👹", SideEnemy, stats.Derived{MaxHP: 200, PAtk: 30, PDef: 20, Init: 9})
	b := NewBattle(hero, []*Combatant{ogre}, WithRNG(func() float64 { return 1.0 }))
	drive(b)
	if b.Phase != PlayerLost {
		t.Fatalf("want PlayerLost, got %v", b.Phase)
	}
}
