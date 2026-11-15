package replay

// filmfacts_presence.go — LA PRESENCE ET LE JOUEUR AUX COMMANDES DANS LE FICHIER DE FAITS (schema des
// faits 24) : les plages de listage des humains (paquets de type 8) et les lectures d `unit-control`,
// avec leurs denominateurs, au complement de la section 1 ([encodeComplementDesEntrees]), avant le
// verdict du fil des morts.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// encodeListagesDeRoster ecrit les plages (TRIEES par debut : delta d horodatage), puis les
// denominateurs. Une plage ouverte ecrit 0, une plage fermee l ecart a son debut plus un.
func encodeListagesDeRoster(w *gwriter, ls []types.RosterListing, st types.RosterListingStats) {
	w.u(uint64(len(ls)))
	var last uint64
	for _, l := range ls {
		w.u(l.FromUS - last)
		last = l.FromUS
		w.u(l.XUID)
		w.i(int64(l.Slot))
		if l.ToUS == 0 {
			w.u(0)
		} else {
			w.u(l.ToUS - l.FromUS + 1)
		}
	}
	w.bool8(st.Scanned)
	for _, v := range [...]int{st.Packets, st.Distinct, st.Refused, st.Entries} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

// decodeListagesDeRoster relit ce que [encodeListagesDeRoster] ecrit ; une liste vide se relit nil.
func decodeListagesDeRoster(r *greader) ([]types.RosterListing, types.RosterListingStats) {
	n := r.compte(4)
	var out []types.RosterListing
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		l := types.RosterListing{FromUS: last, XUID: r.u(), Slot: int(r.i())}
		if fin := r.u(); fin != 0 {
			l.ToUS = l.FromUS + fin - 1
		}
		out = append(out, l)
	}
	var st types.RosterListingStats
	st.Scanned = r.bool8()
	for _, p := range [...]*int{&st.Packets, &st.Distinct, &st.Refused, &st.Entries} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return out, st
}

// encodePresenceEtEntrees ecrit, dans l'ordre du format, les plages de listage, les lectures d `unit-control` et les entrees
// du tableau des scores de `g`.
func encodePresenceEtEntrees(w *gwriter, g *FilmFacts) {
	encodeListagesDeRoster(w, g.RosterListings, g.RosterListingStats)
	encodeControlesDeBipede(w, g.UnitControls, g.UnitControlStats)
	encodeEntreesDuStatborg(w, g.StatborgEntries, g.StatborgEntryStats)
}

// decodePresenceEtEntrees relit ce que [encodePresenceEtEntrees] ecrit.
func decodePresenceEtEntrees(r *greader, g *FilmFacts) {
	g.RosterListings, g.RosterListingStats = decodeListagesDeRoster(r)
	g.UnitControls, g.UnitControlStats = decodeControlesDeBipede(r)
	g.StatborgEntries, g.StatborgEntryStats = decodeEntreesDuStatborg(r)
}

// encodeControlesDeBipede ecrit les lectures d `unit-control` (TRIEES par instant : delta
// d horodatage), puis les denominateurs. Le controleur vaut -1 sans joueur.
func encodeControlesDeBipede(w *gwriter, cs []types.UnitControlRead, st types.UnitControlStats) {
	w.u(uint64(len(cs)))
	var last uint64
	for _, c := range cs {
		w.u(c.TimestampUS - last)
		last = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Chunk))       //nolint:gosec // rang de chunk positif
		w.u(uint64(c.PacketIndex)) //nolint:gosec // rang de paquet positif
		w.i(int64(c.Controller))
	}
	w.bool8(st.Scanned)
}

// decodeControlesDeBipede relit ce que [encodeControlesDeBipede] ecrit ; une liste vide se relit nil.
func decodeControlesDeBipede(r *greader) ([]types.UnitControlRead, types.UnitControlStats) {
	n := r.compte(5)
	var out []types.UnitControlRead
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		c := types.UnitControlRead{TimestampUS: last}
		c.Slot = uint32(r.u())     //nolint:gosec // ecrit depuis un identifiant de 13 bits
		c.Chunk = int(r.u())       //nolint:gosec // rang ecrit positif
		c.PacketIndex = int(r.u()) //nolint:gosec // rang ecrit positif
		c.Controller = int(r.i())
		out = append(out, c)
	}
	return out, types.UnitControlStats{Scanned: r.bool8()}
}
