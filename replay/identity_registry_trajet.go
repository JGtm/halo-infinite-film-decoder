package replay

// identity_registry_trajet.go — LE JOUEUR QUI PORTE UN TRAJET DE VEHICULE.
//
// Un trajet s'ouvre quand le corps d'un occupant cesse de repliquer sa position : le bipede qui
// monte a bord s'arrete, et sa vie se clot avant l'instant de la montee. A cet instant AUCUNE vie
// ne couvre le slot, et la reponse « qui tient ce slot » ([IdentityRegistry.XUIDAt]) tomberait sur
// le pont par slot, donc sur le PREMIER occupant nomme du siege : un siege qu'un bot puis un
// humain se partagent donnerait le nom de l'humain au trajet du bot.
//
// Le film dit pourtant quel corps monte : le record de creation date les corps successifs du slot
// ([corpsLu.recordA]). Le trajet porte l'identite de CE corps, jamais celle d'un autre occupant du
// siege.

import (
	"strconv"
)

// XUIDDuTrajetA rend le joueur dont le CORPS porte un trajet de vehicule ouvert sur `slot` a `tUS`
// (microsecondes, horloge du film). Chaine vide : le trajet n'a pas de joueur nomme.
//
// LE CORPS PRIME SUR LE SIEGE. Quand les records de creation etablissent le corps qui tient le slot
// a l'instant, la reponse est le nom des vies de CE corps — la vie qui couvre l'instant si elle
// est nommee, sinon la premiere vie nommee du corps. Un corps dont aucune vie n'est nommee (un bot,
// un index que la table ne rattache a personne) donne un trajet SANS nom : jamais le nom d'un autre
// occupant du siege. Le cas ne se compte pas a part, il se lit dans la couverture des trajets
// (`coverage.vehicles.rides` contre `ridesNamed`).
//
// DEUX CAS GARDENT LA REPONSE DE [IdentityRegistry.XUIDAt] :
//   - aucun corps n'est etabli a l'instant (slot sans record de creation, instant anterieur au
//     premier record) : le film ne dit aucune frontiere de corps ;
//   - les records du slot ne divergent pas ([corpsLu.indexDivergents]) : tous ses corps successifs
//     sont un seul et meme participant, le pont par slot vaut pour chacun.
func (r IdentityRegistry) XUIDDuTrajetA(slot uint32, tUS uint64) string {
	t := int64(tUS) //nolint:gosec // horloge du film, bien sous 2^63
	corps := r.corpsA(slot, t)
	if !corps.etabli {
		return r.XUIDAt(slot, tUS)
	}
	var premier uint64
	for _, l := range r.Vies() {
		if l.slot != slot || l.xuid == 0 || r.corpsA(slot, l.from) != corps {
			continue
		}
		if t >= l.from && t <= l.to {
			return strconv.FormatUint(l.xuid, 10)
		}
		if premier == 0 {
			premier = l.xuid
		}
	}
	if premier != 0 {
		return strconv.FormatUint(premier, 10)
	}
	if c, ok := r.corps[slot]; ok && !c.indexDivergents() {
		return r.XUIDAt(slot, tUS)
	}
	return ""
}
