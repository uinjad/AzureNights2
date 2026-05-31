package world

import "fmt"

// TimeOfDay is the current phase of the day/night cycle.
type TimeOfDay int

const (
	Dawn TimeOfDay = iota
	Day
	Dusk
	Night
)

func (t TimeOfDay) String() string {
	return [...]string{"dawn", "day", "dusk", "night"}[t%4]
}

// Weather is the current sky condition.
type Weather int

const (
	Clear Weather = iota
	Rain
	Fog
)

func (w Weather) String() string {
	return [...]string{"clear", "rain", "fog"}[w%3]
}

// TicksPerPhase is how many world ticks each part of the day lasts.
const TicksPerPhase = 30

// weatherChangeChance is the per-tick probability that the weather shifts.
const weatherChangeChance = 0.1

// Clock tracks elapsed time, the day phase, and the weather. Time advances on a
// fixed schedule; weather shifts probabilistically using an injected roll, which
// keeps the clock fully deterministic under test.
type Clock struct {
	Tick      int
	TimeOfDay TimeOfDay
	Weather   Weather
}

// Advance moves the clock forward one tick and returns human-readable notes for
// anything that changed, ready for the UI log. roll must return a value in
// [0,1): pass rand.Float64 in production, a stub in tests.
func (c *Clock) Advance(roll func() float64) []string {
	c.Tick++
	var notes []string

	if c.Tick%TicksPerPhase == 0 {
		c.TimeOfDay = (c.TimeOfDay + 1) % 4
		notes = append(notes, fmt.Sprintf("The light shifts: it is now %s.", c.TimeOfDay))
	}
	if roll() < weatherChangeChance {
		c.Weather = (c.Weather + 1) % 3
		notes = append(notes, fmt.Sprintf("The weather turns to %s.", c.Weather))
	}
	return notes
}

// EnemyPowerBonus is the MVP's single "living world affects combat" hook: at
// night, enemies hit a little harder. The app applies it to enemy attack stats
// when a battle starts, so world never needs to import combat.
func (c Clock) EnemyPowerBonus() int {
	if c.TimeOfDay == Night {
		return 3
	}
	return 0
}
