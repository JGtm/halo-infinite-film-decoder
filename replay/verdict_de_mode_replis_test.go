package replay

// verdict_de_mode_replis_test.go — LA TABLE DU DRAPEAU EST LUE SUR TOUT FILM, ET NE COMPTE QUE SUR UN FILM DE DRAPEAU.
//
// Le verdict de mode applique la table du Drapeau a chaque film. Sur un film d'un autre mode (ici : zones, ou `22 B`
// est une duree) une emission saine `22 = n / duree` que cette table jetterait en entier n'est pas un repli du film :
// elle ne se compte pas, et les series que le verdict lit sont celles d'avant la garde des deux canaux. Sur un film de
// Drapeau, la table est celle du film : `24 = 12 / 7034` est jetee et comptee.
//
// MUTATION JOUEE : lire le verdict par `NamedEventsFrom` sur l'enregistreur du document, sans essai, fait rougir le premier
// test (« 6 emission(s) ») ; ne plus lire les evenements du calque du Drapeau dans [attachFlagCarries] fait rougir le second.

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func recDeMode(ms, slot, comp int, a, b int64) types.StatRecord {
	return types.StatRecord{TimeMS: ms, Slot: slot, Round: 0, Comps: map[int]types.StatValue{comp: {A: a, B: b}}}
}

func TestVerdictSurUnFilmDUnAutreModeNeCompteAucuneEmission(t *testing.T) {
	in := FlagInput{Scanned: true, Records: []types.StatRecord{
		recDeMode(4000, 12, 22, 1, 470), recDeMode(5000, 12, 22, 2, 555), recDeMode(6000, 12, 22, 3, 3670),
	}}
	cons := &objectives.ReplisALaConsultation{}
	signals := flagFilmSignalsOf(in, cons)
	if signals.IsFlagFilm() || signals.Grabs != 3 {
		t.Errorf("signaux %+v : film de zones, trois prises lues par `22 A`, aucun film de Drapeau", signals)
	}
	doc := &ReplayDocument{FrameCount: 100, Coverage: &Coverage{}}
	attachFlagCarries(context.Background(), doc, Options{Flag: in,
		ReplisHorsBalayage: ReplisHorsBalayage{Consultations: cons}},
		IdentityRegistry{}, replayClock{origin: 0, step: 100_000, frames: 100}, teamPublication{})
	if n := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 0 {
		t.Errorf("%d emission(s) jetee(s) comptee(s) sur un film d'un autre mode, attendu 0", n)
	}
	if cov := doc.Coverage.FlagCarries; cov == nil || cov.FlagFilm {
		t.Errorf("couverture %+v : le film n'est pas un film de Drapeau", cov)
	}
}

func TestVerdictSurUnFilmDeDrapeauJugeEtCompteSesEmissions(t *testing.T) {
	in := FlagInput{
		Scanned: true,
		Records: []types.StatRecord{
			recDeMode(1000, 12, 24, 1, 0),     // un vol sain
			recDeMode(1500, 12, 24, 12, 7034), // douze vols et 7034 eliminations en portant : emission mal alignee
			recDeMode(9000, 12, 21, 1, 0),     // la capture, sans quoi le film n'est pas reconnu CTF
		},
		Bursts:   []int{9000},
		Spawns:   flagInvariantSpawns(),
		Identity: objectives.FlatRoundIdentity(map[int]string{12: "1"}),
	}
	cons := &objectives.ReplisALaConsultation{}
	signals := flagFilmSignalsOf(in, cons)
	if !signals.IsFlagFilm() || signals.Steals != 1 {
		t.Errorf("signaux %+v : film de Drapeau, un seul vol lu (l'emission 12 / 7034 est jetee)", signals)
	}
	doc := &ReplayDocument{FrameCount: 100, Coverage: &Coverage{},
		Tracks: []Track{flagTestTrack(12, "1", 0, 99, 2, 2)}}
	attachFlagCarries(context.Background(), doc, Options{Flag: in,
		ReplisHorsBalayage: ReplisHorsBalayage{Consultations: cons}},
		IdentityRegistry{}, replayClock{origin: 0, step: 100_000, frames: 100}, teamPublication{})
	// L'emission jetee se compte une fois par serie nommee (`24 A` et `24 B`), quel que soit le nombre de lectures.
	if n := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 2 {
		t.Errorf("%d emission(s) jetee(s) comptee(s) sur un film de Drapeau, attendu 2", n)
	}
	if cov := doc.Coverage.FlagCarries; cov == nil || !cov.FlagFilm {
		t.Errorf("couverture %+v : le film est un film de Drapeau", cov)
	}
}
