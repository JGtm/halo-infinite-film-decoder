package depottest

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// plafondDeLignes est la taille au-dela de laquelle un fichier Go se scinde.
const plafondDeLignes = 500

// plafondsGeles nomme les fichiers qui depassent [plafondDeLignes] et leur taille exacte, gelee le
// 2026-11-12. La liste ne s allonge pas : un fichier neuf tient sous le plafond. Une entree ne monte
// que pour `replay/document_chronicle.go` et `replay/structure_test.go`, qui gagnent des lignes a
// chaque version du schema de publication (une entree de chronique, une forme figee) ; leur montee
// porte sa date ici. Une entree dont le fichier a maigri se baisse ou se retire dans le meme commit.
var plafondsGeles = map[string]int{
	"internal/facts/killsource/e197_identite_paquet_mesure_research_test.go": 651,
	"internal/facts/objectives/assaut_pied_ancre_test.go":                    558,
	"internal/facts/objectives/e1911_manches_mesure_research_test.go":        525,
	"internal/grammar/components_hooks_test.go":                              600,
	"internal/grammar/golden_minibobine_test.go":                             549,
	"internal/grammar/ground_weapon_lifecycle_research_test.go":              566,
	"internal/grammar/i59_anchor_test.go":                                    1138,
	"internal/grammar/imagecle_oracle_n2_research_test.go":                   502,
	"internal/grammar/inventory_position_i22_test.go":                        824,
	"internal/grammar/vehicules_v13_deadstate_test.go":                       580,
	"internal/grammar/vehicules_v2_items_test.go":                            533,
	"internal/grammar/vitesse_filtre_outils_research_test.go":                501,
	"killicon/killicon_test.go":                                              610,
	"replay/assaut_manches_research_test.go":                                 635,
	"replay/attachement_phase0_bord_test.go":                                 522,
	"replay/closures_test.go":                                                598,
	"replay/ctf_retour_zone_research_test.go":                                806,
	"replay/document_chronicle.go":                                           3853, // 2026-11-12, schema 110
	"replay/equipment_uses_join_test.go":                                     573,
	"replay/golden_assembly_test.go":                                         1184,
	"replay/ground_link_research_test.go":                                    813,
	"replay/minifilm_test.go":                                                579,
	"replay/structure_test.go":                                               1498, // 2026-11-12, schema 110
	"replay/t0_mouvement_research_test.go":                                   868,
	"replay/vehicules_v1a_test.go":                                           577,
	"replay/visee_larg114_research_test.go":                                  501,
}

// racineDuModule rend le dossier qui porte `go.mod`, deux niveaux au-dessus de ce fichier.
func racineDuModule(t *testing.T) string {
	t.Helper()
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue : racine du module indeductible")
	}
	racine := filepath.Join(filepath.Dir(ce), "..", "..")
	if _, err := os.Stat(filepath.Join(racine, "go.mod")); err != nil {
		t.Fatalf("pas de go.mod sous %s : %v", racine, err)
	}
	return racine
}

// lignesDesFichiersGo rend le nombre de lignes de chaque fichier Go du module, par chemin relatif a
// barres obliques ; les dossiers caches et `testdata` ne sont pas parcourus.
func lignesDesFichiersGo(t *testing.T, racine string) map[string]int {
	t.Helper()
	lignes := map[string]int{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		nom := d.Name()
		if d.IsDir() {
			if chemin != racine && (strings.HasPrefix(nom, ".") || nom == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(nom, ".go") {
			return nil
		}
		octets, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(racine, chemin)
		if err != nil {
			return err
		}
		lignes[filepath.ToSlash(rel)] = compterLesLignes(octets)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}
	return lignes
}

// compterLesLignes compte les lignes comme `wc -l`, plus une derniere ligne sans saut final.
func compterLesLignes(octets []byte) int {
	n := bytes.Count(octets, []byte{'\n'})
	if len(octets) > 0 && octets[len(octets)-1] != '\n' {
		n++
	}
	return n
}

// TestAucunFichierGoNeDepasseSonPlafond : un fichier Go tient en [plafondDeLignes] lignes, sauf ceux de
// [plafondsGeles], qui ne grossissent pas ; et la liste ne garde aucune entree perimee.
func TestAucunFichierGoNeDepasseSonPlafond(t *testing.T) {
	lignes := lignesDesFichiersGo(t, racineDuModule(t))
	if len(lignes) == 0 {
		t.Fatal("aucun fichier Go trouve : le parcours ne voit pas le module")
	}
	for chemin, n := range lignes {
		gele, connu := plafondsGeles[chemin]
		switch {
		case !connu && n > plafondDeLignes:
			t.Errorf("%s : %d lignes, plafond %d. Le scinder ; la liste des plafonds geles ne s allonge pas.", chemin, n, plafondDeLignes)
		case connu && n > gele:
			t.Errorf("%s : %d lignes, plafond gele %d. Le fichier ne grossit pas (voir plafondsGeles).", chemin, n, gele)
		case connu && n < gele:
			t.Errorf("%s : %d lignes sous son plafond gele %d. Baisser l entree a %d, ou la retirer sous %d.", chemin, n, gele, n, plafondDeLignes+1)
		}
	}
	for chemin := range plafondsGeles {
		if _, existe := lignes[chemin]; !existe {
			t.Errorf("%s : entree de plafondsGeles sans fichier. La retirer.", chemin)
		}
	}
}

// TestCompterLesLignes : une derniere ligne sans saut final compte.
func TestCompterLesLignes(t *testing.T) {
	for texte, veut := range map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\nb\n": 2} {
		if n := compterLesLignes([]byte(texte)); n != veut {
			t.Errorf("%q : %d lignes, attendu %d", texte, n, veut)
		}
	}
}
