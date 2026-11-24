package objectives

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// seat_cuts_test.go — UNE SUITE PAR OCCUPANT D'UN SIEGE : un siege qui change de joueur en cours de manche
// rend les actions des deux joueurs, et seul un vrai changement d'occupant coupe la suite.
//
// ROUGE OBSERVE AVANT LA COUPURE : sur le siege a deux occupants, la plus longue suite non decroissante ne
// gardait que celle de l'arrivant (suites de meme longueur, la derniere l'emporte) et les deux vols du
// partant disparaissaient.

// recVols : un enregistrement de siege dont les compteurs de base sont (frags, morts, assistances) et dont le
// compteur de vols de drapeau (composant 24, valeur A) vaut `vols`.
func recVols(timeMS, slot, round, k, d, a, vols int) types.StatRecord {
	r := recKDA(timeMS, slot, round, k, d, a)
	r.Comps[24] = types.StatValue{A: int64(vols)}
	return r
}

// instantsDesVols rend les instants des vols de drapeau nommes d'un film, dans l'ordre, pour un siege.
func instantsDesVols(recs []types.StatRecord, slot int) []int {
	var out []int
	for _, e := range NamedEventsFrom(recs, ObjectiveTypeFlag, nil) {
		if e.Stat == StatFlagSteals && e.Slot == slot {
			out = append(out, e.TimeMS)
		}
	}
	return out
}

// siegeADeuxOccupants : le siege 12 est tenu par un partant (deux vols, a 10 000 et 20 000 ms) qui quitte a
// 40 000 ms, la retombee de tous les compteurs, puis par un arrivant (un vol, a 50 000 ms). Les deux suites ont
// deux emissions : a longueur egale, la plus longue suite du siege garde la derniere.
func siegeADeuxOccupants() []types.StatRecord {
	return []types.StatRecord{
		recVols(10000, 12, 0, 0, 1, 0, 1), recVols(20000, 12, 0, 1, 1, 0, 2), recVols(30000, 12, 0, 1, 2, 0, 2),
		recVols(40000, 12, 0, 0, 0, 0, 0), // retombee : le partant quitte, l'arrivant repart de zero
		recVols(50000, 12, 0, 0, 1, 0, 1), recVols(60000, 12, 0, 0, 2, 0, 1),
	}
}

// TestSiegeADeuxOccupantsRendDeuxSuites : les vols du partant et celui de l'arrivant sont tous rendus, chacun a
// son instant, et le pont d'identite les nomme chacun par l'occupant de SON instant.
func TestSiegeADeuxOccupantsRendDeuxSuites(t *testing.T) {
	recs := siegeADeuxOccupants()
	if got, want := instantsDesVols(recs, 12), []int{10000, 20000, 50000}; !slices.Equal(got, want) {
		t.Fatalf("vols du siege 12 = %v, attendu %v (deux du partant, un de l'arrivant)", got, want)
	}
	// La frontiere est celle des occupations du pont : meme instant, memes occupants.
	if got := seatCutsOf(recs).of(0, 12); !slices.Equal(got, []int{40000}) {
		t.Fatalf("frontieres du siege 12 = %v, attendu [40000]", got)
	}
	occ := ResolveRoundIdentity(recs, nil, nil).Occupations(0, 12)
	if len(occ) != 2 || occ[0].ToMS != 40000 || occ[1].FromMS != 40000 {
		t.Fatalf("occupations du siege 12 = %+v, attendu deux occupations bornees a 40 000 ms", occ)
	}
}

// TestSiegeADeuxOccupantsLArrivantPlusLong : l'arrivant a la suite la plus longue (la plus longue suite du
// siege ne gardait alors que lui) ; les vols du partant sont rendus de meme.
func TestSiegeADeuxOccupantsLArrivantPlusLong(t *testing.T) {
	recs := []types.StatRecord{
		recVols(10000, 12, 0, 0, 1, 0, 1), recVols(20000, 12, 0, 1, 1, 0, 2), recVols(30000, 12, 0, 1, 2, 0, 2),
		recVols(40000, 12, 0, 0, 0, 0, 0),
		recVols(50000, 12, 0, 0, 1, 0, 1), recVols(60000, 12, 0, 0, 2, 0, 2), recVols(70000, 12, 0, 1, 2, 0, 3),
	}
	if got, want := instantsDesVols(recs, 12), []int{10000, 20000, 50000, 60000, 70000}; !slices.Equal(got, want) {
		t.Fatalf("vols du siege 12 = %v, attendu %v", got, want)
	}
}

