package replay

// document_horloges.go — L HORLOGE DE MANCHE ET LES COMPTES A REBOURS DE REAPPARITION.
//
// # CE QUE LE FILM ECRIT
//
// L entite moteur reecrit l horloge de manche chaque seconde tant qu elle court : la limite de la manche, le
// temps de la minuterie et un etat de 5 bits qui dit s elle est arretee (1), descend (2 a 14) ou monte (15 a 27)
// et a quel rythme. Chaque entite joueur du moteur ecrit, chaque seconde d un compte a rebours de reapparition,
// les secondes restantes et les secondes totales de son joueur (`grammar.canalDesHorloges`).
//
// # CE QUE LE DOCUMENT PUBLIE
//
// `roundClock[]` : la serie des lectures de l horloge — instant, temps de la minuterie, limite, etat —, sur
// l axe de frames du document (une frame avant la frame 0 est negative). Une lecture consecutive identique a la
// precedente n est pas republiee. Une lecture impossible — un temps de minuterie au-dessus de sa limite dans un
// etat qui ne monte pas — est ecartee et comptee, comme celle d un etat que la mesure ne soutient pas : les etats
// 1 (arretee), 6 (descend d une seconde par seconde) et 19 (monte d une seconde par seconde) font a eux trois
// 99,9 % des lectures du parc, chacun des autres moins de 0,02 %, et ces dernieres sont des lectures de hasard.
//
// `respawnCountdowns[]` : par joueur du film (`filmIndex`), les periodes ou son compte a rebours court. Une
// periode est une suite de lectures dont les secondes restantes baissent d une par seconde, ou une lecture seule
// qui vient de commencer (la mort que la fin du match inflige n ecrit que son debut) ; `seconds` est la duree
// totale que le film ecrit. `from` et `to` sont les instants ou le compte valait `seconds` et zero :
// quand la premiere lecture de la suite est deja entamee, ou que la derniere n est pas zero (la vie suivante
// commence avant la fin affichee), l instant se deduit des lectures par la meme arithmetique, et la periode le
// dit (`startRead`, `endRead`).

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// ClockSample est une lecture de l horloge de manche.
type ClockSample struct {
	// T est la frame de la lecture ; negative pour une lecture anterieure a la frame 0.
	T int `json:"t"`
	// Left est le temps de la minuterie, en secondes : le temps restant quand elle descend, le temps ecoule
	// quand elle monte.
	Left float64 `json:"left"`
	// Limit est la limite de la manche, en secondes ; zero pour une manche sans limite de temps.
	Limit float64 `json:"limit"`
	// State est l etat de 5 bits : 1 arretee, 6 descend d une seconde par seconde, 19 monte d une seconde par
	// seconde. Les autres valeurs du jeu ne sont pas publiees.
	State int `json:"state"`
}

// RespawnCountdown est une periode de compte a rebours de reapparition d un joueur.
type RespawnCountdown struct {
	// FilmIndex est l index du joueur dans le film.
	FilmIndex int `json:"filmIndex"`
	// From et To sont les frames ou le compte valait Seconds puis zero.
	From int `json:"from"`
	To   int `json:"to"`
	// Seconds est la duree totale du compte, en secondes.
	Seconds int `json:"seconds"`
	// StartRead : la premiere lecture de la suite valait Seconds (le debut est lu) ; EndRead : la derniere
	// valait zero (la fin est lue). Faux : l instant est deduit des lectures.
	StartRead bool `json:"startRead"`
	EndRead   bool `json:"endRead"`
}

// ClocksCoverage dit ce que la pose des horloges a lu, et ce qu elle a laisse de cote.
type ClocksCoverage struct {
	// Frames : trames delta dont la liste d evenements est lue ; FramesUnread : celles dont elle n est pas
	// localisee, et que le canal ne lit donc pas.
	Frames       int `json:"frames"`
	FramesUnread int `json:"framesUnread"`
	// ClockReads : lectures de l horloge ; ClockPublished : publiees ; ClockImpossible : ecartees comme
	// impossibles ; ClockUnmeasuredState : ecartees faute d un etat mesure ; le reste est la repetition d une
	// lecture identique.
	ClockReads           int `json:"clockReads"`
	ClockPublished       int `json:"clockPublished"`
	ClockImpossible      int `json:"clockImpossible"`
	ClockUnmeasuredState int `json:"clockUnmeasuredState"`
	// CountdownReads : lectures de compte a rebours ; CountdownImpossible : ecartees comme impossibles (secondes
	// restantes au-dessus du total, ou total nul) ; CountdownNoPlayer : celles d une entite que le film ne
	// nomme pas ; CountdownIsolated : suites d une seule lecture qui n est pas un debut de compte (secondes
	// restantes differentes des secondes totales) : un compte a rebours ecrit une lecture par seconde.
	CountdownReads      int `json:"countdownReads"`
	CountdownImpossible int `json:"countdownImpossible"`
	CountdownNoPlayer   int `json:"countdownNoPlayer"`
	CountdownIsolated   int `json:"countdownIsolated"`
	// WithheldUnknownIndex, WithheldReusedIndex, WithheldOutsidePresence : periodes retenues faute d un occupant
	// unique de leur index de joueur : index absent du roster, index que plusieurs entrees portent, periode
	// posterieure a la presence de l unique entree.
	WithheldUnknownIndex    int `json:"withheldUnknownIndex"`
	WithheldReusedIndex     int `json:"withheldReusedIndex"`
	WithheldOutsidePresence int `json:"withheldOutsidePresence"`
	// Countdowns : periodes publiees ; StartDeduced, EndDeduced : celles dont le debut, la fin n est pas lu.
	Countdowns   int `json:"countdowns"`
	StartDeduced int `json:"startDeduced"`
	EndDeduced   int `json:"endDeduced"`
}

