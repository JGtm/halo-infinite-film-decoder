package replay

// film_scan_fils_documents_test.go — LE DOCUMENT ET LES FAITS ENTIERS NE DEPENDENT PAS DU NOMBRE DE FILS.
//
// Les tests des voies comparent les etapes observees sur des bobines que le balayage des positions refuse.
// Celui-ci cuit, de bout en bout, des bobines que ce balayage ACCEPTE (la chaine d apres les positions :
// tirs, images-cles, naissances, portage, capacites, voies) et compare, a 1, 2, 4 et 8 fils, le document
// serialise et les faits encodes, octet a octet.
//
// Les bobines sont des prefixes de film du depot : on n y juge pas la valeur des champs, seulement leur
// egalite d un nombre de fils a l autre. Une bobine que le balayage refuserait fait echouer le test (elle
// ne prouverait rien) : le test ne saute jamais.

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// bobinesCuitesDeBoutEnBout : des prefixes de film que le balayage des positions accepte.
var bobinesCuitesDeBoutEnBout = []string{
	"../testdata/corpus_objectifs/minibobine_objectifs/koth_0a247154",
	"../testdata/corpus_objectifs/minibobine_objectifs/strongholds_7344d24f",
	"../internal/facts/killsource/testdata/minibobine_000d5950",
}

// cuireLaBobine rend le document serialise et les faits encodes d une cuisson complete au nombre de
// fils demande, sur un contexte neuf.
func cuireLaBobine(t *testing.T, dossier string, fils int) (doc, faits []byte) {
	t.Helper()
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dossier, err)
	}
	entry := goldenEntryPourTest(t)
	d, f, err := BuildFromFilmAvecFaits(context.Background(), "bobine", "halo_infinite", film, Options{
		MapQuant: &entry, Vitesse: OptionsVitesse{Fils: fils},
	})
	if err != nil {
		t.Fatalf("cuisson de %s a %d fils : %v (la bobine ne prouve rien)", dossier, fils, err)
	}
	if doc, err = json.Marshal(d); err != nil {
		t.Fatalf("serialisation du document : %v", err)
	}
	if faits, err = EncodeFilmFactsFile(f); err != nil {
		t.Fatalf("encodage des faits : %v", err)
	}
	return doc, faits
}

// TestFilsDocumentEtFaitsIdentiquesDeBoutEnBout : la sortie entiere d une cuisson est la meme a 1, 2, 4 et
// 8 fils.
func TestFilsDocumentEtFaitsIdentiquesDeBoutEnBout(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(max(runtime.GOMAXPROCS(0), MaxFils)))
	for _, dossier := range bobinesCuitesDeBoutEnBout {
		doc1, faits1 := cuireLaBobine(t, dossier, 1)
		if len(doc1) == 0 || len(faits1) == 0 {
			t.Fatalf("%s : sortie vide a un fil", dossier)
		}
		for _, fils := range []int{2, 4, 8} {
			doc, faits := cuireLaBobine(t, dossier, fils)
			if !bytes.Equal(doc, doc1) {
				t.Errorf("%s : le document a %d fils differe de celui a 1 fil (%d / %d octets)", dossier, fils, len(doc), len(doc1))
			}
			if !bytes.Equal(faits, faits1) {
				t.Errorf("%s : les faits a %d fils different de ceux a 1 fil (%d / %d octets)", dossier, fils, len(faits), len(faits1))
			}
		}
	}
}
