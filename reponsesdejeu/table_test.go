package reponsesdejeu

import (
	"os"
	"strings"
	"testing"
)

// TestLaTableEstCelleDeLibfilmshell : le compte de la table reprise (423 lignes, 11 groupes) et quelques
// réponses dont le rang et les points se lisent dans le fichier, pour que le chargement et le fichier ne
// divergent pas en silence.
func TestLaTableEstCelleDeLibfilmshell(t *testing.T) {
	if l, g := Taille(); l != 423 || g != 11 {
		t.Fatalf("table de %d ligne(s) et %d groupe(s), attendu 423 et 11", l, g)
	}
	cas := []struct {
		scripts []string
		rang    uint32
		nom     string
		points  int
	}{
		{[]string{"slayerevents.lua"}, 78, "killPersonalScore", 100},
		{[]string{"slayerevents.lua"}, 49, "assistPersonalScore", 50},
		{[]string{"slayerevents.lua"}, 59, "markAssistPersonalScore", 10},
		{[]string{"slayerevents.lua"}, 31, "playerSuicidePersonalScore", -100},
		{[]string{"arenaflagevents.lua", "flageventstubs.lua"}, 0, "", 0},
	}
	for _, c := range cas {
		g, ok := GroupeDesScripts(c.scripts)
		if !ok {
			t.Fatalf("groupe de %v introuvable", c.scripts)
		}
		r, ok := g.Reponse(c.rang, 0, false)
		if c.nom == "" {
			if ok {
				t.Errorf("%s rang %d : réponse %+v, aucune attendue", g.Nom(), c.rang, r)
			}
			continue
		}
		if !ok || r.Nom != c.nom || r.Points != c.points {
			t.Errorf("%s rang %d : %+v, attendu %s (%d)", g.Nom(), c.rang, r, c.nom, c.points)
		}
	}
}

// TestLeGroupeSeLitAuxFichiersDuFilm : le groupe est celui dont les fichiers sont exactement ceux, parmi les
// fichiers des groupes, que le film déclare ; les autres scripts du film n'y changent rien ; un fichier d'un
// groupe sans les autres, ou deux modes à la fois, n'en désigne aucun.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : accepter un sous-ensemble des fichiers ; accepter un
// sur-ensemble ; tenir compte des scripts hors table.
func TestLeGroupeSeLitAuxFichiersDuFilm(t *testing.T) {
	cas := []struct {
		nom     string
		scripts []string
		groupe  string
	}{
		{"arène : un fichier", []string{"slayerevents.lua"}, "slayerevents.lua"},
		{"arène : deux fichiers et d'autres scripts", []string{"engine.lua", "flageventstubs.lua", "arenaflagevents.lua", "zzz.lua"},
			"arenaflagevents.lua+flageventstubs.lua"},
		{"un seul des deux fichiers du groupe", []string{"flageventstubs.lua"}, ""},
		{"deux modes à la fois", []string{"slayerevents.lua", "oddballevents.lua"}, ""},
		{"aucun fichier connu", []string{"engine.lua"}, ""},
		{"aucun script", nil, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			g, ok := GroupeDesScripts(c.scripts)
			if c.groupe == "" {
				if ok {
					t.Fatalf("groupe %q, aucun attendu", g.Nom())
				}
				return
			}
			if !ok || g.Nom() != c.groupe {
				t.Fatalf("groupe %v, attendu %q", ok, c.groupe)
			}
		})
	}
}

// TestUneReponseSAttribueAuTexteQuiLaConfirme : un rang à texte propre n'est attribué que si le message porte
// ce texte (ou aucun) ; le texte d'une autre réponse la refuse ; un rang incertain exige le texte.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ignorer le texte ; attribuer un rang incertain sans texte ; refuser
// un message sans texte pour un rang certain.
func TestUneReponseSAttribueAuTexteQuiLaConfirme(t *testing.T) {
	slayer, _ := GroupeDesScripts([]string{"slayerevents.lua"})
	colline, ok := GroupeDesScripts([]string{"areacaptureeventstubs.lua", "fbastionevents.lua", "firefightevents.lua"})
	if !ok {
		t.Fatal("groupe de Firefight introuvable")
	}
	const motDuFrag = 0x3d097626
	cas := []struct {
		nom     string
		g       *Groupe
		rang    uint32
		texte   uint32
		aTexte  bool
		attendu bool
	}{
		{"rang certain, texte de la réponse", slayer, 78, motDuFrag, true, true},
		{"rang certain, sans texte", slayer, 78, 0, false, true},
		{"rang certain, texte d'une autre réponse", slayer, 78, 0x365be6b1, true, false},
		{"rang sans texte propre, texte quelconque", slayer, 49, 0x1234, true, true},
		{"rang incertain, sans texte", colline, 828, 0, false, false},
		{"rang inconnu", slayer, 100000, 0, false, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, ok := c.g.Reponse(c.rang, c.texte, c.aTexte); ok != c.attendu {
				t.Errorf("attribuée : %v, attendu %v", ok, c.attendu)
			}
		})
	}
}

// TestLeFichierPorteSonCreditEtSaLicence : la table n'est reprise qu'avec son en-tête de crédit, et la
// licence MIT de libfilmshell est livrée à côté.
func TestLeFichierPorteSonCreditEtSaLicence(t *testing.T) {
	if !strings.Contains(brut, "libfilmshell") || !strings.Contains(brut, "MIT") || !strings.Contains(brut, "ad17521") {
		t.Error("l'en-tête de crédit ne cite pas libfilmshell, sa licence et son commit")
	}
	lic, err := os.ReadFile("data/LICENSE-libfilmshell")
	if err != nil || !strings.Contains(string(lic), "MIT License") || !strings.Contains(string(lic), "OpenSpartan") {
		t.Errorf("licence de libfilmshell absente ou altérée : %v", err)
	}
}
