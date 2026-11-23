package grammar

// equipment_recovery_ordre_test.go — la tete de record lue d un tenant par
// [scanEquipRecoveryPacket] retient EXACTEMENT les candidats de la forme d origine, qui lisait le
// prefixe, puis le handle, puis la fenetre, puis la generation et le bit 16.

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// scanEquipRecoveryPacketReference est la forme d origine du balayage d un paquet, gardee comme
// reference : un handle lu a chaque position dont le prefixe vaut 1, les tests de generation et de
// bit 16 apres la recherche de la fenetre.
func scanEquipRecoveryPacketReference(
	s abilityScanSetup, pay []byte, wins []*equipRecoveryWindow, chunk int, pk FilmPacket,
	last *struct {
		counter uint32
		rank    int
		got     bool
	},
) {
	var active []*equipRecoveryWindow
	for _, w := range wins {
		if w.tsMin <= pk.TimestampUS && pk.TimestampUS <= w.tsMax {
			active = append(active, w)
		}
	}
	if len(active) == 0 {
		return
	}
	total := len(pay) * 8
	gens := s.fc.GenerationsVivantesA(pk.TimestampUS)
	for p := 0; p+bipedHeaderBits+bipedIndexBits <= total; p++ {
		if uint32(source.BitsStricts(pay, p, 1)) != 1 {
			continue
		}
		h := LireHandleDelta(pay, p)
		var w *equipRecoveryWindow
		for _, cand := range active {
			if cand.slot == h.Slot && cand.gen == h.Gen {
				w = cand
				break
			}
		}
		if w == nil {
			continue
		}
		if !gens.Accepte(h) || uint32(source.BitsStricts(pay, p+16, 1)) != 0 {
			continue
		}
		counter, rank, ok := walkEquipRecoveryAt(s, pay, p, total, last)
		if !ok {
			continue
		}
		w.cands = append(w.cands, equipRecovered{
			abilityEmission: abilityEmission{
				Slot: h.Slot, Gen: h.Gen, Chunk: chunk, PacketIndex: pk.Index,
				TimestampUS: pk.TimestampUS, Counter: counter, Rank: rank,
			},
			off: p, dense: uint32(source.BitsStricts(pay, p+17, 1)) == 1,
		})
	}
}

// candidatsDesDeuxFormes balaye `pay` avec les deux formes sur des fenetres neuves et rend les
// candidats de chacune, fenetre par fenetre, ainsi que leur nombre total.
func candidatsDesDeuxFormes(s abilityScanSetup, pay []byte, cles [][2]uint32, chunk int) (ancien, nouveau [][]equipRecovered, n int) {
	mk := func() []*equipRecoveryWindow {
		var ws []*equipRecoveryWindow
		for _, k := range cles {
			ws = append(ws, &equipRecoveryWindow{slot: k[0], gen: k[1], tsMax: ^uint64(0)})
		}
		return ws
	}
	pk := FilmPacket{Index: 3, TimestampUS: 1 << 40}
	wa, wn := mk(), mk()
	last, obs := equipRecoveryHook()
	s.gram.obs = obs
	scanEquipRecoveryPacketReference(s, pay, wa, chunk, pk, last)
	last, obs = equipRecoveryHook()
	s.gram.obs = obs
	scanEquipRecoveryPacket(s, pay, wn, chunk, pk, last)
	for i := range wa {
		ancien = append(ancien, wa[i].cands)
		nouveau = append(nouveau, wn[i].cands)
		n += len(wn[i].cands)
	}
	return ancien, nouveau, n
}