// TestZeroIsoleNeCoupePas : un siege dont UN SEUL occupant voit passer un enregistrement a zero, puis reprend
// son comptage, garde sa suite d'avant : le zero n'est pas un changement d'occupant (l'emission suivante
// prolonge le comptage), et rien n'est coupe.
//
// MUTATION : couper a tout enregistrement tout a zero (la simulation d'origine) donnerait trois vols, le
// premier etant compte deux fois.
func TestZeroIsoleNeCoupePas(t *testing.T) {
	recs := []types.StatRecord{
		recVols(10000, 12, 0, 1, 0, 0, 1),
		recVols(20000, 12, 0, 0, 0, 0, 0), // une ligne videe, sans depart
		recVols(30000, 12, 0, 1, 0, 0, 1),
		recVols(40000, 12, 0, 2, 0, 0, 2),
		recVols(50000, 12, 0, 3, 0, 0, 2),
	}
	if cuts := seatCutsOf(recs).of(0, 12); cuts != nil {
		t.Fatalf("frontieres = %v, attendu aucune : le comptage reprend ou il en etait", cuts)
	}
	if got := len(instantsDesVols(recs, 12)); got != 2 {
		t.Fatalf("%d vols, attendu 2 (la valeur finale du compteur)", got)
	}
}

// TestDepartSansRemplacantNeCoupePas : un siege qui retombe a zero sans que personne ne reprenne le comptage
// (fin de match, depart sans remplacant) ne coupe rien.
func TestDepartSansRemplacantNeCoupePas(t *testing.T) {
	recs := []types.StatRecord{
		recVols(10000, 12, 0, 0, 1, 0, 1), recVols(20000, 12, 0, 1, 1, 0, 2), recVols(30000, 12, 0, 1, 2, 0, 2),
		recVols(40000, 12, 0, 0, 0, 0, 0),
	}
	if cuts := seatCutsOf(recs).of(0, 12); cuts != nil {
		t.Fatalf("frontieres = %v, attendu aucune", cuts)
	}
	if got, want := instantsDesVols(recs, 12), []int{10000, 20000}; !slices.Equal(got, want) {
		t.Fatalf("vols = %v, attendu %v", got, want)
	}
}

// TestFinDeMancheNeCoupePas : une nouvelle manche remet les compteurs a zero pour tout le monde, ce qui n'est
// pas un changement d'occupant : les suites sont separees par la manche, comme avant, et le siege ne porte aucune
// frontiere.
func TestFinDeMancheNeCoupePas(t *testing.T) {
	var recs []types.StatRecord
	t0 := 1000
	for round := 0; round <= 1; round++ {
		recs = append(recs, recVols(t0, 12, round, 0, 0, 0, 0)) // la manche repart de zero
		for n := 1; n <= 3; n++ {
			recs = append(recs, recVols(t0+1000*n, 12, round, n, n, 0, min(n, 2)))
		}
		t0 += 10000
	}
	if cuts := seatCutsOf(recs); len(cuts) != 0 {
		t.Fatalf("frontieres = %v, attendu aucune : une fin de manche n'est pas un changement d'occupant", cuts)
	}
	if got, want := instantsDesVols(recs, 12), []int{2000, 3000, 12000, 13000}; !slices.Equal(got, want) {
		t.Fatalf("vols = %v, attendu %v (deux par manche)", got, want)
	}
}

// TestLongestRunBySeatSansFrontiereEstLongestRun : sans frontiere, la suite est celle de [longestRun], valeur
// pour valeur ; avec une frontiere, la suite de l'arrivant est decalee du dernier point du partant.
func TestLongestRunBySeatSansFrontiereEstLongestRun(t *testing.T) {
	pts := []types.ScorePoint{
		{TimeMS: 1, Slot: 12, Value: 1}, {TimeMS: 2, Slot: 12, Value: 2},
		{TimeMS: 3, Slot: 12, Value: 0}, {TimeMS: 4, Slot: 12, Value: 1},
	}
	if got, want := longestRunBySeat(pts, nil, false, nil), longestRun(pts, false); !slices.Equal(got, want) {
		t.Errorf("sans frontiere : %v, attendu %v", got, want)
	}
	got := longestRunBySeat(pts, []int{3}, false, nil)
	want := []types.ScorePoint{
		{TimeMS: 1, Slot: 12, Value: 1}, {TimeMS: 2, Slot: 12, Value: 2},
		{TimeMS: 3, Slot: 12, Value: 2}, {TimeMS: 4, Slot: 12, Value: 3},
	}
	if !slices.Equal(got, want) {
		t.Errorf("avec une frontiere a 3 ms : %v, attendu %v", got, want)
	}
}
