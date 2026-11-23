package grammar

import (
	"reflect"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// seriesDesFiltresEnPlace : des suites à bords, chacune avec ce que les filtres doivent y faire.
func seriesDesFiltresEnPlace() map[string][]BipedPosition {
	return map[string][]BipedPosition{
		"vide":     {},
		"un point": {bp(1, 0, 0, 0)},
		"tous retirés (isolés)": {
			bp(1, 0, 0, 0), bp(2, 100_000, 0, 0), bp(3, 200_000, 0, 0),
		},
		"aucun retiré": {
			bp(1, 0, 0, 0), bp(2, 0, 1, 1), bp(1, 16, 0.1, 0), bp(2, 16, 1.1, 1), bp(1, 32, 0.2, 0),
		},
		"isolés et téléportations mêlés": {
			bp(1, 0, 0, 0), bp(1, 16, 0.1, 0), bp(1, 32, 900, 0), bp(1, 48, 0.2, 0), bp(1, 64, 0.3, 0),
			bp(2, 0, 5, 5), bp(2, 200_000, 5, 5), bp(2, 200_016, 5, 5), bp(2, 200_032, 5.1, 5),
			bp(3, 40, 7, 7),
		},
		"téléportations exemptées": {
			bp(1, 0, 0, 0), bp(1, 16, 0, 0), bp(1, 32, 900, 0), bp(1, 48, 900, 0), bp(1, 64, 1800, 0),
			bp(1, 80, 1800, 0), bp(1, 96, 1800, 0),
		},
	}
}

// TestFiltresEnPlaceRendentLaSuiteDesFiltresQuiCopient : chaque variante en place rend, sur une copie
// de la suite, exactement ce que rend la fonction exportée, qui ne touche pas à son entrée.
func TestFiltresEnPlaceRendentLaSuiteDesFiltresQuiCopient(t *testing.T) {
	exemptions := TeleportExemptionsOf([]types.TranslocatorTeleport{{Slot: 1, TimestampUS: 32_000}, {Slot: 1, TimestampUS: 64_000}})
	for nom, in := range seriesDesFiltresEnPlace() {
		t.Run(nom, func(t *testing.T) {
			for _, gap := range []int{0, DefaultIsolationGapMS} {
				orig := slices.Clone(in)
				attendu := DropIsolated(in, gap)
				if !reflect.DeepEqual(in, orig) {
					t.Fatalf("DropIsolated a modifié son entrée")
				}
				if got := ecarterLesIsolesEnPlace(slices.Clone(in), gap); !slices.Equal(got, attendu) {
					t.Fatalf("isolés gap=%d : en place %+v, copie %+v", gap, got, attendu)
				}
			}
			for _, x := range []TeleportExemptions{nil, exemptions} {
				for _, vmax := range []float64{0, DefaultMaxSpeedMPS} {
					orig := slices.Clone(in)
					attendu := DropTeleportsExcept(in, vmax, x)
					if !reflect.DeepEqual(in, orig) {
						t.Fatalf("DropTeleportsExcept a modifié son entrée")
					}
					if got := ecarterLesTeleportationsEnPlace(slices.Clone(in), vmax, x); !slices.Equal(got, attendu) {
						t.Fatalf("téléportations vmax=%v exemptions=%v : en place %+v, copie %+v", vmax, x != nil, got, attendu)
					}
				}
			}
			// L'enchaînement de production : isolés puis téléportations.
			attendu := DropTeleportsExcept(DropIsolated(in, DefaultIsolationGapMS), DefaultMaxSpeedMPS, exemptions)
			got := ecarterLesTeleportationsEnPlace(ecarterLesIsolesEnPlace(slices.Clone(in), DefaultIsolationGapMS), DefaultMaxSpeedMPS, exemptions)
			if !slices.Equal(got, attendu) {
				t.Fatalf("enchaînement : en place %+v, copie %+v", got, attendu)
			}
		})
	}
}
