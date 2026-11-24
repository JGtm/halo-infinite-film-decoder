package replay

// equipment_placements_decode.go — LE BALAYAGE DES POSES ET DES EVENEMENTS 103, journalises (hors ligne).
// DEPLACEMENT PUR depuis `equipment_placements.go` (plan des durees d effets, B.1) pour tenir le seuil de
// 500 lignes : aucune ligne de logique n a change.

import (
	"context"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// decodeFilmPlacements décode les poses du film et JOURNALISE ce qu'il en est.
//
// TROIS SORTIES, TROIS PHRASES — et la distinction est le point. Un balayage impossible (film
// illisible) n'est pas un film sans équipement, et un film dont le découpage du bloc de
// réplication n'a pas été tranché n'est ni l'un ni l'autre : il aurait des poses, mais les lire
// à une largeur devinée rendrait du bruit. Les trois se lisent au journal, et la troisième se
// relit ensuite dans l'artefact (`coverage.placements.calibrated`).
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPlacements(ctx context.Context,
	fc *grammar.FilmContext, matchID string, worldRange *profile.Vec3Range,
) ([]types.EquipmentPlacement, grammar.EquipmentPlacementStats) {
	return journaliserPoses(ctx, matchID, decodeFilmPlacementsLecture(fc, worldRange))
}
