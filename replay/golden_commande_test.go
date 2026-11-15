package replay

// golden_commande_test.go — QUI COMMANDE CHAQUE CORPS ET QUAND IL EST SUPPRIME, FIGES POUR LES HUIT BUILDS, ET LA FIN DE
// VIE QUE L ASSEMBLAGE EN TIRE.
//
// Les lectures d `unit-control` et les suppressions de bipede (schema des faits 28) voyagent dans le COMPLEMENT de la
// section 1 du fichier de faits, pas dans le blob des entrees. Elles ne portent ni xuid ni gamertag : un index de joueur du
// film, un slot, un instant. Un fichier par build, `commande_<short8>.bin.gz`, encode par le codec de production
// ([encodeCommandeDesBipedes]). Elles ne se posent pas sur les entrees des goldens d assemblage (le nommage des vies par le
// controle les changerait) : seul [TestGoldenFinsDeVie] les emploie, et fige la fin de chaque piste dans
// `fins_de_vie_<short8>.golden`.
//
// REGENERATION des lectures (jamais d'edition a la main), depuis le film, UN FILM PAR PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks go test ./replay/ -run 'TestGoldenCommandeRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu'elle a ecrit ; on relance sans les drapeaux pour verifier. Les references de fin de
// vie se regenerent DEPUIS ces fixtures, sans film : `go test ./replay/ -run TestGoldenFinsDeVie -update-golden-fins-de-vie`.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/internal/decodelock"
)

// updateGoldenFinsDeVie : LA PORTE DES REFERENCES DE FIN DE VIE, et d elles seules.
var updateGoldenFinsDeVie = flag.Bool("update-golden-fins-de-vie", false,
	"reecrire les testdata/fins_de_vie_<short8>.golden, DEPUIS les fixtures figees")

// commandePath : le fichier des lectures de commande et des suppressions d un build.
func commandePath(short8 string) string {
	return filepath.Join(goldenDir, "commande_"+short8+".bin.gz")
}

// finsDeViePath : la reference de fin de vie d un build.
func finsDeViePath(short8 string) string {
	return filepath.Join(goldenDir, "fins_de_vie_"+short8+".golden")
}

// chargerCommandeDuBuild pose sur `g` les lectures de commande et les suppressions figees d un build. ABSENT = ECHEC.
func chargerCommandeDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(commandePath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("lectures de commande du build %s absentes (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenCommandeRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("commande %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("commande %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	decodeCommandeDesBipedes(r, g)
	if r.err != nil || r.off != len(r.b) || !g.UnitControlStats.Scanned {
		t.Fatalf("commande %s : relecture invalide (err %v, %d/%d octets lus, lues %v)", short8, r.err, r.off,
			len(r.b), g.UnitControlStats.Scanned)
	}
}

// TestGoldenCommandeRegenerate : LA SEULE PORTE D ECRITURE des lectures de commande figees (cf. l en-tete).
func TestGoldenCommandeRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des lectures de commande : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des lectures de commande : " + miniFilmCacheEnv + " non defini")
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant(t)
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			decodelock.PrendreEnAttendant(context.Background(), t, cacheDuDepot(t), "commande-regenerate", b.Short8, 6*time.Hour)
			g, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("decodage de %s : %v", b.Short8, err)
			}
			w := &gwriter{}
			encodeCommandeDesBipedes(w, g)
			var out bytes.Buffer
			zw := gzip.NewWriter(&out)
			if _, err := zw.Write(w.b); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := zw.Close(); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := os.WriteFile(commandePath(b.Short8), out.Bytes(), 0o600); err != nil {
				t.Fatalf("ecriture %s : %v", commandePath(b.Short8), err)
			}
			t.Fatalf("REECRIT %s (%d lectures, %d suppressions, %d octets) — relancer sans -update pour verifier",
				commandePath(b.Short8), len(g.UnitControls), len(g.BipedDeletions), out.Len())
		})
	}
}

// renderFinsDeVie rend, une ligne par piste, la fin lue de son corps, puis la couverture.
func renderFinsDeVie(doc ReplayDocument) string {
	var sb strings.Builder
	sb.WriteString("# fin de vie lue par piste : slot, ouverture, fin, endReason, releaseFrame. Figee depuis\n")
	sb.WriteString("# testdata/inputs_<short8>.bin.gz et testdata/commande_<short8>.bin.gz, sans film.\n")
	for _, tr := range doc.Tracks {
		lacher := "-"
		if tr.ReleaseFrame != nil {
			lacher = fmt.Sprint(*tr.ReleaseFrame)
		}
		raison := tr.EndReason
		if raison == "" {
			raison = "-"
		}
		fmt.Fprintf(&sb, "%d\t%d\t%d\t%s\t%s\n", tr.Slot, tr.StartFrame, tr.EndFrame, raison, lacher)
	}
	if l := doc.Coverage.Tracks.LifeEnds; l != nil {
		fmt.Fprintf(&sb, "# lifeEnds released=%d deleted=%d retaken=%d unread=%d open=%d deletionsUnproven=%d\n",
			l.Released, l.Deleted, l.Retaken, l.Unread, l.Open, l.DeletionsUnproven)
	}
	return sb.String()
}

