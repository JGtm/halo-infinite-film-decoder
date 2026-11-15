package types

// grammar_presence.go — QUI EST LISTE DANS LA POPULATION DE LA SESSION, ET QUAND : les paquets de
// type 8.

// RosterListing est UNE plage ininterrompue pendant laquelle un humain est liste dans la population
// de la session que les paquets de type 8 rafraichissent : du premier paquet qui le nomme avec un
// siege donne au premier paquet suivant qui ne le nomme plus (ou le nomme avec un autre siege).
//
// UN PAQUET NE PART QUE QUAND LA POPULATION CHANGE : une plage ouverte a la fin du film n est pas
// un depart, et une plage qui s ouvre apres le premier paquet est une ARRIVEE en cours de match.
// Un joueur qui part puis revient porte deux plages.
type RosterListing struct {
	// XUID du joueur, tel que le paquet l ecrit.
	XUID uint64
	// Slot est le siege moteur sous lequel le paquet le liste (cinq bits), -1 quand l entree ne porte
	// pas de siege. C est l index que `unit-control`, le fil des morts et le tableau des scores
	// emploient pour designer un joueur : deux joueurs successifs peuvent le porter l un apres l autre.
	Slot int
	// FromUS est l horodatage du premier paquet de la plage, sur l horloge des paquets du film.
	FromUS uint64
	// ToUS est l horodatage du premier paquet qui ne liste plus cette plage ; 0 quand aucun paquet
	// ne la ferme (le joueur est encore liste au dernier paquet).
	ToUS uint64
}

// RosterListingStats compte ce que la lecture des paquets de type 8 a rencontre.
type RosterListingStats struct {
	// Scanned dit que la lecture a eu lieu ; faux = film sans registre, build inconnu ou table des
	// joueurs refusee : une liste vide ne dit alors rien du film.
	Scanned bool
	// Packets est le nombre de paquets de type 8 du film ; Distinct ceux dont le contenu differe du
	// precedent (les seuls lus) ; Refused ceux dont la lecture a deborde ou n a pas rendu toutes les
	// entrees annoncees (ecartes : aucune plage n en sort).
	Packets, Distinct, Refused int
	// Entries est le nombre d entrees lues dans les paquets distincts.
	Entries int
}
