package grammar

// entete_delta_bandes_test.go — LA PASSE DES BANDES D'OBJETS DU MONDE ([echantillonsDesBandes]),
// CONTRE SA FORME POSITION PAR POSITION : meme liste d'echantillons, dans le meme ordre, sur des
// payloads construits (dont les bords), aleatoires, et sur la mini-bobine.

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// echantillonsDesBandesReference est l'ancienne boucle : un en-tete d'objet du monde a chaque bit.
func echantillonsDesBandesReference(pay []byte, appartenance []uint32, n int, wr *profile.Vec3Range,
	lg profile.PrecisionDescriptor, fin int) [][]projSample {
	out := make([][]projSample, n)
	posBits := projPosBits(lg)
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
	var curseurs [maxBandesParPasse]int
	for p := 0; p <= limit && p < fin; p++ {
		h, ok := enteteDObjetDuMonde(pay, p)
		if !ok || int(h.Slot) >= len(appartenance) || appartenance[h.Slot] == 0 {
			continue
		}
		var s projSample
		lu, accepte := false, false
		for i := range n {
			if appartenance[h.Slot]&(1<<uint(i)) == 0 || curseurs[i] > p {
				continue
			}
			if !lu {
				lu = true
				s, accepte = echantillonDuRecord(pay, p, h, wr, lg)
			}
			if !accepte {
				break
			}
			out[i] = append(out[i], s)
			curseurs[i] = p + posBits + 1
		}
	}
	return out
}

// precisionDeTest : une region sur un bit, des axes de 14 bits.
var precisionDeTest = profile.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{14, 14, 14}, Region: 0}

// recordObjetDuMondeDeTest ecrit un record delta d'objet du monde : en-tete, masque (le premier index
// est 0 sauf `sansI0`), puis la position (porte nulle, region, trois quanta ni nuls ni satures).
func recordObjetDuMondeDeTest(rng *rand.Rand, slot uint32, lg profile.PrecisionDescriptor, sansI0 bool) *bitWriter {
	w := &bitWriter{}
	mc := 1 + rng.IntN(worldObjectMaxMaskCnt)
	w.bits(1, 1)
	w.bits(uint64(slot), handleSlotBits)
	w.bits(uint64(rng.IntN(4)), 2)
	w.bits(0, 2)
	w.bits(uint64(mc), 3)
	idx := 0
	if sansI0 {
		idx = 1
	}
	for range mc {
		w.bits(uint64(idx), worldObjectIndexBits)
		idx += 1 + rng.IntN(3)
	}
	w.bits(0, 2)
	w.bits(uint64(lg.Region), int(lg.IndexW))
	for _, largeur := range lg.AxisW {
		w.bits(1+rng.Uint64()%(1<<largeur-2), int(largeur))
	}
	w.bits(uint64(rng.IntN(4)), 2)
	return w
}

// bandesDeTest : trois bandes qui se recouvrent, sur les slots des payloads.
func bandesDeTest() [][]uint32 {
	return [][]uint32{{7, 512, 517}, {512, 1000, 8191}, {7, 517, 1000, 8191}}
}

// payloadsDObjetsDuMonde rend des payloads semes de records d'objet du monde et de fragments.
func payloadsDObjetsDuMonde(rng *rand.Rand, n int) [][]byte {
	out := make([][]byte, 0, n)
	for range n {
		nb := rng.IntN(120)
		pay := bruitDeTest(rng, nb)
		for range rng.IntN(5) {
			slot := slotsDesPayloads[rng.IntN(len(slotsDesPayloads))]
			var w *bitWriter
			if rng.IntN(3) == 0 {
				w = fragmentDEnteteDeTest(rng, slot)
			} else {
				w = recordObjetDuMondeDeTest(rng, slot, precisionDeTest, rng.IntN(5) == 0)
			}
			poserLesBits(pay, rng.IntN(nb*8+1), w)
		}
		out = append(out, pay)
	}
	return out
}

