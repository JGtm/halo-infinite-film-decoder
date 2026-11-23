package lecture

// MessageDeReplique est un message `AIDialog` (genre 75, lecteur `FUN_140f2e634`) de la vue A : une
// réplique prononcée par une unité. Le locuteur est la référence 0 de l'en-tête du message, et l'index
// de joueur de la charge quand la charge en porte un.
//
// Les références sont les index bruts de leur domaine (`FUN_1406d3140`), sans base ; une référence dont
// la garde est à zéro n'est pas portée par le message.
//
// Ce qui est lu avec sa largeur : la priorité (5 bits), l'identifiant du son (32 bits gardés), l'indice
// de réplique (12 bits gardés) et l'index de joueur (5 bits gardés). La queue de la charge (un ou deux
// bits, dont la largeur est lue sur un bit) n'a pas de sens établi : elle est gardée telle que lue.
//
// Vingt octets : sa taille est gelée par `tailles_test.go`.
type MessageDeReplique struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// Son est l'identifiant du son, valable quand [RepliqueSon].
	Son uint32
	// Ref et Gen sont l'index de la référence 0 et sa génération (2 bits), valables quand
	// [RepliqueRef0].
	Ref uint16
	// Ligne est l'indice de la réplique (12 bits), valable quand [RepliqueLigne].
	Ligne uint16
	Gen   uint8
	// Priorite est la classe de priorité de la réplique (5 bits). Le message en porte la valeur ; le
	// jeu la nomme `DialoguePriority`.
	Priorite uint8
	// Joueur est l'index de joueur de la charge ; [RefAbsente] quand la garde du champ est fermée.
	Joueur int8
	// Drapeaux dit quels champs sont portés ([RepliqueRef0], [RepliqueSon], [RepliqueLigne]).
	Drapeaux uint8
	// Queue est la valeur des bits de queue, sur [LargeurQueue] bits (un ou deux) : opaque.
	Queue        uint8
	LargeurQueue uint8
}

// Les drapeaux d'un [MessageDeReplique].
const (
	// RepliqueRef0 : la garde de la référence 0 (le locuteur) est ouverte.
	RepliqueRef0 uint8 = 1 << iota
	// RepliqueSon : l'identifiant du son est porté.
	RepliqueSon
	// RepliqueLigne : l'indice de réplique est porté.
	RepliqueLigne
)

// MessageDAnnonce est un message `Dialogue2D` (genre 76, lecteur `FUN_140f2e87c`) de la vue A : une
// annonce de l'annonceur ou d'un script de mode, avec ses destinataires.
//
// Ce qui est lu avec sa largeur : l'identifiant du son (32 bits gardés), le masque des destinataires
// (32 bits) et l'index de joueur (5 bits gardés). La seconde valeur de 32 bits gardée n'a pas de sens
// établi : elle est gardée telle que lue.
//
// Vingt octets : sa taille est gelée par `tailles_test.go`.
type MessageDAnnonce struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// Son est l'identifiant du son, valable quand [AnnonceSon].
	Son uint32
	// Destinataires est le masque de 32 bits que la charge écrit en dernier, tel que lu : le premier
	// bit lu en est le bit de poids fort.
	Destinataires uint32
	// Opaque est la seconde valeur gardée de la charge, valable quand [AnnonceOpaque] : sens non établi.
	Opaque uint32
	// Joueur est l'index de joueur de la charge ; [RefAbsente] quand la garde du champ est fermée.
	Joueur int8
	// Drapeaux dit quels champs sont portés ([AnnonceSon], [AnnonceOpaque]).
	Drapeaux uint8
}

// Les drapeaux d'un [MessageDAnnonce].
const (
	// AnnonceSon : l'identifiant du son est porté.
	AnnonceSon uint8 = 1 << iota
	// AnnonceOpaque : la seconde valeur gardée est portée.
	AnnonceOpaque
)
