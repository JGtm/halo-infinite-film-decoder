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

// matchMSOfFrameAutorise nomme les SEULS fichiers de production qui posent le debut d'une frame sur l'horloge du match,
// avec la raison. `matchMSOfFrame` rend le debut de la frame, un instant qui bouge avec le pas : il ne sert qu'a une BORNE
// de l'axe ou a la conversion d'un intervalle deja publie en frames, jamais a decider d'un fait ni d'un ordre entre deux
// evenements, qui se posent par `matchMSOfUS`.
var matchMSOfFrameAutorise = map[string]string{
	"flag_carries.go":     "borne de fin de partie des portages encore ouverts : aucun fait ne s'y compare, un portage non ferme reste ouvert",
	"vip_crown.go":        "borne de fin de partie d'une couronne encore portee : meme usage, la fin de l'axe",
	"porteurs_au_sync.go": "conversion en millisecondes des intervalles de port DEJA publies en frames (les portages du document)",
}

// TestInstantExactNEstPasArrondiALaGrille : un instant du film se pose sur l'horloge du match par `matchMSOfUS`, a la
// milliseconde exacte. `matchMSOfFrame` n'est appele, hors de match_clock.go et des tests, que par les fichiers de
// [matchMSOfFrameAutorise] : toute autre forme — sur une ligne (`matchMSOfFrame(frameOf(..))`) ou sur deux
// (`f := frameOf(..)` puis `matchMSOfFrame(f)`, celle de l'ancien code) — echoue ici.
func TestInstantExactNEstPasArrondiALaGrille(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || filepath.Ext(nom) != ".go" || strings.HasSuffix(nom, "_test.go") || nom == "match_clock.go" {
			continue
		}
		src, err := os.ReadFile(nom) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		if _, ok := matchMSOfFrameAutorise[nom]; !ok && strings.Contains(string(src), ".matchMSOfFrame(") {
			t.Errorf("%s appelle matchMSOfFrame : un instant exact se pose par matchMSOfUS, pas arrondi au debut de sa frame ; "+
				"une borne d'axe se declare dans matchMSOfFrameAutorise avec sa raison", nom)
		}
	}
	for nom := range matchMSOfFrameAutorise {
		src, err := os.ReadFile(nom) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil || !strings.Contains(string(src), ".matchMSOfFrame(") {
			t.Errorf("%s est autorise a appeler matchMSOfFrame et ne le fait plus (ou n'existe plus) : retirer l'entree", nom)
		}
	}
}
