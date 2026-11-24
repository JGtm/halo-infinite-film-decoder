package replay

// film_scan_voies_test.go — LES VOIES DU BALAYAGE NE CHANGENT RIEN A LA SORTIE.
//
//	VOIE-ORDRE      les etapes observees (nom, ordre, empreinte), les entrees lues, les replis et les
//	                diagnostics sont les memes a 1, 2, 4 et 8 fils ;
//	VOIE-UN-FIL     a un fil, ou sans contexte chaud, ou marche en erreur : aucune voie, aucune goroutine ;
//	                a plusieurs fils, rien ne survit au balayage, sortie anticipee comprise ;
//	VOIE-MONDE      la voie du monde rend les memes lectures, et ses replis ne se comptent qu une fois ;
//	VOIE-SIEGES     la voie des sieges rend les memes equipes, occupants et entrees du tableau des scores ;
//	VOIE-PANIQUE    la panique d une voie est rejouee une fois sur le fil principal ;
//	VOIE-SOURCE     les lectures des voies sont les memes appels que celles de la chaine.
//
// Les bobines sont celles du decodeur (cf. film_scan_fond_test.go). Le balayage des positions les refuse :
// `scanFilmInputs` y rend l erreur de ce refus apres les premieres etapes. Les tests de l identite jouent donc
// les phases d apres les positions dans l ordre de `scanFilmInputs` ([balayerSansPositions]), depuis le
// depart des voies jusqu a leur fusion ; la marche des trames, elle, reussit sur ces deux bobines, qui
// portent de vraies poses, de vrais socles, des lancers, des equipes et des entrees de tableau des scores.

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/digest"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// sortieDesVoies est ce que le balayage d une bobine a produit : tout ce qui doit etre egal d un nombre de
// fils a l autre.
type sortieDesVoies struct {
	sortieDuFond
	voies bool
	// champs : l empreinte de chaque entree lue, par nom.
	champs map[string]string
	// diagnostics : les diagnostics du contexte, hors `registre_inconnu` (cf. [diagnosticsDuContexte]).
	diagnostics []string
}

// diagnosticsDuContexte rend les diagnostics du contexte, mis en ligne. `registre_inconnu` est ecarte : la
// grammaire ne le signale qu une fois par empreinte de registre et par PROCESSUS (`registryWarned`), donc au
// premier film lu du test, quel que soit le nombre de fils.
func diagnosticsDuContexte(fc *grammar.FilmContext) []string {
	var out []string
	for _, d := range fc.Diagnostics().Relever() {
		if d.Code == grammar.DiagRegistreInconnu {
			continue
		}
		out = append(out, fmt.Sprintf("%s %v", d.Code, d.Attrs))
	}
	return out
}

// nouveauBalayage ouvre le balayage d une bobine au nombre de fils demande, comme `scanFilmInputs` le fait
// avant la marche : contexte neuf, tache de fond.
func nouveauBalayage(t *testing.T, dossier string, fils int, sortie *sortieDesVoies) *filmScan {
	t.Helper()
	film, err := source.LoadDir(dossier, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dossier, err)
	}
	opt := optionsDuFond(t, &sortie.sortieDuFond)
	fc := grammar.NewFilmContextForMap(film, opt.MapQuant, decoupageForce(opt))
	poserLesFils(fc, OptionsVitesse{Fils: fils})
	s := &filmScan{ctx: context.Background(), matchID: "bobine", film: film, fc: fc, opt: opt, world: opt.MapQuant.Range()}
	s.scan = grammar.DefaultScanFilmOptions()
	s.scan.WorldRange = &s.world
	s.scan.CaptureDirs = true
	s.scan.Layout = fc.ImposedLayout()
	s.opt.clock = &stepClock{}
	s.fond = s.demarrerLeFond()
	return s
}

