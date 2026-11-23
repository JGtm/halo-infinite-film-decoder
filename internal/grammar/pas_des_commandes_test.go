package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// pas_des_commandes_test.go — LE PAS DES ENTREES DE CONTROLE : l ecart le plus frequent entre deux entrees lues d une
// meme place, sans trou entre elles. Scenarios synthetiques, sans film.

// entreesALaCadence rend n paquets espaces de `pasUS` (avec une gigue de plus ou moins `gigueUS` un paquet sur deux),
// dont chacun porte l entree de la place `index`.
func entreesALaCadence(index, n int, depuisUS, pasUS, gigueUS uint64) []paquetTC {
	var ps []paquetTC
	t := depuisUS
	for i := range n {
		ps = append(ps, paquetTC{ts: t, entrees: []EntreeDeControle{entreeDeBoutons(index, 0)}})
		t += pasUS
		if i%2 == 0 {
			t += gigueUS
		} else {
			t -= gigueUS
		}
	}
	return ps
}

// TestLePasEstLEcartLePlusFrequentEntreDeuxEntreesLues : la gigue partage les ecarts entre deux rangs voisins, et la
// fenetre de trois millisecondes les reunit ; le pas d un film a la moitie de la cadence est le double ; un ecart rare
// et tres long ne se fait pas le pas. MUTATION : la fenetre reduite a la seule milliseconde la plus peuplee -> ROUGE.
func TestLePasEstLEcartLePlusFrequentEntreDeuxEntreesLues(t *testing.T) {
	_, st, _ := plierLesCommandes(entreesALaCadence(2, 400, 1_000_000, 16_667, 400))
	if st.Gaps != 399 || st.StepUS < 16_000 || st.StepUS > 17_000 || st.StepGaps < 390 {
		t.Errorf("60 Hz : %+v", st)
	}
	_, st, _ = plierLesCommandes(entreesALaCadence(2, 400, 1_000_000, 33_333, 400))
	if st.StepUS < 33_000 || st.StepUS > 34_000 || st.StepGaps < 390 {
		t.Errorf("30 Hz : %+v", st)
	}
	// un ecart rare, tres long, ne se fait pas le pas.
	ps := entreesALaCadence(2, 100, 1_000_000, 16_667, 0)
	ps = append(ps, paquetTC{ts: 5_000_000, entrees: []EntreeDeControle{entreeDeBoutons(2, 0)}})
	_, st, _ = plierLesCommandes(ps)
	if st.Gaps != 100 || st.StepUS != 16_667 || st.StepGaps != 99 {
		t.Errorf("ecart long : %+v, attendu un pas de 16 667 µs sur 99 ecarts, 100 mesures", st)
	}
}

// TestUnTrouRompLaMesureDuPas : l ecart qui traverse un paquet non lu mesure le trou, pas la cadence ; les places
// sont suivies separement ; deux entrees d une place dans le meme paquet et une entree sans bloc ne mesurent rien.
// MUTATION : le trou qui ne rompt pas la mesure -> ROUGE.
func TestUnTrouRompLaMesureDuPas(t *testing.T) {
	_, st, _ := plierLesCommandes([]paquetTC{
		{ts: 1_000, entrees: []EntreeDeControle{entreeDeBoutons(1, 0), entreeDeBoutons(2, 0)}},
		{ts: 17_000, entrees: []EntreeDeControle{entreeDeBoutons(1, 0)}}, // place 1 : 16 ms ; place 2 : rien
		{ts: 20_000, trou: true},
		{ts: 33_000, entrees: []EntreeDeControle{entreeDeBoutons(1, 0), entreeDeBoutons(2, 0)}}, // apres le trou : aucun ecart
		{ts: 49_000, entrees: []EntreeDeControle{entreeDeBoutons(1, 0), entreeDeBoutons(1, 0), {Index: 2}}},
		{ts: 65_000, entrees: []EntreeDeControle{entreeDeBoutons(1, 0)}},
	})
	// ecarts : place 1 : 17000-1000, 49000-33000, 65000-49000 ; place 2 : aucun (l entree sans bloc est muette).
	if st.Gaps != 3 || st.StepGaps != 3 || st.StepUS != 16_000 {
		t.Errorf("compteurs %+v, attendu 3 ecarts de 16 000 µs", st)
	}
}

// TestUnFilmSansEntreeLueNAPasDePas : aucun ecart, aucun pas.
func TestUnFilmSansEntreeLueNAPasDePas(t *testing.T) {
	_, st, _ := plierLesCommandes([]paquetTC{{ts: 10, trou: true}, {ts: 20, trou: true}})
	if st.Gaps != 0 || st.StepUS != 0 || st.StepGaps != 0 || st.PleineCadence() {
		t.Errorf("compteurs %+v", st)
	}
}

// TestLePasEgaliteLePlusCourtGagne : a rangs egaux, le plus court l emporte, quel que soit l ordre de lecture.
func TestLePasEgaliteLePlusCourtGagne(t *testing.T) {
	var p pasDesEntrees
	for _, de := range [][2]uint64{{0, 33_000}, {33_000, 50_000}} {
		for range 3 {
			p.noter(1, de[0])
			p.noter(1, de[1])
			p.rompre()
		}
	}
	var st types.ControlHoldStats
	p.mesurer(&st)
	if st.StepUS != 17_000 || st.StepGaps != 3 || st.Gaps != 6 {
		t.Errorf("compteurs %+v, attendu 17 000 µs sur 3 ecarts", st)
	}
}
