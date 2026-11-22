package signaux

// statborg_figees_test.go — LA LECTURE DES VALEURS FIGEES (composants 28 a 55), sur des vecteurs
// synthetiques construits a la grammaire : un enregistrement de slot 24, liste creuse — un compteur de la
// manche en cours, puis des valeurs figees (le composant 28 + k est la valeur figee du compteur k) pour les
// manches terminees du masque.

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// vecFige ecrit un paquet bit a bit.
type vecFige struct {
	pay []byte
	q   int
}

func (v *vecFige) bits(n int, x uint64) {
	v.pay = setBitsBE(v.pay, v.q, n, x)
	v.q += n
}

// valeur ecrit une valeur a longueur variable sur 8 ou 16 bits (selecteur 0 ou 1), ou le selecteur 3 que
// le lecteur refuse quand `illisible`.
func (v *vecFige) valeur(x uint64, illisible bool) {
	switch {
	case illisible:
		v.bits(2, 3)
	case x < 1<<7:
		v.bits(2, 0)
		v.bits(8, x)
	default:
		v.bits(2, 1)
		v.bits(16, x)
	}
}

// figee ecrit une valeur figee : le bit de defaut, puis la valeur quand elle n'est pas nulle.
func (v *vecFige) figee(x uint64) {
	if x == 0 {
		v.bits(1, 1)
		return
	}
	v.bits(1, 0)
	v.valeur(x, false)
}

// statVecteurFige fabrique un enregistrement de slot 24 et de manche `round` aux composants `idx`
// (creux, au plus sept) : un compteur de la manche vaut (3, 1) ; chaque composant de valeurs figees `i` porte
// le masque `mask` et, par manche du masque, la paire `paires[i][k]`. `illisible` remplace la premiere valeur
// figee par une largeur refusee.
func statVecteurFige(round uint64, idx []uint64, mask uint64, paires map[uint64][][2]uint64, illisible bool) []byte {
	v := &vecFige{pay: make([]byte, 96)}
	v.bits(1, 1)
	v.bits(statIDBits, 24)
	v.bits(statGenBits, statGenValue)
	v.bits(1, 0)
	v.bits(3, uint64(len(idx)))
	for _, i := range idx {
		v.bits(statCompIndexBits, i)
	}
	for _, i := range idx {
		if i < statCompteursDeManche {
			v.bits(statHdrBits, round)
			v.bits(statHdrBits, round)
			v.valeur(3, false)
			v.valeur(1, false)
			v.bits(2, 0)
			continue
		}
		v.bits(statMasqueManchesBits, mask)
		for k, p := range paires[i] {
			if k == 0 && illisible {
				v.bits(1, 0)
				v.valeur(0, true)
				return v.pay
			}
			v.figee(p[0])
			v.figee(p[1])
		}
	}
	return v.pay
}

// recordDuSlot24 rend l'enregistrement du vecteur : slot 24, compteur 2 a (3, 1).
func recordDuSlot24(recs []types.StatRecord) (types.StatRecord, bool) {
	for _, r := range recs {
		if r.Slot == 24 && r.Comps[2] == (types.StatValue{A: 3, B: 1}) {
			return r, true
		}
	}
	return types.StatRecord{}, false
}

// TestDeuxManchesUneFigee : au debut de la manche 1, la manche 0 est figee ; ses valeurs sont rangees sous
// le compteur dont elles sont la valeur, et le compteur de la manche en cours reste lu.
func TestDeuxManchesUneFigee(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurFige(1, []uint64{2, 30}, 0b1, map[uint64][][2]uint64{30: {{7, 9}}}, false), 1000, StatborgSlotBandeSure, &c)
	r, ok := recordDuSlot24(recs)
	if !ok {
		t.Fatalf("l enregistrement n est pas rendu : %v", recs)
	}
	if r.Round != 1 || r.Comps[2] != (types.StatValue{A: 3, B: 1}) {
		t.Errorf("manche %d, compteur %+v : attendu manche 1, (3, 1)", r.Round, r.Comps[2])
	}
	if want := map[int][]types.StatFrozen{2: {{Round: 0, A: 7, B: 9}}}; !reflect.DeepEqual(r.Frozen, want) {
		t.Errorf("valeurs figees %v, attendu %v", r.Frozen, want)
	}
	if _, ok := r.Comps[30]; ok {
		t.Error("une valeur figee est rangee parmi les compteurs de la manche")
	}
}

// TestTroisManchesDeuxFigees : au debut de la manche 2, les manches 0 et 1 sont figees, chaque manche
// portant ses deux valeurs l'une apres l'autre ; une valeur par defaut vaut 0.
func TestTroisManchesDeuxFigees(t *testing.T) {
	pay := statVecteurFige(2, []uint64{2, 30, 31}, 0b11,
		map[uint64][][2]uint64{30: {{7, 0}, {0, 12}}, 31: {{7, 0}, {0, 12}}}, false)
	r, ok := recordDuSlot24(scanFrameForRecords(pay, StatborgSlotBandeSure))
	if !ok {
		t.Fatal("l enregistrement n est pas rendu")
	}
	manches := []types.StatFrozen{{Round: 0, A: 7, B: 0}, {Round: 1, A: 0, B: 12}}
	if want := map[int][]types.StatFrozen{2: manches, 3: manches}; !reflect.DeepEqual(r.Frozen, want) {
		t.Errorf("valeurs figees %v, attendu %v", r.Frozen, want)
	}
}