// empreintes range l empreinte de chaque entree que les voies produisent (monde, sieges) et que la chaine
// publie, par nom.
func (s *filmScan) empreintes() map[string]string {
	out := map[string]string{}
	for _, e := range []struct {
		nom string
		val any
	}{
		{"ZoomEvents", s.in.ZoomEvents}, {"Placements", s.in.Placements}, {"PlacementStats", s.in.PlacementStats},
		{"SpawnEvents", s.in.SpawnEvents}, {"SpawnStats", s.in.SpawnStats}, {"Pads", s.in.Pads},
		{"Vehicles", s.in.Vehicles}, {"Grenades", s.in.Grenades}, {"Projectiles", s.in.Projectiles},
		{"Detonations", s.in.Detonations}, {"FilmTable", s.in.FilmTable}, {"PlayerTeams", s.in.PlayerTeams},
		{"TeamScan", s.in.TeamScan}, {"PlayerEntities", s.in.PlayerEntities}, {"RosterListings", s.in.RosterListings},
		{"RosterListingStats", s.in.RosterListingStats}, {"StatborgEntries", s.in.StatborgEntries},
		{"StatborgEntryStats", s.in.StatborgEntryStats}, {"Deaths", s.in.Deaths}, {"Scripts", s.in.Scripts},
		{"PlayerIndices", s.in.PlayerIndices}, {"FilmClockOriginUS", s.in.FilmClockOriginUS},
		{"Entrees", s.in},
	} {
		compte, somme := digest.Of(e.val)
		out[e.nom] = fmt.Sprintf("%d %s", compte, somme)
	}
	return out
}

// balayerSansPositions rejoue, dans l ordre de `scanFilmInputs` et sans les positions que la bobine refuse,
// la marche, le depart des voies, les phases qui les consomment et la fusion.
func balayerSansPositions(t *testing.T, dossier string, fils int) sortieDesVoies {
	t.Helper()
	var sortie sortieDesVoies
	s := nouveauBalayage(t, dossier, fils, &sortie)
	defer s.fond.joindre()
	s.lireLaMarcheDesTrames()
	s.voies = s.demarrerLesVoies()
	defer s.voies.joindre()
	sortie.voies = s.voies != nil
	s.publierPontDIdentite(s.lirePontDIdentite())
	s.balayerMonde()
	s.balayerEtatsDeMouvement()
	s.balayerPortage()
	s.balayerCapacites()
	s.balayerPont()
	s.voies.fusionner()
	sortie.replis = s.fc.ComptesDesReplis()
	sortie.champs = s.empreintes()
	sortie.diagnostics = diagnosticsDuContexte(s.fc)
	return sortie
}

// balayerEnEntier joue `scanFilmInputs` : la bobine y est refusee aux positions, apres le depart des voies.
func balayerEnEntier(t *testing.T, dossier string, fils int) sortieDesVoies {
	t.Helper()
	sortie := sortieDesVoies{sortieDuFond: balayerLaBobine(t, dossier, fils)}
	return sortie
}

// plusDeFils : les nombres de fils que les tests comparent a un fil.
var plusDeFils = []int{2, 4, 8}

// comparerLesSorties signale tout ecart entre la sortie d un fil et celle de `fils` fils.
func comparerLesSorties(t *testing.T, contexte string, ref, got sortieDesVoies, fils int) {
	t.Helper()
	if !slices.Equal(got.etapes, ref.etapes) {
		t.Errorf("%s a %d fils : etapes observees differentes\n  un fil  %v\n  %d fils %v", contexte, fils, ref.etapes, fils, got.etapes)
	}
	for nom, want := range ref.champs {
		if got.champs[nom] != want {
			t.Errorf("%s a %d fils : l entree %s differe de celle d un fil", contexte, fils, nom)
		}
	}
	if got.replis != ref.replis {
		t.Errorf("%s a %d fils : replis differents\n  un fil  %+v\n  %d fils %+v", contexte, fils, ref.replis, fils, got.replis)
	}
	if !slices.Equal(got.diagnostics, ref.diagnostics) {
		t.Errorf("%s a %d fils : diagnostics differents\n  un fil  %v\n  %d fils %v", contexte, fils, ref.diagnostics,
			fils, got.diagnostics)
	}
}

