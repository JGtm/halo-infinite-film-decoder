package replay

// score_timeline_figees.go — LES VALEURS FIGEES DES MANCHES TERMINEES, A COTE DES COURBES DES JOUEURS.
//
// A chaque manche terminee, le film fige la ligne de chaque joueur : la valeur de chaque emplacement du
// tableau des scores a la fin de la manche ([objectives.FrozenRoundsByOwner]). Le calque les publie par
// joueur et par manche, telles que le film les ecrit ; il ne les ajoute a aucune courbe. La derniere manche
// d'un match n'est jamais figee : elle n'y figure pas, et un match a une manche n'en a aucune.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// FrozenRound porte les valeurs figees d'une manche terminee d'un joueur.
type FrozenRound struct {
	// Round est la manche terminee, 0-based.
	Round int `json:"round"`
	// Values porte, par nom ([objectives.FrozenStats]), la valeur que le film a figee, entiere et dans l'unite
	// du film : `kda` et `accuracy` en centiemes (`accuracy` : touches / tirs, tronque, en centiemes de pour
	// cent), `averageLifeDuration` en dixiemes de seconde, les autres en unites. Une valeur que le film
	// declare par defaut vaut 0.
	Values map[string]int `json:"values"`
}

// FrozenRoundsCoverage dit ce que vaut la lecture des valeurs figees d'un match a plusieurs manches.
type FrozenRoundsCoverage struct {
	// Read est le nombre de manches terminees pour lesquelles au moins un joueur publie porte ses valeurs figees.
	Read int `json:"read"`
	// Expected est le nombre de manches terminees que le film doit figer : les manches retenues moins la
	// derniere.
	Expected int `json:"expected"`
	// PlayersWithout est le nombre de joueurs publies qui ne portent aucune manche figee : joueurs arrives
	// dans la derniere manche, et tous les joueurs quand les entrees du tableau ne nomment pas ses lignes
	// (le pont d'identite des slots n'atteint pas le niveau d'une manche, cf. `statsWithheld`).
	PlayersWithout int `json:"playersWithout"`
	// Discarded est le nombre de couples (joueur, manche) lus pour une manche qui n'est pas une manche
	// retenue suivie d'une autre : une manche qui n'est pas terminee ne se fige pas, ils ne sont pas publies.
	Discarded int `json:"discarded,omitempty"`
}

// frozenRoundsOf rend les manches figees des joueurs nommes par les entrees du tableau, par xuid, et le nombre
// de couples ecartes. `recs` sont les enregistrements dont les slots sont ceux d'`identity`
// ([objectives.EntryResolver.Split]).
func frozenRoundsOf(recs []types.StatRecord, identity map[int]string) (map[string][]FrozenRound, int) {
	byOwner, ecartes := objectives.FrozenRoundsByOwner(recs, func(r types.StatRecord) string { return identity[r.Slot] })
	out := make(map[string][]FrozenRound, len(byOwner))
	for xuid, rounds := range byOwner {
		for _, fr := range rounds {
			v := make(map[string]int, len(fr.Values))
			for name, x := range fr.Values {
				v[name] = int(x)
			}
			out[xuid] = append(out[xuid], FrozenRound{Round: fr.Round, Values: v})
		}
	}
	return out, ecartes
}

// attachFrozenRounds pose sur chaque joueur publie ses manches figees.
func attachFrozenRounds(players []PlayerScore, byXUID map[string][]FrozenRound) {
	for i := range players {
		players[i].FrozenRounds = byXUID[players[i].XUID]
	}
}

// frozenRoundsCoverage rend la couverture des valeurs figees d'un match de `rounds` manches retenues, nil
// sur un match a une manche : il n'en a aucune a figer.
func frozenRoundsCoverage(players []PlayerScore, rounds, discarded int) *FrozenRoundsCoverage {
	if rounds <= 1 {
		return nil
	}
	cov := &FrozenRoundsCoverage{Expected: rounds - 1, Discarded: discarded}
	lues := map[int]bool{}
	for _, p := range players {
		if len(p.FrozenRounds) == 0 {
			cov.PlayersWithout++
		}
		for _, fr := range p.FrozenRounds {
			lues[fr.Round] = true
		}
	}
	cov.Read = len(lues)
	return cov
}
