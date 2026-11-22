package replay

// document_numerotation.go — LA NUMEROTATION DES TIRS D'UN JOUEUR : COMPTER LES TIRS SAUTES, JAMAIS UN RETARD.
//
// Le jeu numérote les tirs de chaque joueur modulo 256, à chaque tir, écrit ou non. Entre deux événements de tir écrits,
// le saut du numéro est le nombre de tirs tirés : la somme des sauts est le nombre de tirs ([ShotStat.Shots]).
//
// UN ÉVÉNEMENT PEUT ARRIVER APRÈS UN ÉVÉNEMENT PLUS RÉCENT : le film écrit un tir (numéro 117) après le tir 121 du même joueur et de
// la même unité. Compté comme un saut, il ajoute 252 tirs qui n'existent pas. Un tel numéro EN RETARD remplit un TROU : un numéro
// que le saut qui a mené à la tête a laissé sans événement (118 à 120 entre 117 et 121) ; ce tir a déjà été compté par ce saut.
//
// LA RÈGLE, PAR STRUCTURE ET NON PAR LE TEMPS. Un numéro est en retard si et seulement s'il se trouve derrière la tête de moins d'un
// demi-tour, dans un trou que l'unité de ce tir a elle-même sauté et qu'aucun retard n'a rempli. Il n'ajoute rien et ne déplace pas
// la tête. Tout autre numéro derrière la tête, y compris un numéro déjà vu comme événement ou qui n'est dans aucun trou, est un saut
// avant, compté comme tel (le numéro revient après un tour de 256). Un
// trou ne se remplit qu'une fois ; un trou sauté entre deux unités (une réapparition, une autre arme tenue) n'est celui d'aucune.
//
// LA TÊTE ET LA POSITION SONT CELLES DU JOUEUR, PAS DE L'UNITÉ : la numérotation continue d'une unité à l'autre ; seuls les trous
// portent l'unité qui les a sautés.

// demiTourDeLaNumerotation est la distance en arrière au-dessous de laquelle un numéro peut être en retard : au-delà, le même numéro
// est plus près en avant, et un saut avant de plus de 128 tirs existe (tirs continus qui ne s'écrivent pas).
const demiTourDeLaNumerotation = 128

// numerotationDuTireur suit la numérotation des tirs d'un joueur.
type numerotationDuTireur struct {
	vu bool
	// tete est le dernier numéro compté, position le nombre de tirs comptés jusqu'à lui, unite l'unité qui l'a écrit.
	tete, position int
	unite          uint32
	// trous range, par position, l'unité qui a sauté le numéro de cette position.
	trous map[int]uint32
}

// saut rend le nombre de tirs que l'événement de numéro `numero`, écrit par l'unité `unite`, ajoute au compte du joueur.
func (n *numerotationDuTireur) saut(numero int, unite uint32) int {
	if !n.vu {
		n.vu, n.tete, n.position, n.unite = true, numero, 1, unite
		return 1
	}
	if recul := (n.tete - numero + 256) % 256; recul > 0 && recul < demiTourDeLaNumerotation {
		if u, ok := n.trous[n.position-recul]; ok && u == unite {
			delete(n.trous, n.position-recul)
			return 0
		}
	}
	saut := (numero - n.tete + 256) % 256
	if n.unite == unite && saut > 1 {
		if n.trous == nil {
			n.trous = map[int]uint32{}
		}
		for k := 1; k < saut; k++ {
			n.trous[n.position+k] = unite
		}
	}
	n.tete, n.unite, n.position = numero, unite, n.position+saut
	for p := range n.trous {
		if p <= n.position-demiTourDeLaNumerotation {
			delete(n.trous, p) // trop loin derrière la tête pour qu'un retard le remplisse
		}
	}
	return saut
}
