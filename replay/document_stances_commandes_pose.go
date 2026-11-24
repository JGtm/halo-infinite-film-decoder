package replay

// document_stances_commandes_pose.go — LA POSE DES COMMANDES TENUES SUR LE DOCUMENT : les trous qui coupent un
// intervalle, les episodes de vehicule qui le retirent, et la passe d assemblage ([assemblage.poserLesCommandes]).
// La decision de publier est dans document_stances_commandes.go.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ridesParSlot rend, par slot de bipede, les episodes de vehicule en frames [premiere, derniere], tries : le
// bipede ne replique plus sa position a bord, et la commande qu il lit n y commande pas le meme geste.
func ridesParSlot(vehicles []VehicleTrack) map[uint32][][2]int {
	out := map[uint32][][2]int{}
	for _, v := range vehicles {
		for _, r := range v.Rides {
			out[r.Slot] = append(out[r.Slot], [2]int{r.T0, r.T1})
		}
	}
	for _, l := range out {
		slices.SortFunc(l, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	}
	return out
}

// sansLesRides rend les passages [premiere, derniere frame] de [t0, t1] que les episodes (tries, bornes comprises)
// laissent a pied.
func sansLesRides(t0, t1 int, rides [][2]int) [][2]int {
	var parts [][2]int
	cur := t0
	for _, r := range rides {
		if r[1] < cur || r[0] > t1 {
			continue
		}
		if r[0] > cur {
			parts = append(parts, [2]int{cur, r[0] - 1})
		}
		cur = max(cur, r[1]+1)
		if cur > t1 {
			break
		}
	}
	if cur <= t1 {
		parts = append(parts, [2]int{cur, t1})
	}
	return parts
}

// sansLesTrous rend les passages [premiere, derniere frame] de l intervalle [f0, f1] que les trous laissent
// lus. Une frame n est muette que si un trou la couvre entiere (debut arrondi a la frame suivante, fin a la
// frame qui le contient), comme pour le tir continu ; un trou plus court qu une frame ne coupe rien.
func (in controlStanceInputs) sansLesTrous(f0, f1 int, holes []types.ContinuousFireHole) [][2]int {
	clock := in.pont.clock
	pieces := [][2]int{{f0, f1}}
	for _, h := range holes {
		a, z := clock.frame(h.StartUS+clock.step-1), clock.frame(h.EndUS)
		if z <= a {
			continue
		}
		last := pieces[len(pieces)-1]
		if a > last[1] || z <= last[0] {
			continue
		}
		pieces = pieces[:len(pieces)-1]
		if a-1 >= last[0] {
			pieces = append(pieces, [2]int{last[0], a - 1})
		}
		if z <= last[1] {
			pieces = append(pieces, [2]int{z, last[1]})
		}
		if len(pieces) == 0 {
			break
		}
	}
	return pieces
}

// poserLesCommandes ajoute les commandes tenues aux etats de mouvement du document, APRES la pose des places
// (le roster tient ses presences), les vies publiees et les vehicules.
func (a *assemblage) poserLesCommandes() {
	if !a.opt.ContinuousFireStats.Scanned || a.doc.Coverage.Stances == nil {
		return
	}
	add, cov := buildControlStances(controlStanceInputs{holds: a.opt.ControlHolds, stats: a.opt.ControlHoldStats,
		tracks: a.doc.Tracks, vehicles: a.doc.Vehicles,
		pont: pontDesPlaces{occ: a.reg.Occupants(), places: nouveauxTireursParPlace(a.doc.Roster),
			clock: replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks}}})
	st := a.doc.Coverage.Stances
	st.Control = &cov
	if len(add) == 0 {
		return
	}
	a.doc.Stances = append(a.doc.Stances, add...)
	sortStances(a.doc.Stances)
	st.Intervals = len(a.doc.Stances)
	if st.ByKind == nil {
		st.ByKind = map[string]int{}
	}
	lives := map[uint32]struct{}{}
	for _, s := range a.doc.Stances {
		lives[s.Slot] = struct{}{}
	}
	for _, s := range add {
		st.ByKind[s.Kind]++
	}
	st.Lives = len(lives)
}
