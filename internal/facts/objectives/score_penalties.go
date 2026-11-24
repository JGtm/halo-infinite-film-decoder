package objectives

import (
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// score_penalties.go — LA PENALITE D'UN SUICIDE OU D'UNE TRAHISON DANS LA COURBE DU SCORE PERSONNEL.
//
// # LE DEFAUT
//
// Le jeu retire des points du score personnel d'un joueur qui se suicide ou qui en tue un de son camp : le film
// ecrit alors une emission dont le score est plus bas que la precedente. La suite d'un siege est lue par la plus
// longue sous-suite non decroissante ([longestRun]), qui prend cette baisse pour du bruit : la courbe publiee ne
// recule jamais, et quand la penalite est le dernier mouvement d'un joueur son total garde la valeur d'avant la
// penalite (cent points de trop contre le score de l'API). Un score qui tombe sous zero (un joueur penalise avant
// d'avoir marque) etait de plus jete comme un ancrage parasite ([emissionHorsDomaine]).
//
// # LE FAIT DU FILM QUI DISTINGUE UNE PENALITE
//
// La penalite n'est pas une baisse quelconque : l'emission qui la porte incremente, DANS LA MEME EMISSION, le
// compteur de suicides (composant [suicideBetrayalComp], valeur A) ou de trahisons (valeur B) du siege, et le score
// tombe de [penaltyPoints] par unite du compteur. Les deux autres baisses du score personnel n'ont pas ce fait :
// le vidage de la ligne d'un joueur qui part (le score retombe a zero, aucun compteur ne bouge) et le changement
// d'occupant d'un siege ([seatCuts]). Une baisse n'est une penalite que si le compteur monte dans la meme emission
// ET que la baisse vaut exactement ses unites ; toute autre baisse reste ecartee comme avant.
//
// [penaltyPoints] est le prix que les scripts des modes donnent aux deux reponses (`playerSuicidePersonalScore`,
// `betrayalPersonalScore` : -100 dans la table des reponses de jeu, tous groupes) : ce n'est pas un seuil, c'est la
// valeur de la penalite, et l'egalite exacte de la baisse avec elle en est la confirmation.
//
// # COMMENT LA SUITE LA PREND
//
// Chaque penalite confirmee ([penaltyMarks]) est reprise dans la suite : les valeurs d'un siege sont reportees a leur
// valeur BRUTE (la valeur lue plus cent points par penalite deja subie), la plus longue sous-suite non decroissante
// est choisie sur cette valeur brute, qui ne recule que sur du bruit, puis chaque point retenu reprend sa valeur
// lue. La courbe rendue recule donc de la penalite a l'instant de l'emission qui la porte, et seulement la ou le
// film l'ecrit.

// suicideBetrayalComp : le composant du tableau qui compte les suicides (valeur A) et les trahisons (valeur B).
const suicideBetrayalComp = 4

// penaltyPoints : les points que le jeu retire du score personnel par suicide ou par trahison.
const penaltyPoints = 100

// penaltyMarks range, par manche puis par slot de joueur, les instants des penalites confirmees et le nombre
// d'unites que chacune porte (un suicide ou une trahison : 1).
type penaltyMarks map[int]map[int]map[int]int

// at rend les penalites du siege dans la manche (instant -> unites) ; nil quand il n'en a pas.
func (m penaltyMarks) at(round, slot int) map[int]int { return m[round][slot] }

// confirms dit que l'emission de l'instant `t` du siege est une penalite confirmee.
func (m penaltyMarks) confirms(round, slot, t int) bool { return m[round][slot][t] > 0 }

// penalised dit que le composant porte le score personnel : le seul dont la penalite se lit.
func (c StatComponent) penalised() bool { return c.Comp == personalScoreComp && c.SideB }

// penalties rend les penalites confirmees des enregistrements pour ce composant ; nil quand il n'est pas penalisable.
func (c StatComponent) penalties(recs []types.StatRecord) penaltyMarks {
	if !c.penalised() {
		return nil
	}
	return penaltyMarksOf(recs)
}

// penaltyMarksOf lit, siege par siege et manche par manche, les emissions qui portent une penalite confirmee : le
// compteur de suicides ou de trahisons monte de k unites dans une emission qui porte aussi le score personnel, et
// ce score vaut le precedent moins k fois [penaltyPoints]. Le precedent est le score de l'emission precedente du
// siege dans la manche, quel que soit son signe ; la premiere emission d'un siege n'a pas de precedent et ne confirme
// rien. Un compteur qui recule (un vidage de ligne) ne monte pas et ne confirme rien non plus.
//
// Les emissions hors des bornes de leur manche ([ResolveRoundBounds]) sont ecartees comme dans les suites.
func penaltyMarksOf(recs []types.StatRecord) penaltyMarks {
	type seat struct{ round, slot int }
	bornes := ResolveRoundBounds(recs)
	bySeat := map[seat][]types.StatRecord{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) || bornes.Excludes(r) {
			continue
		}
		if _, ok := r.Comps[suicideBetrayalComp]; !ok {
			if _, ok := r.Comps[personalScoreComp]; !ok {
				continue
			}
		}
		k := seat{r.Round, r.Slot}
		bySeat[k] = append(bySeat[k], r)
	}
	var out penaltyMarks
	for k, rs := range bySeat {
		for t, units := range penaltiesOfSeat(rs) {
			if out == nil {
				out = penaltyMarks{}
			}
			if out[k.round] == nil {
				out[k.round] = map[int]map[int]int{}
			}
			if out[k.round][k.slot] == nil {
				out[k.round][k.slot] = map[int]int{}
			}
			out[k.round][k.slot][t] = units
		}
	}
	return out
}

