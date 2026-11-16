package damageeffects

import (
	"math"
	"os"
	"strings"
	"testing"
)

// TestLaTableEstCelleDeLibfilmshell : 468 effets dont 253 d'arme et 411 qui retirent des points, et quelques effets dont
// la cause se lit dans le fichier, pour que le chargement et le fichier ne divergent pas en silence.
func TestLaTableEstCelleDeLibfilmshell(t *testing.T) {
	if n, a, p := Taille(); n != 468 || a != 253 || p != 411 {
		t.Fatalf("table de %d effet(s) dont %d d'arme et %d à points, attendu 468, 253 et 411", n, a, p)
	}
	cas := []struct {
		tag    uint32
		touche bool
	}{
		{0x0000563f, true},  // effet d'arme
		{0x00000024, false}, // le décor
		{0xff768294, false}, // une grenade
		{0xffebe84e, true},  // un autre effet d'arme
		{0x12345678, false}, // hors table
	}
	for _, c := range cas {
		if got := EstUneTouche(c.tag); got != c.touche {
			t.Errorf("effet %08x : touche %v, attendu %v", c.tag, got, c.touche)
		}
	}
}

// TestUnCoupSeRepartitEntreBouclierEtCorps : cas pris sur des effets que les films du corpus nomment. Le bouclier
// encaisse d'abord, au multiplicateur du bouclier ; la fraction du coup qu'il n'arrête pas passe au corps, au
// multiplicateur du corps ; un effet qui ne déborde pas laisse le corps intact ; un effet qui ignore le bouclier le
// traverse ; un coup ne retire jamais plus que ce qui reste.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : faire déborder `1b6370f9` ; passer le multiplicateur du corps au bouclier ;
// oublier la fraction arrêtée (le corps verrait le coup entier) ; ignorer le plafond du corps ; honorer l'ignorance du
// bouclier sans le drapeau.
func TestUnCoupSeRepartitEntreBouclierEtCorps(t *testing.T) {
	effet := func(tag uint32, bas, coeffBouclier, coeffCorps float64) Effet {
		e, ok := Lookup(tag)
		if !ok || !e.MontantFixe() || e.Bas != bas || e.CoeffBouclier != coeffBouclier || e.CoeffCorps != coeffCorps {
			t.Fatalf("effet %08x : %+v, attendu un montant fixe de %v (multiplicateurs %v et %v)", tag, e, bas, coeffBouclier, coeffCorps)
		}
		return e
	}
	rafale := effet(0x17d6bd23, 12, 1, 1)       // l'arme la plus nommée du film de référence
	gachette := effet(0x1b6370f9, 29, 1, 0.3)   // ne déborde pas (drapeau 23)
	lourd := effet(0xf81f1a01, 80, 1, 0.5)      // déborde, le corps encaisse moitié
	diffus := effet(0x5f4f28f1, 12, 0.25, 0.75) // le bouclier n'en voit que le quart
	traversant := Effet{Bas: 40, Haut: [2]float64{40, 40}, CoeffBouclier: 1, CoeffCorps: 1, Drapeaux: drapeauIgnoreLeBouclier}
	cas := []struct {
		nom                 string
		e                   Effet
		bouclier, corps     float64
		wantBouclier, wantC float64
	}{
		{"le bouclier tient le coup", rafale, 140, 90, 12, 0},
		{"sans bouclier, le corps prend tout", rafale, 0, 90, 0, 12},
		{"un effet qui ne déborde pas laisse le corps intact", gachette, 20, 90, 20, 0},
		{"sans bouclier, même un effet qui ne déborde pas atteint le corps", gachette, 0, 90, 0, 8.7},
		{"le débordement est la fraction non arrêtée, au multiplicateur du corps", lourd, 50, 90, 50, 15},
		{"le bouclier ne voit que le quart du coup", diffus, 140, 90, 3, 0},
		{"le débordement d'un coup diffus", diffus, 2, 90, 2, 3},
		{"le corps restant plafonne", rafale, 0, 5, 0, 5},
		{"un effet qui ignore le bouclier le traverse", traversant, 140, 90, 0, 40},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			s, b := c.e.Repartir(c.e.Bas, c.bouclier, c.corps)
			if math.Abs(s-c.wantBouclier) > 1e-9 || math.Abs(b-c.wantC) > 1e-9 {
				t.Errorf("(%v, %v), attendu (%v, %v)", s, b, c.wantBouclier, c.wantC)
			}
		})
	}
}

// TestLeFichierPorteSonCreditEtSaLicence : la table n'est reprise qu'avec son en-tête de crédit, et la licence MIT
// de libfilmshell est livrée à côté.
func TestLeFichierPorteSonCreditEtSaLicence(t *testing.T) {
	if !strings.Contains(brut, "libfilmshell") || !strings.Contains(brut, "MIT") || !strings.Contains(brut, "ad17521") {
		t.Error("l'en-tête de crédit ne cite pas libfilmshell, sa licence et son commit")
	}
	lic, err := os.ReadFile("data/LICENSE-libfilmshell")
	if err != nil || !strings.Contains(string(lic), "MIT License") || !strings.Contains(string(lic), "OpenSpartan") {
		t.Errorf("licence de libfilmshell absente ou altérée : %v", err)
	}
}
