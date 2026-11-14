package replay

import "github.com/JGtm/halo-infinite-film-decoder/highlightevent"

// filmfacts_medailles.go — LES MEDAILLES DANS LE COMPLEMENT DE LA SECTION 1 : le temoin de lecture,
// puis la liste (xuid, instant, poids de tri, indice de medaille). Une liste vide se relit nil.

func encodeMedailles(w *gwriter, lues bool, medailles []highlightevent.MedalAward) {
	w.bool8(lues)
	w.u(uint64(len(medailles)))
	for _, m := range medailles {
		w.u(m.XUID)
		w.i(m.TimeMS)
		w.u(uint64(m.TypeHint))
		w.u(uint64(m.MedalType))
	}
}

func decodeMedailles(r *greader) (bool, []highlightevent.MedalAward) {
	lues := r.bool8()
	n := r.compte(4)
	if n == 0 {
		return lues, nil
	}
	out := make([]highlightevent.MedalAward, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		out = append(out, highlightevent.MedalAward{XUID: r.u(), TimeMS: r.i(), TypeHint: int(r.u()), MedalType: int(r.u())})
	}
	return lues, out
}