// penaltiesOfSeat rend les penalites confirmees d'un siege dans une manche : instant -> unites.
func penaltiesOfSeat(rs []types.StatRecord) map[int]int {
	slices.SortStableFunc(rs, func(a, b types.StatRecord) int { return a.TimeMS - b.TimeMS })
	var out map[int]int
	var suicides, betrayals, prev int64
	seen := false
	for _, r := range rs {
		var units int64
		if w, ok := r.Comps[suicideBetrayalComp]; ok {
			units = (w.A - suicides) + (w.B - betrayals)
			suicides, betrayals = w.A, w.B
		}
		v, ok := r.Comps[personalScoreComp]
		if !ok {
			continue
		}
		if seen && units > 0 && v.B == prev-penaltyPoints*units {
			if out == nil {
				out = map[int]int{}
			}
			out[r.TimeMS] = int(units)
		}
		prev, seen = v.B, true
	}
	return out
}

// penalisedRun rend la plus longue sous-suite non decroissante des valeurs BRUTES d'une suite, chaque point reprenant
// sa valeur lue : les penalites confirmees (`pens`, instant -> unites) ne comptent pas comme des baisses. Sans
// penalite c'est exactement [longestRun]. Un instant n'est repris qu'une fois, a son premier point.
func penalisedRun(pts []types.ScorePoint, strict bool, pens map[int]int) []types.ScorePoint {
	if len(pts) == 0 || len(pens) == 0 || strict {
		return longestRun(pts, strict)
	}
	gross := make([]types.ScorePoint, len(pts))
	var cum int64
	for i, p := range pts {
		if i == 0 || p.TimeMS != pts[i-1].TimeMS {
			cum += int64(pens[p.TimeMS])
		}
		gross[i] = types.ScorePoint{TimeMS: p.TimeMS, Slot: p.Slot, Value: p.Value + cum*penaltyPoints}
	}
	idx := longestRunIndexes(gross, false)
	out := make([]types.ScorePoint, len(idx))
	for i, k := range idx {
		out[i] = pts[k]
	}
	return out
}
