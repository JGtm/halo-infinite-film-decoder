package replay

// match_clock_guard_test.go — GARDE-RAIL (CLAUDE.md n° 6) : la conversion match <-> frames
// n'a qu'UNE implementation, celle de match_clock.go.
//
// HISTOIRE DU DEFAUT : la methode `frameOfMatchMS` a ete recopiee trois fois (drapeau, crane,
// couronne) avant d'etre centralisee a l'arrivee du quatrieme consommateur (portage de la
// bombe, schema 34). Une factorisation sans garde-rail re-diverge — lecon du predicat bot
// (8 -> 36 copies apres centralisation). Ce test grep rend la re-divergence ROUGE : toute
// nouvelle declaration d'une methode de conversion hors de match_clock.go echoue ici.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMatchClockSingleImplementation echoue si `frameOfMatchMS`, `matchMSOfFrame`, `matchMSOfUS` ou `framesOfMS`
// est declare ailleurs que dans match_clock.go.
func TestMatchClockSingleImplementation(t *testing.T) {
	decl := regexp.MustCompile(`func \([^)]+\) (frameOfMatchMS|matchMSOfFrame|matchMSOfUS|framesOfMS)\(`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" || e.Name() == "match_clock.go" {
			continue
		}
		src, err := os.ReadFile(e.Name()) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		if m := decl.Find(src); m != nil {
			t.Errorf("%s declare %q : la conversion match <-> frames n'a qu'une implementation "+
				"(match_clock.go) — utiliser ou embarquer matchClock au lieu de recopier", e.Name(), m)
		}
	}
}

// TestInstantExactNEstPasArrondiALaGrille : un instant du film se pose sur l'horloge du match par
// `matchMSOfUS`, a la milliseconde exacte. `matchMSOfFrame(frameOf(us, ..))` rend le DEBUT de la frame qui
// contient l'instant : ecrit trois fois dans le calque drapeau, il y rendait un instant qui bouge avec le pas
// et change le verdict d'un ordre entre deux evenements voisins. Le motif est interdit hors des tests.
func TestInstantExactNEstPasArrondiALaGrille(t *testing.T) {
	motif := regexp.MustCompile(`matchMSOfFrame\(\s*frameOf\(`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(e.Name()) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		if m := motif.Find(src); m != nil {
			t.Errorf("%s ecrit %q : un instant exact se pose par matchMSOfUS, pas arrondi au debut de sa frame", e.Name(), m)
		}
	}
}

// TestToleranceDuPointDuPorteurEstUneDuree : `pointOfXUIDAt` recoit une tolerance en frames tiree de
// `flagCarrierPointToleranceMS` par `framesOfMS` ; un entier litteral (« une image ») redonne une tolerance
// de 100 ms ou de 33 ms selon le pas. Le garde-rail interdit tout appel a tolerance litterale.
func TestToleranceDuPointDuPorteurEstUneDuree(t *testing.T) {
	motif := regexp.MustCompile(`pointOfXUIDAt\([^()]*,\s*\d+\)`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		src, err := os.ReadFile(e.Name()) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		if m := motif.Find(src); m != nil {
			t.Errorf("%s appelle %q : la tolerance est une duree (flagCarrierPointToleranceMS), pas un nombre de frames", e.Name(), m)
		}
	}
}
