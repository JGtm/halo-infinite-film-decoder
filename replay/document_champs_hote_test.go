package replay

// document_champs_hote_test.go — LES CHAMPS QUE L'HOTE REMPLIT A LA REQUETE DANS UN TYPE CUIT.
//
// LE CAS. `calquesALaRequete` (document_shape_test.go) exclut de l'empreinte cuite des CALQUES
// RACINE que la cuisson n'écrit jamais. Un champ réservé à l'hôte DANS un type que la cuisson
// écrit par ailleurs (`VehicleDestruction.CauseKey`, comme `WeaponLabel.Key` avant lui) n'entre
// pas dans cette table : sans le traitement ci-dessous, l'ajouter exigerait de monter
// `SchemaVersion`, donc de faire lire « à re-cuire » tout le parc pour un champ qu'aucun
// artefact ne porte.
//
// CE QUI EST GARDÉ. `empreinte-stockee` et le golden de forme en clair couvrent TOUJOURS le
// champ : l'ajout reste visible et passe en revue. Seule l'empreinte CUITE, qui gouverne le refus
// de régénération, l'ignore. Et ce test rougit si la cuisson l'écrit un jour : un champ « à la
// requête » que la cuisson pose ferait entrer dans l'artefact un contenu que le ratchet ne voit
// plus.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// champsRemplisParLHote : « Type.Champ » -> la justification datée. Ces champs sont absents de
// l'empreinte cuite, et la cuisson ne les écrit jamais.
var champsRemplisParLHote = map[string]string{
	"VehicleDestruction.CauseKey": "2026-11-13 — weapon_key de la cause d'une destruction de véhicule, d'où l'hôte tire le pictogramme ; rempli à la requête, comme WeaponLabel.Key",
}

// champsCuitsDuType : les champs exportés d'un type, rendus et triés comme [champsDe], PRIVÉS
// des champs que l'hôte remplit à la requête.
func champsCuitsDuType(t reflect.Type) []string {
	out := make([]string, 0, t.NumField())
	for f := range t.Fields() {
		if f.PkgPath != "" || champsRemplisParLHote[t.Name()+"."+f.Name] != "" {
			continue
		}
		out = append(out, ligneDeChamp(f))
	}
	sort.Strings(out)
	return out
}

// champHote retrouve le champ « Type.Champ » parmi les types atteignables du document.
func champHote(t *testing.T, cle string) reflect.StructField {
	t.Helper()
	nomType, nomChamp, _ := strings.Cut(cle, ".")
	types := map[string]reflect.Type{}
	collecterTypes(reflect.TypeFor[ReplayDocument](), types, map[reflect.Type]bool{})
	typ, ok := types[nomType]
	if !ok {
		t.Fatalf("%s : le type %q n'est plus atteignable depuis le document — retirer l'entree", cle, nomType)
	}
	f, ok := typ.FieldByName(nomChamp)
	if !ok {
		t.Fatalf("%s : le champ %q n'existe plus — retirer l'entree", cle, nomChamp)
	}
	return f
}

// TestChampsRemplisParLHoteSontOmitemptyEtDates : chaque entrée nomme un champ réel, de type
// chaîne, `omitempty` (un champ absent de l'artefact ne doit rien écrire sur le fil), et porte sa
// justification datée.
func TestChampsRemplisParLHoteSontOmitemptyEtDates(t *testing.T) {
	if len(champsRemplisParLHote) == 0 {
		t.Fatal("`champsRemplisParLHote` est vide : le test ne garde plus rien")
	}
	for cle, raison := range champsRemplisParLHote {
		f := champHote(t, cle)
		if f.Type.Kind() != reflect.String {
			t.Errorf("%s : type %s, une clé remplie par l'hote est une chaine", cle, f.Type)
		}
		if _, opts, _ := strings.Cut(f.Tag.Get("json"), ","); opts != "omitempty" {
			t.Errorf("%s : balise json %q, attendu l'option omitempty seule", cle, f.Tag.Get("json"))
		}
		if !baliseDatee.MatchString(raison) {
			t.Errorf("%s : justification non datee (attendu « AAAA-MM-JJ — raison ») : %q", cle, raison)
		}
	}
}

