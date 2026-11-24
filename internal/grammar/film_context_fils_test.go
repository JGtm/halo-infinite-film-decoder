package grammar

import "testing"

func TestFilsParDefautVautUn(t *testing.T) {
	if n := NewFilmContext(nil).Fils(); n != 1 {
		t.Fatalf("NewFilmContext : %d fils, 1 attendu", n)
	}
	if n := NewFilmContextForMap(nil, nil, nil).Fils(); n != 1 {
		t.Fatalf("NewFilmContextForMap : %d fils, 1 attendu", n)
	}
	if n := (&FilmContext{}).Fils(); n != 1 {
		t.Fatalf("contexte a zero : %d fils, 1 attendu", n)
	}
}

func TestPoserFilsBorneAuMoinsUn(t *testing.T) {
	c := NewFilmContext(nil)
	for _, cas := range []struct{ pose, attendu int }{{4, 4}, {1, 1}, {0, 1}, {-5, 1}, {8, 8}} {
		c.PoserFils(cas.pose)
		if n := c.Fils(); n != cas.attendu {
			t.Fatalf("PoserFils(%d) : Fils() = %d, %d attendu", cas.pose, n, cas.attendu)
		}
	}
}
