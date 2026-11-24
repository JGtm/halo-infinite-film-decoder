package replay

// ammo_reads.go : LES MUNITIONS PORTEES, a chaque changement de valeur (`ammoReads[]`).
//
// CE QUE C EST. Un joueur porte deux armes (emplacements 0 et 1) ; pour chacune, le film ecrit un
// chargeur, une reserve et, pour les armes a charge, une jauge. Deux canaux disent cette meme
// grandeur : le record de bipede des IMAGES-CLES (`inventory[].am`, toutes les vingt secondes
// environ) et les composants de munitions des trames DELTA, transmis au changement (un tir, un
// rechargement, un ramassage). Ce calque les met sur UN axe de temps : une ligne par (vie,
// emplacement) et par changement de valeur, le HUD de munitions et le compte de chargeur par tir
// se lisent dessus.
//
// CE QUI EST PUBLIE EN PLUS DE `inventory[].am` : les lectures des trames delta, absentes de
// `inventory` (qui ne porte que les images-cles) ; les lectures d image-cle y sont reprises pour que
// l axe soit complet, et ne s y repetent que lorsqu elles changent la valeur connue.
//
// CE QUI N EST PAS DIT.
//   - `w` est le RANG d emplacement (0 ou 1, comme `inventory[].am[]` et `loadouts[].k`), pas une
//     famille d arme : l arme de l emplacement se lit par `loadouts` et `weaponChanges`.
//   - `gauge` est la fraction CONSOMMEE, de 0 a 1 sur 4096 niveaux (quantum / 4095), pas ce qui
//     reste : une barre de charge restante affiche le complement (cf. [AmmoSlot.Gauge]).
//   - une ligne ne porte que les champs que le film ecrit : un champ absent est « non transmis »,
//     jamais zero. Une arme a charge au plein n ecrit aucun champ : aucune ligne.
//   - l accord des deux canaux n est pas garanti : ils divergent parfois, et rien ne les arbitre ;
//     la lecture la plus recente fait foi, `src` dit d ou elle vient.
//
// LA PORTE DU CANAL DELTA. Quand le scanner a refuse en bloc le canal munitions d un film (la
// distribution des chargeurs y est contaminee, `InventoryDeltaAmmoRefused`), aucune lecture delta ne
// se publie et la couverture le dit ; les images-cles restent.

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// quantumDeJaugeDelta est le plein de la fraction R(12) d un delta, le meme diviseur que la jauge des
// images-cles ([AmmoSlot.Gauge]).
const quantumDeJaugeDelta = 4095.0

// AmmoRead est une lecture de munitions d une vie et d un emplacement d arme.
type AmmoRead struct {
	// T est l index de frame, sur le meme axe que Point.T.
	T int `json:"t"`
	// Ms est l instant a la milliseconde, depuis l origine de l axe de frames (la frame 0).
	Ms int `json:"ms"`
	// Slot designe la Track concernee, donc une vie.
	Slot uint32 `json:"slot"`
	// W est le rang d emplacement de l arme (0 ou 1).
	W int `json:"w"`
	// Mag et Res sont le chargeur et la reserve ; Gauge la fraction consommee. Chacun est absent
	// quand le film ne l ecrit pas dans cette lecture.
	Mag   *uint32  `json:"mag,omitempty"`
	Res   *uint32  `json:"res,omitempty"`
	Gauge *float32 `json:"gauge,omitempty"`
	// Src dit le canal de la lecture : [GrenadeSrcKeyframe] ou [GrenadeSrcDelta].
	Src string `json:"src"`
}

// AmmoReadCoverage dit ce que le calque a lu et ce qu il a ecarte.
//
// L IDENTITE : read = beforeOrigin + afterEnd + noTrack + unchanged + fromKeyframe + fromDelta.
// `empty` et `deltaRefused` sont hors de l identite : une lecture vide ne porte aucune valeur, et
// un canal refuse n est pas lu.
type AmmoReadCoverage struct {
	// Read : lectures portant au moins une valeur, sur les deux canaux.
	Read int `json:"read"`
	// Empty : emplacements d image-cle sans aucune valeur (une arme a charge au plein n en ecrit pas).
	Empty int `json:"empty,omitempty"`
	// BeforeOrigin et AfterEnd : lectures hors de l axe de frames du match. NoTrack : lectures dont le
	// slot n a aucune piste publiee.
	BeforeOrigin int `json:"beforeOrigin,omitempty"`
	AfterEnd     int `json:"afterEnd,omitempty"`
	NoTrack      int `json:"noTrack,omitempty"`
	// Unchanged : lectures dont chaque valeur egale celle que l axe porte deja pour la vie et
	// l emplacement.
	Unchanged int `json:"unchanged,omitempty"`
	// FromKeyframe et FromDelta : les lignes publiees par canal.
	FromKeyframe int `json:"fromKeyframe"`
	FromDelta    int `json:"fromDelta"`
	// DeltaRefused : le canal delta de ce film est refuse en bloc par le scanner ; aucune lecture
	// delta n est entree dans `read`.
	DeltaRefused bool `json:"deltaRefused,omitempty"`
}

// ammoDatee est une lecture avec son instant exact, pour l ordre de l axe.
type ammoDatee struct {
	r  AmmoRead
	us uint64
}

// ordreDeLaSource classe les canaux a instant egal : l image-cle d abord, que le delta complete.
func ordreDeLaSource(src string) int {
	if src == GrenadeSrcKeyframe {
		return 0
	}
	return 1
}

// jaugeDuQuantum rend la fraction consommee d un quantum R(12) brut.
func jaugeDuQuantum(q uint32) *float32 {
	v := float32(float64(q) / quantumDeJaugeDelta)
	return &v
}

