package replay

// film_scan_images_cles.go — LA MARCHE UNIQUE DES IMAGES-CLES POUR LES BIPEDES.
//
// Armes portees, inventaire et marque de portage sortent d UNE distribution de la phase des
// images-cles (`grammar.ScanEtatsDesImagesCles`), lus par la grammaire sur les records admis. Elle
// est jouee une fois par cuisson, au rang des armes portees ([filmScan.balayerPositions]), en deux
// temps : [filmScan.lireLesImagesCles] lit, [filmScan.publierLesImagesCles] pose et journalise ;
// l inventaire ([filmScan.balayerInventaire]) et la marque ([filmScan.balayerCalquesGardes]) la
// reprennent sans remarcher le film.
//
// LES RECORDS RECUPERES (D1.3, 2026-10-09). Un record non admis dont une fenetre de bits a rendu une
// valeur est marque recupere par la grammaire (`grammar.RecordRecupere`, ADR 0037 IR-6), avec ses
// methodes. Sa valeur entre dans les memes listes que les lectures de la grammaire, dans l ordre
// du film : un client n a qu une grandeur par champ. Ce que la cuisson PUBLIE de la recuperation,
// ce sont les comptes des trois replis (`coverage.fallbacks`, verses avec le rapport du contexte et
// persistes avec lui dans les faits) ; la marque par record ne voyage pas (decision E-8 du plan :
// la forme des faits ne s elargit pas), elle se JOURNALISE ici, par methode.

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// lectureImagesCles est ce que la marche unique des images-cles a rendu : ce que la grammaire dit des
// bipedes, et l erreur de la marche.
type lectureImagesCles struct {
	etats grammar.EtatsDesImagesCles
	err   error
}

// lireLesImagesCles joue la marche unique des images-cles. Elle ne fait que lire : ni journal, ni repli.
// CE QU ELLE TOUCHE DU CONTEXTE : le memo des marches d images-cles, les ancres et les lectures bipedes
// que la marche des trames a rendus chauds, et le canal de l etat complet du bipede, dont les parcours
// de corps posent les largeurs MPP du format le temps de leur boucle puis les restituent.
func (s *filmScan) lireLesImagesCles() lectureImagesCles {
	etats, err := grammar.ScanEtatsDesImagesCles(s.fc, loadoutFamilies(), 0)
	return lectureImagesCles{etats: etats, err: err}
}

// publierLesImagesCles pose ce que la marche a rendu sur le balayage (les phases suivantes le lisent :
// inventaire, marque de portage) et journalise ce que la regle d admission a juge et ce que les
// fenetres ont recupere.
func (s *filmScan) publierLesImagesCles(lu lectureImagesCles) {
	s.etats, s.errEtats = lu.etats, lu.err
	a := s.etats.Admission
	r := recuperesParMethode(s.etats.Recuperes)
	slog.InfoContext(s.ctx, "image-cle : etat complet des bipedes lu par la grammaire",
		"match_id", s.matchID, "bipedes", a.Bipedes, "admis", a.Admis, "sansCorps", a.SansCorps,
		"refusNiFermeNiI22", a.RefusNiFermeNiI22, "refusT1", a.RefusT1, "refusT2", a.RefusT2,
		"refusDebordement", a.RefusDebordement,
		"debordements", a.Debordements, "capaciteHorsDomaine", a.CapaciteHorsDomaine,
		"selectionHorsMasque", a.SelectionHorsMasque,
		"recuperes", len(s.etats.Recuperes), "recuperesArmes", r.armes,
		"recuperesInventaire", r.inventaire, "recuperesMarque", r.marque)
}

// publierLesLoadouts publie les armes portees de la marche unique : leur erreur, la couverture de la
// marche, ses replis et l etape `loadouts`. Rien n est lu ici.
//
// LA MARCHE D'IMAGE-CLÉ SE COMPTE ICI (lot M3.1) : l'élection de l'ancre suivante est un REPLI nommé,
// et son compte voyage dans `coverage.fallbacks`.
func (s *filmScan) publierLesLoadouts() {
	loadouts, marche, err := s.etats.Loadouts, s.etats.Marche, s.errEtats
	if err != nil {
		slog.WarnContext(s.ctx, "keyframes illisibles — rejeu sans armes portées", "err", err, "match_id", s.matchID)
		loadouts = nil
	}
	s.in.Loadouts, s.in.KeyframeWalk = loadouts, marche
	s.opt.Fallbacks.DeclencheN(fallback.NomAncreDImageCleParElection, marche.Elections)
	if marche.BipedesAbsentsEncadres > 0 {
		slog.InfoContext(s.ctx, "image-cle : bipedes manques par la marche entre deux images-cles qui les portent",
			"match_id", s.matchID, "absentsEncadres", marche.BipedesAbsentsEncadres,
			"elections", marche.Elections, "recalages", marche.Recalages)
	}
	s.opt.observe(s.ctx, "loadouts", s.in.Loadouts)
}

// comptesDesRecuperes compte les records recuperes par methode : un record compte dans chaque
// fenetre qui lui a rendu une valeur.
type comptesDesRecuperes struct{ armes, inventaire, marque int }

// recuperesParMethode compte les records recuperes par les fenetres, par methode.
func recuperesParMethode(recs []grammar.RecordRecupere) comptesDesRecuperes {
	var c comptesDesRecuperes
	for _, r := range recs {
		c.armes += unSiVrai(r.Armes)
		c.inventaire += unSiVrai(r.Inventaire)
		c.marque += unSiVrai(r.Marque)
	}
	return c
}

// unSiVrai rend 1 pour vrai, 0 pour faux.
func unSiVrai(b bool) int {
	if b {
		return 1
	}
	return 0
}
