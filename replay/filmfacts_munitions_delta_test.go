package replay

// Tests — LES MUNITIONS DES DELTAS D'INVENTAIRE DANS LE FICHIER DE FAITS.
//
// Un delta transmet, par emplacement, un chargeur, une fraction R(12) brute et une reserve,
// chacun optionnel. Le fichier de faits doit les rendre a l'identique, et surtout garder la
// difference entre « non transmis » (nil) et « transmis a zero ».

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func u32(v uint32) *uint32 { return &v }

// deltasTemoins : un delta sans munition, un delta aux trois champs dont un a zero, un delta
// de deux emplacements aux champs partiels.
func deltasTemoins() []types.InventoryDelta {
	return []types.InventoryDelta{
		{TimestampUS: 1_000_000, Slot: 7, Grenades: []uint32{1, 0, 2}, SelRead: true, Sel: 1, Mask: 3},
		{TimestampUS: 1_500_000, Slot: 7, Ammo: []types.InventoryDeltaAmmo{
			{WeaponSlot: 0, Mag: u32(0), FracQ: u32(4095), Res: u32(120)},
		}},
		{TimestampUS: 2_000_000, Slot: 9, Ammo: []types.InventoryDeltaAmmo{
			{WeaponSlot: 1, Mag: u32(8)},
			{WeaponSlot: 0, FracQ: u32(17), Res: u32(0)},
		}},
	}
}

// TestFaits_MunitionsDesDeltasFontLAllerRetour : tous les champs reviennent, nil reste nil, zero
// reste zero, et un delta sans munition relit `Ammo` nil.
// MUTATIONS — champ Res ou FracQ non ecrit, presence ecrite a l'envers, ordre des champs permute,
// nil relu comme zero : ROUGES.
func TestFaits_MunitionsDesDeltasFontLAllerRetour(t *testing.T) {
	g := &FilmFacts{}
	g.InventoryDeltas = deltasTemoins()
	w := &gwriter{}
	encodeInventaire(w, g)

	relu := &FilmFacts{}
	r := &greader{b: w.b}
	decodeInventaire(r, relu)
	if r.err != nil {
		t.Fatalf("relecture : %v", r.err)
	}
	if r.off != len(r.b) {
		t.Errorf("%d octet(s) non lus sur %d : flux desynchronise", len(r.b)-r.off, len(r.b))
	}
	if !reflect.DeepEqual(relu.InventoryDeltas, g.InventoryDeltas) {
		t.Errorf("deltas relus %+v, attendu %+v", relu.InventoryDeltas, g.InventoryDeltas)
	}
	if relu.InventoryDeltas[0].Ammo != nil {
		t.Error("un delta sans munition doit relire Ammo nil")
	}
	if a := relu.InventoryDeltas[2].Ammo[0]; a.FracQ != nil || a.Res != nil || a.Mag == nil || *a.Mag != 8 {
		t.Errorf("champs partiels mal relus : %+v", a)
	}
}

// TestFaits_MunitionsDesDeltasTronquesRendentUneErreur : un flux coupe au milieu des munitions
// est une erreur, jamais un delta sans munition.
func TestFaits_MunitionsDesDeltasTronquesRendentUneErreur(t *testing.T) {
	g := &FilmFacts{}
	g.InventoryDeltas = deltasTemoins()
	w := &gwriter{}
	encodeInventaire(w, g)
	for coupe := 1; coupe <= 3; coupe++ {
		r := &greader{b: w.b[:len(w.b)-coupe]}
		decodeInventaire(r, &FilmFacts{})
		if r.err == nil {
			t.Errorf("flux coupe de %d octet(s) relu sans erreur", coupe)
		}
	}
}
