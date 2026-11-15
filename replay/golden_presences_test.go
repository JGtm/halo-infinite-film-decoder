package replay

// golden_presences_test.go — LES PLAGES DE LISTAGE DES HUIT BUILDS, FIGEES A COTE DE LEURS ENTREES.
//
// Les plages de listage des humains (paquets de type 8, schema des faits 24) voyagent dans le COMPLEMENT de la
// section 1 du fichier de faits, pas dans le blob des entrees : les `inputs_<short8>.bin.gz` sont relus sans elles.
// Sans leur fichier, les goldens d assemblage dateraient la presence des humains par les seules entites et les vies.
// Meme forme que les entites `ti=9` (golden_entites_test.go) : un fichier par build, `presences_<short8>.bin.gz`,
// encode par le codec de production ([encodeListagesDeRoster]), pose sur les entrees relues.
//
// REGENERATION (jamais d'edition a la main), depuis le film, UN FILM PAR PROCESSUS. Les films portent de VRAIS xuids : la
// table `original<TAB>fictif` du harnais (REPLAY_IDENTITES_TABLE, hors depot) les remplace avant l'ecriture, et
// [TestGoldenPresencesPorteDesXuidsFictifs] refuse une fixture dont un xuid n'est pas au registre des fictifs :
//
//	REPLAY_IDENTITES_TABLE=<table> REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./replay/ -run 'TestGoldenPresencesRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu'elle a ecrit ; on relance sans les drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/internal/decodelock"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// presencesPath : le fichier des plages de listage d'un build.
func presencesPath(short8 string) string {
	return filepath.Join(goldenDir, "presences_"+short8+".bin.gz")
}

// chargerPresencesDuBuild pose sur `g` les plages de listage figees d'un build. ABSENT = ECHEC : un build qui
// les perdrait retomberait en silence sur la presence par les entites.
func chargerPresencesDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(presencesPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("plages de listage du build %s absentes (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenPresencesRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("plages %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("plages %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	g.RosterListings, g.RosterListingStats = decodeListagesDeRoster(r)
	if r.err != nil || r.off != len(r.b) || !g.RosterListingStats.Scanned {
		t.Fatalf("plages %s : relecture invalide (err %v, %d/%d octets lus, lues %v)", short8, r.err, r.off,
			len(r.b), g.RosterListingStats.Scanned)
	}
}

// TestGoldenPresencesRegenerate : LA SEULE PORTE D'ECRITURE des plages figees (cf. l'en-tete).
func TestGoldenPresencesRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des plages de listage : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des plages de listage : " + miniFilmCacheEnv + " non defini")
	}
	tableChemin := os.Getenv(tableIdentitesEnv)
	if tableChemin == "" {
		t.Fatal("les films portent de vrais xuids : " + tableIdentitesEnv + " doit nommer la table original<TAB>fictif")
	}
	table := lireTableIdentites(t, tableChemin)
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant(t)
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			decodelock.PrendreEnAttendant(context.Background(), t, cacheDuDepot(t), "presences-regenerate", b.Short8, 6*time.Hour)
			g, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("decodage de %s : %v", b.Short8, err)
			}
			if n := remplacerXuidsDesListages(g.RosterListings, table); n != len(g.RosterListings) {
				t.Fatalf("%d plage(s) sur %d portent un xuid que la table des identites ne connait pas", len(g.RosterListings)-n, len(g.RosterListings))
			}
			w := &gwriter{}
			encodeListagesDeRoster(w, g.RosterListings, g.RosterListingStats)
			var out bytes.Buffer
			zw := gzip.NewWriter(&out)
			if _, err := zw.Write(w.b); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := zw.Close(); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := os.WriteFile(presencesPath(b.Short8), out.Bytes(), 0o600); err != nil {
				t.Fatalf("ecriture %s : %v", presencesPath(b.Short8), err)
			}
			t.Fatalf("REECRIT %s (%d plages, %d octets) — relancer sans -update pour verifier",
				presencesPath(b.Short8), len(g.RosterListings), out.Len())
		})
	}
}

// remplacerXuidsDesListages applique la table `original<TAB>fictif` aux plages et rend le nombre de plages
// dont le xuid a ete remplace ; un xuid que la table ne connait pas reste tel quel (le garde-rail le refusera).
func remplacerXuidsDesListages(ls []types.RosterListing, table map[uint64]uint64) int {
	n := 0
	for i := range ls {
		if f, ok := table[ls[i].XUID]; ok {
			ls[i].XUID = f
			n++
		}
	}
	return n
}

// TestGoldenPresencesPorteDesXuidsFictifs : aucune fixture de plages ne porte un xuid hors du registre des
// identifiants fictifs `testdata/IDENTITES.tsv` (le depot est public : jamais un vrai xuid).
func TestGoldenPresencesPorteDesXuidsFictifs(t *testing.T) {
	registre := registreIdentitesFictives(t)
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g := &FilmFacts{}
			chargerPresencesDuBuild(t, b.Short8, g)
			for _, l := range g.RosterListings {
				if !registre[l.XUID] {
					t.Errorf("xuid %d hors du registre des identifiants fictifs", l.XUID)
				}
			}
		})
	}
}

// TestRemplacerXuidsDesListages : la table remplace les xuids connus, compte les remplacements et laisse
// intacts les autres champs et les xuids inconnus.
func TestRemplacerXuidsDesListages(t *testing.T) {
	ls := []types.RosterListing{{XUID: 1, Slot: 3, FromUS: 10, ToUS: 20}, {XUID: 2, Slot: 4, FromUS: 30}}
	n := remplacerXuidsDesListages(ls, map[uint64]uint64{1: 900})
	if n != 1 || ls[0] != (types.RosterListing{XUID: 900, Slot: 3, FromUS: 10, ToUS: 20}) || ls[1].XUID != 2 {
		t.Fatalf("plages %+v, %d remplacement(s)", ls, n)
	}
}
