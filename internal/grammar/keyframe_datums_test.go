package grammar

// keyframe_datums_test.go — LA TABLE D ENTITES D UNE IMAGE-CLE, RETROUVEE PAR SES EN-TETES (A2.H) : une table
// ecrite au bit pres selon `FUN_142e2bfd0` (eid, type, poignee, drapeaux, octet ; n puis la creation ; m), la
// generation 0, les emplacements vides, un candidat fantome dans les donnees d une autre entree, une table
// tronquee, puis la liaison que la table pose au monde.

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

const sansPoignee = 0xffffffff

// ecrireEnteteDeTable ecrit les 108 bits d un en-tete : eid, type, poignee, drapeaux (4), octet (8).
func (w *bitWriter) ecrireEnteteDeTable(id, ty, poignee uint32, drapeaux, octet uint64) {
	w.bits(uint64(id), 32)
	w.bits(uint64(ty), 32)
	w.bits(uint64(poignee), 32)
	w.bits(drapeaux, 4)
	w.bits(octet, 8)
}

// ecrireEntreeVide ecrit un emplacement vide : type 0xffffffff, rien ne suit.
func (w *bitWriter) ecrireEntreeVide() {
	w.ecrireEnteteDeTable(sansPoignee, sansPoignee, sansPoignee, 0, 0)
}

// ecrireEntreeVivante ecrit une entite vivante : un en-tete de drapeaux 3, n = 12 octets de creation, une
// structure de creation de motif quelconque, m = 0.
func (w *bitWriter) ecrireEntreeVivante(id, ty uint32) {
	w.ecrireEnteteDeTable(id, ty, 0x00001234, drapeauxAvecEtat, 0x05)
	const n = 12
	w.bits(uint64(n), 32)
	for range n {
		w.bits(0xa5, 8)
	}
	w.bits(0, 32)
}

// ecrireEntreeSupprimee ecrit le reste d une suppression : eid et type conserves, aucune poignee, n = m = 0.
func (w *bitWriter) ecrireEntreeSupprimee(id, ty uint32) {
	w.ecrireEnteteDeTable(id, ty, sansPoignee, drapeauxSansObjet, 0)
	w.bits(0, 32+32)
}

// eidDeTest compose un eid : generation en tete, emplacement dans les autres bits.
func eidDeTest(gen, slot uint32) uint32 { return gen<<30 | slot }

// payloadDeTable : le bit qui precede la table, les entrees `ecrire`, puis de quoi lire n et m derriere la derniere.
func payloadDeTable(ecrire func(w *bitWriter)) []byte {
	var w bitWriter
	w.bit(1)
	ecrire(&w)
	w.bits(0, 300)
	return w.buf
}

// TestLaTableDEntitesSeLitParSesEnTetes : huit emplacements — une generation 0 AVANT la premiere entree reconnue, des
// emplacements vides, le reste d une suppression de generation 0 ENTRE deux entrees reconnues, une generation 0 APRES la
// derniere — rendent les six entrees non vides, a emplacements croissants, avec leur eid complet et leur type. Les
// emplacements vides ne rendent rien. MUTATIONS — la seconde passe retiree : les trois generations 0 manquent ; la
// generation 0 admise des la premiere passe : ROUGE (voir [TestUnCandidatDeGenerationZeroDansLesDonneesNEstPasUneEntree]).
func TestLaTableDEntitesSeLitParSesEnTetes(t *testing.T) {
	pay := payloadDeTable(func(w *bitWriter) {
		w.ecrireEntreeVivante(eidDeTest(0, 0), 36)   // generation 0, avant toute entree reconnue
		w.ecrireEntreeVide()                         // 1
		w.ecrireEntreeVivante(eidDeTest(1, 2), 3)    // premiere entree reconnue
		w.ecrireEntreeVivante(eidDeTest(2, 3), 35)   // une grosse creation
		w.ecrireEntreeSupprimee(eidDeTest(0, 4), 41) // reste de suppression, generation 0, entre deux voisines
		w.ecrireEntreeVide()                         // 5
		w.ecrireEntreeVivante(eidDeTest(3, 6), 5)    // derniere entree reconnue
		w.ecrireEntreeVivante(eidDeTest(0, 7), 10)   // generation 0 apres la derniere
	})
	want := []EnteteDeTable{
		{eidDeTest(0, 0), 36}, {eidDeTest(1, 2), 3}, {eidDeTest(2, 3), 35},
		{eidDeTest(0, 4), 41}, {eidDeTest(3, 6), 5}, {eidDeTest(0, 7), 10},
	}
	got, ecartes := EntetesDeTable(pay)
	if !reflect.DeepEqual(got, want) || ecartes != 0 {
		t.Fatalf("entrees %v (ecartes %d), attendu %v (0)", got, ecartes, want)
	}
	table, _ := TableDeDatums(pay)
	if e := table[4]; e != (EnteteDeTable{eidDeTest(0, 4), 41}) || len(table) != len(want) {
		t.Fatalf("table par emplacement %v : le reste de suppression du slot 4 manque ou la table est incomplete", table)
	}
	if e := want[0]; e.Slot() != 0 || e.Gen() != 0 {
		t.Fatalf("Slot/Gen de %v", e)
	}
}

