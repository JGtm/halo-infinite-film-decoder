package replay

// filmfacts_presence.go — LA PRESENCE ET LE JOUEUR AUX COMMANDES DANS LE FICHIER DE FAITS (schema des
// faits 24) : les plages de listage des humains (paquets de type 8) et les lectures d `unit-control`,
// avec leurs denominateurs, puis les suppressions de bipede (schema des faits 28), au complement de la section 1 ([encodeComplementDesEntrees]), avant le
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

// encodePresenceEtEntrees ecrit, dans l'ordre du format, les plages de listage, les lectures d `unit-control` et les suppressions de bipede, puis les
// entrees du tableau des scores de `g`.
func encodePresenceEtEntrees(w *gwriter, g *FilmFacts) {
	encodeListagesDeRoster(w, g.RosterListings, g.RosterListingStats)
	encodeCommandeDesBipedes(w, g)
	encodeEntreesDuStatborg(w, g.StatborgEntries, g.StatborgEntryStats)
	encodeApparences(w, g.Appearances, g.AppearanceStats)
}

// decodePresenceEtEntrees relit ce que [encodePresenceEtEntrees] ecrit.
func decodePresenceEtEntrees(r *greader, g *FilmFacts) {
	g.RosterListings, g.RosterListingStats = decodeListagesDeRoster(r)
	decodeCommandeDesBipedes(r, g)
	g.StatborgEntries, g.StatborgEntryStats = decodeEntreesDuStatborg(r)
	g.Appearances, g.AppearanceStats = decodeApparences(r)
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

// encodeCommandeDesBipedes ecrit ce que la marche lit du corps d un bipede : le joueur aux commandes, puis la suppression.
func encodeCommandeDesBipedes(w *gwriter, g *FilmFacts) {
	encodeControlesDeBipede(w, g.UnitControls, g.UnitControlStats)
	encodeSuppressionsDeBipede(w, g.BipedDeletions, g.BipedDeletionStats)
}

// decodeCommandeDesBipedes relit ce que [encodeCommandeDesBipedes] ecrit.
func decodeCommandeDesBipedes(r *greader, g *FilmFacts) {
	g.UnitControls, g.UnitControlStats = decodeControlesDeBipede(r)
	g.BipedDeletions, g.BipedDeletionStats = decodeSuppressionsDeBipede(r)
}

// encodeSuppressionsDeBipede ecrit les suppressions de bipede (TRIEES par instant : delta
// d horodatage), puis les denominateurs.
func encodeSuppressionsDeBipede(w *gwriter, ds []types.BipedDeletion, st types.BipedDeletionStats) {
	w.u(uint64(len(ds)))
	var last uint64
	for _, d := range ds {
		w.u(d.TimestampUS - last)
		last = d.TimestampUS
		w.u(uint64(d.Slot))
		w.u(uint64(d.Chunk))       //nolint:gosec // rang de chunk positif
		w.u(uint64(d.PacketIndex)) //nolint:gosec // rang de paquet positif
	}
	w.u(uint64(st.Deletions))         //nolint:gosec // compteurs positifs
	w.u(uint64(st.DeletionsUnproven)) //nolint:gosec // compteurs positifs
}

// decodeSuppressionsDeBipede relit ce que [encodeSuppressionsDeBipede] ecrit ; une liste vide se relit nil.
func decodeSuppressionsDeBipede(r *greader) ([]types.BipedDeletion, types.BipedDeletionStats) {
	n := r.compte(4)
	var out []types.BipedDeletion
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		d := types.BipedDeletion{TimestampUS: last}
		d.Slot = uint32(r.u())     //nolint:gosec // ecrit depuis un identifiant de 13 bits
		d.Chunk = int(r.u())       //nolint:gosec // rang ecrit positif
		d.PacketIndex = int(r.u()) //nolint:gosec // rang ecrit positif
		out = append(out, d)
	}
	st := types.BipedDeletionStats{Deletions: int(r.u()), DeletionsUnproven: int(r.u())} //nolint:gosec // compteurs ecrits positifs
	return out, st
}
