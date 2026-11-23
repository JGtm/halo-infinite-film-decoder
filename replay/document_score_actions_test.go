package replay

import (
	"reflect"
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

// presenceDesActions : un intervalle de présence certain, de `from` à `to`.
func presenceDesActions(from, to int) []PresenceInterval {
	return []PresenceInterval{{From: from, To: to}}
}

// placeDeTest rend un pointeur vers l'index `i` : la forme d'une place dans les actions et les éliminations.
func placeDeTest(i int) *int { return &i }

// Xuids fictifs evidents (9990...), dans l ordre des lettres : les tests trient par xuid.
const (
	xuidA = "9990000000000001"
	xuidB = "9990000000000002"
	xuidC = "9990000000000003"
	xuidD = "9990000000000004"
	xuidE = "9990000000000005"
	xuidF = "9990000000000006"
	xuidG = "9990000000000007"
)

// rosterDesActions : trois humains présents tout le film aux index 0, 1 et 2, un bot à l'index 3, un humain D
// remplacé par E sur l'index 4 (D jusqu'à la frame 49, E à partir de la 50), un humain F parti à la frame 10 sur
// l'index 6 (le bot qui reprend l'index n'est pas au roster, comme `buildRoster` l'écarte), un humain G sans
// aucune présence sur l'index 7.
func rosterDesActions() []RosterEntry {
	return []RosterEntry{
		{XUID: xuidA, FilmIndex: 0, Presence: presenceDesActions(0, finDesActions-1)},
		{XUID: xuidB, FilmIndex: 1, Presence: presenceDesActions(0, finDesActions-1)},
		{XUID: xuidC, FilmIndex: 2, Presence: presenceDesActions(0, finDesActions-1)},
		{FilmIndex: 3, Bot: true, Presence: presenceDesActions(0, finDesActions-1)},
		{XUID: xuidD, FilmIndex: 4, Presence: presenceDesActions(0, 49)},
		{XUID: xuidE, FilmIndex: 4, Presence: presenceDesActions(50, finDesActions-1)},
		{XUID: xuidF, FilmIndex: 6, Presence: presenceDesActions(0, 10)},
		{XUID: xuidG, FilmIndex: 7},
	}
}

// Rangs et mots texte de `slayerevents.lua` (reponsesdejeu/data/reponses.tsv).
const (
	rangFrag, texteFrag                             = 78, 0x3d097626
	rangTrahison, texteTrahison                     = 118, 0x365be6b1
	rangAssistance, rangSuicide                     = 49, 31
	rangInconnuDuGroupe, rangAnnonce, rangZeroPoint = 5, 16, 266
	scriptsDeLArene                                 = "slayerevents.lua"
	origineDesActions, pasDesActions, finDesActions = 1_000_000, 16_000, 100
)

// instantDeLaFrame rend un instant du film qui tombe dans la frame `t`.
func instantDeLaFrame(t int) uint64 { return origineDesActions + uint64(t)*pasDesActions + 5 }

// messageDesActions écrit un message de jeu du groupe de test à l'instant `ts`.
func messageDesActions(ts uint64, rang uint32, texte uint32, aTexte bool, victime int8, masque uint32) grammar.EvenementDeJeuLu {
	return grammar.EvenementDeJeuLu{TS: ts, Rang: rang, Texte: texte, ATexte: aTexte, Destinataires: masque, Participants: [2]int8{victime, -1}}
}

// entreeDesActions : les messages lus, sur un film dont toutes les trames se lisent.
func entreeDesActions(scripts []string, events ...grammar.EvenementDeJeuLu) ScoreActionInput {
	return ScoreActionInput{Read: true, Events: events, Scripts: scripts,
		Stats: grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: len(events)}}
}

