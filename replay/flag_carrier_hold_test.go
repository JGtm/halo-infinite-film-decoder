package replay

import "testing"

// TestPositionTenueAtraversUnTrou : un corps immobile (deux points voisins a 0,15 m au plus) garde sa
// position a travers un trou de lecture borne en duree ; un corps qui a bouge, un trou trop long ou un
// trou deja couvert par la tolerance ne changent pas.
func TestPositionTenueAtraversUnTrou(t *testing.T) {
	pt := func(ms int, x float32) Point { return Point{us: uint64(ms) * 1000, X: x} }
	for _, c := range []struct {
		nom string
		pts []Point
		ms  int
		x   float32
		ok  bool
	}{
		{"trou de 957 ms, points a 5 cm : tenue", []Point{pt(1000, 10), pt(1957, 10.05)}, 1500, 10, true},
		{"trou de 957 ms, points a 2 m : non tenue", []Point{pt(1000, 10), pt(1957, 12)}, 1500, 0, false},
		{"trou sous la tolerance : point le plus proche inchange", []Point{pt(1000, 10), pt(1300, 10.05)}, 1180, 10.05, true},
		{"trou de 2001 ms, points a 5 cm : non tenue", []Point{pt(1000, 10), pt(3001, 10.05)}, 2000, 0, false},
		{"trou de 2000 ms, points a 5 cm : tenue (borne incluse)", []Point{pt(1000, 10), pt(3000, 10.05)}, 2000, 10, true},
		{"instant apres le dernier point : non tenue", []Point{pt(1000, 10), pt(1100, 10)}, 2000, 0, false},
	} {
		p, ok := pointOfXUIDAt([]Track{{Points: c.pts}}, uint64(c.ms)*1000)
		if ok != c.ok || (ok && p.X != c.x) {
			t.Errorf("%s : (x=%v, ok=%v), attendu (x=%v, ok=%v)", c.nom, p.X, ok, c.x, c.ok)
		}
	}
	// Deux vies : le trou entre le dernier point d une vie et le premier de la suivante n est jamais tenu.
	deux := []Track{{Points: []Point{pt(1000, 10)}}, {Points: []Point{pt(1957, 10)}}}
	if _, ok := pointOfXUIDAt(deux, 1_500_000); ok {
		t.Error("deux vies : le trou entre vies ne doit pas etre tenu")
	}
}
