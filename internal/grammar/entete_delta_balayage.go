package grammar

import (
	"math/bits"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// entete_delta_balayage.go — LE REJET RAPIDE D'UNE POSITION, POUR LES BALAYAGES BIT A BIT D'UN
// EN-TETE DE RECORD DELTA.
//
// # LE PROBLEME
//
// Les balayages qui ancrent un record delta testent l'en-tete a CHAQUE bit d'un payload, et
// l'immense majorite des positions est rejetee des les premiers bits : prefixe delta, puis, a
// seize bits de la, deux bits de porte nuls, puis le compteur de masque. Chaque test relisait un
// mot de 64 bits pour quelques bits.
//
// # CE QUE FAIT [prochainEnteteDelta]
//
// Il lit UN mot de 64 bits et en tire, d'un coup, les positions qui passent ces trois tests sur les
// quarante positions environ que le mot couvre. Il ne decide RIEN d'autre : il rend la premiere
// position qui NE PEUT PAS ETRE ECARTEE, et l'appelant lui applique son test complet, inchange
// (slot de la bande, generation, masque, region...). Une position sautee a echoue a un test que
// l'appelant joue lui aussi, avant toute lecture qui pourrait paniquer ou compter : l'ensemble des
// positions acceptees et leur ordre sont donc ceux de la boucle position par position.
//
// Le test de tete est PUR : aucune lecture ne deplace de curseur, aucun compteur, aucun diagnostic.

// Position, relativement au prefixe, des bits que le test de tete lit : le prefixe est a 0, la porte
// nulle aux bits 16 et 17, le compteur de masque (trois bits) aux bits 18 a 20.
const (
	porteDeLEnteteDelta    = 16 // premier des deux bits de porte, qui doivent etre nuls
	compteurDeLEnteteDelta = 18 // premier des trois bits du compteur de masque
	derniereLectureDelta   = 20 // dernier bit lu derriere le prefixe
)

// prochainEnteteDelta rend la plus petite position q, p <= q < limite, dont l'en-tete peut etre
// celui d'un record delta : prefixe a 1, deux bits nuls a q+16 et q+17, compteur de masque (trois
// bits a q+18) d'au moins `mcMin`, qui vaut 1 ou 2. Il rend `limite` si aucune position ne convient.
//
// Les bits au-dela du tampon valent zero ; l'appelant garantit p >= 0 et que les positions
// jugees laissent q+20 dans le tampon, de sorte que la lecture rejoint celle de l'en-tete complet.
func prochainEnteteDelta(pay []byte, p, limite, mcMin int) int {
	for p < limite {
		i := p >> 3
		w := source.BitsAt(pay, i<<3, 64)
		// Le bit d'indice q (depuis le poids fort) de `w << k` est le bit q+k de `w`.
		c := w &^ (w << porteDeLEnteteDelta) &^ (w << (porteDeLEnteteDelta + 1))
		if mcMin >= 2 {
			c &= w<<compteurDeLEnteteDelta | w<<(compteurDeLEnteteDelta+1)
		} else {
			c &= w<<compteurDeLEnteteDelta | w<<(compteurDeLEnteteDelta+1) | w<<(compteurDeLEnteteDelta+2)
		}
		// Seules comptent les positions du mot dont les vingt bits suivants sont dans le mot, et
		// qui ne precedent pas p.
		c &= ^uint64(0) >> uint(p&7)
		c &^= 1<<derniereLectureDelta - 1
		if c != 0 {
			return min(i<<3+bits.LeadingZeros64(c), limite)
		}
		p = i<<3 + 64 - derniereLectureDelta
	}
	return limite
}

// prochainPrefixeNEW rend la plus petite position q, p <= q < limite, qui ouvre un record de
// creation : les trois bits de tete `0`, `0`, `1` (pas un delta, type NEW), que
// [archetypeDeLEnTeteNEW] exige en premier. Il rend `limite` si aucune position ne convient.
//
// Meme lecture par mot que [prochainEnteteDelta], pour un test de trois bits. L'appelant garantit
// p >= 0 et que les positions jugees laissent leur en-tete de creation entier dans le tampon.
func prochainPrefixeNEW(pay []byte, p, limite int) int {
	for p < limite {
		i := p >> 3
		w := source.BitsAt(pay, i<<3, 64)
		// Le bit d'indice q de `w << k` est le bit q+k de `w`.
		c := ^w & ^(w << 1) & (w << 2)
		c &= ^uint64(0) >> uint(p&7)
		c &^= 1<<dernierBitDuPrefixeNEW - 1
		if c != 0 {
			return min(i<<3+bits.LeadingZeros64(c), limite)
		}
		p = i<<3 + 64 - dernierBitDuPrefixeNEW
	}
	return limite
}

// dernierBitDuPrefixeNEW : le prefixe de creation occupe les bits q a q+2 ; le mot juge les positions
// dont ces trois bits sont dans le mot.
const dernierBitDuPrefixeNEW = 2
