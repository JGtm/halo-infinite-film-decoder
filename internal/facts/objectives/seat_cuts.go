package objectives

// L ordre de ces imports entre dans l empreinte de la couche (revision/cadre.go) : ne pas regrouper.
import (
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// seat_cuts.go — LES SUITES D'UN SIEGE QUI CHANGE D'OCCUPANT : UNE SUITE PAR OCCUPANT, PAS UNE SEULE POUR LE SIEGE.
//
// # LE DEFAUT
//
// Un compteur de siege est lu par la plus longue sous-suite non decroissante de la manche ([longestRun]).
// Quand un joueur quitte et qu'un autre prend le siege, le film remet les compteurs du siege a zero et le
// nouvel occupant repart de la : la suite du siege recule, [longestRun] n'en garde qu'une des deux (a
// longueur egale, la derniere) et les actions du partant (ses vols, ses prises, ses captures) disparaissent
// du compte, alors que le film les a ecrites.
//
// # LA FRONTIERE EST UN FAIT DU FILM, ET ELLE EST DEJA LUE
//
// L'instant ou le siege change d'occupant est celui de la retombee de ses compteurs de base, confirmee par
// la reprise du comptage d'un autre occupant ([seatChanges]) : le pont d'identite s'en sert deja pour
// nommer chaque occupation ([Occupation], [RoundIdentity.At]). Cet instant est celui ou la plage de listage
// du partant se ferme dans les paquets de la session, a quelques millisecondes pres (mesure : 1 a 7 ms sur
// quatre films, quatre changements sur quatre). Une retombee que rien ne confirme (une ligne videe, une fin
// de manche, un depart sans remplacant) ne coupe rien : [seatChanges] la laisse en place.
//
// COUPER LA SUITE A CES INSTANTS ET NOMMER PAR [RoundIdentity.At] SONT LA MEME DECISION : une action datee
// avant la frontiere est celle de l'occupant d'avant, une action datee a partir d'elle celle de l'occupant
// d'apres, comme [Occupation.Covers] (borne basse incluse, haute exclue).
//
// # OU LA COUPURE S'APPLIQUE
//
// Aux seules suites dont le lecteur nomme PAR L'INSTANT : les evenements nommes ([NamedEventsFrom]). Les
// lecteurs qui attribuent la suite d'un siege ENTIER a un seul joueur (le pont d'identite, les totaux par
// siege, les courbes que le pont plat attribue par slot) gardent la suite du siege : la couper les ferait
// attribuer a un joueur la somme de deux occupants. Les courbes nommees par les entrees du tableau des scores
// ([EntryResolver.Split]) sont deja coupees par occupant, par la plage de listage.

// seatCuts range, par manche puis par slot de joueur, les instants ou le siege change d'occupant.
type seatCuts map[int]map[int][]int

// seatCutsOf lit les changements d'occupant de tous les sieges des enregistrements. Un film sans siege
// recycle rend une table vide, et [seatCuts.of] rend alors nil partout : les suites sont celles d'avant.
func seatCutsOf(recs []types.StatRecord) seatCuts {
	out := seatCuts{}
	for round, bySlot := range coreEmissionsBySeat(recs) {
		for slot, ems := range bySlot {
			changes := seatChanges(ems)
			if len(changes) == 0 {
				continue
			}
			if out[round] == nil {
				out[round] = map[int][]int{}
			}
			out[round][slot] = changes
		}
	}
	return out
}

// of rend les instants de changement d'occupant du siege dans la manche ; nil quand il n'en a pas.
func (s seatCuts) of(round, slot int) []int { return s[round][slot] }

// longestRunBySeat rend la plus longue sous-suite de CHAQUE occupation du siege, mises bout a bout : la suite
// de l'occupant d'apres est decalee de la derniere valeur de l'occupant d'avant, si bien que la suite rendue
// reste croissante, que ses increments sont ceux des deux occupants, et que son dernier point est le total
// du siege (partant et arrivant). Une occupation sans emission n'ajoute rien.
//
// Sans frontiere (`cuts` vide), c'est exactement [longestRun] : un siege qui garde son occupant ne change pas.
func longestRunBySeat(pts []types.ScorePoint, cuts []int, strict bool) []types.ScorePoint {
	if len(cuts) == 0 {
		return longestRun(pts, strict)
	}
	var out []types.ScorePoint
	var offset int64
	from := 0
	for k := 0; k <= len(cuts); k++ {
		to := len(pts)
		if k < len(cuts) {
			to = from
			for to < len(pts) && pts[to].TimeMS < cuts[k] {
				to++
			}
		}
		if from < to {
			kept := longestRun(pts[from:to], strict)
			for _, p := range kept {
				out = append(out, types.ScorePoint{TimeMS: p.TimeMS, Slot: p.Slot, Value: p.Value + offset})
			}
			offset += kept[len(kept)-1].Value
		}
		from = to
	}
	return out
}
