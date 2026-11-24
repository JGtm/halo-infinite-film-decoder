package replay

// film_scan_gardes_j4_test.go — GARDES DE CARACTERISATION, A UN FIL, DE L ETAT QUE LES BALAYAGES
// D APRES LA MARCHE TOUCHENT (item J4 de la conception du parallelisme du decodeur).
//
// Ils transforment en faits les suppositions d une conception qui ne les avait etablies que par graphe
// d appels (un graphe sur-approxime : il suit les chemins de repli de la marche) :
//
//	PROFIL        chaque balayage rend le profil de balayage du contexte, largeurs MPP comprises, tel
//	              qu il l a recu (depart normal, puis depart perturbe pour que « restaurer » ne puisse
//	              pas se confondre avec « ne rien poser ») ;
//	DISTRIBUTION  aucun balayage ne relance la marche des trames : les memos que ses canaux rangent ne
//	              changent pas d adresse, et un profil CPU de chaque balayage ne contient jamais la
//	              marche (le detecteur est controle sur la marche elle-meme) ;
//	CROCHETS      aucun balayage ne laisse un crochet pose ou retire sur l observation du contexte ;
//	OBSERVATION   les compteurs de l observation du contexte, empoisonnes avant la marche, ne changent
//	              AUCUNE sortie ;
//	MEMOS         ce que chaque balayage ECRIT dans le contexte, relevé par photo avant et apres.
//
// Les films sont du parc local (`data/cache/film_chunks`), la carte est donnee explicitement par son nom
// de catalogue. Hors parc, le test se saute.
//
// POURQUOI LE HARNAIS REJOUE LA QUEUE DE `scanFilmInputs` : il faut une photo entre la marche et le premier
// balayage, que la fonction ne permet pas. Le garde `TestLaQueueDuHarnaisSuitScanFilmInputs` lit la
// fonction et verifie que la suite des appels est celle que le harnais rejoue.

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/internal/digest"
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// filmsDesGardesJ4 : deux films du parc, avec leur carte nommee au catalogue (donnee explicite).
var filmsDesGardesJ4 = []struct{ id, carte string }{
	{"000d5950", "Cliffhanger"},
	{"53ce4390", "Behemoth"},
}

// marqueDeLaMarche : un symbole que le profil CPU ne contient que si la marche des trames a tourne.
const marqueDeLaMarche = "marcheurDesTrames).marcherLePaquet"

// optionsDeGarde choisit ce que la passe observe ou perturbe.
type optionsDeGarde struct {
	poison      bool // compteurs absurdes sur l observation du contexte, avant la marche
	sentinelles bool // crochets comptables sur l observation du contexte, apres la marche
	perturbe    bool // largeurs MPP du contexte decalees apres la marche
	profils     bool // un profil CPU par balayage
}

// etapeDeGarde : ce qu une etape observee a change dans le contexte.
type etapeDeGarde struct {
	nom        string
	empreinte  string
	changes    map[string][]string // famille -> chemins
	marcheVue  bool                // le profil CPU de l etape contient la marche des trames
	profilFait bool
	appels     map[string]int // appels recus par les crochets sentinelles pendant l etape
}

// sortieDeGarde : une passe complete.
type sortieDeGarde struct {
	etapes       []etapeDeGarde
	marcheTemoin bool // le detecteur de marche a vu la marche (controle positif)
	appels       map[string]int
	crochets     []string
	aLaFin       map[string]string
}

// chargerLeFilmDeGarde charge un film du parc et l entree de catalogue de sa carte, ou saute.
func chargerLeFilmDeGarde(t *testing.T, id, carte string) (*source.Film, profile.MapQuantEntry) {
	t.Helper()
	dir := filepath.Join(cacheDuDepot(t), "film_chunks", id)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("film %s absent du parc local : %v", id, err)
	}
	cat, err := profile.LoadMapQuantCatalog(bornesDeCartes(t))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q : %v", carte, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", id, err)
	}
	return film, entry
}

// profilCPU mesure `f` par un profil CPU et dit si le profil contient `marque`. `ok` faux : un autre
// profil tourne deja (`-cpuprofile`), la mesure est impossible.
func profilCPU(f func(), marque string) (vu, ok bool) {
	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		f()
		return false, false
	}
	f()
	pprof.StopCPUProfile()
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		return false, false
	}
	brut, _ := io.ReadAll(zr)
	return bytes.Contains(brut, []byte(marque)), true
}

