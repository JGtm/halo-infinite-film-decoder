package replay

// document_score_actions.go — LE SCORE PERSONNEL PAR ACTION, LU DANS LE FILM ET DATE SUR L'AXE DE FRAMES.
//
// CE QUE C'EST. Le script de mode du jeu envoie à chaque joueur une réponse de score personnel quand il agit
// (`killPersonalScore` 100, `assistPersonalScore` 50, `flagCaptureScore` 300, `betrayalPersonalScore` -100...) :
// un message de jeu (genre 82 de la vue A), que le film porte avec l'instant de sa trame et le masque de ses
// destinataires. Le document publie chaque réponse qui donne du score, au joueur qui la reçoit. Rien n'est
// calculé : aucun barème ne s'applique de l'extérieur, ce sont les points que le jeu a donnés.
//
// LE MESSAGE NE NOMME RIEN. Son rang (la position de sa réponse dans l'arbre des événements des scripts du
// mode) ne se lit qu'avec le groupe de scripts du film, que la table de `reponsesdejeu` résout en nom et en
// points. Un film dont le mode est inconnu de la table (`coverage.scoreActions.groupMissing`) ne publie rien.
//
// UNE ACTION N'EST PAS LE SCORE DE L'API. L'API compte des points que le jeu ne nomme pas en réponse
// (`withoutPoints` en compte les messages sans points, pas les points manquants), et un mode que ses auteurs
// ont écrit sans barème (Assaut) n'a aucune réponse qui donne du score : ce n'est pas un défaut, la
// couverture le dit. La somme des actions d'un joueur ne vaut son score personnel que là où toutes les
// réponses sont nommées.
//
// L'HORLOGE. L'instant d'un message est celui de sa trame sur l'horloge du film, la même que celle des
// positions : la frame est `(TS - origine) / pas`, l'origine étant le premier paquet de position. Une action
// avant la frame 0 ou après la dernière est comptée, jamais ramenée dans l'axe.
//
// LE DESTINATAIRE ET LA CIBLE. Le masque et le premier participant du texte sont des index de joueur du film,
// ceux du roster (`filmIndex`). Un index que le roster donne à un seul joueur humain devient son xuid ; un
// index sans xuid (un bot, un siège repris par un remplaçant) n'en donne pas, et l'action est comptée.
// La cible est le joueur que le libellé de la réponse nomme : l'ennemi tué (`%pe`), le joueur trahi (`%pa`,
// le premier participant : le destinataire est alors celui qui trahit).

import (
	"cmp"
	"slices"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/reponsesdejeu"
)

// ScoreAction est UNE réponse de score personnel : quand, qui la reçoit, laquelle, combien.
type ScoreAction struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// XUID est le joueur qui reçoit la réponse, en décimal — même forme que Track.XUID.
	XUID string `json:"xuid"`
	// Name est le nom de la réponse dans le script du mode (`killPersonalScore`) : une clé stable, jamais un
	// libellé d'interface.
	Name string `json:"name"`
	// Points est le score personnel que la réponse donne, négatif pour une pénalité.
	Points int `json:"points"`
	// Target est le joueur que le libellé de la réponse nomme, en décimal. Absent quand la réponse ne nomme
	// personne ou que le roster ne donne pas de xuid à ce joueur.
	Target string `json:"target,omitempty"`
}

// ScoreActionInput est ce que l'assemblage reçoit : les messages de jeu lus dans le film et les scripts qui
// en fixent le sens. Read est le témoin de lecture : faux, le calque n'est pas produit.
type ScoreActionInput struct {
	Read    bool
	Events  []grammar.EvenementDeJeuLu
	Stats   grammar.StatsEvenementsDeJeu
	Scripts []string
}

