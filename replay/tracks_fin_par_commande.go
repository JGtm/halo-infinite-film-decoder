package replay

// tracks_fin_par_commande.go — COMMENT LE CORPS D UNE PISTE A FINI, LU DANS LE JOUEUR QUI LE COMMANDE.
//
// Le film ecrit, pour chaque corps, qui le commande (`unit-control`, [types.UnitControlRead]) et le
// moment ou le jeu le supprime ([types.BipedDeletion]). Un joueur mort lache la commande de son corps ;
// une manche qui se termine ou une remise en place supprime le corps encore commande. Ces deux lectures
// disent comment une vie a fini ; elles ne disent pas si cette fin est une mort, et ce fichier ne le juge
// pas : il publie la fin lue ([Track.EndReason]) et la frame du lacher ([Track.ReleaseFrame]) telles que
// le film les ecrit, sans les comparer a la fin de la piste.
//
// Methode reprise de libfilmshell (Dend, OpenSpartan, licence MIT, `src/owners.rs`, `src/motion.rs` et `src/lives.rs`) :
// une vie finit par une mort quand son joueur lache la commande du corps, par la suppression du corps
// quand une manche la termine.
//
// LES CORPS D UN SLOT. Les lectures et les suppressions d un slot, dans l ordre des instants, se rangent
// en corps. Un corps s ouvre a la premiere lecture qui met un joueur aux commandes ; il se ferme au
// premier des trois evenements suivants :
//
//	lecture sans joueur     le joueur lache la commande : [EndReasonReleased], a la frame de la lecture ;
//	suppression du slot     le corps est supprime encore commande : [EndReasonDeleted] ;
//	lecture d un autre      le slot est repris par un autre joueur sans que la suppression se lise :
//	joueur                  [EndReasonDeleted] aussi, et compte a part ([LifeEndCoverage.Retaken]) ; la
//	                        lecture ouvre le corps suivant.
//
// Une lecture sans joueur avant toute lecture commandee ne dit rien du corps : le corps n est pas ouvert.
// A un meme instant, la suppression precede les lectures : un slot supprime puis recree dans le meme paquet
// ouvre son corps neuf.
//
// UNE PISTE EST RAPPORTEE AU CORPS QUI LA PORTE : le dernier corps ouvert a la frame de son premier point,
// s il n est pas deja ferme a cette frame ou avant (une fin a la frame meme du premier point est celle de la piste qui precede). Son premier enregistrement `unit-control` borne ainsi la
// naissance du corps, sans fenetre de temps. La fin du corps ne compte pour la piste que si elle tombe au plus tard
// a l ouverture de la piste suivante du meme slot : un corps que deux pistes se partagent ne ferme que la
// derniere.
//
// UNE FIN A L OUVERTURE DE LA PISTE SUIVANTE EST CELLE DE LA PISTE QUI LA PRECEDE : un corps repris, lache ou supprime a la frame meme ou le slot
// ouvre sa piste suivante ferme la premiere. UNE FIN HORS DE L AXE N EST PAS UNE FIN DE VIE : un corps qui finit a la frame de fin du film ou
// au-dela (le film cesse d ecrire, il lache ses vies) ne publie ni `endReason` ni `releaseFrame` ; il se compte ([LifeEndCoverage.FilmEnd]).

import (
	"cmp"
	"math"
	"slices"
)

// Les fins de vie que le film ecrit ([Track.EndReason]).
const (
	// EndReasonReleased : le joueur de la vie lache la commande de son corps.
	EndReasonReleased = "released"
	// EndReasonDeleted : le corps est supprime encore commande (fin de manche, remise en place, depart), ou
	// le slot est repris par un autre joueur.
	EndReasonDeleted = "deleted"
)

// LifeEndCoverage dit comment les pistes publiees finissent, lu dans le film. Les six premiers comptes
// somment le nombre de pistes publiees.
type LifeEndCoverage struct {
	// Released : les pistes dont le joueur lache la commande du corps.
	Released int `json:"released"`
	// Deleted : les pistes dont le corps est supprime encore commande, la suppression lue.
	Deleted int `json:"deleted"`
	// Retaken : les pistes dont le slot est repris par un autre joueur, la suppression non lue ; elles
	// portent aussi `deleted`.
	Retaken int `json:"retaken"`
	// Unread : les pistes dont aucune lecture ne met un joueur aux commandes du corps.
	Unread int `json:"unread"`
	// Open : les pistes dont un joueur commande le corps sans qu aucun evenement de fin ne se lise avant la
	// piste suivante du slot (survivant).
	Open int `json:"open"`
	// FilmEnd : les pistes dont le corps finit a la frame de fin du film ou au-dela, hors de l axe du document : le film cesse
	// d ecrire, ce n est pas une fin de vie, et aucun champ n est publie pour elles.
	FilmEnd int `json:"filmEnd"`
	// DeletionsUnproven : les suppressions d un bipede lues sur un debut de liste non prouve, que le jeu ne
	// prend pas et que le document ne publie pas.
	DeletionsUnproven int `json:"deletionsUnproven"`
}

// finLue est la fin d une piste telle que le film l ecrit ; la raison vide dit qu aucune lecture ne la dit.
type finLue struct {
	raison string
	// relache est la frame du lacher de commande, pour [EndReasonReleased] seulement.
	relache int
}

// finNature distingue les trois evenements qui ferment un corps ; [finAucune] : le corps est ouvert.
type finNature uint8

const (
	finAucune finNature = iota
	finRelache
	finSupprime
	finRepris
)

// corpsCommande est un corps d un slot, de la lecture qui l ouvre a l evenement qui le ferme.
type corpsCommande struct {
	// ouvert est la frame de la premiere lecture commandee, joueur l index lu.
	ouvert, joueur int
	// fin est la frame de l evenement qui ferme le corps, nature ce qu il est.
	fin    int
	nature finNature
}

