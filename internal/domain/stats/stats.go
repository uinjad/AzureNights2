// Package stats defines character attributes and the rules that turn the six
// primary attributes into the derived values combat actually consumes.
//
// It is a pure leaf of the domain: no dependencies, no I/O, no global state.
// Higher layers (class growth, equipment) feed into the inputs here; they never
// reach around it. That isolation is what lets us tune the formulas freely and
// cover them with fast, deterministic unit tests.
package stats

// Primary holds the six core attributes, in the spirit of L2 Interlude.
// Fighters lean on STR/DEX/CON; casters lean on INT/WIT/MEN.
type Primary struct {
	STR int
	DEX int
	CON int
	INT int
	WIT int
	MEN int
}

// Derived holds the combat-facing values computed from primary attributes and
// level. Equipment bonuses are applied by the character layer on top of these.
type Derived struct {
	MaxHP int
	MaxMP int
	PAtk  int // physical attack
	MAtk  int // magical attack
	PDef  int // physical defense
	MDef  int // magical defense
	Init  int // initiative: higher acts earlier in the turn order
	Crit  int
}

func Derive(p Primary, level int) Derived {
	crit := p.DEX
	if crit > 50 {
		crit = 50
	}
	return Derived{
		MaxHP: 50 + p.CON*8 + level*10,
		MaxMP: 20 + p.MEN*6 + level*5,
		PAtk:  p.STR*2 + level,
		MAtk:  p.INT*2 + level,
		PDef:  p.CON + level,
		MDef:  p.MEN + level,
		Init:  p.DEX,
		Crit:  crit,
	}
}
