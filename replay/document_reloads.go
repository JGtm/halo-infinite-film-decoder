package replay

// document_reloads.go — LES RECHARGEMENTS D ARME, lus dans les messages `weapon_reload` du film.
//
// CE QUE C EST. Le film ecrit un message de rechargement dans la liste d evenements de la trame (genre 38 de
// la vue A) : l index de joueur de l auteur et l unite qui recharge, sans l arme. Le canal de la grammaire
// rend un rechargement par (trame, auteur, unite) : les copies que le jeu ecrit dans la meme trame sont
// retirees et comptees (`coverage.reloads.copies`). Ce calque pose chaque rechargement sur l axe de frames,
// avec l index de joueur du film tel que le message l ecrit : le meme espace que l index de tireur d un tir.
//
// L INSTANT ET LES COPIES. Chaque ligne porte aussi l instant de la trame a la milliseconde (`ms`) et le nombre de
// copies du message que le jeu a ecrites dans la meme trame (`copies`, absent quand il n y en a pas) : le calque
// publie le compte, jamais ce qu une copie veut dire.
//
// CE QUE CA NE DONNE PAS. Ni l arme, ni la fin du rechargement, ni la vie : un client compte des
// rechargements par joueur, rien de plus. Une trame dont la liste d evenements s arrete sur un message non lu
// perd ceux qui la suivent (`coverage.reloads.stoppedFrames`) : sur ces films le compte est un plancher.

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Reload est UN rechargement, sur l axe de frames du document.
type Reload struct {
	// T est l index de frame de la trame qui ecrit le message, sur le meme axe que Point.T.
	T int `json:"t"`
	// Place est l index de joueur de l auteur tel que le message l ecrit : le meme espace que l index de
	// tireur d un tir et que `roster[].filmIndex`.
	Place int `json:"place"`
	// Ms est l instant de la trame a la milliseconde, depuis l origine de l axe de frames (la frame 0) : la
	// meme convention que `gestures[].ms`.
	Ms int `json:"ms"`
	// Copies est le nombre de copies du message que le jeu a ecrites dans la meme trame (meme auteur, memes
	// references) et que le calque a retirees ; absent quand le message est ecrit une seule fois. Le calque
	// publie le compte, pas ce qu il veut dire. La somme sur tout le film est `coverage.reloads.copies`, aux
	// rechargements ecartes pres.
	Copies int `json:"copies,omitempty"`
}

// ReloadCoverage dit ce que le calque a lu et ce qu il a ecarte.
//
// L IDENTITE : read = copies + withoutPlayer + beforeOrigin + afterEnd + published.
type ReloadCoverage struct {
	// Frames, ClosedFrames, StoppedFrames : les trames delta du film et ce que la lecture de leur liste
	// d evenements en a fait, lue jusqu au terminateur ou arretee par un message non lu (ceux qui suivent
	// l arret sont perdus).
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	// Read : les messages de rechargement lus. Copies : ceux qui copient un message de la meme trame
	// (meme auteur, memes references) et ne sont pas des rechargements de plus.
	Read   int `json:"read"`
	Copies int `json:"copies"`
	// WithoutPlayer : rechargements dont le message ne porte pas d index de joueur — ecartes.
	WithoutPlayer int `json:"withoutPlayer"`
	// BeforeOrigin et AfterEnd : rechargements hors de l axe de frames du match — ecartes.
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// Published : les rechargements publies.
	Published int `json:"published"`
}

// buildReloads pose les rechargements lus sur l axe de frames.
func buildReloads(in []types.ReloadRead, st types.ReloadStats, axe axeDeFrames) ([]Reload, ReloadCoverage) {
	cov := ReloadCoverage{Frames: st.Frames, ClosedFrames: st.Closed, StoppedFrames: st.Stopped,
		Read: st.Messages + st.Copies, Copies: st.Copies}
	var out []Reload
	for _, m := range in {
		if m.Player == types.ReloadNoPlayer {
			cov.WithoutPlayer++
			continue
		}
		t, refus := frameDeLAction(m.TimestampUS, axe)
		switch refus {
		case refusAucun:
			out = append(out, Reload{T: t, Place: int(m.Player), Ms: axe.millisecondes(m.TimestampUS), Copies: m.Copies})
		case refusAvantOrigine, refusSansOrigine:
			cov.BeforeOrigin++
		case refusApresFin:
			cov.AfterEnd++
		}
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b Reload) int { return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Place, b.Place)) })
	return out, cov
}

// poserLesRechargements publie le calque : seulement quand la marche des trames a lu au moins une trame.
func (a *assemblage) poserLesRechargements() {
	if a.opt.ReloadStats.Frames == 0 {
		return
	}
	reloads, cov := buildReloads(a.opt.Reloads, a.opt.ReloadStats, axeDeFrames{a.origin, a.step, a.doc.FrameCount})
	a.doc.Reloads = reloads
	a.doc.Coverage.Reloads = &cov
	slog.InfoContext(a.ctx, "rejeu : rechargements", "lus", cov.Read, "copies", cov.Copies, "sansJoueur", cov.WithoutPlayer,
		"avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd, "publies", cov.Published,
		"tramesArretees", cov.StoppedFrames, "trames", cov.Frames)
}
