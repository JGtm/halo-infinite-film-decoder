package grammar

// evenements_de_jeu.go — LES MESSAGES DE JEU DU FILM, DEDOUBLONNES (genre 82 de la vue A).
//
// Le jeu envoie à chaque joueur concerné une copie de la réponse du script de mode (kill, assistance,
// capture, score personnel...), et peut renvoyer une copie 10 à 25 ticks plus tard avec le même numéro de
// séquence. Le canal rend, dans l'ordre du flux, chaque message de jeu lu par la marche des trames
// ([lecture.VueA.Jeu]) une seule fois : une copie qui répète le rang, la séquence, le mot texte et le
// masque d'un message vu dans la seconde qui précède n'est pas rendue. Une trame entière renvoyée (tous ses messages,
// au moins deux, répètent ceux d'une même trame de moins de 3 s) ne l'est pas non plus. Il ne nomme aucune réponse : le
// rang ne se lit qu'avec le groupe de scripts du film ([ScanScriptsDuFilm]).

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// fenetreDesCopiesDeJeu est la durée, en microsecondes, dans laquelle une copie renvoyée par le jeu se
// reconnaît à son identité.
const fenetreDesCopiesDeJeu = 1_000_000

// fenetreDuRenvoiDeTrame est la durée, en microsecondes, dans laquelle une trame que le jeu renvoie en entier
// se reconnaît : tous ses messages (au moins [messagesMinDuRenvoiDeTrame]) répètent, à clé identique, ceux d'une
// même trame précédente. Un renvoi a été observé à 2,365 s (35 messages identiques un à un, une élimination
// comptée deux fois) ; la fenêtre d'une seconde, seule, le laissait passer. Elle ne vaut que pour la trame
// entière : des messages ISOLÉS de clé identique existent à 1,0 à 1,95 s d'écart et sont légitimes.
const fenetreDuRenvoiDeTrame = 3_000_000

// messagesMinDuRenvoiDeTrame est le nombre de messages à partir duquel une trame entièrement répétée passe pour
// un renvoi : un message seul répété relève de la règle d'une seconde, jamais de celle-ci.
const messagesMinDuRenvoiDeTrame = 2

// EvenementDeJeuLu est un message de jeu d'une trame delta ([lecture.EvenementDeJeu], sans son début : il ne
// situe qu'un bit d'un paquet), avec l'horodatage de la trame (microsecondes, horloge du film).
type EvenementDeJeuLu struct {
	TS            uint64
	Rang          uint32
	Texte         uint32
	Destinataires uint32
	Participants  [2]int8
	Sequence      uint8
	ATexte        bool
}

// StatsEvenementsDeJeu dit ce que le canal a vu : les trames delta, celles dont la vue A est lue jusqu'a son
// terminateur et celles qu'un message non lu arrete (les messages derriere cet arret ne se lisent pas), les
// messages rendus et les copies renvoyees qu'il a retirees.
type StatsEvenementsDeJeu struct {
	Trames, Terminees, Arretees, Messages, Copies int
}

// copieDeJeu est l'identité d'un message pour la détection des copies renvoyées.
type copieDeJeu struct {
	rang, texte, destinataires uint32
	sequence                   uint8
}

// canalDesEvenementsDeJeu recueille les messages de jeu des trames delta d'un film, sans les copies.
type canalDesEvenementsDeJeu struct {
	lus   []EvenementDeJeuLu
	stats StatsEvenementsDeJeu
	vus   map[copieDeJeu]uint64
}

func nouveauCanalDesEvenementsDeJeu() *canalDesEvenementsDeJeu {
	return &canalDesEvenementsDeJeu{vus: map[copieDeJeu]uint64{}}
}

// Interets : le canal ne lit que la vue A rangée de chaque trame.
func (c *canalDesEvenementsDeJeu) Interets() []Interet { return nil }

// Clore : rien — le canal se lit par son champ.
func (c *canalDesEvenementsDeJeu) Clore(BilanDeMarche) {}

// Tete range les messages de jeu de la vue A de la trame `p`, copies renvoyées exceptées.
func (c *canalDesEvenementsDeJeu) Tete(p *lecture.Paquet) {
	c.stats.Trames++
	switch p.VueA.Etat {
	case lecture.VueTerminee:
		c.stats.Terminees++
	case lecture.VueArretee:
		c.stats.Arretees++
	}
	if c.estUnRenvoiDeTrame(p) {
		c.stats.Copies += len(p.VueA.Jeu)
		return
	}
	for _, e := range p.VueA.Jeu {
		k := copieDeJeu{e.Rang, e.Texte, e.Destinataires, e.Sequence}
		if ts, vu := c.vus[k]; vu && p.TS >= ts && p.TS-ts < fenetreDesCopiesDeJeu {
			c.stats.Copies++
			continue
		}
		c.vus[k] = p.TS
		c.lus = append(c.lus, EvenementDeJeuLu{TS: p.TS, Rang: e.Rang, Texte: e.Texte, Destinataires: e.Destinataires,
			Participants: e.Participants, Sequence: e.Sequence, ATexte: e.ATexte})
		c.stats.Messages++
	}
}

// PremierParticipant rend l'index de joueur du film que le premier emplacement participant du texte du
// message nomme (`%pa`, `%pe`), ou faux quand le message n'en porte pas.
func (e EvenementDeJeuLu) PremierParticipant() (int, bool) {
	if e.Participants[0] == lecture.RefAbsente {
		return 0, false
	}
	return int(e.Participants[0]), true
}

// estUnRenvoiDeTrame dit si la trame `p` est le renvoi entier d'une trame précédente : au moins
// [messagesMinDuRenvoiDeTrame] messages, tous de clé (rang, mot texte, masque, séquence) déjà vue, tous à
// l'horodatage de la même trame, vieille de moins de [fenetreDuRenvoiDeTrame]. Une trame qui apporte un seul
// message neuf ou une autre origine n'en est pas un. Le renvoi se compte parmi les copies, message par message ;
// il ne rafraîchit pas l'horodatage des clés, pour que la fenêtre parte de la trame d'origine.
func (c *canalDesEvenementsDeJeu) estUnRenvoiDeTrame(p *lecture.Paquet) bool {
	if len(p.VueA.Jeu) < messagesMinDuRenvoiDeTrame {
		return false
	}
	var origine uint64
	for i, e := range p.VueA.Jeu {
		ts, vu := c.vus[copieDeJeu{e.Rang, e.Texte, e.Destinataires, e.Sequence}]
		if !vu || p.TS < ts || p.TS-ts >= fenetreDuRenvoiDeTrame || (i > 0 && ts != origine) {
			return false
		}
		origine = ts
	}
	return true
}
