package grammar

// vehicle_creation_parent.go — LE PARENT D'UN VEHICULE, LU A SA CREATION.
//
// Une piece montee (tourelle d'un Warthog, du Falcon, du Wraith, du Scorpion) est creee comme un objet
// `ti=40` a part entiere ; quand le film la rattache a son chassis, le masque de son record de creation
// annonce `i10 object-parent-state` (et non `i40`), dont la branche « attache » nomme le parent par son
// handle : la categorie 1 de `FUN_1406d3140`, de base [parentHandleBase], les deux bits de queue etant la
// generation ([parentHandleValueMask]). Le meme composant nomme le parent d'un bipede monte a bord
// ([occupancyFromRecord]) et d'un equipement porte.
//
// LA LECTURE EST CELLE DU RECORD, PAS UNE AUTRE MARCHE : la boucle de composants de production, depuis le
// premier composant du masque, jusqu'a i10 (la meme que [readGroundWeaponAmmo] pour les munitions), avec son
// propre curseur. Elle ne change aucun bit du chemin de creation.
//
// CE QU'ELLE NE DIT PAS. Un record qui n'annonce pas i10, qui ne l'atteint pas (composant non porte devant
// lui) ou qui le lit detache ne nomme aucun parent : l'appelant n'a alors AUCUNE lecture, et une lecture
// absente n'est pas une lecture « sans parent ». Les films du plus ancien format ne rattachent pas leurs
// pieces par i10.

import (
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// readCreationParent rend le parent attache que le composant i10 du record de creation nomme, lu dans la
// boucle de composants de `arch` a partir de `compStart` sous le masque `mask` ; faux sans lecture.
func readCreationParent(pay []byte, compStart int, mask []int, arch Archetype, prof ProfilDeBalayage) (types.LifeKey, bool) {
	var bits uint64
	hasParent := false
	for _, i := range mask {
		if arch.component(i) == compObjectParentState {
			hasParent = true
		}
		bits |= uint64(1) << uint(i&63)
	}
	if !hasParent {
		return types.LifeKey{}, false
	}
	br := LecteurSur(pay)
	br.PoserContexte(ContexteDeLecture{Profil: prof})
	br.SetBitPos(compStart)
	tr := EntityTrace{TypeIndex: VehicleTypeIndex, Mask: bits, DesyncAt: -1}
	traverseComponentLoop(br, arch, &tr)
	for _, c := range tr.Comps {
		st, ok := c.ParentOf()
		if !ok || !c.Ported || !st.Attached {
			continue
		}
		return types.LifeKey{Slot: (st.Quant16 & parentHandleValueMask) + parentHandleBase, Gen: st.Quant16 >> 30}, true
	}
	return types.LifeKey{}, false
}
