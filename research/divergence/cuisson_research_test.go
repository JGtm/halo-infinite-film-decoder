//go:build research

package divergence

// cuisson_research_test.go : CUIT UN FILM VERS SES FAITS (`<id>.filmfacts.bin`), l entree de
// `TestFaitsDiff` (plan marche complete, A2.V et A2.H : differentiel de rejeu avant / apres un item, canal par
// canal). La cuisson est `replay.BuildFromFilmAvecFaits` sous l entree de catalogue du niveau, sans les
// options de l hote : c est la meme avant et apres, et seul le decodeur change entre les deux.
//
//	CUISSON_RACINE=<film_chunks> CUISSON_CARTE=<map_quant_bounds.json> CUISSON_FILM=<id> CUISSON_NIVEAU=<module> \
//	  CUISSON_SORTIE=<dossier> go test -tags=research ./research/divergence/ -run TestCuisson -v

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func TestCuisson(t *testing.T) {
	racine, chemin, id, niveau, sortie := os.Getenv("CUISSON_RACINE"), os.Getenv("CUISSON_CARTE"),
		os.Getenv("CUISSON_FILM"), os.Getenv("CUISSON_NIVEAU"), os.Getenv("CUISSON_SORTIE")
	if racine == "" || chemin == "" || id == "" || niveau == "" || sortie == "" {
		t.Skip("CUISSON_RACINE, CUISSON_CARTE, CUISSON_FILM, CUISSON_NIVEAU et CUISSON_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatal(err)
	}
	entree, err := entreeDuNiveau(cat, niveau)
	if err != nil {
		t.Fatal(err)
	}
	film, err := source.LoadDir(filepath.Join(racine, id), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, faits, err := replay.BuildFromFilmAvecFaits(context.Background(), id, "halo_infinite", film, replay.Options{MapQuant: entree})
	if err != nil {
		t.Fatal(err)
	}
	blob, err := replay.EncodeFilmFactsFile(faits)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sortie, 0o755); err != nil { //nolint:gosec // dossier de sortie de l operateur
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sortie, id+".filmfacts.bin"), blob, 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}
