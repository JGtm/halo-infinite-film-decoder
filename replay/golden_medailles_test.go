package replay

// golden_medailles_test.go — LES MEDAILLES DES HUIT BUILDS, FIGEES A COTE DE LEURS ENTREES, ET LE
// CODEC QUI LES PORTE.
//
// Les medailles voyagent dans le COMPLEMENT de la section 1 du fichier de faits, pas dans le blob
// des entrees : les `inputs_<short8>.bin.gz` sont relus sans elles, et les goldens d assemblage ne
// publieraient aucun `medals`. Meme forme que l equipement porte (golden_equipement_test.go) : un
// fichier par build, `medailles_<short8>.bin.gz`, encode par le codec de production
// ([encodeMedailles]), pose sur les entrees relues ([chargerGoldenBuild], [loadGoldenInputs]).
//
// REGENERATION (jamais d edition a la main), depuis le film, par le balayage de la production,
// UN FILM PAR PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./replay/ -run 'TestGoldenMedaillesRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu elle a ecrit ; on relance sans les drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/highlightevent"
)

// medaillesPath : le fichier des medailles d un build.
func medaillesPath(short8 string) string {
	return filepath.Join(goldenDir, "medailles_"+short8+".bin.gz")
}

// chargerMedaillesDuBuild pose sur `g` les medailles figees d un build. ABSENT = ECHEC : les huit
// builds les portent, et un build qui les perdrait publierait un calque absent.
func chargerMedaillesDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(medaillesPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("medailles du build %s absentes (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenMedaillesRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("medailles %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("medailles %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	g.MedalsRead, g.Medals = decodeMedailles(r)
	if r.err != nil || r.off != len(r.b) || !g.MedalsRead {
		t.Fatalf("medailles %s : relecture invalide (err %v, %d/%d octets lus, lues %v)", short8, r.err,
			r.off, len(r.b), g.MedalsRead)
	}
}

// TestGoldenMedaillesRegenerate : LA SEULE PORTE D ECRITURE des medailles figees (cf. l en-tete).
func TestGoldenMedaillesRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des medailles des builds : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des medailles des builds : " + miniFilmCacheEnv + " non defini")
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant(t)
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			g, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("decodage de %s : %v", b.Short8, err)
			}
			n := ecrireMedaillesDuBuild(t, b.Short8, g)
			t.Fatalf("REECRIT %s (%d medailles, %d octets) — relancer sans -update pour verifier",
				medaillesPath(b.Short8), len(g.Medals), n)
		})
	}
}

// renderMedailles publie le calque des medailles dans le golden d assemblage : sa couverture, le
// compte par nom, la premiere et la derniere medaille.
func renderMedailles(p func(string, ...any), doc ReplayDocument) {
	p("## MEDAILLES — posees par (ms - originMs) / pas, nom anglais ; absentes si le morceau des temps forts n a pas ete lu")
	c := doc.Coverage.Medals
	if c == nil {
		p("couverture absente : morceau des temps forts non lu")
		return
	}
	p("%d lue(s) · %d posee(s) · %d sans origine · %d avant la frame 0 · %d apres la fin · %d sans nom",
		c.Decoded, c.Published, c.NoOrigin, c.BeforeOrigin, c.AfterEnd, c.UnknownName)
	parNom := map[string]int{}
	for _, m := range doc.Medals {
		parNom[fmt.Sprintf("%d %s", m.ID, m.Name)]++
	}
	for _, k := range slices.Sorted(maps.Keys(parNom)) {
		p("  %s x%d", k, parNom[k])
	}
	if n := len(doc.Medals); n > 0 {
		f, l := doc.Medals[0], doc.Medals[n-1]
		p("premiere : t=%d ms=%d xuid=%s id=%d · derniere : t=%d ms=%d xuid=%s id=%d", f.T, f.MS, f.XUID, f.ID, l.T, l.MS, l.XUID, l.ID)
	}
	p("")
}

