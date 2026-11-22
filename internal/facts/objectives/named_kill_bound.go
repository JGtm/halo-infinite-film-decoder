package objectives

import "github.com/JGtm/halo-infinite-film-decoder/types"

// named_kill_bound.go — LA GARDE DES COMPTEURS NOMMES : AUCUN NE DEPASSE LES ELIMINATIONS DU JOUEUR DANS LA MANCHE.
//
// Deux bornes que le film donne lui-meme, et qu'aucun match reel ne viole : un compteur d'eliminations nomme
// (tirs a la tete, corps a corps, grenades, armes lourdes) compte des eliminations, donc il ne depasse pas les
// eliminations du joueur dans la manche ; la plus longue serie ne compte que des eliminations consecutives, donc elle
// ne les depasse pas non plus. Les eliminations sont celles que le tableau publie pour le meme slot et la meme manche.
//
// QUAND ELLE S'APPLIQUE. AVANT la selection de la suite ([StatComponent.series]) : une emission mal alignee qui porte
// une valeur impossible ne prend pas la place de l'emission juste en gagnant le departage entre suites de meme longueur.
// Une emission est ecartee EN ENTIER des qu'un canal borne de son composant depasse la borne : les valeurs d'un
// composant sortent de la meme emission, et un canal impossible prouve qu'elle est mal alignee ([withoutNegativeChannels]).
//
// CE QUI RESTE. Chaque emission ecartee est comptee, par compteur, dans [ReplisALaConsultation.EmissionsGardees] ;
// un compteur dont toutes les emissions sont ecartees est absent du tableau, jamais mis a zero. Un slot sans
// emission d'eliminations dans la manche a zero eliminations : un compteur a zero n'est jamais emis.

// killBounds rend, par slot puis par manche, les eliminations du joueur : la derniere valeur de la serie d'eliminations
// telle que le tableau la publie (le vidage de la ligne d'un joueur qui part n'y compte pas).
func killBounds(recs []types.StatRecord) map[int]map[int]int64 {
	out := map[int]map[int]int64{}
	for slot, byRound := range SeriesByRound(recs, KillsComponent().ClearsDropped(), false, nil) {
		out[slot] = map[int]int64{}
		for round, pts := range byRound {
			if n := len(pts); n > 0 {
				out[slot][round] = pts[n-1].Value
			}
		}
	}
	return out
}

// killBoundedSides dit, pour un composant, laquelle de ses deux valeurs (A, B) porte un compteur borne par les
// eliminations ([StatComponent.KillBounded] d'une entree de [ScoreboardStats]).
func killBoundedSides(comp int) (a, b bool) {
	for _, st := range ScoreboardStats() {
		if !st.Component.KillBounded || st.Component.Comp != comp {
			continue
		}
		if st.Component.SideB {
			b = true
		} else {
			a = true
		}
	}
	return a, b
}

// withinKillBound retire des emissions `pairs` du composant de `c`, pour les manches reelles, celles dont une valeur
// bornee depasse les eliminations du joueur dans la manche, et note chacune dans `cons`. Un (slot, manche) dont
// toutes les emissions sont ecartees disparait : une suite vide n'est jamais rendue.
func withinKillBound(recs []types.StatRecord, c StatComponent, pairs map[int]map[int][]valuePair,
	cons *ReplisALaConsultation) map[int]map[int][]valuePair {
	boundedA, boundedB := killBoundedSides(c.Comp)
	if !c.KillBounded || (!boundedA && !boundedB) {
		return pairs
	}
	bounds, real := killBounds(recs), RealRounds(recs)
	for slot, byRound := range pairs {
		for round, pts := range byRound {
			if !real[round] {
				continue
			}
			limit := bounds[slot][round]
			kept := pts[:0]
			for _, p := range pts {
				if (boundedA && p.a > limit) || (boundedB && p.b > limit) {
					cons.noterEmissionGardee(c.key(), slot, round, p.t)
					continue
				}
				kept = append(kept, p)
			}
			if len(kept) == 0 {
				delete(byRound, round)
				continue
			}
			byRound[round] = kept
		}
	}
	return pairs
}