// TestBuildScoreActions : chaque message à points se pose une fois par destinataire, au xuid que le roster donne à
// son index À L'INSTANT du message, à la frame `(TS - origine) / pas`, avec la cible que son libellé nomme ; chaque
// refus a son compteur, et `complete` n'est vrai que si rien n'est perdu.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : poser une seule copie par message au lieu d'une par destinataire ; lire le
// destinataire dans le premier participant ; nommer le destinataire plutôt que la victime pour une trahison ; ignorer
// la présence (un humain parti reçoit les messages du bot qui prend son index) ; accepter l'index d'un bot ou d'un
// humain sans présence ; compter une action avant la frame 0 ou après la dernière ; publier une réponse à zéro point ;
// ne pas trier ; déclarer complet un film qui a perdu une action ou un rang.
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
			[]grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 1, masqueDesActions(0))},
			[]ScoreAction{{T: 3, XUID: xuidA, Name: "killPersonalScore", Points: 100, Target: xuidB}},
			ScoreActionCoverage{ScoreMessages: 1, Scored: 1, Published: 1, Complete: true},
		},
		{
			"trahison : le destinataire est le traître, le premier participant est le trahi", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(0), rangTrahison, texteTrahison, true, 2, masqueDesActions(1))},
			[]ScoreAction{{T: 0, XUID: xuidB, Name: "betrayalPersonalScore", Points: -100, Target: xuidC}},
			ScoreActionCoverage{ScoreMessages: 1, Scored: 1, Published: 1, Complete: true},
		},
		{
			"un message pour deux destinataires, triés par frame puis joueur", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(instantDeLaFrame(9), rangAssistance, 0, false, -1, masqueDesActions(2, 0)),
				messageDesActions(instantDeLaFrame(2), rangSuicide, 0, false, -1, masqueDesActions(1)),
			},
			[]ScoreAction{
				{T: 2, XUID: xuidB, Name: "playerSuicidePersonalScore", Points: -100},
				{T: 9, XUID: xuidA, Name: "assistPersonalScore", Points: 50},
				{T: 9, XUID: xuidC, Name: "assistPersonalScore", Points: 50},
			},
			ScoreActionCoverage{ScoreMessages: 2, Scored: 3, Published: 3, Complete: true},
		},
		{
			"un bot est posé à sa place ; un index inconnu et un humain sans présence ne reçoivent pas", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(3), rangAssistance, 0, false, -1, masqueDesActions(3, 20, 7, 0))},
			[]ScoreAction{
				{T: 3, Place: placeDeTest(3), Name: "assistPersonalScore", Points: 50},
				{T: 3, XUID: xuidA, Name: "assistPersonalScore", Points: 50},
			},
			ScoreActionCoverage{ScoreMessages: 1, Scored: 4, Published: 2, NoRecipient: 1, NotPresent: 1, BotRecipient: 1},
		},
		{
			"un remplaçant sur le même index : à chacun son intervalle", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(instantDeLaFrame(10), rangAssistance, 0, false, -1, masqueDesActions(4)),
				messageDesActions(instantDeLaFrame(60), rangAssistance, 0, false, -1, masqueDesActions(4)),
			},
			[]ScoreAction{
				{T: 10, XUID: xuidD, Name: "assistPersonalScore", Points: 50},
				{T: 60, XUID: xuidE, Name: "assistPersonalScore", Points: 50},
			},
			ScoreActionCoverage{ScoreMessages: 2, Scored: 2, Published: 2, Complete: true},
		},
		{
			"un humain parti : ses messages d'avant lui reviennent, ceux d'après (le bot de son index) non", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(instantDeLaFrame(10), rangAssistance, 0, false, -1, masqueDesActions(6)),
				messageDesActions(instantDeLaFrame(11), rangAssistance, 0, false, -1, masqueDesActions(6)),
				messageDesActions(instantDeLaFrame(80), rangAssistance, 0, false, -1, masqueDesActions(6)),
			},
			[]ScoreAction{{T: 10, XUID: xuidF, Name: "assistPersonalScore", Points: 50}},
			ScoreActionCoverage{ScoreMessages: 3, Scored: 3, Published: 1, NotPresent: 2},
		},
		{
			"une victime bot est posée à sa place ; une victime que le roster ne résout pas : action publiée sans cible, et comptée", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 3, masqueDesActions(0)),
				messageDesActions(instantDeLaFrame(20), rangFrag, texteFrag, true, 6, masqueDesActions(0)),
			},
			[]ScoreAction{
				{T: 3, XUID: xuidA, Name: "killPersonalScore", Points: 100, TargetPlace: placeDeTest(3)},
				{T: 20, XUID: xuidA, Name: "killPersonalScore", Points: 100},
			},
			ScoreActionCoverage{ScoreMessages: 2, Scored: 2, Published: 2, NoTarget: 1, BotTarget: 1, Complete: true},
		},
		{
			"avant l'origine, à la dernière frame, après la fin", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(origineDesActions-1, rangSuicide, 0, false, -1, masqueDesActions(0)),
				messageDesActions(instantDeLaFrame(finDesActions-1), rangSuicide, 0, false, -1, masqueDesActions(0)),
				messageDesActions(origineDesActions+finDesActions*pasDesActions, rangSuicide, 0, false, -1, masqueDesActions(0)),
			},
			[]ScoreAction{{T: finDesActions - 1, XUID: xuidA, Name: "playerSuicidePersonalScore", Points: -100}},
			ScoreActionCoverage{ScoreMessages: 3, Scored: 3, Published: 1, BeforeOrigin: 1, AfterEnd: 1},
		},
		{
			"axe sans pas : rien ne se pose", []string{scriptsDeLArene}, axeDeFrames{origin: origineDesActions, frames: finDesActions},
			[]grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(0), rangSuicide, 0, false, -1, masqueDesActions(0))},
			nil,
			ScoreActionCoverage{ScoreMessages: 1, Scored: 1, NoOrigin: 1},
		},
		{
			"annonce, zéro point, rang hors de l'arbre, texte d'une autre réponse : aucune action", []string{scriptsDeLArene}, axe,
			[]grammar.EvenementDeJeuLu{
				messageDesActions(instantDeLaFrame(0), rangAnnonce, 0, false, -1, masqueDesActions(0)),
				messageDesActions(instantDeLaFrame(0), rangZeroPoint, 0, false, -1, masqueDesActions(0)),
				messageDesActions(instantDeLaFrame(0), rangInconnuDuGroupe, 0, false, -1, masqueDesActions(0)),
				messageDesActions(instantDeLaFrame(0), rangFrag, texteTrahison, true, 1, masqueDesActions(0)),
			},
			nil,
			ScoreActionCoverage{Unrelated: 1, NamedWithoutPoints: 1, OutsideTree: 1, UnconfirmedRank: 1},
		},
		{
			"mode inconnu de la table : rien n'est nommé", []string{"engine.lua"}, axe,
			[]grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(0), rangFrag, texteFrag, true, 1, masqueDesActions(0))},
			nil,
			ScoreActionCoverage{GroupMissing: true, OutsideTree: 1},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			actions, cov := buildScoreActions(entreeDesActions(c.scripts, c.events...), rosterDesActions(), c.axe, nil, "")
			if !reflect.DeepEqual(actions, c.actions) {
				t.Errorf("actions %+v, attendu %+v", actions, c.actions)
			}
			c.cov.Frames, c.cov.ClosedFrames, c.cov.Messages = 10, 10, len(c.events)
			if !c.cov.GroupMissing {
				c.cov.Group = scriptsDeLArene
			}
			if cov != c.cov {
				t.Errorf("couverture %+v, attendu %+v", cov, c.cov)
			}
		})
	}
}

