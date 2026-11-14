package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func marcheAuPaquet(ts uint64, chunk, index int) *MarcheDistribuee {
	return &MarcheDistribuee{Paquet: &lecture.Paquet{TS: ts, Chunk: chunk, Index: index}}
}

// TestControleDesUnitesLitLaTete : la porte fermee ne donne aucun controleur ; la porte ouverte donne
// l index, et le second index quand sa porte l est aussi.
func TestControleDesUnitesLitLaTete(t *testing.T) {
	var c controleDesUnites
	m := marcheAuPaquet(1000, 2, 7)
	c.recevoir(m, 513, []uint64{0, 0, 0, 0})
	c.recevoir(m, 514, []uint64{1, 9, 0, 0})
	c.recevoir(m, 515, []uint64{1, 31, 1, 40})
	got := c.attente
	want := []types.UnitControlRead{
		{Slot: 513, TimestampUS: 1000, Chunk: 2, PacketIndex: 7, Controller: -1},
		{Slot: 514, TimestampUS: 1000, Chunk: 2, PacketIndex: 7, Controller: 9},
		{Slot: 515, TimestampUS: 1000, Chunk: 2, PacketIndex: 7, Controller: 31},
	}
	if len(got) != len(want) {
		t.Fatalf("lectures en attente = %d, attendu %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("lecture %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
}

// TestControleDesUnitesSoldeLaTrame : un slot qui n est pas un bipede est ecarte, une
// lecture re-publiee par le chemin d inference n est gardee qu une fois, et l attente est videe.
func TestControleDesUnitesSoldeLaTrame(t *testing.T) {
	var c controleDesUnites
	m := marcheAuPaquet(500, 1, 3)
	c.recevoir(m, 513, []uint64{1, 4, 0, 0})
	c.recevoir(m, 513, []uint64{1, 4, 0, 0})
	c.recevoir(m, 900, []uint64{1, 5, 0, 0})
	c.solder(func(slot uint32) bool { return slot == 513 })
	if len(c.lus) != 1 || c.lus[0].Slot != 513 {
		t.Fatalf("lectures gardees = %+v, attendu la seule lecture du bipede, une fois", c.lus)
	}
	if len(c.attente) != 0 {
		t.Fatalf("l attente doit etre videe, %d restantes", len(c.attente))
	}
	m2 := marcheAuPaquet(600, 1, 4)
	c.recevoir(m2, 513, []uint64{0, 0, 0, 0})
	c.solder(func(uint32) bool { return true })
	pub := c.publier()
	if len(pub) != 2 || pub[0].TimestampUS != 500 || pub[1].Controller != -1 {
		t.Fatalf("publication = %+v, attendu deux lectures triees par instant", pub)
	}
}
