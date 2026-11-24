package replay

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// document_repliques_faits.go — LES REPLIQUES ET LES ANNONCES DANS LE COMPLEMENT DE LA SECTION 1 : les comptes du canal, puis
// chaque replique (horodatage en delta, drapeaux, son, unite qui parle et sa generation, indice, priorite, index de joueur) et
// chaque annonce (horodatage en delta, drapeau, son, masque, index de joueur). Une liste vide se relit nil.
//
// Ce fichier porte le codec ; l appel de [encodeRepliques] et de [decodeRepliques] dans l ordre du format est dans
// `encodeCanauxDeLaMarche` et `decodeCanauxDeLaMarche` (filmfacts_rechargements.go).

// Les drapeaux d une replique rangee dans les faits.
const (
	drapeauRepliqueSon = 1 << iota
	drapeauRepliqueLigne
	drapeauRepliqueLocuteur
)

func encodeRepliques(w *gwriter, in grammar.RepliquesLues) {
	st := in.Stats
	for _, v := range []int{st.Trames, st.Terminees, st.Arretees, st.Repliques, st.Annonces} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(in.Repliques)))
	var dernier uint64
	for _, m := range in.Repliques {
		w.u(m.TS - dernier)
		dernier = m.TS
		var drapeaux byte
		for k, v := range []bool{m.ASon, m.ALigne, m.ALocuteur} {
			if v {
				drapeaux |= 1 << k
			}
		}
		w.byte8(drapeaux)
		w.u(uint64(m.Son))
		w.u(uint64(m.Locuteur))
		w.u(uint64(m.GenLocuteur))
		w.u(uint64(m.Ligne))
		w.u(uint64(m.Priorite))
		w.i(int64(m.Joueur))
	}
	w.u(uint64(len(in.Annonces)))
	dernier = 0
	for _, m := range in.Annonces {
		w.u(m.TS - dernier)
		dernier = m.TS
		w.bool8(m.ASon)
		w.u(uint64(m.Son))
		w.u(uint64(m.Destinataires))
		w.i(int64(m.Joueur))
	}
}

func decodeRepliques(r *greader) grammar.RepliquesLues {
	var out grammar.RepliquesLues
	st := &out.Stats
	for _, v := range []*int{&st.Trames, &st.Terminees, &st.Arretees, &st.Repliques, &st.Annonces} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(8)
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		var m grammar.RepliqueLue
		dernier += r.u()
		m.TS = dernier
		drapeaux := r.byte8()
		m.ASon, m.ALigne, m.ALocuteur = drapeaux&drapeauRepliqueSon != 0, drapeaux&drapeauRepliqueLigne != 0, drapeaux&drapeauRepliqueLocuteur != 0
		m.Son = uint32(r.u())        //nolint:gosec // R(32) relu
		m.Locuteur = uint16(r.u())   //nolint:gosec // index d'unite relu
		m.GenLocuteur = uint8(r.u()) //nolint:gosec // generation sur deux bits
		m.Ligne = uint16(r.u())      //nolint:gosec // R(12) relu
		m.Priorite = uint8(r.u())    //nolint:gosec // R(5) relu
		m.Joueur = int8(r.i())       //nolint:gosec // indice R(5) ou -1
		out.Repliques = append(out.Repliques, m)
	}
	n = r.compte(5)
	dernier = 0
	for k := 0; k < n && r.err == nil; k++ {
		var m grammar.AnnonceLue
		dernier += r.u()
		m.TS = dernier
		m.ASon = r.bool8()
		m.Son = uint32(r.u())           //nolint:gosec // R(32) relu
		m.Destinataires = uint32(r.u()) //nolint:gosec // R(32) relu
		m.Joueur = int8(r.i())          //nolint:gosec // indice R(5) ou -1
		out.Annonces = append(out.Annonces, m)
	}
	return out
}
