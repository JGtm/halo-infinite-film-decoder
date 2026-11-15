package objectives

import (
	"slices"
	"sort"
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entry_identity.go — QUI EST LE JOUEUR D'UNE LIGNE DU TABLEAU DES SCORES, PAR LECTURE.
//
// Chaque entite du statborg porte son entree : l'identifiant moteur du joueur dont elle est la ligne
// ([types.StatborgEntry]). Les paquets de type 8 listent, sous cet identifiant, le joueur qui le tient
// et pendant quelles plages ([types.RosterListing]). Le joueur d'un enregistrement est donc celui que
// les plages listent sous l'identifiant de son entite a son instant : aucune proximite de temps, aucun
// appariement de compteurs, la meme association pour toutes les manches d'un film.

// microsecondesParMS convertit l'instant d'un enregistrement (ms) sur l'horloge des paquets (us).
const microsecondesParMS = 1000

// slotVirtuelMin est le premier slot des lignes nommees par [EntryResolver.Split] : au-dela de tout slot du
// tableau des scores, pour qu'aucun slot reel ne s'y confonde.
const slotVirtuelMin = 1000

// EntryResolver nomme les enregistrements du tableau des scores par les entrees des entites et les plages
// de listage des joueurs.
type EntryResolver struct {
	// entrees par slot d'entite, triees par instant.
	entrees map[int][]types.StatborgEntry
	// plages par identifiant moteur.
	plages map[int][]types.RosterListing
	// originUS est l'horodatage du premier paquet du film : l'origine des instants des enregistrements.
	originUS uint64
}

// NewEntryResolver range les entrees et les plages. Les lignes d'equipe et les plages sans siege ne
// nomment personne. L'appelant s'assure d'abord que les entrees prouvent la disposition du tableau ([EntriesProveLayout]).
func NewEntryResolver(entries []types.StatborgEntry, listings []types.RosterListing, originUS uint64) *EntryResolver {
	r := &EntryResolver{entrees: map[int][]types.StatborgEntry{}, plages: map[int][]types.RosterListing{}, originUS: originUS}
	for _, e := range entries {
		if e.Kind == types.StatborgEntryPlayer {
			r.entrees[e.Slot] = append(r.entrees[e.Slot], e)
		}
	}
	for _, es := range r.entrees {
		slices.SortStableFunc(es, func(a, b types.StatborgEntry) int { return compareUS(a.TS, b.TS) })
	}
	for _, l := range listings {
		if l.Slot >= 0 {
			r.plages[l.Slot] = append(r.plages[l.Slot], l)
		}
	}
	return r
}

// EntriesProveLayout dit que les entrees lues sont celles du tableau des scores du film : le tableau s'ouvre sur ses deux lignes
// d'equipe, donc deux numeros d'equipe distincts se lisent en genre equipe. Quand la lecture ne tombe pas sur le composant d'entree
// (disposition du tableau que le film ne prouve pas), toutes les entites rendent la valeur nulle : un seul identifiant pour toutes les
// lignes de joueur, qui nommerait un seul joueur sur chacune ; une lecture fortuite de genre equipe ne prouve pas deux lignes.
func EntriesProveLayout(entries []types.StatborgEntry) bool {
	teams := map[int]bool{}
	for _, e := range entries {
		if e.Kind == types.StatborgEntryTeam {
			teams[e.ID] = true
		}
	}
	return len(teams) >= 2
}

func compareUS(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// idAt rend l'identifiant moteur de la ligne du slot d'entite `slot` a l'instant `us` : l'entree de la
// derniere image-cle qui ne le suit pas et qui n'est pas anterieure a `plancherUS` (le debut de la manche
// de l'enregistrement : une manche reattribue les lignes), ou a defaut la premiere qui le suit.
func (r *EntryResolver) idAt(slot int, us, plancherUS uint64) (int, bool) {
	es := r.entrees[slot]
	if len(es) == 0 {
		return 0, false
	}
	i := sort.Search(len(es), func(k int) bool { return es[k].TS > us })
	if i > 0 && es[i-1].TS >= plancherUS {
		return es[i-1].ID, true
	}
	if i < len(es) {
		return es[i].ID, true
	}
	return es[len(es)-1].ID, true
}

// holderAt rend le xuid que les plages listent sous l'identifiant `id` a l'instant `us`.
func (r *EntryResolver) holderAt(id int, us uint64) (uint64, bool) {
	var best *types.RosterListing
	for i := range r.plages[id] {
		l := &r.plages[id][i]
		if l.FromUS <= us && (l.ToUS == 0 || us < l.ToUS) && (best == nil || l.FromUS >= best.FromUS) {
			best = l
		}
	}
	if best == nil {
		return 0, false
	}
	return best.XUID, true
}

// Holder rend le joueur de la ligne du slot de record `slot` (le double du slot d'entite) a l'instant
// `timeMS` d'un enregistrement, la manche commencant a `roundStartMS`.
func (r *EntryResolver) Holder(slot, timeMS, roundStartMS int) (uint64, bool) {
	us := r.instantUS(timeMS)
	id, ok := r.idAt(slot/2, us, r.instantUS(roundStartMS))
	if !ok {
		return 0, false
	}
	return r.holderAt(id, us)
}

// instantUS pose un instant d'enregistrement (ms) sur l'horloge des paquets.
func (r *EntryResolver) instantUS(ms int) uint64 {
	return r.originUS + uint64(max(ms, 0))*microsecondesParMS //nolint:gosec // instant positif
}

// Split rend les enregistrements dont chaque ligne de joueur nommee prend un slot VIRTUEL propre a
// (slot, joueur), et la table slot virtuel -> xuid. Les enregistrements des lignes que personne ne
// tient, et ceux des equipes, gardent leur slot : les lecteurs de series en aval n'ont plus qu'une
// identite plate a poser, quel que soit le nombre de manches et de joueurs successifs d'un slot.
func (r *EntryResolver) Split(recs []types.StatRecord) ([]types.StatRecord, map[int]string) {
	type cle struct {
		slot int
		xuid uint64
	}
	virtuels := map[cle]int{}
	identite := map[int]string{}
	out := make([]types.StatRecord, len(recs))
	debuts := RoundStartsMS(recs)
	for i, rec := range recs {
		out[i] = rec
		if IsTeamSlot(rec.Slot) {
			continue
		}
		xuid, ok := r.Holder(rec.Slot, rec.TimeMS, debuts[rec.Round])
		if !ok {
			continue
		}
		k := cle{rec.Slot, xuid}
		v, vu := virtuels[k]
		if !vu {
			v = slotVirtuelMin + len(virtuels)
			virtuels[k] = v
			identite[v] = strconv.FormatUint(xuid, 10)
		}
		out[i].Slot = v
	}
	return out, identite
}
