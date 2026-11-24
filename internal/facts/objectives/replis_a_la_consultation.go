package objectives

// replis_a_la_consultation.go — LE REPLI QUI SE DECLENCHE A LA CONSULTATION, COMPTE PAR
// EVENEMENT DISTINCT (lot J8.7-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision du
// superviseur du 2026-09-28).
//
// # LE PROBLEME
//
// `repli_emission_hors_domaine_jetee` (les filtres de [rawSeriesByRound] et [rawSeriesByKey]) ne vit pas
// dans une construction qui rend un resultat unique : il se declenche dans des LECTURES que chaque
// calque de `replay` et la cuisson (`replaybuild`) refont sur les memes enregistrements. Compter par
// consultation ferait dependre le compte du nombre de calques qui lisent la meme emission — un compte
// qui mesure l architecture, pas le film.
//
// # LA REGLE
//
// Un enregistreur PAR DOCUMENT, partage par POINTEUR : toutes les lectures d un document (les copies
// d un [RoundIdentity], et les series que chaque calque redemande) y notent la CLE de l evenement, et
// le compte est la taille de l ensemble. N lectures du meme evenement le comptent une fois ; le compte
// ne depend ni de l ordre ni du nombre des lectures.
//
//	emission jetee          (composant, cote, slot, manche, instant) — la cle de la serie et l instant
//
// POURQUOI PAR DOCUMENT ET PAS PAR RESOLVEUR. Un document resout son identite par manche jusqu a
// quatre fois (le pont de la cuisson, puis la couronne, les compteurs de joueurs multi-manche, et le
// drapeau ou le crane sans pont fourni) : un enregistreur par resolveur compterait deux fois l emission
// que deux resolutions lisent. La cuisson cree donc l enregistreur et le passe aux options du rejeu
// (`replay.ReplisHorsBalayage.Consultations`), que la table de versement lit en fin d assemblage.
//
// Nil ne compte rien : c est la valeur des outils hors production (`cmd/`) et des tests qui ne
// regardent pas les replis. Le garde-rail `archlint/film_consultations_comptees_test.go` interdit le
// nil litteral dans les lectures de `replay` et de `replaybuild`.

import (
	"sync"

	"github.com/JGtm/halo-infinite-film-decoder/internal/constat"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ReplisALaConsultation retient les evenements DISTINCTS du repli qui se declenche a la lecture. La valeur zero est prete a l emploi ; elle se partage par pointeur.
type ReplisALaConsultation struct {
	mu        sync.Mutex
	emissions map[cleDEmission]struct{}
	// gardees : les emissions que la garde des compteurs nommes ecarte ([withinKillBound]), par serie et instant.
	gardees map[cleDEmission]struct{}
	// diag : les DIAGNOSTICS des lectures faites sous cet enregistreur (bornes de deroulage, serie
	// non chronologique — lot J12.3, ADR 0034 D-4). L assemblage qui le porte les releve.
	diag constat.Diagnostics
}

// cleDEmission : la serie (composant, cote, slot, manche) et l instant de l emission jetee.
type cleDEmission struct {
	key    statSlotKey
	slot   int
	round  int
	timeMS int
}

// Diagnostics rend les diagnostics des lectures faites sous cet enregistreur (lot J12.3). nil
// sur un enregistreur nil : les outils hors production n en recueillent pas.
func (r *ReplisALaConsultation) Diagnostics() *constat.Diagnostics {
	if r == nil {
		return nil
	}
	return &r.diag
}

// Essai ouvre un enregistreur A PART, dont le compte d'emissions ne rejoint jamais celui de r : la lecture du verdict de mode juge
// les emissions par la table du Drapeau, qui n'est celle du film que si le verdict la reconnait, et le compte du film se
// fait alors dans les lectures du calque du Drapeau. Seuls les diagnostics se versent ([ReplisALaConsultation.AdopterLesDiagnostics]).
// Nil sur un enregistreur nil : rien ne se compte.
func (r *ReplisALaConsultation) Essai() *ReplisALaConsultation {
	if r == nil {
		return nil
	}
	return &ReplisALaConsultation{}
}

// AdopterLesDiagnostics verse dans r les seuls diagnostics de `essai` (un diagnostic se journalise une fois : `essai` est
// vide apres).
func (r *ReplisALaConsultation) AdopterLesDiagnostics(essai *ReplisALaConsultation) {
	if r == nil || essai == nil {
		return
	}
	essai.diag.Verser(&r.diag)
}

// noterEmissionJetee note une emission que le filtre de domaine jette (`repli_emission_hors_domaine_jetee`).
func (r *ReplisALaConsultation) noterEmissionJetee(key statSlotKey, rec types.StatRecord) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.emissions == nil {
		r.emissions = map[cleDEmission]struct{}{}
	}
	r.emissions[cleDEmission{key: key, slot: rec.Slot, round: rec.Round, timeMS: rec.TimeMS}] = struct{}{}
}

// noterEmissionGardee note une emission que la garde des compteurs nommes ecarte ([withinKillBound]). Ce n'est pas un repli :
// l'emission ne remplace rien, elle est retiree, et le compte ne se verse pas au registre des replis.
func (r *ReplisALaConsultation) noterEmissionGardee(key statSlotKey, slot, round, timeMS int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.gardees == nil {
		r.gardees = map[cleDEmission]struct{}{}
	}
	r.gardees[cleDEmission{key: key, slot: slot, round: round, timeMS: timeMS}] = struct{}{}
}

// EmissionsGardees rend, par nom de compteur nomme ([ScoreboardStat.Name]), le nombre d'emissions DISTINCTES que la garde
// des compteurs nommes a ecartees, ou nil quand elle n'en a ecarte aucune. Comme les replis a la consultation, le compte
// ne depend ni du nombre ni de l'ordre des lectures qui rencontrent la meme emission.
func (r *ReplisALaConsultation) EmissionsGardees() map[string]int {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out map[string]int
	for cle := range r.gardees {
		for _, st := range tousLesCompteursDuTableau() {
			if st.Component.key() != cle.key {
				continue
			}
			if out == nil {
				out = map[string]int{}
			}
			out[st.Name]++
		}
	}
	return out
}

// ComptesDesReplis rend le nombre d evenements DISTINCTS notes. Nil : zero.
func (r *ReplisALaConsultation) ComptesDesReplis() ComptesDesReplis {
	if r == nil {
		return ComptesDesReplis{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return ComptesDesReplis{
		EmissionsHorsDomaineJetees: len(r.emissions),
	}
}