// TestUneManchePasDeValeurFigee : un enregistrement de la premiere manche ne porte aucune valeur figee.
func TestUneManchePasDeValeurFigee(t *testing.T) {
	r, ok := recordDuSlot24(scanFrameForRecords(statVecteurFige(0, []uint64{2}, 0, nil, false), StatborgSlotBandeSure))
	if !ok {
		t.Fatal("l enregistrement n est pas rendu")
	}
	if r.Frozen != nil {
		t.Errorf("valeurs figees sur la premiere manche : %v", r.Frozen)
	}
}

// TestUneValeurFigeeIllisibleAbandonneLEnregistrement : une valeur figee a une largeur que le lecteur refuse
// prouve un ancrage fortuit ; l'enregistrement entier est abandonne et compte, comme pour un compteur.
func TestUneValeurFigeeIllisibleAbandonneLEnregistrement(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurFige(1, []uint64{2, 30}, 0b1, map[uint64][][2]uint64{30: {{7, 9}}}, true), 1000, StatborgSlotBandeSure, &c)
	if _, ok := recordDuSlot24(recs); ok {
		t.Fatal("l enregistrement a la valeur figee illisible est rendu")
	}
	if c.EnregistrementsAbandonnes == 0 {
		t.Error("l enregistrement est abandonne sans etre compte")
	}
	if c.ComposantsArretes != 0 {
		t.Errorf("%d arret(s) : une valeur figee illisible abandonne, elle n arrete pas", c.ComposantsArretes)
	}
}

// TestUnMasqueAuDelaDesManchesAbandonne : un masque qui designe une manche au-dela de StatborgMancheMax n'est
// pas une emission du moteur.
func TestUnMasqueAuDelaDesManchesAbandonne(t *testing.T) {
	pay := statVecteurFige(1, []uint64{2, 30}, 1<<(StatborgMancheMax+1), map[uint64][][2]uint64{30: {{7, 9}}}, false)
	if _, ok := recordDuSlot24(scanFrameForRecords(pay, StatborgSlotBandeSure)); ok {
		t.Fatal("un masque hors des manches est lu")
	}
}

// TestPrecisionAuCentieme : la precision figee est lue telle que le film l'ecrit, en centiemes de pour cent
// (45 touches sur 73 tirs : 6164).
func TestPrecisionAuCentieme(t *testing.T) {
	pay := statVecteurFige(1, []uint64{2, 34, 35}, 0b1, map[uint64][][2]uint64{34: {{4, 73}}, 35: {{45, 6164}}}, false)
	r, ok := recordDuSlot24(scanFrameForRecords(pay, StatborgSlotBandeSure))
	if !ok {
		t.Fatal("l enregistrement n est pas rendu")
	}
	if got := r.Frozen[7]; !reflect.DeepEqual(got, []types.StatFrozen{{Round: 0, A: 45, B: 6164}}) {
		t.Errorf("tirs au but et precision %v, attendu (45, 6164)", got)
	}
}

// TestLEntreeArreteApresLesValeursFigees : apres les valeurs figees, l'entree (57) est d'un autre lecteur ;
// la lecture s'y arrete, le dit, et garde le compteur et les valeurs figees lus avant elle.
func TestLEntreeArreteApresLesValeursFigees(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurFige(1, []uint64{2, 30, 57}, 0b1, map[uint64][][2]uint64{30: {{7, 9}}}, false), 1000, StatborgSlotBandeSure, &c)
	r, ok := recordDuSlot24(recs)
	if !ok || len(r.Frozen[2]) != 1 {
		t.Fatalf("le compteur et la valeur figee lus avant l entree sont perdus : %v", recs)
	}
	if c.ComposantsArretes == 0 {
		t.Error("l arret sur l entree n est pas compte")
	}
}

// TestSansCompteurPasDEnregistrement : un enregistrement qui ne porte que des valeurs figees n'a pas de numero
// de manche ; il n'est pas rendu, et il est compte.
func TestSansCompteurPasDEnregistrement(t *testing.T) {
	var c LectureDuStatborg
	recs := scanFrameAvecReplis(statVecteurFige(0, []uint64{30}, 0b1, map[uint64][][2]uint64{30: {{7, 9}}}, false), 1000, StatborgSlotBandeSure, &c)
	for _, r := range recs {
		if r.Slot == 24 && len(r.Frozen) > 0 {
			t.Fatalf("un enregistrement sans compteur est rendu : %v", r)
		}
	}
	if c.EnregistrementsAbandonnes == 0 {
		t.Error("l enregistrement sans compteur n est pas compte")
	}
}
