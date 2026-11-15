package replay

// document_medals.go — LES MEDAILLES DU FIL DES TEMPS FORTS, datees sur l axe de frames.
//
// CE QUE C EST. Le morceau des temps forts du film porte, a cote des kills et des morts, une ligne
// par medaille gagnee : le joueur (xuid), l instant et le couple d octets qui identifie la medaille
// (`grammar.ParseHighlightEvents`). Le document les publie telles quelles, sur le MEME axe que tout
// le reste, sans le moindre calcul de score.
//
// # L HORLOGE : AUCUN CALAGE, ET LA REGLE QUI LE DIT
//
// L instant d une medaille est sur l horloge des temps forts, dont le zero est le premier paquet du
// film : le zero de `originMs`, qui vaut `premier paquet de position - premier paquet du film`. La
// frame est donc `(ms - originMs) / frameIntervalMs`, par la division entiere que le document
// emploie partout ([frameOfOriginMS]). Deux regles s en deduisent :
//
//   - SANS ORIGINE ETABLIE, RIEN NE SE POSE. Une soustraction par zero decalerait chaque medaille de
//     3,6 s a 50,8 s selon le match ; elles sont comptees (`coverage.medals.noOrigin`) et non
//     publiees.
//   - LE CALAGE DU FIL DES MORTS N INTERVIENT PAS. `deathOffsetMs` est un plateau cherche par
//     appariement dans une fenetre : son centre bouge de plusieurs dizaines de ms d un film a l autre, quand
//     le zero des temps forts se confond avec le premier paquet du film a une image pres.
//
// # CE QUI N EST PAS PUBLIE
//
// Les points de score personnel d une medaille : l API ne compte aucun point pour une medaille (la somme de ses recompenses
// est le score personnel du joueur), et la table du film ne porte que l indice et le nom. Le score par action est un autre
// calque, lu dans les messages de jeu (document_score_actions.go), pas celui-ci.

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/highlightevent"
	"github.com/JGtm/halo-infinite-film-decoder/medalname"
)

// Medal est UNE medaille gagnee : quand, par qui, laquelle.
type Medal struct {
	// T est l index de frame, sur le meme axe que Point.T.
	T int `json:"t"`
	// MS est l instant de la medaille sur l horloge des temps forts, en millisecondes : la meme que
	// celle des morts du fil et du kill-feed. T n en est que l arrondi a la grille ; MS ne depend
	// pas du pas.
	MS int64 `json:"ms"`
	// XUID est le joueur qui a gagne la medaille, en decimal — meme forme que Track.XUID.
	XUID string `json:"xuid"`
	// ID est l indice de medaille du jeu, celui que le film porte. Il identifie la medaille ; le
	// nom n est qu une lecture de table, et un indice absent de la table garde le sien.
	ID int `json:"id"`
	// Name est le nom ANGLAIS de la medaille : une cle de referentiel, jamais un libelle d interface
	// (la locale se resout cote lecteur, comme pour les armes). Absent quand le couple d octets du
	// film n est pas dans la table des medailles connues : jamais le nom d une voisine.
	Name string `json:"name,omitempty"`
}

// MedalInput est ce que l assemblage recoit des medailles : la lecture du film, et le temoin qui
// dit qu elle a eu lieu. Une entree de DONNEES, comme `Deaths`.
type MedalInput struct {
	// Read : le morceau des temps forts a ete lu. Faux = pas de lecture, le calque n est pas
	// produit et la couverture est absente : une liste vide est alors une mesure, pas un silence.
	Read   bool
	Awards []highlightevent.MedalAward
}

// MedalCoverage dit ce que le calque a pose et ce qu il a laisse de cote. Trois refus (`noOrigin`,
// `beforeOrigin`, `afterEnd`) s ajoutent a `published` pour valoir `decoded` ; `unknownName` compte, parmi les publiees,
// celles sans nom.
type MedalCoverage struct {
	// Decoded est le nombre de medailles du morceau des temps forts.
	Decoded int `json:"decoded"`
	// Published est le nombre de medailles posees sur l axe.
	Published int `json:"published"`
	// NoOrigin : medailles non posees parce que l origine du document n est pas etablie.
	NoOrigin int `json:"noOrigin"`
	// BeforeOrigin : medailles anterieures a la frame 0.
	BeforeOrigin int `json:"beforeOrigin"`
	// AfterEnd : medailles posterieures a la derniere frame.
	AfterEnd int `json:"afterEnd"`
	// UnknownName compte les medailles publiees SANS nom (couple d octets hors de la table) ; c est
	// un sous-ensemble de Published.
	UnknownName int `json:"unknownName"`
}

// buildMedals pose les medailles sur l axe de frames. `originMS` est l origine publiee du document
// (nil = non etablie) et `frames` son nombre de frames. Les medailles rendues sont triees par
// frame, puis xuid, puis indice.
func buildMedals(in MedalInput, originMS *int64, intervalMS, frames int) ([]Medal, MedalCoverage) {
	cov := MedalCoverage{Decoded: len(in.Awards)}
	var out []Medal
	for _, a := range in.Awards {
		switch {
		case originMS == nil || intervalMS <= 0:
			cov.NoOrigin++
			continue
		case a.TimeMS < *originMS:
			cov.BeforeOrigin++
			continue
		}
		t := frameOfOriginMS(a.TimeMS, *originMS, intervalMS)
		if t >= frames {
			cov.AfterEnd++
			continue
		}
		name, _ := medalname.Lookup(a.TypeHint, a.MedalType)
		if name == "" {
			cov.UnknownName++
		}
		out = append(out, Medal{T: t, MS: a.TimeMS, XUID: strconv.FormatUint(a.XUID, 10), ID: a.MedalType, Name: name})
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b Medal) int {
		return cmp.Or(cmp.Compare(a.T, b.T), strings.Compare(a.XUID, b.XUID), cmp.Compare(a.ID, b.ID))
	})
	return out, cov
}

// poserLesMedailles publie le calque des medailles : seulement quand le morceau des temps forts a
// ete lu, et APRES la couverture, qu il complete.
func (a *assemblage) poserLesMedailles() {
	if !a.opt.Medals.Read {
		return
	}
	medals, cov := buildMedals(a.opt.Medals, a.doc.OriginMs, a.interval, a.doc.FrameCount)
	a.doc.Medals = medals
	a.doc.Coverage.Medals = &cov
}
