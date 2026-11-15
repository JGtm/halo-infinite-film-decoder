package types

// grammar_controle.go — LE JOUEUR AUX COMMANDES D UN BIPEDE, TEL QUE `unit-control` L ECRIT, ET LA
// SUPPRESSION DU CORPS.

// UnitControlRead est UNE lecture de la tete du composant `unit-control` d un bipede : l index du
// joueur qui le commande, ou son absence. Un delta ne porte le composant que quand il CHANGE, et
// l enregistrement de creation du bipede le porte : la liste est une suite de transitions, et c est
// l assemblage qui la rapporte aux vies.
type UnitControlRead struct {
	// Slot est l identifiant bas du bipede (13 bits), le meme que celui des trajectoires : UNE VIE,
	// pas un joueur, puisque le slot migre aux reapparitions.
	Slot uint32
	// TimestampUS est l horodatage du paquet porteur, sur l horloge de `BipedPosition`.
	TimestampUS uint64
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// Controller est l index (0..31) du joueur aux commandes, sur cinq bits ; -1 quand la porte de
	// tete est fermee : aucun joueur ne commande le bipede a cet instant (mort du joueur, bipede
	// sans joueur, ou bot d une partie qui n en declare pas). Cet index est celui que la table des
	// joueurs et le fil des morts emploient pour designer un participant.
	Controller int
}

// UnitControlStats dit si la lecture de `unit-control` a eu lieu.
type UnitControlStats struct {
	// Scanned dit que la marche a eu lieu ; faux = les lectures n ont pas ete tentees, et une liste
	// vide ne dit rien.
	Scanned bool
}

// BipedDeletion est la SUPPRESSION d un bipede : le record DEL de son slot dans les trames delta.
// Le jeu supprime le corps d un joueur mort quelques secondes apres sa mort, et supprime encore
// commande celui qu une fin de manche ou une remise en place retire.
type BipedDeletion struct {
	// Slot est l identifiant bas du bipede supprime, celui de [UnitControlRead] et des trajectoires.
	Slot uint32
	// TimestampUS est l horodatage du paquet porteur, sur l horloge de `BipedPosition`.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le record dans le film.
	Chunk, PacketIndex int
}

// BipedDeletionStats dit ce que la lecture des suppressions de bipede a rencontre.
type BipedDeletionStats struct {
	// Deletions est le nombre de suppressions publiees.
	Deletions int
	// DeletionsUnproven est le nombre de records DEL d un bipede suivi que la marche a lus sur un
	// debut de liste non prouve : le jeu ne les applique pas au monde, ils ne sont pas publies.
	DeletionsUnproven int
}
