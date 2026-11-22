package signaux

// statborg_compteur_arrete_test.go — UN COMPTEUR DE LA MANCHE QUI NE SE DECODE PAS PROUVE UN ANCRAGE
// FORTUIT ([decodeComponentsAvecArret]).
//
// Les vecteurs sont synthetiques, construits a la grammaire de la production : un enregistrement de
// slot 24, liste creuse de deux composants. Le premier (8) se decode, valeurs (0, 16) ; seul le
// second change d'un vecteur a l'autre : un compteur de la manche lisible (temoin), un compteur de
// la manche illisible (selecteur de largeur 3, 64 bits, que le lecteur refuse), ou l'entree (57),
// composant d'un autre lecteur, illisible de la meme facon.
//
// MUTATION JOUEE : retirer `if !ok && i < statCompteursDeManche` rougit
// `TestUnCompteurIllisibleAbandonneLEnregistrement`, et lui seul.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// statComposantLisible et statComposantIllisible sont les deux formes du second composant.
const (
	statComposantLisible = iota
	statComposantIllisible
)

// statVecteurDeuxCompos fabrique un paquet portant UN enregistrement ancre au bit 1 : slot 24, manche 0,
// composants [8, second] ; le composant 8 vaut (0, 16), le second vaut (5, 0) ou porte un selecteur de
// largeur 3 selon `forme`.
func statVecteurDeuxCompos(second uint64, forme int) []byte {
	pay := make([]byte, 64)
	pay = setBitsBE(pay, 0, 1, 1)
	pay = setBitsBE(pay, 1, statIDBits, 24)
	pay = setBitsBE(pay, 1+statIDBits, statGenBits, statGenValue)
	m := 1 + statIDBits + statGenBits
	pay = setBitsBE(pay, m, 1, 0)   // forme CREUSE
	pay = setBitsBE(pay, m+1, 3, 2) // deux composants
	pay = setBitsBE(pay, m+4, statCompIndexBits, 8)
	pay = setBitsBE(pay, m+4+statCompIndexBits, statCompIndexBits, second)
	q := m + 4 + 2*statCompIndexBits
	pay, q = statEcrireComposant(pay, q, 0, 16) // composant 8 : en-tetes nuls, A = 0, B = 16
	if forme == statComposantLisible {
		pay, _ = statEcrireComposant(pay, q, 5, 0)
		return pay
	}
	pay = setBitsBE(pay, q+2*statHdrBits, 2, 3) // canal A : selecteur 3, une largeur que le lecteur refuse
	return pay
}

// statEcrireComposant ecrit a `q` un composant de manche 0 aux canaux A et B sur 8 bits, sans canal
// conditionnel, et rend le paquet ecrit et le bit qui le suit.
func statEcrireComposant(pay []byte, q int, a, b uint64) ([]byte, int) {
	pay = setBitsBE(pay, q, 2*statHdrBits, 0) // les deux en-tetes de manche, nuls
	q += 2 * statHdrBits
	pay = setBitsBE(pay, q, 2, 0)
	pay = setBitsBE(pay, q+2, 8, a)
	pay = setBitsBE(pay, q+10, 2, 0)
	pay = setBitsBE(pay, q+12, 8, b)
	pay = setBitsBE(pay, q+20, 2, 0)
	return pay, q + 22
}

// statPorteSeize dit si l'un des enregistrements rendus porte le composant 8 du vecteur, valeurs (0, 16).
func statPorteSeize(recs []types.StatRecord) bool {
	for _, r := range recs {
		if v, ok := r.Comps[8]; ok && r.Slot == 24 && v.A == 0 && v.B == 16 {
			return true
		}
	}
	return false
}

// TestUnCompteurIllisibleAbandonneLEnregistrement : quand un compteur de la manche annonce ne se
// decode pas, l'enregistrement entier est abandonne et compte, et le composant lu avant lui n'est pas
// publie ; ce n'est pas un arret de lecture.
func TestUnCompteurIllisibleAbandonneLEnregistrement(t *testing.T) {
	var c LectureDuStatborg
	if recs := scanFrameAvecReplis(statVecteurDeuxCompos(9, statComposantIllisible), 1000, StatborgSlotBandeSure, &c); statPorteSeize(recs) {
		t.Fatalf("le composant lu avant un compteur illisible est publie : %v", recs)
	}
	if c.EnregistrementsAbandonnes == 0 {
		t.Error("l enregistrement au compteur illisible est abandonne sans etre compte")
	}
	if c.ComposantsArretes != 0 {
		t.Errorf("%d arret(s) compte(s) : un compteur illisible abandonne l enregistrement, il ne l arrete pas", c.ComposantsArretes)
	}
}

// TestUnCompteurLisibleGardeLEnregistrement : temoin positif, le meme vecteur au second compteur
// lisible ressort en entier, sans arret.
func TestUnCompteurLisibleGardeLEnregistrement(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurDeuxCompos(9, statComposantLisible), 1000, StatborgSlotBandeSure, &c)
	if !statPorteSeize(recs) {
		t.Fatalf("l enregistrement sain n est plus lu : %v", recs)
	}
	for _, r := range recs {
		if v, ok := r.Comps[9]; ok && r.Slot == 24 && v.A != 5 {
			t.Errorf("composant 9 = %+v, attendu A = 5", v)
		}
	}
}

// TestUnComposantDUnAutreLecteurArreteSansAbandonner : l'entree (57) n'est pas lue par le lecteur des
// compteurs ; son echec arrete la boucle, le dit, et garde le compteur lu avant elle.
func TestUnComposantDUnAutreLecteurArreteSansAbandonner(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurDeuxCompos(57, statComposantIllisible), 1000, StatborgSlotBandeSure, &c)
	if !statPorteSeize(recs) {
		t.Fatalf("le compteur lu avant l entree est perdu : %v", recs)
	}
	if c.ComposantsArretes == 0 {
		t.Error("l arret sur l entree n est pas compte")
	}
}
