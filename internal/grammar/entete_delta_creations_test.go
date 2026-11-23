package grammar

// entete_delta_creations_test.go — LA PASSE DES CREATIONS ([passeDesCreations.payload]) ET LE
// PREFIXE DE RECORD NEW QU'ELLE CHERCHE ([prochainPrefixeNEW]), CONTRE LEUR FORME POSITION PAR
// POSITION.

import (
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func prochainPrefixeNEWReference(pay []byte, p, limite int) int {
	for q := p; q < limite; q++ {
		if source.BitsTolerants(pay, q, 3) == 1 {
			return q
		}
	}
	return limite
}

// rechercheNEWAvecDefauts est [prochainPrefixeNEW] recopiee avec un defaut au choix.
func rechercheNEWAvecDefauts(pasEnPlus int, sansBorneBasse bool) func(pay []byte, p, limite int) int {
	return func(pay []byte, p, limite int) int {
		for p < limite {
			i := p >> 3
			w := source.BitsAt(pay, i<<3, 64)
			c := ^w & ^(w << 1) & (w << 2)
			if !sansBorneBasse {
				c &= ^uint64(0) >> uint(p&7)
			}
			c &^= 1<<dernierBitDuPrefixeNEW - 1
			if c != 0 {
				return min(i<<3+bits.LeadingZeros64(c), limite)
			}
			p = i<<3 + 64 - dernierBitDuPrefixeNEW + pasEnPlus
		}
		return limite
	}
}

func divergeLaRechercheNEW(f func(pay []byte, p, limite int) int) (divergences, trouvees int) {
	rng := rand.New(rand.NewPCG(41, 2))
	for _, pay := range payloadsDuTestDeLaRecherche(rng) {
		total := len(pay) * 8
		for p := 0; p <= total+3; p++ {
			for _, limite := range []int{p, p + 1, p + 3, p + 30, p + 61, p + 62, p + 63, total, total + 9} {
				want := prochainPrefixeNEWReference(pay, p, limite)
				if f(pay, p, limite) != want {
					divergences++
				}
				if want < limite {
					trouvees++
				}
			}
		}
	}
	return divergences, trouvees
}

func TestProchainPrefixeNEWEgaleLaRechercheParPosition(t *testing.T) {
	d, trouvees := divergeLaRechercheNEW(prochainPrefixeNEW)
	if d != 0 || trouvees == 0 {
		t.Fatalf("%d divergences, %d positions trouvees", d, trouvees)
	}
	t.Logf("%d recherches qui trouvent une position, aucune divergence", trouvees)
}

func TestProchainPrefixeNEWDetecteSesMutants(t *testing.T) {
	if d, _ := divergeLaRechercheNEW(rechercheNEWAvecDefauts(0, false)); d != 0 {
		t.Fatalf("la copie sans defaut diverge %d fois", d)
	}
	if d, _ := divergeLaRechercheNEW(rechercheNEWAvecDefauts(3, false)); d == 0 {
		t.Error("le mutant du bord de mot (positions sautees) n'est pas detecte")
	}
	if d, _ := divergeLaRechercheNEW(rechercheNEWAvecDefauts(0, true)); d == 0 {
		t.Error("le mutant sans borne basse n'est pas detecte")
	}
}

// payloadReference est l'ancienne boucle de [passeDesCreations.payload].
func (ps *passeDesCreations) payloadReference(pay []byte, lieu lieuDuPaquet, fin int) {
	clear(ps.curseurs)
	total := len(pay) * 8
	for p := 0; p <= total-woNewHeaderBits && p < fin; p++ {
		ti, ok := archetypeDeLEnTeteNEW(pay, p)
		if !ok {
			continue
		}
		for k, w := range ps.marches {
			if ps.curseurs[k] > p || w.archetype() != ti {
				continue
			}
			slot, gen, ok := matchWorldObjectNewHeader(pay, p, w.band, ti)
			if !ok {
				continue
			}
			cre, ok := w.creationA(pay, p, types.LifeKey{Slot: slot, Gen: gen}, lieu, &ps.sts[k])
			if !ok {
				continue
			}
			ps.outs[k] = append(ps.outs[k], cre)
			ps.curseurs[k] = cre.AfterBit
		}
	}
}

// marchesDeLaBobine rend les marches de creation des archetypes de la bobine.
func marchesDeLaBobine(t *testing.T, fc *FilmContext, wr profile.Vec3Range) []equipCreationWalk {
	t.Helper()
	var marches []equipCreationWalk
	for _, ti := range archetypesDeCreation() {
		band := worldObjectSlotBand(fc, int(ti))
		if len(band) == 0 {
			continue
		}
		w, err := fc.marcheDeCreation(ti, &wr, band)
		if err != nil {
			t.Fatalf("marche de creation ti=%d : %v", ti, err)
		}
		marches = append(marches, w)
	}
	if len(marches) < 2 {
		t.Fatalf("%d archetype(s) de creation sur la bobine", len(marches))
	}
	return marches
}

// comparerLesCreations joue les deux formes de la passe sur `pay` et rend les creations de la
// reference.
func comparerLesCreations(t *testing.T, marches []equipCreationWalk, pay []byte, lieu lieuDuPaquet, fin int) []types.EquipmentCreation {
	t.Helper()
	nouvelle, reference := nouvellePasseDesCreations(marches), nouvellePasseDesCreations(marches)
	nouvelle.payload(pay, lieu, fin)
	reference.payloadReference(pay, lieu, fin)
	if !reflect.DeepEqual(nouvelle.outs, reference.outs) || !reflect.DeepEqual(nouvelle.sts, reference.sts) {
		t.Fatalf("chunk %d paquet %d len=%d fin=%d : creations ou comptes differents", lieu.chunk, lieu.pk.Index, len(pay), fin)
	}
	var out []types.EquipmentCreation
	for _, o := range reference.outs {
		out = append(out, o...)
	}
	return out
}

// TestPasseDesCreationsEgaleSaFormeSurLaMiniBobine : sur chaque payload delta de la bobine, ses
// decalages, ses troncatures et ses precedes de zeros, la passe des archetypes de creation rend les
// memes creations et les memes comptes que la forme position par position. Les creations acceptees
// sont ensuite replantees seules, a chaque position de 0 a 135 bits d'un tampon nul.
func TestPasseDesCreationsEgaleSaFormeSurLaMiniBobine(t *testing.T) {
	fc, wr := contexteDesObjetsDuMonde(t)
	marches := marchesDeLaBobine(t, fc, wr)
	rng := rand.New(rand.NewPCG(51, 6))
	payloads, creations, bord := 0, 0, 0
	var graines []struct {
		bits []byte
		n    int
		lieu lieuDuPaquet
	}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			lieu := lieuDuPaquet{chunk: c, pk: pk}
			for _, pay := range variantesDeBord(rng, pk.Payload(data)) {
				total := len(pay) * 8
				for _, fin := range []int{total, total / 2, 7, 0} {
					creations += len(comparerLesCreations(t, marches, pay, lieu, fin))
				}
				payloads++
			}
			if len(graines) < 40 {
				pay := pk.Payload(data)
				for _, cre := range comparerLesCreations(t, marches, pay, lieu, len(pay)*8) {
					w := &bitWriter{}
					for k := cre.BitPos; k < cre.AfterBit; k++ {
						w.bit(uint64(pay[k>>3] >> (7 - uint(k&7)) & 1))
					}
					graines = append(graines, struct {
						bits []byte
						n    int
						lieu lieuDuPaquet
					}{w.buf, w.n, lieu})
				}
			}
		}
	}
	for _, g := range graines {
		w := &bitWriter{buf: g.bits, n: g.n}
		for off := range 136 {
			juste := (off + g.n + 7) / 8
			for _, longueur := range []int{juste, juste - 1, juste + 1} {
				pay := make([]byte, longueur)
				poserLesBits(pay, off, w)
				for _, fin := range []int{longueur * 8, off, off + 1} {
					bord += len(comparerLesCreations(t, marches, pay, g.lieu, fin))
				}
			}
		}
	}
	if creations == 0 || bord == 0 {
		t.Fatalf("creations exercees : %d sur la bobine, %d aux bords", creations, bord)
	}
	t.Logf("%d payloads (variantes comprises), %d creations acceptees ; %d creations replantees, %d acceptations aux bords",
		payloads, creations, len(graines), bord)
}
