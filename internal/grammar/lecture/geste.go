package lecture

// MessageDeGeste est un message de geste de la vue A : le lancer amorcé (`biped_throw_initiate`, genre 39,
// lecteur `FUN_140c6a58c`) ou le coup de mêlée (`biped_melee_initiate`, genre 40, lecteur `FUN_140ff8d70`).
// Les deux messages finissent sur l'index de joueur de l'auteur (`FUN_1407f2058`, cinq bits) ; leur charge
// propre (tête, arme, variante) n'est pas gardée.
//
// Les références sont les index bruts de leur domaine (`FUN_1406d3140`), sans base ; une référence dont
// la garde est à zéro n'est pas portée par le message.
//
// Seize octets : sa taille est gelée par `tailles_test.go`.
type MessageDeGeste struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// Ref et Gen sont les index des trois références et leurs générations (2 bits) ; valables quand
	// leur drapeau [GesteRef0] à [GesteRef2] est posé.
	Ref [3]uint16
	Gen [3]uint8
	// Joueur est l'index de joueur de l'auteur ; [RefAbsente] quand le message n'en porte pas.
	Joueur int8
	// Genre est 39 ou 40.
	Genre uint8
	// Drapeaux dit quelles références sont portées ([GesteRef0] à [GesteRef2]).
	Drapeaux uint8
}

// Les drapeaux d'un [MessageDeGeste].
const (
	// GesteRef0 à GesteRef2 : la garde de la référence d'indice 0, 1 ou 2 est ouverte.
	GesteRef0 uint8 = 1 << iota
	GesteRef1
	GesteRef2
)
