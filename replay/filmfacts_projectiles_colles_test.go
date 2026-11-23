package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLesProjectilesCollesFontLAllerRetourDesFaits : le drapeau de lecture, les comptes du canal et chaque champ d une lecture
// (horodatage en delta, projectile, parent, noeud, les trois valeurs brutes dont les bornes 0 et 65 535, drapeau de creation)
// se relisent a l identique ; une liste vide se relit nil.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : une valeur de pose perdue ou echangee avec une autre, le drapeau de creation, la
// generation du parent ou du createur, un drapeau, le tag, l ecart a la creation, un compte du canal.
func TestLesProjectilesCollesFontLAllerRetourDesFaits(t *testing.T) {
	cas := []struct {
		nom   string
		lus   []types.ProjectileAttachRead
		stats types.ProjectileAttachStats
	}{
		{"non lu", nil, types.ProjectileAttachStats{}},
		{"absent", nil, types.ProjectileAttachStats{Scanned: true, Absent: true}},
		{"lu, aucun collage", nil, types.ProjectileAttachStats{Scanned: true, Frames: 9000, Records: 12}},
		{"collages", []types.ProjectileAttachRead{
			{TimestampUS: 4_517_903_000, Life: types.LifeKey{Slot: 1133, Gen: 1}, Parent: types.LifeKey{Slot: 561, Gen: 1}, Node: 22,
				Pose: [3]uint32{32214, 33936, 33041}, BornUS: 4_517_800_000, HasBirth: true, Tag: 0x580b8831, HasTag: true,
				Creator: types.LifeKey{Slot: 512, Gen: 1}, HasCreator: true},
			{TimestampUS: 4_517_903_000, Life: types.LifeKey{Slot: 1200, Gen: 3}, Parent: types.LifeKey{Slot: 8191 + 512, Gen: 3},
				Node: 65535, Pose: [3]uint32{0, 65535, 1}, Created: true, BornUS: 4_517_903_000, HasBirth: true, Tag: 0xffffffff, HasTag: true},
			{TimestampUS: 4_700_000_000, Life: types.LifeKey{Slot: 1133, Gen: 1}, Parent: types.LifeKey{Slot: 700, Gen: 2}, Node: 0},
		}, types.ProjectileAttachStats{Scanned: true, Frames: 9000, UnreadFrames: 4, Records: 70, Read: 3, Created: 1, Desynced: 2,
			Repeats: 1, NoBirth: 1, NoTag: 1, NoCreator: 2, WideLocations: 5}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			w := &gwriter{}
			encodeProjectilesColles(w, c.lus, c.stats)
			r := &greader{b: w.b}
			lus, stats := decodeProjectilesColles(r)
			if r.err != nil || r.off != len(r.b) {
				t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
			}
			if !reflect.DeepEqual(lus, c.lus) || stats != c.stats {
				t.Errorf("relu %+v %+v, attendu %+v %+v", lus, stats, c.lus, c.stats)
			}
		})
	}
}

// TestLeCodecDesProjectilesCollesRefuseUnCompteImpossible : un compte de lectures que les octets restants ne peuvent pas porter
// est refuse, pas alloue.
func TestLeCodecDesProjectilesCollesRefuseUnCompteImpossible(t *testing.T) {
	w := &gwriter{}
	w.bool8(true)
	w.bool8(false)
	for range 11 {
		w.u(0)
	}
	w.u(1 << 40)
	r := &greader{b: w.b}
	decodeProjectilesColles(r)
	if r.err == nil {
		t.Error("un compte de 2^40 lectures sur quelques octets doit etre refuse")
	}
}
