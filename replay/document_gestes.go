package replay

// document_gestes.go — LES GESTES DU JOUEUR, lus dans les messages de la vue A : le coup de mêlée et le lancer
// amorcé.
//
// CE QUE C EST. Le film écrit, dans la liste d événements de la trame, un message quand un joueur ENGAGE un
// geste : `biped_melee_initiate` (genre 40, le coup part) et `biped_throw_initiate` (genre 39, le lancer est
// amorcé). Chaque ligne dit quand (la frame et la milliseconde), qui (`place`, l index de joueur du film tel que
// le message l écrit, le même espace que `reloads[].place` et `roster[].filmIndex`) et quel geste (`kind`).
//
// CE QUE CA NE DIT PAS. Ni le résultat ni la durée : un coup part qu il touche ou non (`damage[]` dit ce qui
// touche), et une amorce n est pas une grenade (`grenades[]` dit le projectile, qui naît plus tard). Les deux
// sont des faits distincts : l amorce et la sortie. Ni la vie : la place se relie à la vie par le roster et sa
// présence, comme celle des rechargements.
//
// LA PORTE DE CADENCE. Ces gestes ne se publient que sur un film dont les commandes sont enregistrées à pleine
// cadence ([assemblage.cadencePleineDesCommandes]) ; sur un autre la couverture compte les gestes lus comme
// retenus (`gated`) et le calque reste vide.
//
// Le canal de la grammaire rend un geste par (trame, genre, joueur, références) : les copies du même message dans
// la même trame sont retirées et comptées (`coverage.gestures.copies`). Une trame dont la liste d événements
// s arrête sur un message non lu perd ceux qui la suivent (`coverage.gestures.stoppedFrames`) : le compte d un tel
// film est un plancher.

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Les genres de geste, tels que `gestures[].kind` les écrit.
const (
	// GestureKindMelee : le coup de mêlée part.
	GestureKindMelee = "melee"
	// GestureKindThrowStart : le lancer est amorcé ; le projectile part plus tard (`grenades[]`).
	GestureKindThrowStart = "throwStart"
)

// microsecondesParMilliseconde convertit l horloge du film (microsecondes) en millisecondes.
const microsecondesParMilliseconde = 1000

// Gesture est UN geste, sur l axe de frames du document.
type Gesture struct {
	// T est l index de frame de la trame qui écrit le message, sur le même axe que Point.T.
	T int `json:"t"`
	// Ms est l instant de la trame à la milliseconde, depuis l origine de l axe de frames (la frame 0).
	Ms int `json:"ms"`
	// Place est l index de joueur de l auteur tel que le message l écrit : le même espace que
	// `reloads[].place` et que `roster[].filmIndex`.
	Place int `json:"place"`
	// Kind est le geste : [GestureKindMelee] ou [GestureKindThrowStart].
	Kind string `json:"kind"`
}

// GestureCoverage dit ce que le calque a lu et ce qu il a ecarte.
//
// L IDENTITE : read = copies + gated + withoutPlayer + beforeOrigin + afterEnd + published.
type GestureCoverage struct {
	// Frames, ClosedFrames, StoppedFrames : les trames delta du film et ce que la lecture de leur liste
	// d événements en a fait, lue jusqu au terminateur ou arrêtée par un message non lu (ceux qui suivent
	// l arrêt sont perdus).
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	// Read : les messages de geste lus. Copies : ceux qui copient un message de la même trame (même genre,
	// même auteur, mêmes références) et ne sont pas des gestes de plus.
	Read   int `json:"read"`
	Copies int `json:"copies"`
	// Gated : gestes retenus par la porte de cadence (le film n enregistre pas ses commandes à pleine cadence).
	Gated int `json:"gated"`
	// WithoutPlayer : gestes dont le message ne porte pas d index de joueur — écartés.
	WithoutPlayer int `json:"withoutPlayer"`
	// BeforeOrigin et AfterEnd : gestes hors de l axe de frames du match — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// Published : les gestes publiés.
	Published int `json:"published"`
}

