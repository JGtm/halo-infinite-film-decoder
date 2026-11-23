package grammar

// entete_delta_payloads_test.go — LES PAYLOADS DES TESTS DIFFERENTIELS DES BALAYAGES D'EN-TETE
// DELTA : bruit, records bipedes complets et fragments d'en-tete poses a des positions de bit
// quelconques, dont les bords (debut, fin, cheval sur un mot de 64 bits, payload plus court qu'un
// mot).

import (
	"math/rand/v2"
	"testing"
)

// slotsDesPayloads sont les slots que les payloads de test portent et que les bandes de test
// retiennent.
var slotsDesPayloads = []uint32{7, 512, 517, 1000, 8191}

// poserLesBits ecrit (OU) les bits de `src` dans `dst` a partir du bit `off` ; les bits qui
// tomberaient hors de `dst` sont perdus.
func poserLesBits(dst []byte, off int, src *bitWriter) {
	for k := range src.n {
		if src.buf[k>>3]>>(7-uint(k&7))&1 == 0 {
			continue
		}
		if at := off + k; at >= 0 && at>>3 < len(dst) {
			dst[at>>3] |= 1 << (7 - uint(at&7))
		}
	}
}

// bruitDeTest rend n octets de bruit : nul, clairseme ou dense.
func bruitDeTest(rng *rand.Rand, n int) []byte {
	out := make([]byte, n)
	switch rng.IntN(3) {
	case 0:
	case 1:
		for i := range out {
			out[i] = byte(rng.Uint32()) & byte(rng.Uint32()) & byte(rng.Uint32())
		}
	default:
		for i := range out {
			out[i] = byte(rng.Uint32())
		}
	}
	return out
}

// recordBipedDeTest ecrit un record biped complet (en-tete, masque, i0) de slot et de compteur
// de masque donnes, aux quanta tires au hasard.
func recordBipedDeTest(rng *rand.Rand, slot uint32, tag, masque uint64) *bitWriter {
	w := &bitWriter{}
	writeBipedRecord(w, slot, tag, masque, rng.Uint64()%(1<<13), rng.Uint64()%(1<<13), rng.Uint64()%(1<<14))
	return w
}

// fragmentDEnteteDeTest ecrit un en-tete delta et son masque, suivis de bits quelconques :
// prefixe, slot, generation, deux bits de porte (nuls une fois sur deux), compteur, index
// croissants (depuis zero une fois sur deux).
func fragmentDEnteteDeTest(rng *rand.Rand, slot uint32) *bitWriter {
	w := &bitWriter{}
	w.bits(1, 1)
	w.bits(uint64(slot), bipedSlotBits)
	w.bits(uint64(rng.IntN(4)), 2)
	if rng.IntN(2) == 0 {
		w.bits(0, 2)
	} else {
		w.bits(uint64(rng.IntN(4)), 2)
	}
	mc := rng.IntN(8)
	w.bits(uint64(mc), 3)
	idx := rng.IntN(2)
	for range mc {
		w.bits(uint64(idx), 6)
		idx += 1 + rng.IntN(5)
	}
	for range 40 + rng.IntN(60) {
		w.bit(uint64(rng.IntN(2)))
	}
	return w
}

// payloadsAleatoires rend n payloads de 0 a 119 octets, semes de vrais records et de fragments
// d'en-tete a des positions de bit quelconques.
func payloadsAleatoires(rng *rand.Rand, n int) [][]byte {
	out := make([][]byte, 0, n)
	for range n {
		nb := rng.IntN(120)
		pay := bruitDeTest(rng, nb)
		for range rng.IntN(5) {
			slot := slotsDesPayloads[rng.IntN(len(slotsDesPayloads))]
			var w *bitWriter
			if rng.IntN(2) == 0 {
				w = recordBipedDeTest(rng, slot, uint64(rng.IntN(4)), uint64(1+rng.IntN(7)))
			} else {
				w = fragmentDEnteteDeTest(rng, slot)
			}
			poserLesBits(pay, rng.IntN(nb*8+1), w)
		}
		out = append(out, pay)
	}
	return out
}

// payloadsDeBord rend, pour chaque position de depart de 0 a 135 bits (tous les alignements, et le
// cheval sur la frontiere d'un mot de 64 bits), un payload qui porte un record biped minimal
// (compteur de masque 2, deux quanta de generation 1) et rien d'autre : au plus juste (le record
// finit au dernier bit utile ou dans le dernier octet), puis un octet de moins (le record ne tient
// plus), puis suivi d'un octet de queue.
func payloadsDeBord(rng *rand.Rand, slot uint32) [][]byte {
	var out [][]byte
	for off := range 136 {
		w := recordBipedDeTest(rng, slot, 1, bipedMinMaskCnt)
		juste := (off + w.n + 7) / 8
		for _, longueur := range []int{juste, juste - 1, juste + 1} {
			pay := make([]byte, longueur)
			poserLesBits(pay, off, w)
			out = append(out, pay)
		}
	}
	return out
}

// bandeDesPayloads est la bande de slots des tests : les slots des payloads, pas les autres.
func bandeDesPayloads(t *testing.T) SlotBand {
	t.Helper()
	m := map[uint32]bool{}
	for _, s := range slotsDesPayloads {
		m[s] = true
	}
	return NewSlotBand(m)
}

// decalerLesBits rend `pay` precede de `k` bits de bruit : tout record y change d'alignement.
func decalerLesBits(rng *rand.Rand, pay []byte, k int) []byte {
	return precederDe(pay, k, func() uint64 { return uint64(rng.IntN(2)) })
}

// precederDeZeros rend `pay` precede de `k` bits nuls.
func precederDeZeros(pay []byte, k int) []byte {
	return precederDe(pay, k, func() uint64 { return 0 })
}

func precederDe(pay []byte, k int, bit func() uint64) []byte {
	w := &bitWriter{}
	for range k {
		w.bit(bit())
	}
	for i := range len(pay) * 8 {
		w.bit(uint64(pay[i>>3] >> (7 - uint(i&7)) & 1))
	}
	return w.buf
}

// variantesDeBord rend `pay`, ses decalages de 1 a 7 bits (autre alignement, bord de mot deplace) et
// ses troncatures de 1 a 9 octets (le dernier record arrive au bord du tampon, ou le depasse) et
// ses precedes de 61 a 130 bits nuls.
func variantesDeBord(rng *rand.Rand, pay []byte) [][]byte {
	out := [][]byte{pay}
	for k := 1; k <= 7; k++ {
		out = append(out, decalerLesBits(rng, pay, k))
	}
	for k := 1; k <= 9 && k < len(pay); k++ {
		out = append(out, pay[:len(pay)-k])
	}
	// Une longue plage de zeros devant : la recherche franchit des mots entiers sans candidat.
	for _, zeros := range []int{61, 62, 63, 64, 65, 100, 127, 130} {
		out = append(out, precederDeZeros(pay, zeros))
	}
	return out
}
