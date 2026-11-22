package types

// grammar_commandes.go — LA COMMANDE DE DEPLACEMENT TENUE, TELLE QUE LA VUE DE CONTROLE LA PORTE.
//
// L entree de controle d un participant porte, avec ses gachettes ([ContinuousFireBurst]), le couple analogique
// du deplacement : deux scalaires de six bits, `0x1f` a zero exact. Comme une gachette, l etat de la commande
// persiste jusqu a l entree lue suivante du joueur : une commande TENUE est l intervalle qui va de la premiere
// entree lue qui la pose (un scalaire hors du zero exact) a la premiere entree lue qui ne la pose plus, ses
// trous (paquets dont la vue de controle n est pas lue) etant portes et jamais combles. Le document publie
// ces intervalles comme des etats de mouvement (`stances[]`), jamais la serie brute des valeurs.

// MovementMoveCommand est le genre publie tel quel a `stances[].kind` : la commande de deplacement est non
// nulle.
const MovementMoveCommand = "moveCommand"

// ControlHold est UNE commande de deplacement tenue lue dans la vue de controle.
type ControlHold struct {
	// FilmIndex est l index de controle de l entree : la PLACE du joueur, comme [ContinuousFireBurst.FilmIndex].
	FilmIndex int
	// StartUS / EndUS bornent la commande tenue, horloge du film ; StartBound / EndBound disent ce qui les a
	// posees (`ContinuousFireBound*`).
	StartUS, EndUS       uint64
	StartBound, EndBound string
	// Entries est le nombre d entrees lues qui tiennent la commande pendant l intervalle.
	Entries int
	// Holes sont les passages INCONNUS de l intervalle : tenue avant et apres, non lue entre les deux.
	Holes []ContinuousFireHole
}

// ControlHoldStats dit ce que la lecture des commandes a rendu. Les trous de la vue de controle eux-memes
// sont ceux de [ContinuousFireStats].
type ControlHoldStats struct {
	// Holds est le nombre de commandes tenues rendues ; WithHole celles qu un trou touche (une borne ou un
	// passage interieur) ; InnerHoles le nombre de passages interieurs.
	Holds, WithHole, InnerHoles int
	// Entries est le nombre d entrees lues qui portent le bloc de commande.
	Entries int
}
