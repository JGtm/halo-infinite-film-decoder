package grammar

import (
	"cmp"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Ces tests gardent l'équivalence de deux formes rapides avec leur forme d'origine, conservée ici
// comme référence : l'en-tête de record biped dont la largeur d'i0 n'est calculée qu'après les
// rejets de tête, et la fenêtre glissante de mots filtrée par table de bits avant la recherche
// dans le jeu.

// matchBipedHeaderReference est la forme d'origine de [matchBipedHeader] : largeur d'i0 évaluée
// avant toute vérification.
func matchBipedHeaderReference(pay []byte, p, total int, slots SlotBand, gens *GenerationsVivantes, lay profile.I0Layout) (int, uint32, []int, bool) {
	need := lay.TotalBits()
	if uint32(source.BitsStricts(pay, p, 1)) != 1 {
		return 0, 0, nil, false
	}
	h := LireHandleDelta(pay, p)
	if !slots.Has(h.Slot) || !gens.Accepte(h) {
		return 0, 0, nil, false
	}
	if uint32(source.BitsStricts(pay, p+16, 2)) != 0 {
		return 0, 0, nil, false
	}
	mc := int(uint32(source.BitsStricts(pay, p+18, 3)))
	if mc < bipedMinMaskCnt || mc > bipedMaxMaskCnt {
		return 0, 0, nil, false
	}
	i0 := p + bipedHeaderBits + bipedIndexBits*mc
	if i0+need > total {
		return 0, 0, nil, false
	}
	idx, ok := ascendingFromZero(pay, p+bipedHeaderBits, mc)
	if !ok {
		return 0, 0, nil, false
	}
	if !i0AbsoluDeLaRegion(pay, i0, lay) {
		return 0, 0, nil, false
	}
	return i0, h.Slot, idx, true
}

// comparerEntetes compare les deux formes à CHAQUE bit de pay et rend le nombre d'acceptations.
func comparerEntetes(t *testing.T, pay []byte, slots SlotBand, gens *GenerationsVivantes, lay profile.I0Layout) int {
	t.Helper()
	total := len(pay) * 8
	acceptes := 0
	minRecord := deltaBipedMinRecord(lay.TotalBits()) // bornes de la marche de production
	for p := 0; p+minRecord <= total; p++ {
		i0, s, idx, ok := matchBipedHeader(pay, p, total, slots, gens, lay)
		ri0, rs, ridx, rok := matchBipedHeaderReference(pay, p, total, slots, gens, lay)
		if ok != rok || i0 != ri0 || s != rs || !reflect.DeepEqual(idx, ridx) {
			t.Fatalf("p=%d : (%d,%d,%v,%v) obtenu, (%d,%d,%v,%v) attendu", p, i0, s, idx, ok, ri0, rs, ridx, rok)
		}
		if ok {
			acceptes++
		}
	}
	return acceptes
}

func TestMatchBipedHeader_EquivalentALaFormeDOrigine_Bobine(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("bobine %s : %v", bobineFamilles, err)
	}
	fc := NewFilmContext(film)
	pos, err := ScanBipedPositions(fc, scanOptWorld())
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	m := map[uint32]bool{}
	for _, p := range pos {
		m[p.Slot] = true
	}
	slots := NewSlotBand(m)
	acceptes, payloads := 0, 0
	for c := 1; c <= bobineChunks; c++ {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			payloads++
			pay := pk.Payload(data)
			acceptes += comparerEntetes(t, pay, slots, fc.GenerationsVivantesA(pk.TimestampUS), cliffLayout)
			acceptes += comparerEntetes(t, pay, slots, nil, cliffLayout)
		}
	}
	if acceptes == 0 {
		t.Fatal("aucune acceptation exercée sur la bobine")
	}
	t.Logf("bobine : %d payloads, %d acceptations", payloads, acceptes)
}

