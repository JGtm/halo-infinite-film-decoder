package replay

// shots_visee.go — LA VISEE PUBLIEE SUR UN TIR : celle du tireur a l instant du tir.
//
// LA SOURCE EST LA VISEE i21 DU TIREUR, pas le champ de visee du record de tir. Le film replique
// `unit-desired-aiming-vector` (i21) a chaque changement : dans le record de position du bipede, et
// dans un record SANS position quand le bipede ne replique plus la sienne (un occupant de vehicule,
// cf. `grammar.ScanBipedAimOnly`). La derniere lecture du slot tireur AU PLUS TARD a l instant du
// tir EST sa visee a cet instant, tant qu elle appartient a la meme vie (`viseeTenueMaxUS`). La tenir
// bat l interpolation entre la lecture d avant et celle d apres : une interpolation ne rend jamais
// l angle exact de la lecture suivante, et la mesure contre la visee exacte des tirs dont le record
// la porte (chronique du schema 110) l a tranchee en faveur de la lecture tenue.
//
// LE CHAMP DU RECORD N EST QU UN REPLI, NOMME ET COMPTE. Le record de tir 36 n ecrit sa visee que
// sous ses deux drapeaux nuls (`grammar.modalAimBitFrom`) ; elle sert quand le tireur n a aucune
// lecture i21 tenable. Sans l une ni l autre, le tir se publie sans direction, et c est compte aussi.

import (
	"cmp"
	"math"
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// viseeTenueMaxUS borne l age de la lecture i21 tenue pour un tir : au-dela, elle appartient a une
// autre vie du slot (meme seuil que la decoupe des vies).
const viseeTenueMaxUS = lifeGapUS

// ShotAimCoverage dit D OU VIENT la direction de chaque tir pose sur la carte (a pied ou en
// vehicule, avant la porte des pistes publiees) : Track + Record + None = Attached + Unpublished.
type ShotAimCoverage struct {
	// Track : visee i21 du tireur, la derniere au plus tard a l instant du tir.
	Track int `json:"track"`
	// SameTick : parmi Track, celles lues dans le paquet meme du tir.
	SameTick int `json:"sameTick"`
	// Record : champ de visee du record de tir, faute de visee i21 tenable (repli
	// `fallback.NomViseeDuTirLueDansLeRecord`).
	Record int `json:"record"`
	// None : ni l une ni l autre, le tir se publie sans direction.
	None int `json:"none"`
}

// lectureVisee est une lecture i21 d un slot : son instant et ses deux angles, en degres.
type lectureVisee struct {
	tUS  uint64
	h, p float32
}

// viseurDesTirs pose la visee des tirs : les lectures i21 par slot, triees par instant, et la
// couverture qu il tient a jour pour les deux portes des tirs (a pied, en vehicule).
type viseurDesTirs struct {
	parSlot map[uint32][]lectureVisee
	cov     *ShotAimCoverage
	fb      *fallback.Compteur
}

// nouveauViseur indexe par slot les lectures i21 des records de position et celles des records sans
// position. Les ex aequo gardent l ordre d entree (positions d abord).
func nouveauViseur(pos []grammar.BipedPosition, sansPosition []grammar.BipedAim, fb *fallback.Compteur) *viseurDesTirs {
	v := &viseurDesTirs{parSlot: map[uint32][]lectureVisee{}, cov: &ShotAimCoverage{}, fb: fb}
	for _, p := range pos {
		h, ok := p.AimHeadingDeg()
		if !ok {
			continue
		}
		pi, _ := p.AimPitchDeg()
		v.parSlot[p.Slot] = append(v.parSlot[p.Slot], lectureVisee{tUS: p.TimestampUS, h: h, p: pi})
	}
	for _, a := range sansPosition {
		v.parSlot[a.Slot] = append(v.parSlot[a.Slot],
			lectureVisee{tUS: a.TimestampUS, h: a.AimHeadingDeg(), p: a.AimPitchDeg()})
	}
	for _, ls := range v.parSlot {
		slices.SortStableFunc(ls, func(a, b lectureVisee) int { return cmp.Compare(a.tUS, b.tUS) })
	}
	return v
}

// viser pose sur `s` la visee du tir `e`, tire par le bipede du slot `slot`.
func (v *viseurDesTirs) viser(s *Shot, e grammar.FireEvent, slot uint32) {
	if l, ok := v.tenue(slot, e.TimestampUS); ok {
		s.H, s.P = headingForJSON(l.h), pitchForJSON(l.p)
		v.cov.Track++
		if l.tUS == e.TimestampUS {
			v.cov.SameTick++
		}
		return
	}
	h, ok := e.AimHeadingDeg()
	if !ok {
		v.cov.None++
		return
	}
	v.fb.Declenche(fallback.NomViseeDuTirLueDansLeRecord)
	s.H, s.P = headingForJSON(float32(h)), pitchForJSON(elevationDuVecteur(e.Aim))
	v.cov.Record++
}

// tenue rend la derniere lecture i21 du slot au plus tard a `tUS`, si elle a au plus
// `viseeTenueMaxUS`.
func (v *viseurDesTirs) tenue(slot uint32, tUS uint64) (lectureVisee, bool) {
	ls := v.parSlot[slot]
	i := sort.Search(len(ls), func(i int) bool { return ls[i].tUS > tUS }) - 1
	if i < 0 || tUS-ls[i].tUS > viseeTenueMaxUS {
		return lectureVisee{}, false
	}
	return ls[i], true
}

// elevationDuVecteur rend l elevation, en degres, d un vecteur de visee (meme sens que le cap :
// atan2 de la composante verticale sur la norme horizontale).
func elevationDuVecteur(a [3]float32) float32 {
	return float32(math.Atan2(float64(a[2]), math.Hypot(float64(a[0]), float64(a[1]))) * 180 / math.Pi)
}
