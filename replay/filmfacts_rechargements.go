package replay

import "github.com/JGtm/halo-infinite-film-decoder/types"

// filmfacts_rechargements.go — LES RECHARGEMENTS D ARME DANS LE COMPLEMENT DE LA SECTION 1 : les comptes du
// canal, puis chaque rechargement (horodatage en delta, index de joueur de l auteur, -1 quand le message n en
// porte pas, nombre de copies retirees). Une liste vide se relit nil.

func encodeRechargements(w *gwriter, rs []types.ReloadRead, st types.ReloadStats) {
	for _, v := range []int{st.Frames, st.Closed, st.Stopped, st.Messages, st.Copies} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(rs)))
	var dernier uint64
	for _, m := range rs {
		w.u(m.TimestampUS - dernier)
		dernier = m.TimestampUS
		w.i(int64(m.Player))
		w.u(uint64(m.Copies)) //nolint:gosec // compte positif
	}
}

func decodeRechargements(r *greader) ([]types.ReloadRead, types.ReloadStats) {
	var st types.ReloadStats
	for _, v := range []*int{&st.Frames, &st.Closed, &st.Stopped, &st.Messages, &st.Copies} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(3)
	var out []types.ReloadRead
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		out = append(out, types.ReloadRead{TimestampUS: dernier, Player: int8(r.i()), //nolint:gosec // indice R(5) ou -1
			Copies: int(r.u())}) //nolint:gosec // compte ecrit positif
	}
	return out, st
}

// encodeCanauxDeLaMarche et decodeCanauxDeLaMarche : les canaux de la marche des trames que le complement de
// la section 1 porte, dans l ORDRE du format.
func encodeCanauxDeLaMarche(w *gwriter, g *FilmFacts) {
	encodeJeu(w, g)
	encodeDegats(w, g)
	encodeRechargements(w, g.Reloads, g.ReloadStats)
	encodeRepliques(w, g.Repliques)
	encodeGestes(w, g.Gestures, g.GestureStats)
	encodeCommandes(w, g.ControlHolds, g.ControlHoldStats)
	encodeArmesTirees(w, g.DrawnWeapons, g.DrawnWeaponStats)
	encodeVitalites(w, g.Vitals, g.VitalStats)
}

func decodeCanauxDeLaMarche(r *greader, g *FilmFacts) {
	decodeJeu(r, g)
	decodeDegats(r, g)
	g.Reloads, g.ReloadStats = decodeRechargements(r)
	g.Repliques = decodeRepliques(r)
	g.Gestures, g.GestureStats = decodeGestes(r)
	g.ControlHolds, g.ControlHoldStats = decodeCommandes(r)
	g.DrawnWeapons, g.DrawnWeaponStats = decodeArmesTirees(r)
	g.Vitals, g.VitalStats = decodeVitalites(r)
}
