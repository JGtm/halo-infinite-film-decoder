package decfilm

// presence.go — QUI EST LISTE DANS LA SESSION, ET QUI COMMANDE QUEL CORPS, PAR LA FACADE.
//
// Les deux faits sont lus par la grammaire ([grammar.ScanRosterListings], le canal des etats de mouvement de
// la marche des trames) et persistent dans les faits du film ([replay.FilmFactsFile]).

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// RosterListing : une plage pendant laquelle un humain est liste sous un siege dans la population de la session
// (paquets de type 8) ; `ToUS` vaut 0 quand aucun paquet ne la ferme.
type RosterListing = types.RosterListing

// RosterListingStats : ce que la lecture des paquets de type 8 a rencontre ; `Scanned` faux, aucune plage n est lue.
type RosterListingStats = types.RosterListingStats

// UnitControlRead : une lecture du joueur aux commandes d un bipede (`unit-control`) ; `Controller` vaut -1
// quand aucun joueur ne le commande.
type UnitControlRead = types.UnitControlRead

// UnitControlStats : ce que la lecture d `unit-control` a rencontre ; `Scanned` faux, aucune lecture n est faite.
type UnitControlStats = types.UnitControlStats

// ScanRosterListings lit les paquets de type 8 d un film DEJA CHARGE et rend, par humain, ses plages de listage
// dans l ordre des instants. Sans table des joueurs lisible, la liste est vide et `Scanned` faux.
func ScanRosterListings(film *source.Film) ([]RosterListing, RosterListingStats) {
	return grammar.ScanRosterListings(film)
}

// ScanUnitControls marche les trames du film du contexte `fc` et rend les lectures d `unit-control` de ses
// bipedes, dans l ordre des instants. Le contexte doit porter l entree de catalogue de la carte
// ([NewFilmContextForMap]) : sans elle la marche ne lit pas le film.
func ScanUnitControls(fc *grammar.FilmContext) ([]UnitControlRead, UnitControlStats, error) {
	m, err := grammar.ScanMarcheDesTramesAvec(fc, grammar.LecturesDeLaMarche{})
	return m.UnitControls, m.UnitControlStats, err
}
