//go:build research

package grammar

// entete_delta_films_research_test.go — LES QUATRE BALAYAGES D'EN-TETE DELTA, CONFRONTES A DE VRAIS
// FILMS : sur CHAQUE payload delta, chaque balayage de production rend ce que rend sa forme position
// par position (ancrage des bipedes, bandes d'objets du monde, visees, creations).
//
// Les films ne sont pas dans le depot. L'operateur donne leur correspondance (la meme que celle de
// `cmd/film-bench`), le catalogue des bornes de carte et les noms retenus :
//
//	ENTETE_DELTA_CORRESPONDANCE=<correspondance.tsv> ENTETE_DELTA_CATALOGUE=<map_quant_bounds.json> \
//	  ENTETE_DELTA_FILMS=film06,film09 \
//	  go test -tags=research -count=1 -timeout 60m -run '^TestLesEntetesDeltaSurDeVraisFilms$' ./internal/grammar/

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestLesEntetesDeltaSurDeVraisFilms(t *testing.T) {
	corr, catalogue, noms := os.Getenv("ENTETE_DELTA_CORRESPONDANCE"), os.Getenv("ENTETE_DELTA_CATALOGUE"),
		os.Getenv("ENTETE_DELTA_FILMS")
	if corr == "" || catalogue == "" || noms == "" {
		t.Skip("ENTETE_DELTA_CORRESPONDANCE, ENTETE_DELTA_CATALOGUE et ENTETE_DELTA_FILMS sont requis")
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue illisible : %v", err)
	}
	lignes := lireLaCorrespondance(t, corr)
	for _, nom := range strings.Split(noms, ",") {
		l, ok := lignes[strings.TrimSpace(nom)]
		if !ok {
			t.Fatalf("film %q absent de la correspondance", nom)
		}
		t.Run(nom, func(t *testing.T) { comparerLesEntetesDUnFilm(t, ouvrirUnFilm(t, l, cat)) })
	}
}

// bilanDesEntetes compte, par balayage, les payloads juges et les acceptations de la reference.
type bilanDesEntetes struct{ payloads, bipedes, echantillons, visees, creations int }

func comparerLesEntetesDUnFilm(t *testing.T, fc *FilmContext) {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage d'i0 : %v", err)
	}
	fc.PoserLargeursObjetDuMondeDepuisDecoupage(lay)
	wr := profile.QuantRangeCEBiped()
	lg := fc.ProfilDeBalayage().LargeursObjetDuMonde()
	slots, ctx := fc.BipedSlots(), fc.ContexteDeLecture()
	var bandes [][]uint32
	for _, ti := range archetypesAPistes() {
		bandes = append(bandes, slotsDeLaBande(worldObjectSlotBand(fc, ti)))
	}
	app := appartenanceDesBandes(bandes)
	var marches []equipCreationWalk
	for _, ti := range archetypesDeCreation() {
		if band := worldObjectSlotBand(fc, int(ti)); len(band) > 0 {
			w, err := fc.marcheDeCreation(ti, &wr, band)
			if err != nil {
				t.Fatalf("marche de creation ti=%d : %v", ti, err)
			}
			marches = append(marches, w)
		}
	}
	var bilan bilanDesEntetes
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay, gens, total := pk.Payload(data), fc.GenerationsVivantesA(pk.TimestampUS), len(pk.Payload(data))*8
			bilan.payloads++
			var wantB, gotB []enregistrementBipede
			walkDeltaBipedPayloadJusquaReference(pay, slots, lay, gens, total, func(d deltaBipedRecord) {
				wantB = append(wantB, enregistrementBipede{}.de(d))
			})
			walkDeltaBipedPayloadJusqua(pay, slots, lay, gens, total, func(d deltaBipedRecord) {
				gotB = append(gotB, enregistrementBipede{}.de(d))
			})
			if !reflect.DeepEqual(wantB, gotB) {
				t.Fatalf("chunk %d paquet %d : ancrage des bipedes different (%d, reference %d)", c, pk.Index, len(gotB), len(wantB))
			}
			bilan.bipedes += len(wantB)
			wantE := echantillonsDesBandesReference(pay, app, len(bandes), &wr, lg, total)
			if !reflect.DeepEqual(wantE, echantillonsDesBandes(pay, app, len(bandes), &wr, lg, total)) {
				t.Fatalf("chunk %d paquet %d : bandes d'objets differentes", c, pk.Index)
			}
			for _, e := range wantE {
				bilan.echantillons += len(e)
			}
			wantV := scanBipedAimRecordsJusquaReference(pay, slots, gens, ctx, total)
			if !reflect.DeepEqual(wantV, scanBipedAimRecordsJusqua(pay, slots, gens, ctx, total)) {
				t.Fatalf("chunk %d paquet %d : visees differentes", c, pk.Index)
			}
			bilan.visees += len(wantV)
			bilan.creations += len(comparerLesCreations(t, marches, pay, lieuDuPaquet{chunk: c, pk: pk}, total))
		}
	}
	if bilan.payloads == 0 || bilan.bipedes == 0 {
		t.Fatalf("aucune acceptation exercee : %+v", bilan)
	}
	t.Logf("%d payloads delta ; acceptations de la reference : %d records bipedes, %d echantillons d'objets, %d visees, %d creations",
		bilan.payloads, bilan.bipedes, bilan.echantillons, bilan.visees, bilan.creations)
}
