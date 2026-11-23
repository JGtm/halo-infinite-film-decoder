package replay

import "github.com/JGtm/halo-infinite-film-decoder/types"

// filmfacts_gestes.go — LES GESTES DANS LE COMPLEMENT DE LA SECTION 1 : les comptes du canal, puis chaque
// geste (horodatage en delta, index de joueur de l auteur, -1 quand le message n en porte pas, geste). Une
// liste vide se relit nil.

func encodeGestes(w *gwriter, gs []types.GestureRead, st types.GestureStats) {
	for _, v := range []int{st.Frames, st.Closed, st.Stopped,
		st.Melee.Messages, st.Melee.Copies, st.Throw.Messages, st.Throw.Copies} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(gs)))
	var dernier uint64
	for _, g := range gs {
		w.u(g.TimestampUS - dernier)
		dernier = g.TimestampUS
		w.i(int64(g.Player))
		w.u(uint64(g.Kind))
	}
}

func decodeGestes(r *greader) ([]types.GestureRead, types.GestureStats) {
	var st types.GestureStats
	for _, v := range []*int{&st.Frames, &st.Closed, &st.Stopped,
		&st.Melee.Messages, &st.Melee.Copies, &st.Throw.Messages, &st.Throw.Copies} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(3)
	var out []types.GestureRead
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		out = append(out, types.GestureRead{TimestampUS: dernier, Player: int8(r.i()), //nolint:gosec // indice R(5) ou -1
			Kind: types.GestureKind(r.u())}) //nolint:gosec // geste 1 ou 2 ; une autre valeur se relit telle quelle, le document l ignore
	}
	return out, st
}
