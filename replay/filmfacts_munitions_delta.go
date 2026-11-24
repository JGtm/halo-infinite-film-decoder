package replay

// filmfacts_munitions_delta.go : les munitions de chaque delta d'inventaire dans le fichier de
// faits.
//
// La grammaire lit deja l'etat de munitions que transmet un delta ([types.InventoryDeltaAmmo] :
// chargeur, fraction brute R(12), reserve), mais le fichier de faits ne gardait que les
// grenades, la selection et le masque : une publication rejouee depuis les faits perdait ces
// munitions. Elles suivent maintenant, a la fin de chaque delta, dans l'ordre du film.
//
// Format d'un delta, apres le masque : le nombre d'emplacements transmis (varint), puis pour
// chacun le rang d'emplacement (entier signe) et trois champs optionnels, dans l'ordre chargeur,
// fraction R(12) BRUTE, reserve, chacun precede d'un octet de presence. Une valeur absente reste
// nil au decodage : « non transmis » n'est jamais lu comme zero.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// coutMinimalDeltaInventaire est le nombre minimal d'octets d'un delta d'inventaire dans le
// fichier : horodatage, slot, nombre de grenades, SelRead, Sel, Mask et nombre de munitions, un
// octet chacun au moins. Il borne le compte lu par [greader.compte].
const coutMinimalDeltaInventaire = 7

// coutMinimalMunitionDelta : rang d'emplacement et trois octets de presence.
const coutMinimalMunitionDelta = 4

// encodeMunitionsDelta ecrit les munitions d'un delta d'inventaire.
func encodeMunitionsDelta(w *gwriter, ammo []types.InventoryDeltaAmmo) {
	w.u(uint64(len(ammo)))
	for _, a := range ammo {
		w.i(int64(a.WeaponSlot))
		for _, v := range [...]*uint32{a.Mag, a.FracQ, a.Res} {
			w.bool8(v != nil)
			if v != nil {
				w.u(uint64(*v))
			}
		}
	}
}

// decodeMunitionsDelta relit les munitions d'un delta d'inventaire ; nil quand il n'y en a pas.
func decodeMunitionsDelta(r *greader) []types.InventoryDeltaAmmo {
	n := r.compte(coutMinimalMunitionDelta)
	if n == 0 {
		return nil
	}
	out := make([]types.InventoryDeltaAmmo, 0, n)
	for j := 0; j < n && r.err == nil; j++ {
		a := types.InventoryDeltaAmmo{WeaponSlot: int(r.i())}
		for _, v := range [...]**uint32{&a.Mag, &a.FracQ, &a.Res} {
			if r.bool8() {
				x := uint32(r.u())
				*v = &x
			}
		}
		out = append(out, a)
	}
	return out
}
