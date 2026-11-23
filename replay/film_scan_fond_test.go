package replay

// film_scan_fond_test.go — LA TACHE DE FOND DU BALAYAGE NE CHANGE RIEN A LA SORTIE.
//
//	VOIE-IDENTIQUE  les etapes observees (noms, ordre, empreintes), l erreur du balayage, les replis du
//	                contexte et les entrees lues sont les memes a 1, 2, 4 et 8 fils ;
//	VOIE-UN-FIL     a un fil, aucune tache n est creee ; a plus d un fil, aucune goroutine ne survit au
//	                balayage, sortie anticipee comprise ;
//	VOIE-PANIQUE    une panique de la tache est rejouee, une fois, sur le fil qui attend la valeur ;
//	VOIE-SOURCE     les lectures de la tache sont celles des consommateurs, et l instruction `go` de
//	                `replay` n est que dans le fichier de fond.
//
// Les bobines sont celles du decodeur : un prefixe contigu de film (en-tete, table des joueurs, listes de
// la session, temps forts). Le balayage des positions les refuse (aucune image-cle de bipede utilisable),
// donc `scanFilmInputs` y rend l erreur de ce refus apres les premieres etapes ; les deux phases qui
// consomment la table et les scripts se jouent a part, dans le meme ordre qu en production.

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/internal/digest"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// bobinesDuFond : les deux prefixes de film du decodeur que ces tests lisent.
var bobinesDuFond = []string{
	"../internal/facts/killsource/testdata/minibobine_000d5950",
	"../internal/facts/killsource/testdata/minibobine_e5adf7b2",
}

// rosterDAppointDuFond complete le roster que les morts donnent : des xuids absents du film.
var rosterDAppointDuFond = []uint64{42, 7}

// sortieDuFond est ce que le balayage d une bobine a produit : tout ce qui doit etre egal d un nombre
// de fils a l autre.
type sortieDuFond struct {
	etapes []string // `nom\tcompte\tempreinte`, dans l ordre observe
	err    string
	replis grammar.ComptesDesReplis
	// entrees : les valeurs que les consommateurs de la tache posent, par nom.
	entrees []string
}

// optionsDuFond : les options d un balayage de test, l observateur rangeant chaque etape.
func optionsDuFond(t *testing.T, sortie *sortieDuFond) Options {
	t.Helper()
	entry := goldenEntryPourTest(t)
	return Options{
		MapQuant: &entry, RosterXUIDs: rosterDAppointDuFond,
		Observe: func(step string, v any) {
			compte, somme := digest.Of(v)
			sortie.etapes = append(sortie.etapes, fmt.Sprintf("%s\t%d\t%s", step, compte, somme))
		},
	}
}

// balayerLaBobine rejoue `scanFilmInputs` sur une bobine, au nombre de fils demande, avec un contexte neuf.
func balayerLaBobine(t *testing.T, dossier string, fils int) sortieDuFond {
	t.Helper()
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dossier, err)
	}
	var sortie sortieDuFond
	opt := optionsDuFond(t, &sortie)
	fc := grammar.NewFilmContextForMap(film, opt.MapQuant, decoupageForce(opt))
	poserLesFils(fc, OptionsVitesse{Fils: fils})
	_, err = scanFilmInputs(context.Background(), "bobine", film, fc, opt)
	if err != nil {
		sortie.err = err.Error()
	}
	sortie.replis = fc.ComptesDesReplis()
	return sortie
}

// balayerLesPhasesDeFin joue, dans l ordre de `scanFilmInputs`, le depart de la tache puis les trois
// phases qui consomment ses valeurs (le pont, les etats de mouvement avec leurs scripts, le pont avec
// la table des joueurs), sans la marche ni les positions que la bobine refuse.
func balayerLesPhasesDeFin(t *testing.T, dossier string, fils int) sortieDuFond {
	t.Helper()
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dossier, err)
	}
	var sortie sortieDuFond
	opt := optionsDuFond(t, &sortie)
	fc := grammar.NewFilmContextForMap(film, opt.MapQuant, decoupageForce(opt))
	poserLesFils(fc, OptionsVitesse{Fils: fils})
	s := &filmScan{ctx: context.Background(), matchID: "bobine", film: film, fc: fc, opt: opt, world: opt.MapQuant.Range()}
	s.scan = grammar.DefaultScanFilmOptions()
	s.scan.WorldRange = &s.world
	s.scan.CaptureDirs = true
	s.scan.Layout = fc.ImposedLayout()
	s.fond = s.demarrerLeFond()
	defer s.fond.joindre()
	s.publierPontDIdentite(s.lirePontDIdentite())
	s.balayerEtatsDeMouvement()
	s.balayerPont()
	sortie.replis = fc.ComptesDesReplis()
	for _, e := range []struct {
		nom string
		val any
	}{
		{"morts", s.in.Deaths}, {"medailles", s.in.Medals}, {"indices", s.in.PlayerIndices},
		{"origine", s.in.FilmClockOriginUS}, {"scripts", s.in.Scripts}, {"table", s.in.FilmTable},
		{"listages", s.in.RosterListings}, {"equipes", s.in.PlayerTeams}, {"entites", s.in.PlayerEntities},
		{"fil", s.in.DeathsFeed},
	} {
		compte, somme := digest.Of(e.val)
		sortie.entrees = append(sortie.entrees, fmt.Sprintf("%s\t%d\t%s", e.nom, compte, somme))
	}
	return sortie
}

