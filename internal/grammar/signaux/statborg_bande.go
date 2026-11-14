package signaux

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/datums"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// statborg_bande.go — JUSQU'OU VONT LES SLOTS DE JOUEUR DU STATBORG, D'APRES LA TABLE DE DATUMS.
//
// Le statborg replique UNE entite par equipe puis une par joueur, dans les index de datum
// contigus a partir de 3 : les index 3 et 4 portent les deux equipes, les suivants les joueurs, un
// par siege. Chaque entite leve le composant 57 de son archetype (`statborg-entry-index-and-type`)
// dans le bitmap de son datum, et la table de datums que le film porte avant chaque image-cle
// ([datums.LireBlocDeDatums]) dit lesquelles sont vivantes. Les enregistrements d'entite portent
// 2 x cet index comme slot.
//
// Un joueur qui part libere son entite, que le suivant reprend : a une image-cle, l'index du milieu
// de la bande peut etre vacant alors que le suivant est pris. La bande tolere UN index vacant isole.

const (
	// statborgPremierIndex est l'index de la premiere entite de statborg (la premiere equipe).
	statborgPremierIndex = 3
	// statborgComposantEntree est le composant que toute entite de statborg leve : l'index et le
	// type de l'entree.
	statborgComposantEntree = 57
)

// BandeDuStatborg est ce que les tables de datums d'un film disent des slots de joueur.
type BandeDuStatborg struct {
	// SlotMax est le dernier slot de joueur lu : 2 x l'index de la derniere entite de la bande
	// contigue, la plus loin atteinte sur les tables de datums du film, bornee a [StatborgSlotMax].
	// [StatborgSlotBandeSure] quand aucune table ne porte la bande.
	SlotMax int
	// ParDefaut dit qu'aucune table ne portait la bande et que la lecture s'en tient aux huit
	// premiers joueurs : `repli_bande_statborg_par_defaut`. Faux sur un film absent.
	ParDefaut bool
	// Refuses est le nombre de blocs de type 1 que [datums.LireBlocDeDatums] a refuses.
	Refuses int
}

// LaBandeDuStatborg lit la bande des slots de joueur dans les tables de datums du film. Un film dont
// aucune table n'est lue, absente ou refusee, rend [StatborgSlotBandeSure] et le dit ([BandeDuStatborg.ParDefaut]).
func LaBandeDuStatborg(film *source.Film) BandeDuStatborg {
	if film == nil {
		return BandeDuStatborg{SlotMax: StatborgSlotBandeSure}
	}
	var b BandeDuStatborg
	best := 0
	for pos := range film.NumChunks() {
		for _, p := range film.Packets(pos) {
			if p.Type != int(datums.PacketTypeDatums) {
				continue
			}
			bloc, err := datums.LireBlocDeDatums(p.Payload)
			if err != nil {
				b.Refuses++
				continue
			}
			best = max(best, 2*dernierIndexDeBande(bloc.Entrees))
		}
	}
	b.SlotMax = min(max(best, StatborgSlotBandeSure), StatborgSlotMax)
	b.ParDefaut = best == 0
	return b
}

// dernierIndexDeBande rend l'index de la derniere entite de statborg de la bande contigue qui part
// de [statborgPremierIndex], ou 0 quand cet index n'est pas une entite de statborg vivante.
func dernierIndexDeBande(entrees []datums.DatumEntry) int {
	vivante := func(i int) bool {
		return i < len(entrees) && entrees[i].Vivante() && entrees[i].Composant(statborgComposantEntree)
	}
	dernier := 0
	for i := statborgPremierIndex; i < len(entrees) && i <= StatborgSlotMax/2; i++ {
		switch {
		case vivante(i):
			dernier = i
		case vivante(i + 1):
			// un index vacant isole : l'entite d'un joueur parti, pas la fin de la bande.
		default:
			return dernier
		}
	}
	return dernier
}