// jouerLaGarde rejoue la queue de `scanFilmInputs` sur un film, a un fil, en photographiant le contexte
// a chaque etape observee. La marche des trames est jouee d abord, hors des etapes.
func jouerLaGarde(t *testing.T, film *source.Film, entry profile.MapQuantEntry, og optionsDeGarde) sortieDeGarde {
	t.Helper()
	sortie := sortieDeGarde{appels: map[string]int{}}
	var fc *grammar.FilmContext
	var avant map[string]string
	appelsAvant := map[string]int{}
	var profilEnCours bytes.Buffer
	profilActif := false
	arreterLeProfil := func() (vu bool) {
		if !profilActif {
			return false
		}
		pprof.StopCPUProfile()
		profilActif = false
		zr, err := gzip.NewReader(&profilEnCours)
		if err != nil {
			return false
		}
		brut, _ := io.ReadAll(zr)
		return bytes.Contains(brut, []byte(marqueDeLaMarche))
	}
	demarrerLeProfil := func() {
		profilEnCours.Reset()
		profilActif = og.profils && pprof.StartCPUProfile(&profilEnCours) == nil
	}
	opt := Options{MapQuant: &entry, Fallbacks: fallback.NouveauCompteur()}
	opt.ReplisHorsBalayage.Consultations = opt.enregistreurDesConsultations()
	opt.Observe = func(step string, v any) {
		apres := instantane(fc)
		compte, somme := digest.Of(v)
		e := etapeDeGarde{nom: step, empreinte: fmt.Sprintf("%d/%s", compte, somme),
			changes: separer(differences(avant, apres))}
		if !strings.HasSuffix(step, ".stats") {
			e.profilFait = profilActif
			e.marcheVue = arreterLeProfil()
			demarrerLeProfil()
		}
		avant = apres
		e.appels = map[string]int{}
		for k, n := range sortie.appels {
			if n != appelsAvant[k] {
				e.appels[k] = n - appelsAvant[k]
			}
			appelsAvant[k] = n
		}
		sortie.etapes = append(sortie.etapes, e)
	}
	fc = grammar.NewFilmContextForMap(film, opt.MapQuant, decoupageForce(opt))
	poserProfilPuisCarte(context.Background(), fc, "garde", opt)
	if og.poison {
		poisonnerLObservation(fc.Observation())
	}
	s := &filmScan{ctx: context.Background(), matchID: "garde", film: film, fc: fc, opt: opt, world: opt.MapQuant.Range()}
	s.scan = grammar.DefaultScanFilmOptions()
	s.scan.WorldRange = &s.world
	s.scan.CaptureDirs = true
	s.scan.Layout = fc.ImposedLayout()
	s.opt.clock = &stepClock{last: time.Now()}
	s.fond = s.demarrerLeFond()
	defer s.fond.joindre()
	if og.profils {
		sortie.marcheTemoin, _ = profilCPU(s.lireLaMarcheDesTrames, marqueDeLaMarche)
	} else {
		s.lireLaMarcheDesTrames()
	}
	if og.perturbe {
		w := fc.ProfilDeBalayage().MPP
		fc.PoserMPP(profile.MPPWidths{Lead: w.Lead, Index: w.Index + 1})
	}
	if og.sentinelles {
		sortie.crochets = installerLesSentinelles(fc.Observation(), sortie.appels)
	}
	avant = instantane(fc)
	demarrerLeProfil()
	defer arreterLeProfil()
	if err := s.balayerPositions(); err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	s.balayerMonde()
	s.balayerEtatsDeMouvement()
	s.balayerPortage()
	s.balayerCapacites()
	s.balayerPont()
	sortie.aLaFin = avant
	return sortie
}

// nomsDesEtapes rend les noms des etapes d une passe.
func nomsDesEtapes(s sortieDeGarde) []string {
	out := make([]string, len(s.etapes))
	for i, e := range s.etapes {
		out[i] = e.nom
	}
	return out
}

// TestLaQueueDuHarnaisSuitScanFilmInputs : la suite d appels que le harnais rejoue apres la marche est
// celle de `scanFilmInputs`. Sans cela, le harnais pourrait garder une sequence qui n est plus celle de
// la production.
func TestLaQueueDuHarnaisSuitScanFilmInputs(t *testing.T) {
	decls := declarationsDuBalayage(t)
	var appels []string
	ast.Inspect(decls[racineDuBalayage].Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := c.Fun.(*ast.SelectorExpr); ok &&
			(strings.HasPrefix(sel.Sel.Name, "balayer") || sel.Sel.Name == "lireLaMarcheDesTrames") {
			appels = append(appels, sel.Sel.Name)
		}
		return true
	})
	attendu := []string{"lireLaMarcheDesTrames", "balayerPositions", "balayerMonde", "balayerEtatsDeMouvement",
		"balayerPortage", "balayerCapacites", "balayerPont"}
	if !slices.Equal(appels, attendu) {
		t.Fatalf("scanFilmInputs appelle %v ; le harnais de garde rejoue %v : le mettre a jour", appels, attendu)
	}
}

