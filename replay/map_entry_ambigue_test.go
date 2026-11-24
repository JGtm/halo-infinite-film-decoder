package replay

import (
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// TestResoudreCarteDuCatalogueAmbigue verrouille le choix de carte par signature : deux cartes
// qui la portent font échouer la résolution avec la même erreur à chaque appel (sans choix
// aléatoire), et une seule carte est rendue.
func TestResoudreCarteDuCatalogueAmbigue(t *testing.T) {
	lay := profile.I0Layout{AxisW: [3]uint{15, 15, 17}, Region: 0}
	entree := profile.MapQuantEntry{Module: "module_test", AxisWidths: [3]uint{15, 15, 17}}

	t.Run("deux cartes de meme signature", func(t *testing.T) {
		cat := &profile.MapQuantCatalog{Maps: map[string]profile.MapQuantEntry{
			"carte_b": entree,
			"carte_a": entree,
		}}
		_, _, err := resoudreCarteDuCatalogue(cat, lay)
		if err == nil {
			t.Fatal("deux cartes de meme signature : erreur attendue, aucun choix ne doit etre fait")
		}
		premiere := err.Error()
		if !strings.Contains(premiere, "carte_a, carte_b") {
			t.Fatalf("l'erreur doit nommer les cartes triees par nom, obtenu : %s", premiere)
		}
		for i := range 100 {
			_, _, err := resoudreCarteDuCatalogue(cat, lay)
			if err == nil || err.Error() != premiere {
				t.Fatalf("appel %d : erreur differente (%v), attendu %q", i, err, premiere)
			}
		}
	})

	t.Run("une seule carte de la signature", func(t *testing.T) {
		cat := &profile.MapQuantCatalog{Maps: map[string]profile.MapQuantEntry{
			"carte_seule": entree,
			"autre_carte": {Module: "autre", AxisWidths: [3]uint{10, 10, 10}},
		}}
		for i := range 100 {
			nom, got, err := resoudreCarteDuCatalogue(cat, lay)
			if err != nil {
				t.Fatalf("appel %d : erreur inattendue : %v", i, err)
			}
			if nom != "carte_seule" || got.Module != "module_test" {
				t.Fatalf("appel %d : carte %q (module %q), attendu carte_seule", i, nom, got.Module)
			}
		}
	})
}
