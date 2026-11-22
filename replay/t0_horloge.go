package replay

import (
	"log/slog"
	"math"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// t0_horloge.go — LE COUP D'ENVOI, DATE PAR L'HORLOGE DE MANCHE DU JEU.
//
// # LA DEFINITION
//
// Le coup d'envoi d'un match est l'instant ou l'horloge de manche de la PREMIERE manche demarre. Le jeu
// la reecrit chaque seconde tant qu'elle court (cf. document_horloges.go) : limite de la manche, temps de la
// minuterie, etat (1 arretee, 6 descend d'une seconde par seconde, 19 monte d'une seconde par seconde).
//
// # LA REGLE
//
// La PREMIERE lecture retenue qui COURT (etat 6 ou 19, [jugerLaLecture]) donne le coup d'envoi : on la ramene
// a l'instant ou la minuterie valait sa valeur de depart, par l'arithmetique de la lecture elle-meme — une
// seconde de minuterie est une seconde de film. La valeur de depart est la limite de la manche quand elle
// descend (temps restant = limite - ecoule), zero quand elle monte (temps de la minuterie = ecoule) :
//
//	depart(descend) = instant de la lecture - (limite - temps restant)
//	depart(monte)   = instant de la lecture - temps de la minuterie
//
// Une lecture arretee (etat 1, valeur de depart) qui precede directement la premiere lecture qui court ne
// date rien : elle dit que le depart est LU ([T0FilmCoverage.StartRead]) ; sans elle (film qui commence
// horloge en marche) le depart se deduit de la seule valeur lue.
//
// Seule la premiere manche date le coup d'envoi : un match a plusieurs manches a plusieurs departs d'horloge,
// et la premiere lecture qui court est celle de la premiere. AUCUN SEUIL : la regle ne compare rien a une
// duree ecrite.
//
// # CE QUE LE FILM NE DIT PAS EST REFUSE
//
// Un film sans aucune lecture qui court n'a pas de coup d'envoi : `t0FilmMs` est absent et
// `coverage.t0Film.reason` vaut [t0FilmReasonNoRunningClock]. Aucun repli sur le mouvement des joueurs.
// SANS ORIGINE, PAS DE COUP D'ENVOI : le resultat est un instant sur l'horloge du fil des eliminations, et
// sans origine cette horloge n'est pas etablie ; la couverture est alors absente, la passe n'a pas tourne.
//
// # L'HORLOGE DU RESULTAT
//
// `t0FilmMs` est sur la meme horloge qu'`originMs` (celle du fil des eliminations) :
// `originMs + (depart - premier paquet de position) / 1000`, a la milliseconde (arrondi vers le bas). Le
// depart peut preceder la frame 0 (la marge est alors negative) : le film peut commencer apres le depart de
// l'horloge.

const (
	// t0FilmSourceRoundClock : la source du coup d'envoi, l'horloge de manche du film.
	t0FilmSourceRoundClock = "roundClock"
	// t0FilmReasonNoRunningClock : aucune lecture retenue de l'horloge ne court.
	t0FilmReasonNoRunningClock = "noRunningClock"
)

// coupDEnvoiLisible dit si la passe du coup d'envoi tourne : une origine de la frame 0 sur l'horloge du fil, et
// la marche des trames qui lit l'horloge. C'est aussi la garde de `layers` pour `t0FilmMs`.
func coupDEnvoiLisible(doc *ReplayDocument, opt Options) bool {
	return doc.OriginMs != nil && opt.Horloges.Stats.Trames > 0
}

// poserLeCoupDEnvoi pose `t0FilmMs` et son verdict sur le document.
//
// APRES `composerLaCouverture` (le verdict vit dans `doc.Coverage`, qui n'existe qu'a elle) et AVANT les calques
// qui lisent le coup d'envoi (l'etat des zones, `poserObjectifsVivants`). Il se lit sur les lectures de la
// grammaire (`Options.Horloges`), pas sur la serie publiee, qui ne republie pas une lecture identique.
func (a *assemblage) poserLeCoupDEnvoi() {
	if !coupDEnvoiLisible(&a.doc, a.opt) {
		return
	}
	t0, cov := coupDEnvoiParHorloge(a.opt.Horloges.Horloges, a.origin, *a.doc.OriginMs)
	a.doc.T0FilmMs, a.doc.Coverage.T0Film = t0, cov
	if t0 == nil {
		slog.WarnContext(a.ctx, "rejeu : coup d'envoi NON date — aucune lecture de l'horloge de manche ne court",
			"match_id", a.matchID, "lectures", len(a.opt.Horloges.Horloges), "raison", cov.Reason)
	}
}

// coupDEnvoiParHorloge applique la regle de l'en-tete aux lectures de l'horloge de manche, dans l'ordre du
// film. `origin` est l'horodatage (microsecondes) du premier paquet de position, `originMs` l'instant de la
// frame 0 sur l'horloge du fil. Rend l'instant, ou nil et le verdict qui dit pourquoi.
func coupDEnvoiParHorloge(reads []grammar.LectureHorloge, origin uint64, originMs int64) (*int64, *T0FilmCoverage) {
	cov := &T0FilmCoverage{Source: t0FilmSourceRoundClock}
	arretee := false
	for _, r := range reads {
		if jugerLaLecture(r) != lectureRetenue {
			continue
		}
		if r.Etat == etatHorlogeArretee {
			arretee = true
			continue
		}
		cov.Detected, cov.StartRead = true, arretee
		cov.MarginMs = millisecondesVersLeBas(departDeLaMinuterieUS(r) - int64(origin)) //nolint:gosec // horodatage de film, tient dans int64
		t0 := originMs + cov.MarginMs
		return &t0, cov
	}
	cov.Reason = t0FilmReasonNoRunningClock
	return nil, cov
}

// departDeLaMinuterieUS rend l'instant (microsecondes du film) ou la minuterie valait sa valeur de depart,
// pour une lecture retenue qui court.
func departDeLaMinuterieUS(r grammar.LectureHorloge) int64 {
	ecoule := float64(secondesDeLHorloge(r.Reste))
	if r.Etat == etatHorlogeDescend {
		ecoule = float64(secondesDeLHorloge(r.Limite)) - ecoule
	}
	return int64(r.TS) - int64(math.Round(ecoule*1e6)) //nolint:gosec // horodatage de film, tient dans int64
}

// millisecondesVersLeBas convertit des microsecondes en millisecondes, arrondi vers le bas (aussi pour un
// ecart negatif).
func millisecondesVersLeBas(us int64) int64 {
	ms := us / 1000
	if us%1000 < 0 {
		ms--
	}
	return ms
}