// avecHuitProcesseurs regle GOMAXPROCS a 8 : `FilsEffectifs` plafonne le nombre de fils dessus.
func avecHuitProcesseurs(t *testing.T) {
	t.Helper()
	ancien := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(ancien) })
}

// TestVoieFondIdentiqueATousLesFils (VOIE-IDENTIQUE).
func TestVoieFondIdentiqueATousLesFils(t *testing.T) {
	avecHuitProcesseurs(t)
	for _, dossier := range bobinesDuFond {
		for _, phase := range []struct {
			nom string
			run func(*testing.T, string, int) sortieDuFond
		}{{"scanFilmInputs", balayerLaBobine}, {"phases de fin", balayerLesPhasesDeFin}} {
			ref := phase.run(t, dossier, 1)
			if len(ref.etapes) == 0 {
				t.Fatalf("%s %s : aucune etape observee a un fil, le test ne prouverait rien", dossier, phase.nom)
			}
			for _, fils := range []int{2, 4, 8} {
				got := phase.run(t, dossier, fils)
				if !slices.Equal(got.etapes, ref.etapes) || got.err != ref.err || got.replis != ref.replis ||
					!slices.Equal(got.entrees, ref.entrees) {
					t.Errorf("%s %s a %d fils : sortie differente de celle d un fil\n  etapes un fil %v\n  etapes %d fils %v\n"+
						"  erreur %q / %q\n  replis %+v / %+v\n  entrees %v / %v", dossier, phase.nom, fils,
						ref.etapes, fils, got.etapes, ref.err, got.err, ref.replis, got.replis, ref.entrees, got.entrees)
				}
			}
		}
	}
}

// TestVoieFondLitCeQueLeFilmPorte : la comparaison ci-dessus porte sur de vraies valeurs — la table des
// joueurs, les temps forts et les scripts de la premiere bobine ne sont pas vides.
func TestVoieFondLitCeQueLeFilmPorte(t *testing.T) {
	avecHuitProcesseurs(t)
	sortie := balayerLesPhasesDeFin(t, bobinesDuFond[0], 4)
	vides := map[string]bool{}
	for _, l := range sortie.entrees {
		champs := strings.Split(l, "\t")
		vides[champs[0]] = champs[1] == "0"
	}
	for _, nom := range []string{"morts", "table", "listages", "origine"} {
		if vides[nom] {
			t.Errorf("entree %q vide : la bobine ne porte pas ce que le test compare", nom)
		}
	}
}

