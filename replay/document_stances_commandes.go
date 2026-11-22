package replay

// document_stances_commandes.go — LA COMMANDE DE DEPLACEMENT TENUE, PUBLIEE COMME UN ETAT DE MOUVEMENT : la
// commande non nulle, lue dans la vue de controle (`stances[]`, genre `moveCommand`).
//
// CE QUE LA LECTURE DONNE. La grammaire rend, par place et par genre, l intervalle ou la commande est tenue :
// de la premiere entree lue qui la pose a la premiere entree lue qui ne la pose plus, ses trous portes
// (`grammar/commandes_de_controle.go`). La place est l index de controle de l entree : l occupant de la place a
// cet instant est le joueur, et une vie du joueur porte l intervalle.
//
// CE QU ON EN FAIT. Un intervalle lu se coupe aux limites de chaque vie publiee dont le corps est celui de
// l occupant, et se tait sur les passages que la lecture n a pas atteints (les trous, comme pour le tir
// continu : une frame n est muette que si un trou la couvre entiere). Une commande lue dont aucune vie
// publiee ne recouvre l intervalle, ou dont aucune des vies qui le recouvrent n est celle de l occupant, n est
// pas publiee et se compte. Rien n est devine : jamais de serie brute des valeurs, jamais un intervalle sans
// la lecture qui le borne.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// StanceControlCoverage dit ce que la lecture des commandes tenues a rendu et ce que la publication en a fait.
//
// L IDENTITE : holds = published + noTrack + noPlayer. Une commande publiee donne un intervalle par vie et par
// passage lu ; `intervals` les compte, et `coverage.stances.byKind` les ventile.
type StanceControlCoverage struct {
	// Holds : les commandes tenues lues ; WithHole : celles qu un trou touche ; InnerHoles : leurs passages
	// interieurs muets.
	Holds      int `json:"holds"`
	WithHole   int `json:"withHole"`
	InnerHoles int `json:"innerHoles"`
	// Entries : les entrees de controle lues qui portent le bloc de commande.
	Entries int `json:"entries"`
	// Published : les commandes qui donnent au moins un intervalle ; Intervals : les intervalles publies.
	Published int `json:"published"`
	Intervals int `json:"intervals"`
	// NoTrack : les commandes qu aucune vie publiee ne recouvre ; NoPlayer : celles que recouvrent des vies
	// dont aucune n est celle de l occupant de la place.
	NoTrack  int `json:"noTrack"`
	NoPlayer int `json:"noPlayer"`
}

// controlStanceInputs porte ce dont la publication des commandes a besoin.
type controlStanceInputs struct {
	holds  []types.ControlHold
	stats  types.ControlHoldStats
	tracks []Track
	occ    occupantsDesSlots
	places tireursParPlace
	clock  replayClock
}

// buildControlStances replie les commandes tenues en intervalles par vie et rend leur couverture.
func buildControlStances(in controlStanceInputs) ([]Stance, StanceControlCoverage) {
	cov := StanceControlCoverage{Holds: in.stats.Holds, WithHole: in.stats.WithHole, InnerHoles: in.stats.InnerHoles,
		Entries: in.stats.Entries}
	if in.clock.step == 0 {
		return nil, cov
	}
	var out []Stance
	for _, h := range in.holds {
		made, overlap := in.publier(h, &out)
		switch {
		case made > 0:
			cov.Published++
			cov.Intervals += made
		case overlap:
			cov.NoPlayer++
		default:
			cov.NoTrack++
		}
	}
	sortStances(out)
	return out, cov
}

// publier pose la commande `h` sur les vies de son occupant, rend le nombre d intervalles posees et dit si
// une vie publiee recouvre l intervalle.
func (in controlStanceInputs) publier(h types.ControlHold, out *[]Stance) (made int, overlap bool) {
	f0, f1 := in.clock.frame(h.StartUS), in.clock.frame(h.EndUS)
	pieces := in.sansLesTrous(f0, f1, h.Holes)
	for _, tr := range in.tracks {
		a, z := max(f0, tr.StartFrame), min(f1, tr.EndFrame)
		if z < a {
			continue
		}
		overlap = true
		idx, _ := in.places.tireur(h.FilmIndex, a)
		if pi, ok := in.occ.indexA(tr.Slot, int64(in.clock.origin+uint64(a)*in.clock.step)); !ok || pi != idx { //nolint:gosec // frame positive
			continue
		}
		for _, p := range pieces {
			if t0, t1 := max(p[0], a), min(p[1], z); t0 <= t1 {
				*out = append(*out, Stance{Slot: tr.Slot, Kind: types.MovementMoveCommand, T0: t0, T1: t1})
				made++
			}
		}
	}
	return made, overlap
}

// sansLesTrous rend les passages [premiere, derniere frame] de l intervalle [f0, f1] que les trous laissent
// lus. Une frame n est muette que si un trou la couvre entiere (debut arrondi a la frame suivante, fin a la
// frame qui le contient), comme pour le tir continu ; un trou plus court qu une frame ne coupe rien.
func (in controlStanceInputs) sansLesTrous(f0, f1 int, holes []types.ContinuousFireHole) [][2]int {
	pieces := [][2]int{{f0, f1}}
	for _, h := range holes {
		a, z := in.clock.frame(h.StartUS+in.clock.step-1), in.clock.frame(h.EndUS)
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
// (le roster tient ses presences) et les vies publiees.
func (a *assemblage) poserLesCommandes() {
	if !a.opt.ContinuousFireStats.Scanned || a.doc.Coverage.Stances == nil {
		return
	}
	add, cov := buildControlStances(controlStanceInputs{holds: a.opt.ControlHolds, stats: a.opt.ControlHoldStats,
		tracks: a.doc.Tracks, occ: a.reg.Occupants(), places: nouveauxTireursParPlace(a.doc.Roster),
		clock: replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks}})
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