// evenementDeCommande est une lecture de commande ou une suppression d un slot.
type evenementDeCommande struct {
	us          uint64
	frame       int
	suppression bool
	// joueur est l index lu, -1 sans joueur et pour une suppression.
	joueur int
}

// finsParCommande rapporte chaque piste de `tracks` au corps qui la porte et rend sa fin lue, dans l ordre des
// pistes, avec les comptes de la couverture. `origin` et `step` sont ceux de la grille du document.
func finsParCommande(tracks []Track, p LecturesDePresence, origin, step uint64, frames int) ([]finLue, LifeEndCoverage) {
	corps := corpsDeCommandeParSlot(p, origin, step)
	cov := LifeEndCoverage{DeletionsUnproven: p.DeletionStats.DeletionsUnproven}
	fins := make([]finLue, len(tracks))
	parSlot := map[uint32][]int{}
	for i, t := range tracks {
		parSlot[t.Slot] = append(parSlot[t.Slot], i)
	}
	for slot, pistes := range parSlot {
		slices.SortStableFunc(pistes, func(a, b int) int { return cmp.Compare(tracks[a].StartFrame, tracks[b].StartFrame) })
		for k, i := range pistes {
			suivante := math.MaxInt
			if k+1 < len(pistes) {
				suivante = tracks[pistes[k+1]].StartFrame
			}
			fins[i] = finDeLaPiste(corps[slot], tracks[i].StartFrame, suivante, frames, &cov)
		}
	}
	return fins, cov
}

// finDeLaPiste rend la fin de la piste qui s ouvre a la frame `debut` et que la piste suivante du slot
// remplace a la frame `suivante`, et la compte.
func finDeLaPiste(corps []corpsCommande, debut, suivante, frames int, cov *LifeEndCoverage) finLue {
	var c *corpsCommande
	for i := range corps {
		if corps[i].ouvert > debut {
			break
		}
		c = &corps[i]
	}
	if c == nil || (c.nature != finAucune && c.fin <= debut) {
		cov.Unread++
		return finLue{}
	}
	if c.nature == finAucune || c.fin > suivante {
		cov.Open++
		return finLue{}
	}
	if c.fin >= frames {
		cov.FilmEnd++
		return finLue{}
	}
	switch c.nature {
	case finRelache:
		cov.Released++
		return finLue{raison: EndReasonReleased, relache: c.fin}
	case finRepris:
		cov.Retaken++
	default:
		cov.Deleted++
	}
	return finLue{raison: EndReasonDeleted}
}

// corpsDeCommandeParSlot range les lectures de commande et les suppressions en corps, par slot.
func corpsDeCommandeParSlot(p LecturesDePresence, origin, step uint64) map[uint32][]corpsCommande {
	evts := map[uint32][]evenementDeCommande{}
	for _, c := range p.Controls {
		evts[c.Slot] = append(evts[c.Slot], evenementDeCommande{us: c.TimestampUS,
			frame: frameOf(c.TimestampUS, origin, step), joueur: c.Controller})
	}
	for _, d := range p.Deletions {
		evts[d.Slot] = append(evts[d.Slot], evenementDeCommande{us: d.TimestampUS,
			frame: frameOf(d.TimestampUS, origin, step), suppression: true, joueur: -1})
	}
	out := make(map[uint32][]corpsCommande, len(evts))
	for slot, es := range evts {
		slices.SortStableFunc(es, comparerLesEvenements)
		out[slot] = ouvrirLesCorps(es)
	}
	return out
}

// comparerLesEvenements ordonne les evenements d un slot sur l instant, la suppression avant les lectures.
func comparerLesEvenements(a, b evenementDeCommande) int {
	if c := cmp.Compare(a.us, b.us); c != 0 {
		return c
	}
	switch {
	case a.suppression && !b.suppression:
		return -1
	case !a.suppression && b.suppression:
		return 1
	}
	return cmp.Compare(a.joueur, b.joueur)
}

// ouvrirLesCorps deroule les evenements d un slot, dans l ordre, et rend ses corps.
func ouvrirLesCorps(es []evenementDeCommande) []corpsCommande {
	var corps []corpsCommande
	ouvert := false
	fermer := func(e evenementDeCommande, nature finNature) {
		c := &corps[len(corps)-1]
		c.fin, c.nature = e.frame, nature
		ouvert = false
	}
	for _, e := range es {
		switch {
		case e.suppression:
			if ouvert {
				fermer(e, finSupprime)
			}
		case e.joueur < 0:
			if ouvert {
				fermer(e, finRelache)
			}
		default:
			if ouvert && corps[len(corps)-1].joueur != e.joueur {
				fermer(e, finRepris)
			}
			if !ouvert {
				corps = append(corps, corpsCommande{ouvert: e.frame, joueur: e.joueur})
				ouvert = true
			}
		}
	}
	return corps
}

// poserLesFinsParCommande pose sur chaque piste la fin que le film ecrit de son corps, et la couverture. Une
// lecture de commande qui n a pas eu lieu ne dit rien : aucun champ n est pose et la couverture reste absente.
func (a *assemblage) poserLesFinsParCommande() {
	p := a.opt.Presences
	if !p.ControlStats.Scanned {
		return
	}
	fins, cov := finsParCommande(a.doc.Tracks, p, a.origin, a.step, a.doc.FrameCount)
	for i, f := range fins {
		a.doc.Tracks[i].EndReason = f.raison
		if f.raison == EndReasonReleased {
			frame := f.relache
			a.doc.Tracks[i].ReleaseFrame = &frame
		}
	}
	a.trackCov.LifeEnds = &cov
}