// ScoreActionCoverage dit ce que le calque a lu, publié et écarté.
//
// Un message qui donne du score se compte une fois par destinataire (`scored`) ; chacun est publié ou écarté
// pour une seule raison : `scored` = `published` + `noRecipient` + `noOrigin` + `beforeOrigin` + `afterEnd`.
type ScoreActionCoverage struct {
	// Group est le groupe de scripts du film dans la table des réponses ; vide quand GroupMissing.
	Group string `json:"group,omitempty"`
	// GroupMissing : aucun groupe de la table ne correspond aux scripts du film.
	GroupMissing bool `json:"groupMissing"`
	// Frames, ClosedFrames, StoppedFrames : les trames delta du film et ce que la lecture de leur vue A en a
	// fait, lue jusqu'au terminateur ou arrêtée par un message non lu (ceux qui suivent l'arrêt sont perdus).
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	// ViewAUnreadable : la lecture s'est arrêtée sans qu'un seul message ne se lise, aucune réponse n'est
	// connue de ce film.
	ViewAUnreadable bool `json:"viewAUnreadable"`
	// Messages est le nombre de messages de jeu lus, copies renvoyées exceptées ; Copies, celui des copies
	// que le jeu a renvoyées et que la lecture a retirées.
	Messages int `json:"messages"`
	Copies   int `json:"copies"`
	// WithoutPoints est le nombre de messages qui ne donnent pas de score : annonce, médaille, ligne du fil
	// des éliminations, ou réponse que la table ne connaît pas pour ce rang.
	WithoutPoints int `json:"withoutPoints"`
	// Scored est le nombre de (message, destinataire) qui donnent du score ; Published, ceux posés sur
	// l'axe ; les quatre suivants, ceux écartés (cf. le type).
	Scored       int `json:"scored"`
	Published    int `json:"published"`
	NoRecipient  int `json:"noRecipient"`
	NoOrigin     int `json:"noOrigin"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// NoTarget compte, parmi les actions publiées, celles dont le libellé nomme un joueur que le roster ne
	// nomme pas.
	NoTarget int `json:"noTarget"`
}

// axeDeFrames est l'axe sur lequel une action se pose : l'origine (microsecondes du film), le pas et le
// nombre de frames. Un pas nul est une origine non établie.
type axeDeFrames struct {
	origin, step uint64
	frames       int
}

// xuidsParIndex rend, pour chaque index de joueur du film que le roster donne à UN SEUL joueur humain, son
// xuid. Un index partagé par deux joueurs ou tenu par un bot n'y est pas.
func xuidsParIndex(roster []RosterEntry) map[int]string {
	out := map[int]string{}
	vus := map[int]int{}
	for _, e := range roster {
		vus[e.FilmIndex]++
		if !e.Bot && e.XUID != "" {
			out[e.FilmIndex] = e.XUID
		}
	}
	for i, n := range vus {
		if n > 1 {
			delete(out, i)
		}
	}
	return out
}

// buildScoreActions pose les réponses de score du film sur l'axe de frames. Les actions rendues sont triées
// par frame, joueur, nom et points.
func buildScoreActions(in ScoreActionInput, roster []RosterEntry, axe axeDeFrames) ([]ScoreAction, ScoreActionCoverage) {
	cov := ScoreActionCoverage{
		Frames: in.Stats.Trames, ClosedFrames: in.Stats.Terminees, StoppedFrames: in.Stats.Arretees,
		Messages: in.Stats.Messages, Copies: in.Stats.Copies,
		ViewAUnreadable: in.Stats.Messages == 0 && in.Stats.Arretees > 0,
	}
	groupe, ok := reponsesdejeu.GroupeDesScripts(in.Scripts)
	if !ok {
		cov.GroupMissing = true
		cov.WithoutPoints = len(in.Events)
		return nil, cov
	}
	cov.Group = groupe.Nom()
	joueurs := xuidsParIndex(roster)
	var out []ScoreAction
	for _, e := range in.Events {
		r, ok := groupe.Reponse(e.Rang, e.Texte, e.ATexte)
		if !ok {
			cov.WithoutPoints++
			continue
		}
		cible, nommee := cibleDeLaReponse(r, e, joueurs)
		for i := 0; i < grammar.ParticipantsDuFil; i++ {
			if e.Destinataires&(1<<uint(grammar.ParticipantsDuFil-1-i)) == 0 {
				continue
			}
			cov.Scored++
			t, refus := placerSurLAxe(e.TS, joueurs[i], axe)
			switch refus {
			case refusSansXUID:
				cov.NoRecipient++
			case refusSansOrigine:
				cov.NoOrigin++
			case refusAvantOrigine:
				cov.BeforeOrigin++
			case refusApresFin:
				cov.AfterEnd++
			default:
				out = append(out, ScoreAction{T: t, XUID: joueurs[i], Name: r.Nom, Points: r.Points, Target: cible})
				if nommee && cible == "" {
					cov.NoTarget++
				}
			}
		}
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b ScoreAction) int {
		return cmp.Or(cmp.Compare(a.T, b.T), strings.Compare(a.XUID, b.XUID), strings.Compare(a.Name, b.Name),
			cmp.Compare(a.Points, b.Points), strings.Compare(a.Target, b.Target))
	})
	return out, cov
}

// cibleDeLaReponse rend le xuid du joueur que le libellé de la réponse nomme (`%pe`, `%pa`) et si le libellé
// en nomme un.
func cibleDeLaReponse(r reponsesdejeu.Reponse, e grammar.EvenementDeJeuLu, joueurs map[int]string) (string, bool) {
	if !strings.Contains(r.Libelle, "%pa") && !strings.Contains(r.Libelle, "%pe") {
		return "", false
	}
	i, ok := e.PremierParticipant()
	if !ok {
		return "", true
	}
	return joueurs[i], true
}

// refusDePose dit pourquoi une action n'est pas posée sur l'axe ; refusAucun = posée.
type refusDePose int

const (
	refusAucun refusDePose = iota
	refusSansXUID
	refusSansOrigine
	refusAvantOrigine
	refusApresFin
)

// placerSurLAxe rend la frame de l'instant `ts` pour le joueur `xuid`, ou la raison de son refus.
func placerSurLAxe(ts uint64, xuid string, axe axeDeFrames) (int, refusDePose) {
	switch {
	case xuid == "":
		return 0, refusSansXUID
	case axe.step == 0:
		return 0, refusSansOrigine
	case ts < axe.origin:
		return 0, refusAvantOrigine
	}
	t := int((ts - axe.origin) / axe.step)
	if t >= axe.frames {
		return 0, refusApresFin
	}
	return t, refusAucun
}

// poserLesScoresParAction publie le calque : seulement quand la marche des trames a lu les messages de jeu,
// et APRES le roster dont il tire les xuids et la couverture qu'il complète.
func (a *assemblage) poserLesScoresParAction() {
	if !a.opt.ScoreActions.Read {
		return
	}
	actions, cov := buildScoreActions(a.opt.ScoreActions, a.doc.Roster, axeDeFrames{a.origin, a.step, a.doc.FrameCount})
	a.doc.ScoreActions = actions
	a.doc.Coverage.ScoreActions = &cov
}
