package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Les instants des cas sont des multiples du pas : la frame d un instant est instant / pas.
const (
	pasDeLaGrille = 33_000
	sansLacher    = -1000
)

func lectureDeCommande(slot uint32, frame, joueur int) types.UnitControlRead {
	return types.UnitControlRead{Slot: slot, TimestampUS: uint64(frame+10) * pasDeLaGrille, Controller: joueur}
}

func suppressionDeTest(slot uint32, frame int) types.BipedDeletion {
	return types.BipedDeletion{Slot: slot, TimestampUS: uint64(frame+10) * pasDeLaGrille}
}

func pisteDeTest(slot uint32, debut int) Track { return Track{Slot: slot, StartFrame: debut} }

// L origine de la grille est dix pas apres l instant zero : une lecture peut tomber a une frame negative.
const origineDeLaGrille = 10 * pasDeLaGrille

func finsDeTest(t *testing.T, tracks []Track, p LecturesDePresence) ([]finLue, LifeEndCoverage) {
	t.Helper()
	p.ControlStats.Scanned = true
	return finsParCommande(tracks, p, origineDeLaGrille, pasDeLaGrille)
}

func TestFinsParCommande_Cas(t *testing.T) {
	cas := []struct {
		nom     string
		pistes  []Track
		lus     LecturesDePresence
		raisons []string
		relache []int
		cov     LifeEndCoverage
	}{
		{"lacher apres la fin", []Track{pisteDeTest(3, 100)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 99, 4), lectureDeCommande(3, 180, -1)}},
			[]string{"released"}, []int{180}, LifeEndCoverage{Released: 1}},
		{"suppression encore commande", []Track{pisteDeTest(3, 100)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 100, 4)},
				Deletions: []types.BipedDeletion{suppressionDeTest(3, 400)}},
			[]string{"deleted"}, []int{sansLacher}, LifeEndCoverage{Deleted: 1}},
		{"slot repris par un autre joueur : deleted et retaken", []Track{pisteDeTest(3, 100), pisteDeTest(3, 300)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 100, 4), lectureDeCommande(3, 290, 7)}},
			[]string{"deleted", ""}, []int{sansLacher, sansLacher}, LifeEndCoverage{Retaken: 1, Open: 1}},
		{"aucune lecture : absent", []Track{pisteDeTest(3, 100)}, LecturesDePresence{},
			[]string{""}, []int{sansLacher}, LifeEndCoverage{Unread: 1}},
		{"un lacher avant toute lecture commandee n ouvre pas de corps", []Track{pisteDeTest(3, 100)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 150, -1)}},
			[]string{""}, []int{sansLacher}, LifeEndCoverage{Unread: 1}},
		{"le lacher de la vie precedente n est pas pris par la suivante", []Track{pisteDeTest(3, 100), pisteDeTest(3, 300)},
			LecturesDePresence{Controls: []types.UnitControlRead{
				lectureDeCommande(3, 100, 4), lectureDeCommande(3, 200, -1), lectureDeCommande(3, 298, 4)}},
			[]string{"released", ""}, []int{200, sansLacher}, LifeEndCoverage{Released: 1, Open: 1}},
		{"une lecture apres le debut de la vie suivante n est pas prise", []Track{pisteDeTest(3, 100), pisteDeTest(3, 300)},
			LecturesDePresence{Controls: []types.UnitControlRead{
				lectureDeCommande(3, 100, 4), lectureDeCommande(3, 350, -1)}},
			[]string{"", "released"}, []int{sansLacher, 350}, LifeEndCoverage{Open: 1, Released: 1}},
		{"lacher a frame negative : lecture, pas absence", []Track{pisteDeTest(3, -2)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, -3, 4), lectureDeCommande(3, -1, -1)}},
			[]string{"released"}, []int{-1}, LifeEndCoverage{Released: 1}},
		{"lacher a la frame zero", []Track{pisteDeTest(3, -1)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, -1, 4), lectureDeCommande(3, 0, -1)}},
			[]string{"released"}, []int{0}, LifeEndCoverage{Released: 1}},
		{"suppression puis recreation au meme instant : la suppression precede", []Track{pisteDeTest(3, 100), pisteDeTest(3, 300)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 100, 4), lectureDeCommande(3, 250, 4)},
				Deletions: []types.BipedDeletion{suppressionDeTest(3, 250)}},
			[]string{"deleted", ""}, []int{sansLacher, sansLacher}, LifeEndCoverage{Deleted: 1, Open: 1}},
		{"un corps ferme avant la piste n est pas celui de la piste", []Track{pisteDeTest(3, 300)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 100, 4), lectureDeCommande(3, 200, -1)}},
			[]string{""}, []int{sansLacher}, LifeEndCoverage{Unread: 1}},
		{"deux slots independants", []Track{pisteDeTest(3, 100), pisteDeTest(4, 100)},
			LecturesDePresence{Controls: []types.UnitControlRead{lectureDeCommande(3, 100, 4), lectureDeCommande(3, 150, -1),
				lectureDeCommande(4, 100, 5)}, DeletionStats: types.BipedDeletionStats{DeletionsUnproven: 2}},
			[]string{"released", ""}, []int{150, sansLacher}, LifeEndCoverage{Released: 1, Open: 1, DeletionsUnproven: 2}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			fins, cov := finsDeTest(t, c.pistes, c.lus)
			for i, f := range fins {
				if f.raison != c.raisons[i] {
					t.Errorf("piste %d : raison %q, attendu %q", i, f.raison, c.raisons[i])
				}
				if f.raison == EndReasonReleased && f.relache != c.relache[i] {
					t.Errorf("piste %d : lacher a la frame %d, attendu %d", i, f.relache, c.relache[i])
				}
			}
			if cov != c.cov {
				t.Errorf("couverture %+v, attendu %+v", cov, c.cov)
			}
		})
	}
}

func TestPoserLesFinsParCommande_Pointeur(t *testing.T) {
	a := &assemblage{origin: origineDeLaGrille, step: pasDeLaGrille}
	a.doc.Tracks = []Track{pisteDeTest(3, -1)}
	a.opt.Presences = LecturesDePresence{ControlStats: types.UnitControlStats{Scanned: true},
		Controls: []types.UnitControlRead{lectureDeCommande(3, -1, 4), lectureDeCommande(3, 0, -1)}}
	a.poserLesFinsParCommande()
	tr := a.doc.Tracks[0]
	if tr.EndReason != EndReasonReleased || tr.ReleaseFrame == nil || *tr.ReleaseFrame != 0 {
		t.Fatalf("piste %+v : lacher a la frame zero attendu, pointeur non nul", tr)
	}
	if a.trackCov.LifeEnds == nil || a.trackCov.LifeEnds.Released != 1 {
		t.Fatalf("couverture %+v", a.trackCov.LifeEnds)
	}
}

func TestPoserLesFinsParCommande_LectureAbsente(t *testing.T) {
	a := &assemblage{origin: origineDeLaGrille, step: pasDeLaGrille}
	a.doc.Tracks = []Track{pisteDeTest(3, 10)}
	a.poserLesFinsParCommande()
	if a.doc.Tracks[0].EndReason != "" || a.doc.Tracks[0].ReleaseFrame != nil || a.trackCov.LifeEnds != nil {
		t.Fatal("sans lecture de commande, aucun champ ni couverture")
	}
}
