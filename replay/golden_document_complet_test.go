package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// renderDocumentComplet imprime CHAQUE champ publie du document, et chaque champ de chaque bloc
// de couverture, avec son compte et l empreinte de son JSON canonique.
//
// LE RENDU EST DERIVE DE LA STRUCTURE PAR REFLEXION, et c est le point : un calque ou un compteur
// ajoute au document apparait dans le golden sans que personne ait a penser a l y inscrire, et un
// calque qui disparait (une ligne d assemblage retiree) change sa ligne. Les rendus ecrits a la
// main (`renderAssembly`) restent la lecture humaine ; celui-ci est le filet.
func renderDocumentComplet(p func(string, ...any), doc ReplayDocument) {
	p("## DOCUMENT COMPLET — chaque champ publie, son compte et l empreinte de son contenu")
	renderChampsPublies(p, "", reflect.ValueOf(doc))
	p("")
}

// renderChampsPublies parcourt les champs d une structure (et, en profondeur, ceux de ses
// sous-structures) dans l ordre de declaration, sous le nom JSON qu ils portent dans le document.
func renderChampsPublies(p func(string, ...any), prefixe string, v reflect.Value) {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		nom, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if nom == "-" {
			continue
		}
		fv := v.Field(i)
		if f.Anonymous && nom == "" && fv.Kind() == reflect.Struct {
			renderChampsPublies(p, prefixe, fv)
			continue
		}
		if nom == "" {
			nom = f.Name
		}
		chemin := prefixe + nom
		p("%s : %s", chemin, resumeChampPublie(fv))
		interieur := fv
		if interieur.Kind() == reflect.Pointer && !interieur.IsNil() {
			interieur = interieur.Elem()
		}
		if interieur.Kind() == reflect.Struct {
			renderChampsPublies(p, chemin+".", interieur)
		}
	}
}

// resumeChampPublie : « absent » pour un pointeur nul, sinon le compte (tranche, table) et
// l empreinte du JSON du champ. Le JSON d une table est trie par cle : l empreinte est stable.
func resumeChampPublie(fv reflect.Value) string {
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return "absent"
	}
	octets, err := json.Marshal(fv.Interface())
	if err != nil {
		return "illisible : " + err.Error()
	}
	somme := sha256.Sum256(octets)
	empreinte := hex.EncodeToString(somme[:6])
	switch fv.Kind() {
	case reflect.Slice, reflect.Map:
		return "n=" + strconv.Itoa(fv.Len()) + " · " + empreinte
	case reflect.Struct, reflect.Pointer:
		return empreinte
	default:
		return string(octets) + " · " + empreinte
	}
}

// TestRenduCompletNommeChaqueChampPublie : le rendu complet porte une ligne pour chaque clef
// du document serialise et pour chaque clef de `coverage`. Un champ qui n y figurerait pas
// pourrait disparaitre sans bruit.
func TestRenduCompletNommeChaqueChampPublie(t *testing.T) {
	doc := buildGolden(t)
	rendu := renduCompletDe(doc)

	octets, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	var brut map[string]json.RawMessage
	if err := json.Unmarshal(octets, &brut); err != nil {
		t.Fatalf("relecture : %v", err)
	}
	var couverture map[string]json.RawMessage
	if err := json.Unmarshal(brut["coverage"], &couverture); err != nil {
		t.Fatalf("relecture de coverage : %v", err)
	}
	for cle := range brut {
		if !strings.Contains(rendu, "\n"+cle+" : ") {
			t.Errorf("la cle publiee %q n a pas de ligne dans le rendu complet", cle)
		}
	}
	for cle := range couverture {
		if !strings.Contains(rendu, "\ncoverage."+cle+" : ") {
			t.Errorf("la cle publiee coverage.%s n a pas de ligne dans le rendu complet", cle)
		}
	}
}

// TestRenduCompletVoitUnCalqueRetire : retirer un calque du document change le rendu complet.
func TestRenduCompletVoitUnCalqueRetire(t *testing.T) {
	doc := ReplayDocument{Chatter: []Chatter{{}}, Gestures: []Gesture{{}}}
	avant := renduCompletDe(doc)
	doc.Chatter = nil
	if avant == renduCompletDe(doc) {
		t.Error("le rendu complet ne voit pas un calque retire")
	}
}

func renduCompletDe(doc ReplayDocument) string {
	var b strings.Builder
	renderDocumentComplet(func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }, doc)
	return b.String()
}

// exigerDocumentsIdentiques compare deux documents sur leur JSON ENTIER, tel qu il serait publie.
//
// LE RENDU TEXTE (`renderAssembly`) EST UN RESUME : il ne dit rien d un calque qu il n imprime
// pas. Une egalite « depuis les faits = depuis le film » ne vaut que sur ce qui est publie.
func exigerDocumentsIdentiques(t *testing.T, message string, a, b ReplayDocument) {
	t.Helper()
	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("serialisation du premier document : %v", err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("serialisation du second document : %v", err)
	}
	if string(ja) == string(jb) {
		return
	}
	t.Errorf("%s (%d octets contre %d).\nchamps qui different :\n%s", message, len(ja), len(jb),
		champsDifferents(a, b))
}

// champsDifferents liste les chemins publies dont la ligne du rendu complet n est pas la meme.
func champsDifferents(a, b ReplayDocument) string {
	la, lb := strings.Split(renduCompletDe(a), "\n"), strings.Split(renduCompletDe(b), "\n")
	var out []string
	for i := range la {
		if i < len(lb) && la[i] != lb[i] {
			out = append(out, "  - "+la[i]+"\n    contre "+lb[i])
		}
	}
	if len(out) == 0 {
		return "  (aucune ligne du rendu complet ne differe)"
	}
	return strings.Join(out, "\n")
}
