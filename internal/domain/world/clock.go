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

// TicksPerPhase is how many world ticks each part of the day lasts. With a
// one-second tick that is one minute per phase — slow enough to feel ambient.
const TicksPerPhase = 60

// weatherChangeChance is the per-tick probability that the weather shifts. Kept
// low so the sky doesn't flicker every few seconds.
const weatherChangeChance = 0.02

// Clock tracks elapsed time, the day phase, and the weather. Time advances on a
// fixed schedule; weather shifts probabilistically using an injected roll.
type Clock struct {
	Tick      int
	TimeOfDay TimeOfDay
	Weather   Weather
}

// Advance moves the clock forward one tick and returns human-readable notes for
// anything that changed. roll must return a value in [0,1).
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

// EnemyPowerBonus is the living-world combat hook: at night enemies hit harder.
func (c Clock) EnemyPowerBonus() int {
	if c.TimeOfDay == Night {
		return 3
	}
	return 0
}
