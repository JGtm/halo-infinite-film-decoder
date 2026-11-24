package replay

// sieges_et_occupants_test.go — LE SYNC ET LA CUISSON NOMMENT AVEC LE MEME REGISTRE.
//
//	SO-UNIQUE   la table des sieges du film et les occupants par entite ne se lisent, en production,
//	            que dans [lireSiegesEtOccupantsLecture] (la table) et [lireLesOccupants] (les equipes
//	            et le tableau des scores) ; [lireSiegesEtOccupantsLecture] a DEUX appelants, la lecture
//	            de l'etage de balayage de la cuisson ([filmScan.lireSieges]) et [lireSiegesEtOccupants],
//	            que le document interne du sync est seul a appeler ; [lireLesOccupants] en a deux, elle
//	            aussi : [lireSiegesEtOccupantsLecture] et la voie des sieges de la cuisson a plusieurs
//	            fils ([lancerLaVoieDesSieges]), qui lit le meme registre sur son contexte derive ;
//	SO-OPTIONS  le document interne pose les quatre lectures dans les options de l'assemblage — celles
//	            que la cuisson y pose par `FilmInputs.applyTo`.
//
// La parite sur de vrais films (le sync et la cuisson nomment les memes vies) est mesuree par
// `sync/killcollector/porteurs_au_sync_parite_research_test.go`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// lecturesDuNommage : les trois lectures du film que [lireSiegesEtOccupantsLecture] est seule a appeler.
var lecturesDuNommage = []string{"ScanFilmPlayerTable", "ScanPlayerTeams", "ScanRosterListings"}

// lecteurAutorise : la fonction qui appelle chacune de ces lectures, et la seule.
var lecteurAutorise = map[string]string{"ScanFilmPlayerTable": "lireSiegesEtOccupantsLecture",
	"ScanPlayerTeams": "lireLesOccupants", "ScanRosterListings": "lireSiegesEtOccupantsLecture"}

// appelsParFonction rend, pour chaque fonction de production du paquet, les noms simples qu'elle
// appelle.
func appelsParFonction(t *testing.T) map[string][]string {
	t.Helper()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	out := map[string][]string{}
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if appele, ok := nomAppele(call); ok {
						out[fn.Name.Name] = append(out[fn.Name.Name], appele)
					}
				}
				return true
			})
		}
	}
	return out
}

// TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants (SO-UNIQUE). Mutations vues rouges : rappeler
// `grammar.ScanPlayerTeams` dans la garde du drapeau du sync ; lire la table du film a part dans
// `lireSieges` ; ne plus appeler [lireSiegesEtOccupants] depuis `DocumentInterne`.
func TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants(t *testing.T) {
	var appelantsDeLaLecture, appelantsDuSync, appelantsDesOccupants []string
	for fn, appels := range appelsParFonction(t) {
		for _, lecture := range lecturesDuNommage {
			if fn != lecteurAutorise[lecture] && slices.Contains(appels, lecture) {
				t.Errorf("%s appelle %s hors de %s : le sync et la cuisson ne "+
					"liraient plus le meme registre", fn, lecture, lecteurAutorise[lecture])
			}
		}
		if slices.Contains(appels, "lireLesOccupants") {
			appelantsDesOccupants = append(appelantsDesOccupants, fn)
		}
		if slices.Contains(appels, "lireSiegesEtOccupantsLecture") {
			appelantsDeLaLecture = append(appelantsDeLaLecture, fn)
		}
		if slices.Contains(appels, "lireSiegesEtOccupants") {
			appelantsDuSync = append(appelantsDuSync, fn)
		}
	}
	slices.Sort(appelantsDeLaLecture)
	slices.Sort(appelantsDuSync)
	slices.Sort(appelantsDesOccupants)
	if want := []string{"lancerLaVoieDesSieges", "lireSiegesEtOccupantsLecture"}; !slices.Equal(appelantsDesOccupants, want) {
		t.Errorf("appelants de lireLesOccupants = %v, attendu %v (lecture complete et voie des sieges)",
			appelantsDesOccupants, want)
	}
	if want := []string{"lireSieges", "lireSiegesEtOccupants"}; !slices.Equal(appelantsDeLaLecture, want) {
		t.Errorf("appelants de lireSiegesEtOccupantsLecture = %v, attendu %v (cuisson et composition du sync)",
			appelantsDeLaLecture, want)
	}
	if want := []string{"DocumentInterne"}; !slices.Equal(appelantsDuSync, want) {
		t.Errorf("appelants de lireSiegesEtOccupants = %v, attendu %v (document interne du sync)",
			appelantsDuSync, want)
	}
}

// TestLeDocumentInternePoseLesSiegesEtLesOccupants (SO-OPTIONS). Mutation vue rouge : retirer un des
// quatre champs de `optionsDuRegistre`.
func TestLeDocumentInternePoseLesSiegesEtLesOccupants(t *testing.T) {
	so := siegesEtOccupants{
		table:   grammar.FilmPlayerTable{Occupied: 3},
		equipes: map[int]int{8: 1},
		rapport: grammar.TeamScanReport{Read: 4},
		entites: grammar.PlayerEntityScan{Scanned: true, Entities: []grammar.PlayerEntity{{Slot: 1550, Index: 8}}},
	}
	opt := EntreePorteursAuSync{}.optionsDuRegistre(nil, so)
	if opt.FilmTable.Occupied != 3 || opt.PlayerTeams[8] != 1 || opt.TeamScan.Read != 4 ||
		!opt.PlayerEntities.Scanned || len(opt.PlayerEntities.Entities) != 1 {
		t.Fatalf("options du document interne : table %+v, equipes %v, rapport %+v, entites %+v — attendu "+
			"les quatre lectures de lireSiegesEtOccupants", opt.FilmTable, opt.PlayerTeams, opt.TeamScan,
			opt.PlayerEntities)
	}
}
