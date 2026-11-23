package grammar

import (
	"math/bits"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// entete_table_balayage.go — LE REJET RAPIDE D'UNE POSITION, POUR LES BALAYAGES BIT A BIT D'UN
// EN-TETE D'ENTREE DE TABLE D'ENTITES.
//
// # LE PROBLEME
//
// La recherche des en-tetes de la table d'une image-cle ([EntetesDeTable]) et celle des en-tetes de
// joueur gere ([slotsDEntetesExacts]) testent l'eid a CHAQUE bit d'un payload. L'eid porte la
// generation (deux bits de tete) puis l'emplacement sur trente bits, que la table borne par
// [kfTableCap] : les dix-sept bits qui suivent la generation sont NULS pour toute entree valide.
// Presque toutes les positions d'un payload echouent a ce test, et chacune relisait un mot de 64 bits
// pour quelques bits.
//
// # CE QUE FAIT [prochainEnteteDeTable]
//
// Il lit UN mot de 64 bits et en tire, d'un coup, les positions dont l'eid passe ce test sur les
// quarante-six positions que le mot couvre : generation demandee (nulle, ou non nulle) et dix-sept
// bits d'emplacement nuls. Il ne decide RIEN d'autre : il rend la premiere position qui NE PEUT PAS
// ETRE ECARTEE, et l'appelant lui applique son test complet, inchange (type, drapeaux, poignee,
// croissance des emplacements...). Une position sautee a echoue au test d'eid que l'appelant joue
// lui aussi, en premier : l'ensemble des positions acceptees et leur ordre sont donc ceux de la
// boucle position par position.
//
// Le test de tete est PUR : aucun compteur, aucun diagnostic, aucune panique.

// Les bits que le test d'eid lit, relativement a la position : la generation occupe les bits 0 et 1,
// l'emplacement les bits 2 a 31, dont les bits 2 a 18 doivent etre nuls sous [kfTableCap].
const (
	premierBitHautDuSlot     = 2  // premier bit de l'emplacement, le plus fort
	largeurHauteDuSlot       = 17 // bits de l'emplacement qui doivent etre nuls : 30 - 13, avec kfTableCap = 1 << 13
	derniereLectureDeLEntree = premierBitHautDuSlot + largeurHauteDuSlot - 1
	// positionsParMotDeTable : les positions d'un mot dont tous les bits lus sont dans le mot.
	positionsParMotDeTable = 64 - derniereLectureDeLEntree
)

// prochainEnteteDeTable rend la plus petite position q, p <= q < limite, dont l'eid peut etre celui
// d'une entree : generation nulle si `genNulle`, non nulle sinon, et dix-sept bits nuls a q+2..q+18
// (emplacement sous [kfTableCap]). Il rend `limite` si aucune position ne convient.
//
// Les bits au-dela du tampon valent zero ([source.BitsAt]) ; l'appelant garantit p >= 0.
func prochainEnteteDeTable(pay []byte, p, limite int, genNulle bool) int {
	for p < limite {
		i := p >> 3
		w := source.BitsAt(pay, i<<3, 64)
		// Le bit d'indice q (depuis le poids fort) de `w << k` est le bit q+k de `w` ; les bits que
		// le decalage amene de l'autre cote du mot valent zero, ce qui ecarte d'office une plage
		// qui deborde du mot.
		z := ^w
		z2 := z & (z << 1) // z2[q] : les bits q et q+1 sont nuls
		z4 := z2 & (z2 << 2)
		z8 := z4 & (z4 << 4)
		z16 := z8 & (z8 << 8)
		plage := (z16 & (z << 16)) << premierBitHautDuSlot // les dix-sept bits q+2..q+18 sont nuls
		if genNulle {
			plage &= z2
		} else {
			plage &= w | w<<1
		}
		// Seules comptent les positions qui ne precedent pas p.
		plage &= ^uint64(0) >> uint(p&7)
		if plage != 0 {
			return min(i<<3+bits.LeadingZeros64(plage), limite)
		}
		p = i<<3 + positionsParMotDeTable
	}
	return limite
}