// TestVoieOrdreDesEtapesIdentique (VOIE-ORDRE) : a 2, 4 et 8 fils, les voies tournent (le test le verifie)
// et la sortie est celle d un fil, etape par etape. Mutations vues rouges : une voie qui publie hors de son
// tour (`balayerMonde` publie les vehicules avant les poses), un repli verse deux fois (le fil principal lit
// les poses sur le contexte du film avant de deriver), la fusion omise, une voie qui lit sur le contexte du
// parent (course de donnees, `-race`).
func TestVoieOrdreDesEtapesIdentique(t *testing.T) {
	avecHuitProcesseurs(t)
	for _, dossier := range bobinesDuFond {
		ref := balayerSansPositions(t, dossier, 1)
		if ref.voies {
			t.Fatalf("%s : des voies a un fil", dossier)
		}
		if len(ref.etapes) == 0 {
			t.Fatalf("%s : aucune etape observee a un fil, le test ne prouverait rien", dossier)
		}
		for _, fils := range plusDeFils {
			got := balayerSansPositions(t, dossier, fils)
			if !got.voies {
				t.Fatalf("%s a %d fils : aucune voie lancee, le test ne prouverait rien", dossier, fils)
			}
			comparerLesSorties(t, dossier, ref, got, fils)
		}
		// Les memes bobines par `scanFilmInputs` : etapes et erreur du refus des positions.
		refEntier := balayerEnEntier(t, dossier, 1)
		for _, fils := range plusDeFils {
			got := balayerEnEntier(t, dossier, fils)
			if !slices.Equal(got.etapes, refEntier.etapes) || got.err != refEntier.err || got.replis != refEntier.replis {
				t.Errorf("%s scanFilmInputs a %d fils : sortie differente de celle d un fil", dossier, fils)
			}
		}
	}
}

// TestVoieNeLitQueDeVraiesValeurs : la comparaison porte sur de vraies valeurs. La premiere bobine porte des
// poses, des socles, des lancers, des equipes et des entrees du tableau des scores ; la seconde des socles et
// des equipes.
func TestVoieNeLitQueDeVraiesValeurs(t *testing.T) {
	avecHuitProcesseurs(t)
	vide := func(valeur string) bool { return strings.HasPrefix(valeur, "0 ") }
	sortie := balayerSansPositions(t, bobinesDuFond[0], 4)
	for _, nom := range []string{"Placements", "Pads", "Grenades", "PlayerTeams", "StatborgEntries", "FilmTable"} {
		if vide(sortie.champs[nom]) {
			t.Errorf("entree %s vide sur la premiere bobine : elle ne prouve rien", nom)
		}
	}
	if sortie.replis.PistesDuMondeApresLaMarche == 0 || sortie.replis.CreationsDuMondeApresLaMarche == 0 {
		t.Errorf("replis du monde a zero (%+v) : la bobine ne prouve pas le compte unique", sortie.replis)
	}
}

// TestVoieMondeIdentique (VOIE-MONDE) : la lunette, les poses, les pieces engendrees, les socles, les
// vehicules, les lancers, les projectiles et les detonations sont ceux d un fil, aux largeurs des poses.
// Mutation vue rouge : la voie lit les socles avant les poses (largeurs nulles).
func TestVoieMondeIdentique(t *testing.T) {
	avecHuitProcesseurs(t)
	monde := []string{"ZoomEvents", "Placements", "PlacementStats", "SpawnEvents", "SpawnStats", "Pads", "Vehicles",
		"Grenades", "Projectiles", "Detonations"}
	for _, dossier := range bobinesDuFond {
		ref := balayerSansPositions(t, dossier, 1)
		for _, fils := range plusDeFils {
			got := balayerSansPositions(t, dossier, fils)
			for _, nom := range monde {
				if got.champs[nom] != ref.champs[nom] {
					t.Errorf("%s a %d fils : %s differe de celui d un fil", dossier, fils, nom)
				}
			}
		}
	}
}

