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
// mode) ne se lit qu'avec le groupe de scripts du film, que la table de `reponsesdejeu` résout en nom, genre
// et points. Un film dont le mode est inconnu de la table (`coverage.scoreActions.groupMissing`) ne publie rien.
//
// QUATRE SORTES DE MESSAGE QUI NE DONNENT PAS DE POINTS, ET LA COUVERTURE LES SEPARE :
//   - `unrelated` : l'arbre nomme la réponse et elle ne donne pas de score (annonce, médaille, ligne du fil) ;
//   - `namedWithoutPoints` : l'arbre nomme une réponse de score que le jeu paie zéro (le détournement d'un
//     Phantom) : elle n'est pas publiée ;
//   - `outsideTree` : le rang n'est pas dans l'arbre. L'arbre liste les événements qui sont des réponses : un rang
//     hors de lui est le plus souvent un événement qui n'en est pas une (la majorité des messages d'un film), ou une
//     réponse que la table ne connaît pas (un mode que ses auteurs ont écrit eux-mêmes) ;
//   - `unconfirmedRank` : l'arbre place une réponse à ce rang, mais le message ne la confirme pas (texte d'une autre
//     réponse, rang incertain sans son texte). Le score qu'il porte, s'il en porte, est perdu.
//
// `complete` EST LE VERDICT : la somme des points d'un joueur ne vaut son score personnel que s'il est vrai, c'est-à-dire
// si aucune trame n'est arrêtée, si la vue A se lit, si le groupe de scripts est connu, si aucun rang n'est non confirmé,
// si chaque action à points a été posée et si le tableau du film ne montre aucun point que le calque ne publie pas (le
// tableau ne recule pas : les pénalités n'y entrent pas ; un tableau en dessous n'accuse pas la lecture). Faux, le consommateur n'affiche pas de score par action (« données
// insuffisantes »). Vrai est NECESSAIRE et pas SUFFISANT : la table ne dit pas ce qu'un mode écrit par ses auteurs
// (Assaut) paie hors de son arbre, et ses points sont ceux du jeu actuel (les véhicules d'un BTB de HI_1_11_0 se paient
// autrement), ce que le tableau ne révèle pas toujours (points de véhicule que ni le tableau ni le score personnel ne comptent).
//
// L'HORLOGE. L'instant d'un message est celui de sa trame sur l'horloge du film, la même que celle des
// positions : la frame est `(TS - origine) / pas`, l'origine étant le premier paquet de position. Une action
// avant la frame 0 ou après la dernière est comptée, jamais ramenée dans l'axe.
//
// LE DESTINATAIRE ET LA CIBLE. Le masque et le premier participant du texte sont des index de joueur du film,
// ceux du roster (`filmIndex`). Un index se résout en xuid À L'INSTANT DU MESSAGE : parmi les humains du roster
// qui le portent, celui dont la présence (`presence`) couvre la frame. Un humain parti garde son index dans le
// roster alors qu'un bot déclaré au même index n'y figure pas (`buildRoster` l'écarte) : un message d'après
// son départ ne lui revient donc pas, il est compté (`notPresent`). Un index sans humain du roster (un bot,
// un siège que le film ne nomme pas) n'a pas de xuid : `noRecipient`. La cible est le joueur que le libellé
// de la réponse nomme : l'ennemi tué (`%pe`), le joueur trahi (`%pa`, le premier participant : le destinataire
// est alors celui qui trahit).

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
	// personne ou que le roster ne résout pas ce joueur à cet instant.
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
// Chaque message lu est l'un de cinq : `scoreMessages` (réponse à points), `namedWithoutPoints`, `unrelated`,
// `outsideTree` ou `unconfirmedRank` ; leur somme est `messages`. Un message à points se compte une fois par destinataire
// (`scored`) ; chacun est publié ou écarté pour une seule raison : `scored` = `published` + `noRecipient` +
// `notPresent` + `noOrigin` + `beforeOrigin` + `afterEnd`.
type ScoreActionCoverage struct {
	// Complete : la lecture des actions à points est entière. Vrai seulement si aucune trame n'est arrêtée, la
	// vue A se lit, le groupe de scripts est connu, aucun rang n'est non confirmé et chaque action à points est
	// publiée et aucun joueur n'a un manque (ScoreboardMissing) : le tableau du film ne recule pas, donc les pénalités n'y
	// entrent pas, et un tableau en dessous des points publiés (ScoreboardBehind) n'accuse pas la lecture. Nécessaire et non
	// suffisant pour que la somme des points d'un joueur vaille son score personnel : les points viennent de la table des
	// réponses, qui peut différer du jeu (actions de véhicule des builds anciens) ou ne rien savoir d'un mode écrit hors de
	// l'arbre (cf. l'en-tête du fichier).
	Complete bool `json:"complete"`
	// Group est le groupe de scripts du film dans la table des réponses ; vide quand GroupMissing.
	Group string `json:"group,omitempty"`
	// GroupMissing : aucun groupe de la table ne correspond aux scripts du film.
	GroupMissing bool `json:"groupMissing"`
	// ScoreboardCompared est le nombre de joueurs comparés au tableau du film (une action publiée ou une ligne du
	// tableau). ScoreboardMissing : ceux dont le dernier score du tableau dépasse la somme de leurs actions à points
	// POSITIFS publiées (le tableau montre des points que le calque ne publie pas) ; ScoreboardBehind : ceux dont il est
	// en dessous (le tableau est en retard sur les points publiés). Un joueur sans total au tableau y vaut zéro. Les
	// trois valent zéro quand le film ne publie aucun joueur du tableau (`scoreTimeline.players` absent).
	ScoreboardCompared int `json:"scoreboardCompared"`
	ScoreboardMissing  int `json:"scoreboardMissing"`
	ScoreboardBehind   int `json:"scoreboardBehind"`
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
	// ScoreMessages : messages dont la réponse donne des points. Unrelated : annonces, médailles, lignes du fil.
	// NamedWithoutPoints : réponses de score que le jeu paie zéro (non publiées). OutsideTree : rangs hors de l'arbre.
	// UnconfirmedRank : rangs de l'arbre que le message ne confirme pas.
	ScoreMessages      int `json:"scoreMessages"`
	Unrelated          int `json:"unrelated"`
	NamedWithoutPoints int `json:"namedWithoutPoints"`
	OutsideTree        int `json:"outsideTree"`
	UnconfirmedRank    int `json:"unconfirmedRank"`
	// Scored est le nombre de (message, destinataire) à points ; Published, ceux posés sur l'axe ; les cinq
	// suivants, ceux écartés : sans humain du roster à cet index, humain parti à l'instant du message, origine non
	// établie, avant la frame 0, après la dernière.
	Scored       int `json:"scored"`
	Published    int `json:"published"`
	NoRecipient  int `json:"noRecipient"`
	NotPresent   int `json:"notPresent"`
	NoOrigin     int `json:"noOrigin"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// NoTarget compte les messages à points dont le libellé nomme un joueur que le roster ne
	// résout pas à cet instant.
	NoTarget int `json:"noTarget"`
}

// axeDeFrames est l'axe sur lequel une action se pose : l'origine (microsecondes du film), le pas et le
// nombre de frames. Un pas nul est une origine non établie.
type axeDeFrames struct {
	origin, step uint64
	frames       int
}

// refusDePose dit pourquoi une action n'est pas posée ; refusAucun = posée.
type refusDePose int

const (
	refusAucun refusDePose = iota
	refusSansOrigine
	refusAvantOrigine
	refusApresFin
)

// frameDeLAction rend la frame de l'instant `ts` (microsecondes du film), ou la raison de son refus.
func frameDeLAction(ts uint64, axe axeDeFrames) (int, refusDePose) {
	switch {
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

// humainsParIndex rend, pour chaque index de joueur du film, les humains du roster qui le portent : plusieurs
// quand un joueur est remplacé sur le même index.
func humainsParIndex(roster []RosterEntry) map[int][]RosterEntry {
	out := map[int][]RosterEntry{}
	for _, e := range roster {
		if !e.Bot && e.XUID != "" {
			out[e.FilmIndex] = append(out[e.FilmIndex], e)
		}
	}
	return out
}

// presentALaFrame dit que l'entrée tient sa place à la frame `t` : un de ses intervalles de présence la couvre, de
// `from` à `toMax` (à défaut `to`). Une entrée sans intervalle n'est présente à aucun instant.
func presentALaFrame(e RosterEntry, t int) bool {
	for _, p := range e.Presence {
		fin := p.To
		if p.ToMax != nil && *p.ToMax > fin {
			fin = *p.ToMax
		}
		if p.From <= t && t <= fin {
			return true
		}
	}
	return false
}

// issueDuDestinataire dit ce que devient un index de joueur à une frame.
type issueDuDestinataire int

const (
	destinataireSansHumain issueDuDestinataire = iota // aucun humain du roster ne porte l'index
	destinataireAbsent                                // des humains le portent, aucun n'est présent à la frame
	destinataireAmbigu                                // plusieurs humains présents à la frame : aucun n'est choisi
	destinataireResolu                                // un seul humain présent
)

// xuidALaFrame résout l'index de joueur `i` à la frame `t`.
func xuidALaFrame(humains map[int][]RosterEntry, i, t int) (string, issueDuDestinataire) {
	cands := humains[i]
	if len(cands) == 0 {
		return "", destinataireSansHumain
	}
	var xuid string
	n := 0
	for _, e := range cands {
		if presentALaFrame(e, t) {
			xuid = e.XUID
			n++
		}
	}
	switch n {
	case 0:
		return "", destinataireAbsent
	case 1:
		return xuid, destinataireResolu
	}
	return "", destinataireAmbigu
}

// ciblePossible dit que le libellé de la réponse nomme un joueur (`%pa`, `%pe`).
func ciblePossible(r reponsesdejeu.Reponse) bool {
	return strings.Contains(r.Libelle, "%pa") || strings.Contains(r.Libelle, "%pe")
}

// classerLeMessage compte le message dans la couverture et rend la réponse qui donne des points, ou faux.
func classerLeMessage(groupe *reponsesdejeu.Groupe, e grammar.EvenementDeJeuLu, cov *ScoreActionCoverage) (reponsesdejeu.Reponse, bool) {
	r, nature := groupe.Reponse(e.Rang, e.Texte, e.ATexte)
	switch {
	case nature == reponsesdejeu.NatureHorsArbre:
		cov.OutsideTree++
	case nature == reponsesdejeu.NatureNonConfirmee:
		cov.UnconfirmedRank++
	case nature == reponsesdejeu.NatureSansScore:
		cov.Unrelated++
	case r.Points == 0:
		cov.NamedWithoutPoints++
	default:
		cov.ScoreMessages++
		return r, true
	}
	return reponsesdejeu.Reponse{}, false
}

// destinatairesDuMessage pose le message à points `e` chez chacun de ses destinataires et rend les actions posées.
func destinatairesDuMessage(e grammar.EvenementDeJeuLu, r reponsesdejeu.Reponse, humains map[int][]RosterEntry, axe axeDeFrames,
	cov *ScoreActionCoverage) []ScoreAction {
	t, refus := frameDeLAction(e.TS, axe)
	cible := ""
	if refus == refusAucun && ciblePossible(r) {
		cible = cibleDeLAction(e, humains, t, cov)
	}
	var out []ScoreAction
	for i := 0; i < grammar.ParticipantsDuFil; i++ {
		if e.Destinataires&(1<<uint(grammar.ParticipantsDuFil-1-i)) == 0 {
			continue
		}
		cov.Scored++
		xuid, res := "", destinataireResolu
		if refus == refusAucun {
			xuid, res = xuidALaFrame(humains, i, t)
		}
		switch {
		case refus == refusSansOrigine:
			cov.NoOrigin++
		case refus == refusAvantOrigine:
			cov.BeforeOrigin++
		case refus == refusApresFin:
			cov.AfterEnd++
		case res == destinataireSansHumain || res == destinataireAmbigu:
			cov.NoRecipient++
		case res == destinataireAbsent:
			cov.NotPresent++
		default:
			out = append(out, ScoreAction{T: t, XUID: xuid, Name: r.Nom, Points: r.Points, Target: cible})
		}
	}
	return out
}

// buildScoreActions pose les réponses de score du film sur l'axe de frames. Les actions rendues sont triées
// par frame, joueur, nom, points et cible.
func buildScoreActions(in ScoreActionInput, roster []RosterEntry, axe axeDeFrames, tableau *ScoreTimeline) ([]ScoreAction, ScoreActionCoverage) {
	cov := ScoreActionCoverage{
		Frames: in.Stats.Trames, ClosedFrames: in.Stats.Terminees, StoppedFrames: in.Stats.Arretees,
		Messages: in.Stats.Messages, Copies: in.Stats.Copies,
		ViewAUnreadable: in.Stats.Messages == 0 && in.Stats.Arretees > 0,
	}
	groupe, ok := reponsesdejeu.GroupeDesScripts(in.Scripts)
	if !ok {
		cov.GroupMissing = true
		cov.OutsideTree = len(in.Events)
		return nil, cov
	}
	cov.Group = groupe.Nom()
	humains := humainsParIndex(roster)
	var out []ScoreAction
	for _, e := range in.Events {
		if r, aPoints := classerLeMessage(groupe, e, &cov); aPoints {
			out = append(out, destinatairesDuMessage(e, r, humains, axe, &cov)...)
		}
	}
	cov.Published = len(out)
	comparerAuTableau(out, scoreFinalDuTableau(tableau), &cov)
	cov.Complete = cov.Frames > 0 && cov.StoppedFrames == 0 && !cov.ViewAUnreadable && cov.UnconfirmedRank == 0 &&
		cov.Published == cov.Scored && cov.ScoreboardMissing == 0
	slices.SortStableFunc(out, func(a, b ScoreAction) int {
		return cmp.Or(cmp.Compare(a.T, b.T), strings.Compare(a.XUID, b.XUID), strings.Compare(a.Name, b.Name),
			cmp.Compare(a.Points, b.Points), strings.Compare(a.Target, b.Target))
	})
	return out, cov
}

// cibleDeLAction résout le joueur que le libellé de la réponse nomme (le premier participant du texte) à la
// frame `t`, ou "" et le compte `noTarget`.
func cibleDeLAction(e grammar.EvenementDeJeuLu, humains map[int][]RosterEntry, t int, cov *ScoreActionCoverage) string {
	if i, ok := e.PremierParticipant(); ok {
		if xuid, res := xuidALaFrame(humains, i, t); res == destinataireResolu {
			return xuid
		}
	}
	cov.NoTarget++
	return ""
}

// poserLesScoresParAction publie le calque : seulement quand la marche des trames a lu les messages de jeu,
// et APRES le roster (et ses présences) dont il tire les xuids, et la couverture qu'il complète.
func (a *assemblage) poserLesScoresParAction() {
	if !a.opt.ScoreActions.Read {
		return
	}
	actions, cov := buildScoreActions(a.opt.ScoreActions, a.doc.Roster, axeDeFrames{a.origin, a.step, a.doc.FrameCount}, a.doc.ScoreTimeline)
	a.doc.ScoreActions = actions
	a.doc.Coverage.ScoreActions = &cov
}
