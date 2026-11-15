package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// degats_de_joueur_test.go — le message `damage_aftermath` (genre 0) de la vue A se range avec son effet, l'index
// de joueur de son auteur et son échelle ; le canal les rend avec l'horodatage de leur trame.

// ecrireDegatDeJoueur écrit un `damage_aftermath` : en-tête à gardes de référence fermées, effet (R(1) puis R(32)),
// index de joueur (R(1) à zéro puis R(5), ou R(1) à un), direction R(19), le reste de la charge à zéro.
func (w *bitWriter) ecrireDegatDeJoueur(tag *uint64, joueur *uint64, echelle uint64) {
	w.ecrireEnTeteDeMessage(GenreDegatsApres)
	if tag == nil {
		w.bit(0)
	} else {
		w.bit(1)
		w.bits(*tag, 32)
	}
	if joueur == nil {
		w.bit(1) // FUN_1407f2058 : porte à un, aucun index
	} else {
		w.bit(0)
		w.bits(*joueur, 5)
	}
	w.bits(0, largeurDirection)
	w.bit(0) // [+0x20]
	w.bits(echelle, 5)
	w.bits(0, 5+6)
	w.bit(0)       // consumeGateR(5)
	w.bits(0, 15)  // les quinze drapeaux
	w.bits(0, 1+3) // R(1), R(3)
	w.bits(0, 5+5)
	w.bits(0, 1+4)
	w.bit(1)     // consumeGate0R(10) : porte à un, rien
	w.bits(0, 4) // [+0x58]
	w.bit(0)     // consumeGateR(32)
	w.bits(0, 4) // bitLen(10)
	w.bit(0)     // la dernière garde
}

func valeur(v uint64) *uint64 { return &v }

// TestToutDegatDeJoueurSeRangeTelQuIlEstLu : l'effet, l'index de joueur ([lecture.RefAbsente] quand la porte est
// fermée), l'échelle et les drapeaux des gardes.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : lire l'index avec la polarité inverse de la porte ; ignorer la porte de
// l'effet ; lire l'échelle dans un autre champ ; consommer un bit de trop (le message suivant se décalerait).
func TestToutDegatDeJoueurSeRangeTelQuIlEstLu(t *testing.T) {
	cas := []struct {
		nom     string
		tag     *uint64
		joueur  *uint64
		echelle uint64
		attendu lecture.DegatDeJoueur
	}{
		{"effet, joueur et échelle pleine", valeur(0x0000563f), valeur(7), 15,
			lecture.DegatDeJoueur{Tag: 0x563f, Joueur: 7, Echelle: 15, Drapeaux: lecture.DegatTag}},
		{"sans effet, joueur 0", nil, valeur(0), 3,
			lecture.DegatDeJoueur{Joueur: 0, Echelle: 3}},
		{"effet sans joueur (une chute)", valeur(0xffffffff), nil, 0,
			lecture.DegatDeJoueur{Tag: 0xffffffff, Joueur: lecture.RefAbsente, Drapeaux: lecture.DegatTag}},
		{"joueur 31, échelle 31", valeur(1), valeur(31), 31,
			lecture.DegatDeJoueur{Tag: 1, Joueur: 31, Echelle: 31, Drapeaux: lecture.DegatTag}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var w bitWriter
			w.bit(1)
			w.ecrireDegatDeJoueur(c.tag, c.joueur, c.echelle)
			w.bit(0) // terminateur
			w.bits(0, 16)
			a := lireSous(w.buf, grammaireRecente())
			c.attendu.Debut = 1
			if !a.Porte || len(a.Degats) != 1 || a.Degats[0] != c.attendu {
				t.Fatalf("vue A %+v, dégât attendu %+v", a, c.attendu)
			}
			var p lecture.Paquet
			rangerLaVueA(&p, &a)
			if len(p.VueA.Degats) != 1 || p.VueA.Degats[0] != c.attendu {
				t.Errorf("dégât rangé %+v, attendu %+v", p.VueA.Degats, c.attendu)
			}
		})
	}
}

// TestLeCanalDesDegatsLesRendAvecLHorodatageDeLaTrame : chaque message de la vue A d'une trame sort avec son
// horodatage et ses drapeaux ; les trames et les messages sont comptés.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : perdre l'horodatage ; perdre un drapeau de garde ; ne pas compter les
// trames sans message.
func TestLeCanalDesDegatsLesRendAvecLHorodatageDeLaTrame(t *testing.T) {
	k := nouveauCanalDesDegats()
	var vide lecture.Paquet
	vide.TS = 5
	vide.VueA.Etat = lecture.VueArretee
	k.Tete(&vide)
	var p lecture.Paquet
	p.TS = 1_000
	p.VueA.Etat = lecture.VueTerminee
	p.VueA.Degats = []lecture.DegatDeJoueur{
		{Tag: 0x563f, Joueur: 4, Echelle: 15, Victime: 9, Responsable: 2, Drapeaux: lecture.DegatTag | lecture.DegatVictime | lecture.DegatResponsable},
		{Joueur: lecture.RefAbsente},
	}
	k.Tete(&p)
	want := []DegatLu{
		{TS: 1_000, Tag: 0x563f, Victime: 9, Responsable: 2, Joueur: 4, Echelle: 15, ATag: true, AVictime: true, AResponsable: true},
		{TS: 1_000, Joueur: lecture.RefAbsente},
	}
	if len(k.lus) != len(want) || k.lus[0] != want[0] || k.lus[1] != want[1] {
		t.Errorf("dégâts rendus %+v, attendu %+v", k.lus, want)
	}
	if k.stats != (StatsDegats{Trames: 2, Terminees: 1, Arretees: 1, Messages: 2}) {
		t.Errorf("comptes %+v, attendu 2 trames (une terminee, une arretee) et 2 messages", k.stats)
	}
}
