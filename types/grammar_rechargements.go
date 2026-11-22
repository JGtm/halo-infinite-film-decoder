package types

// grammar_rechargements.go — LES RECHARGEMENTS D ARME, TELS QUE LA VUE A DES TRAMES LES PORTE.
//
// Un message `weapon_reload` (genre 38) porte l index de joueur de l auteur et l unite qui recharge ; il ne
// nomme pas l arme. Le canal de la grammaire rend un rechargement par (trame, auteur, unite) : le jeu ecrit
// souvent le meme message deux fois dans la meme trame, et ces copies se comptent ([ReloadStats.Copies]).

// ReloadNoPlayer est le [ReloadRead.Player] d un message qui ne porte pas d index de joueur.
const ReloadNoPlayer = -1

// ReloadRead est UN rechargement lu dans la vue A d une trame delta.
type ReloadRead struct {
	// TimestampUS est l horodatage de la trame, horloge du film.
	TimestampUS uint64
	// Player est l index de joueur de l auteur tel que le message l ecrit (cinq bits, 0..31), ou
	// [ReloadNoPlayer].
	Player int8
}

// ReloadStats dit ce que le canal a vu. Sans ces denominateurs, « N rechargements » ne se juge pas : une
// trame dont la liste d evenements s arrete sur un message non lu perd ceux qui le suivent.
type ReloadStats struct {
	// Frames est le nombre de trames delta lues ; Closed celles dont la vue A est lue jusqu a son
	// terminateur ; Stopped celles qu un message non lu arrete.
	Frames, Closed, Stopped int
	// Messages est le nombre de rechargements rendus ; Copies celui des messages qui copient un message
	// de la meme trame (meme auteur, memes references) et ne sont pas rendus.
	Messages, Copies int
}
