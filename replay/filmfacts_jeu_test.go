package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	vuea "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// TestLesMessagesDeJeuFontLAllerRetourDesFaits : le témoin de lecture, les noms de script, les comptes du
// canal et chaque champ d'un message (horodatage, rang, séquence, mot texte, masque, deux participants,
// dont l'absent) se relisent à l'identique ; une liste vide se relit nil.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : perdre le signe d'un participant (l'absent relu 255), la séquence,
// le témoin de lecture, ou échanger deux champs.
func TestLesMessagesDeJeuFontLAllerRetourDesFaits(t *testing.T) {
	absent := vuea.RefAbsente
	cas := []struct {
		nom string
		in  FilmInputs
	}{
		{"non lu", FilmInputs{}},
		{"lu, aucun message", FilmInputs{GameEventsRead: true, Scripts: []string{"slayerevents.lua"},
			GameEventStats: grammar.StatsEvenementsDeJeu{Trames: 3, Terminees: 3}}},
		{"messages", FilmInputs{
			GameEventsRead: true,
			Scripts:        []string{"arenaflagevents.lua", "flageventstubs.lua"},
			GameEventStats: grammar.StatsEvenementsDeJeu{Trames: 9000, Terminees: 8990, Arretees: 10, Messages: 3, Copies: 2},
			GameEvents: []grammar.EvenementDeJeuLu{
				{TS: 4_517_903_000, Rang: 78, Sequence: 255, Destinataires: 1 << 31, Participants: [2]int8{absent, absent}},
				{TS: 4_517_904_000, Rang: 0xffffffff, Sequence: 0, Texte: 0x365be6b1, ATexte: true, Destinataires: 0xffffffff,
					Participants: [2]int8{31, 0}},
				{TS: 0, Rang: 1, Sequence: 1, Texte: 7, ATexte: true, Destinataires: 1, Participants: [2]int8{absent, 17}},
			}}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			g := &FilmFacts{FilmInputs: c.in}
			w := &gwriter{}
			encodeJeu(w, g)
			r := &greader{b: w.b}
			var relu FilmFacts
			decodeJeu(r, &relu)
			if r.err != nil || r.off != len(r.b) {
				t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
			}
			if !reflect.DeepEqual(relu.FilmInputs.GameEvents, c.in.GameEvents) ||
				relu.GameEventsRead != c.in.GameEventsRead || relu.GameEventStats != c.in.GameEventStats ||
				!reflect.DeepEqual(relu.Scripts, c.in.Scripts) {
				t.Errorf("relu %+v, attendu %+v", relu.FilmInputs, c.in)
			}
		})
	}
}
