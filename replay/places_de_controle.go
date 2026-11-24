package replay

// places_de_controle.go — LA PLACE DE CONTROLE DE CHAQUE JOUEUR, LUE DANS SES TIRS.
//
// # LE FAIT
//
// L index de tireur d un record de tir (`grammar.FireEvent.FilmIndex`) et l index de controle d une
// rafale (`types.ContinuousFireBurst.PlayerIndex`) sont le MEME espace : la PLACE de controle du
// joueur, que le jeu reattribue quand un joueur part et qu un autre arrive. Elle n est PAS l index
// du roster : un arrivant en cours de match joue a la place d un partant pendant que le roster le
// numerote autrement.
//
// Un record de tir ecrit, dans le MEME record, la place du tireur et l unite qui tire (reference 0).
// Quand cette unite est un bipede dont le registre connait l occupant a l instant du tir, le record
// LIE cette place a ce joueur, a cette frame ([liaisonDePlace]). C est la lecture : elle ne suppose
// rien, elle ne se trompe que si le registre se trompe d occupant.
//
// # LES REGLES, DANS L ORDRE OU ELLES S APPLIQUENT
//
//	P1 LIAISON        une liaison est un fait : a sa frame, ce joueur tient cette place. Deux joueurs
//	                  lies a la meme place a la meme frame se contredisent : cette frame n est a
//	                  personne, et les deux liaisons sont comptees (`contested`).
//	P2 TENUE LUE      entre deux liaisons CONSECUTIVES d une place qui nomment le meme joueur, ce
//	                  joueur tient la place : si un autre y avait ete lu entre les deux, elles ne
//	                  seraient pas consecutives. Un joueur qui revient apres un autre sur la meme
//	                  place, ou qui change de place, ouvre une nouvelle tenue.
//	P3 PRESENCE       un joueur dont TOUTES les liaisons designent la meme place la tient pendant
//	                  toute sa presence ([viesParJoueur]) : il n a qu une place a tenir, et le jeu
//	                  n en change pas en cours de presence. Un joueur lu a plusieurs places n est
//	                  pas etendu : rien ne dit laquelle il tenait hors de ses liaisons.
//	P4 RELAIS         entre la tenue d un joueur et la tenue suivante d un autre sur la meme place, la
//	                  part de l ecart que les presences des deux se disputent n est tenue par
//	                  PERSONNE : le film ne dit pas lequel des deux tient la place a cet instant.
//
// Une frame que les tenues ne couvrent pas n est pas lue : le siege du roster la decide, et
// c est le repli nomme `repli_tireur_par_le_siege_de_la_place` ([tireursDesPlaces]).
//
// Sur un film ou l index de tireur n est pas la place ([accordIndexUnite.estLaPlace]), rien n est lu.
//
// PUR : aucune I/O, aucune goroutine ; aucune sortie ne depend de l ordre de parcours d une map.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// ControlPlaceCoverage dit ce que la lecture des places de controle a lu et ce qu elle a decide
// (`coverage.controlPlaces`). Les trois comptes de tirs et les trois comptes de rafales
// partitionnent les tirs et les rafales dont le tireur a ete decide.
type ControlPlaceCoverage struct {
	// Read : la lecture a eu lieu (l index de tireur est la place sur ce film).
	Read bool `json:"read"`
	// Bindings : liaisons DISTINCTES (place, frame, joueur) entre une place et un joueur, hors celles
	// que P1 conteste. Deux tirs du meme joueur, a la meme place, dans la meme frame, font UNE liaison :
	// ce n est pas un compte de tirs (`shotsByPlace` et ses deux voisins comptent les tirs). Players : joueurs
	// lies ; MultiPlace : joueurs lus a plusieurs places ; Handovers : relais lus (deux tenues
	// consecutives de joueurs differents sur une place) ; Returns : relais ou un joueur reprend une
	// place qu un autre a tenue entre-temps ; Contested : liaisons que P1 conteste.
	Bindings   int `json:"bindings"`
	Players    int `json:"players"`
	MultiPlace int `json:"multiPlace"`
	Handovers  int `json:"handovers"`
	Returns    int `json:"returns"`
	Contested  int `json:"contested"`
	// ShotsByPlace / BurstsByPlace : tirs et rafales dont le tireur est celui que la place lue
	// designe. ShotsBySeat / BurstsBySeat : ceux que la lecture ne tient a personne et que le siege du
	// roster rend (repli `repli_tireur_par_le_siege_de_la_place`). ShotsUnheld / BurstsUnheld : ceux
	// que ni l une ni l autre ne tient : l index du record reste, comme avant la lecture des places.
	ShotsByPlace  int `json:"shotsByPlace"`
	ShotsBySeat   int `json:"shotsBySeat"`
	ShotsUnheld   int `json:"shotsUnheld"`
	BurstsByPlace int `json:"burstsByPlace"`
	BurstsBySeat  int `json:"burstsBySeat"`
	BurstsUnheld  int `json:"burstsUnheld"`
}

