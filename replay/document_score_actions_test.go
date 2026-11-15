package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// masqueDesActions rend le masque des destinataires qui désigne les index de joueur donnés : le premier bit du masque
// est le participant 0.
func masqueDesActions(index ...int) uint32 {
	var m uint32
	for _, i := range index {
		m |= 1 << uint(grammar.ParticipantsDuFil-1-i)
	}
	return m
}

// rosterDesActions : trois humains aux index 0, 1 et 2, un bot à l'index 3, deux joueurs qui se partagent l'index 4.
func rosterDesActions() []RosterEntry {
	return []RosterEntry{
		{XUID: "2533491259474651", FilmIndex: 0},
		{XUID: "2534026679052315", FilmIndex: 1},
		{XUID: "2534144863254370", FilmIndex: 2},
		{FilmIndex: 3, Bot: true},
		{XUID: "2534999957182692", FilmIndex: 4},
		{XUID: "2535676564474216", FilmIndex: 4},
	}
}

// Rangs et mots texte de `slayerevents.lua` (reponsesdejeu/data/reponses.tsv).
const (
	rangFrag, texteFrag                             = 78, 0x3d097626
	rangTrahison, texteTrahison                     = 118, 0x365be6b1
	rangAssistance, rangSuicide                     = 49, 31
	rangInconnuDuGroupe                             = 5
	scriptsDeLArene                                 = "slayerevents.lua"
	origineDesActions, pasDesActions, finDesActions = 1_000_000, 16_000, 100
)

// messageDesActions écrit un message de jeu du groupe de test à l'instant `ts`.
func messageDesActions(ts uint64, rang uint32, texte uint32, aTexte bool, victime int8, masque uint32) grammar.EvenementDeJeuLu {
	return grammar.EvenementDeJeuLu{TS: ts, Rang: rang, Texte: texte, ATexte: aTexte, Destinataires: masque, Participants: [2]int8{victime, -1}}
}

