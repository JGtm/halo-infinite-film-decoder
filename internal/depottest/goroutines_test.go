package depottest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fichiersAutorisesAGoroutines nomme les seuls fichiers de production qui portent une instruction
// `go`. Le decodeur a TROIS points de lancement, tous nommes ici, et tout autre est refuse : l'assistant
// de fils de la grammaire (`internal/grammar/fils.go`), la tache de fond du balayage de `replay` (une
// goroutine qui lit, sans contexte de film, ce qui ne depend que du film charge ; elle est toujours attendue
// par l'appelant) et les voies du balayage (files de lectures d'apres la marche, chacune sur un contexte
// derive, pendant que le fil principal deroule sa chaine : l'assistant de fils joint avant de rendre la
// main et ne le permettrait pas ; elles sont toujours jointes avant la sortie du balayage). Le nombre de
// goroutines qui calculent en meme temps est borne par `Fils`+3 (voir `OptionsVitesse.Fils`). Les deux
// autres entrees sont des outils qui l'entourent (le garde-memoire et le profilage de la commande de
// mesure) et ne decodent rien.
var fichiersAutorisesAGoroutines = map[string]bool{
	"internal/grammar/fils.go":  true,
	"replay/film_scan_fond.go":  true,
	"replay/film_scan_voies.go": true,
	"memguard/memguard.go":      true,
	"cmd/film-bench/profil.go":  true,
}

// TestUneSeuleSourceDeGoroutines (le nom est historique : il y a trois points de lancement nommes) :
// aucune instruction `go` dans le code de production du module
// (fichiers non `_test.go`, toutes contraintes de compilation confondues) hors des fichiers de
// [fichiersAutorisesAGoroutines] ; et la liste ne garde aucune entree sans instruction `go`.
func TestUneSeuleSourceDeGoroutines(t *testing.T) {
	racine := racineDuModule(t)
	vus := map[string]bool{}
	var fautifs []string
	fset := token.NewFileSet()
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
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, chemin, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(racine, chemin)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		ast.Inspect(f, func(n ast.Node) bool {
			if _, ok := n.(*ast.GoStmt); ok {
				vus[rel] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}
	for rel := range vus {
		if !fichiersAutorisesAGoroutines[rel] {
			fautifs = append(fautifs, rel)
		}
	}
	sort.Strings(fautifs)
	for _, rel := range fautifs {
		t.Errorf("%s : instruction `go` hors des fichiers autorises ; lancer des fils par executerEnParallele (internal/grammar/fils.go)", rel)
	}
	for rel := range fichiersAutorisesAGoroutines {
		if !vus[rel] {
			t.Errorf("%s : autorise mais sans instruction `go` ; retirer l'entree", rel)
		}
	}
}
