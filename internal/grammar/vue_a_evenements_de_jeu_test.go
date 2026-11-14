package grammar

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// vue_a_evenements_de_jeu_test.go — tous les messages `PlayerGameEventSmall` (genre 82) de la vue A se
// rangent avec leur rang, leur séquence, leur mot texte, leurs deux premiers participants et leur
// masque, couple de participants ou non ; le canal les rend sans les copies que le jeu renvoie.

// entreeDeTexte est un emplacement du sac texte d'un message écrit par les tests.
type entreeDeTexte struct {
	sousType uint64
	absent   bool   // participant dont la porte `FUN_1407f2058` est à 1 : aucun indice
	indice   uint64 // indice R(5) d'un participant présent
}

// ecrireEvenementDeJeu écrit un `PlayerGameEventSmall` : en-tête, rang R(32), séquence R(8), sac principal
// vide, sac texte (fermé si `texte` est nil), puis le masque des destinataires.
func (w *bitWriter) ecrireEvenementDeJeu(rang, sequence uint64, texte *uint64, entrees []entreeDeTexte, masque uint64) {
	w.ecrireEnTeteDeMessage(GenreEvenementJoueurCourt)
	w.bits(rang, 32)
	w.bits(sequence, 8)
	w.bits(0, int(largeurCompteSacCourt))
	if texte == nil {
		w.bit(0) // porte « texte » fermée
	} else {
		w.bit(1)
		w.bits(*texte, 32)
		w.bits(uint64(len(entrees)), 3)
		for _, e := range entrees {
			w.bits(e.sousType, 3)
			switch {
			case e.sousType != sacTexteParticipant:
			case e.absent:
				w.bit(1)
			default:
				w.bit(0)
				w.bits(e.indice, 5)
			}
		}
	}
	w.bits(masque, ParticipantsDuFil)
}

func motTexte(v uint64) *uint64 { return &v }

// TestToutMessageDeJeuSeRangeTelQuIlEstLu : le rang, la séquence, le mot texte, les deux premiers
// participants (dans l'ordre de l'écriture, [lecture.RefAbsente] quand il en manque) et le masque.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne ranger que les messages à couple ; compter un emplacement qui
// n'est pas un participant ; garder le troisième participant ; lire la séquence dans le rang.
func TestToutMessageDeJeuSeRangeTelQuIlEstLu(t *testing.T) {
	const absent = lecture.RefAbsente
	cas := []struct {
		nom      string
		texte    *uint64
		entrees  []entreeDeTexte
		attendu  lecture.EvenementDeJeu
		couple   bool
		sequence uint64
	}{
		{"sans sac texte", nil, nil, lecture.EvenementDeJeu{Participants: [2]int8{absent, absent}}, false, 7},
		{"mot texte seul", motTexte(0x3d097626), nil,
			lecture.EvenementDeJeu{Texte: 0x3d097626, ATexte: true, Participants: [2]int8{absent, absent}}, false, 0},
		{"un participant, la victime nommée", motTexte(0x365be6b1), []entreeDeTexte{{sousType: sacTexteParticipant, indice: 9}},
			lecture.EvenementDeJeu{Texte: 0x365be6b1, ATexte: true, Participants: [2]int8{9, absent}}, false, 255},
		{"deux participants, un couple", motTexte(1), []entreeDeTexte{
			{sousType: sacTexteParticipant, indice: 4}, {sousType: sacTexteParticipant, indice: 11}},
			lecture.EvenementDeJeu{Texte: 1, ATexte: true, Participants: [2]int8{4, 11}}, true, 3},
		{"participant sans indice puis un indice", motTexte(1), []entreeDeTexte{
			{sousType: sacTexteParticipant, absent: true}, {sousType: sacTexteParticipant, indice: 31}},
			lecture.EvenementDeJeu{Texte: 1, ATexte: true, Participants: [2]int8{absent, 31}}, false, 3},
		{"un drapeau entre deux participants", motTexte(1), []entreeDeTexte{
			{sousType: sacTexteParticipant, indice: 2}, {sousType: sacTexteDrapeau}, {sousType: sacTexteParticipant, indice: 5}},
			lecture.EvenementDeJeu{Texte: 1, ATexte: true, Participants: [2]int8{2, 5}}, false, 3},
		{"trois participants : les deux premiers", motTexte(1), []entreeDeTexte{
			{sousType: sacTexteParticipant, indice: 1}, {sousType: sacTexteParticipant, indice: 2},
			{sousType: sacTexteParticipant, indice: 3}},
			lecture.EvenementDeJeu{Texte: 1, ATexte: true, Participants: [2]int8{1, 2}}, false, 3},
	}
	const masque = 1<<(ParticipantsDuFil-1-5) | 1<<(ParticipantsDuFil-1-30)
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var w bitWriter
			w.bit(1)
			w.ecrireEvenementDeJeu(78, c.sequence, c.texte, c.entrees, masque)
			w.bit(0) // terminateur
			w.bits(0, 16)
			a := lireSous(w.buf, grammaireRecente())
			c.attendu.Debut, c.attendu.Rang, c.attendu.Sequence, c.attendu.Destinataires = 1, 78, uint8(c.sequence), masque
			if !a.Porte || len(a.Jeu) != 1 || a.Jeu[0] != c.attendu {
				t.Fatalf("vue A %+v, jeu attendu %+v", a, c.attendu)
			}
			if got := len(a.Fil) == 1; got != c.couple {
				t.Errorf("message à couple rangé dans le fil : %v, attendu %v", got, c.couple)
			}
			var p lecture.Paquet
			rangerLaVueA(&p, &a)
			if !slices.Equal(p.VueA.Jeu, []lecture.EvenementDeJeu{c.attendu}) {
				t.Errorf("jeu rangé %+v, attendu %+v", p.VueA.Jeu, c.attendu)
			}
		})
	}
}

