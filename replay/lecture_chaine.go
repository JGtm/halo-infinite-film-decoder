package replay

// lecture_chaine.go — LES VALEURS QUE RENDENT LES LECTURES DE LA CHAINE DES POSITIONS ET DU PORTAGE.
//
// Chaque balayage de la chaine (poussees, tirs, armes tenues, ramassages, inventaire delta, capacites,
// equipement, camouflage, grappin, impulsions, charges) se coupe en une methode `lire…`, qui ne fait que
// lire le film et rend l une de ces valeurs, et une methode `publier…`, qui journalise, verse les replis,
// pose les entrees et observe l etape. Le modele est `lecture_monde.go` ; ici les lectures n ont pas de
// decision de repli a transporter, seulement leur resultat, leurs statistiques et leur erreur.

// lectureStats est ce qu une lecture du film a rendu : sa valeur, ses statistiques et son erreur.
type lectureStats[V, S any] struct {
	val V
	st  S
	err error
}

// lectureNue est ce qu une lecture du film a rendu quand elle n a pas de statistiques : sa valeur et
// son erreur.
type lectureNue[V any] struct {
	val V
	err error
}

// lireAvecStats range le resultat (valeur, statistiques, erreur) d un balayage de la grammaire.
func lireAvecStats[V, S any](v V, st S, err error) lectureStats[V, S] {
	return lectureStats[V, S]{val: v, st: st, err: err}
}
