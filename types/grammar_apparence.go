package types

// grammar_apparence.go — L'APPARENCE DES SPARTANS TELLE QUE LE FILM L'ECRIT : le bloc de personnalisation de
// chaque humain dans les paquets de type 8, livre en codes bruts.

// AppearancePair est UNE paire du bloc d'apparence d'un joueur : deux mots de 32 bits que le film
// ecrit cote a cote. La cle est la meme pour tous les joueurs d'un meme build, la valeur est propre au
// joueur. Les deux sont lus en petit-boutiste sur les octets du bloc, dans l'ordre du flux.
//
// LE CONTRAT : des CODES OPAQUES. Le decodeur ne dit pas ce qu'ils designent (piece d'armure,
// revetement, couleur...) ; la table de noms est hors du decodeur. Un code est stable pour un joueur
// d'un film a l'autre tant que celui-ci ne change pas d'equipement.
type AppearancePair struct {
	Key, Value uint32
}

// PlayerAppearance est le bloc d'apparence d'UN humain, tel que le dernier paquet de type 8 qui le porte non
// nul l'ecrit. Chaque octet du bloc est range UNE fois : dans `Pairs` ou dans `Rest`.
//
// LA FORME DU BLOC. Quand sa tete se lit proprement (un compteur n plausible, un mot nul, puis n paires
// qui tiennent dans la largeur du bloc), `Pairs` porte les n paires, dans l'ordre du film, et `Rest`
// les octets qui les suivent ; la tete (n et le mot nul) est implicite. Sinon `Pairs` est vide et `Rest`
// porte le bloc entier.
//
// `Rest` est rendu SANS SA QUEUE DE ZEROS : le profil du build donne la largeur du bloc (`Size`), la
// queue omise se retrouve en completant de zeros jusqu'a `Size` moins la tete et les paires. Les codes
// sont opaques : rien ici ne dit ce qu'ils designent.
type PlayerAppearance struct {
	// XUID du joueur, tel que le paquet l'ecrit.
	XUID uint64
	// Size est la largeur du bloc en octets pour le build du film (profil du build).
	Size int
	// Pairs sont les paires (cle, valeur) lues ; vide quand la structure ne se lit pas.
	Pairs []AppearancePair
	// Rest sont les octets du bloc qui ne sont pas dans `Pairs`, sans la queue de zeros.
	Rest []byte
}

// AppearanceStats compte ce que la lecture de l'apparence a rencontre. Les compteurs de joueurs ne
// comptent chacun qu'un humain une fois, dans l'ordre : lu en paires, lu en brut, ou non charge.
type AppearanceStats struct {
	// Scanned dit que la lecture a eu lieu : faux = film sans registre ou sans section d'identification.
	Scanned bool
	// UnknownBuild dit que le build du film n'est pas dans le profil : la largeur du bloc est inconnue,
	// aucune apparence n'est lue (ce n'est pas une erreur du film).
	UnknownBuild bool
	// Packets est le nombre de paquets de type 8 du film ; Distinct ceux dont le contenu differe du precedent
	// (les seuls lus) ; Refused ceux dont la lecture a deborde ou n'a pas rendu toutes les entrees annoncees.
	Packets, Distinct, Refused int
	// NoXUID est le nombre d'entrees sans identifiant de joueur, ecartees.
	NoXUID int
	// Pairs est le nombre de joueurs dont le bloc se lit en paires ; Raw ceux dont le bloc est rendu entier
	// parce que sa tete ne se lit pas ; Unloaded ceux dont tous les blocs sont nuls (apparence non chargee).
	Pairs, Raw, Unloaded int
	// Changed est le nombre de joueurs dont le bloc non nul DIFFERE entre deux paquets du meme film ; la regle
	// est que le dernier bloc non nul fait foi.
	Changed int
}
