package replay

// document_arme_tiree.go : L ARME TIREE EN MAIN, a chaque changement (`drawnWeapons[]`).
//
// CE QUE C EST. Le film ecrit, dans le record du bipede, le jeu d armes desire (`i42`) quand il
// change : l emplacement d arme que le joueur veut en main principale et, hors d un changement en
// cours, celui qu il tient. Chaque ligne dit quand (la frame), qui (le slot de la piste) et quel
// emplacement : `d` en main principale, `d2` en seconde main. C est la meme grandeur que
// `inventory[].d`, lue a CHAQUE changement et non aux seules images-cles (toutes les vingt secondes
// environ).
//
// CE QUE C EST EN PLUS DE CE QUE LE DOCUMENT PUBLIE DEJA. `weaponChanges[]` dit quelle FAMILLE
// d arme entre ou sort d un emplacement (une prise, un lacher, un echange) ; `loadouts[]` et
// `inventory[]` disent ce que le joueur porte. Aucun ne dit quel emplacement il tient a un instant
// donne entre deux images-cles : c est ce calque. Il ne nomme aucune arme : l emplacement se relie
// a l arme par `loadouts[].w` et `weaponChanges[].k`, comme `inventory[].d`.
//
// CE QU IL NE DIT PAS. Ni la fin d un changement en cours (`d` est l emplacement DESIRE : il precede
// l arme effectivement en main), ni la vie (le slot se relie a la vie par le temps, comme les
// autres calques par slot). `d` absent = le film n ecrit aucun emplacement ; l emplacement 0 est une
// valeur.

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// DrawnWeapon est UN changement du jeu d armes desire d une vie, sur l axe de frames du document.
type DrawnWeapon struct {
	// T est l index de frame de la trame delta qui porte la lecture, sur le meme axe que Point.T.
	T int `json:"t"`
	// Slot designe la Track concernee.
	Slot uint32 `json:"slot"`
	// D est l emplacement d arme desire en main principale, la meme grandeur qu `Inventory.D` (index
	// 0 a 3). Pointeur : l emplacement 0 est une valeur ; absent = le film n en designe aucun.
	D *int `json:"d,omitempty"`
	// D2 est l emplacement desire en seconde main ; absent hors ambidextrie.
	D2 *int `json:"d2,omitempty"`
}

// DrawnWeaponCoverage dit ce que le calque a lu et ce qu il a ecarte.
//
// L IDENTITE : read = noTrack + beforeOrigin + afterEnd + published. `slotUnbound` et `duplicates`
// ne sont PAS dans `read` : ce sont les lectures que le canal a ecartees avant de les rendre.
type DrawnWeaponCoverage struct {
	// ComponentAbsent : l archetype bipede du film ne declare pas `i42` ; aucune lecture n a eu lieu.
	ComponentAbsent bool `json:"componentAbsent,omitempty"`
	// Read : lectures rendues par le canal (un changement par slot et par trame).
	Read int `json:"read"`
	// SlotUnbound : lectures ecartees parce que leur slot n est pas lie a un bipede dans la table
	// d entites de la marche. Duplicates : la meme lecture re-publiee par une seconde traversee du
	// meme record.
	SlotUnbound int `json:"slotUnbound,omitempty"`
	Duplicates  int `json:"duplicates,omitempty"`
	// NoTrack : lectures dont le slot n a aucune piste publiee. BeforeOrigin et AfterEnd : hors de
	// l axe de frames du match.
	NoTrack      int `json:"noTrack,omitempty"`
	BeforeOrigin int `json:"beforeOrigin,omitempty"`
	AfterEnd     int `json:"afterEnd,omitempty"`
	// Published : les changements publies.
	Published int `json:"published"`
}

// DrawnWeaponInput est ce que la marche des trames donne au calque : les changements lus et ce que
// leur canal a vu.
type DrawnWeaponInput struct {
	Reads []types.DrawnWeaponRead
	Stats types.DrawnWeaponStats
}

// emplacementPublie rend l emplacement lu, nil pour -1 (aucun emplacement designe).
func emplacementPublie(v int8) *int {
	if v < 0 {
		return nil
	}
	e := int(v)
	return &e
}

// buildDrawnWeapons projette les lectures sur l axe de frames : rien hors de l axe, rien dont le slot
// n a pas de piste publiee. Le tri est celui de la marche (instant, puis slot), conserve.
func buildDrawnWeapons(in DrawnWeaponInput, publies map[uint32]bool, axe axeDeFrames) ([]DrawnWeapon, DrawnWeaponCoverage) {
	st := in.Stats
	cov := DrawnWeaponCoverage{ComponentAbsent: st.Absent, Read: len(in.Reads),
		SlotUnbound: st.SlotUnbound, Duplicates: st.Duplicates}
	var out []DrawnWeapon
	for _, r := range in.Reads {
		t, refus := frameDeLAction(r.TimestampUS, axe)
		switch {
		case refus == refusAvantOrigine || refus == refusSansOrigine:
			cov.BeforeOrigin++
		case refus == refusApresFin:
			cov.AfterEnd++
		case !publies[r.Slot]:
			cov.NoTrack++
		default:
			out = append(out, DrawnWeapon{T: t, Slot: r.Slot, D: emplacementPublie(r.Principal), D2: emplacementPublie(r.Second)})
		}
	}
	cov.Published = len(out)
	slices.SortStableFunc(out, func(a, b DrawnWeapon) int { return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Slot, b.Slot)) })
	return out, cov
}

// poserLesArmesTirees publie le calque : seulement quand la marche des trames a ete jouee, une lecture
// absente n etant pas « aucun changement ».
func (a *assemblage) poserLesArmesTirees() {
	if !a.opt.DrawnWeapons.Stats.Scanned {
		return
	}
	armes, cov := buildDrawnWeapons(a.opt.DrawnWeapons, publishedSlots(a.doc.Tracks), axeDeFrames{a.origin, a.step, a.doc.FrameCount})
	a.doc.DrawnWeapons = armes
	a.doc.Coverage.DrawnWeapons = &cov
	slog.InfoContext(a.ctx, "rejeu : armes tirees", "lues", cov.Read, "slotNonLie", cov.SlotUnbound, "doublons", cov.Duplicates,
		"sansPiste", cov.NoTrack, "avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd, "publiees", cov.Published,
		"composantAbsent", cov.ComponentAbsent)
}
