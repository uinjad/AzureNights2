package world

import "testing"

func neverShifts() float64  { return 1.0 } // above the change chance
func alwaysShifts() float64 { return 0.0 } // below it

func TestTimeAdvancesOnSchedule(t *testing.T) {
	var c Clock // tick 0, dawn, clear

	for i := 0; i < TicksPerPhase-1; i++ {
		if notes := c.Advance(neverShifts); len(notes) != 0 {
			t.Fatalf("no change expected at tick %d, got %v", c.Tick, notes)
		}
	}
	if c.TimeOfDay != Dawn {
		t.Fatalf("still dawn before the threshold, got %s", c.TimeOfDay)
	}
	if notes := c.Advance(neverShifts); c.TimeOfDay != Day || len(notes) == 0 {
		t.Errorf("should advance to day with a note, got %s notes=%v", c.TimeOfDay, notes)
	}
}

func TestWeatherCyclesOnRoll(t *testing.T) {
	var c Clock
	c.Advance(alwaysShifts)
	if c.Weather != Rain {
		t.Errorf("clear -> rain, got %s", c.Weather)
	}
	c.Advance(alwaysShifts)
	if c.Weather != Fog {
		t.Errorf("rain -> fog, got %s", c.Weather)
	}
	c.Advance(alwaysShifts)
	if c.Weather != Clear {
		t.Errorf("fog -> clear, got %s", c.Weather)
	}
}

func TestNightStrengthensEnemies(t *testing.T) {
	if (Clock{TimeOfDay: Day}).EnemyPowerBonus() != 0 {
		t.Error("daytime should give no bonus")
	}
	if (Clock{TimeOfDay: Night}).EnemyPowerBonus() <= 0 {
		t.Error("night should strengthen enemies")
	}
}
