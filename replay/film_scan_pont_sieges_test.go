package replay

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/filmtest"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// TestLaCuissonLitLaTableAuxSiegesDeLaListe : sur un film dont deux lignes de la table sont permutees, la lecture des
// sieges et des occupants de la cuisson donne a chaque occupant le siege que les paquets de type 8 lui donnent, pas le
// rang de sa ligne. Le sync emploie la meme lecture (`TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants`).
// MUTATIONS — la table brute dans `lireSiegesEtOccupants` : ROUGE.
func TestLaCuissonLitLaTableAuxSiegesDeLaListe(t *testing.T) {
	const dossier = "../internal/facts/killsource/testdata/minibobine_000d5950"
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatal(err)
	}
	filmtest.PermuterDeuxLignesDeLaTable(t, film)
	so := lireSiegesEtOccupants(context.Background(), film, grammar.NewFilmContext(film), "minibobine", nil)
	listes := map[uint64]int{}
	for _, l := range so.listages {
		listes[l.XUID] = l.Slot
	}
	if len(so.table.Seats) == 0 || !so.statsListages.Scanned {
		t.Fatalf("%d sieges, listes lues %v", len(so.table.Seats), so.statsListages.Scanned)
	}
	for _, s := range so.table.Seats {
		if s.FilmIndex != listes[s.XUID] {
			t.Fatalf("xuid %d : siege %d, la liste dit %d", s.XUID, s.FilmIndex, listes[s.XUID])
		}
	}
}
