package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	vuea "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// TestLesDegatsFontLAllerRetourDesFaits : le témoin de lecture, les comptes du canal et chaque champ d'un message
// (horodatage, effet, deux unités, index de joueur dont l'absent, échelle, gardes) se relisent à l'identique ; une
// liste vide se relit nil.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : perdre le signe de l'index (l'absent relu 255), un drapeau de garde, le
// témoin de lecture, un compte du canal, ou échanger deux champs.
func TestLesDegatsFontLAllerRetourDesFaits(t *testing.T) {
	cas := []struct {
		nom string
		in  FilmInputs
	}{
		{"non lu", FilmInputs{}},
		{"lu, aucun message", FilmInputs{DamagesRead: true, DamageStats: grammar.StatsDegats{Trames: 3, Terminees: 3}}},
		{"messages", FilmInputs{
			DamagesRead: true,
			DamageStats: grammar.StatsDegats{Trames: 9000, Terminees: 8990, Arretees: 10, Messages: 3, VitalesNonLues: 4},
			Damages: []grammar.DegatLu{
				{TS: 4_517_903_000, Tag: 0x563f, Victime: 9, Responsable: 2, Joueur: 4, Echelle: 15, ATag: true, AVictime: true, AResponsable: true},
				{TS: 4_517_904_000, Tag: 0xffffffff, Victime: 0xffff, Responsable: 0xffff, Joueur: vuea.RefAbsente, Echelle: 31, ATag: true},
				{TS: 0, Joueur: 31, AVictime: true},
			}}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			g := &FilmFacts{FilmInputs: c.in}
			w := &gwriter{}
			encodeDegats(w, g)
			r := &greader{b: w.b}
			var relu FilmFacts
			decodeDegats(r, &relu)
			if r.err != nil || r.off != len(r.b) {
				t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
			}
			if !reflect.DeepEqual(relu.FilmInputs.Damages, c.in.Damages) || relu.DamagesRead != c.in.DamagesRead || relu.DamageStats != c.in.DamageStats {
				t.Errorf("relu %+v, attendu %+v", relu.FilmInputs, c.in)
			}
		})
	}
}