// TestVoieMondeCompteLesReplisUneSeuleFois (VOIE-MONDE) : les replis que releve la voie du monde (pistes et
// creations « apres la marche », largeurs MPP) ne sont comptes ni deux fois, ni zero fois : ceux d un fil.
// Mutation vue rouge : le fil principal relit les poses sur son contexte avant de deriver les voies.
func TestVoieMondeCompteLesReplisUneSeuleFois(t *testing.T) {
	avecHuitProcesseurs(t)
	for _, dossier := range bobinesDuFond {
		ref := balayerSansPositions(t, dossier, 1).replis
		for _, fils := range plusDeFils {
			got := balayerSansPositions(t, dossier, fils).replis
			if got.PistesDuMondeApresLaMarche != ref.PistesDuMondeApresLaMarche ||
				got.CreationsDuMondeApresLaMarche != ref.CreationsDuMondeApresLaMarche ||
				got.LargeursMPPCalibrees != ref.LargeursMPPCalibrees || got.LargeursMPPParDefaut != ref.LargeursMPPParDefaut {
				t.Errorf("%s a %d fils : replis du monde %d/%d/%d/%d, un fil %d/%d/%d/%d (pistes, creations, calibrees, defaut)",
					dossier, fils, got.PistesDuMondeApresLaMarche, got.CreationsDuMondeApresLaMarche,
					got.LargeursMPPCalibrees, got.LargeursMPPParDefaut, ref.PistesDuMondeApresLaMarche,
					ref.CreationsDuMondeApresLaMarche, ref.LargeursMPPCalibrees, ref.LargeursMPPParDefaut)
			}
		}
	}
}

// TestVoieSiegesIdentique (VOIE-SIEGES) : la table du film, les listages, les equipes, les occupants et les
// entrees du tableau des scores sont ceux d un fil. Mutation vue rouge : la voie des sieges lit sur le
// contexte du parent (course de donnees sur les memos, `-race`).
func TestVoieSiegesIdentique(t *testing.T) {
	avecHuitProcesseurs(t)
	sieges := []string{"FilmTable", "RosterListings", "RosterListingStats", "PlayerTeams", "TeamScan", "PlayerEntities",
		"StatborgEntries", "StatborgEntryStats"}
	for _, dossier := range bobinesDuFond {
		ref := balayerSansPositions(t, dossier, 1)
		for _, fils := range plusDeFils {
			got := balayerSansPositions(t, dossier, fils)
			for _, nom := range sieges {
				if got.champs[nom] != ref.champs[nom] {
					t.Errorf("%s a %d fils : %s differe de celui d un fil", dossier, fils, nom)
				}
			}
		}
	}
}

// voiesAUnFilAucuneGoroutine (VOIE-UN-FIL, appelee par TestVoieAUnFilAucuneGoroutine) : a un fil, aucune voie, aucune derivation, aucune goroutine ;
// les accesseurs d un pointeur nil rendent « rien d avance » ; a plusieurs fils, rien ne survit au balayage,
// sortie anticipee comprise.
func voiesAUnFilAucuneGoroutine(t *testing.T) {
	t.Helper()
	avecHuitProcesseurs(t)
	for _, fils := range []int{0, 1} {
		var sortie sortieDesVoies
		s := nouveauBalayage(t, bobinesDuFond[0], fils, &sortie)
		s.lireLaMarcheDesTrames()
		avant := runtime.NumGoroutine()
		if v := s.demarrerLesVoies(); v != nil {
			t.Fatalf("Fils=%d : des voies sont lancees", fils)
		}
		if got := runtime.NumGoroutine(); got != avant {
			t.Errorf("Fils=%d : %d goroutines avant, %d apres le depart", fils, avant, got)
		}
		var rien *voiesDuBalayage
		if _, ok := rien.poses(); ok {
			t.Errorf("Fils=%d : un pointeur nil rend une valeur d avance", fils)
		}
		if _, ok := rien.occupants(); ok {
			t.Errorf("Fils=%d : un pointeur nil rend des occupants d avance", fils)
		}
		rien.fusionner()
		rien.joindre()
	}
	// A plusieurs fils : le balayage qui sort tot (erreur des positions) attend les voies.
	avant := runtime.NumGoroutine()
	balayerLaBobine(t, bobinesDuFond[0], 4)
	balayerSansPositions(t, bobinesDuFond[0], 4)
	if !goroutinesRetombees(avant) {
		t.Errorf("%d goroutines avant, %d apres le balayage a 4 fils", avant, runtime.NumGoroutine())
	}
}

