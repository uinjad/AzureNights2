package stats

import "testing"

func TestDerive(t *testing.T) {
	cases := []struct {
		name  string
		p     Primary
		level int
		want  Derived
	}{
		{
			name:  "fighter build leans on STR and CON",
			p:     Primary{STR: 10, DEX: 7, CON: 9, INT: 2, WIT: 3, MEN: 4},
			level: 1,
			want:  Derived{MaxHP: 132, MaxMP: 49, PAtk: 21, MAtk: 5, PDef: 10, MDef: 5, Init: 7, Crit: 7},
		},
		{
			name:  "mage build leans on INT and MEN",
			p:     Primary{STR: 3, DEX: 6, CON: 4, INT: 10, WIT: 8, MEN: 9},
			level: 1,
			want:  Derived{MaxHP: 92, MaxMP: 79, PAtk: 7, MAtk: 21, PDef: 5, MDef: 10, Init: 6, Crit: 6},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Derive(tc.p, tc.level); got != tc.want {
				t.Errorf("Derive(%+v, %d)\n got  %+v\n want %+v", tc.p, tc.level, got, tc.want)
			}
		})
	}
}
