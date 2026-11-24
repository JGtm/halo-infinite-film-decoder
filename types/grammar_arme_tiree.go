package types

// grammar_arme_tiree.go : LE JEU D ARMES DESIRE DU BIPEDE, LU A CHAQUE CHANGEMENT dans les trames
// delta (composant `i42 biped-desired-weapon-set`).

// DrawnWeaponRead est UNE lecture du jeu d armes desire d un bipede dans une trame delta. Une trame
// ne porte `i42` que quand le jeu change : la suite des lectures d une vie est la suite de ses
// changements d arme, la ou les images-cles ne donnent l emplacement qu une fois toutes les vingt
// secondes environ (`KeyframeInventory.DrawnSlot`, la meme grandeur).
type DrawnWeaponRead struct {
	// TimestampUS : l horodatage du paquet porteur, sur la meme horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Slot : le slot du bipede, comparable a BipedPosition.Slot.
	Slot uint32
	// Principal : l emplacement desire en main principale (0 a 3), egal a l emplacement degaine hors
	// d un changement en cours ; -1 quand le film n en ecrit aucun.
	Principal int8
	// Second : l emplacement desire en seconde main ; -1 hors ambidextrie.
	Second int8
}

// DrawnWeaponStats sont les denominateurs de la lecture du jeu d armes desire.
type DrawnWeaponStats struct {
	// Scanned : la marche des trames a ete jouee et son canal clos ; faux = aucune lecture n a eu lieu,
	// ce qui n est pas « zero changement ».
	Scanned bool
	// Absent : l archetype bipede du film ne declare pas le composant `i42` ; rien n a pu etre lu.
	Absent bool
	// Read : lectures retenues.
	Read int
	// SlotUnbound : lectures ecartees parce que leur slot n est pas lie a un bipede dans la table
	// d entites de la marche.
	SlotUnbound int
	// Duplicates : la meme lecture, meme slot et meme horodatage, re-publiee par une seconde
	// traversee du meme record.
	Duplicates int
}
