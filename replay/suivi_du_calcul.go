package replay

// suivi_du_calcul.go — L'OBSERVATEUR DES FILS QUI CALCULENT, PORTE PAR LE CONTEXTE DE LA CUISSON.
//
// Les tests qui comptent les fils ([grammar.FilmContext.SuivreLeCalcul]) posent leur observateur sur le
// `context.Context` qu'ils passent a la cuisson : il appartient a CET appel, ne traverse aucun autre film
// et ne laisse aucun etat de paquet. En production personne ne le pose : la lecture est une recherche de
// cle absente par cuisson, sans effet sur la sortie.

import "context"

// cleDuSuiviDuCalcul est la cle de valeur du contexte qui porte l'observateur.
type cleDuSuiviDuCalcul struct{}

// suiviDuCalculDe rend l'observateur que `ctx` porte, nul s'il n'en porte pas.
func suiviDuCalculDe(ctx context.Context) func(delta int) {
	f, _ := ctx.Value(cleDuSuiviDuCalcul{}).(func(delta int))
	return f
}
