package replay

import (
	"context"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// lectureDArmesTiree : six lectures sur l axe {origine 1 s, pas 100 ms, 10 frames} : une valide en main principale
// seule, une valide aux deux mains, une sans emplacement, une sur un slot sans piste, une avant l origine et une
// apres la fin.
func lectureDArmesTiree() (DrawnWeaponInput, map[uint32]bool, axeDeFrames) {
	axe := axeDeFrames{origin: 1_000_000, step: 100_000, frames: 10}
	reads := []types.DrawnWeaponRead{
		{TimestampUS: 500_000, Slot: 3, Principal: 0, Second: -1},
		{TimestampUS: 1_000_000, Slot: 3, Principal: 0, Second: -1},
		{TimestampUS: 1_250_000, Slot: 3, Principal: 2, Second: 1},
		{TimestampUS: 1_250_000, Slot: 4, Principal: -1, Second: -1},
		{TimestampUS: 1_300_000, Slot: 9, Principal: 1, Second: -1},
		{TimestampUS: 2_000_000, Slot: 3, Principal: 1, Second: -1},
	}
	st := types.DrawnWeaponStats{Scanned: true, Read: len(reads), SlotUnbound: 2, Duplicates: 1}
	return DrawnWeaponInput{Reads: reads, Stats: st}, map[uint32]bool{3: true, 4: true}, axe
}

func ptr(v int) *int { return &v }

// TestLArmeTireeProjetteSurLAxeEtCompteChaqueEcart : l emplacement 0 reste une valeur (pointeur non nil), -1 devient
// absent, les lectures avant l origine, apres la fin ou sur un slot sans piste sont comptees et non publiees, et
// l identite de la couverture tient.
// MUTATIONS : -1 publie tel quel, 0 pris pour absent, garde de piste retiree, borne de fin retiree -> ROUGE.
func TestLArmeTireeProjetteSurLAxeEtCompteChaqueEcart(t *testing.T) {
	in, publies, axe := lectureDArmesTiree()
	got, cov := buildDrawnWeapons(in, publies, axe)
	want := []DrawnWeapon{
		{T: 0, Slot: 3, D: ptr(0)},
		{T: 2, Slot: 3, D: ptr(2), D2: ptr(1)},
		{T: 2, Slot: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lignes %+v, attendu %+v", got, want)
	}
	if cov.Read != 6 || cov.Published != 3 || cov.BeforeOrigin != 1 || cov.NoTrack != 1 || cov.AfterEnd != 1 ||
		cov.SlotUnbound != 2 || cov.Duplicates != 1 {
		t.Errorf("couverture %+v", cov)
	}
	if cov.Read != cov.NoTrack+cov.BeforeOrigin+cov.AfterEnd+cov.Published {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
}

// TestLaPoseDeLArmeTireeSuitLaMarche : sans marche jouee le calque et sa couverture sont absents ; avec elle, le
// calque se publie, vide ou non, et la couverture dit `componentAbsent` quand l archetype ne transmet pas `i42`.
// MUTATION : la pose sans garde de marche -> ROUGE.
func TestLaPoseDeLArmeTireeSuitLaMarche(t *testing.T) {
	in, _, axe := lectureDArmesTiree()
	poser := func(in DrawnWeaponInput) *assemblage {
		a := &assemblage{ctx: context.Background(), origin: axe.origin, step: axe.step}
		a.doc.FrameCount = axe.frames
		a.doc.Coverage = &Coverage{}
		a.doc.Tracks = []Track{{Slot: 3}}
		a.opt.DrawnWeapons = in
		a.poserLesArmesTirees()
		return a
	}
	nonJouee := in
	nonJouee.Stats.Scanned = false
	if a := poser(nonJouee); a.doc.DrawnWeapons != nil || a.doc.Coverage.DrawnWeapons != nil {
		t.Error("marche non jouee : le calque et sa couverture doivent etre absents")
	}
	if a := poser(in); len(a.doc.DrawnWeapons) != 2 || a.doc.Coverage.DrawnWeapons == nil {
		t.Errorf("marche jouee : %d ligne(s), couverture %+v", len(a.doc.DrawnWeapons), a.doc.Coverage.DrawnWeapons)
	}
	absent := DrawnWeaponInput{Stats: types.DrawnWeaponStats{Scanned: true, Absent: true}}
	if a := poser(absent); len(a.doc.DrawnWeapons) != 0 || !a.doc.Coverage.DrawnWeapons.ComponentAbsent {
		t.Errorf("composant absent : couverture %+v", a.doc.Coverage.DrawnWeapons)
	}
}

// TestLesArmesTireesFontLAllerRetourDesFaits : drapeaux, comptes et chaque lecture (instant en delta, slot,
// emplacements dont -1) se relisent a l identique ; une liste vide se relit nil.
func TestLesArmesTireesFontLAllerRetourDesFaits(t *testing.T) {
	in, _, _ := lectureDArmesTiree()
	cas := []DrawnWeaponInput{in, {Stats: types.DrawnWeaponStats{Scanned: true, Absent: true}}, {}}
	for _, c := range cas {
		w := &gwriter{}
		encodeArmesTirees(w, c.Reads, c.Stats)
		r := &greader{b: w.b}
		reads, st := decodeArmesTirees(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("relecture : err %v, %d octet(s) non lus", r.err, len(r.b)-r.off)
		}
		if !reflect.DeepEqual(reads, c.Reads) || st != c.Stats {
			t.Errorf("relu %+v %+v, attendu %+v %+v", reads, st, c.Reads, c.Stats)
		}
	}
}
