package lecture

// MessageDeRechargement est un message `weapon_reload` (genre 38, lecteur `FUN_1407f0ff8`) de la vue A :
// les trois références d'entité de son en-tête, les quatre booléens de sa charge et l'index de joueur de
// l'auteur (`FUN_1407f2058`, cinq bits).
//
// Les références sont les index bruts de leur domaine (`FUN_1406d3140`), sans base ; une référence dont
// la garde est à zéro n'est pas portée par le message.
//
// Seize octets : sa taille est gelée par `tailles_test.go`.
type MessageDeRechargement struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// Ref et Gen sont les index des trois références et leurs générations (2 bits) ; valables quand
	// leur drapeau [RechargementRef0] à [RechargementRef2] est posé.
	Ref [3]uint16
	Gen [3]uint8
	// Joueur est l'index de joueur de l'auteur ; [RefAbsente] quand le message n'en porte pas.
	Joueur int8
	// Booleens porte les quatre `FUN_1406cf008` de la charge, dans l'ordre de la lecture : le bit k
	// de poids faible est le k-ième lu.
	Booleens uint8
	// Drapeaux dit quelles références sont portées ([RechargementRef0] à [RechargementRef2]).
	Drapeaux uint8
}

// Les drapeaux d'un [MessageDeRechargement].
const (
	// RechargementRef0 à RechargementRef2 : la garde de la référence d'indice 0, 1 ou 2 est ouverte.
	RechargementRef0 uint8 = 1 << iota
	RechargementRef1
	RechargementRef2
)
