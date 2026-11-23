package replay

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// lireEquipementPorte lit l equipement porte : les objets ti=37 crees attaches, que la marche des
// trames a deja recueillis (grammar/equipement_porte.go), et la position a la creation des objets
// qu ils engendrent, aux bornes de la carte. CE QU ELLE TOUCHE DU CONTEXTE : le memo `equipementPorte`
// de la marche (ou une marche de repli qui l ecrit, si la marche des trames a echoue).
func (s *filmScan) lireEquipementPorte() lectureStats[[]types.CarriedEquipmentRead, types.CarriedEquipmentStats] {
	return lireAvecStats(grammar.ScanEquipementPorte(s.fc, &s.world))
}

// publierEquipementPorte journalise la lecture de l equipement porte, le pose et observe l etape
// `carriedEquipment`.
func (s *filmScan) publierEquipementPorte(lu lectureStats[[]types.CarriedEquipmentRead, types.CarriedEquipmentStats]) {
	reads, st := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "equipement porte illisible — rejeu sans energie d equipement ni pose de faille",
			"err", lu.err, "match_id", s.matchID)
		reads, st = nil, types.CarriedEquipmentStats{}
	}
	s.in.CarriedEquipment, s.in.CarriedEquipmentStats = reads, st
	s.opt.observe(s.ctx, "carriedEquipment", s.in.CarriedEquipment)
}
