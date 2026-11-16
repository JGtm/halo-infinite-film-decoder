package replay

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// filmfacts_degats.go — LES MESSAGES `damage_aftermath` DU FILM ET CE QU'ILS RETIRENT, DANS LE COMPLEMENT DE LA
// SECTION 1 : le témoin de lecture, les comptes du canal, puis chaque message (horodatage, effet, deux unités, index de
// joueur de l'auteur, échelle, gardes) suivi de ses points (bouclier et corps en dixièmes, drapeau « mesuré »). Une liste
// vide se relit nil.

func encodeDegats(w *gwriter, g *FilmFacts) {
	w.bool8(g.DamagesRead)
	for _, v := range []int{g.DamageStats.Trames, g.DamageStats.Terminees, g.DamageStats.Arretees, g.DamageStats.Messages, g.DamageStats.VitalesNonLues} {
		w.u(uint64(v))
	}
	w.u(uint64(len(g.Damages)))
	for i, d := range g.Damages {
		w.u(d.TS)
		w.u(uint64(d.Tag))
		w.u(uint64(d.Victime))
		w.u(uint64(d.GenVictime))
		w.u(uint64(d.Responsable))
		w.i(int64(d.Joueur))
		w.byte8(d.Echelle)
		var drapeaux byte
		for k, v := range []bool{d.ATag, d.AVictime, d.AResponsable} {
			if v {
				drapeaux |= 1 << k
			}
		}
		var p PointsDeDegat
		if i < len(g.DamagePoints) {
			p = g.DamagePoints[i]
		}
		if p.Mesure {
			drapeaux |= 1 << 3
		}
		w.byte8(drapeaux)
		w.u(uint64(p.Bouclier))
		w.u(uint64(p.Corps))
	}
}

func decodeDegats(r *greader, g *FilmFacts) {
	g.DamagesRead = r.bool8()
	st := &g.DamageStats
	for _, v := range []*int{&st.Trames, &st.Terminees, &st.Arretees, &st.Messages, &st.VitalesNonLues} {
		*v = int(r.u())
	}
	g.Damages, g.DamagePoints = nil, nil
	n := r.compte(10)
	for k := 0; k < n && r.err == nil; k++ {
		var d grammar.DegatLu
		d.TS = r.u()
		d.Tag = uint32(r.u())         //nolint:gosec // R(32) relu
		d.Victime = uint16(r.u())     //nolint:gosec // index d'unite relu
		d.GenVictime = uint8(r.u())   //nolint:gosec // generation sur deux bits
		d.Responsable = uint16(r.u()) //nolint:gosec // idem
		d.Joueur = int8(r.i())        //nolint:gosec // indice R(5) ou -1
		d.Echelle = r.byte8()
		drapeaux := r.byte8()
		d.ATag, d.AVictime, d.AResponsable = drapeaux&1 != 0, drapeaux&2 != 0, drapeaux&4 != 0
		p := PointsDeDegat{Mesure: drapeaux&8 != 0}
		p.Bouclier, p.Corps = uint16(r.u()), uint16(r.u()) //nolint:gosec // dixiemes de point, au plus 65535
		g.Damages = append(g.Damages, d)
		g.DamagePoints = append(g.DamagePoints, p)
	}
}
