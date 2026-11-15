package decfilm_test

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// TestScanRosterListingsSansTableLisibleNeLitRien : une bobine sans `chunk_00` ne porte pas la largeur du bloc de
// personnalisation des paquets de type 8 : la facade ne rend aucune plage et le dit (`Scanned` faux), jamais une
// liste lue au hasard. La lecture des paquets eux-memes est tenue par les tests de la grammaire.
// MUTATIONS — une plage rendue sans table, `Scanned` vrai : ROUGES.
func TestScanRosterListingsSansTableLisibleNeLitRien(t *testing.T) {
	film, err := decfilm.LoadDir("../replay/testdata/minifilm_000d5950", nil)
	if err != nil {
		t.Fatal(err)
	}
	listings, stats := decfilm.ScanRosterListings(film)
	if stats.Scanned || len(listings) != 0 || stats.Packets != 0 {
		t.Fatalf("lecture sans table : %+v, %d plages", stats, len(listings))
	}
}

// TestScanUnitControlsSansRegistreRendUneErreur : la marche des trames exige le registre du film ; une bobine sans
// `chunk_00` rend l erreur de la grammaire et aucune lecture, jamais une liste vide presentee comme mesuree.
// MUTATIONS — l erreur avalee, `Scanned` vrai avec l erreur : ROUGES.
func TestScanUnitControlsSansRegistreRendUneErreur(t *testing.T) {
	film, err := decfilm.LoadDir("../replay/testdata/minifilm_000d5950", nil)
	if err != nil {
		t.Fatal(err)
	}
	reads, stats, err := decfilm.ScanUnitControls(decfilm.NewFilmContext(film))
	if err == nil || len(reads) != 0 || stats.Scanned {
		t.Fatalf("lecture sans registre : %v, %d lectures, %+v", err, len(reads), stats)
	}
}

// TestScanStatborgEntriesSansRegistreNeLitRien : une bobine sans `chunk_00` n'a pas de registre, donc pas d'archetype du
// tableau des scores : la facade ne rend aucune entree et le dit (`Scanned` faux).
// MUTATIONS — une entree rendue sans registre, `Scanned` vrai : ROUGES.
func TestScanStatborgEntriesSansRegistreNeLitRien(t *testing.T) {
	film, err := decfilm.LoadDir("../replay/testdata/minifilm_000d5950", nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, stats := decfilm.ScanStatborgEntries(decfilm.NewFilmContext(film))
	if stats.Scanned || len(entries) != 0 {
		t.Fatalf("lecture sans registre : %+v, %d entrees", stats, len(entries))
	}
}
