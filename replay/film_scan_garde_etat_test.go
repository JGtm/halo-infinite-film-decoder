package replay

// film_scan_garde_etat_test.go — LES OUTILS DES GARDES DE CARACTERISATION DE L ETAT QUE TOUCHENT LES
// BALAYAGES D APRES LA MARCHE (item J4 de la conception du parallelisme).
//
// Trois outils, sans rien ajouter a la production :
//
//   - [instantane] : une photo du contexte du film par REFLEXION. Les champs non exportes se lisent par
//     leurs accesseurs de genre (nil, longueur, entier, booleen, adresse), jamais par `Interface` ; un
//     pointeur ou une map reaffectee change d adresse, donc se voit. Un memo ECRIT pendant un balayage
//     est un chemin dont la photo change ;
//   - [poisonnerLObservation] : donne des compteurs absurdes a l observation PARTAGEE du contexte, pour
//     prouver par la sortie qu aucun consommateur ne les lit ;
//   - [installerLesSentinelles] : pose sur chaque crochet de l observation partagee un crochet qui compte
//     ses appels, pour voir si un balayage en retire ou en remplace un.

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// profondeurMaxDeLInstantane borne la descente dans les pointeurs : le contexte est un arbre de peu de
// niveaux, et la borne protege d un cycle que `vus` n aurait pas vu.
const profondeurMaxDeLInstantane = 6

// instantane photographie le contexte : chemin -> valeur lisible. Le film (immuable) et les verrous ne
// sont pas photographies.
func instantane(fc *grammar.FilmContext) map[string]string {
	out := map[string]string{}
	photographier("", reflect.ValueOf(fc).Elem(), 0, map[uintptr]bool{}, out)
	return out
}

// photographier descend dans `v` et range une ligne par feuille.
func photographier(chemin string, v reflect.Value, prof int, vus map[uintptr]bool, out map[string]string) {
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		if t.PkgPath() == "sync" || t.PkgPath() == "sync/atomic" {
			return
		}
		for i := range v.NumField() {
			nom := t.Field(i).Name
			if chemin == "" && nom == "film" {
				continue
			}
			photographier(joindre(chemin, nom), v.Field(i), prof, vus, out)
		}
	case reflect.Pointer:
		if v.IsNil() {
			out[chemin] = "nil"
			return
		}
		out[chemin] = fmt.Sprintf("@%x", v.Pointer())
		if v.Elem().Kind() == reflect.Struct && prof < profondeurMaxDeLInstantane && !vus[v.Pointer()] {
			vus[v.Pointer()] = true
			photographier(chemin, v.Elem(), prof+1, vus, out)
		}
	case reflect.Map:
		if v.IsNil() {
			out[chemin] = "nil"
			return
		}
		out[chemin] = fmt.Sprintf("map@%x len=%d", v.Pointer(), v.Len())
	case reflect.Slice:
		if v.IsNil() {
			out[chemin] = "nil"
			return
		}
		out[chemin] = fmt.Sprintf("len=%d", v.Len())
	case reflect.Func, reflect.Interface, reflect.Chan:
		if v.IsNil() {
			out[chemin] = "nil"
		} else {
			out[chemin] = "pose"
		}
	case reflect.Bool:
		out[chemin] = fmt.Sprint(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out[chemin] = fmt.Sprint(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out[chemin] = fmt.Sprint(v.Uint())
	case reflect.String:
		out[chemin] = v.String()
	case reflect.Array:
		for i := range v.Len() {
			photographier(fmt.Sprintf("%s[%d]", chemin, i), v.Index(i), prof, vus, out)
		}
	}
}

// joindre compose un chemin pointe.
func joindre(chemin, nom string) string {
	if chemin == "" {
		return nom
	}
	return chemin + "." + nom
}

// differences rend les chemins dont la valeur differe entre deux photos, tries.
func differences(avant, apres map[string]string) []string {
	var out []string
	for k, v := range apres {
		if avant[k] != v {
			out = append(out, k)
		}
	}
	for k := range avant {
		if _, ok := apres[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// classeDeChemin range un chemin de photo dans sa famille : l observation, le profil de balayage, les
// rapports (replis, diagnostics) ou un memo de contexte.
func classeDeChemin(chemin string) string {
	tete, _, _ := strings.Cut(chemin, ".")
	switch tete {
	case "obs":
		return "obs"
	case "bal":
		return "bal"
	case "replis", "diag":
		return "rapport"
	}
	return "memo"
}

// separer repartit des chemins par famille.
func separer(chemins []string) map[string][]string {
	out := map[string][]string{}
	for _, c := range chemins {
		k := classeDeChemin(c)
		out[k] = append(out[k], c)
	}
	return out
}

// poisonnerLObservation donne une valeur absurde a chaque compteur de l observation : les entiers a une
// grande valeur, les histogrammes a une entree de plus. Les crochets (fonctions) ne sont pas touches.
func poisonnerLObservation(o *grammar.Observation) {
	v := reflect.ValueOf(o).Elem()
	for i := range v.NumField() {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Int:
			f.SetInt(7_000_001)
		case reflect.Map:
			if f.IsNil() {
				f.Set(reflect.MakeMap(f.Type()))
			}
			poisonnerUneMap(f)
		}
	}
}

// poisonnerUneMap ajoute une entree absurde aux histogrammes entiers que l observation porte.
func poisonnerUneMap(m reflect.Value) {
	t := m.Type()
	switch {
	case t.Elem().Kind() == reflect.Int && t.Key().Kind() == reflect.Int:
		m.SetMapIndex(reflect.ValueOf(999), reflect.ValueOf(7_000_001))
	case t.Elem().Kind() == reflect.Int && t.Key().Kind() == reflect.Uint32:
		m.SetMapIndex(reflect.ValueOf(uint32(999)), reflect.ValueOf(7_000_001))
	case t.Key().Kind() == reflect.String && t.Elem().Kind() == reflect.Map:
		inner := reflect.MakeMap(t.Elem())
		inner.SetMapIndex(reflect.ValueOf(999), reflect.ValueOf(7_000_001))
		m.SetMapIndex(reflect.ValueOf("poison"), inner)
	}
}

// installerLesSentinelles pose sur chaque crochet de l observation un crochet qui ne fait que compter
// ses appels dans `appels` (par nom de champ). Rend les noms des crochets poses.
func installerLesSentinelles(o *grammar.Observation, appels map[string]int) []string {
	v := reflect.ValueOf(o).Elem()
	var noms []string
	for i := range v.NumField() {
		f, ft := v.Field(i), v.Type().Field(i)
		if f.Kind() != reflect.Func || !f.CanSet() {
			continue
		}
		nom := ft.Name
		f.Set(reflect.MakeFunc(f.Type(), func([]reflect.Value) []reflect.Value {
			appels[nom]++
			sortie := make([]reflect.Value, f.Type().NumOut())
			for j := range sortie {
				sortie[j] = reflect.Zero(f.Type().Out(j))
			}
			return sortie
		}))
		noms = append(noms, nom)
	}
	return noms
}
