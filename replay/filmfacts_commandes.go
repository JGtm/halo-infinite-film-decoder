package replay

// filmfacts_commandes.go — LES COMMANDES TENUES DANS LE COMPLEMENT DE LA SECTION 1 : chaque commande (debut
// en delta, duree, place, genre, bornes, entrees, trous) puis les compteurs de la lecture. Une liste vide se
// relit nil.

import "github.com/JGtm/halo-infinite-film-decoder/types"

func encodeCommandes(w *gwriter, hs []types.ControlHold, st types.ControlHoldStats) {
	w.u(uint64(len(hs)))
	var dernier uint64
	for _, h := range hs {
		w.u(h.StartUS - dernier)
		dernier = h.StartUS
		w.u(h.EndUS - h.StartUS)
		w.u(uint64(h.FilmIndex)) //nolint:gosec // index de controle, 0..31
		w.str(h.StartBound)
		w.str(h.EndBound)
		w.u(uint64(h.Entries)) //nolint:gosec // compte positif
		w.u(uint64(len(h.Holes)))
		for _, t := range h.Holes {
			w.u(t.StartUS - h.StartUS)
			w.u(t.EndUS - t.StartUS)
		}
	}
	for _, v := range []int{st.Holds, st.WithHole, st.InnerHoles, st.Entries} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

func decodeCommandes(r *greader) ([]types.ControlHold, types.ControlHoldStats) {
	n := r.compte(8)
	var out []types.ControlHold
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		h := types.ControlHold{StartUS: dernier}
		h.EndUS = h.StartUS + r.u()
		h.FilmIndex = int(r.u()) //nolint:gosec // ecrit depuis un index 0..31
		h.StartBound, h.EndBound = r.str(), r.str()
		h.Entries = int(r.u()) //nolint:gosec // compte ecrit positif
		nh := int(r.u())       //nolint:gosec // compte ecrit positif
		for j := 0; j < nh && r.err == nil; j++ {
			t := types.ContinuousFireHole{StartUS: h.StartUS + r.u()}
			t.EndUS = t.StartUS + r.u()
			h.Holes = append(h.Holes, t)
		}
		out = append(out, h)
	}
	var st types.ControlHoldStats
	for _, v := range []*int{&st.Holds, &st.WithHole, &st.InnerHoles, &st.Entries} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return out, st
}
