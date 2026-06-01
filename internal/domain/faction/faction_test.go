package faction

import "testing"

func table(t *testing.T) *Table {
	t.Helper()
	tbl, err := NewTable(1.08, 0.95,
		map[ID]string{"solar": "Solar Order", "illumite": "Illumite League", "lawful": "Lawful Union"},
		map[ID]ID{"solar": "illumite", "illumite": "lawful", "lawful": "solar"},
	)
	if err != nil {
		t.Fatalf("NewTable: %v", err)
	}
	return tbl
}

func TestCycleRelations(t *testing.T) {
	tbl := table(t)
	cases := []struct {
		a, b ID
		want Relation
	}{
		{"solar", "illumite", Advantage},
		{"illumite", "solar", Disadvantage},
		{"illumite", "lawful", Advantage},
		{"lawful", "solar", Advantage},
		{"solar", "lawful", Disadvantage},
		{"solar", "solar", Even},
		{Neutral, "solar", Even},
		{"solar", Neutral, Even},
	}
	for _, c := range cases {
		if got := tbl.Relation(c.a, c.b); got != c.want {
			t.Errorf("Relation(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestDamageMultiplier(t *testing.T) {
	tbl := table(t)
	if got := tbl.DamageMultiplier("solar", "illumite"); got != 1.08 {
		t.Errorf("advantage = %v, want 1.08", got)
	}
	if got := tbl.DamageMultiplier("illumite", "solar"); got != 0.95 {
		t.Errorf("disadvantage = %v, want 0.95", got)
	}
	if got := tbl.DamageMultiplier("solar", "solar"); got != 1.0 {
		t.Errorf("even = %v, want 1.0", got)
	}
}

func TestRejectsUnknownBeats(t *testing.T) {
	if _, err := NewTable(1.0, 1.0, map[ID]string{"solar": "Solar"}, map[ID]ID{"solar": "ghost"}); err == nil {
		t.Error("beats targeting an unknown faction should error")
	}
}