// Le codec rend ce qu on lui donne : le temoin de lecture, les quatre champs de chaque medaille, et
// une liste vide se relit nil.
func TestMedaillesCodecAllerRetour(t *testing.T) {
	cas := []struct {
		nom       string
		lues      bool
		medailles []highlightevent.MedalAward
	}{
		{"non lu", false, nil},
		{"lu sans medaille", true, nil},
		{"une medaille", true, []highlightevent.MedalAward{{XUID: 2534026679052315, TimeMS: 70542, TypeHint: 100, MedalType: 0}}},
		{"plusieurs, instant nul et indice maximal", true, []highlightevent.MedalAward{
			{XUID: 2534144863254370, TimeMS: 0, TypeHint: 50, MedalType: 255},
			{XUID: 2534144863254370, TimeMS: 52240, TypeHint: 250, MedalType: 114},
		}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			w := &gwriter{}
			encodeMedailles(w, c.lues, c.medailles)
			r := &greader{b: w.b}
			lues, got := decodeMedailles(r)
			if r.err != nil || r.off != len(r.b) {
				t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
			}
			if lues != c.lues || !slices.Equal(got, c.medailles) {
				t.Errorf("lues %v, medailles %v ; attendu %v, %v", lues, got, c.lues, c.medailles)
			}
		})
	}
}

// ecrireMedaillesDuBuild écrit le fichier des médailles d'un build, par le codec de production, et rend
// sa taille.
func ecrireMedaillesDuBuild(t *testing.T, short8 string, g *FilmFacts) int {
	t.Helper()
	w := &gwriter{}
	encodeMedailles(w, g.MedalsRead, g.Medals)
	var out bytes.Buffer
	zw, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(w.b); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.WriteFile(medaillesPath(short8), out.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture %s : %v", medaillesPath(short8), err)
	}
	return out.Len()
}

// TestGoldenMedaillesReecrireIdentites : LA PORTE DE REECRITURE des xuids des fichiers de médailles. Même
// table et même règle que les fixtures d'entrées (golden_inputs_identites_test.go) : la régénération depuis
// le film rend les xuids du film, cette porte les remplace par les fictifs. Elle échoue toujours en nommant
// ce qu'elle a écrit.
func TestGoldenMedaillesReecrireIdentites(t *testing.T) {
	chemin := os.Getenv(tableIdentitesEnv)
	if chemin == "" {
		t.Skip("reecriture des xuids des medailles : " + tableIdentitesEnv + " non defini")
	}
	table := lireTableIdentites(t, chemin)
	var ecrits []string
	for _, b := range goldenBuilds() {
		g := &FilmFacts{}
		chargerMedaillesDuBuild(t, b.Short8, g)
		n := remplacerXuidsDesEntrees(g, table)
		ecrireMedaillesDuBuild(t, b.Short8, g)
		ecrits = append(ecrits, fmt.Sprintf("%s (%d xuid(s))", medaillesPath(b.Short8), n))
	}
	t.Fatalf("%d fichier(s) de medailles reecrit(s) : %s ; relancer sans %s pour verifier",
		len(ecrits), strings.Join(ecrits, ", "), tableIdentitesEnv)
}

// xuidsDecimauxDUnTexte rend les nombres décimaux de seize chiffres d'un texte : la forme d'un xuid.
var xuidsDecimauxDUnTexte = regexp.MustCompile(`\b\d{16}\b`)

// TestGoldensDAssemblagePortentDesXuidsFictifs : aucun golden d'assemblage n'écrit un xuid hors du registre
// des identifiants fictifs. Les goldens publient des xuids (roster, première et dernière médaille) : un
// fichier régénéré depuis un film dont les identités n'ont pas été remplacées est refusé ici.
func TestGoldensDAssemblagePortentDesXuidsFictifs(t *testing.T) {
	registre := registreIdentitesFictives(t)
	chemins, err := filepath.Glob(filepath.Join(goldenDir, "assembly_*.golden"))
	if err != nil || len(chemins) == 0 {
		t.Fatalf("goldens d'assemblage introuvables (%v)", err)
	}
	for _, chemin := range chemins {
		src, err := os.ReadFile(chemin) //nolint:gosec // chemin issu d'un Glob du dossier de goldens
		if err != nil {
			t.Fatalf("lecture de %s : %v", chemin, err)
		}
		for _, m := range xuidsDecimauxDUnTexte.FindAllString(string(src), -1) {
			x, err := strconv.ParseUint(m, 10, 64)
			if err == nil && !registre[x] {
				t.Errorf("%s : le nombre %s n'est pas au registre des identifiants fictifs", chemin, m)
			}
		}
	}
}
