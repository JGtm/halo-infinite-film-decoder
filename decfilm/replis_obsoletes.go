package decfilm

import "github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"

// replis_obsoletes.go — LES NOMS DE REPLI QUE LE DECODEUR NE DECLENCHE PLUS.
//
// Ces deux noms (la precision par arme sautee, les distances de touche desactivees) sont decides
// et comptes par l'hote : aucun chemin du decodeur ne les declenche. Leur entree reste au
// registre, sans site dans la brique, pour qu'un `Declenche` de l'hote se rattache a un nom connu.
// Les constantes gardent leur valeur et partent a la prochaine version majeure.

// NomPrecisionParArmePasseSautee nomme un repli que l'hote declare et compte lui-meme.
//
// Deprecated: aucun chemin du decodeur ne declenche plus ce repli ; la constante est retiree a la
// prochaine version majeure.
const NomPrecisionParArmePasseSautee = fallback.NomPrecisionParArmePasseSautee

// NomDistancesDeToucheDesactivees nomme un repli que l'hote declare et compte lui-meme.
//
// Deprecated: aucun chemin du decodeur ne declenche plus ce repli ; la constante est retiree a la
// prochaine version majeure.
const NomDistancesDeToucheDesactivees = fallback.NomDistancesDeToucheDesactivees
