package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// TestCodecDesHorloges : les lectures de l horloge et des comptes a rebours, et leurs
// denominateurs, font l aller-retour a l identique ; une section vide se relit vide, a la longueur du format.
func TestCodecDesHorloges(t *testing.T) {
	h := grammar.HorlogesLues{
		Horloges: []grammar.LectureHorloge{
			{TS: 4_519_395_277, Limite: 1311, Reste: 1311, Etat: 1},
			{TS: 4_543_811_366, Limite: 1311, Reste: 1311, Etat: 6},
			{TS: 4_544_312_367, Limite: 1311, Reste: 1309, Etat: 6},
		},
		Comptes: []grammar.LectureCompteARebours{
			{TS: 4_553_208_861, Slot: 55, Joueur: 3, Reste: 8, Total: 8},
			{TS: 4_554_206_257, Slot: 55, Joueur: 3, Actif: true, Reste: 7, Total: 8},
			{TS: 4_554_206_257, Slot: 80, Joueur: -1, Reste: 906, Total: 906},
		},
		Stats: grammar.StatsDesHorloges{Trames: 7042, TramesNonLues: 44, Horloges: 3, Comptes: 3, CompteSansJoueur: 1,
			Creations: 384},
	}
	w := &gwriter{}
	encodeHorloges(w, h)
	r := &greader{b: w.b}
	if got := decodeHorloges(r); r.err != nil || r.off != len(w.b) || !reflect.DeepEqual(got, h) {
		t.Fatalf("aller-retour : %+v (erreur %v, %d octets lus sur %d), attendu %+v", got, r.err, r.off, len(w.b), h)
	}

	w = &gwriter{}
	encodeHorloges(w, grammar.HorlogesLues{})
	if len(w.b) != 8 {
		t.Errorf("section vide : %d octets, attendu 8 (deux comptes et six compteurs a zero)", len(w.b))
	}
	r = &greader{b: w.b}
	vide := decodeHorloges(r)
	if r.err != nil || len(vide.Horloges)+len(vide.Comptes) != 0 || vide.Stats != (grammar.StatsDesHorloges{}) {
		t.Errorf("section vide relue : %+v (erreur %v)", vide, r.err)
	}
}
