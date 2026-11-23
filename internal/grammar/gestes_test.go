package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLesChargesDeGesteGardentLIndexDeJoueur : le coup de mêlée (R(3) de tête, R(2) de suite si la tête vaut 1,
// R(2), l arme et la variante, l auteur) et le lancer amorcé (R(1) ; R(3) ou garde et R(4) ; l auteur) lisent le même
// nombre de bits qu avant et rangent l index de joueur de l auteur sur le lecteur ; une garde à un le laisse
// absent. MUTATIONS : la garde de l auteur inversée, l index sur quatre bits, l auteur non rangé -> ROUGE.
func TestLesChargesDeGesteGardentLIndexDeJoueur(t *testing.T) {
	cas := []struct {
		nom    string
		charge func(*Lecteur) bool
		ecrire func(w *bitWriter)
		joueur int8
	}{
		{"mêlée, tête courte, auteur nommé", chargeCorpsACorps,
			func(w *bitWriter) {
				w.bits(0, 3)
				w.bits(0, 2)
				w.bit(0)
				w.bits(0, 32)
				w.bit(0)
				w.bits(12, 5)
			}, 12},
		{"mêlée, tête avec suite, auteur absent", chargeCorpsACorps,
			func(w *bitWriter) {
				w.bits(1, 3)
				w.bits(2, 2)
				w.bits(1, 2)
				w.bit(0)
				w.bits(0, 32)
				w.bit(1)
			}, lecture.RefAbsente},
		{"lancer, branche à trois bits", chargeLancerInitie,
			func(w *bitWriter) { w.bit(0); w.bits(5, 3); w.bit(0); w.bits(31, 5) }, 31},
		{"lancer, branche à garde et quatre bits", chargeLancerInitie,
			func(w *bitWriter) { w.bit(1); w.bit(0); w.bits(9, 4); w.bit(1) }, lecture.RefAbsente},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var w bitWriter
			c.ecrire(&w)
			fin := w.n
			w.bits(0xffff, 16)
			br := LecteurSur(w.buf)
			br.gesteLu = lecture.MessageDeGeste{Joueur: 7} // une valeur périmée ne doit pas survivre
			if !c.charge(br) {
				t.Fatal("la charge refuse un flux valide")
			}
			if br.BitPos() != fin {
				t.Errorf("la charge s arrête au bit %d, attendu %d", br.BitPos(), fin)
			}
			if br.gesteLu.Joueur != c.joueur {
				t.Errorf("auteur %d, attendu %d", br.gesteLu.Joueur, c.joueur)
			}
		})
	}
}

// TestRecueillirUnGesteRangeLeGenreLaPositionEtLesReferences : seuls les genres 39 et 40 sont recueillis ; le
// message porte sa position, son genre, l auteur de la charge et les références que l en-tête a gardées.
// MUTATIONS : un genre de plus ou de moins, le genre non rangé, un drapeau de référence mal posé -> ROUGE.
func TestRecueillirUnGesteRangeLeGenreLaPositionEtLesReferences(t *testing.T) {
	br := LecteurSur(nil)
	br.gesteLu = lecture.MessageDeGeste{Joueur: 4}
	br.refsDuMessage = [3]refDeMessage{{presente: true, index: 77, gen: 2}, {}, {presente: true, index: 5, gen: 1}}
	var out FluxVueA
	if recueillirUnGeste(br, &out, GenreRechargement, 10) || len(out.Gestes) != 0 {
		t.Fatal("le genre 38 n est pas un geste")
	}
	if !recueillirUnGeste(br, &out, GenreCorpsACorps, 123) {
		t.Fatal("le genre 40 est un geste")
	}
	want := lecture.MessageDeGeste{Debut: 123, Ref: [3]uint16{77, 0, 5}, Gen: [3]uint8{2, 0, 1}, Joueur: 4,
		Genre: GenreCorpsACorps, Drapeaux: lecture.GesteRef0 | lecture.GesteRef2}
	if len(out.Gestes) != 1 || out.Gestes[0] != want {
		t.Errorf("rangé %+v, attendu %+v", out.Gestes, want)
	}
	br.refsDuMessage = [3]refDeMessage{}
	if !recueillirUnGeste(br, &out, GenreLancerInitie, 200) || out.Gestes[1].Genre != GenreLancerInitie || out.Gestes[1].Drapeaux != 0 {
		t.Errorf("lancer amorcé rangé %+v", out.Gestes)
	}
}

// TestLeCanalDesGestesRetireLesCopiesParGenre : deux messages de la même trame de même genre, même auteur et
// mêmes références sont un seul geste ; le même auteur avec l autre genre, un autre auteur ou d autres
// références en sont d autres ; le même message dans une trame suivante reste un geste ; les copies se comptent
// par genre et les trames par état de leur vue A.
// MUTATIONS : le genre hors de l identité, les références hors de l identité, l identité gardée d une trame à
// l autre, un compte de copies posé sur le mauvais genre -> ROUGE.
func TestLeCanalDesGestesRetireLesCopiesParGenre(t *testing.T) {
	msg := func(genre uint8, joueur int8, ref uint16) lecture.MessageDeGeste {
		return lecture.MessageDeGeste{Genre: genre, Joueur: joueur, Ref: [3]uint16{ref}, Gen: [3]uint8{1}, Drapeaux: lecture.GesteRef0}
	}
	c := nouveauCanalDesGestes()
	trame := func(ts uint64, etat lecture.EtatDeVue, ms ...lecture.MessageDeGeste) {
		p := &lecture.Paquet{TS: ts}
		p.VueA.Etat, p.VueA.Gestes = etat, ms
		c.Tete(p)
	}
	trame(100, lecture.VueTerminee, msg(40, 3, 17), msg(40, 3, 17), msg(39, 3, 17), msg(39, 3, 17), msg(39, 3, 17),
		msg(40, 4, 17), msg(40, 3, 18))
	trame(200, lecture.VueArretee, msg(40, 3, 17))
	trame(300, lecture.VueTerminee, msg(39, lecture.RefAbsente, 5), msg(39, lecture.RefAbsente, 5))

	attendus := []types.GestureRead{
		{TimestampUS: 100, Player: 3, Kind: types.GestureMelee}, {TimestampUS: 100, Player: 3, Kind: types.GestureThrowStart},
		{TimestampUS: 100, Player: 4, Kind: types.GestureMelee}, {TimestampUS: 100, Player: 3, Kind: types.GestureMelee},
		{TimestampUS: 200, Player: 3, Kind: types.GestureMelee}, {TimestampUS: 300, Player: types.GestureNoPlayer, Kind: types.GestureThrowStart},
	}
	if got := c.out; len(got) != len(attendus) {
		t.Fatalf("%d gestes, attendu %d : %+v", len(got), len(attendus), got)
	} else {
		for i := range got {
			if got[i] != attendus[i] {
				t.Errorf("geste %d : %+v, attendu %+v", i, got[i], attendus[i])
			}
		}
	}
	want := types.GestureStats{Frames: 3, Closed: 2, Stopped: 1,
		Melee: types.GestureCounts{Messages: 4, Copies: 1}, Throw: types.GestureCounts{Messages: 2, Copies: 3}}
	if c.stats != want {
		t.Errorf("comptes %+v, attendu %+v", c.stats, want)
	}
}
