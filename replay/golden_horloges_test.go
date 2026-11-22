package replay

// golden_horloges_test.go — L'HORLOGE DE MANCHE ET LES COMPTES A REBOURS DES HUIT BUILDS, FIGES A COTE DE LEURS ENTREES.
//
// Les lectures de l'horloge (`FilmInputs.Horloges` : lectures, comptes a rebours et statistiques de la marche) datent le
// coup d'envoi et alimentent `roundClock` et `respawnCountdowns`. Elles voyagent dans un fichier par build,
// `horloges_<short8>.bin.gz`, encode par le codec de production ([encodeHorloges]), pose sur les entrees relues
// ([chargerGoldenBuild], [loadGoldenInputs]) ; la section du blob des entrees, vide dans les fixtures figes avant elle, est
// ecrasee. Elles ne portent aucun xuid : des horodatages, deux quanta et un etat pour l'horloge ; un slot, un index de
// joueur du film et des secondes pour les comptes (cf. [TestGoldenHorlogesNePorteAucunIdentifiant]).
//
// REGENERATION (jamais d'edition a la main), depuis le film, UN FILM PAR PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./replay/ -run 'TestGoldenHorlogesRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu'elle a ecrit ; on relance sans les drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// horlogesPath : le fichier des horloges d'un build.
func horlogesPath(short8 string) string {
	return filepath.Join(goldenDir, "horloges_"+short8+".bin.gz")
}

// chargerHorlogesDuBuild pose sur `g` les horloges figees d'un build. ABSENT = ECHEC : les huit builds les portent, et un
// build qui les perdrait publierait un coup d'envoi absent.
func chargerHorlogesDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(horlogesPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("horloges du build %s absentes (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenHorlogesRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("horloges %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("horloges %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	g.Horloges = decodeHorloges(r)
	if r.err != nil || r.off != len(r.b) || g.Horloges.Stats.Trames == 0 {
		t.Fatalf("horloges %s : relecture invalide (err %v, %d/%d octets lus, %d trame(s))", short8, r.err,
			r.off, len(r.b), g.Horloges.Stats.Trames)
	}
}

// ecrireHorlogesDuBuild ecrit le fichier des horloges d'un build, par le codec de production, et rend sa taille.
func ecrireHorlogesDuBuild(t *testing.T, short8 string, g *FilmFacts) int {
	t.Helper()
	w := &gwriter{}
	encodeHorloges(w, g.Horloges)
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
	if err := os.WriteFile(horlogesPath(short8), out.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture %s : %v", horlogesPath(short8), err)
	}
	return out.Len()
}

// TestGoldenHorlogesRegenerate : LA SEULE PORTE D'ECRITURE des horloges figees (cf. l'en-tete).
func TestGoldenHorlogesRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des horloges des builds : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des horloges des builds : " + miniFilmCacheEnv + " non defini")
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
			n := ecrireHorlogesDuBuild(t, b.Short8, g)
			t.Fatalf("REECRIT %s (%d lecture(s) d'horloge, %d de compte a rebours, %d octets) — relancer sans -update pour verifier",
				horlogesPath(b.Short8), len(g.Horloges.Horloges), len(g.Horloges.Comptes), n)
		})
	}
}

// TestGoldenHorlogesNePorteAucunIdentifiant : les fixtures d'horloge ne portent ni xuid ni gamertag, par construction (une
// lecture est un horodatage, deux quanta de 16 bits et un etat de 5 bits ; un compte a rebours un slot, un index de joueur du
// film et deux secondes de 10 bits) ; le test refuse une valeur hors de ces domaines, que le depot public ne doit pas
// recevoir, et verifie que le fichier ne contient aucune suite de seize chiffres.
func TestGoldenHorlogesNePorteAucunIdentifiant(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g := &FilmFacts{}
			chargerHorlogesDuBuild(t, b.Short8, g)
			for _, h := range g.Horloges.Horloges {
				if h.Etat >= 1<<5 {
					t.Errorf("lecture %+v hors du domaine d un etat de 5 bits", h)
				}
			}
			for _, c := range g.Horloges.Comptes {
				if c.Slot >= 1<<16 || c.Joueur < -1 || c.Joueur > 31 || c.Reste >= 1<<10 || c.Total >= 1<<10 {
					t.Errorf("compte %+v hors des domaines d un slot, d un index de joueur du film et de secondes de 10 bits", c)
				}
			}
			raw, err := os.ReadFile(horlogesPath(b.Short8)) //nolint:gosec // chemin construit depuis la table des builds
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}
			zr, err := gzip.NewReader(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("gzip : %v", err)
			}
			defer func() { _ = zr.Close() }()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(zr); err != nil {
				t.Fatalf("decompression : %v", err)
			}
			if run := suiteDeChiffres(buf.Bytes()); run >= 16 {
				t.Errorf("le fichier porte une suite de %d chiffres ASCII : un identifiant de joueur ?", run)
			}
		})
	}
}

// suiteDeChiffres rend la plus longue suite de chiffres ASCII consecutifs de `b`.
func suiteDeChiffres(b []byte) int {
	best, cur := 0, 0
	for _, c := range b {
		if c >= '0' && c <= '9' {
			cur++
			best = max(best, cur)
			continue
		}
		cur = 0
	}
	return best
}