// TestLeCanalDesEvenementsDeJeuRetireLesCopies : une copie qui répète le rang, la séquence, le mot texte et
// le masque dans la seconde qui suit est retirée ; une autre séquence, un autre masque, un autre mot texte,
// ou la même identité une seconde après, sont gardés.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas comparer le masque (les copies par destinataire sont
// distinctes) ; comparer à la dernière copie rendue plutôt qu'à la première vue ; fenêtre infinie.
func TestLeCanalDesEvenementsDeJeuRetireLesCopies(t *testing.T) {
	base := lecture.EvenementDeJeu{Debut: 99, Rang: 78, Sequence: 4, Destinataires: 1 << 31}
	avec := func(f func(*lecture.EvenementDeJeu)) lecture.EvenementDeJeu { e := base; f(&e); return e }
	cas := []struct {
		nom    string
		suite  []lecture.EvenementDeJeu
		ts     []uint64
		gardes []uint64 // horodatages des messages rendus
	}{
		{"copie dans la seconde", []lecture.EvenementDeJeu{base, base}, []uint64{1_000, 400_000}, []uint64{1_000}},
		{"copie à la seconde pile", []lecture.EvenementDeJeu{base, base}, []uint64{1_000, 1_001_000}, []uint64{1_000, 1_001_000}},
		{"autre séquence", []lecture.EvenementDeJeu{base, avec(func(e *lecture.EvenementDeJeu) { e.Sequence = 5 })},
			[]uint64{1_000, 2_000}, []uint64{1_000, 2_000}},
		{"autre masque", []lecture.EvenementDeJeu{base, avec(func(e *lecture.EvenementDeJeu) { e.Destinataires = 1 << 30 })},
			[]uint64{1_000, 2_000}, []uint64{1_000, 2_000}},
		{"autre mot texte", []lecture.EvenementDeJeu{base, avec(func(e *lecture.EvenementDeJeu) { e.Texte, e.ATexte = 8, true })},
			[]uint64{1_000, 2_000}, []uint64{1_000, 2_000}},
		{"trois copies : la fenêtre part de la première", []lecture.EvenementDeJeu{base, base, base},
			[]uint64{1_000, 700_000, 1_100_000}, []uint64{1_000, 1_100_000}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			k := nouveauCanalDesEvenementsDeJeu()
			for i, e := range c.suite {
				var p lecture.Paquet
				p.TS = c.ts[i]
				p.VueA.Jeu = []lecture.EvenementDeJeu{e}
				k.Tete(&p)
			}
			var vus []uint64
			for _, e := range k.lus {
				vus = append(vus, e.TS)
			}
			if !slices.Equal(vus, c.gardes) {
				t.Errorf("messages rendus aux horodatages %v, attendu %v", vus, c.gardes)
			}
		})
	}
}
