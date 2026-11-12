package decfilm

// infos_de_match.go — LES INFOS DE MATCH QUE `chunk_00` PORTE, PAR LA FACADE.
//
// Le fait est lu par la grammaire (`grammar.ReadFilmIdentity`) et persiste avec l identite du film dans les
// faits ([replay.FilmFactsFile.Identity]) ; aucun consommateur n en depend dans ce module : le rejeu ne
// publie rien de neuf. Un champ que le film ne porte pas est absent (pointeur nil), jamais zero.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// InfosDeMatch : ce que `chunk_00` dit du match enregistre — la variante de partie (nom, description, score
// de victoire, duree, delai de reapparition, prolongation, dotation, options), les identifiants (asset et
// version) de la variante de carte et l heure de debut. `VarianteLue` dit si la variante a ete lue.
type InfosDeMatch = profile.InfosDeMatch

// ErrNoFilmIdentity : le `chunk_00` ne porte aucune section d identification (les films de formats anciens
// n en ont pas) : ses infos de match ne sont pas lues.
const ErrNoFilmIdentity = grammar.ErrNoFilmIdentity

// ErrChunk00Truncated : le tampon s arrete avant la fin de la section d identification.
const ErrChunk00Truncated = grammar.ErrChunk00Truncated

// ReadMatchInfo lit les infos de match d un `chunk_00` DEJA DECOMPRESSE (ce que rend [ReadFilmChunk]).
// Les erreurs sont typees ([ErrNoFilmIdentity], [ErrChunk00Truncated]), jamais une panique ni une
// valeur inventee : une variante que la grammaire ne lit pas en entier rend `VarianteLue` faux et aucun de
// ses champs.
func ReadMatchInfo(chunk0 []byte) (InfosDeMatch, error) {
	id, err := grammar.ReadFilmIdentity(chunk0)
	if err != nil {
		return InfosDeMatch{}, err
	}
	return id.InfosDeMatch, nil
}