// TestVoieRepliAUnFilSiMarcheEnErreur (VOIE-UN-FIL) : une marche en erreur, ou pas encore jouee, ou un
// contexte qui n est pas chaud : aucune voie, aucune goroutine, et la chaine lit en serie la meme sortie. Une
// marche en erreur relance, dans plusieurs balayages, une marche de repli que seul le contexte principal mene.
func TestVoieRepliAUnFilSiMarcheEnErreur(t *testing.T) {
	avecHuitProcesseurs(t)
	dossier := bobinesDuFond[0]
	ref := balayerSansPositions(t, dossier, 1)
	cas := []struct {
		nom     string
		prepare func(s *filmScan)
		attendu bool // des voies sont-elles lancees ?
	}{
		{"marche en erreur", func(s *filmScan) { s.lireLaMarcheDesTrames(); s.errMarche = errors.New("marche refusee") }, false},
		{"marche pas jouee", func(s *filmScan) {}, false},
		{"marche reussie", func(s *filmScan) { s.lireLaMarcheDesTrames() }, true},
	}
	for _, c := range cas {
		var sortie sortieDesVoies
		s := nouveauBalayage(t, dossier, 4, &sortie)
		c.prepare(s)
		avant := runtime.NumGoroutine()
		s.voies = s.demarrerLesVoies()
		if (s.voies != nil) != c.attendu {
			t.Errorf("%s : voies lancees = %v, attendu %v", c.nom, s.voies != nil, c.attendu)
		}
		if !c.attendu && runtime.NumGoroutine() > avant {
			t.Errorf("%s : %d goroutines avant, %d apres le depart", c.nom, avant, runtime.NumGoroutine())
		}
		s.voies.joindre()
		s.fond.joindre()
	}
	// Sans voie, la chaine d un contexte dont la marche est faite mais a un fil rend la sortie de reference.
	if got := balayerSansPositions(t, dossier, 1); !slices.Equal(got.etapes, ref.etapes) || got.replis != ref.replis {
		t.Error("deux balayages a un fil rendent deux sorties")
	}
}

// TestVoiePaniqueDUneVoieRelayee (VOIE-PANIQUE) : la panique d une voie est rejouee, telle quelle, sur le fil
// principal qui attend la premiere valeur que la voie n a pas rendue ; elle n est rejouee qu une fois ; les
// valeurs rendues avant la panique restent lisibles ; la jointure ne la rejoue pas.
func TestVoiePaniqueDUneVoieRelayee(t *testing.T) {
	rattraper := func(fn func()) (p any) {
		defer func() { p = recover() }()
		fn()
		return nil
	}
	nouveau := func(panique bool) (*voiesDuBalayage, *attendue[int], *attendue[int]) {
		l := &voie{}
		avant, apres := nouvelleAttendue[int](l), nouvelleAttendue[int](l)
		l.lancer(func() {
			livrer(avant, 7)
			if panique {
				panic("panique de la voie")
			}
			livrer(apres, 8)
		})
		return &voiesDuBalayage{monde: &voieDuMonde{voie: l}, sieges: &voieDesSieges{voie: &voie{fini: closed()}}}, avant, apres
	}

	v, avant, apres := nouveau(true)
	if got := attendreVoie(v, avant); got != 7 {
		t.Fatalf("valeur rendue avant la panique = %d, attendu 7", got)
	}
	if p := rattraper(func() { attendreVoie(v, apres) }); p != "panique de la voie" {
		t.Fatalf("panique rejouee = %v, attendu la panique de la voie", p)
	}
	if p := rattraper(func() { attendreVoie(v, apres); v.joindre(); v.joindre() }); p != nil {
		t.Errorf("la panique est rejouee une seconde fois : %v", p)
	}

	// Personne n attend de valeur : la jointure rejoue la panique, une fois.
	g, _, _ := nouveau(true)
	if p := rattraper(g.joindre); p != "panique de la voie" {
		t.Errorf("jointure : panique rejouee = %v, attendu la panique de la voie", p)
	}
	if p := rattraper(g.joindre); p != nil {
		t.Errorf("seconde jointure : panique %v", p)
	}

	// Pas de panique : la jointure rend la main et la voie est arretee ou terminee.
	h, _, _ := nouveau(false)
	if p := rattraper(h.joindre); p != nil {
		t.Errorf("jointure sans panique : %v", p)
	}
}

// closed rend un canal deja ferme : la voie factice qui n a rien a attendre.
func closed() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