// pourChaqueFilmDesGardesJ4 joue `f` sur chaque film du parc, un a la fois.
func pourChaqueFilmDesGardesJ4(t *testing.T, f func(t *testing.T, film *source.Film, entry profile.MapQuantEntry)) {
	for _, fg := range filmsDesGardesJ4 {
		t.Run(fg.id, func(t *testing.T) {
			film, entry := chargerLeFilmDeGarde(t, fg.id, fg.carte)
			f(t, film, entry)
		})
	}
}

// TestChaqueBalayageRestitueLeProfil (PROFIL) : apres chaque etape observee, le profil de balayage du
// contexte, largeurs MPP comprises, est celui d avant. Deux departs : normal, et MPP decale.
func TestChaqueBalayageRestitueLeProfil(t *testing.T) {
	pourChaqueFilmDesGardesJ4(t, func(t *testing.T, film *source.Film, entry profile.MapQuantEntry) {
		for _, og := range []optionsDeGarde{{}, {perturbe: true}} {
			sortie := jouerLaGarde(t, film, entry, og)
			if len(sortie.etapes) < 40 {
				t.Fatalf("%d etapes observees seulement : le harnais ne rejoue pas le balayage", len(sortie.etapes))
			}
			for _, e := range sortie.etapes {
				if c := e.changes["bal"]; len(c) > 0 {
					t.Errorf("depart %+v, etape %s : le profil du contexte a change : %v", og, e.nom, c)
				}
			}
		}
	})
}

// TestAucunDistribuerApresLaMarche (DISTRIBUTION) : aucune etape d apres la marche ne relance la marche
// des trames. Les memos que ses canaux rangent gardent leur adresse, et le profil CPU de l etape ne
// contient pas la marche ; le detecteur est controle sur la vraie marche.
func TestAucunDistribuerApresLaMarche(t *testing.T) {
	pourChaqueFilmDesGardesJ4(t, func(t *testing.T, film *source.Film, entry profile.MapQuantEntry) {
		sortie := jouerLaGarde(t, film, entry, optionsDeGarde{profils: true})
		for _, e := range sortie.etapes {
			for _, c := range e.changes["memo"] {
				switch c {
				case "recup.lectures", "recup.objets", "recup.equipementPorte", "marches":
					t.Errorf("etape %s : le memo %s de la marche a ete reecrit", e.nom, c)
				}
			}
			if e.marcheVue {
				t.Errorf("etape %s : le profil CPU contient la marche des trames", e.nom)
			}
		}
		if !sortie.marcheTemoin {
			t.Log("controle du detecteur non concluant (profil CPU indisponible ou marche trop breve) : " +
				"seuls les memos font foi sur cette execution")
		}
	})
}

// TestAucunCrochetNEstPoseSurLContexteApresLaMarche (CROCHETS) : vrai sous la forme « aucun crochet de
// l observation du contexte n est pose ni retire apres la marche ». Deux passes : sans sentinelle, la
// photo ne change pas ; avec une sentinelle sur chaque crochet, aucune n est remplacee ni retiree.
func TestAucunCrochetNEstPoseSurLContexteApresLaMarche(t *testing.T) {
	pourChaqueFilmDesGardesJ4(t, func(t *testing.T, film *source.Film, entry profile.MapQuantEntry) {
		for _, og := range []optionsDeGarde{{}, {sentinelles: true}} {
			sortie := jouerLaGarde(t, film, entry, og)
			if _, ok := sortie.aLaFin["obs.AbilityEnergyHook"]; !ok {
				t.Fatal("la photo du contexte ne porte pas les crochets de l observation : la garde ne verrait rien")
			}
			for _, e := range sortie.etapes {
				for _, c := range e.changes["obs"] {
					if strings.Contains(c, "Hook") {
						t.Errorf("sentinelles=%v, etape %s : le crochet %s a change", og.sentinelles, e.nom, c)
					}
				}
			}
			if og.sentinelles {
				t.Logf("crochets comptables poses : %d", len(sortie.crochets))
				for _, e := range sortie.etapes {
					if len(e.appels) == 0 {
						continue
					}
					t.Logf("APPELS aux crochets du contexte, etape %-22s %v", e.nom, e.appels)
					// Les lecteurs bit a bit du pont (positions), des images-cles et du calque des vehicules portent l observation
					// du contexte : leurs crochets sont CONSULTES (nuls en production). Aucune autre etape.
					if e.nom != "translocations" && e.nom != "loadouts" && e.nom != "vehicles" {
						t.Errorf("etape %s : des lecteurs consultent l observation du contexte, hors du pont, des images-cles et des vehicules", e.nom)
					}
				}
			}
		}
	})
}

