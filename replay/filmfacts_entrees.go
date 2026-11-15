package replay

// filmfacts_entrees.go — DE QUI EST CHAQUE LIGNE DU TABLEAU DES SCORES, DANS LE FICHIER DE FAITS (schema des faits
// 27) : les entrees des entites du tableau aux images-cles, le masque des manches decidees et leurs denominateurs, au complement de la section 1
// ([encodeComplementDesEntrees]), avant le verdict du fil des morts.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// encodeEntreesDuStatborg ecrit les entrees (dans l'ordre du flux : delta d'horodatage), puis les denominateurs.
func encodeEntreesDuStatborg(w *gwriter, es []types.StatborgEntry, st types.StatborgEntryStats) {
	w.u(uint64(len(es)))
	var last uint64
	for _, e := range es {
		w.i(int64(e.TS) - int64(last)) //nolint:gosec // horodatages moteur, bien en deca de 2^63
		last = e.TS
		w.u(uint64(e.Slot))
		w.u(uint64(e.Kind))
		w.u(uint64(e.ID))
	}
	w.bool8(st.Scanned)
	for _, v := range [...]int{st.Records, st.Read, st.Vacant, st.Unreached} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(st.DecidedRounds))
}

// decodeEntreesDuStatborg relit ce que [encodeEntreesDuStatborg] ecrit ; une liste vide se relit nil.
func decodeEntreesDuStatborg(r *greader) ([]types.StatborgEntry, types.StatborgEntryStats) {
	n := r.compte(4)
	var out []types.StatborgEntry
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last = uint64(int64(last) + r.i())                                                                   //nolint:gosec // horodatages moteur
		out = append(out, types.StatborgEntry{TS: last, Slot: int(r.u()), Kind: int(r.u()), ID: int(r.u())}) //nolint:gosec // ecrits positifs
	}
	var st types.StatborgEntryStats
	st.Scanned = r.bool8()
	for _, p := range [...]*int{&st.Records, &st.Read, &st.Vacant, &st.Unreached} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	st.DecidedRounds = uint32(r.u()) //nolint:gosec // masque de trente-deux manches
	return out, st
}