func TestMatchBipedHeader_EquivalentALaFormeDOrigine_Construit(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	layouts := []profile.I0Layout{
		cliffLayout,
		{GateBits: profile.DefaultI0GateBits, AxisW: [3]uint{10, 11, 12}, Region: 0},
		{GateBits: profile.DefaultI0GateBits + 2, AxisW: [3]uint{13, 13, 14}, Region: 1},
	}
	acceptes := 0
	for tour := range 60 {
		lay := layouts[tour%len(layouts)]
		w := &bitWriter{}
		w.bits(rng.Uint64()&0x7FF, rng.IntN(40))
		for range 1 + rng.IntN(12) {
			rec := recordBipedSynthetique{
				Lay: lay, Region: uint64(lay.Region), Slot: uint32(rng.IntN(40)),
				Tag: uint64(rng.IntN(4)), MaskCount: uint64(bipedMinMaskCnt + rng.IntN(bipedMaxMaskCnt-bipedMinMaskCnt+1)),
				Q: [3]uint64{rng.Uint64() >> 40, rng.Uint64() >> 40, rng.Uint64() >> 40},
			}
			if rng.IntN(4) == 0 {
				rec.Region ^= 1 // région refusée
			}
			rec.ecrire(w)
			if rng.IntN(2) == 0 {
				w.bits(rng.Uint64(), 1+rng.IntN(60))
			}
		}
		if tour%5 == 0 { // record coupé : i0 déborde la fin du payload
			writeBipedHeaderEtMasque(w, 3, 1, 3)
		}
		if tour%4 == 1 { // dernier record qui finit exactement à la fin du payload (borne i0+largeur == total)
			const mc = 3
			longueur := bipedHeaderBits + bipedIndexBits*mc + lay.TotalBits()
			w.bits(0, (8-(w.n+longueur)%8)%8)
			recordBipedSynthetique{Lay: lay, Region: uint64(lay.Region), Slot: 5, Tag: 1, MaskCount: mc}.ecrire(w)
		}
		m := map[uint32]bool{}
		for s := range uint32(40) {
			m[s] = rng.IntN(3) != 0
		}
		m[5] = true
		var gens *GenerationsVivantes
		switch tour % 3 {
		case 1:
			gens = ToutesLesGenerations()
		case 2:
			gens = NouvellesGenerationsVivantes([]types.LifeKey{{Slot: 1, Gen: 1}, {Slot: 2, Gen: 2}, {Slot: 3, Gen: 1}})
		}
		acceptes += comparerEntetes(t, w.buf, NewSlotBand(m), gens, lay)
	}
	if acceptes < 100 {
		t.Fatalf("acceptations exercées insuffisantes : %d", acceptes)
	}
	t.Logf("construit : %d acceptations", acceptes)
}

// motsParRecordReference est la forme d'origine de [motsParRecord] : recherche dans le jeu à chaque
// position de bit.
func motsParRecordReference(pay []byte, marches []lecture.Record, wantTI int, jeux ...map[uint32]bool) [][]recordFamilies {
	out := make([][]recordFamilies, len(jeux))
	recs := append([]lecture.Record(nil), marches...)
	if len(recs) == 0 {
		return out
	}
	slices.SortFunc(recs, func(a, b lecture.Record) int { return cmp.Compare(a.Debut, b.Debut) })
	starts := make([]int, len(recs))
	for i, r := range recs {
		starts[i] = int(r.Debut)
	}
	byRec := make([]map[int][]uint32, len(jeux))
	order := make([][]int, len(jeux))
	for k := range jeux {
		byRec[k] = map[int][]uint32{}
	}
	var w uint32
	for b := range len(pay) * 8 {
		w = w<<1 | uint32(source.BitAt(pay, b))
		if b < 31 {
			continue
		}
		for k, jeu := range jeux {
			if !jeu[w] {
				continue
			}
			ri := recordContaining(starts, b-31)
			if ri < 0 || int(recs[ri].TI) != wantTI {
				continue
			}
			if byRec[k][ri] == nil {
				order[k] = append(order[k], ri)
			}
			byRec[k][ri] = append(byRec[k][ri], w)
		}
	}
	for k := range jeux {
		out[k] = make([]recordFamilies, 0, len(order[k]))
		for _, ri := range order[k] {
			out[k] = append(out[k], recordFamilies{Rec: recs[ri], Families: byRec[k][ri]})
		}
	}
	return out
}

