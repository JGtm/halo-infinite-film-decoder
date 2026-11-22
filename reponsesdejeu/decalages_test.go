package reponsesdejeu

import "testing"

var scriptsGrandeBataille = []string{"bigteambattle.lua", "btbslayerevents.lua"}

// TestLeDecalageDuBuildDeclareLitLeRangUnCranPlusLoin : sur le build et le groupe déclarés, le rang du film
// désigne la réponse du rang suivant (un cran sur 179 à 263, deux sur 447 à 460) ; sur un autre build, ou sans
// build, le même rang se lit tel quel ; un rang décalé que la table ne contient pas est hors de l'arbre.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : deux crans au lieu d'un sur le premier intervalle ; le décalage
// appliqué à tous les builds ; le décalage appliqué à un autre groupe ; une borne de l'intervalle déplacée.
func TestLeDecalageDuBuildDeclareLitLeRangUnCranPlusLoin(t *testing.T) {
	cas := []struct {
		nom      string
		build    string
		rang     uint32
		nomRep   string
		points   int
		nature   Nature
		decalage uint32
	}{
		{"aide par IEM", "HI_1_11_0", 227, "empAssistPersonalScoreResponse", 50, NatureScore, 1},
		{"Ghost détruit", "HI_1_11_0", 231, "ghostDestroyResponse", 50, NatureScore, 1},
		{"Wraith détruit", "HI_1_11_0", 240, "wraithDestroyResponse", 100, NatureScore, 1},
		{"Banshee détourné", "HI_1_11_0", 250, "bansheeHijackResponse", 25, NatureScore, 1},
		{"hors de l'arbre", "HI_1_11_0", 263, "", 0, NatureHorsArbre, 1},
		{"première borne", "HI_1_11_0", 179, "runningRiotMythicTeam", 0, NatureSansScore, 1},
		{"avant la première borne", "HI_1_11_0", 178, "killingFrenzyResponse", 0, NatureSansScore, 0},
		{"après la dernière borne du premier intervalle", "HI_1_11_0", 264, "", 0, NatureHorsArbre, 0},
		{"deux crans", "HI_1_11_0", 447, "remoteDetonationMedalMythicEnemy", 0, NatureSansScore, 2},
		{"deux crans, dernière borne", "HI_1_11_0", 460, "deathRaceResponse", 0, NatureSansScore, 2},
		{"entre les intervalles", "HI_1_11_0", 300, "", 0, NatureHorsArbre, 0},
		{"build 1.12 : Falcon détruit tel quel", "HI_1_12_0", 231, "falconDestroyResponse", 75, NatureScore, 0},
		{"build 1.12 : médaille telle quelle", "HI_1_12_0", 227, "lawnmowerResponse", 0, NatureSansScore, 0},
		{"build 1.12 : Wraith détourné tel quel", "HI_1_12_0", 263, "wraithHijackResponse", 25, NatureScore, 0},
		{"build 1.10 : non déclaré, tel quel", "HI_1_10_0", 231, "falconDestroyResponse", 75, NatureScore, 0},
		{"sans build : tel quel", "", 227, "lawnmowerResponse", 0, NatureSansScore, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			g, ok := GroupeDuFilm(scriptsGrandeBataille, c.build)
			if !ok {
				t.Fatal("groupe de la Grande bataille introuvable")
			}
			if d := g.Decalage(c.rang); d != c.decalage {
				t.Errorf("décalage du rang %d : %d, attendu %d", c.rang, d, c.decalage)
			}
			r, n := g.Reponse(c.rang, 0, false)
			if n != c.nature || r.Nom != c.nomRep || r.Points != c.points {
				t.Errorf("rang %d : %q %d points (%v), attendu %q %d points (%v)", c.rang, r.Nom, r.Points, n, c.nomRep, c.points, c.nature)
			}
		})
	}
}

// TestLeTexteConfirmeLeRangDecale : un message à mot texte se confirme par la réponse du rang décalé, et par aucune
// autre : le même message lu sur un build sans décalage n'est pas confirmé.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : lire le texte contre le rang du film ; décaler de deux crans le premier
// intervalle ; décaler tous les builds.
func TestLeTexteConfirmeLeRangDecale(t *testing.T) {
	cas := []struct {
		nom        string
		build      string
		rang, mot  uint32
		attendu    Nature
		attenduNom string
	}{
		{"179 : texte de la ligne du fil du rang 180", "HI_1_11_0", 179, 0x5f6f6f9a, NatureSansScore, "runningRiotMythicTeam"},
		{"449 : texte de la ligne du fil du rang 451", "HI_1_11_0", 449, 0xfff8b704, NatureSansScore, "ninjaMedalMythicTeamResponse"},
		{"179 : le texte du rang 179 n'est plus celui du rang lu", "HI_1_11_0", 179, 0xea7812d1, NatureNonConfirmee, ""},
		{"179 sans décalage : le texte décalé est refusé", "HI_1_12_0", 179, 0x5f6f6f9a, NatureNonConfirmee, ""},
		{"449 sans décalage : le texte décalé est refusé", "HI_1_12_0", 449, 0xfff8b704, NatureNonConfirmee, ""},
		{"179 sans décalage : son propre texte", "HI_1_12_0", 179, 0xea7812d1, NatureSansScore, "runningRiotResponse"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			g, ok := GroupeDuFilm(scriptsGrandeBataille, c.build)
			if !ok {
				t.Fatal("groupe de la Grande bataille introuvable")
			}
			if r, n := g.Reponse(c.rang, c.mot, true); n != c.attendu || r.Nom != c.attenduNom {
				t.Errorf("rang %d : %q (%v), attendu %q (%v)", c.rang, r.Nom, n, c.attenduNom, c.attendu)
			}
		})
	}
}

// TestLeDecalageNeVautQuePourSonGroupe : le même build lu sur un autre groupe de scripts ne décale aucun rang.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : appliquer les décalages du build sans regarder le groupe.
func TestLeDecalageNeVautQuePourSonGroupe(t *testing.T) {
	for _, scripts := range [][]string{{"slayerevents.lua"}, {"bigteambattle.lua", "btbstockpileevents.lua"}} {
		g, ok := GroupeDuFilm(scripts, "HI_1_11_0")
		if !ok {
			t.Fatalf("groupe de %v introuvable", scripts)
		}
		for _, rang := range []uint32{179, 227, 231, 263, 447, 460} {
			if d := g.Decalage(rang); d != 0 {
				t.Errorf("%s, rang %d : décalage %d, aucun attendu", g.Nom(), rang, d)
			}
		}
	}
}

// TestLesDecalagesDeclaresSontCoherents : chaque ligne déclarée désigne un groupe de la table, un intervalle
// non vide et un cran non nul, et deux lignes du même build et du même groupe ne se recouvrent pas.
func TestLesDecalagesDeclaresSontCoherents(t *testing.T) {
	connus := map[string]bool{}
	for _, g := range groupes {
		connus[g.nom] = true
	}
	for i, d := range decalages {
		if !connus[d.groupe] || d.build == "" || d.premier > d.dernier || d.cran == 0 {
			t.Errorf("ligne %d mal formée : %+v", i, d)
		}
		for _, e := range decalages[:i] {
			if e.build == d.build && e.groupe == d.groupe && d.premier <= e.dernier && e.premier <= d.dernier {
				t.Errorf("les lignes %+v et %+v se recouvrent", e, d)
			}
		}
	}
}
