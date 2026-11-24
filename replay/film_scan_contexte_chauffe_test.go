package replay

// film_scan_contexte_chauffe_test.go — UN CONTEXTE DEJA LU PAR L'HOTE DONNE LE MEME COMPTE DE REPLIS A TOUT
// NOMBRE DE FILS.
//
// Un hote qui lit les poses et les vehicules sur un contexte (`LireLesVehicules`, puis les memos du monde)
// avant de le passer en `Options.Contexte` y laisse des memos de pistes et de creations deja comptes au
// repli « apres la marche ». Les voies derivees les relevent de nouveau ; la fusion ne doit pas les recompter :
// le compte des replis, le document et les faits sont ceux d'un fil.

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// cuireSurUnContexteDejaLu rend les replis du contexte, le document et les faits d'une cuisson dont le
// contexte a ete lu par l'hote avant (marche commune, poses, vehicules).
func cuireSurUnContexteDejaLu(t *testing.T, dossier string, fils int) (grammar.ComptesDesReplis, []byte, []byte) {
	t.Helper()
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dossier, err)
	}
	entry := goldenEntryPourTest(t)
	fc := grammar.NewFilmContextForMap(film, &entry, nil)
	if _, err := grammar.LireLaMarcheCommune(fc); err != nil {
		t.Fatalf("marche commune : %v", err)
	}
	wr := entry.Range()
	decodeFilmPlacementsLecture(fc, &wr)
	if _, err := LireLesVehicules(fc, &wr); err != nil {
		t.Fatalf("lecture des vehicules : %v", err)
	}
	d, f, err := BuildFromFilmAvecFaits(context.Background(), "bobine", "halo_infinite", film,
		Options{MapQuant: &entry, Contexte: fc, Vitesse: OptionsVitesse{Fils: fils}})
	if err != nil {
		t.Fatalf("cuisson a %d fils : %v", fils, err)
	}
	doc, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("document : %v", err)
	}
	faits, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	return fc.ComptesDesReplis(), doc, faits
}

// TestFilsContexteDejaLuMemesReplisATousLesFils : replis du contexte, document et faits identiques a 1, 2, 4
// et 8 fils quand l'hote a lu le contexte avant la cuisson.
func TestFilsContexteDejaLuMemesReplisATousLesFils(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(max(runtime.GOMAXPROCS(0), MaxFils)))
	dossier := "../internal/facts/killsource/testdata/minibobine_000d5950"
	replis1, doc1, faits1 := cuireSurUnContexteDejaLu(t, dossier, 1)
	if replis1.PistesDuMondeApresLaMarche == 0 {
		t.Fatalf("la bobine ne compte aucun repli de pistes du monde : le test ne prouve rien")
	}
	for _, fils := range []int{2, 4, 8} {
		replis, doc, faits := cuireSurUnContexteDejaLu(t, dossier, fils)
		if replis != replis1 {
			t.Errorf("replis a %d fils : %+v, a 1 fil : %+v", fils, replis, replis1)
		}
		if !bytes.Equal(doc, doc1) || !bytes.Equal(faits, faits1) {
			t.Errorf("la sortie a %d fils differe de celle a 1 fil", fils)
		}
	}
}
