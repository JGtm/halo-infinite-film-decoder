package grammar

// grenade_life_test.go — LE LANCER PORTE LE HANDLE DU PROJECTILE QU'IL CRÉE (GrenadeThrow.Life).
//
// Le lancer est le record NEW de l'entité projectile : son en-tête écrit le handle (slot, génération)
// avant l'index de type et le motif d'amorce. Le banc est un flux CONSTRUIT, comme celui de
// `grenade_amorce_test.go` : les mini-bobines ne portent aucun paquet delta.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// grenadeNewRecord écrit un en-tête de record NEW complet — pas un delta, `recNew`, le handle — avant
// l'index de type et l'amorce du lancer, tel que le film écrit la création d'un projectile.
func grenadeNewRecord(slot, gen uint32, id, index uint32) []byte {
	a := profile.AmorceGrenadeDeReference()
	w := &bitw{}
	w.put(0, 1) // pas un delta
	w.put(1, 2) // recNew
	w.put(uint64(slot), handleSlotBits)
	w.put(uint64(gen), handleGenBits)
	w.put(uint64(ProjectileTypeIndex)>>bitsTypeIndexBasDuMotif&1, 1)
	w.put(a.MarqueurDe(ProjectileTypeIndex), a.Bits)
	w.put(uint64(id), 32)
	w.pad(a.IndexAuteurBit - a.Bits - 32)
	w.put(uint64(index), profile.AmorceGrenadeIndexBits)
	w.pad(32)
	return w.buf
}

func TestLancerPorteLeHandleDeSonProjectile(t *testing.T) {
	got := scanGrenadesDeReference(grenadeNewRecord(1179, 1, GrenadePlasma, 3))
	if len(got) != 1 {
		t.Fatalf("%d lancer(s) lu(s), attendu 1", len(got))
	}
	if !got[0].HasLife || got[0].Life != (types.LifeKey{Slot: 1179, Gen: 1}) {
		t.Errorf("handle %+v (lu : %v), attendu 1179/1", got[0].Life, got[0].HasLife)
	}
	if got[0].FilmIndex != 3 {
		t.Errorf("auteur %d, attendu 3 : la lecture du handle ne doit pas déplacer celle de l'auteur", got[0].FilmIndex)
	}
}

// TestLancerSansEnTeteNAPasDeHandle : un motif trop près du début du payload pour qu'un en-tête le
// précède ne fait pas inventer un handle.
func TestLancerSansEnTeteNAPasDeHandle(t *testing.T) {
	got := scanGrenadesDeReference(buildGrenadeRecord(5, GrenadePlasma, 3))
	if len(got) != 1 || got[0].HasLife {
		t.Fatalf("lancers %+v : un lancer sans handle lu est attendu (son en-tête commencerait avant le payload)", got)
	}
}

// TestLeHandleNeChangeAucuneAutreLecture : les champs qui existaient avant le handle se lisent à
// l'identique, au même bit, sur les quatre rangs de grenade.
func TestLeHandleNeChangeAucuneAutreLecture(t *testing.T) {
	for rang, id := range GrenadeTypeIDsByRank() {
		got := scanGrenadesDeReference(grenadeNewRecord(1100+uint32(rang), 2, id, uint32(7+rang)))
		if len(got) != 1 {
			t.Fatalf("rang %d : %d lancer(s)", rang, len(got))
		}
		if r, connu := got[0].Rank(); !connu || r != rang || got[0].FilmIndex != 7+rang || got[0].TypeID != id {
			t.Errorf("rang %d : %+v", rang, got[0])
		}
		if got[0].Life != (types.LifeKey{Slot: 1100 + uint32(rang), Gen: 2}) {
			t.Errorf("rang %d : handle %+v", rang, got[0].Life)
		}
	}
}