// lecturesDeMunitions rassemble les lectures des deux canaux, une ligne par emplacement qui porte
// une valeur, sans les situer encore sur l axe. `refus` ferme le canal delta.
func lecturesDeMunitions(kf []types.KeyframeInventory, deltas []types.InventoryDelta, refus bool, cov *AmmoReadCoverage) []ammoDatee {
	var out []ammoDatee
	for _, r := range kf {
		if !r.AmmoRead {
			continue
		}
		for w, a := range ammoSlotsOf(r) {
			if a.Mag == nil && a.Res == nil && a.Gauge == nil {
				cov.Empty++
				continue
			}
			out = append(out, ammoDatee{AmmoRead{Slot: r.Slot, W: w, Mag: a.Mag, Res: a.Res, Gauge: a.Gauge, Src: GrenadeSrcKeyframe}, r.TimestampUS})
		}
	}
	if refus {
		return out
	}
	for _, d := range deltas {
		for _, a := range d.Ammo {
			if a.Mag == nil && a.Res == nil && a.FracQ == nil {
				continue
			}
			l := AmmoRead{Slot: d.Slot, W: a.WeaponSlot, Mag: a.Mag, Res: a.Res, Src: GrenadeSrcDelta}
			if a.FracQ != nil {
				l.Gauge = jaugeDuQuantum(*a.FracQ)
			}
			out = append(out, ammoDatee{l, d.TimestampUS})
		}
	}
	return out
}

// valeurDejaConnue dit que chaque champ que la lecture porte egale la derniere valeur connue de
// l emplacement, et met la derniere valeur connue a jour.
func valeurDejaConnue(connue *AmmoRead, l AmmoRead) bool {
	egal := true
	if l.Mag != nil {
		egal = egal && connue.Mag != nil && *connue.Mag == *l.Mag
		connue.Mag = l.Mag
	}
	if l.Res != nil {
		egal = egal && connue.Res != nil && *connue.Res == *l.Res
		connue.Res = l.Res
	}
	if l.Gauge != nil {
		egal = egal && connue.Gauge != nil && *connue.Gauge == *l.Gauge
		connue.Gauge = l.Gauge
	}
	return egal
}

// buildAmmoReads met les lectures des deux canaux sur l axe de frames : rien hors de l axe, rien dont
// le slot n a pas de piste publiee, et une seule ligne par changement de valeur pour une vie et un
// emplacement. L ordre est celui de l horloge du film, a instant egal l image-cle puis le slot, le
// rang d emplacement : il ne depend pas de l ordre des entrees.
func buildAmmoReads(kf []types.KeyframeInventory, deltas []types.InventoryDelta, refus bool,
	publies map[uint32]bool, axe axeDeFrames) ([]AmmoRead, AmmoReadCoverage) {
	cov := AmmoReadCoverage{DeltaRefused: refus}
	var gardees []ammoDatee
	for _, d := range lecturesDeMunitions(kf, deltas, refus, &cov) {
		cov.Read++
		t, refusDePose := frameDeLAction(d.us, axe)
		switch {
		case refusDePose == refusAvantOrigine || refusDePose == refusSansOrigine:
			cov.BeforeOrigin++
		case refusDePose == refusApresFin:
			cov.AfterEnd++
		case !publies[d.r.Slot]:
			cov.NoTrack++
		default:
			d.r.T, d.r.Ms = t, axe.millisecondes(d.us)
			gardees = append(gardees, d)
		}
	}
	slices.SortStableFunc(gardees, func(a, b ammoDatee) int {
		return cmp.Or(cmp.Compare(a.us, b.us), cmp.Compare(ordreDeLaSource(a.r.Src), ordreDeLaSource(b.r.Src)),
			cmp.Compare(a.r.Slot, b.r.Slot), cmp.Compare(a.r.W, b.r.W))
	})
	var out []AmmoRead
	connues := map[[2]uint32]*AmmoRead{}
	for _, d := range gardees {
		cle := [2]uint32{d.r.Slot, uint32(d.r.W)} //nolint:gosec // rang d emplacement 0 ou 1
		if connues[cle] == nil {
			connues[cle] = &AmmoRead{}
		}
		if valeurDejaConnue(connues[cle], d.r) {
			cov.Unchanged++
			continue
		}
		out = append(out, d.r)
		if d.r.Src == GrenadeSrcDelta {
			cov.FromDelta++
		} else {
			cov.FromKeyframe++
		}
	}
	return out, cov
}

// poserLesMunitions publie le calque et sa couverture, quand l inventaire a ete lu (meme garde que
// `coverage.inventory` : un inventaire illisible n est pas « aucune munition »).
func (a *assemblage) poserLesMunitions() {
	if a.opt.Inventory == nil || a.doc.Coverage == nil {
		return
	}
	reads, cov := buildAmmoReads(a.opt.Inventory, a.opt.InventoryDeltas, a.opt.InventoryDeltaAmmoRefused,
		publishedSlots(a.doc.Tracks), axeDeFrames{a.origin, a.step, a.doc.FrameCount})
	a.doc.AmmoReads = reads
	a.doc.Coverage.AmmoReads = &cov
	slog.InfoContext(a.ctx, "rejeu : munitions portees", "lues", cov.Read, "vides", cov.Empty, "avantOrigine", cov.BeforeOrigin,
		"apresFin", cov.AfterEnd, "sansPiste", cov.NoTrack, "inchangees", cov.Unchanged,
		"imagesCles", cov.FromKeyframe, "delta", cov.FromDelta, "canalDeltaRefuse", cov.DeltaRefused)
}
