package character

import (
	"errors"
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/item"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
)

func fixtureTree(t *testing.T) *class.Tree {
	t.Helper()
	tree, err := class.NewTree("adventurer",
		class.Class{
			ID:    "adventurer",
			Name:  "Adventurer",
			Bonus: stats.Primary{STR: 4, DEX: 4, CON: 4, INT: 4, WIT: 4, MEN: 4},
			Advances: []class.Advance{
				{To: "fighter", MinLevel: 5},
				{To: "mage", MinLevel: 5},
			},
		},
		class.Class{ID: "fighter", Name: "Fighter", Bonus: stats.Primary{STR: 6, CON: 4}},
		class.Class{ID: "mage", Name: "Mage", Bonus: stats.Primary{INT: 6, MEN: 4}},
	)
	if err != nil {
		t.Fatalf("fixture tree: %v", err)
	}
	return tree
}

func TestNewStartsAtRootFullPools(t *testing.T) {
	tree := fixtureTree(t)
	c, err := New("Hero", tree)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.ClassID != "adventurer" || c.Level != 1 {
		t.Fatalf("want adventurer/level 1, got %q/%d", c.ClassID, c.Level)
	}
	d, _ := c.EffectiveStats(tree)
	if d.MaxHP != 92 || d.MaxMP != 49 || d.PAtk != 9 {
		t.Errorf("unexpected base stats: %+v", d)
	}
	if c.HP != d.MaxHP || c.MP != d.MaxMP {
		t.Errorf("pools not full: HP %d/%d MP %d/%d", c.HP, d.MaxHP, c.MP, d.MaxMP)
	}
}

func TestEquipmentAddsBonus(t *testing.T) {
	tree := fixtureTree(t)
	c, _ := New("Hero", tree)

	before, _ := c.EffectiveStats(tree)
	err := c.Equip(tree, item.Item{ID: "sword", Name: "Iron Sword", Slot: item.Weapon, Bonus: stats.Derived{PAtk: 10}})
	if err != nil {
		t.Fatalf("Equip: %v", err)
	}
	after, _ := c.EffectiveStats(tree)
	if after.PAtk != before.PAtk+10 {
		t.Errorf("weapon bonus not applied: %d -> %d", before.PAtk, after.PAtk)
	}
}

func TestUnequipClampsPools(t *testing.T) {
	tree := fixtureTree(t)
	c, _ := New("Hero", tree)

	_ = c.Equip(tree, item.Item{ID: "robe", Name: "Padded Robe", Slot: item.Armor, Bonus: stats.Derived{MaxHP: 50}})
	withArmor, _ := c.EffectiveStats(tree)
	c.HP = withArmor.MaxHP // fill up to the raised ceiling

	if err := c.Unequip(tree, item.Armor); err != nil {
		t.Fatalf("Unequip: %v", err)
	}
	base, _ := c.EffectiveStats(tree)
	if c.HP != base.MaxHP {
		t.Errorf("HP not clamped after unequip: %d, want %d", c.HP, base.MaxHP)
	}
}

func TestAddXPLevelsUpAndRestores(t *testing.T) {
	tree := fixtureTree(t)
	c, _ := New("Hero", tree)
	c.HP = 1 // simulate having taken damage

	gained, err := c.AddXP(tree, XPForNext(1))
	if err != nil {
		t.Fatalf("AddXP: %v", err)
	}
	if gained != 1 || c.Level != 2 {
		t.Fatalf("want 1 level to lvl 2, got %d to %d", gained, c.Level)
	}
	d, _ := c.EffectiveStats(tree)
	if c.HP != d.MaxHP {
		t.Errorf("level up should restore HP: %d/%d", c.HP, d.MaxHP)
	}
}

func TestAddXPMultiLevel(t *testing.T) {
	tree := fixtureTree(t)
	c, _ := New("Hero", tree)

	// 100 (lvl1->2) + 200 (lvl2->3) crossed, 50 left over.
	gained, _ := c.AddXP(tree, 350)
	if gained != 2 || c.Level != 3 || c.XP != 50 {
		t.Errorf("want 2 levels to lvl 3 with 50 xp, got %d to %d with %d", gained, c.Level, c.XP)
	}
}

func TestAdvanceRequiresLevelThenBoostsStats(t *testing.T) {
	tree := fixtureTree(t)
	c, _ := New("Hero", tree)

	if err := c.Advance(tree, "fighter"); !errors.Is(err, class.ErrNotAdvanceable) {
		t.Fatalf("advance at level 1 should fail, got %v", err)
	}

	if _, err := c.AddXP(tree, 1000); err != nil { // reach level 5
		t.Fatalf("AddXP: %v", err)
	}
	if c.Level != 5 {
		t.Fatalf("expected level 5, got %d", c.Level)
	}
	if err := c.Advance(tree, "fighter"); err != nil {
		t.Fatalf("advance at level 5 should succeed: %v", err)
	}
	if c.ClassID != "fighter" {
		t.Errorf("class not changed, got %q", c.ClassID)
	}
	d, _ := c.EffectiveStats(tree)
	if d.PAtk != 25 || d.MaxHP != 164 {
		t.Errorf("fighter stats wrong: PAtk %d, MaxHP %d", d.PAtk, d.MaxHP)
	}
}