// payloadsDeBordObjets : un record d'objet du monde a chaque position de depart de 0 a 135 bits,
// au plus juste, un octet de moins, un octet de plus.
func payloadsDeBordObjets(rng *rand.Rand) [][]byte {
	var out [][]byte
	for off := range 136 {
		w := recordObjetDuMondeDeTest(rng, 517, precisionDeTest, false)
		juste := (off + w.n + 7) / 8
		for _, longueur := range []int{juste, juste - 1, juste + 1} {
			pay := make([]byte, longueur)
			poserLesBits(pay, off, w)
			out = append(out, pay)
		}
	}
	return out
}

// comparerLesBandes rend le nombre d'echantillons de la reference apres comparaison, pour
// plusieurs bornes `fin` et pour une bande, deux et trois.
func comparerLesBandes(t *testing.T, pay []byte, bandes [][]uint32, wr *profile.Vec3Range,
	lg profile.PrecisionDescriptor) int {
	t.Helper()
	total, acceptes := len(pay)*8, 0
	for n := 1; n <= len(bandes); n++ {
		app := appartenanceDesBandes(bandes[:n])
		for _, fin := range []int{total, total + 7, total / 2, 1, 0, -2, total - 21} {
			want := echantillonsDesBandesReference(pay, app, n, wr, lg, fin)
			got := echantillonsDesBandes(pay, app, n, wr, lg, fin)
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("len=%d n=%d fin=%d : %+v, reference %+v", len(pay), n, fin, got, want)
			}
			for _, b := range want {
				acceptes += len(b)
			}
		}
	}
	return acceptes
}

func TestEchantillonsDesBandesEgalentLaFormePositionParPosition(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 4))
	wr := profile.QuantRangeCEBiped()
	bandes := bandesDeTest()
	acceptes, payloads := 0, 0
	for _, pay := range payloadsDeBordObjets(rng) {
		acceptes += comparerLesBandes(t, pay, bandes, &wr, precisionDeTest)
		payloads++
	}
	bord := acceptes
	for _, pay := range payloadsDObjetsDuMonde(rng, 4000) {
		acceptes += comparerLesBandes(t, pay, bandes, &wr, precisionDeTest)
		payloads++
	}
	if bord == 0 || acceptes == bord {
		t.Fatalf("acceptations exercees : %d de bord, %d aleatoires", bord, acceptes-bord)
	}
	t.Logf("%d payloads, %d echantillons acceptes (%d sur les payloads de bord)", payloads, acceptes, bord)
}

// TestEchantillonsDesBandesEgalentLeurFormeSurLaMiniBobine : sur chaque payload delta de la bobine,
// et, pour un payload sur cinq, ses decalages, ses troncatures et ses precedes de zeros ; les trois
// bandes a pistes, plusieurs bornes `fin`.
func TestEchantillonsDesBandesEgalentLeurFormeSurLaMiniBobine(t *testing.T) {
	fc, wr := contexteDesObjetsDuMonde(t)
	lg := fc.ProfilDeBalayage().LargeursObjetDuMonde()
	var bandes [][]uint32
	for _, ti := range archetypesAPistes() {
		bandes = append(bandes, slotsDeLaBande(worldObjectSlotBand(fc, ti)))
	}
	rng := rand.New(rand.NewPCG(71, 5))
	acceptes, payloads, rangs := 0, 0, 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			variantes := variantesDeBord(rng, pk.Payload(data))
			if rangs++; rangs%5 != 0 {
				variantes = variantes[:1] // les variantes de bord d'un payload sur cinq : le test serait trop long
			}
			for _, pay := range variantes {
				acceptes += comparerLesBandes(t, pay, bandes, &wr, lg)
				payloads++
			}
		}
	}
	if acceptes == 0 {
		t.Fatal("aucun echantillon sur la bobine : le test ne prouve rien")
	}
	t.Logf("%d payloads de la bobine, %d echantillons acceptes (tous `fin` confondus)", payloads, acceptes)
}
