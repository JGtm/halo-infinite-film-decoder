package objectives

import (
	"math/bits"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// rounds_played.go — LES MANCHES QUE LE FILM A JOUEES, d'apres leurs issues.
//
// Le tableau des scores porte, par manche, une issue qui reste nulle tant que la manche n'est pas jouee
// ([types.StatborgEntryStats.DecidedRounds], lue aux images-cles). Un enregistrement de valeur dont le numero de manche
// depasse la derniere manche qui a une issue d'un rang n'appartient a aucune manche jouee : c'est un ancrage fortuit qui
// porte un numero de manche valide, et la decision des manches ([ResolveRounds]) le tient pour une manche entiere quand
// il en porte assez. Le rang de plus est celui de la manche en cours a la fin du film, dont l'issue n'est ecrite dans
// aucune image-cle quand le film s'arrete avant la suivante.

// SplitUnplayedRounds separe les enregistrements des manches que le film a jouees de ceux des manches qu'il n'a pas jouees, et
// rend les numeros de ces dernieres. `decided` est le masque des manches qui ont une issue.
func SplitUnplayedRounds(recs []types.StatRecord, decided uint32) (kept []types.StatRecord, unplayed []int) {
	last := bits.Len32(decided) // rang de la derniere manche jouee a l'issue connue, plus un
	kept = make([]types.StatRecord, 0, len(recs))
	seen := map[int]bool{}
	for _, r := range recs {
		if r.Round <= last {
			kept = append(kept, r)
			continue
		}
		seen[r.Round] = true
	}
	for round := range seen {
		unplayed = append(unplayed, round)
	}
	slices.Sort(unplayed)
	return kept, unplayed
}
