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
//
// UN ÉVÉNEMENT PEUT AUSSI ÊTRE RELU : le film réécrit un événement déjà écrit (même tireur, même unité, même arme, même numéro) dans
// les paquets qui suivent, parfois dans le même, et mêlé à des tirs neufs (mesuré sur 508bd2fb : les numéros 3, 2, 4, 3 d'un même
// paquet, après le 2 du paquet précédent, tous de la même unité et de la même arme). Un numéro déjà écrit qui revient derrière la tête
// de moins d'un demi-tour peut être cette relecture ou un tour de 256 ; compté comme un tour, il ajoute 253 à 255 tirs qui n'existent
// pas (845 tirs estimés pour 55 événements). La règle est un FAIT PHYSIQUE : un tour demande au moins 129 coups entre la tête et ce
// numéro, et aucune arme du titre ne tire plus vite que la cadence maximale de sa table ([cadenceMaximaleDesArmes]). Un numéro que la
// même unité a déjà écrit, derrière la tête de moins d'un demi-tour, et dont le temps écoulé depuis la tête ne laisse pas tirer ce nombre
// de coups à cette cadence, est une relecture : il n'ajoute rien et ne déplace pas la tête. Sinon, c'est un tour. Aucun seuil de durée
// n'est posé : il suit la cadence de la table.

// cadenceMaximaleDesArmes est la plus grande cadence, en coups par seconde, que la table des armes du titre
// ([continuousWeaponsByTag]) donne à une arme : aucune arme du titre ne tire plus vite.
func cadenceMaximaleDesArmes() float64 {
	cadence := 0.0
	for _, a := range continuousWeaponsByTag {
		cadence = max(cadence, a.rate, a.rate0)
	}
	return cadence
}

// demiTourDeLaNumerotation est la distance en arrière au-dessous de laquelle un numéro peut être en retard : au-delà, le même numéro
// est plus près en avant, et un saut avant de plus de 128 tirs existe (tirs continus qui ne s'écrivent pas).
const demiTourDeLaNumerotation = 128

// numerotationDuTireur suit la numérotation des tirs d'un joueur.
type numerotationDuTireur struct {
	vu bool
	// tete est le dernier numéro compté, position le nombre de tirs comptés jusqu'à lui, unite l'unité qui l'a écrit.
	tete, position int
	unite          uint32
	// instant est l'horodatage, en microsecondes, de l'événement qui a placé la tête.
	instant uint64
	// trous range, par position, l'unité qui a sauté le numéro de cette position.
	trous map[int]uint32
	// ecrits range, par position, l'unité qui a écrit l'événement compté à cette position (les relectures se reconnaissent par eux).
	ecrits map[int]uint32
}

// saut rend le nombre de tirs que l'événement de numéro `numero`, écrit par l'unité `unite` à l'instant `instant` (microsecondes, sur
// l'horloge du film), ajoute au compte du joueur. Les événements arrivent dans l'ordre des instants.
func (n *numerotationDuTireur) saut(numero int, unite uint32, instant uint64) int {
	if !n.vu {
		n.vu, n.tete, n.position, n.unite, n.instant = true, numero, 1, unite, instant
		n.ecrits = map[int]uint32{1: unite}
		return 1
	}
	if recul := (n.tete - numero + 256) % 256; recul > 0 && recul < demiTourDeLaNumerotation {
		if u, ok := n.trous[n.position-recul]; ok && u == unite {
			delete(n.trous, n.position-recul)
			return 0
		}
	}
	saut := (numero - n.tete + 256) % 256
	if n.relecture(numero, unite, instant) {
		return 0
	}
	if n.unite == unite && saut > 1 {
		if n.trous == nil {
			n.trous = map[int]uint32{}
		}
		for k := 1; k < saut; k++ {
			n.trous[n.position+k] = unite
		}
	}
	n.tete, n.unite, n.position, n.instant = numero, unite, n.position+saut, instant
	n.ecrits[n.position] = unite
	for p := range n.trous {
		if p <= n.position-demiTourDeLaNumerotation {
			delete(n.trous, p) // trop loin derrière la tête pour qu'un retard le remplisse
		}
	}
	for p := range n.ecrits {
		if p <= n.position-demiTourDeLaNumerotation {
			delete(n.ecrits, p) // trop loin derrière la tête pour qu'une relecture le répète
		}
	}
	return saut
}

// relecture dit que le numéro est la relecture d'un événement que la même unité a déjà écrit, derrière la tête de moins d'un demi-tour :
// revenir à ce numéro par un tour demanderait plus de coups que la cadence maximale des armes n'en laisse dans le temps écoulé depuis la tête.
func (n *numerotationDuTireur) relecture(numero int, unite uint32, instant uint64) bool {
	recul := (n.tete - numero + 256) % 256
	if recul == 0 || recul >= demiTourDeLaNumerotation {
		return false
	}
	if u, ok := n.ecrits[n.position-recul]; !ok || u != unite {
		return false // aucun événement de cette unité à cette position : rien à relire
	}
	ecoule := max(float64(int64(instant)-int64(n.instant)), 0) / 1e6 //nolint:gosec // horloge du film
	return float64(256-recul) > 1+cadenceMaximaleDesArmes()*ecoule
}
