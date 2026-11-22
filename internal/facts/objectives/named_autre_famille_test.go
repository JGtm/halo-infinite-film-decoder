package objectives

// named_autre_famille_test.go — LA TABLE DU DRAPEAU, LUE SUR UN FILM D'UN AUTRE MODE, NE JUGE RIEN ET NE COMPTE RIEN
// ([NamedEventsFromAutreFamille]).
//
// Le verdict de mode applique la table du Drapeau a tout film. Sur un film de zones, `22 = n / duree` est une emission
// saine (`22 B` est une duree) que la table du Drapeau, qui nomme `22 A` et `22 B`, jetterait en entier : elle serait
// comptee comme un repli du film alors qu'elle n'en est pas un. Chaque test est un garde-fou : faire juger cette lecture
// par la table du Drapeau, ou lui laisser verser son compte au document, fait rougir « autre famille » ; retirer le
// jugement de la lecture du film de Drapeau fait rougir « famille du film ».

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// corpusAutreFamille : un joueur de zones (slot 12) dont `22 = n / duree` est sain, un vol aberrant `24 = 12 / 7034`
// et une emission negative de `21 A`.
func corpusAutreFamille() []types.StatRecord {
	emission := func(ms, comp int, a, b int64) types.StatRecord {
		return types.StatRecord{TimeMS: ms, Slot: 12, Round: 0, Comps: map[int]types.StatValue{comp: {A: a, B: b}}}
	}
	return []types.StatRecord{
		emission(4000, 22, 1, 470), emission(5000, 22, 2, 555), emission(6000, 22, 3, 3670),
		emission(7000, 24, 12, 7034),
		emission(8000, 21, -4, 0),
	}
}

// comptes rend, par statistique, le nombre d'evenements.
func comptes(evs []NamedEvent) map[string]int {
	out := map[string]int{}
	for _, e := range evs {
		out[e.Stat]++
	}
	return out
}

func TestLectureDUneAutreFamilleNeJugeRienEtNeCompteRien(t *testing.T) {
	recs := corpusAutreFamille()
	cons := &ReplisALaConsultation{}
	evs := NamedEventsFromAutreFamille(recs, ObjectiveTypeFlag, []string{StatFlagCaptures, StatFlagSteals, StatFlagGrabs}, cons)
	if n := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 0 {
		t.Errorf("%d emission(s) jetee(s) comptee(s) au document par une lecture d'une autre famille, attendu 0", n)
	}
	got := comptes(evs)
	// Les trois prises de `22 A` sont lues (la table du Drapeau ne les juge pas), et le vol aberrant se deroule comme
	// avant la garde des deux canaux : douze vols.
	if got[StatFlagGrabs] != 3 || got[StatFlagSteals] != 12 {
		t.Errorf("evenements lus %v, attendu 3 prises et 12 vols (series inchangees hors de la famille du film)", got)
	}
	for stat := range got {
		if !slices.Contains([]string{StatFlagCaptures, StatFlagSteals, StatFlagGrabs}, stat) {
			t.Errorf("statistique %q lue : le verdict ne lit que ses trois emplacements", stat)
		}
	}
	// Les series sont celles d'une lecture sans jugement de la table complete, restreinte aux trois statistiques.
	var attendus []NamedEvent
	for _, e := range namedEventsOfTable(recs, ObjectiveTypeFlag, namedStatSlots[ObjectiveTypeFlag], nil, nil) {
		if e.Stat == StatFlagCaptures || e.Stat == StatFlagSteals || e.Stat == StatFlagGrabs {
			attendus = append(attendus, e)
		}
	}
	if !slices.Equal(evs, attendus) {
		t.Errorf("evenements lus %v, attendus %v", evs, attendus)
	}
}

func TestLectureDeLaFamilleDuFilmJugeEtCompte(t *testing.T) {
	recs := corpusAutreFamille()
	cons := &ReplisALaConsultation{}
	got := comptes(NamedEventsFrom(recs, ObjectiveTypeFlag, cons))
	// `22 = n / duree` (canal B au-dela de la borne) et `24 = 12 / 7034` sont jetes en entier, sur les deux canaux.
	if got[StatFlagGrabs] != 0 || got[StatFlagSteals] != 0 {
		t.Errorf("evenements lus %v : les emissions aberrantes de la famille du film doivent etre jetees", got)
	}
	// Trois emissions de `22`, une de `24` et la valeur negative de `21 A`, chacune sur ses deux series nommees.
	if n := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 10 {
		t.Errorf("%d emission(s) jetee(s) comptee(s), attendu 10", n)
	}
}

func TestEssaiNeVerseQueSesDiagnostics(t *testing.T) {
	recs := corpusAutreFamille()
	cons := &ReplisALaConsultation{}
	essai := cons.Essai()
	NamedEventsFrom(recs, ObjectiveTypeFlag, essai)
	cons.AdopterLesDiagnostics(essai)
	if n := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 0 {
		t.Fatalf("l'essai a verse %d emission(s) a l'enregistreur du document", n)
	}
	if n := essai.ComptesDesReplis().EmissionsHorsDomaineJetees; n != 10 {
		t.Errorf("l'essai a compte %d emission(s), attendu 10", n)
	}
	var nul *ReplisALaConsultation
	nul.AdopterLesDiagnostics(essai)
	if nul.Essai() != nil {
		t.Error("l'essai d'un enregistreur nil doit etre nil")
	}
}
