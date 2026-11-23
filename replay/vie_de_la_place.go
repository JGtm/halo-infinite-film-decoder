package replay

// vie_de_la_place.go — QUELLE VIE PORTE CE QUE LA PLACE FAIT : le pont entre une PLACE du film (l index de controle
// ou de tireur d une entree de la vue de controle) et la vie publiee ([Track]) de son occupant.
//
// Une place n est pas un joueur : son occupant change aux remplacements, et c est le roster, par intervalle de
// presence, qui dit qui la tient a une frame ([tireursParPlace]). Une vie publiee est tenue par un joueur, dit par
// le registre d identite ([occupantsDesSlots]). Une commande lue a la place P a la frame F se pose sur la vie dont
// le joueur est celui qui tient P a F, et sur aucune autre. UNE seule decision pour tous les calques qui posent
// ce que la vue de controle lit : elle ne se recopie pas (garde-rail
// `TestLaVieDeLaPlaceSeDecideEnUnSeulPoint`).

// pontDesPlaces porte ce que la decision demande : les occupants de chaque place, les joueurs de chaque slot, et
// l horloge qui date une frame.
type pontDesPlaces struct {
	occ    occupantsDesSlots
	places tireursParPlace
	clock  replayClock
}

// vieDeLaPlace dit que la vie `tr` est celle de l occupant de la place `place` a la frame `fr`. Sans occupant
// UNIQUE de la place a cette frame, l occupant est le joueur de meme index que la place ; un slot dont le joueur
// n est pas etabli a cet instant n est la vie de personne.
func (p pontDesPlaces) vieDeLaPlace(tr Track, place, fr int) bool {
	idx, _ := p.places.tireur(place, fr)
	pi, ok := p.occ.indexA(tr.Slot, int64(instantDeFrame(p.clock, fr))) //nolint:gosec // instant d un film, bien sous 2^63 µs
	return ok && pi == idx
}