// TestGoldenFinsDeVie : la fin de chaque piste, lue dans la commande et la suppression des corps, est-elle celle figee, et
// les comptes de la couverture somment-ils les pistes ?
func TestGoldenFinsDeVie(t *testing.T) {
	var ecrits []string
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			chargerCommandeDuBuild(t, b.Short8, g)
			doc := assemblerGoldenBuild(t, b, g, entry)
			controlerLesComptesDeFinDeVie(t, doc)
			got := renderFinsDeVie(doc)
			if *updateGoldenFinsDeVie {
				if err := os.WriteFile(finsDeViePath(b.Short8), []byte(got), 0o600); err != nil {
					t.Fatalf("ecriture %s : %v", finsDeViePath(b.Short8), err)
				}
				ecrits = append(ecrits, finsDeViePath(b.Short8))
				return
			}
			want, err := os.ReadFile(finsDeViePath(b.Short8)) //nolint:gosec // chemin construit depuis la table
			if err != nil {
				t.Fatalf("reference absente (%s) : %v — regenerer avec -update-golden-fins-de-vie", finsDeViePath(b.Short8), err)
			}
			if string(want) != got {
				t.Errorf("la fin de vie a change par rapport a %s.\n%s", finsDeViePath(b.Short8),
					premierEcartAssembly(string(want), got))
			}
		})
	}
	if len(ecrits) > 0 {
		t.Fatalf("%d reference(s) reecrite(s) : %s ; relancer sans -update-golden-fins-de-vie pour verifier",
			len(ecrits), strings.Join(ecrits, ", "))
	}
}

// controlerLesComptesDeFinDeVie : les cinq comptes somment les pistes ; `deleted` et `released` portent la forme voulue.
func controlerLesComptesDeFinDeVie(t *testing.T, doc ReplayDocument) {
	t.Helper()
	l := doc.Coverage.Tracks.LifeEnds
	if l == nil {
		t.Fatal("coverage.tracks.lifeEnds absent alors que la commande est lue")
	}
	if somme := l.Released + l.Deleted + l.Retaken + l.Unread + l.Open; somme != len(doc.Tracks) {
		t.Errorf("les comptes somment %d piste(s) pour %d publiee(s) : %+v", somme, len(doc.Tracks), *l)
	}
	var lachers, supprimes int
	for _, tr := range doc.Tracks {
		switch tr.EndReason {
		case EndReasonReleased:
			lachers++
			if tr.ReleaseFrame == nil {
				t.Errorf("piste du slot %d : released sans releaseFrame", tr.Slot)
			}
		case EndReasonDeleted:
			supprimes++
			if tr.ReleaseFrame != nil {
				t.Errorf("piste du slot %d : deleted avec releaseFrame", tr.Slot)
			}
		case "":
		default:
			t.Errorf("piste du slot %d : endReason %q inconnu", tr.Slot, tr.EndReason)
		}
	}
	if lachers != l.Released || supprimes != l.Deleted+l.Retaken {
		t.Errorf("pistes released %d / deleted %d, couverture %+v", lachers, supprimes, *l)
	}
}

// TestGoldenCommandeNePorteAucunIdentifiant : les fixtures de commande ne portent ni xuid ni gamertag, par construction
// (un slot, un instant, un index de joueur du film) ; le test refuse une valeur hors de ces domaines, que le depot public
// ne doit pas recevoir.
func TestGoldenCommandeNePorteAucunIdentifiant(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g := &FilmFacts{}
			chargerCommandeDuBuild(t, b.Short8, g)
			for _, c := range g.UnitControls {
				if c.Slot >= 1<<16 || c.Controller < -1 || c.Controller > 31 {
					t.Errorf("lecture %+v hors des domaines d un slot et d un index de joueur du film", c)
				}
			}
			for _, d := range g.BipedDeletions {
				if d.Slot >= 1<<16 {
					t.Errorf("suppression %+v hors du domaine d un slot", d)
				}
			}
		})
	}
}

// TestGoldenFinsDeVie_AjouteSeulementSesChamps : le document assemble avec la commande lue, moins `tracks[].endReason`,
// `tracks[].releaseFrame` et `coverage.tracks.lifeEnds`, est le document assemble sans elle, octet pour octet.
func TestGoldenFinsDeVie_AjouteSeulementSesChamps(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			avec, entry := chargerGoldenBuild(t, b)
			chargerCommandeDuBuild(t, b.Short8, avec)
			sans, _ := chargerGoldenBuild(t, b)
			docAvec := assemblerGoldenBuild(t, b, avec, entry)
			docSans := assemblerGoldenBuild(t, b, sans, entry)
			docAvec.Tracks = slices.Clone(docAvec.Tracks)
			for i := range docAvec.Tracks {
				docAvec.Tracks[i].EndReason, docAvec.Tracks[i].ReleaseFrame = "", nil
			}
			docAvec.Coverage.Tracks.LifeEnds = nil
			a, errA := json.Marshal(docAvec)
			s, errS := json.Marshal(docSans)
			if errA != nil || errS != nil {
				t.Fatalf("serialisation : %v / %v", errA, errS)
			}
			if !bytes.Equal(a, s) {
				t.Errorf("la lecture de la commande change autre chose que ses champs : %d octets contre %d", len(a), len(s))
			}
		})
	}
}
