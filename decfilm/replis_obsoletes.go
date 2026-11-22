package decfilm

import "github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"

// replis_obsoletes.go — LES NOMS DE REPLI QUE LE REGISTRE NE PORTE PLUS.
//
// Ces deux replis (la precision par arme sautee, les distances de touche desactivees) sont decides
// et comptes par l'hote : le decodeur n'a aucun site pour eux, et leur entree a quitte `Table`.
// Les constantes restent exportees, sans changement de valeur, pour la compatibilite des
// consommateurs qui les citent ; elles ne designent plus aucune entree du registre.

// NomPrecisionParArmePasseSautee nomme un repli que l'hote declare et compte lui-meme.
//
// Deprecated: le registre du decodeur ne declare plus ce repli (aucun site ici, la passe est tenue
// par l'hote) ; la constante est gardee pour la compatibilite.
const NomPrecisionParArmePasseSautee = fallback.NomPrecisionParArmePasseSautee

// NomDistancesDeToucheDesactivees nomme un repli que l'hote declare et compte lui-meme.
//
// Deprecated: le registre du decodeur ne declare plus ce repli (aucun site ici, la degradation est
// tenue par l'hote) ; la constante est gardee pour la compatibilite.
const NomDistancesDeToucheDesactivees = fallback.NomDistancesDeToucheDesactivees
