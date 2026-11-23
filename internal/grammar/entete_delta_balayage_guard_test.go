package grammar

// entete_delta_balayage_guard_test.go — LE REJET RAPIDE D'UNE POSITION RESTE UNIQUE ET RESTE
// BRANCHE.
//
// Les balayages bit a bit d'un en-tete de record delta (ancrage des bipedes, bandes d'objets du
// monde, visees, creations) ecartent les positions par [prochainEnteteDelta] ou
// [prochainPrefixeNEW], qui lisent un mot de 64 bits pour une quarantaine de positions. Ce fichier
// interdit deux derives : qu'une de ces boucles revienne au test position par position, et qu'un
// autre site recopie la lecture par mot (decalages de la porte et du compteur de masque) au lieu
// d'appeler le helper. Les equivalences sont prouvees par les tests differentiels voisins, qui
// gardent l'ancienne boucle comme reference.

import (
	"strings"
	"testing"
)

func TestLesBalayagesDEnteteDeltaPassentParLeRejetRapide(t *testing.T) {
	branches := []struct {
		fichier, appel, ancienneBoucle string
	}{
		{"delta_biped_walk.go", "prochainEnteteDelta(", "for p := 0; p+minRecord <= total && p < fin; {"},
		{"offline_aim_only.go", "prochainEnteteDelta(", "for p := 0; p+bipedHeaderBits <= total && p < fin; {"},
		{"pistes_du_monde.go", "prochainEnteteDelta(", "for p := 0; p <= limit && p < fin; p++ {"},
		{"creations_du_monde.go", "prochainPrefixeNEW(", "for p := 0; p <= total-woNewHeaderBits && p < fin; p++ {"},
	}
	for _, b := range branches {
		appelee := false
		for i, ligne := range lignesDeCode(t, b.fichier) {
			if strings.Contains(ligne, b.appel) {
				appelee = true
			}
			if strings.Contains(ligne, b.ancienneBoucle) {
				t.Errorf("%s:%d reprend la boucle position par position :\n\t%s\n"+
					"Les positions que le prefixe, la porte ou le compteur de masque ecartent se sautent par %s",
					b.fichier, i+1, ligne, b.appel)
			}
		}
		if !appelee {
			t.Errorf("%s n'appelle plus %s : la boucle teste l'en-tete a chaque bit", b.fichier, b.appel)
		}
	}
}

func TestLaLectureParMotDEnteteDeltaEstUnique(t *testing.T) {
	const helper = "entete_delta_balayage.go"
	motifs := []string{"<< porteDeLEnteteDelta", "<< compteurDeLEnteteDelta", "(w << 16)", "(w << 17)", "w<<18"}
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		if nom == helper {
			continue
		}
		for i, ligne := range lignesDeCode(t, nom) {
			for _, m := range motifs {
				if strings.Contains(ligne, m) {
					t.Errorf("%s:%d recopie la lecture par mot de l'en-tete delta :\n\t%s\n"+
						"Elle vit dans prochainEnteteDelta (%s).", nom, i+1, ligne, helper)
				}
			}
		}
	}
}

// TestLeCompteurDeMasqueMinimalDuBipedeReste1Ou2 : [prochainEnteteDelta] ne sait ecarter qu'un
// compteur de masque inferieur a 1 ou a 2. Un minimum du bipede hors de ces valeurs demande de
// l'apprendre au helper (sa borne reste necessaire a 2, jamais suffisante au-dela).
func TestLeCompteurDeMasqueMinimalDuBipedeReste1Ou2(t *testing.T) {
	if bipedMinMaskCnt != 1 && bipedMinMaskCnt != 2 {
		t.Fatalf("bipedMinMaskCnt = %d : prochainEnteteDelta ne connait que 1 et 2", bipedMinMaskCnt)
	}
}