// TestChampsRemplisParLHoteRestentDansLaFormeEntiere : le champ est ABSENT de l'empreinte cuite et
// PRESENT dans la forme entière (stockée et servie) : il ne disparaît jamais du golden en clair.
func TestChampsRemplisParLHoteRestentDansLaFormeEntiere(t *testing.T) {
	entiere := documentShapeRender(reflect.TypeFor[ReplayDocument]())
	cuite := documentShapeRenderCuite()
	for cle := range champsRemplisParLHote {
		f := champHote(t, cle)
		ligne := ligneDeChamp(f)
		if !strings.Contains(entiere, "  "+ligne+"\n") {
			t.Errorf("%s : absent de la forme entiere — le golden en clair ne le montrerait plus", cle)
		}
		if strings.Contains(cuite, "  "+ligne+"\n") {
			t.Errorf("%s : present dans l'empreinte cuite, alors que la cuisson ne l'ecrit pas", cle)
		}
	}
}

// TestLaCuissonNEcritJamaisLaCleDeCauseDeDestruction : LE TEST QUI ROUGIT SI LA CUISSON ÉCRIT LE
// CHAMP. Une destruction dont la cause est RÉSOLUE (le cas où une clé aurait un sens) est cuite par
// la vraie pose ; `causeKey` ne doit ni être posé ni sortir sur le fil.
func TestLaCuissonNEcritJamaisLaCleDeCauseDeDestruction(t *testing.T) {
	connue := causeConnue(t)
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 3, connue),
		rideDeTest(250, 282, xuidConducteur, 0)))
	attachVehicleDestructions(context.Background(), doc, vehicleDestructionInputs{roster: rosterDeTest()})
	d := doc.Vehicles[0].Destruction
	if d == nil || d.Cause == nil || *d.Cause != connue {
		t.Fatalf("pre-condition : la destruction doit porter la cause %d, obtenu %+v", connue, d)
	}
	if d.CauseKey != "" {
		t.Errorf("la cuisson a ecrit CauseKey = %q : ce champ est rempli par l'hote a la requete", d.CauseKey)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if strings.Contains(string(raw), "causeKey") {
		t.Errorf("la cle `causeKey` sort sur le fil de la cuisson : %s", raw)
	}
}

// ecritLeChampHote reconnaît, dans une source Go, une AFFECTATION du champ nommé : `.Champ =` ou
// `Champ:` dans un littéral.
func ecritLeChampHote(src, champ string) bool {
	return regexp.MustCompile(`\.` + champ + `\s*=[^=]|\b` + champ + `\s*:`).MatchString(src)
}

// TestAucunFichierDeProductionNEcritLesChampsDeLHote : la garde de source, qui complète le test
// de comportement (une écriture sur une branche que ce test ne parcourt pas passerait entre les
// deux). Les déclarations de type ne sont pas des affectations.
func TestAucunFichierDeProductionNEcritLesChampsDeLHote(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("balayage des fichiers du paquet : %v", err)
	}
	var lus int
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		lus++
		for cle := range champsRemplisParLHote {
			_, champ, _ := strings.Cut(cle, ".")
			// les declarations `CauseKey string ...` ne sont ni `.CauseKey =` ni `CauseKey:`
			if ecritLeChampHote(string(src), champ) {
				t.Errorf("%s ecrit %s a la CUISSON : ce champ est rempli par l'hote a la requete — "+
					"le retirer de la cuisson, ou le sortir de `champsRemplisParLHote` et remonter SchemaVersion",
					f, cle)
			}
		}
	}
	if lus < 20 {
		t.Fatalf("balayage muet : %d fichiers de production lus", lus)
	}
}
