package chattername

import (
	"regexp"
	"strings"
	"testing"
)

// TestHashVecteursConnus : la fonction de hachage rend les valeurs publiees de MurmurHash3 x86 32 bits, graine 0.
// MUTATIONS : une rotation, une constante ou le melange final change -> ROUGE.
func TestHashVecteursConnus(t *testing.T) {
	for nom, voulu := range map[string]uint32{"": 0, "test": 0xba6bd213, "Hello, world!": 0xc0363e43, "default": 0x42c9679f} {
		if got := Hash(nom); got != voulu {
			t.Errorf("Hash(%q) = %08x, voulu %08x", nom, got, voulu)
		}
	}
}

// TestChaqueNomEstProuveParSonHachage : pour CHAQUE ligne de la table, le hachage du nom egale le hachage enregistre
// pour l entree des tables du jeu (colonne 2). C est la preuve d un nom ; sans elle il n entre pas dans la table.
// MUTATIONS : un nom modifie d une lettre, un hachage recopie d une autre ligne -> ROUGE.
func TestChaqueNomEstProuveParSonHachage(t *testing.T) {
	if err := verifier(); err != nil {
		t.Fatal(err)
	}
	if Len() < 250 {
		t.Fatalf("table tronquee : %d noms", Len())
	}
}

// TestLesNomsSontDesMotsEnMinuscules : un nom est de minuscules et de chiffres, sans separateur (la forme de ceux dont la
// preuve est faite), deux indices ne portent pas le meme nom, et les indices restent dans les deux plages nommees.
func TestLesNomsSontDesMotsEnMinuscules(t *testing.T) {
	forme := regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	vus := map[string]int{}
	for i, e := range table {
		if !forme.MatchString(e.nom) {
			t.Errorf("indice %d : nom %q hors forme", i, e.nom)
		}
		if j, deja := vus[e.nom]; deja {
			t.Errorf("indices %d et %d : meme nom %q", j, i, e.nom)
		}
		vus[e.nom] = i
		if i >= 915 && (i < 2732 || i >= 2957) {
			t.Errorf("indice %d : hors des deux plages nommees", i)
		}
	}
}

// TestLookupNeNommeQueLesBuildsMesures : un build dont la disposition n est pas mesuree, ou absent, ne recoit aucun nom ; un
// indice sans nom prouve non plus. Les noms observes sur les films des builds 1.12 et 1.13 se rendent.
// MUTATIONS : le garde de build retire, un indice voisin rendu -> ROUGE.
func TestLookupNeNommeQueLesBuildsMesures(t *testing.T) {
	cas := []struct {
		build string
		ligne int
		nom   string
		ok    bool
	}{
		{build113, 850, "shieldbreak", true},
		{build112, 2788, "shieldsdepleted", true},
		{build113, 2926, "seeingenemynearby", true},
		{build113, 2897, "lamentsquaddeath", true},
		{"HI_1_11_0", 2926, "", false}, // l autre disposition : l indice y vaut 205 de plus
		{"HI_1_14_0", 2926, "", false}, // build non mesure
		{"", 2926, "", false},          // film sans build lu
		{build113, 2822, "", false},    // entree dont le nom n est pas prouve
		{build113, 3131, "", false},    // hors des plages nommees
	}
	for _, c := range cas {
		nom, ok := Lookup(c.build, c.ligne)
		if nom != c.nom || ok != c.ok {
			t.Errorf("Lookup(%q, %d) = %q, %v ; voulu %q, %v", c.build, c.ligne, nom, ok, c.nom, c.ok)
		}
	}
}

// TestLeFichierNePorteAucunTexteDuJeu : les lignes de donnees sont un tableau a trois colonnes, sans phrase.
func TestLeFichierNePorteAucunTexteDuJeu(t *testing.T) {
	for _, l := range strings.Split(brut, "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if strings.ContainsAny(l, " .!?'\"") {
			t.Errorf("ligne avec ponctuation ou espace : %q", l)
		}
	}
}
