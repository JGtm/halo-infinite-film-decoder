package grammar

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// vue_a_charges_effets_test.go — les charges des genres 13, 27, 93 et 117 de la vue A, ecrites avec leurs
// largeurs en litteraux (celles du temoin libfilmshell pour 13, 27 et 117 ; pour 93, la seule charge inhibee,
// la seule que les films montrent). Chaque message est suivi d un genre deja lu (22, sortie de vehicule : dix bits),
// puis du terminateur : une charge lue a une autre largeur decale ces dix bits et la lecture ne s arrete
// plus au bit attendu.

// ecrireUnSuivant ecrit le message qui suit la charge sous test : un genre 22 et ses dix bits, puis le
// terminateur. Rend le bit qui suit le terminateur.
func (w *bitWriter) ecrireUnSuivant() int {
	w.ecrireEnTeteDeMessage(22)
	w.bits(0x2a5, 10)
	w.bit(0) // terminateur
	fin := w.n
	w.bits(0, 16)
	return fin
}

// verifierLaLectureJusquAuBout lit `buf` et exige une vue A portee jusqu au bit `fin` avec les genres
// `attendus`.
func verifierLaLectureJusquAuBout(t *testing.T, nom string, buf []byte, fin int, attendus []int) {
	t.Helper()
	a := lireSous(buf, grammaireRecente())
	if !a.Porte || a.Fin != fin || !slices.Equal(a.Genres, attendus) {
		t.Errorf("%s : vue A %+v, attendu portee jusqu au bit %d avec les genres %v", nom, a, fin, attendus)
	}
}

// TestLesChargesCourtesSeTraversentSurToutesLeursBranches : un message de chaque genre, sur chacune de
// ses branches lues, laisse la lecture aller jusqu au terminateur. MUTATIONS — le R(7) large de 13 lu sur 6
// bits, son R(1) de queue omis, le R(2) de 27 lu sur 1 bit, l en-tete du 93 lu sur 14 bits, l identifiant d effet
// de 117 omis ou lu sans sa porte : ROUGES.
func TestLesChargesCourtesSeTraversentSurToutesLeursBranches(t *testing.T) {
	largeurs := profile.LargeursAxeParDefautDuBuild(niveauPosition)
	position := int(largeurs[0] + largeurs[1] + largeurs[2])
	cas := []struct {
		nom     string
		genre   int
		charge  func(w *bitWriter)
		attendu []int
	}{
		{"13 large", 13, func(w *bitWriter) { w.bit(1); w.bits(0x55, 7); w.bit(1) }, nil},
		{"13 court", 13, func(w *bitWriter) { w.bit(0); w.bits(2, 2); w.bit(0) }, nil},
		{"27", 27, func(w *bitWriter) { w.bits(3, 2) }, nil},
		{"93 inhibe", 93, func(w *bitWriter) { w.bits(0x4a5a, 15); w.bit(1) }, nil},
		{"117 sans effet", 117, func(w *bitWriter) {
			w.bit(0)
			w.bit(1)
			w.bits(0x155, position)
			w.bit(1)
			w.bits(0x2aa, position)
		}, nil},
		{"117 avec effet", 117, func(w *bitWriter) {
			w.bit(1)
			w.bits(0xa1344fc2, 32)
			w.bit(1)
			w.bits(0x155, position)
			w.bit(1)
			w.bits(0x2aa, position)
		}, nil},
	}
	for _, c := range cas {
		var w bitWriter
		w.bit(1) // bit de configuration
		w.ecrireEnTeteDeMessage(uint64(c.genre))
		c.charge(&w)
		fin := w.ecrireUnSuivant()
		verifierLaLectureJusquAuBout(t, c.nom, w.buf, fin, []int{c.genre, 22})
	}
}

// TestLeGenre93NonInhibeArreteLaLecture : sans inhibition le lecteur lit une branche que aucun film ne
// montre ; la charge rend faux, et la lecture s arrete apres le genre, comme avant, pour chaque valeur de la
// branche.
func TestLeGenre93NonInhibeArreteLaLecture(t *testing.T) {
	for k := range 4 {
		var w bitWriter
		w.bit(1)
		w.ecrireEnTeteDeMessage(93)
		w.bits(0x1234, 15)
		w.bit(0)
		w.bits(uint64(k), 2)
		w.ecrireUnSuivant()
		fin := 1 + 1 + LargeurGenreVueA
		a := lireSous(w.buf, grammaireRecente())
		if a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{93}) {
			t.Errorf("93 non inhibe, branche %d : %+v, attendu arretee au bit %d apres le seul genre 93", k, a, fin)
		}
	}
}

// TestUneChargeCourteTronqueeArreteLaLecture : un message dont la charge deborde le payload arrete la
// lecture apres son genre (le genre 27, deux bits, ne deborde jamais le dernier octet : sans cas).
func TestUneChargeCourteTronqueeArreteLaLecture(t *testing.T) {
	cas := []struct {
		nom    string
		genre  uint64
		charge func(w *bitWriter)
	}{
		{"13 large", 13, func(w *bitWriter) { w.bit(1); w.bits(5, 3) }},
		{"93", 93, func(w *bitWriter) { w.bits(0x1234, 12) }},
		{"117", 117, func(w *bitWriter) { w.bit(1); w.bits(0xcafe, 16) }},
	}
	for _, c := range cas {
		var w bitWriter
		w.bit(1)
		w.ecrireEnTeteDeMessage(c.genre)
		c.charge(&w)
		for w.n%8 != 0 {
			w.bit(0)
		}
		fin := 1 + 1 + LargeurGenreVueA
		a := lireSous(w.buf, grammaireRecente())
		if a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{int(c.genre)}) {
			t.Errorf("%s tronque : %+v, attendu arretee au bit %d", c.nom, a, fin)
		}
	}
}
