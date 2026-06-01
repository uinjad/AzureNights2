package storage

import (
	"path/filepath"
	"testing"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/character"
	"github.com/uinjad/AzureNights2/internal/domain/item"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

func TestExistsTracksTheFile(t *testing.T) {
	repo := NewFileRepo(filepath.Join(t.TempDir(), "save.json"))
	if repo.Exists() {
		t.Error("no save should exist yet")
	}

	reg, err := content.Load()
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	hero, _ := character.New("Aria", reg.Classes)
	if err := repo.Save(&app.Snapshot{Hero: hero, MapID: "forest"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !repo.Exists() {
		t.Error("save should exist after writing")
	}
}

func TestSaveLoadPreservesTheWholeChain(t *testing.T) {
	repo := NewFileRepo(filepath.Join(t.TempDir(), "save.json"))

	reg, _ := content.Load()
	hero, _ := character.New("Aria", reg.Classes)
	hero.Gold = 250
	hero.HP = 40
	if err := hero.Equip(reg.Classes, reg.Items["iron_sword"]); err != nil {
		t.Fatalf("Equip: %v", err)
	}

	in := &app.Snapshot{
		Hero:      hero,
		MapID:     "forest",
		PlayerPos: world.Point{X: 5, Y: 1},
		Clock:     world.Clock{Tick: 90, TimeOfDay: world.Dusk, Weather: world.Fog},
		Spawns:    []app.Spawn{{Pos: world.Point{X: 4, Y: 4}, DefID: "skeleton"}},
	}
	if err := repo.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if out.Hero.Name != "Aria" || out.Hero.Gold != 250 || out.Hero.HP != 40 {
		t.Errorf("hero core fields not preserved: %+v", out.Hero)
	}
	if out.Hero.ClassID != "adventurer" {
		t.Errorf("class not preserved: %q", out.Hero.ClassID)
	}
	if _, ok := out.Hero.Equipment[item.Weapon]; !ok {
		t.Error("equipped weapon not preserved")
	}
	if out.PlayerPos != (world.Point{X: 5, Y: 1}) {
		t.Errorf("position not preserved: %+v", out.PlayerPos)
	}
	if out.Clock.Tick != 90 || out.Clock.TimeOfDay != world.Dusk || out.Clock.Weather != world.Fog {
		t.Errorf("clock not preserved: %+v", out.Clock)
	}
	if len(out.Spawns) != 1 || out.Spawns[0].DefID != "skeleton" {
		t.Errorf("spawns not preserved: %+v", out.Spawns)
	}

	// The real proof: the class+level+equipment pipeline still computes after a
	// full disk round-trip.
	d, err := out.Hero.EffectiveStats(reg.Classes)
	if err != nil {
		t.Fatalf("EffectiveStats after load: %v", err)
	}
	if d.PAtk != 19 { // 9 base (adventurer, lvl 1) + 10 from the iron sword
		t.Errorf("derived stats wrong after load, PAtk = %d", d.PAtk)
	}
}

func TestPlugsIntoSessionThroughPort(t *testing.T) {
	repo := NewFileRepo(filepath.Join(t.TempDir(), "save.json"))
	reg, _ := content.Load()

	s := app.New(reg, repo)
	if err := s.NewGame("Aria"); err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	s.Hero.Gold = 77
	if err := s.Save(); err != nil {
		t.Fatalf("Save via session: %v", err)
	}

	s2 := app.New(reg, repo)
	if err := s2.LoadGame(); err != nil {
		t.Fatalf("LoadGame via session: %v", err)
	}
	if s2.Hero.Gold != 77 {
		t.Errorf("gold not restored through the port: %d", s2.Hero.Gold)
	}
}