// TestUnCandidatDeGenerationZeroDansLesDonneesNEstPasUneEntree : le mot de 64 bits `[eid][type]` d un emplacement
// (generation 1, slot 4, type 2) se retrouve DANS LES DONNEES de creation d une autre entree, suivi d un octet d en-tete dont les
// trois bits de poids fort sont poses (0x3e) : l invariant le refuse, le vrai en-tete du slot 4 (type 6) donne son type.
// Une coincidence de generation 0 qui satisfait tous les invariants mais dont l emplacement n est pas strictement compris entre
// les deux voisines est refusee. MUTATION — l invariant de l octet retire : le slot 4 prend le type 2, ROUGE.
func TestUnCandidatDeGenerationZeroDansLesDonneesNEstPasUneEntree(t *testing.T) {
	pay := payloadDeTable(func(w *bitWriter) {
		w.ecrireEntreeVivante(eidDeTest(1, 2), 3)
		// slot 3, type 13 (136 octets de creation) : ses donnees portent un fantome de generation 1 sur le slot 4 / type
		// 2, a l octet 0x3e, puis un fantome de generation 0 qui satisfait tous les invariants (slot 9, reste de suppression).
		w.ecrireEnteteDeTable(eidDeTest(2, 3), 13, 0x00001234, drapeauxAvecEtat, 0x05)
		w.bits(136, 32)
		debut := w.n
		w.bits(0xa5, 7)
		w.ecrireEnteteDeTable(eidDeTest(1, 4), 2, 0x00000001, 0x1, 0x3e)
		w.bits(0, 32) // n = 0 : seul l octet 0x3e refuse ce fantome
		w.ecrireEnteteDeTable(eidDeTest(0, 9), 41, sansPoignee, drapeauxSansObjet, 0)
		w.bits(0, 64)
		for w.n < debut+136*8 { // le reste de la creation, sans en-tete
			w.bits(0xa5, 8)
		}
		w.bits(0, 32)
		w.ecrireEntreeVivante(eidDeTest(1, 4), 6) // le vrai slot 4
	})
	table, _ := TableDeDatums(pay)
	if e, ok := table[4]; !ok || e != (EnteteDeTable{eidDeTest(1, 4), 6}) {
		t.Errorf("slot 4 : %v (%v), attendu generation 1, type 6", e, ok)
	}
	if e, ok := table[9]; ok {
		t.Errorf("le fantome du slot 9 (donnees du slot 3) est entre dans la table : %v", e)
	}
}

// TestUneEntreeDontLaCroissanceEstRompueEstEcartee : un en-tete valide dont l emplacement ne depasse pas celui de la
// derniere entree retenue est refuse et son emplacement compte comme ecarte ; la table garde la sequence croissante.
func TestUneEntreeDontLaCroissanceEstRompueEstEcartee(t *testing.T) {
	pay := payloadDeTable(func(w *bitWriter) {
		w.ecrireEntreeVivante(eidDeTest(1, 5), 3)
		w.ecrireEntreeVivante(eidDeTest(1, 2), 3) // l emplacement recule
		w.ecrireEntreeVivante(eidDeTest(1, 8), 3)
	})
	got, ecartes := EntetesDeTable(pay)
	want := []EnteteDeTable{{eidDeTest(1, 5), 3}, {eidDeTest(1, 8), 3}}
	if !reflect.DeepEqual(got, want) || ecartes != 1 {
		t.Fatalf("entrees %v (ecartes %d), attendu %v (1)", got, ecartes, want)
	}
}

