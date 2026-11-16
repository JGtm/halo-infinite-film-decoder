package grammar

// film_player_table_listages.go — LE SIEGE D'UN OCCUPANT DE LA TABLE DU FILM, LU AUX LISTES DE LA SESSION.
//
// La table de `chunk_00` dit QUI est assis au coup d'envoi ; son rang dit OU seulement quand le rang est
// l'index de participant. Il ne l'est pas toujours : sur un film, un occupant est rangé à la ligne 2 de la
// table et les paquets de type 8 le listent au siège 8, que la lecture des chunks, le tableau des équipes et
// les records de création confirment. Employée telle quelle, la table donnait à cet occupant un index que
// personne ne porte : il perdait son équipe, et sa place ne se comptait plus parmi celles du match.
//
// LES PAQUETS DE TYPE 8 ÉCRIVENT LE SIÈGE DE CHAQUE JOUEUR ([types.RosterListing.Slot]) : c'est la lecture
// directe de l'index. Quand ils listent tous les occupants de la table dès le premier paquet, chacun sous
// un siège que personne d'autre ne porte, ce siège REMPLACE le rang. La table reste la liste de ceux qui
// sont là au coup d'envoi ; elle n'a plus de vacant intercalé à départager, puisque plus aucun index ne
// vient du rang ([FilmPlayerTable.Lue]).

import (
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TableDuFilmAuxListages est ce que les lecteurs du film consomment : la table de `chunk_00` dont les sieges sont
// ceux des listes, les listes elles-memes et le nombre de sieges dont l'index differe du rang.
type TableDuFilmAuxListages struct {
	// Table est la table lue, sieges corriges ([SiegesParLesListages]) ; refusee, elle porte sa cause.
	Table FilmPlayerTable
	// Corriges est le nombre de sieges dont l'index a remplace le rang.
	Corriges int
	// Listages et StatsListages sont les plages de type 8 que la correction a lues.
	Listages      []types.RosterListing
	StatsListages types.RosterListingStats
}

// ScanFilmPlayerTableAuxListages est LA lecture de la table des joueurs que tout lecteur du film emploie : la table de
// `chunk_00` ([ScanFilmPlayerTable]) puis ses sieges lus aux listes de la session. L'erreur est celle de la lecture de la
// table et n'accompagne que son refus. Aucun lecteur n'appelle [ScanFilmPlayerTable] seul : il garderait le rang d'une
// ligne que les listes donnent a un autre siege (`TestAucunLecteurNeLitLaTableSansLesListages`).
func ScanFilmPlayerTableAuxListages(film *source.Film) (TableDuFilmAuxListages, error) {
	table, err := ScanFilmPlayerTable(film)
	out := TableDuFilmAuxListages{}
	out.Listages, out.StatsListages = ScanRosterListings(film)
	out.Table, out.Corriges = SiegesParLesListages(table, out.Listages, out.StatsListages)
	return out, err
}

// SiegesParLesListages rend la table dont les sieges sont ceux que les paquets de type 8 listent, et le
// nombre de sieges dont l'index differe du rang de la table. La table est rendue telle quelle (zero siege
// corrige) quand elle n'est pas lue, quand la lecture des listes n'a pas eu lieu, ou que l'une des
// conditions suivantes manque :
//
//  1. chaque occupant de la table est liste, et sa plage la plus ancienne s'ouvre au premier paquet de type 8
//     du film (il est la au coup d'envoi, comme la table le dit) ;
//  2. chaque plage porte un siege ;
//  3. les sieges listes sont deux a deux distincts.
//
// Une table dont tous les rangs sont deja les sieges listes est rendue telle quelle aussi : rien n'est a
// corriger, et son vacant intercale, s'il en a un, reste a departager par les records de creation.
func SiegesParLesListages(t FilmPlayerTable, listings []types.RosterListing, st types.RosterListingStats,
) (FilmPlayerTable, int) {
	if t.Refusal != FilmTableRead || len(t.Seats) == 0 || !st.Scanned || len(listings) == 0 {
		return t, 0
	}
	premierUS := ^uint64(0)
	for _, l := range listings {
		if l.Slot < 0 {
			return t, 0
		}
		premierUS = min(premierUS, l.FromUS)
	}
	sieges := map[uint64]int{}
	for _, l := range listings {
		if l.FromUS != premierUS {
			continue
		}
		if _, deja := sieges[l.XUID]; deja {
			return t, 0
		}
		sieges[l.XUID] = l.Slot
	}
	out := t
	out.Seats = slices.Clone(t.Seats)
	pris, changes := map[int]bool{}, 0
	for i, s := range out.Seats {
		siege, liste := sieges[s.XUID]
		if !liste || pris[siege] {
			return t, 0
		}
		pris[siege] = true
		if siege != s.FilmIndex {
			changes++
		}
		out.Seats[i].FilmIndex = siege
	}
	if changes == 0 {
		return t, 0
	}
	slices.SortStableFunc(out.Seats, func(a, b FilmPlayerSeat) int { return a.FilmIndex - b.FilmIndex })
	out.InterleavedVacant = false
	return out, changes
}