// personne est le joueur d une liaison contestee (P1) : il ne tient rien, et il separe les tenues qu il coupe.
const personne = -1

// liaisonDePlace : un record de tir lie `place` a `joueur` (index de joueur du registre), a `frame`.
type liaisonDePlace struct {
	place, joueur, frame int
}

// tenueDePlace : `joueur` tient la place sur les frames [de, a], bornes incluses.
type tenueDePlace struct {
	joueur, de, a int
}

// placesDeControle repond « quel joueur tient cette place a cette frame ».
type placesDeControle struct {
	// parPlace : les tenues de chaque place, triees par frame et disjointes.
	parPlace map[int][]tenueDePlace
	cov      ControlPlaceCoverage
}

// lirePlacesDeControle lit les places de controle des tirs du film. `vies` donne, par index de
// joueur, la premiere et la derniere frame de sa presence ([viesParJoueur]). `actif` faux : rien
// n est lu (l index de tireur n est pas la place sur ce film).
func lirePlacesDeControle(fire []grammar.FireEvent, occ occupantsDesSlots, horloge replayClock,
	vies map[int][2]int, actif bool) placesDeControle {
	out := placesDeControle{parPlace: map[int][]tenueDePlace{}, cov: ControlPlaceCoverage{Read: actif}}
	if !actif || occ.vide() {
		return out
	}
	liaisons, contestees := liaisonsDesTirs(fire, occ, horloge)
	out.cov.Contested = contestees
	placesDe := map[int]map[int]bool{} // joueur -> places lues
	parPlace := map[int][]liaisonDePlace{}
	for _, l := range liaisons {
		parPlace[l.place] = append(parPlace[l.place], l)
		if l.joueur == personne {
			continue
		}
		out.cov.Bindings++
		if placesDe[l.joueur] == nil {
			placesDe[l.joueur] = map[int]bool{}
		}
		placesDe[l.joueur][l.place] = true
	}
	out.cov.Players = len(placesDe)
	for _, ps := range placesDe {
		if len(ps) > 1 {
			out.cov.MultiPlace++
		}
	}
	fidele := func(joueur int) bool { return len(placesDe[joueur]) == 1 }
	for place, ls := range parPlace {
		ts := tenuesLues(ls)
		relais, retours := relaisDeLaPlace(ts)
		out.cov.Handovers += relais
		out.cov.Returns += retours
		out.parPlace[place] = etendreAuxPresences(ts, vies, fidele)
	}
	return out
}

// liaisonsDesTirs rend les liaisons (P1) triees par place puis frame, et le nombre de liaisons
// contestees : a une frame ou deux joueurs se partagent une place, une seule liaison au joueur
// `personne` les remplace.
func liaisonsDesTirs(fire []grammar.FireEvent, occ occupantsDesSlots, horloge replayClock) ([]liaisonDePlace, int) {
	var tous []liaisonDePlace
	for _, e := range fire {
		if !e.HasShooter || !e.Unit.Present || e.FilmIndex < 0 {
			continue
		}
		pi, ok := occ.indexA(e.Unit.Slot, int64(e.TimestampUS)) //nolint:gosec // horloge du film
		if !ok {
			continue
		}
		tous = append(tous, liaisonDePlace{place: e.FilmIndex, joueur: pi, frame: horloge.frame(e.TimestampUS)})
	}
	slices.SortFunc(tous, func(a, b liaisonDePlace) int {
		return cmp.Or(cmp.Compare(a.place, b.place), cmp.Compare(a.frame, b.frame), cmp.Compare(a.joueur, b.joueur))
	})
	tous = slices.Compact(tous)
	out := tous[:0:0]
	contestees := 0
	for i := 0; i < len(tous); {
		j := i
		for j < len(tous) && tous[j].place == tous[i].place && tous[j].frame == tous[i].frame {
			j++
		}
		if j-i > 1 { // deux joueurs a la meme place a la meme frame : P1
			contestees += j - i
			tous[i].joueur = personne
		}
		out = append(out, tous[i])
		i = j
	}
	return out, contestees
}

