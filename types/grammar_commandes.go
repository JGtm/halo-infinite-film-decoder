package types

// grammar_commandes.go — LES COMMANDES TENUES, TELLES QUE LA VUE DE CONTROLE LES PORTE : le deplacement, l accroupi
// et le saut.
//
// L entree de controle d un participant porte, avec ses gachettes ([ContinuousFireBurst]), trois commandes : le
// couple analogique du deplacement (deux scalaires de six bits, `0x1f` a zero exact), et deux bits des drapeaux
// `+0x14` — le bouton d accroupi et le bouton de saut. Comme une gachette, l etat d une commande persiste
// jusqu a l entree lue suivante du joueur : une commande TENUE est l intervalle qui va de la premiere entree lue
// qui la pose a la premiere entree lue qui ne la pose plus, ses trous (paquets dont la vue de controle n est pas
// lue) etant portes et jamais combles. Le document publie ces intervalles comme des etats de mouvement
// (`stances[]`), jamais la serie brute des valeurs.
//
// Une commande dit ce que le JOUEUR demande, pas ce que le corps fait : le bouton d accroupi est aussi celui de la
// glissade, et le bouton de saut est enfonce parfois sans que le corps decolle.

// Les genres de commande tenue, publies tels quels a `stances[].kind` (et portes par [ControlHold.Kind]).
const (
	// MovementMoveCommand : la commande de deplacement est non nulle.
	MovementMoveCommand = "moveCommand"
	// MovementCrouchCommand : le bouton d accroupi est tenu (bit de valeur 1 des drapeaux). Le jeu n a qu un bouton
	// pour l accroupi et la glissade : l intervalle couvre les deux, sans les distinguer.
	MovementCrouchCommand = "crouchCommand"
	// MovementJumpCommand : le bouton de saut est tenu (bit de valeur 4 des drapeaux). C est un bouton enfonce, pas
	// un saut effectue : le saut mesure sur le corps reste `jumpDerived`.
	MovementJumpCommand = "jumpCommand"
)

// ControlHold est UNE commande tenue lue dans la vue de controle.
type ControlHold struct {
	// Kind est le genre de la commande : `MovementMoveCommand`, `MovementCrouchCommand` ou `MovementJumpCommand`.
	Kind string
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

// ControlHoldCount est le compte des commandes tenues d UN genre.
type ControlHoldCount struct {
	// Holds est le nombre de commandes tenues rendues ; WithHole celles qu un trou touche (une borne ou un
	// passage interieur) ; InnerHoles le nombre de passages interieurs.
	Holds, WithHole, InnerHoles int
}

// ControlHoldStats dit ce que la lecture des commandes a rendu. Les trous de la vue de controle eux-memes
// sont ceux de [ContinuousFireStats].
type ControlHoldStats struct {
	// Holds est le nombre de commandes de deplacement tenues rendues ; WithHole celles qu un trou touche ; InnerHoles
	// le nombre de passages interieurs.
	//
	// Deprecated: ces trois comptes sont ceux de la commande de deplacement seule, comme en v1.5.0 ; ils sont tenus a
	// jour et egaux a Move.Holds, Move.WithHole et Move.InnerHoles. Le compte de chaque genre vit dans Move, Crouch et
	// Jump, qu il faut lire de preference.
	Holds, WithHole, InnerHoles int
	// Move, Crouch et Jump comptent les commandes tenues de chaque genre.
	Move, Crouch, Jump ControlHoldCount
	// Entries est le nombre d entrees lues qui portent le bloc de commande.
	Entries int
	// Gaps est le nombre d ecarts mesures entre deux entrees lues d une meme place, sans trou entre elles ;
	// StepGaps est le nombre de ceux qui portent le pas, et StepUS le pas : l ecart le plus frequent, en
	// microsecondes. Zero quand aucun ecart n a ete mesure.
	Gaps, StepGaps int
	StepUS         int64
}

// Les deux constantes de la PORTE DE CADENCE : la condition pour que les commandes d accroupi et de saut se
// publient. Elles se lisent sur le pas des entrees ([ControlHoldStats.StepUS]), mesure sur les 22 films du corpus
// de reference (builds 1.4.1 a 1.13.0) : 15 films a 60 Hz, 4 films a 30 Hz, 3 films sans entree lue.
const (
	// PasDeCommande60HzMaxUS est le pas le plus grand, en microsecondes, d un film enregistre a pleine cadence
	// (une entree par tick, 60 Hz). Mesure : les films a 60 Hz ont un pas de 16 640 a 16 679 µs, ceux a 30 Hz de
	// 33 364 a 33 380 µs, et aucun film n a un pas entre les deux. 25 000 est le milieu des deux ticks (16 667 et
	// 33 333) : plus de 8 000 µs de marge de chaque cote. A verifier sur des films hors corpus.
	PasDeCommande60HzMaxUS = 25_000
	// EcartsDuPasMin est le nombre minimal d ecarts qui portent le pas pour qu il soit mesurable. Mesure : sur les 19
	// films dont les entrees sont lues, les ecarts qui portent le pas sont au moins 54 086 (le film le plus
	// pauvre, a 30 Hz) et pesent au moins 86 % des ecarts mesures ; les 3 films sans entree lue (une build 1.4.1 et
	// deux films sans section de build) n en ont aucun. 1 000 ecarts font 17 s d enregistrement d une place a
	// 60 Hz : sous ce compte le pas serait celui d un echantillon plutot que celui d un film, et la part de plus de
	// 86 % des ecarts qui le portent ne se mesurerait plus a un point pres. A verifier sur des films hors corpus.
	EcartsDuPasMin = 1_000
)

// PleineCadence dit que les commandes de ce film sont enregistrees a pleine cadence : le pas est mesurable
// ([EcartsDuPasMin]) et ne depasse pas [PasDeCommande60HzMaxUS]. Un film sous la cadence ne publie ni accroupi ni
// saut : un appui d un seul tick n y est vu qu une fois sur deux.
func (s ControlHoldStats) PleineCadence() bool {
	return s.StepGaps >= EcartsDuPasMin && s.StepUS > 0 && s.StepUS <= PasDeCommande60HzMaxUS
}