func TestRecuperationEquipement_TeteDUnTenant_MemesCandidats(t *testing.T) {
	film := chargerBobineFamilles(t)
	s, err := resolveAbilityScan(NewFilmContext(film))
	if err != nil {
		t.Fatalf("balayage i48 : %v", err)
	}
	r := recordI48DuSlot(t, s)
	retenus := 0

	// Payloads construits : la forme dense d un vrai record i48, sous chaque generation, avec le bit 16
	// intact puis leve, et la porte levee puis baissee.
	for gen := uint32(0); gen < 4; gen++ {
		for _, bit16 := range []bool{false, true} {
			for _, porte := range []bool{true, false} {
				pay := enFormeDense(r, gen)
				if bit16 {
					ecrisBits(pay, 16, 1, 1)
				}
				if !porte {
					pay[17/8] &^= 1 << (7 - 17%8)
				}
				cles := [][2]uint32{{r.Slot, 1}, {r.Slot, 2}, {r.Slot, gen}}
				a, n, nb := candidatsDesDeuxFormes(s, pay, cles, r.Chunk)
				retenus += nb
				if !reflect.DeepEqual(a, n) {
					t.Errorf("gen=%d bit16=%v porte=%v : ancien %+v, nouveau %+v", gen, bit16, porte, a, n)
				}
			}
		}
	}

	// Payloads aleatoires : les fenetres sont prises aux handles lus dans le payload meme, donc la
	// recherche de fenetre reussit a de nombreuses positions.
	rng := rand.New(rand.NewSource(7))
	for range 300 {
		pay := make([]byte, 40+rng.Intn(160))
		for i := range pay {
			pay[i] = byte(rng.Intn(256))
		}
		var cles [][2]uint32
		for range 40 {
			h := LireHandleDelta(pay, rng.Intn(len(pay)*8-40))
			cles = append(cles, [2]uint32{h.Slot, h.Gen})
		}
		a, n, nb := candidatsDesDeuxFormes(s, pay, cles, r.Chunk)
		retenus += nb
		if !reflect.DeepEqual(a, n) {
			t.Fatalf("payload aleatoire : ancien %+v, nouveau %+v", a, n)
		}
	}

	// Paquets reels de la mini-bobine : toutes les vies du slot de la chaine i48.
	cles := [][2]uint32{{slotChaineI48, 0}, {slotChaineI48, 1}, {slotChaineI48, 2}, {slotChaineI48, 3}}
	vus := 0
	for _, c := range s.fc.ChunkNumbers() {
		data, pks, ok := s.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			a, n, nb := candidatsDesDeuxFormes(s, pk.Payload(data), cles, c)
			vus++
			retenus += nb
			if !reflect.DeepEqual(a, n) {
				t.Fatalf("chunk %d paquet %d : ancien %+v, nouveau %+v", c, pk.Index, a, n)
			}
		}
	}
	if vus == 0 {
		t.Fatal("aucun paquet delta de la mini-bobine : le test ne prouverait rien")
	}
	if retenus == 0 {
		t.Fatal("aucun candidat retenu par aucune des deux formes : le test ne prouverait rien")
	}
	t.Logf("%d paquets reels, %d candidats retenus au total", vus, retenus)
}

// BenchmarkRecuperationEquipement compare la forme d origine et la forme courante sur les paquets
// delta de la mini-bobine.
func BenchmarkRecuperationEquipement(b *testing.B) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		b.Skip(err)
	}
	s, err := resolveAbilityScan(NewFilmContext(film))
	if err != nil {
		b.Fatal(err)
	}
	type paquet struct {
		pay   []byte
		pk    FilmPacket
		chunk int
	}
	var pqs []paquet
	for _, c := range s.fc.ChunkNumbers() {
		data, pks, ok := s.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type == PacketTypeDelta {
				pqs = append(pqs, paquet{pk.Payload(data), pk, c})
			}
		}
	}
	for _, f := range []struct {
		nom string
		fn  func(abilityScanSetup, []byte, []*equipRecoveryWindow, int, FilmPacket, *struct {
			counter uint32
			rank    int
			got     bool
		})
	}{{"origine", scanEquipRecoveryPacketReference}, {"courante", scanEquipRecoveryPacket}} {
		b.Run(f.nom, func(b *testing.B) {
			last, obs := equipRecoveryHook()
			s.gram.obs = obs
			for range b.N {
				for _, q := range pqs {
					ws := make([]equipRecoveryWindow, 24)
					ps := make([]*equipRecoveryWindow, len(ws))
					for i := range ws {
						ws[i] = equipRecoveryWindow{slot: slotChaineI48 + uint32(i), gen: 1, tsMax: ^uint64(0)}
						ps[i] = &ws[i]
					}
					f.fn(s, q.pay, ps, q.chunk, q.pk, last)
				}
			}
		})
	}
}
