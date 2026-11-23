//go:build research

package replay

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// Sonde jetable E.1c, non commitee.
func TestZZE1c(t *testing.T) {
	ids := strings.Split(os.Getenv("E1C_IDS"), ",")
	racine := os.Getenv("E1C_RACINE")
	avant := os.Getenv("E1C_AVANT")
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data/titles/halo_infinite/reference/map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		blob, err := os.ReadFile(filepath.Join(racine, "data/cache/film_facts/halo_infinite", id+".filmfacts.bin"))
		if err != nil {
			t.Fatal(err)
		}
		head, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			t.Fatal(err)
		}
		var ff *FilmFactsFile
		var entry profile.MapQuantEntry
		var nom string
		for n, m := range cat.Maps {
			if m.Module != head.MapModule {
				continue
			}
			if f, e := DecodeFilmFactsFile(blob, m); e == nil {
				ff, entry, nom = f, m, n
				break
			}
		}
		if ff == nil {
			t.Fatalf("%s : aucune entree de catalogue ne relit les faits (module %s)", id, head.MapModule)
		}
		doc := BuildFromFacts(context.Background(), id, "halo_infinite", ff, Options{MapQuant: &entry})
		// avant
		var old struct {
			Eliminations []json.RawMessage `json:"eliminations"`
			ScoreActions []json.RawMessage `json:"scoreActions"`
			Roster       []RosterEntry     `json:"roster"`
		}
		fh, _ := os.Open(filepath.Join(avant, id+".json.gz"))
		zr, _ := gzip.NewReader(fh)
		_ = json.NewDecoder(zr).Decode(&old)
		fh.Close()
		var humNew []json.RawMessage
		bots, botVic, hum, vicHum := 0, 0, 0, 0
		perPlace := map[int]int{}
		for _, e := range doc.Eliminations {
			if e.XUID != "" {
				b, _ := json.Marshal(e)
				humNew = append(humNew, b)
				hum++
				if e.VictimPlace != nil {
					vicHum++
				}
			} else {
				bots++
				perPlace[*e.Place]++
			}
			if e.VictimPlace != nil {
				botVic++
			}
		}
		same := len(humNew) == len(old.Eliminations)
		diff := ""
		for i := range humNew {
			var a, b map[string]any
			_ = json.Unmarshal(humNew[i], &a)
			if i < len(old.Eliminations) {
				_ = json.Unmarshal(old.Eliminations[i], &b)
			}
			// la victime bot est le seul ecart permis (victimPlace absent avant)
			delete(a, "victimPlace")
			if !reflect.DeepEqual(a, b) {
				same = false
				diff = fmt.Sprintf("%v != %v", a, b)
				break
			}
		}
		teams := []int{}
		if tl := doc.ScoreTimeline; tl != nil {
			for _, tm := range tl.Teams {
				if n := len(tm.Total); n > 0 {
					teams = append(teams, tm.Total[n-1].V)
				}
			}
		}
		var cov EliminationCoverage
		if doc.Coverage.Eliminations != nil {
			cov = *doc.Coverage.Eliminations
		}
		sa := doc.Coverage.ScoreActions
		fmt.Printf("E1C %s carte=%s total=%d humains=%d botsAuteur=%d botsParPlace=%v victimesBot=%d humainsInchanges=%v %s equipes=%v cov=%+v scoreActions{scored=%d published=%d noRecipient=%d botRecipient=%d botTarget=%d complete=%v ancienScoreActions=%d nouveau=%d}\n",
			id, nom, len(doc.Eliminations), hum, bots, perPlace, botVic, same, diff, teams, cov,
			sa.Scored, sa.Published, sa.NoRecipient, sa.BotRecipient, sa.BotTarget, sa.Complete, len(old.ScoreActions), len(doc.ScoreActions))
	}
}
