package replay

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// filmfacts_jeu.go — LES MESSAGES DE JEU ET LES SCRIPTS DU FILM DANS LE COMPLEMENT DE LA SECTION 1 : le
// témoin de lecture, les noms de script, les comptes du canal, puis les messages (horodatage, rang,
// séquence, mot texte, masque des destinataires, deux participants). Une liste vide se relit nil.

func encodeJeu(w *gwriter, g *FilmFacts) {
	w.bool8(g.GameEventsRead)
	w.u(uint64(len(g.Scripts)))
	for _, s := range g.Scripts {
		w.str(s)
	}
	st := g.GameEventStats
	for _, v := range []int{st.Trames, st.Terminees, st.Arretees, st.Messages, st.Copies} {
		w.u(uint64(v))
	}
	w.u(uint64(len(g.GameEvents)))
	for _, e := range g.GameEvents {
		w.u(e.TS)
		w.u(uint64(e.Rang))
		w.byte8(e.Sequence)
		w.bool8(e.ATexte)
		w.u(uint64(e.Texte))
		w.u(uint64(e.Destinataires))
		for _, p := range e.Participants {
			w.i(int64(p))
		}
	}
}

func decodeJeu(r *greader, g *FilmFacts) {
	g.GameEventsRead = r.bool8()
	g.Scripts = nil
	for range r.compte(1) {
		g.Scripts = append(g.Scripts, r.str())
	}
	st := &g.GameEventStats
	for _, v := range []*int{&st.Trames, &st.Terminees, &st.Arretees, &st.Messages, &st.Copies} {
		*v = int(r.u())
	}
	g.GameEvents = nil
	n := r.compte(8)
	for k := 0; k < n && r.err == nil; k++ {
		var e grammar.EvenementDeJeuLu
		e.TS = r.u()
		e.Rang = uint32(r.u()) //nolint:gosec // R(32) relu
		e.Sequence = r.byte8()
		e.ATexte = r.bool8()
		e.Texte = uint32(r.u())         //nolint:gosec // R(32) relu
		e.Destinataires = uint32(r.u()) //nolint:gosec // R(32) relu
		for i := range e.Participants {
			e.Participants[i] = int8(r.i()) //nolint:gosec // indice R(5) ou -1
		}
		g.GameEvents = append(g.GameEvents, e)
	}
}
