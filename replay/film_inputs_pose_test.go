package replay

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// champsDesFaitsQuApplyToNePosePas : les champs de `FilmInputs` que `applyTo` ne verse PAS dans
// `Options`, et ce qui les consomme a la place. Tout autre champ doit y etre pose.
var champsDesFaitsQuApplyToNePosePas = map[string]string{
	"Positions": "argument `positions` de BuildFromPositions",
	"Fire":      "argument `fire` de BuildFromPositions",
	"ZoomEvents": "reduit en la fermeture `Options.Scoped` par buildScopedLookup : une fermeture " +
		"ne se compare pas, la lunette est verifiee par ses propres tests",
}

// TestApplyToPoseChaqueChampDesFaits : chaque champ de `FilmInputs` se retrouve dans `Options`
// apres `applyTo`.
//
// L ENTREE EST FABRIQUEE PAR REFLEXION : chaque champ recoit une valeur distincte (une tranche
// d un element, un entier propre au champ), puis on cherche ces valeurs dans les options. Retirer
// la ligne d assemblage d un calque (repliques, gestes, arme tiree, collages, vitalites, apparences,
// commandes de controle...) laisse son champ introuvable : le test nomme le champ perdu.
// UN CHAMP NEUF DE `FilmInputs` EST EXIGE ICI : il est pose, ou il figure a la table ci-dessus
// avec sa raison.
func TestApplyToPoseChaqueChampDesFaits(t *testing.T) {
	var in FilmInputs
	compteur := uint64(1 << 40)
	empreintesParChamp := map[string][]any{}
	vin := reflect.ValueOf(&in).Elem()
	for i := range vin.NumField() {
		f := vin.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		var marques []any
		remplirAvecMarques(vin.Field(i), &compteur, &marques)
		empreintesParChamp[f.Name] = marques
	}

	var opt Options
	in.applyTo(&opt)

	vus := map[any]bool{}
	collecterEmpreintes(reflect.ValueOf(&opt).Elem(), vus, map[uintptr]bool{})

	var perdus []string
	for nom, marques := range empreintesParChamp {
		if _, exempte := champsDesFaitsQuApplyToNePosePas[nom]; exempte {
			continue
		}
		trouve := false
		for _, m := range marques {
			if vus[m] {
				trouve = true
				break
			}
		}
		if len(marques) == 0 {
			trouve = champBooleenPose(t, nom)
		}
		if !trouve {
			perdus = append(perdus, nom)
		}
	}
	sort.Strings(perdus)
	for _, nom := range perdus {
		t.Errorf("FilmInputs.%s n est pose nulle part dans Options par applyTo : le calque qu il porte "+
			"disparait du document sans qu aucun autre test ne le voie", nom)
	}
	for nom := range champsDesFaitsQuApplyToNePosePas {
		if _, ok := empreintesParChamp[nom]; !ok {
			t.Errorf("la table d exceptions nomme %q, qui n est plus un champ de FilmInputs", nom)
		}
	}
}

// champBooleenPose verifie un champ qui ne porte que des booleens ou de petits entiers (aucune
// marque distinctive possible) : le mettre a vrai, seul, doit changer les options.
func champBooleenPose(t *testing.T, nom string) bool {
	t.Helper()
	sansFermeture := func(in FilmInputs) Options {
		var o Options
		in.applyTo(&o)
		o.Scoped = nil
		return o
	}
	base := sansFermeture(FilmInputs{})
	var in FilmInputs
	allume(reflect.ValueOf(&in).Elem().FieldByName(nom))
	return !reflect.DeepEqual(base, sansFermeture(in))
}

// allume met a vrai les booleens et a un les petits entiers d une valeur.
func allume(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int8, reflect.Int16, reflect.Uint8, reflect.Uint16:
		if v.CanInt() {
			v.SetInt(1)
		} else {
			v.SetUint(1)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				allume(v.Field(i))
			}
		}
	}
}

// remplirAvecMarques pose une valeur distincte dans chaque feuille posable et rend, pour chacune,
// la marque qu on cherchera : le pointeur de donnees d une tranche, l adresse d un pointeur, la
// valeur d un entier de 32 bits ou plus, d un flottant ou d une chaine.
func remplirAvecMarques(v reflect.Value, compteur *uint64, marques *[]any) {
	switch v.Kind() {
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		v.Set(s)
		*marques = append(*marques, struct {
			p uintptr
			n int
		}{v.Pointer(), 1})
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		*marques = append(*marques, v.Pointer())
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		*marques = append(*marques, v.Pointer())
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
		*compteur++
		if v.CanInt() {
			v.SetInt(int64(*compteur))
			*marques = append(*marques, int64(*compteur))
		} else {
			v.SetUint(*compteur)
			*marques = append(*marques, *compteur)
		}
	case reflect.Float32, reflect.Float64:
		*compteur++
		v.SetFloat(float64(*compteur))
		*marques = append(*marques, float64(*compteur))
	case reflect.String:
		*compteur++
		s := fmt.Sprintf("marque-%d", *compteur)
		v.SetString(s)
		*marques = append(*marques, s)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				remplirAvecMarques(v.Field(i), compteur, marques)
			}
		}
	}
}

// collecterEmpreintes recense, dans une valeur d options, les memes marques que
// remplirAvecMarques. Elle descend dans les structures et les pointeurs, jamais dans les
// elements d une tranche ni dans une fermeture.
func collecterEmpreintes(v reflect.Value, vus map[any]bool, visites map[uintptr]bool) {
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() > 0 {
			vus[struct {
				p uintptr
				n int
			}{v.Pointer(), v.Len()}] = true
		}
	case reflect.Map:
		if !v.IsNil() {
			vus[v.Pointer()] = true
		}
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		vus[v.Pointer()] = true
		if visites[v.Pointer()] {
			return
		}
		visites[v.Pointer()] = true
		collecterEmpreintes(v.Elem(), vus, visites)
	case reflect.Int, reflect.Int32, reflect.Int64:
		vus[v.Int()] = true
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		vus[v.Uint()] = true
	case reflect.Float32, reflect.Float64:
		vus[v.Float()] = true
	case reflect.String:
		vus[v.String()] = true
	case reflect.Struct:
		for i := range v.NumField() {
			collecterEmpreintes(v.Field(i), vus, visites)
		}
	}
}

// TestApplyToReconstruitLaLunette : `Options.Scoped` est une fermeture, que la table de champs
// ci-dessus ne peut pas comparer. Une entree et une sortie de lunette sur un slot doivent s y lire.
func TestApplyToReconstruitLaLunette(t *testing.T) {
	in := FilmInputs{ZoomEvents: []grammar.ZoomEvent{
		{TimestampUS: 1_000_000, Slot: 5, Level: 1},
		{TimestampUS: 2_000_000, Slot: 5, Level: 0},
	}}
	var opt Options
	in.applyTo(&opt)
	if opt.Scoped == nil {
		t.Fatal("applyTo ne pose pas Options.Scoped")
	}
	if got := opt.Scoped(5, 1_500_000); got != 1 {
		t.Errorf("palier de lunette en pleine visee = %d, attendu 1", got)
	}
	if got := opt.Scoped(5, 2_500_000); got != 0 {
		t.Errorf("palier de lunette apres la sortie = %d, attendu 0", got)
	}
}