// TestUneTableTronqueeNeRendQueSesEntreesCompletes : un payload coupe au milieu d une entree ne rend pas cette entree, et un
// payload plus court qu un en-tete ne rend rien, sans panique.
func TestUneTableTronqueeNeRendQueSesEntreesCompletes(t *testing.T) {
	pay := payloadDeTable(func(w *bitWriter) {
		w.ecrireEntreeVivante(eidDeTest(1, 2), 3)
		w.ecrireEntreeVivante(eidDeTest(1, 3), 35)
	})
	complete, _ := EntetesDeTable(pay)
	if len(complete) != 2 {
		t.Fatalf("table complete : %v", complete)
	}
	// coupee dans la creation du second : son en-tete (108 bits) tient encore, mais pas de quoi lire n et m derriere lui
	coupee := pay[:(1+108+32+12*8+32+108+16)/8+1]
	got, _ := EntetesDeTable(coupee)
	if want := []EnteteDeTable{{eidDeTest(1, 2), 3}}; !reflect.DeepEqual(got, want) {
		t.Errorf("table tronquee : %v, attendu %v", got, want)
	}
	for _, p := range [][]byte{nil, {0x80}, make([]byte, 20)} {
		if got, ec := EntetesDeTable(p); len(got) != 0 || ec != 0 {
			t.Errorf("payload de %d octets : %v (%d)", len(p), got, ec)
		}
	}
}

// TestLaTableLieLeTypeEtLaGenerationDesEntrees : une liaison deja posee par la chaine de l image-cle sous un AUTRE type est
// remplacee par celle de la table, qui porte l eid complet (une autre generation ne la lit pas) ; une liaison qui repond de meme
// est gardee avec sa provenance. MUTATIONS — la liaison existante jamais remplacee ; la generation oubliee (liaison de generation
// inconnue) : ROUGES.
func TestLaTableLieLeTypeEtLaGenerationDesEntrees(t *testing.T) {
	w := NewWorld(nil)
	w.BindImageCle(1, 2, 9) // la chaine dit type 9 pour le slot 2 ; la table dit 3
	w.BindImageCle(1, 3, 35)
	table := map[uint32]EnteteDeTable{
		2: {eidDeTest(1, 2), 3},
		3: {eidDeTest(2, 3), 35},
		4: {eidDeTest(0, 4), 41}, // jamais lie : posee, de generation 0
	}
	if posees := lierLesDatums(w, table); posees != 2 {
		t.Fatalf("liaisons posees %d, attendu 2 (slots 2 et 4)", posees)
	}
	if ti, ok := w.ArchetypeForEID(eidDeTest(1, 2)); !ok || ti != 3 {
		t.Errorf("slot 2 : (%d, %v), attendu le type de la table, 3", ti, ok)
	}
	if _, ok := w.ArchetypeForEID(eidDeTest(2, 2)); ok {
		t.Error("slot 2 : une autre generation lit la liaison de la table")
	}
	if ti, ok := w.ArchetypeForEID(eidDeTest(0, 4)); !ok || ti != 41 {
		t.Errorf("slot 4 (generation 0) : (%d, %v), attendu 41", ti, ok)
	}
	if l := w.liaisonDu(3); l != lecture.LiaisonImageCle {
		t.Errorf("slot 3 : provenance %d, la liaison de la chaine qui repond de meme doit etre gardee", l)
	}
	if l := w.liaisonDu(2); l != lecture.LiaisonDatum {
		t.Errorf("slot 2 : provenance %d, attendu la recuperation de la table", l)
	}
}

// TestUneTailleDeCreationAuDelaDeLaBorneEstRefusee : un en-tete de drapeaux 3 dont n depasse [tailleMaxDeCreation] n est pas une
// entree, un en-tete dont n vaut la borne l est. MUTATION — la borne retiree : le premier cas devient une entree, ROUGE.
func TestUneTailleDeCreationAuDelaDeLaBorneEstRefusee(t *testing.T) {
	for _, c := range []struct {
		n       uint32
		attendu int
	}{{tailleMaxDeCreation, 1}, {tailleMaxDeCreation + 1, 0}} {
		pay := payloadDeTable(func(w *bitWriter) {
			w.ecrireEnteteDeTable(eidDeTest(1, 2), 3, 0x00001234, drapeauxAvecEtat, 0x05)
			w.bits(uint64(c.n), 32)
			for range c.n {
				w.bits(0xa5, 8)
			}
			w.bits(0, 32)
		})
		if got, _ := EntetesDeTable(pay); len(got) != c.attendu {
			t.Errorf("n = %d : %d entree(s), attendu %d", c.n, len(got), c.attendu)
		}
	}
}

// BindDatum pose la liaison souple de generation inconnue que pose l anticipation ([World.lierCommeDatum]) : les tests et les
// instruments en ont besoin pour poser un slot dont la generation n est pas connue.
func (w *World) BindDatum(slot, typeIndex uint32) {
	w.lierCommeDatum(slot, typeIndex, lecture.LiaisonDatum)
}
