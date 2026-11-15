package replay

// filmfacts_horloges.go — L HORLOGE DE MANCHE ET LES COMPTES A REBOURS DANS LES FAITS PERSISTES.
//
// Les lectures voyagent BRUTES, dans l ordre du flux : l horloge (deux quanta de 16 bits et l etat), les
// comptes a rebours (slot, index de joueur rattache, drapeau, secondes restantes et totales) et les
// denominateurs du canal. La publication ([poserLesHorloges]) rejoue depuis elles.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// encodeHorloges ecrit les lectures des horloges (horodatages non decroissants : delta) et les compteurs.
func encodeHorloges(w *gwriter, h grammar.HorlogesLues) {
	w.u(uint64(len(h.Horloges)))
	var last uint64
	for _, l := range h.Horloges {
		w.u(l.TS - last)
		last = l.TS
		w.u(uint64(l.Limite))
		w.u(uint64(l.Reste))
		w.u(uint64(l.Etat))
	}
	w.u(uint64(len(h.Comptes)))
	last = 0
	for _, c := range h.Comptes {
		w.u(c.TS - last)
		last = c.TS
		w.u(uint64(c.Slot))
		w.i(int64(c.Joueur))
		w.bool8(c.Actif)
		w.u(uint64(c.Reste))
		w.u(uint64(c.Total))
	}
	for _, v := range compteursHorloges(&h.Stats) {
		w.u(uint64(*v)) //nolint:gosec // compteurs positifs
	}
}

// decodeHorloges relit ce que [encodeHorloges] a ecrit, dans le meme ordre.
func decodeHorloges(r *greader) grammar.HorlogesLues {
	var h grammar.HorlogesLues
	n := r.compte(4)
	h.Horloges = make([]grammar.LectureHorloge, 0, n)
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		h.Horloges = append(h.Horloges, grammar.LectureHorloge{TS: last, Limite: uint16(r.u()), //nolint:gosec // 16 bits ecrits
			Reste: uint16(r.u()), Etat: uint8(r.u())}) //nolint:gosec // 16 et 5 bits ecrits
	}
	n = r.compte(6)
	h.Comptes = make([]grammar.LectureCompteARebours, 0, n)
	last = 0
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		c := grammar.LectureCompteARebours{TS: last, Slot: uint32(r.u())} //nolint:gosec // slot ecrit
		c.Joueur = int(r.i())
		c.Actif = r.bool8()
		c.Reste, c.Total = uint16(r.u()), uint16(r.u()) //nolint:gosec // 10 bits ecrits
		h.Comptes = append(h.Comptes, c)
	}
	for _, v := range compteursHorloges(&h.Stats) {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return h
}

// compteursHorloges rend les compteurs des denominateurs, dans l ORDRE DU FORMAT : ajouter un compteur,
// c est l ajouter ICI et monter `filmFactsMagic`.
func compteursHorloges(st *grammar.StatsDesHorloges) []*int {
	return []*int{&st.Trames, &st.TramesNonLues, &st.Horloges, &st.Comptes, &st.CompteSansJoueur, &st.Creations}
}
