package content

import (
	"strings"
	"testing"

	"github.com/uinjad/AzureNights2/internal/domain/faction"
)

func TestLoadAssemblesRegistry(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if reg.Classes.Root().ID != "adventurer" {
		t.Errorf("unexpected root class: %q", reg.Classes.Root().ID)
	}
	if _, ok := reg.Skills["arcane_bolt"]; !ok {
		t.Error("arcane_bolt skill missing")
	}
	if reg.Factions.Relation("solar", "illumite") != faction.Advantage {
		t.Error("solar should beat illumite")
	}
	if _, ok := reg.Items["iron_sword"]; !ok {
		t.Error("iron_sword item missing")
	}
	if _, ok := reg.Enemies["goblin"]; !ok {
		t.Error("goblin enemy missing")
	}
	m, ok := reg.Maps["forest"]
	if !ok {
		t.Fatal("forest map missing")
	}
	if m.Map.W != 10 || m.Map.H != 7 {
		t.Errorf("forest size wrong: %dx%d", m.Map.W, m.Map.H)
	}
	if !m.Map.Walkable(m.Spawn) {
		t.Error("spawn tile should be walkable")
	}
	if _, ok := reg.Maps["cavern"]; !ok {
		t.Error("cavern map missing")
	}
	if len(m.Portals) == 0 || m.Portals[0].ToMap != "cavern" {
		t.Errorf("forest should have a portal to the cavern, got %+v", m.Portals)
	}
	if len(m.Rests) == 0 {
		t.Error("forest should have a campfire")
	}
	if reg.Quests == nil || len(reg.Quests.All()) < 2 {
		t.Fatalf("expected at least two quests, got %v", reg.Quests)
	}
	if q, ok := reg.Quests.Get("cull_the_goblins"); !ok || q.Objectives[0].Target != "goblin" {
		t.Errorf("cull quest should target goblins, got %+v", q)
	}
}

func TestEnumParsersRejectUnknown(t *testing.T) {
	if _, err := parseDamageKind("psychic"); err == nil {
		t.Error("unknown damage kind should error")
	}
	if _, err := parseSlot("ring"); err == nil {
		t.Error("unknown slot should error")
	}
	if _, err := parseTileKind("lava"); err == nil {
		t.Error("unknown tile kind should error")
	}
}

func TestParseMapRejectsRaggedRows(t *testing.T) {
	bad := []byte(`{"name":"Bad","spawn":{"x":0,"y":0},
		"legend":{".":{"kind":"grass","emoji":"🌿","walkable":true}},
		"rows":["..",".",".."]}`)
	if _, err := parseMap(bad); err == nil || !strings.Contains(err.Error(), "width") {
		t.Errorf("ragged rows should fail on width, got %v", err)
	}
}

func TestParseMapRejectsUnknownLegend(t *testing.T) {
	bad := []byte(`{"name":"Bad","spawn":{"x":0,"y":0},
		"legend":{".":{"kind":"grass","emoji":"🌿","walkable":true}},
		"rows":["X"]}`)
	if _, err := parseMap(bad); err == nil || !strings.Contains(err.Error(), "legend") {
		t.Errorf("unknown legend char should fail, got %v", err)
	}
}
