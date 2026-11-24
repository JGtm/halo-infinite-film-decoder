package replay

// repli_siege_sans_tenant_test.go — UN REPLI COMPTE CE QU IL REMPLACE.
//
// `repli_tireur_par_le_siege_de_la_place` compte les tirs et les rafales dont le tireur vient du siege du roster. Ceux qu aucune
// lecture ni aucun siege ne tient gardent l index du record : rien n y est remplace, ils se comptent a part (`shotsUnheld`,
// `burstsUnheld`).

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestUnTirSansTenantNeDeclenchePasLeRepliDuSiege : sur un film sans roster, tous les tirs sont sans tenant ; le repli vaut 0 et les
// tirs sont comptes sous `shotsUnheld` dans le document publie.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : declencher le repli pour toute source autre que la place lue.
func TestUnTirSansTenantNeDeclenchePasLeRepliDuSiege(t *testing.T) {
	fire := []grammar.FireEvent{tirDeTest(3, 1, instantDeLaFrame(3), 1), tirDeTest(3, 2, instantDeLaFrame(4), 1)}
	a := assemblageDeTouches(true, fire, nil)
	a.doc.Roster = nil
	tirs := a.tirsDuCombat()
	a.doc.Coverage.ControlPlaces = &a.placesCov
	cp := documentJSON(t, a.doc)["coverage"].(map[string]any)["controlPlaces"].(map[string]any)
	if cp["shotsUnheld"] != float64(2) || cp["shotsBySeat"] != float64(0) {
		t.Errorf("controlPlaces = %v, attendu 2 tirs sans tenant et aucun par le siege", cp)
	}
	if n := a.opt.Fallbacks.Compte(fallback.NomTireurParLeSiegeDeLaPlace); n != 0 {
		t.Errorf("repli compte %d fois, attendu 0 : aucun tireur n est remplace", n)
	}
	if tirs[0].FilmIndex != 3 || tirs[1].FilmIndex != 3 {
		t.Errorf("les tirs sans tenant ont change d index : %d, %d", tirs[0].FilmIndex, tirs[1].FilmIndex)
	}
}

// TestUneRafaleSansTenantNeDeclenchePasLeRepliDuSiege : meme regle pour les rafales.
func TestUneRafaleSansTenantNeDeclenchePasLeRepliDuSiege(t *testing.T) {
	doc := fbDoc()
	horloge := vsClock()
	horloge.fb = fallback.NouveauCompteur()
	cov := ControlPlaceCoverage{}
	e := entreesDesRafales{lues: []types.ContinuousFireBurst{fbRafale(3, 20, 25, 1)},
		stats:  types.ContinuousFireStats{Scanned: true},
		places: tireursDesPlaces{}, lecture: &cov}
	buildFireBursts(context.Background(), doc, e, occupantsPlats(map[uint32]int{10: 3}), horloge)
	if cov.BurstsUnheld != 1 || cov.BurstsBySeat != 0 {
		t.Errorf("comptes = %+v : attendu 1 rafale sans tenant", cov)
	}
	if n := horloge.fb.Compte(fallback.NomTireurParLeSiegeDeLaPlace); n != 0 {
		t.Errorf("repli compte %d fois, attendu 0", n)
	}
}
