//go:build research

package grammar

// lot1_attrib_helpers_test.go — ADAPTATEURS de recherche autour du numerateur productionise
// (weapon_hits.go). Les instruments consomment les tirs sous la forme historique attribShot. Ces
// helpers NE RE-DECODENT RIEN : attribCollectShots delegue a ScanFilmWeaponShots. Une seule copie
// du decodage / pairing vit dans la production ; ceci n'est que du confort de recherche.

import "testing"

// attribShot : un tir horodate, forme historique consommee par les instruments de recherche.
type attribShot struct {
	ts   uint64
	att  uint64
	wid  uint64
	fidx int
	has  bool
}

// attribCollectShots decode les tirs LONGS 0xD2 via le code de production (ScanFilmWeaponShots) et
// les mappe vers la forme historique attribShot.
func attribCollectShots(t *testing.T, dir string, n int) []attribShot {
	t.Helper()
	shots, err := ScanFilmWeaponShots(dir, n)
	if err != nil {
		t.Fatalf("collecte des tirs : %v", err)
	}
	out := make([]attribShot, len(shots))
	for i, s := range shots {
		out[i] = attribShot{ts: s.TimestampUS, att: s.Attacker, wid: s.WeaponID, fidx: s.FilmIndex, has: s.HasPair}
	}
	return out
}

// attribMin / attribMax : bornes d'un echantillon non vide (utilitaires de recherche).
func attribMin(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func attribMax(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}
