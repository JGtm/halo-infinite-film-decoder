package replay

import (
	"context"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func u(v uint32) *uint32 { return &v }

// entreesDeMunitions : deux images-cles et quatre deltas sur l axe {origine 1 s, pas 100 ms, 20 frames}.
func entreesDeMunitions() ([]types.KeyframeInventory, []types.InventoryDelta, map[uint32]bool, axeDeFrames) {
	g := 0.5
	kf := []types.KeyframeInventory{
		// avant l origine : ecartee
		{TimestampUS: 900_000, Slot: 3, AmmoRead: true, Ammo: [types.InventorySlotCount]types.SlotAmmo{{Mag: u(8), Res: u(40)}}},
		// slot 3 : chargeur et reserve de l emplacement 0, jauge de l emplacement 1, rien (plein) pour le reste
		{TimestampUS: 1_000_000, Slot: 3, AmmoRead: true, Ammo: [types.InventorySlotCount]types.SlotAmmo{
			{Mag: u(8), Res: u(40)}, {Gauge: &g}}},
		// slot 9 : sans piste
		{TimestampUS: 1_000_000, Slot: 9, AmmoRead: true, Ammo: [types.InventorySlotCount]types.SlotAmmo{{Mag: u(1)}}},
	}
	deltas := []types.InventoryDelta{
		// un tir : le chargeur baisse, la reserve n est pas transmise
		{TimestampUS: 1_300_000, Slot: 3, Ammo: []types.InventoryDeltaAmmo{{WeaponSlot: 0, Mag: u(7)}}},
		// la meme valeur re-transmise : aucune ligne
		{TimestampUS: 1_400_000, Slot: 3, Ammo: []types.InventoryDeltaAmmo{{WeaponSlot: 0, Mag: u(7)}}},
		// jauge brute 4095 : fraction 1, et un emplacement sans valeur
		{TimestampUS: 1_500_000, Slot: 3, Ammo: []types.InventoryDeltaAmmo{{WeaponSlot: 1, FracQ: u(4095)}, {WeaponSlot: 0}}},
		// apres la fin de l axe
		{TimestampUS: 9_000_000, Slot: 3, Ammo: []types.InventoryDeltaAmmo{{WeaponSlot: 0, Mag: u(0)}}},
	}
	return kf, deltas, map[uint32]bool{3: true}, axeDeFrames{origin: 1_000_000, step: 100_000, frames: 20}
}

// TestLesMunitionsMettentLesDeuxCanauxSurUnAxeEtCompteChaqueEcart : l image-cle d abord a instant egal ; une
// valeur deja connue n est pas republiee ; une lecture partielle ne porte que ses champs ; la jauge est le
// quantum sur 4095 ; hors axe et sans piste se comptent ; l identite de la couverture tient.
// MUTATIONS : jauge sur 4096, comparaison de tous les champs d une lecture partielle, tri sans source,
// garde de piste ou borne de fin retiree -> ROUGE.
func TestLesMunitionsMettentLesDeuxCanauxSurUnAxeEtCompteChaqueEcart(t *testing.T) {
	kf, deltas, publies, axe := entreesDeMunitions()
	got, cov := buildAmmoReads(kf, deltas, false, publies, axe)
	un := float32(1)
	demi := float32(0.5)
	want := []AmmoRead{
		{T: 0, Ms: 0, Slot: 3, W: 0, Mag: u(8), Res: u(40), Src: "kf"},
		{T: 0, Ms: 0, Slot: 3, W: 1, Gauge: &demi, Src: "kf"},
		{T: 3, Ms: 300, Slot: 3, W: 0, Mag: u(7), Src: "delta"},
		{T: 5, Ms: 500, Slot: 3, W: 1, Gauge: &un, Src: "delta"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lignes %+v, attendu %+v", got, want)
	}
	attendu := AmmoReadCoverage{Read: 8, Empty: 2, BeforeOrigin: 1, AfterEnd: 1, NoTrack: 1, Unchanged: 1, FromKeyframe: 2, FromDelta: 2}
	if cov != attendu {
		t.Errorf("couverture %+v, attendu %+v", cov, attendu)
	}
	if cov.Read != cov.BeforeOrigin+cov.AfterEnd+cov.NoTrack+cov.Unchanged+cov.FromKeyframe+cov.FromDelta {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
}

// TestLesMunitionsDeltaRefuseesNeSePublientPas : un canal delta refuse en bloc n entre pas dans la lecture ; les
// images-cles restent et la couverture le dit.
func TestLesMunitionsDeltaRefuseesNeSePublientPas(t *testing.T) {
	kf, deltas, publies, axe := entreesDeMunitions()
	got, cov := buildAmmoReads(kf, deltas, true, publies, axe)
	if len(got) != 2 || cov.FromDelta != 0 || !cov.DeltaRefused || cov.Read != 4 {
		t.Errorf("canal refuse : %d ligne(s), couverture %+v", len(got), cov)
	}
}

// TestLOrdreDesMunitionsNeDependPasDeLOrdreDesEntrees : les memes lectures donnees a l envers rendent le meme
// axe.
func TestLOrdreDesMunitionsNeDependPasDeLOrdreDesEntrees(t *testing.T) {
	kf, deltas, publies, axe := entreesDeMunitions()
	a, _ := buildAmmoReads(kf, deltas, false, publies, axe)
	for i, j := 0, len(kf)-1; i < j; i, j = i+1, j-1 {
		kf[i], kf[j] = kf[j], kf[i]
	}
	for i, j := 0, len(deltas)-1; i < j; i, j = i+1, j-1 {
		deltas[i], deltas[j] = deltas[j], deltas[i]
	}
	b, _ := buildAmmoReads(kf, deltas, false, publies, axe)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("l ordre des entrees change l axe :\n%+v\n%+v", a, b)
	}
}

// TestLaPoseDesMunitionsSuitLInventaire : un inventaire non lu (nil) ne publie ni calque ni couverture ; un
// inventaire lu, meme vide, publie la couverture.
func TestLaPoseDesMunitionsSuitLInventaire(t *testing.T) {
	kf, deltas, _, axe := entreesDeMunitions()
	poser := func(inv []types.KeyframeInventory) *assemblage {
		a := &assemblage{ctx: context.Background(), origin: axe.origin, step: axe.step}
		a.doc.FrameCount = axe.frames
		a.doc.Coverage = &Coverage{}
		a.doc.Tracks = []Track{{Slot: 3}}
		a.opt.Inventory, a.opt.InventoryDeltas = inv, deltas
		a.poserLesMunitions()
		return a
	}
	if a := poser(nil); a.doc.AmmoReads != nil || a.doc.Coverage.AmmoReads != nil {
		t.Error("inventaire non lu : ni calque ni couverture")
	}
	if a := poser(kf); len(a.doc.AmmoReads) != 4 || a.doc.Coverage.AmmoReads == nil {
		t.Errorf("inventaire lu : %d ligne(s)", len(a.doc.AmmoReads))
	}
	if a := poser([]types.KeyframeInventory{}); a.doc.Coverage.AmmoReads == nil || len(a.doc.AmmoReads) != 2 {
		t.Errorf("inventaire lu et vide : %d ligne(s), couverture %+v", len(a.doc.AmmoReads), a.doc.Coverage.AmmoReads)
	}
}