// jeuxEchantillonnes tire des jeux de mots du payload (acceptations garanties) et des cas limites :
// jeu nil, vide, d'une clé, de clés partageant des bits, grand jeu, mots absents.
func jeuxEchantillonnes(rng *rand.Rand, pay []byte) []map[uint32]bool {
	mot := func(b int) uint32 { return uint32(source.BitsStricts(pay, b, 32)) }
	tirer := func(n int) map[uint32]bool {
		m := map[uint32]bool{}
		for range n {
			if len(pay)*8 > 40 {
				m[mot(rng.IntN(len(pay)*8-32))] = true
			}
		}
		return m
	}
	jeux := []map[uint32]bool{nil, {}, tirer(1), tirer(5), tirer(60), tirer(400), carrierMarkViews,
		{0xFFFFFFFF: true, 0: true}, {0xDEADBEEF: true}}
	jeux = append(jeux, map[uint32]bool{7: false, mot(0): true}) // clé à valeur fausse : absente du jeu
	return jeux
}

func comparerMots(t *testing.T, pay []byte, recs []lecture.Record, jeux []map[uint32]bool) int {
	t.Helper()
	got := motsParRecord(pay, recs, keyframeBipedTI, jeux...)
	want := motsParRecordReference(pay, recs, keyframeBipedTI, jeux...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorties différentes :\n obtenu  %+v\n attendu %+v", got, want)
	}
	n := 0
	for _, rfs := range want {
		for _, rf := range rfs {
			n += len(rf.Families)
		}
	}
	return n
}

func TestMotsParRecord_EquivalentALaFormeDOrigine_Bobine(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("bobine %s : %v", bobineFamilles, err)
	}
	fc := NewFilmContext(film)
	rng := rand.New(rand.NewPCG(3, 5))
	retenus, payloads := 0, 0
	for c := 1; c <= bobineChunks; c++ {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			recs := recordsDIdentite(WalkKeyframeWorld(pay))
			payloads++
			retenus += comparerMots(t, pay, recs, jeuxEchantillonnes(rng, pay))
		}
	}
	if payloads == 0 || retenus == 0 {
		t.Fatalf("rien d'exercé : %d payloads, %d occurrences retenues", payloads, retenus)
	}
	t.Logf("bobine : %d payloads d'image-clé, %d occurrences retenues", payloads, retenus)
}

func TestMotsParRecord_EquivalentALaFormeDOrigine_Construit(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 17))
	retenus := 0
	for range 40 {
		pay := make([]byte, 200+rng.IntN(3000))
		for i := range pay {
			pay[i] = byte(rng.Uint32())
		}
		var recs []lecture.Record
		for d := 0; d < len(pay)*8-100; d += 100 + rng.IntN(900) {
			ti := keyframeBipedTI
			if rng.IntN(4) == 0 {
				ti = GroundWeaponTypeIndex
			}
			recs = append(recs, lecture.Record{TI: int16(ti), Debut: uint32(d)}) //nolint:gosec // borne par la taille du payload
		}
		retenus += comparerMots(t, pay, recs, jeuxEchantillonnes(rng, pay))
	}
	if retenus < 100 {
		t.Fatalf("occurrences retenues insuffisantes : %d", retenus)
	}
	t.Logf("construit : %d occurrences retenues", retenus)
}

// TestPrefiltreDeMots_AucunFauxNegatif : toute clé du jeu passe le préfiltre, quelle que soit la
// taille du jeu.
func TestPrefiltreDeMots_AucunFauxNegatif(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for _, n := range []int{0, 1, 2, 63, 64, 1000, 5000, 70000} {
		jeu := map[uint32]bool{}
		for range n {
			jeu[rng.Uint32()] = true
		}
		f := nouveauPrefiltreDeMots(jeu)
		for k := range jeu {
			if !f.peutContenir(k) {
				t.Fatalf("n=%d : clé %#x écartée", n, k)
			}
		}
	}
}
