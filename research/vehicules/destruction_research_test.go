//go:build research

package vehicules

// destruction_research_test.go : INSTRUMENT DE MESURE DE LA DESTRUCTION DE VEHICULE PUBLIEE.
//
// Il cuit UN film par la chaine de production (`replay.BuildFromFilm`) et ecrit, dans le dossier de
// sortie, `<id8>.json` : les vies de vehicule, le roster et la couverture des vehicules du document.
// Le meme instrument sert avant et apres un changement du calque : les deux sorties se comparent.
//
//	DEST_RACINE   dossier `film_chunks` (un sous-dossier par id8)
//	DEST_CARTE    catalogue `map_quant_bounds.json`
//	DEST_FILM     id8:niveau (le niveau est le `module` du catalogue, ex. fo05_desert)
//	DEST_SORTIE   dossier de sortie

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// sortieDeCuisson est ce que l instrument garde du document : ce dont la mesure a besoin.
type sortieDeCuisson struct {
	ID              string                  `json:"id"`
	Niveau          string                  `json:"niveau"`
	SchemaVersion   int                     `json:"schemaVersion"`
	FrameIntervalMS int                     `json:"frameIntervalMs"`
	FrameCount      int                     `json:"frameCount"`
	OriginMs        *int64                  `json:"originMs,omitempty"`
	Roster          []replay.RosterEntry    `json:"roster"`
	Vehicles        []replay.VehicleTrack   `json:"vehicles"`
	Coverage        *replay.VehicleCoverage `json:"vehicleCoverage"`
	Fallbacks       any                     `json:"fallbacks,omitempty"`
	CookMs          int64                   `json:"cookMs"`
	// CalageMS, VehDeaths et FeedDeaths sont les entrees brutes de la mesure : le calage du fil des
	// morts sur l horloge du film, les morts de vehicule que la marche a lues, le fil des morts.
	CalageMS   *int64              `json:"calageMs,omitempty"`
	VehDeaths  []types.ObjectDeath `json:"vehDeaths,omitempty"`
	FeedDeaths []types.Death       `json:"feedDeaths,omitempty"`
}

func TestCuireLesVehicules(t *testing.T) {
	racine, carte, film, sortie := os.Getenv("DEST_RACINE"), os.Getenv("DEST_CARTE"), os.Getenv("DEST_FILM"), os.Getenv("DEST_SORTIE")
	if racine == "" || carte == "" || film == "" || sortie == "" {
		t.Skip("DEST_RACINE / DEST_CARTE / DEST_FILM / DEST_SORTIE absents : instrument de mesure saute")
	}
	id8, niveau, _ := strings.Cut(film, ":")
	if err := os.MkdirAll(sortie, 0o755); err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(carte)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	noms := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	var entree *profile.MapQuantEntry
	for _, n := range noms {
		if e := cat.Maps[n]; e.Module == niveau {
			c := e
			entree = &c
			break
		}
	}
	if entree == nil {
		t.Fatalf("aucune entree de catalogue pour le module %q", niveau)
	}
	f, err := source.LoadDir(filepath.Join(racine, id8), nil)
	if err != nil {
		t.Fatalf("film %s : %v", id8, err)
	}
	t0 := time.Now()
	doc, faits, err := replay.BuildFromFilmAvecFaits(context.Background(), id8, "halo_infinite", f, replay.Options{MapQuant: entree})
	if err != nil {
		t.Fatalf("cuisson de %s : %v", id8, err)
	}
	out := sortieDeCuisson{
		ID: id8, Niveau: niveau, SchemaVersion: doc.SchemaVersion, FrameIntervalMS: doc.FrameIntervalMS,
		FrameCount: doc.FrameCount, OriginMs: doc.OriginMs, Roster: doc.Roster, Vehicles: doc.Vehicles,
		CookMs: time.Since(t0).Milliseconds(),
	}
	if faits != nil {
		out.VehDeaths, out.FeedDeaths = faits.Facts.Vehicles.Deaths, faits.Facts.Deaths
	}
	if doc.Coverage != nil {
		out.Coverage = doc.Coverage.Vehicles
		out.Fallbacks = doc.Coverage.Fallbacks
		out.CalageMS = doc.Coverage.Bridge.DeathOffsetMs
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sortie, id8+".json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s : %d vies, %d octets, %d ms", id8, len(doc.Vehicles), len(b), out.CookMs)
}