// TestVoieArretEntreDeuxLectures : une voie arretee ne lit plus apres la lecture en cours, et la jointure rend
// la main sans panique ; les valeurs qu elle n a pas lues sont closes (aucun lecteur ne reste bloque).
func TestVoieArretEntreDeuxLectures(t *testing.T) {
	l := &voie{}
	premiere, seconde := nouvelleAttendue[int](l), nouvelleAttendue[int](l)
	debut, fin := make(chan struct{}), make(chan struct{})
	l.lancer(func() {
		livrer(premiere, 1)
		close(debut)
		<-fin
		if l.arretee() {
			return
		}
		livrer(seconde, 2)
	})
	<-debut
	l.arret.Store(true)
	close(fin)
	<-l.fini
	if l.panique != nil {
		t.Fatalf("voie arretee sur demande : panique %v", l.panique)
	}
	select {
	case <-seconde.pret:
	default:
		t.Fatal("valeur non lue de la voie arretee : canal reste ouvert")
	}
}

// TestLesLecturesDesVoiesSontCellesDeLaChaine (VOIE-SOURCE). Mutations vues rouges : ajouter une lecture aux
// voies que la chaine ne fait pas ; ajouter une troisieme instruction `go` dans `replay` ; mettre
// film_scan_voies.go dans fichiersDuBalayage.
func TestLesLecturesDesVoiesSontCellesDeLaChaine(t *testing.T) {
	appels := appelsParFonction(t)
	// Chaque lecture qu une voie fait est faite aussi, par le meme nom, par la chaine en serie.
	lecturesDeLaChaine := map[string]string{
		"lireLaLunette": "lireLunette", "decodeFilmPlacementsLecture": "lirePoses",
		"decodeFilmSpawnEventsLecture": "lirePiecesEngendrees", "decodeFilmPadScansLecture": "lireSocles",
		"decodeFilmVehicleScanLecture": "lireVehicules", "lireLesGrenades": "lireGrenades",
		"lireLesProjectiles": "lireProjectiles", "lireLesDetonations": "lireDetonations",
		"lireLesOccupants": "lireSiegesEtOccupantsLecture",
	}
	vues := map[string]bool{}
	for _, voie := range []string{"lire", "lancerLaVoieDesSieges"} {
		for _, appele := range appels[voie] {
			if consommateur, ok := lecturesDeLaChaine[appele]; ok {
				vues[appele] = true
				if !slices.Contains(appels[consommateur], appele) {
					t.Errorf("la voie lit %s, que %s ne lit pas", appele, consommateur)
				}
			}
		}
	}
	for lecture := range lecturesDeLaChaine {
		if !vues[lecture] {
			t.Errorf("la lecture %s n est faite par aucune voie : la table du garde est perimee", lecture)
		}
	}
	for _, fichier := range []string{"film_scan_voies.go", "film_scan_voie_monde.go", "film_scan_voie_sieges.go"} {
		if slices.Contains(fichiersDuBalayage, fichier) {
			t.Errorf("%s est dans fichiersDuBalayage : le garde des etapes compterait ses lectures deux fois", fichier)
		}
	}
	// Une voie ne publie, ne journalise, ne verse ni n observe : aucun de ces appels dans ses fichiers.
	interdits := []string{"observe", "NoterReplis", "Declenche", "DeclencheN", "InfoContext", "WarnContext", "ErrorContext"}
	for _, fichier := range []string{"film_scan_voie_monde.go", "film_scan_voie_sieges.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), fichier, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if nom, ok := nomAppele(call); ok && slices.Contains(interdits, nom) {
					t.Errorf("%s appelle %s : une voie ne publie rien", fichier, nom)
				}
			}
			return true
		})
	}
}

// TestLesDiagnosticsDeLaVoieSontVerses : les diagnostics que la voie du monde signale (couverture des
// lancers) arrivent chez le parent, une fois. Un diagnostic note sur la voie et jamais verse se perdrait.
func TestLesDiagnosticsDeLaVoieSontVerses(t *testing.T) {
	avecHuitProcesseurs(t)
	ref := balayerSansPositions(t, bobinesDuFond[0], 1)
	got := balayerSansPositions(t, bobinesDuFond[0], 4)
	if len(ref.diagnostics) == 0 {
		t.Fatal("la bobine ne signale aucun diagnostic : le test ne prouverait rien")
	}
	if !slices.Equal(got.diagnostics, ref.diagnostics) {
		t.Errorf("diagnostics a 4 fils %v, un fil %v", got.diagnostics, ref.diagnostics)
	}
}
