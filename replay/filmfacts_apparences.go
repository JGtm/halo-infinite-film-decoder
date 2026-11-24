package replay

// filmfacts_apparences.go — L'APPARENCE DES SPARTANS DANS LE FICHIER DE FAITS : le bloc d'apparence de chaque humain
// (paquets de type 8) et les compteurs de sa lecture, au complement de la section 1 apres le tableau des scores
// ([encodePresenceEtEntrees]). Le rejeu se reconstruit depuis les faits : l'apparence y est, sinon elle serait perdue
// a la relecture. Une liste vide se relit nil.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// encodeApparences ecrit les apparences (triees sur le xuid), puis les compteurs de leur lecture.
func encodeApparences(w *gwriter, as []types.PlayerAppearance, st types.AppearanceStats) {
	w.u(uint64(len(as)))
	for _, a := range as {
		w.u(a.XUID)
		w.u(uint64(a.Size)) //nolint:gosec // largeur du profil, positive
		w.u(uint64(len(a.Pairs)))
		for _, p := range a.Pairs {
			w.u(uint64(p.Key))
			w.u(uint64(p.Value))
		}
		w.str(string(a.Rest))
	}
	w.bool8(st.Scanned)
	w.bool8(st.UnknownBuild)
	for _, v := range [...]int{st.Packets, st.Distinct, st.Refused, st.NoXUID, st.Pairs, st.Raw, st.Unloaded, st.Changed} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

// decodeApparences relit ce que [encodeApparences] ecrit.
func decodeApparences(r *greader) ([]types.PlayerAppearance, types.AppearanceStats) {
	n := r.compte(4)
	var out []types.PlayerAppearance
	for k := 0; k < n && r.err == nil; k++ {
		a := types.PlayerAppearance{XUID: r.u(), Size: int(r.u())} //nolint:gosec // largeur ecrite positive
		if np := r.compte(2); np > 0 {
			a.Pairs = make([]types.AppearancePair, 0, np)
			for i := 0; i < np && r.err == nil; i++ {
				a.Pairs = append(a.Pairs, types.AppearancePair{Key: uint32(r.u()), Value: uint32(r.u())}) //nolint:gosec // mots de 32 bits ecrits
			}
		}
		if rest := r.str(); rest != "" {
			a.Rest = []byte(rest)
		}
		out = append(out, a)
	}
	var st types.AppearanceStats
	st.Scanned, st.UnknownBuild = r.bool8(), r.bool8()
	for _, p := range [...]*int{&st.Packets, &st.Distinct, &st.Refused, &st.NoXUID, &st.Pairs, &st.Raw, &st.Unloaded, &st.Changed} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return out, st
}
