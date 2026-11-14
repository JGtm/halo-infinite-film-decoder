package replay

// lectures_de_presence.go — QUI EST LISTE DANS LA SESSION, tel que l assemblage le consomme.

import (
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// LecturesDePresence regroupe la lecture des plages de listage des humains dans la population de la
// session (paquets de type 8) et celle du joueur aux commandes de chaque bipede (`unit-control`), avec leurs
// denominateurs. Les statistiques voyagent avec la liste : une liste
// vide sans elles ne distingue pas un film sans arrivee ni depart d un film dont la lecture n a pas eu
// lieu (`Scanned` faux).
type LecturesDePresence struct {
	Listings     []types.RosterListing
	ListingStats types.RosterListingStats
	Controls     []types.UnitControlRead
	ControlStats types.UnitControlStats
}

// listagesParXUID range les plages de listage par xuid en decimal, la forme de [RosterEntry.XUID], dans
// l ordre ou elles sont lues.
func listagesParXUID(ls []types.RosterListing) map[string][]types.RosterListing {
	if len(ls) == 0 {
		return nil
	}
	out := make(map[string][]types.RosterListing)
	for _, l := range ls {
		x := strconv.FormatUint(l.XUID, 10)
		out[x] = append(out[x], l)
	}
	return out
}
