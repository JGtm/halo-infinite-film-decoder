package types

// grammar_gestes.go — LES GESTES DU JOUEUR, TELS QUE LA VUE A DES TRAMES LES PORTE.
//
// Deux messages de la vue A disent qu'un joueur ENGAGE un geste : `biped_melee_initiate` (le coup de mêlée
// part) et `biped_throw_initiate` (le lancer est amorcé). Chacun porte l'index de joueur de l'auteur. Le
// message dit l'instant où le geste part, jamais son résultat : un coup part qu'il touche ou non, une amorce
// n'est pas une grenade. Le canal de la grammaire rend un geste par (trame, genre, auteur, références) ; les
// copies d'un même message dans la même trame se comptent ([GestureStats]) sans devenir des gestes de plus.

// GestureKind dit quel geste un message engage. La valeur zéro n'est aucun geste.
type GestureKind uint8

// Les gestes que la vue A engage.
const (
	// GestureMelee : le coup de mêlée part (message `biped_melee_initiate`, genre 40).
	GestureMelee GestureKind = 1
	// GestureThrowStart : le lancer est amorcé (message `biped_throw_initiate`, genre 39). Le projectile naît
	// plus tard : l'amorce et la sortie sont deux faits.
	GestureThrowStart GestureKind = 2
)

// GestureNoPlayer est le [GestureRead.Player] d'un message qui ne porte pas d'index de joueur.
const GestureNoPlayer = -1

// GestureRead est UN geste lu dans la vue A d'une trame delta.
type GestureRead struct {
	// TimestampUS est l'horodatage de la trame, horloge du film.
	TimestampUS uint64
	// Player est l'index de joueur de l'auteur tel que le message l'écrit (cinq bits, 0..31), ou
	// [GestureNoPlayer].
	Player int8
	// Kind est le geste engagé.
	Kind GestureKind
}

// GestureCounts est le compte des messages d'un geste : ceux que le canal rend, et les copies qu'il retire.
type GestureCounts struct {
	// Messages est le nombre de gestes rendus ; Copies celui des messages qui copient un message de la même
	// trame (même genre, même auteur, mêmes références) et ne sont pas rendus.
	Messages, Copies int
}

// GestureStats dit ce que le canal a vu. Sans ces dénominateurs, « N gestes » ne se juge pas : une trame
// dont la liste d'événements s'arrête sur un message non lu perd ceux qui le suivent.
type GestureStats struct {
	// Frames est le nombre de trames delta lues ; Closed celles dont la vue A est lue jusqu'à son
	// terminateur ; Stopped celles qu'un message non lu arrête.
	Frames, Closed, Stopped int
	// Melee et Throw sont les comptes des deux messages.
	Melee, Throw GestureCounts
}
