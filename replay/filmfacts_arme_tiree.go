package replay

import "github.com/JGtm/halo-infinite-film-decoder/types"

// filmfacts_arme_tiree.go : LE JEU D ARMES DESIRE DANS LE COMPLEMENT DE LA SECTION 1 : les deux
// drapeaux et les comptes du canal, puis chaque lecture (horodatage en delta, slot, emplacement
// principal et emplacement de seconde main, entiers signes : -1 = aucun). Une liste vide se relit nil.

// coutMinimalArmeTiree : horodatage, slot, deux emplacements, un octet chacun au moins.
const coutMinimalArmeTiree = 4

func encodeArmesTirees(w *gwriter, rs []types.DrawnWeaponRead, st types.DrawnWeaponStats) {
	w.bool8(st.Scanned)
	w.bool8(st.Absent)
	for _, v := range []int{st.Read, st.SlotUnbound, st.Duplicates} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(rs)))
	var dernier uint64
	for _, m := range rs {
		w.u(m.TimestampUS - dernier)
		dernier = m.TimestampUS
		w.u(uint64(m.Slot))
		w.i(int64(m.Principal))
		w.i(int64(m.Second))
	}
}

func decodeArmesTirees(r *greader) ([]types.DrawnWeaponRead, types.DrawnWeaponStats) {
	var st types.DrawnWeaponStats
	st.Scanned, st.Absent = r.bool8(), r.bool8()
	for _, v := range []*int{&st.Read, &st.SlotUnbound, &st.Duplicates} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(coutMinimalArmeTiree)
	var out []types.DrawnWeaponRead
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		out = append(out, types.DrawnWeaponRead{TimestampUS: dernier, Slot: uint32(r.u()), //nolint:gosec // slot ecrit tel quel
			Principal: int8(r.i()), Second: int8(r.i())}) //nolint:gosec // emplacements -1 a 3
	}
	return out, st
}
