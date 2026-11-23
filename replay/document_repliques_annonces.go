package replay

// document_repliques_annonces.go — LES ANNONCES DE L ANNONCEUR ET DES SCRIPTS DE MODE, lues dans les messages `Dialogue2D`.
//
// CE QUE C EST. Le film ecrit une annonce dans la liste d evenements de la trame (genre 76 de la vue A) : l identifiant du son,
// un masque de 32 bits des destinataires, un index de joueur et une seconde valeur dont le sens n est pas etabli. Ce calque pose
// chaque annonce sur l axe de frames avec son son et son masque.
//
// UN MESSAGE PAR DESTINATAIRE, UNE ANNONCE PAR SON ET PAR FRAME. Mesure sur un film de huit joueurs : les 951 messages ne portent
// jamais qu UN bit de masque, dans les bits 24 a 31 (le bit 25 n y est jamais pose), et se regroupent en 481 lignes sur l axe (six tombent avant son origine) — le jeu
// ecrit l annonce une fois par destinataire, dans la meme trame. Les messages d une meme frame qui portent le meme son sont donc
// une seule ligne dont le masque est l union de leurs masques ; les messages ainsi repris se comptent
// (`coverage.announcements.merged`). Le film ne dit pas quelle variante audio a joue : le document ne le dit pas non plus.
//
// LES DESTINATAIRES RESTENT UN MASQUE. Le bit k du masque ne se relie a aucun joueur de facon prouvee : sur le film mesure les
// ensembles de bits d un son d equipe (par exemple les bits 0, 4, 5 et 6) tombent sur les deux equipes du roster quel que soit le
// sens dans lequel on numerote les bits (le bit 24 + i pour le joueur i, ou le bit 31 - i), et les medailles et les morts voisines
// ne tranchent pas. Le masque est publie tel que le film l ecrit, en hexadecimal sur huit chiffres, le premier bit lu etant le
// bit de poids fort. L index de joueur du message n est pas publie : son sens n est pas etabli non plus (il reste dans les faits
// du film).

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Announcement est UNE annonce, sur l axe de frames du document.
type Announcement struct {
	// T est l index de frame de la trame qui ecrit le message, sur le meme axe que Point.T.
	T int `json:"t"`
	// Sound est l identifiant du son, en hexadecimal sur huit chiffres. Absent quand le message n en porte pas.
	Sound string `json:"sound,omitempty"`
	// Mask est le masque des destinataires, en hexadecimal sur huit chiffres : l union des masques des messages de la frame
	// qui portent ce son, tel que le film les ecrit (le premier bit lu est le bit de poids fort). Ses bits ne sont pas relies a
	// des joueurs.
	Mask string `json:"mask"`
}

// AnnouncementCoverage dit ce que le calque a lu, publie et ecarte.
//
// L IDENTITE : read = beforeOrigin + afterEnd + merged + published.
type AnnouncementCoverage struct {
	// Frames, ClosedFrames, StoppedFrames : comme `coverage.chatter`.
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	// Read : les annonces lues. BeforeOrigin et AfterEnd : celles hors de l axe de frames du match — ecartees. Merged : celles
	// reprises dans la ligne d un message de la meme frame et du meme son. Published : les lignes publiees.
	Read         int `json:"read"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	Merged       int `json:"merged"`
	Published    int `json:"published"`
}

// cleDeLigneDAnnonce identifie une ligne d annonce : la frame et le son.
type cleDeLigneDAnnonce struct {
	t    int
	son  uint32
	sonL bool
}

// buildAnnouncements pose les annonces lues sur l axe de frames, une ligne par son et par frame. Les lignes sont triees par
// frame, dans l ordre du film a frame egale.
func buildAnnouncements(in grammar.RepliquesLues, axe axeDeFrames) ([]Announcement, AnnouncementCoverage) {
	st := in.Stats
	cov := AnnouncementCoverage{Frames: st.Trames, ClosedFrames: st.Terminees, StoppedFrames: st.Arretees, Read: len(in.Annonces)}
	var out []Announcement
	masques := []uint32{}
	vues := map[cleDeLigneDAnnonce]int{}
	for _, m := range in.Annonces {
		t, refus := frameDeLAction(m.TS, axe)
		switch refus {
		case refusAvantOrigine, refusSansOrigine:
			cov.BeforeOrigin++
			continue
		case refusApresFin:
			cov.AfterEnd++
			continue
		}
		k := cleDeLigneDAnnonce{t: t, son: m.Son, sonL: m.ASon}
		if i, ok := vues[k]; ok {
			masques[i] |= m.Destinataires
			cov.Merged++
			continue
		}
		vues[k] = len(out)
		masques = append(masques, m.Destinataires)
		l := Announcement{T: t}
		if m.ASon {
			l.Sound = hexaSurHuitChiffres(m.Son)
		}
		out = append(out, l)
	}
	for i := range out {
		out[i].Mask = hexaSurHuitChiffres(masques[i])
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b Announcement) int { return cmp.Compare(a.T, b.T) })
	return out, cov
}

// poserLesAnnonces publie le calque des annonces et sa couverture.
func (a *assemblage) poserLesAnnonces(in grammar.RepliquesLues, axe axeDeFrames) {
	ann, cov := buildAnnouncements(in, axe)
	a.doc.Announcements = ann
	a.doc.Coverage.Announcements = &cov
	slog.InfoContext(a.ctx, "rejeu : annonces", "lues", cov.Read, "avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd,
		"reprises", cov.Merged, "publiees", cov.Published, "tramesArretees", cov.StoppedFrames, "trames", cov.Frames)
}