// GestureInput est ce que la marche des trames donne au calque : les gestes lus et ce que leur canal a vu.
type GestureInput struct {
	Reads []types.GestureRead
	Stats types.GestureStats
}

// millisecondes rend l instant `ts` (microsecondes du film) en millisecondes depuis l origine de l axe ;
// `ts` n est pas avant l origine (la frame de l action l a vérifié).
func (axe axeDeFrames) millisecondes(ts uint64) int {
	return int((ts - axe.origin) / microsecondesParMilliseconde) //nolint:gosec // horloge du film
}

// nomDuGeste rend l identifiant publié d un geste, ou faux pour une valeur que le canal ne rend pas.
func nomDuGeste(k types.GestureKind) (string, bool) {
	switch k {
	case types.GestureMelee:
		return GestureKindMelee, true
	case types.GestureThrowStart:
		return GestureKindThrowStart, true
	}
	return "", false
}

// buildGestures pose les gestes lus sur l axe de frames. `pleineCadence` faux : aucun geste ne se publie, tous
// ceux que le canal rend se comptent en `gated`.
func buildGestures(in GestureInput, axe axeDeFrames, pleineCadence bool) ([]Gesture, GestureCoverage) {
	st := in.Stats
	cov := GestureCoverage{Frames: st.Frames, ClosedFrames: st.Closed, StoppedFrames: st.Stopped,
		Read:   st.Melee.Messages + st.Melee.Copies + st.Throw.Messages + st.Throw.Copies,
		Copies: st.Melee.Copies + st.Throw.Copies}
	var out []Gesture
	for _, g := range in.Reads {
		nom, connu := nomDuGeste(g.Kind)
		switch {
		case !connu:
			continue
		case !pleineCadence:
			cov.Gated++
			continue
		case g.Player == types.GestureNoPlayer:
			cov.WithoutPlayer++
			continue
		}
		t, refus := frameDeLAction(g.TimestampUS, axe)
		switch refus {
		case refusAucun:
			out = append(out, Gesture{T: t, Ms: axe.millisecondes(g.TimestampUS), Place: int(g.Player), Kind: nom})
		case refusAvantOrigine, refusSansOrigine:
			cov.BeforeOrigin++
		case refusApresFin:
			cov.AfterEnd++
		}
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b Gesture) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Ms, b.Ms), cmp.Compare(a.Place, b.Place), cmp.Compare(a.Kind, b.Kind))
	})
	return out, cov
}

// cadencePleineDesCommandes est LA PORTE DE CADENCE des gestes : vrai quand les commandes du film sont
// enregistrées à pleine cadence, le verdict que la grammaire mesure sur le pas des entrées de contrôle
// ([types.ControlHoldStats.PleineCadence]) et que `accroupi` et `saut` consultent aussi. Un film dont ce pas
// n est pas mesuré est sous la porte. Les rechargements ne passent pas par elle.
func (a *assemblage) cadencePleineDesCommandes() bool { return a.opt.ControlHoldStats.PleineCadence() }

// poserLesGestes publie le calque : seulement quand la marche des trames a lu au moins une trame.
func (a *assemblage) poserLesGestes() {
	if a.opt.Gestures.Stats.Frames == 0 {
		return
	}
	gestes, cov := buildGestures(a.opt.Gestures, axeDeFrames{a.origin, a.step, a.doc.FrameCount}, a.cadencePleineDesCommandes())
	a.doc.Gestures = gestes
	a.doc.Coverage.Gestures = &cov
	slog.InfoContext(a.ctx, "rejeu : gestes", "lus", cov.Read, "copies", cov.Copies, "retenus", cov.Gated, "sansJoueur", cov.WithoutPlayer,
		"avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd, "publies", cov.Published,
		"tramesArretees", cov.StoppedFrames, "trames", cov.Frames)
}
