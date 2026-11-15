package types

// grammar_statborg.go — QUI EST LA LIGNE DU TABLEAU DES SCORES : l'entree de chaque entite du statborg.

// Les genres d'entree du tableau des scores.
const (
	// StatborgEntryPlayer : la ligne d'un joueur ; [StatborgEntry.ID] est son identifiant moteur.
	StatborgEntryPlayer = 0
	// StatborgEntryTeam : la ligne d'une equipe ; [StatborgEntry.ID] est son numero d'equipe.
	StatborgEntryTeam = 1
)

// StatborgEntry est l'entree d'une entite du tableau des scores, lue a une image-cle : de qui est la ligne.
// L'identifiant moteur d'un joueur est le siege que les paquets de type 8 listent ([RosterListing.Slot]) ;
// le joueur qui le tient a un instant se lit dans ces plages.
type StatborgEntry struct {
	// Slot est le slot de replication de l'entite, la moitie du slot de ses enregistrements de valeurs.
	Slot int
	// TS est l'horodatage de l'image-cle qui la porte, sur l'horloge des paquets du film.
	TS uint64
	// Kind est le genre de la ligne ([StatborgEntryPlayer], [StatborgEntryTeam]).
	Kind int
	// ID est l'identifiant moteur du joueur, ou le numero de l'equipe.
	ID int
}

// StatborgEntryStats compte ce que la lecture des entrees a rencontre.
type StatborgEntryStats struct {
	// Scanned dit que la lecture a eu lieu ; faux = registre absent ou composant inattendu, une liste
	// vide ne dit alors rien du film.
	Scanned bool
	// Records est le nombre de records d'entite du tableau des scores vus aux images-cles, Read ceux dont
	// l'entree est lue, Vacant ceux dont l'entite n'est la ligne de personne (genre inconnu), Unreached
	// ceux dont le composant n'a pas ete atteint par la marche.
	Records, Read, Vacant, Unreached int
	// DecidedRounds est le masque des manches dont une entite du tableau porte une issue (gagnee, perdue, egalite)
	// a une image-cle : le bit `r` est leve quand l'issue de la manche `r` n'est pas nulle. Une manche n'a d'issue
	// qu'une fois jouee ; la derniere peut ne l'avoir ecrite dans aucune image-cle.
	DecidedRounds uint32
}
