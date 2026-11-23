//go:build research

package grammar

// world_essai_films_research_test.go — LES ESSAIS DU MONDE SUR DE VRAIS FILMS : sur chaque trame
// delta, depuis le monde reel de la marche, les essais de la production (lecture d essai aux positions
// que la localisation essaie et a un echantillon des autres, records de bipedes de la calibration,
// liste recuperee) sont joues avec la copie complete et avec le journal d annulation ; le monde
// ENTIER qui en sort doit etre le meme, champ par champ.
//
// Les films ne sont pas dans le depot. L operateur donne leur correspondance (celle de
// `cmd/film-bench`), le catalogue des bornes de carte, et les noms retenus :
//
//	ESSAIS_CORRESPONDANCE=<correspondance.tsv> ESSAIS_CATALOGUE=<map_quant_bounds.json> \
//	  ESSAIS_FILMS=film06,film09 \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestEssaiDuMondeSurDeVraisFilms$' -v ./internal/grammar/

import (
	"os"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

const (
	positionsEchantillonnees = 6 // positions prises hors de celles que la localisation essaie
	trameDesBipedes          = 16
)

func TestEssaiDuMondeSurDeVraisFilms(t *testing.T) {
	corr, catalogue, noms := os.Getenv("ESSAIS_CORRESPONDANCE"), os.Getenv("ESSAIS_CATALOGUE"), os.Getenv("ESSAIS_FILMS")
	if corr == "" || catalogue == "" || noms == "" {
		t.Skip("ESSAIS_CORRESPONDANCE, ESSAIS_CATALOGUE et ESSAIS_FILMS sont requis")
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue illisible : %v", err)
	}
	lignes := lireLaCorrespondance(t, corr)
	for _, nom := range strings.Split(noms, ",") {
		l, ok := lignes[strings.TrimSpace(nom)]
		if !ok {
			t.Fatalf("film %q absent de la correspondance", nom)
		}
		t.Run(nom, func(t *testing.T) { essaisSurUnFilm(t, ouvrirUnFilm(t, l, cat)) })
	}
}

func essaisSurUnFilm(t *testing.T, fc *FilmContext) {
	t.Helper()
	m, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	var b bilanDEssais
	trames := 0
	m.parcourir(func(tr *trameLue) bool {
		trames++
		pay, w, cfg := tr.paquet.Payload, m.monde, m.cfg
		nb, extra := len(pay)*8, motFacultatifDEnTete(cfg)
		positions := map[int]bool{}
		if tr.debut >= 0 {
			positions[tr.debut] = true
		}
		for _, s := range []int{marchLocateStrict(pay, w, cfg), marchLocateFallback(pay, w, cfg)} {
			if s >= 0 {
				positions[s] = true
			}
		}
		if d, _ := localiserLaListe(pay, w, cfg); d >= 0 {
			positions[d] = true
		}
		for _, c := range candidatsDeTete(pay, nb, w) {
			if c-extra >= 0 {
				positions[c-extra] = true
			}
		}
		for s, n := 2+trames%7, 0; s < nb && n < positionsEchantillonnees; s += 1 + nb/positionsEchantillonnees {
			positions[s] = true
			n++
		}
		for s := range positions {
			b.exiger(t, w, "lecture d essai", corpsDeLectureDEssai(pay, cfg, s),
				func(c *World) any { return lectureDEssai(pay, c, cfg, s) })
			b.exiger(t, w, "liste recuperee", corpsDeListeRecuperee(pay, cfg, s), nil)
		}
		if trames%trameDesBipedes == 0 {
			b.exiger(t, w, "records de bipedes", corpsDeRecordsBipedes(pay, cfg, 3),
				func(c *World) any { return recordsBipedesPropres(pay, c, cfg, 3) })
		}
		return true
	})
	t.Logf("%d trames ; %d essais compares (monde complet, copie complete contre journal), dont %d modifient "+
		"la table avant la restauration", trames, b.essais, b.modifiants)
	if b.essais == 0 || b.modifiants == 0 {
		t.Fatalf("aucun essai qui ecrive : %d essais, %d modifiants", b.essais, b.modifiants)
	}
}