const (
	// Les etats de l horloge que la mesure soutient.
	etatHorlogeArretee  = 1
	etatHorlogeDescend  = 6
	etatHorlogeMontante = 19
	// toleranceDeCompteUS : un compte a rebours baisse d une seconde par seconde ; deux lectures qui s ecartent de
	// plus de cela de ce que leurs secondes restantes annoncent ne sont pas du meme compte.
	toleranceDeCompteUS = 1_500_000
	// repetitionDeCompteUS : deux lectures de meme valeur plus proches que cela sont la meme.
	repetitionDeCompteUS = 500_000
)

// secondesDeLHorloge rend les secondes d un quantum de l horloge de manche : la loi de dequantification du jeu.
func secondesDeLHorloge(q uint16) float64 {
	return float64(grammar.DequantEndpoint(uint64(q), 0, grammar.RoundTimerMax, 16, false, true))
}

// serieDeLHorloge replie les lectures de l horloge en serie publiee, et compte ce qu elle ecarte.
func serieDeLHorloge(reads []grammar.LectureHorloge, origin, step uint64, cov *ClocksCoverage) []ClockSample {
	var out []ClockSample
	cov.ClockReads = len(reads)
	for _, r := range reads {
		left, limit := secondesDeLHorloge(r.Reste), secondesDeLHorloge(r.Limite)
		if r.Etat != etatHorlogeArretee && r.Etat != etatHorlogeDescend && r.Etat != etatHorlogeMontante {
			cov.ClockUnmeasuredState++
			continue
		}
		if r.Reste > r.Limite && r.Etat != etatHorlogeMontante {
			cov.ClockImpossible++
			continue
		}
		s := ClockSample{T: frameOf(r.TS, origin, step), Left: left, Limit: limit, State: int(r.Etat)}
		if k := len(out) - 1; k >= 0 && out[k].Left == s.Left && out[k].Limit == s.Limit && out[k].State == s.State {
			continue
		}
		out = append(out, s)
	}
	cov.ClockPublished = len(out)
	return out
}

// periodeEnCours est la suite de lectures d un compte a rebours que le pliage suit.
type periodeEnCours struct {
	joueur   int
	premier  grammar.LectureCompteARebours
	dernier  grammar.LectureCompteARebours
	total    int
	lectures int
}

// suite dit si la lecture `r` prolonge la periode : moins de secondes restantes, a l ecart de temps que cette
// baisse annonce. Une lecture manquante entre deux lectures ne coupe pas le compte.
func (p *periodeEnCours) suite(r grammar.LectureCompteARebours) bool {
	if int(r.Total) != p.total || r.Reste >= p.dernier.Reste || r.TS < p.dernier.TS {
		return false
	}
	annonce := int64(p.dernier.Reste-r.Reste) * 1_000_000
	ecart := int64(r.TS-p.dernier.TS) - annonce
	return ecart >= -toleranceDeCompteUS && ecart <= toleranceDeCompteUS
}

// repete dit que la lecture `r` redit la derniere de la periode.
func (p *periodeEnCours) repete(r grammar.LectureCompteARebours) bool {
	return int(r.Total) == p.total && r.Reste == p.dernier.Reste && r.TS-p.dernier.TS < repetitionDeCompteUS
}

// periodesDeCompte replie les lectures des comptes a rebours en periodes par joueur du film. Un compte n est
// publie que si son index de joueur n a eu qu un occupant : [occupantUnique] retient les autres et les compte.
func periodesDeCompte(reads []grammar.LectureCompteARebours, origin, step uint64, roster []RosterEntry,
	cov *ClocksCoverage) []RespawnCountdown {
	out := plierLesComptes(lecturesDeCompteValides(reads, cov), origin, step, cov)
	out = slices.DeleteFunc(out, func(c RespawnCountdown) bool { return !occupantUnique(roster, c, cov) })
	cov.Countdowns = len(out)
	return out
}