// TestLeVerdictCompletExigeUneLectureSansPerte : `complete` n'est vrai que si aucune trame n'est arrêtée, la vue A se
// lit, le groupe est connu, aucun rang de l'arbre n'est non confirmé et chaque action à points est publiée ; un rang hors de l'arbre n'y change rien ; les comptes du canal sont
// repris tels quels, et une vue A arrêtée sans qu'un seul message se lise est illisible.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas regarder les trames arrêtées ; ne pas regarder les rangs non confirmés ; compter un rang hors de l'arbre comme un manque ;
// déclarer complet un film sans trame ; déclarer illisible toute vue A arrêtée ; ne pas reprendre les comptes.
func TestLeVerdictCompletExigeUneLectureSansPerte(t *testing.T) {
	frag := messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 1, masqueDesActions(0))
	horsArbre := messageDesActions(instantDeLaFrame(3), rangInconnuDuGroupe, 0, false, -1, masqueDesActions(0))
	inconnu := messageDesActions(instantDeLaFrame(3), rangFrag, texteTrahison, true, 1, masqueDesActions(0))
	perdu := messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 1, masqueDesActions(20))
	cas := []struct {
		nom       string
		scripts   []string
		stats     grammar.StatsEvenementsDeJeu
		events    []grammar.EvenementDeJeuLu
		complet   bool
		illisible bool
	}{
		{"lu jusqu'au bout", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: 1}, []grammar.EvenementDeJeuLu{frag}, true, false},
		{"une trame arrêtée après des messages", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 4, Arretees: 6, Messages: 1, Copies: 2},
			[]grammar.EvenementDeJeuLu{frag}, false, false},
		{"arrêtée sans message", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 4, Arretees: 6}, nil, false, true},
		{"un rang de l'arbre que le message ne confirme pas", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: 2}, []grammar.EvenementDeJeuLu{frag, inconnu}, false, false},
		{"un rang hors de l'arbre n'est pas un manque", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: 2}, []grammar.EvenementDeJeuLu{frag, horsArbre}, true, false},
		{"une action perdue (destinataire sans humain)", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: 2},
			[]grammar.EvenementDeJeuLu{frag, perdu}, false, false},
		{"groupe absent", []string{"engine.lua"}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10, Messages: 1}, []grammar.EvenementDeJeuLu{frag}, false, false},
		{"aucune trame", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{}, nil, false, false},
		{"lue jusqu'au bout sans message", []string{scriptsDeLArene}, grammar.StatsEvenementsDeJeu{Trames: 10, Terminees: 10}, nil, true, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			in := ScoreActionInput{Read: true, Events: c.events, Stats: c.stats, Scripts: c.scripts}
			_, cov := buildScoreActions(in, rosterDesActions(), axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}, nil, "")
			if cov.Complete != c.complet || cov.ViewAUnreadable != c.illisible || cov.Frames != c.stats.Trames ||
				cov.ClosedFrames != c.stats.Terminees || cov.StoppedFrames != c.stats.Arretees || cov.Messages != c.stats.Messages ||
				cov.Copies != c.stats.Copies {
				t.Errorf("couverture %+v pour les comptes %+v : complet %v, illisible %v attendus", cov, c.stats, c.complet, c.illisible)
			}
		})
	}
}