// TestBuildScoreActions : chaque message à points se pose une fois par destinataire, au xuid que le roster donne à
// son index, à la frame `(TS - origine) / pas`, avec la cible que son libellé nomme ; chaque refus a son compteur.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : poser une seule copie par message au lieu d'une par destinataire ; lire le
// destinataire dans le premier participant ; nommer le destinataire plutôt que la victime pour une trahison ; accepter
// l'index d'un bot ou d'un siège partagé ; compter une action avant la frame 0 ou après la dernière ; ne pas trier.
func TestBuildScoreActions(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	cas := []struct {
		nom     string
		scripts []string
		axe     axeDeFrames
		events  []grammar.EvenementDeJeuLu
		actions []ScoreAction
		cov     ScoreActionCoverage
	}{
		{
			"frag : le destinataire reçoit, la victime est la cible", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions+3*pasDesActions+5, rangFrag, texteFrag, true, 1, masqueDesActions(0))},
			[]ScoreAction{{T: 3, XUID: "2533491259474651", Name: "killPersonalScore", Points: 100, Target: "2534026679052315"}},
			ScoreActionCoverage{Scored: 1, Published: 1},
		},
		{
			"trahison : le destinataire est le traître, le premier participant est le trahi", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions, rangTrahison, texteTrahison, true, 2, masqueDesActions(1))},
			[]ScoreAction{{T: 0, XUID: "2534026679052315", Name: "betrayalPersonalScore", Points: -100, Target: "2534144863254370"}},
			ScoreActionCoverage{Scored: 1, Published: 1},
		},
		{
			"un message pour deux destinataires, triés par frame puis joueur", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(origineDesActions+9*pasDesActions, rangAssistance, 0, false, -1, masqueDesActions(2, 0)),
				messageDesActions(origineDesActions+2*pasDesActions, rangSuicide, 0, false, -1, masqueDesActions(1)),
			},
			[]ScoreAction{
				{T: 2, XUID: "2534026679052315", Name: "playerSuicidePersonalScore", Points: -100},
				{T: 9, XUID: "2533491259474651", Name: "assistPersonalScore", Points: 50},
				{T: 9, XUID: "2534144863254370", Name: "assistPersonalScore", Points: 50},
			},
			ScoreActionCoverage{Scored: 3, Published: 3},
		},
		{
			"un bot, un siège partagé et un index inconnu ne reçoivent pas", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions, rangAssistance, 0, false, -1, masqueDesActions(3, 4, 20, 0))},
			[]ScoreAction{{T: 0, XUID: "2533491259474651", Name: "assistPersonalScore", Points: 50}},
			ScoreActionCoverage{Scored: 4, Published: 1, NoRecipient: 3},
		},
		{
			"une victime que le roster ne nomme pas : action publiée sans cible, et comptée", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions, rangFrag, texteFrag, true, 3, masqueDesActions(0))},
			[]ScoreAction{{T: 0, XUID: "2533491259474651", Name: "killPersonalScore", Points: 100}},
			ScoreActionCoverage{Scored: 1, Published: 1, NoTarget: 1},
		},
		{
			"avant l'origine, à la dernière frame, après la fin", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(origineDesActions-1, rangSuicide, 0, false, -1, masqueDesActions(0)),
				messageDesActions(origineDesActions+(finDesActions-1)*pasDesActions+pasDesActions-1, rangSuicide, 0, false, -1, masqueDesActions(0)),
				messageDesActions(origineDesActions+finDesActions*pasDesActions, rangSuicide, 0, false, -1, masqueDesActions(0)),
			},
			[]ScoreAction{{T: finDesActions - 1, XUID: "2533491259474651", Name: "playerSuicidePersonalScore", Points: -100}},
			ScoreActionCoverage{Scored: 3, Published: 1, BeforeOrigin: 1, AfterEnd: 1},
		},
		{
			"axe sans pas : rien ne se pose", []string{scriptsDeLArene}, axeDeFrames{origin: origineDesActions, frames: finDesActions},
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions, rangSuicide, 0, false, -1, masqueDesActions(0))},
			nil,
			ScoreActionCoverage{Scored: 1, NoOrigin: 1},
		},
		{
			"réponse sans points, texte d'une autre réponse : aucune action", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(origineDesActions, rangInconnuDuGroupe, 0, false, -1, masqueDesActions(0)),
				messageDesActions(origineDesActions, rangFrag, texteTrahison, true, 1, masqueDesActions(0)),
			},
			nil,
			ScoreActionCoverage{WithoutPoints: 2},
		},
		{
			"mode inconnu de la table : rien n'est nommé", []string{"engine.lua"}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(origineDesActions, rangFrag, texteFrag, true, 1, masqueDesActions(0))},
			nil,
			ScoreActionCoverage{GroupMissing: true, WithoutPoints: 1},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			in := ScoreActionInput{Read: true, Events: c.events, Scripts: c.scripts}
			actions, cov := buildScoreActions(in, rosterDesActions(), c.axe)
			if !slices.Equal(actions, c.actions) {
				t.Errorf("actions %+v, attendu %+v", actions, c.actions)
			}
			if !c.cov.GroupMissing {
				c.cov.Group = scriptsDeLArene
			}
			if cov != c.cov {
				t.Errorf("couverture %+v, attendu %+v", cov, c.cov)
			}
		})
	}
}

// TestLaCouvertureDitLaVueAIllisible : une vue A arrêtée sans qu'un seul message se lise est illisible ; arrêtée
// après des messages lus, ou lue jusqu'au bout sans message, elle ne l'est pas ; les comptes du canal sont repris.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : déclarer illisible toute vue A arrêtée ; ne pas reprendre les comptes.
func TestLaCouvertureDitLaVueAIllisible(t *testing.T) {
	cas := []struct {
		nom        string
		stats      grammar.StatsEvenementsDeJeu
		illisible  bool
		copiesDite int
	}{
		{"arrêtée sans message", grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 4, Arretees: 6}, true, 0},
		{"arrêtée après des messages", grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 4, Arretees: 6, Messages: 3, Copies: 2}, false, 2},
		{"lue jusqu'au bout sans message", grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10}, false, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, cov := buildScoreActions(ScoreActionInput{Read: true, Stats: c.stats, Scripts: []string{scriptsDeLArene}}, nil,
				axeDeFrames{origin: 0, step: pasDesActions, frames: finDesActions})
			if cov.ViewAUnreadable != c.illisible || cov.Frames != c.stats.Trames || cov.ClosedFrames != c.stats.Terminees ||
				cov.StoppedFrames != c.stats.Arretees || cov.Messages != c.stats.Messages || cov.Copies != c.copiesDite {
				t.Errorf("couverture %+v pour les comptes %+v", cov, c.stats)
			}
		})
	}
}
