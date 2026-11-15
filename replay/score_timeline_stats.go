package replay

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// score_timeline_stats.go — LES COMPTEURS NOMMES DU TABLEAU DES SCORES, PAR JOUEUR : la table
// `objectives.ScoreboardStats` posee sur la grille de frames, par slot, a la suite des quatre compteurs
// de base de `PlayerScore`. Un film a plusieurs manches que le pont des slots nomme n'en publie aucun ([buildPlayerScores]).

// namedStatSet porte les series d'un compteur nomme pour tous les slots de joueur.
type namedStatSet struct {
	stat objectives.ScoreboardStat
	set  scoreSeriesSet
}

// namedStatSets sont les series de tous les compteurs nommes, dans l'ordre de la table.
type namedStatSets []namedStatSet

// loadNamedStats decode les series de chaque compteur nomme. Un maximum ne porte que ses courbes
// par manche : le cumul de plusieurs maximums ne veut rien dire.
func loadNamedStats(recs []types.StatRecord, cons *objectives.ReplisALaConsultation) namedStatSets {
	stats := objectives.ScoreboardStats()
	out := make(namedStatSets, 0, len(stats))
	for _, st := range stats {
		set := scoreSeriesSet{byRound: objectives.SeriesByRound(recs, st.Component, false, cons)}
		if !st.Peak {
			set = loadScoreSeries(recs, st.Component, false, cons)
		}
		out = append(out, namedStatSet{stat: st, set: set})
	}
	return out
}

// at rend les compteurs nommes publiables d'un slot, par nom ; nil quand aucun n'a de point.
func (n namedStatSets) at(slot int, c scoreClock) map[string]ScoreSeries {
	var out map[string]ScoreSeries
	for _, s := range n {
		series := s.set.at(slot, c)
		if series.empty() {
			continue
		}
		if out == nil {
			out = map[string]ScoreSeries{}
		}
		out[s.stat.Name] = series
	}
	return out
}

// namedStatsByXUID rend, par xuid, les compteurs nommes que `owner` nomme manche par manche, par nom ; un maximum ne
// porte que ses courbes par manche.
func namedStatsByXUID(recs []types.StatRecord, owner slotOwner, c scoreClock) map[string]map[string]ScoreSeries {
	out := map[string]map[string]ScoreSeries{}
	for _, st := range objectives.ScoreboardStats() {
		for xuid, byRound := range playerRoundsByXUID(recs, st.Component, owner, c.cons, !st.Peak) {
			series := ScoreSeries{Rounds: scoreRoundsOf(byRound, c)}
			if !st.Peak {
				series = seriesOfRounds(byRound, c)
			}
			if series.empty() {
				continue
			}
			if out[xuid] == nil {
				out[xuid] = map[string]ScoreSeries{}
			}
			out[xuid][st.Name] = series
		}
	}
	return out
}
