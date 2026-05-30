package class

import (
	"errors"
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

// fixtureTree builds the minimal MVP tree: a base class that branches once.
func fixtureTree(t *testing.T) *Tree {
	t.Helper()
	tree, err := NewTree("adventurer",
		Class{
			ID:    "adventurer",
			Name:  "Adventurer",
			Bonus: stats.Primary{STR: 4, DEX: 4, CON: 4, INT: 4, WIT: 4, MEN: 4},
			Advances: []Advance{
				{To: "fighter", MinLevel: 5},
				{To: "mage", MinLevel: 5},
			},
		},
		Class{ID: "fighter", Name: "Fighter", Bonus: stats.Primary{STR: 6, CON: 4}, Skills: []string{"power_strike"}},
		Class{ID: "mage", Name: "Mage", Bonus: stats.Primary{INT: 6, MEN: 4}, Skills: []string{"firebolt"}},
	)
	if err != nil {
		t.Fatalf("building fixture tree: %v", err)
	}
	return tree
}

func TestNewTreeRejectsDanglingAdvance(t *testing.T) {
	_, err := NewTree("adventurer",
		Class{ID: "adventurer", Advances: []Advance{{To: "ghost", MinLevel: 1}}},
	)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("want ErrUnknownClass, got %v", err)
	}
}

func TestOptionsRespectLevelGate(t *testing.T) {
	tree := fixtureTree(t)
	if opts := tree.Options("adventurer", 4); len(opts) != 0 {
		t.Errorf("level 4 should offer no advancements, got %d", len(opts))
	}
	if opts := tree.Options("adventurer", 5); len(opts) != 2 {
		t.Errorf("level 5 should offer 2 advancements, got %d", len(opts))
	}
}

func TestAdvance(t *testing.T) {
	tree := fixtureTree(t)

	got, err := tree.Advance("adventurer", "fighter", 5)
	if err != nil {
		t.Fatalf("valid advance failed: %v", err)
	}
	if got.ID != "fighter" {
		t.Errorf("want fighter, got %q", got.ID)
	}
	if _, err := tree.Advance("adventurer", "fighter", 4); !errors.Is(err, ErrNotAdvanceable) {
		t.Errorf("under-level advance should fail with ErrNotAdvanceable, got %v", err)
	}
}

func TestCumulativePrimary(t *testing.T) {
	tree := fixtureTree(t)
	got, ok := tree.CumulativePrimary("fighter")
	if !ok {
		t.Fatal("path to fighter not found")
	}
	// adventurer base (4 each) + fighter bonus (STR+6, CON+4)
	want := stats.Primary{STR: 10, DEX: 4, CON: 8, INT: 4, WIT: 4, MEN: 4}
	if got != want {
		t.Errorf("CumulativePrimary(fighter)\n got  %+v\n want %+v", got, want)
	}
}