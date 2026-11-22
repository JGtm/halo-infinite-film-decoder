package decfilm

import "github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"

// replis_obsoletes.go — LES NOMS DE REPLI QUE PLUS RIEN NE DECLENCHE.
//
// Ces deux noms designaient des replis que l'hote decidait et comptait lui-meme. Ils n'ont plus
// d'entree au registre : un `Declenche` sur l'un d'eux est signale comme un repli hors registre.
// Les constantes gardent leur type et leur valeur pour que le code qui les cite compile encore.

// NomPrecisionParArmePasseSautee nommait la passe de precision par arme sautee par l'hote.
//
// Deprecated: ce repli n'est plus au registre ; la constante est retiree a la prochaine version
// majeure.
const NomPrecisionParArmePasseSautee fallback.Nom = "repli_precision_par_arme_passe_sautee"

// NomDistancesDeToucheDesactivees nommait les distances de touche desactivees par l'hote.
//
// Deprecated: ce repli n'est plus au registre ; la constante est retiree a la prochaine version
// majeure.
const NomDistancesDeToucheDesactivees fallback.Nom = "repli_distances_de_touche_desactivees"
