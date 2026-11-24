package replay

// filmfacts_vitalites.go : LES CHANGEMENTS DE SANTE ET DE BOUCLIER DANS LE COMPLEMENT DE LA SECTION 1 : les comptes
// du canal, puis chaque lecture (horodatage en delta, slot, generation et drapeau de naissance dans un seul entier,
// sante et bouclier en flottants). Une liste vide se relit nil.

// coutMinimalVitalite : horodatage, slot et generation, un octet chacun au moins, puis deux flottants de quatre octets.
const coutMinimalVitalite = 3 + 8

func encodeVitalites(w *gwriter, rs []VitalRead, st VitalStats) {
	w.bool8(st.Scanned)
	for _, v := range []int{st.Read, st.Births, st.Unchanged} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(rs)))
	var dernier uint64
	for _, m := range rs {
		w.u(m.TimestampUS - dernier)
		dernier = m.TimestampUS
		w.u(uint64(m.Slot))
		w.u(uint64(m.Gen)<<1 | uint64(unSiVrai(m.Birth)))
		w.f32(m.Health)
		w.f32(m.Shield)
	}
}

func decodeVitalites(r *greader) ([]VitalRead, VitalStats) {
	var st VitalStats
	st.Scanned = r.bool8()
	for _, v := range []*int{&st.Read, &st.Births, &st.Unchanged} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(coutMinimalVitalite)
	var out []VitalRead
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		m := VitalRead{TimestampUS: dernier, Slot: uint32(r.u())} //nolint:gosec // slot ecrit tel quel
		g := r.u()
		m.Gen, m.Birth = uint8(g>>1), g&1 == 1 //nolint:gosec // generation ecrite sur un octet
		m.Health, m.Shield = r.f32(), r.f32()
		out = append(out, m)
	}
	return out, st
}
