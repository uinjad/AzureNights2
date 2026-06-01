package content

import (
	"strings"
	"testing"
)

func TestLoadAssemblesRegistry(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if reg.Classes.Root().ID != "adventurer" {
		t.Errorf("unexpected root class: %q", reg.Classes.Root().ID)
	}
	if _, ok := reg.Skills["firebolt"]; !ok {
		t.Error("firebolt skill missing")
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
	if m.Map.W != 7 || m.Map.H != 6 {
		t.Errorf("forest size wrong: %dx%d", m.Map.W, m.Map.H)
	}
	if !m.Map.Walkable(m.Spawn) {
		t.Error("spawn tile should be walkable")
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