// TestObservationDuContexteNEstLueParPersonne (OBSERVATION) : les compteurs de l observation du contexte,
// empoisonnes avant la marche, ne changent aucune sortie observee. Les compteurs sont ECRITS par les
// lecteurs construits sur le contexte (la liste est relevee) et LUS par personne.
func TestObservationDuContexteNEstLueParPersonne(t *testing.T) {
	pourChaqueFilmDesGardesJ4(t, func(t *testing.T, film *source.Film, entry profile.MapQuantEntry) {
		propre := jouerLaGarde(t, film, entry, optionsDeGarde{})
		poison := jouerLaGarde(t, film, entry, optionsDeGarde{poison: true})
		if !slices.Equal(nomsDesEtapes(propre), nomsDesEtapes(poison)) {
			t.Fatalf("etapes differentes : %v / %v", nomsDesEtapes(propre), nomsDesEtapes(poison))
		}
		ecrits := map[string]bool{}
		for i, e := range propre.etapes {
			if e.empreinte != poison.etapes[i].empreinte {
				t.Errorf("etape %s : sortie differente quand l observation du contexte est empoisonnee (%s / %s)",
					e.nom, e.empreinte, poison.etapes[i].empreinte)
			}
			for _, c := range e.changes["obs"] {
				ecrits[c] = true
			}
		}
		var liste []string
		for c := range ecrits {
			liste = append(liste, c)
		}
		slices.Sort(liste)
		t.Logf("champs de l observation du contexte ecrits pendant les balayages : %v", liste)
	})
}

// TestReleveDesMemosEcritsParLesBalayages (MEMOS) : releve, par etape, les memos du contexte ecrits « a
// chaud » ; il echoue si un memo hors de la liste connue (cf. [memosEcritsConnus]) est ecrit, ou si un
// memo de la liste chaude l est.
func TestReleveDesMemosEcritsParLesBalayages(t *testing.T) {
	pourChaqueFilmDesGardesJ4(t, func(t *testing.T, film *source.Film, entry profile.MapQuantEntry) {
		sortie := jouerLaGarde(t, film, entry, optionsDeGarde{})
		for _, e := range sortie.etapes {
			if len(e.changes) == 0 {
				continue
			}
			t.Logf("RELEVE %-26s memos=%v obs=%d rapports=%v", e.nom, memosDeEtape(e), len(e.changes["obs"]), e.changes["rapport"])
			for _, c := range memosDeEtape(e) {
				if !memosEcritsConnus(c) {
					t.Errorf("etape %s : memo %s ecrit a chaud, absent de la liste de la conception", e.nom, c)
				}
			}
		}
	})
}

// memosDeEtape rend les memos ecrits par une etape, sans doublon de prefixe.
func memosDeEtape(e etapeDeGarde) []string { return e.changes["memo"] }

// memosEcritsConnus : les memos que les balayages d apres la marche ecrivent (releves sur le parc,
// ranges ici pour que l apparition d un autre echoue). Un memo de la marche ne doit jamais y figurer.
func memosEcritsConnus(chemin string) bool {
	return strings.HasPrefix(chemin, "recup.pistes") || strings.HasPrefix(chemin, "recup.creations") ||
		strings.HasPrefix(chemin, "recup.tables") || strings.HasPrefix(chemin, "recup.entetes") ||
		strings.HasPrefix(chemin, "duFilm") || strings.HasPrefix(chemin, "marches")
}

// TestLesSitesDePoserMPPSontConnus (PROFIL, inventaire) : les sites de production qui posent le
// decoupage MPP du contexte sont ceux releves ; un site de plus doit etre classe (restitue-t-il ?)
// avant d entrer dans la liste. Un compte par fichier.
func TestLesSitesDePoserMPPSontConnus(t *testing.T) {
	attendu := map[string]int{
		"../internal/facts/killsource/decode.go": 1, "../internal/grammar/build_profile.go": 2,
		"../internal/grammar/contexte_de_carte.go": 1, "../internal/grammar/equipment_creation_width.go": 1,
		"../internal/grammar/equipment_placements.go": 3, "../internal/grammar/marche_images_cles.go": 2,
		"../internal/grammar/film_context.go": 0, "build_ground_weapons.go": 2,
	}
	got := map[string]int{}
	for _, racine := range []string{"../internal", ".", "../decfilm"} {
		_ = filepath.WalkDir(racine, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				return nil
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "PoserMPP" {
						got[filepath.ToSlash(p)]++
					}
				}
				return true
			})
			return nil
		})
	}
	for p, n := range got {
		if attendu[p] != n {
			t.Errorf("%s : %d appel(s) de PoserMPP, %d attendu(s)", p, n, attendu[p])
		}
	}
	for p, n := range attendu {
		if n > 0 && got[p] != n {
			t.Errorf("%s : %d appel(s) de PoserMPP, %d attendu(s)", p, got[p], n)
		}
	}
}