// lecturesDeCompteValides ecarte et compte les lectures impossibles ou sans joueur, et rend les autres triees par
// joueur puis par temps.
func lecturesDeCompteValides(reads []grammar.LectureCompteARebours, cov *ClocksCoverage) []grammar.LectureCompteARebours {
	var valides []grammar.LectureCompteARebours
	cov.CountdownReads = len(reads)
	for _, r := range reads {
		switch {
		case r.Total == 0 || r.Reste > r.Total:
			cov.CountdownImpossible++
		case r.Joueur < 0:
			cov.CountdownNoPlayer++
		default:
			valides = append(valides, r)
		}
	}
	slices.SortStableFunc(valides, func(a, b grammar.LectureCompteARebours) int {
		return cmp.Or(cmp.Compare(a.Joueur, b.Joueur), cmp.Compare(a.TS, b.TS))
	})
	return valides
}

// plierLesComptes suit les lectures triees et rend les periodes closes, triees et sans doublon : une periode dont
// la lecture est seule et ne commence pas le compte est ecartee et comptee.
func plierLesComptes(valides []grammar.LectureCompteARebours, origin, step uint64, cov *ClocksCoverage) []RespawnCountdown {
	var out []RespawnCountdown
	var cur *periodeEnCours
	clore := func() {
		switch {
		case cur == nil:
		case cur.lectures < 2 && int(cur.premier.Reste) != cur.total:
			cov.CountdownIsolated++
		default:
			out = append(out, periodeDeCompte(cur, origin, step, cov))
		}
		cur = nil
	}
	for _, r := range valides {
		if cur != nil && cur.joueur == r.Joueur && cur.repete(r) {
			continue
		}
		if cur != nil && cur.joueur == r.Joueur && cur.suite(r) {
			cur.dernier = r
			cur.lectures++
			continue
		}
		clore()
		cur = &periodeEnCours{joueur: r.Joueur, premier: r, dernier: r, total: int(r.Total), lectures: 1}
	}
	clore()
	slices.SortStableFunc(out, func(a, b RespawnCountdown) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.FilmIndex, b.FilmIndex), cmp.Compare(a.To, b.To))
	})
	return slices.CompactFunc(out, func(a, b RespawnCountdown) bool {
		return a.FilmIndex == b.FilmIndex && a.From == b.From && a.To == b.To
	})
}

// occupantUnique dit si l index de joueur du compte `c` n a eu qu un occupant, et que le compte tombe pendant sa
// presence (une presence vide : l entree n est presente a aucun instant, le compte est retenu). Un arrivant qui reprend un siege lit l index de son predecesseur : le compte n est alors pas celui
// de l occupant que le roster nomme, et il est retenu et compte.
func occupantUnique(roster []RosterEntry, c RespawnCountdown, cov *ClocksCoverage) bool {
	var entree *RosterEntry
	n := 0
	for i := range roster {
		if roster[i].FilmIndex == c.FilmIndex {
			entree = &roster[i]
			n++
		}
	}
	switch {
	case n == 0:
		cov.WithheldUnknownIndex++
		return false
	case n > 1:
		cov.WithheldReusedIndex++
		return false
	}
	for _, p := range entree.Presence {
		fin := p.To
		if p.ToMax != nil {
			fin = *p.ToMax
		}
		if p.From <= c.From && c.From <= fin {
			return true
		}
	}
	cov.WithheldOutsidePresence++
	return false
}

// periodeDeCompte ferme une periode : `from` est l instant de la premiere lecture reculee de ce qu elle avait
// deja compte, `to` celui de la derniere avancee de ce qu il lui restait.
func periodeDeCompte(p *periodeEnCours, origin, step uint64, cov *ClocksCoverage) RespawnCountdown {
	const seconde = 1_000_000
	debut := int64(p.premier.TS) - int64(p.total-int(p.premier.Reste))*seconde
	fin := int64(p.dernier.TS) + int64(p.dernier.Reste)*seconde
	c := RespawnCountdown{FilmIndex: p.joueur, Seconds: p.total,
		From:      frameOf(uint64(max(debut, 0)), origin, step),
		To:        frameOf(uint64(max(fin, 0)), origin, step),
		StartRead: int(p.premier.Reste) == p.total, EndRead: p.dernier.Reste == 0}
	if !c.StartRead {
		cov.StartDeduced++
	}
	if !c.EndRead {
		cov.EndDeduced++
	}
	return c
}

// poserLesHorloges pose l horloge de manche et les comptes a rebours du film sur le document, et leur couverture.
func (a *assemblage) poserLesHorloges() {
	h := a.opt.Horloges
	cov := ClocksCoverage{Frames: h.Stats.Trames, FramesUnread: h.Stats.TramesNonLues}
	a.doc.RoundClock = serieDeLHorloge(h.Horloges, a.origin, a.step, &cov)
	a.doc.RespawnCountdowns = periodesDeCompte(h.Comptes, a.origin, a.step, a.doc.Roster, &cov)
	a.doc.Coverage.Clocks = &cov
}

// poserLesRegles pose les regles de la variante du film sur le document.
func (a *assemblage) poserLesRegles() {
	if r := rulesOf(a.opt.FilmIdentity); !r.vide() {
		a.doc.Rules = &r
	}
}