// goroutinesRetombees attend que le nombre de goroutines revienne a `avant` (la goroutine qui vient de
// terminer sort un instant apres avoir ferme ses canaux).
func goroutinesRetombees(avant int) bool {
	fin := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > avant {
		if time.Now().After(fin) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
	return true
}

// TestVoieAUnFilAucuneGoroutine (VOIE-UN-FIL) : a un fil, ou sans reglage, aucune tache n est creee et
// les accesseurs rendent « rien d avance » ; a plus d un fil, rien ne survit au balayage.
func TestVoieAUnFilAucuneGoroutine(t *testing.T) {
	avecHuitProcesseurs(t)
	film, err := source.LoadDir(bobinesDuFond[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fils := range []int{0, 1} {
		fc := grammar.NewFilmContext(film)
		poserLesFils(fc, OptionsVitesse{Fils: fils})
		s := &filmScan{film: film, fc: fc}
		avant := runtime.NumGoroutine()
		if f := s.demarrerLeFond(); f != nil {
			t.Fatalf("Fils=%d : une tache de fond est creee", fils)
		}
		if got := runtime.NumGoroutine(); got != avant {
			t.Errorf("Fils=%d : %d goroutines avant, %d apres le depart", fils, avant, got)
		}
		var rien *fondDuFilm
		if rien.lesFaits() != nil || rien.laTable() != nil {
			t.Errorf("Fils=%d : un fond nil rend une valeur d avance", fils)
		}
		if scripts, lus := rien.lesScripts(); lus || scripts != nil {
			t.Errorf("Fils=%d : un fond nil rend des scripts d avance", fils)
		}
		rien.joindre()
	}
	// A plusieurs fils : le balayage qui sort tot (erreur des positions) attend la tache.
	avant := runtime.NumGoroutine()
	balayerLaBobine(t, bobinesDuFond[0], 4)
	balayerLesPhasesDeFin(t, bobinesDuFond[0], 4)
	if !goroutinesRetombees(avant) {
		t.Errorf("%d goroutines avant, %d apres le balayage a 4 fils", avant, runtime.NumGoroutine())
	}
	// Les voies du balayage (film_scan_voies.go) : meme exigence.
	voiesAUnFilAucuneGoroutine(t)
}

// TestVoiePaniqueRelayee (VOIE-PANIQUE) : la panique du roster, levee dans la tache, est rejouee telle
// quelle sur le fil qui attend les faits ; elle n est rejouee qu une fois, et la jointure ne la rejoue pas.
func TestVoiePaniqueRelayee(t *testing.T) {
	film, err := source.LoadDir(bobinesDuFond[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	nouveau := func() *fondDuFilm {
		f := &fondDuFilm{
			faits:   attendue[grammar.FaitsDuFilmSeul]{pret: make(chan struct{})},
			scripts: attendue[[]string]{pret: make(chan struct{})},
			table:   attendue[tableLueDAvance]{pret: make(chan struct{})},
		}
		go f.lireLeFond(film, func([]types.Death) []uint64 { panic("panique du roster") })
		return f
	}
	rattraper := func(fn func()) (p any) {
		defer func() { p = recover() }()
		fn()
		return nil
	}

	f := nouveau()
	if p := rattraper(func() { f.lesFaits() }); p != "panique du roster" {
		t.Fatalf("panique rejouee = %v, attendu la panique de la tache", p)
	}
	if p := rattraper(func() { f.lesScripts(); f.laTable(); f.joindre() }); p != nil {
		t.Errorf("la panique est rejouee une seconde fois : %v", p)
	}

	// Personne n attend de valeur : la jointure rejoue la panique.
	g := nouveau()
	if p := rattraper(g.joindre); p != "panique du roster" {
		t.Errorf("jointure : panique rejouee = %v, attendu la panique de la tache", p)
	}
	if p := rattraper(g.joindre); p != nil {
		t.Errorf("seconde jointure : panique %v", p)
	}
}

// TestLesLecturesDuFondSontCellesDesConsommateurs (VOIE-SOURCE). Mutations vues rouges : lire la table
// dans le fond avec `ScanFilmPlayerTable` ; ajouter une seconde instruction `go` dans `replay`.
func TestLesLecturesDuFondSontCellesDesConsommateurs(t *testing.T) {
	appelants := map[string][]string{}
	for fn, appels := range appelsParFonction(t) {
		for _, lecture := range []string{"ScanFilmPlayerTableAuxListages", "ScanScriptsDuFilm", "LireLesFaitsDuFilmSeul"} {
			if slices.Contains(appels, lecture) {
				appelants[lecture] = append(appelants[lecture], fn)
			}
		}
	}
	for lecture, want := range map[string][]string{
		"ScanFilmPlayerTableAuxListages": {"lireLeFond", "lireSiegesEtOccupantsLecture"},
		"ScanScriptsDuFilm":              {"balayerEtatsDeMouvement", "lireLeFond"},
		"LireLesFaitsDuFilmSeul":         {"lireLeFond"},
	} {
		got := appelants[lecture]
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("appelants de %s = %v, attendu %v", lecture, got, want)
		}
	}

	// LES `go` DE REPLAY : un dans le fichier de fond, un dans celui des voies.
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var gos []string
	fset := token.NewFileSet()
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if _, ok := n.(*ast.GoStmt); ok {
				gos = append(gos, nom)
			}
			return true
		})
	}
	if want := []string{"film_scan_fond.go", "film_scan_voies.go"}; !slices.Equal(gos, want) {
		t.Errorf("instructions `go` de replay dans %v, attendu %v", gos, want)
	}
	if slices.Contains(fichiersDuBalayage, "film_scan_fond.go") {
		t.Error("film_scan_fond.go est dans fichiersDuBalayage : le garde des etapes compterait ses lectures deux fois")
	}
}
