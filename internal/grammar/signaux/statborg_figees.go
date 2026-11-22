package signaux

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// statborg_figees.go — les VALEURS FIGEES DES MANCHES TERMINEES, composants 28 a 55 de l'archetype du
// statborg (`statborg-finalized-rounds-values-stat-component`, lecteur du jeu FUN_142ed3c50, que
// `grammar.consumeStatborgFinalized` porte deja pour la marche).
//
// # La grammaire
//
//	[32 bits : masque des manches terminees, bit k = manche k]
//	puis, pour chaque bit a 1, dans l'ordre des manches :
//	[1 bit defaut][si 0 : valeur A a longueur variable][1 bit defaut][si 0 : valeur B a longueur variable]
//
// La valeur a longueur variable est celle des compteurs de la manche ([readStatVarWidth]). Le composant
// 28 + k est la valeur figee du composant k de la manche en cours : meme paire (A, B), meme unite.
//
// # Ce qui est mesure (9 films a manches, 13 films lus)
//
//   - L'enchainement des largeurs : apres les composants 28 a 55 et les 64 bits des issues (56), l'entree
//     (57) se lit sur ses 40 bits en une entree de joueur valide ou en entree videe dans 280
//     enregistrements sur 280 ; une lecture qui deraperait n'y rendrait pas une entree.
//   - L'ordre A, B par manche (et non tous les A puis tous les B) : sur les films a trois manches, les
//     valeurs de la manche 0 relues au debut de la manche 2 sont celles lues au debut de la manche 1 ;
//     dans l'autre ordre elles ne le sont pas.
//   - Une valeur declaree par defaut vaut 0 : aucun zero n'est jamais ecrit en clair (12 236 composants),
//     et les compteurs a zero (manches gagnees du perdant, KDA nul) sont declares par defaut.
//   - Le masque d'un enregistrement de la manche r porte les manches 0 a r-1 ; celui de la fin du film
//     ne porte jamais la derniere manche : elle n'est pas figee.
//   - Les valeurs suivent le JOUEUR de la ligne, pas le slot : quand les lignes sont reattribuees au
//     debut d'une manche, un joueur emporte ses manches figees sur sa nouvelle ligne.
//   - Une valeur figee ne change pas d'une emission a l'autre (aucune sur les 9 films).
//
// Une selection de largeur 3 (64 bits) n'apparait que dans des ancrages fortuits : le lecteur la
// refuse, comme celui des compteurs de la manche.

const (
	// statComposantsFiges est le nombre de composants de valeurs figees de l'archetype, un par composant
	// de la manche en cours ; ils suivent ceux-ci ([statCompteursDeManche]).
	statComposantsFiges = 28
	// statFigeesFin est le premier index qui n'est plus une valeur figee : les issues des manches (56),
	// puis l'entree de la ligne (57), qui ont leurs propres lecteurs.
	statFigeesFin = statCompteursDeManche + statComposantsFiges
	// statMasqueManchesBits est la largeur du masque des manches terminees.
	statMasqueManchesBits = 32
)

// decodeFrozenComponent lit le composant de valeurs figees qui commence a `p` : ses valeurs, une par
// manche terminee dans l'ordre des manches, et sa largeur. Faux quand la lecture deborde le paquet,
// qu'une valeur porte une largeur que le lecteur refuse ou sort du domaine des compteurs
// ([statMaxCounter]), ou que le masque designe une manche au-dela de [StatborgMancheMax] : aucune de ces
// formes n'est une emission du moteur.
func decodeFrozenComponent(pay []byte, p int) ([]types.StatFrozen, int, bool) {
	if p < 0 || p+statMasqueManchesBits > len(pay)*8 {
		return nil, 0, false
	}
	mask := source.BitsTronques(pay, p, statMasqueManchesBits)
	if mask>>(StatborgMancheMax+1) != 0 {
		return nil, 0, false
	}
	q := p + statMasqueManchesBits
	var out []types.StatFrozen
	for round := 0; round <= StatborgMancheMax; round++ {
		if mask>>uint(round)&1 == 0 {
			continue
		}
		a, wa, ok := readFrozenValue(pay, q)
		if !ok {
			return nil, 0, false
		}
		b, wb, ok := readFrozenValue(pay, q+wa)
		if !ok {
			return nil, 0, false
		}
		q += wa + wb
		out = append(out, types.StatFrozen{Round: round, A: a, B: b})
	}
	return out, q - p, true
}

// readFrozenValue lit une valeur figee : [1 bit defaut][si 0 : valeur a longueur variable]. Une valeur
// par defaut vaut 0 et occupe un bit.
func readFrozenValue(pay []byte, p int) (int64, int, bool) {
	if p+1 > len(pay)*8 {
		return 0, 0, false
	}
	if source.BitsTronques(pay, p, 1) == 1 {
		return 0, 1, true
	}
	v, w, ok := readStatVarWidth(pay, p+1)
	if !ok || v > statMaxCounter || v < -statMaxCounter {
		return 0, 0, false
	}
	return v, 1 + w, true
}
