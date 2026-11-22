package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLaChargeDeRechargementLitQuatreBooleensEtLAuteur : quatre R(1) dans l ordre de la lecture, puis
// l index de joueur de l auteur (R(1) ; a 0, R(5)). MUTATIONS : la porte de l auteur inversee, un booleen
// de moins ou de plus, l index sur quatre bits -> ROUGE.
func TestLaChargeDeRechargementLitQuatreBooleensEtLAuteur(t *testing.T) {
	cas := []struct {
		nom      string
		ecrire   func(w *bitWriter)
		booleens uint8
		joueur   int8
		bits     int
	}{
		{"auteur nomme", func(w *bitWriter) { w.bits(0b1011, 4); w.bit(0); w.bits(21, 5) }, 0b1101, 21, 10},
		{"sans auteur", func(w *bitWriter) { w.bits(0b0100, 4); w.bit(1) }, 0b0010, lecture.RefAbsente, 5},
		{"place maximale", func(w *bitWriter) { w.bits(0, 4); w.bit(0); w.bits(31, 5) }, 0, 31, 10},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var w bitWriter
			c.ecrire(&w)
			br := LecteurSur(w.buf)
			if !chargeRechargement(br) {
				t.Fatal("la charge refuse un flux valide")
			}
			if br.BitPos() != c.bits {
				t.Errorf("la charge s arrete au bit %d, attendu %d", br.BitPos(), c.bits)
			}
			if m := br.rechargementLu; m.Booleens != c.booleens || m.Joueur != c.joueur {
				t.Errorf("lu booleens %04b joueur %d, attendu %04b et %d", m.Booleens, m.Joueur, c.booleens, c.joueur)
			}
		})
	}
}

// TestLeCanalDesRechargementsRetireLesCopiesDeLaTrame : deux messages de la meme trame qui ne different que
// par leurs booleens sont un seul rechargement ; un autre auteur ou d autres references en sont d autres ; le
// meme message dans une trame suivante reste un rechargement, et les trames se comptent par etat de leur vue A.
// MUTATIONS : l identite sans les references, les booleens dans l identite, l identite gardee d une trame a
// l autre -> ROUGE.
func TestLeCanalDesRechargementsRetireLesCopiesDeLaTrame(t *testing.T) {
	msg := func(joueur int8, ref uint16, booleens uint8) lecture.MessageDeRechargement {
		return lecture.MessageDeRechargement{Joueur: joueur, Ref: [3]uint16{ref}, Gen: [3]uint8{1},
			Drapeaux: lecture.RechargementRef0, Booleens: booleens}
	}
	c := nouveauCanalDesRechargements()
	trame := func(ts uint64, etat lecture.EtatDeVue, ms ...lecture.MessageDeRechargement) {
		p := &lecture.Paquet{TS: ts}
		p.VueA.Etat, p.VueA.Rechargements = etat, ms
		c.Tete(p)
	}
	trame(100, lecture.VueTerminee, msg(3, 17, 3), msg(3, 17, 3), msg(3, 17, 11), msg(4, 17, 3), msg(3, 18, 3))
	trame(200, lecture.VueArretee, msg(3, 17, 3))
	trame(300, lecture.VueTerminee, msg(lecture.RefAbsente, 5, 2), msg(lecture.RefAbsente, 5, 2))

	attendus := []types.ReloadRead{{TimestampUS: 100, Player: 3}, {TimestampUS: 100, Player: 4}, {TimestampUS: 100, Player: 3},
		{TimestampUS: 200, Player: 3}, {TimestampUS: 300, Player: types.ReloadNoPlayer}}
	if got := c.out; len(got) != len(attendus) {
		t.Fatalf("%d rechargements, attendu %d : %+v", len(got), len(attendus), got)
	} else {
		for i := range got {
			if got[i] != attendus[i] {
				t.Errorf("rechargement %d : %+v, attendu %+v", i, got[i], attendus[i])
			}
		}
	}
	want := types.ReloadStats{Frames: 3, Closed: 2, Stopped: 1, Messages: 5, Copies: 3}
	if c.stats != want {
		t.Errorf("comptes %+v, attendu %+v", c.stats, want)
	}
}
