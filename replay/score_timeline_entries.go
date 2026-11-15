package replay

// score_timeline_entries.go — LES JOUEURS DU TABLEAU DES SCORES, NOMMES PAR LA LECTURE DES ENTREES.
//
// Chaque ligne du tableau porte l'identifiant moteur du joueur dont elle est la ligne, et les paquets de type 8
// listent, sous cet identifiant, le joueur qui le tient et pendant quelles plages ([objectives.EntryResolver]).
// Le joueur d'un enregistrement se lit donc ; il n'y a plus ni appariement des compteurs aux lignes du match, ni
// instants de mort, ni identite par manche, et un film a plusieurs manches se nomme comme un autre. Sans entrees
// lues (film dont les images-cles ne portent pas le composant, plages non lues), le tableau se nomme par le pont
// d'identite des slots ([buildPlayerScores]).

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entryResolverOf rend le resolveur des entrees du film, ou nil quand elles ne se lisent pas : entrees ou plages non
// lues, ou aucune ligne de joueur.
func entryResolverOf(opt Options) *objectives.EntryResolver {
	if !opt.StatborgEntryStats.Scanned || !opt.Presences.ListingStats.Scanned || len(opt.StatborgEntries) == 0 {
		return nil
	}
	return objectives.NewEntryResolver(opt.StatborgEntries, opt.Presences.Listings, opt.FilmClockOriginUS)
}

// buildPlayerScoresOf rend les compteurs vivants des joueurs : nommes par les entrees quand elles se lisent, par le
// pont d'identite des slots sinon. Le second retour compte les joueurs que ce pont prive de compteurs nommes (film a
// plusieurs manches) ; il est nul quand les entrees nomment, et chaque joueur y est une seule entree, ses lignes successives
// mises bout a bout.
func buildPlayerScoresOf(recs []types.StatRecord, flat map[int]string, lines []types.PlayerLine,
	deaths []types.Death, entries *objectives.EntryResolver, c scoreClock) ([]PlayerScore, int) {
	if entries == nil {
		return buildPlayerScores(recs, flat, lines, deaths, c)
	}
	named, identity := entries.Split(recs)
	return buildPlayerScoresByRound(named, func(_, slot int) string { return identity[slot] }, c, true)
}

// scoreInputOfPlayedRounds rend l'entree du calque de score dont les enregistrements des manches que le film n'a pas jouees
// sont ecartes, et les numeros de ces manches ([objectives.SplitUnplayedRounds]). Sans issues lues (entrees non lues, aucune
// entite), l'entree est celle de l'appelant.
func scoreInputOfPlayedRounds(opt Options) (*ScoreInput, []int) {
	in := opt.Score
	if in == nil || !opt.StatborgEntryStats.Scanned || opt.StatborgEntryStats.Read == 0 {
		return in, nil
	}
	kept, unplayed := objectives.SplitUnplayedRounds(in.Records, opt.StatborgEntryStats.DecidedRounds)
	if len(unplayed) == 0 {
		return in, nil
	}
	played := *in
	played.Records = kept
	return &played, unplayed
}
