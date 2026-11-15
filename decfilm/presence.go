package decfilm

// presence.go — QUI EST LISTE DANS LA SESSION, ET QUI COMMANDE QUEL CORPS, ET QUAND IL EST SUPPRIME, PAR LA FACADE.
//
// Les trois faits sont lus par la grammaire ([grammar.ScanRosterListings], le canal des etats de mouvement et les records de
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

// StatborgEntry : de qui est une ligne du tableau des scores, lue a une image-cle ; `ID` est l'identifiant moteur du
// joueur (le siege des paquets de type 8) ou le numero de l'equipe, selon `Kind`.
type StatborgEntry = types.StatborgEntry

// StatborgEntryStats : ce que la lecture des entrees du tableau des scores a rencontre ; `Scanned` faux, aucune
// entree n'est lue.
type StatborgEntryStats = types.StatborgEntryStats

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

// ScanStatborgEntries lit, aux images-cles du film du contexte `fc`, l'entree de chaque entite du tableau des
// scores : de qui est la ligne. Sans registre du film, la liste est vide et `Scanned` faux.
func ScanStatborgEntries(fc *grammar.FilmContext) ([]StatborgEntry, StatborgEntryStats) {
	return grammar.ScanStatborgEntries(fc)
}

// BipedDeletion : la suppression d un bipede, le record DEL de son slot dans les trames delta.
type BipedDeletion = types.BipedDeletion

// BipedDeletionStats : ce que la lecture des suppressions de bipede a rencontre ; `DeletionsUnproven` compte les
// suppressions d un debut de liste non prouve, que le jeu n applique pas et que la lecture ne publie pas.
type BipedDeletionStats = types.BipedDeletionStats

// ScanBipedDeletions marche les trames du film du contexte `fc` et rend les suppressions de ses bipedes, dans
// l ordre des instants. Memes conditions que [ScanUnitControls].
func ScanBipedDeletions(fc *grammar.FilmContext) ([]BipedDeletion, BipedDeletionStats, error) {
	m, err := grammar.ScanMarcheDesTramesAvec(fc, grammar.LecturesDeLaMarche{})
	return m.BipedDeletions, m.BipedDeletionStats, err
}
