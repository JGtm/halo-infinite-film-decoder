package grammar

// entete_delta_visee_test.go — LE BALAYAGE DES VISEES SANS POSITION ([scanBipedAimRecordsJusqua]),
// CONTRE SA FORME POSITION PAR POSITION : memes visees, dans le meme ordre, sur des payloads
// construits (dont les bords) et aleatoires.

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

// scanBipedAimRecordsJusquaReference est l'ancienne boucle : un test d'en-tete complet a chaque bit.
func scanBipedAimRecordsJusquaReference(payload []byte, slots SlotBand, gens *GenerationsVivantes,
	ctx ContexteDeLecture, fin int) []viseeLue {
	total := len(payload) * 8
	var out []viseeLue
	br := LecteurSur(payload)
	br.PoserContexte(ctx)
	for p := 0; p+bipedHeaderBits <= total && p < fin; {
		at, slot, ok := matchAimOnlyRecord(br, payload, p, slots, gens)
		if !ok {
			p++
			continue
		}
		var d componentDirs
		readAimingVectorComponent(payload, at, total, &d)
		if !d.HasYaw {
			p++
			continue
		}
		out = append(out, viseeLue{BipedAim: BipedAim{Slot: slot, YawRaw: d.YawRaw, PitchRaw: d.PitchRaw}, bit: p})
		p = at + 1 + aimYawBits + aimPitchBits
	}
	return out
}

// resultatDeVisee : ce qu'une forme rend, ou la panique qu'elle leve.
type resultatDeVisee struct {
	visees  []viseeLue
	panique any
}

func jouerLaVisee(f func() []viseeLue) (r resultatDeVisee) {
	defer func() { r.panique = recover() }()
	r.visees = f()
	return r
}

// masquesDeViseeDeTest : des masques qui portent i21 (la visee), apres des composants lisibles ou non.
var masquesDeViseeDeTest = [][]int{{21}, {21, 25}, {1, 21}, {2, 21, 25}, {1, 2, 3, 21}, {4, 21}, {5, 21}, {3, 4, 21, 30}, {1, 2}}

// recordDeViseeDeTest ecrit un record de visee (sans i0) ; les bits de queue de `aimRecord` sont
// retires.
func recordDeViseeDeTest(rng *rand.Rand, slot uint32) *bitWriter {
	idx := masquesDeViseeDeTest[rng.IntN(len(masquesDeViseeDeTest))]
	zeros := uint32(0)
	if rng.IntN(8) == 0 {
		zeros = uint32(1 + rng.IntN(3))
	}
	b := aimRecord(slot, uint32(rng.IntN(4)), zeros, idx, uint32(rng.IntN(1<<aimYawBits)),
		uint32(rng.IntN(1<<aimPitchBits)))
	return &bitWriter{buf: b, n: bipedHeaderBits + bipedIndexBits*len(idx) + 1 + aimYawBits + aimPitchBits}
}

func payloadsDeVisees(rng *rand.Rand, n int) [][]byte {
	out := make([][]byte, 0, n)
	for range n {
		nb := rng.IntN(120)
		pay := bruitDeTest(rng, nb)
		for range rng.IntN(5) {
			slot := slotsDesPayloads[rng.IntN(len(slotsDesPayloads))]
			w := recordDeViseeDeTest(rng, slot)
			if rng.IntN(4) == 0 {
				w = fragmentDEnteteDeTest(rng, slot)
			}
			poserLesBits(pay, rng.IntN(nb*8+1), w)
		}
		out = append(out, pay)
	}
	return out
}

func payloadsDeBordVisees(rng *rand.Rand) [][]byte {
	var out [][]byte
	for off := range 136 {
		w := recordDeViseeDeTest(rng, 517)
		juste := (off + w.n + 7) / 8
		for _, longueur := range []int{juste, juste - 1, juste + 1} {
			pay := make([]byte, longueur)
			poserLesBits(pay, off, w)
			out = append(out, pay)
		}
	}
	return out
}

// comparerLesVisees rend le nombre de visees de la reference, et le nombre de paniques communes.
func comparerLesVisees(t *testing.T, pay []byte, slots SlotBand) (visees, paniques int) {
	t.Helper()
	total := len(pay) * 8
	for _, gens := range gensDeTest() {
		for _, fin := range []int{total, total + 7, total / 2, 1, 0, -2, total - 21} {
			want := jouerLaVisee(func() []viseeLue {
				return scanBipedAimRecordsJusquaReference(pay, slots, gens, ContexteParDefaut(), fin)
			})
			got := jouerLaVisee(func() []viseeLue {
				return scanBipedAimRecordsJusqua(pay, slots, gens, ContexteParDefaut(), fin)
			})
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("len=%d fin=%d : %+v, reference %+v", len(pay), fin, got, want)
			}
			visees += len(want.visees)
			if want.panique != nil {
				paniques++
			}
		}
	}
	return visees, paniques
}

func TestBalayageDesViseesEgaleSaFormePositionParPosition(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 9))
	slots := bandeDesPayloads(t)
	visees, paniques, payloads := 0, 0, 0
	for _, pay := range payloadsDeBordVisees(rng) {
		v, p := comparerLesVisees(t, pay, slots)
		visees, paniques, payloads = visees+v, paniques+p, payloads+1
	}
	bord := visees
	for _, pay := range payloadsDeVisees(rng, 4000) {
		v, p := comparerLesVisees(t, pay, slots)
		visees, paniques, payloads = visees+v, paniques+p, payloads+1
	}
	if bord == 0 || visees == bord {
		t.Fatalf("acceptations exercees : %d de bord, %d aleatoires", bord, visees-bord)
	}
	t.Logf("%d payloads, %d visees acceptees (%d sur les payloads de bord), %d paniques identiques",
		payloads, visees, bord, paniques)
}

// TestBalayageDesViseesEgaleSaFormeSurLaMiniBobine : sur chaque payload delta de la bobine, avec sa
// bande de slots, ses decalages, ses troncatures et ses precedes de zeros, ses generations vivantes
// datees au paquet et plusieurs bornes `fin`.
func TestBalayageDesViseesEgaleSaFormeSurLaMiniBobine(t *testing.T) {
	fc, _ := contexteDesObjetsDuMonde(t)
	slots := fc.BipedSlots()
	ctx := fc.ContexteDeLecture()
	rng := rand.New(rand.NewPCG(81, 3))
	visees, payloads := 0, 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			gens := fc.GenerationsVivantesA(pk.TimestampUS)
			for _, pay := range variantesDeBord(rng, pk.Payload(data)) {
				for _, fin := range []int{len(pay) * 8, len(pay) * 4, 100} {
					want := scanBipedAimRecordsJusquaReference(pay, slots, gens, ctx, fin)
					got := scanBipedAimRecordsJusqua(pay, slots, gens, ctx, fin)
					if !reflect.DeepEqual(want, got) {
						t.Fatalf("chunk %d paquet %d fin %d : %+v, reference %+v", c, pk.Index, fin, got, want)
					}
					visees += len(want)
				}
				payloads++
			}
		}
	}
	if visees == 0 {
		t.Fatal("aucune visee sur la bobine : le test ne prouve rien")
	}
	t.Logf("%d payloads de la bobine, %d visees (tous `fin` confondus)", payloads, visees)
}