// tenuesLues rend les tenues d UNE place (P2) : chaque suite de liaisons consecutives du meme
// joueur, de la premiere a la derniere. `ls` est triee par frame.
func tenuesLues(ls []liaisonDePlace) []tenueDePlace {
	var out []tenueDePlace
	for _, l := range ls {
		if n := len(out); n > 0 && out[n-1].joueur == l.joueur {
			out[n-1].a = l.frame
			continue
		}
		out = append(out, tenueDePlace{joueur: l.joueur, de: l.frame, a: l.frame})
	}
	return out
}

// relaisDeLaPlace compte les relais d une place (le tenant change) et, parmi eux, les retours (le
// nouveau tenant l avait deja tenue). Une frame contestee n est pas un tenant.
func relaisDeLaPlace(ts []tenueDePlace) (relais, retours int) {
	vus := map[int]bool{}
	dernier := personne
	for _, t := range ts {
		if t.joueur == personne || t.joueur == dernier {
			continue
		}
		if dernier != personne {
			relais++
			if vus[t.joueur] {
				retours++
			}
		}
		vus[t.joueur] = true
		dernier = t.joueur
	}
	return relais, retours
}

// etendreAuxPresences applique P3 puis P4 aux tenues triees d une place : chaque tenue d un joueur
// `fidele` s etend a sa presence ([viesParJoueur]), sans passer la tenue voisine, et la part d un
// ecart que deux presences se disputent n est a personne. Les tenues du joueur `personne` (frame
// contestee) bornent leurs voisines puis disparaissent.
func etendreAuxPresences(ts []tenueDePlace, vies map[int][2]int, fidele func(joueur int) bool) []tenueDePlace {
	out := slices.Clone(ts)
	// Les bornes que la presence PERMET : le coeur seul pour un joueur lu a plusieurs places ou sans presence connue.
	debuts, fins := make([]int, len(ts)), make([]int, len(ts))
	for i, t := range ts {
		debuts[i], fins[i] = t.de, t.a
		if v, ok := vies[t.joueur]; ok && t.joueur != personne && fidele(t.joueur) {
			debuts[i], fins[i] = min(t.de, v[0]), max(t.a, v[1])
		}
	}
	if len(out) > 0 {
		out[0].de, out[len(out)-1].a = debuts[0], fins[len(out)-1]
	}
	for i := 0; i+1 < len(ts); i++ {
		// L ECART ENTRE DEUX TENUES : chacune s y etend jusqu a la tenue de l autre ; ou les deux
		// etendues se recouvrent, la place n est a personne (P4).
		finAvant := min(fins[i], ts[i+1].de-1)
		debutApres := max(debuts[i+1], ts[i].a+1)
		if finAvant >= debutApres {
			finAvant, debutApres = debutApres-1, finAvant+1
		}
		out[i].a, out[i+1].de = finAvant, debutApres
	}
	return slices.DeleteFunc(out, func(t tenueDePlace) bool { return t.joueur == personne })
}

// joueurA rend le joueur (index de roster) qui tient la place `place` a la frame `fr`.
func (p placesDeControle) joueurA(place, fr int) (int, bool) {
	for _, t := range p.parPlace[place] {
		if fr >= t.de && fr <= t.a {
			return t.joueur, true
		}
	}
	return 0, false
}

// viesParJoueur rend, par index de joueur, la premiere et la derniere frame ou il est present : ses
// intervalles de presence au roster (`RosterEntry.Presence`, jusqu a `toMax`, la borne d affichage) et
// ses vies publiees (une vie appartient au joueur que le registre lit sur son slot a son debut, a son
// milieu ou a sa fin). C est la presence que P3 donne en etendue a un joueur qui n a qu une place.
func viesParJoueur(tracks []Track, roster []RosterEntry, occ occupantsDesSlots, horloge replayClock) map[int][2]int {
	out := map[int][2]int{}
	etendre := func(pi, de, a int) {
		if v, vu := out[pi]; vu {
			de, a = min(v[0], de), max(v[1], a)
		}
		out[pi] = [2]int{de, a}
	}
	for _, r := range roster {
		for _, iv := range r.Presence {
			fin := iv.To
			if iv.ToMax != nil {
				fin = max(fin, *iv.ToMax)
			}
			etendre(r.FilmIndex, iv.From, max(fin, iv.From))
		}
	}
	for _, tr := range tracks {
		for _, fr := range [...]int{tr.StartFrame, (tr.StartFrame + tr.EndFrame) / 2, tr.EndFrame} {
			if pi, ok := occ.indexA(tr.Slot, int64(horloge.origin+uint64(fr)*horloge.step)); ok { //nolint:gosec // horloge du film
				etendre(pi, tr.StartFrame, tr.EndFrame)
			}
		}
	}
	return out
}
