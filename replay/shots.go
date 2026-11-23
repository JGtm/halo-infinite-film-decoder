package replay

// Rattachement des ÉVÉNEMENTS DE TIR (record film type 105) aux trajectoires du rejeu.
//
// L'ÉVÉNEMENT PORTE DÉJÀ SON AUTEUR. Le champ FilmIndex est écrit dans le film ; il n'est ni
// deviné ni voté. Ce fichier ne cherche donc pas QUI a tiré — il cherche OÙ ce joueur se
// trouvait, pour poser le tir sur la carte.
//
// POURQUOI CE N'EST PAS IMMÉDIAT : les positions sont indexées par SLOT de biped, et un slot
// change à chaque réapparition. Rien dans l'événement de tir ne donne le slot. Le pont
// slot -> joueur est construit ailleurs (owners.go, à partir du fil des morts) ; ici on ne fait
// que l'appliquer, en comptant tout ce qu'on écarte.
//
// LE VOTE PAR ÉGALITÉ DES CAPS DE VISÉE A ÉTÉ SUPPRIMÉ le 2026-07-28. Il désignait le tireur en
// comparant la direction du tir aux directions de visée des bipeds — une inférence, lisible dans
// ~44 % des records seulement, et qui contredisait la lecture sur 4 slots. Mesure du retrait :
// 496 -> 475 tirs publiés, et 4 -> 0 désaccords.

import (
	"cmp"
	"math"
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// shotPosToleranceUS : écart temporel maximal entre l'événement et l'échantillon de position
// utilisé (les records de position arrivent à ~60 Hz).
const shotPosToleranceUS = 120_000

// slotTrack est l'index temporel des positions d'un slot, pour la recherche par instant.
type slotTrack struct {
	// all est le nuage des positions dont la piste est extraite : PARTAGE, JAMAIS COPIE ni modifie.
	all []grammar.BipedPosition
	// idx range les rangs de `all` qui sont a ce slot, TRIES par TimestampUS (les ex aequo gardent
	// l ordre du film). Un rang tient sur 4 octets : une position, 176.
	idx []int32
}

// n est le nombre de positions de la piste.
func (s slotTrack) n() int { return len(s.idx) }

// pt rend la i-eme position de la piste, dans l ordre des instants.
func (s slotTrack) pt(i int) grammar.BipedPosition { return s.all[s.idx[i]] }

// ts rend l instant de la i-eme position de la piste.
func (s slotTrack) ts(i int) uint64 { return s.all[s.idx[i]].TimestampUS }

// at renvoie l'échantillon le plus proche de tUS et son écart.
func (s slotTrack) at(tUS uint64) (grammar.BipedPosition, uint64) {
	i := sort.Search(s.n(), func(i int) bool { return s.ts(i) >= tUS })
	best, bestD := grammar.BipedPosition{}, uint64(math.MaxUint64)
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= s.n() {
			continue
		}
		d := absDiffU64(s.ts(j), tUS)
		if d < bestD {
			bestD, best = d, s.pt(j)
		}
	}
	return best, bestD
}

// orphanShot est un événement de tir que la porte du BIPÈDE a écarté, avec la cause du rejet.
//
// IL EXISTE PARCE QUE LE REJET N'EST PAS LA FIN DE L'HISTOIRE. Un joueur EMBARQUÉ cesse de
// répliquer la position de son bipède (primitive V1a.4) : ses tirs sortent d'ici sous
// `reasonNoSlot`, non parce que le film ignorerait qui a tiré — l'événement porte son tireur —
// mais parce qu'il n'y a AUCUNE position de bipède à poser sur la carte. La seconde porte, celle
// du VÉHICULE (`vehicle_shots.go`), reprend exactement cette liste et lui donne la position du
// véhicule occupé. LA CAUSE VOYAGE AVEC L'ÉVÉNEMENT : sans elle, la couverture ne saurait pas
// quel compteur décrémenter quand la seconde porte rattache.
type orphanShot struct {
	ev     grammar.FireEvent
	reason rejectReason
}

// orphelinsDesTirs est ce que la porte du vehicule recoit de celle du bipede : les orphelins, et le
// viseur des tirs, dont elle partage les lectures i21 et la couverture.
type orphelinsDesTirs struct {
	tirs   []orphanShot
	viseur *viseurDesTirs
}

// buildShots rattache les events de tir aux slots et produit les tirs publiables, AVEC la
// couverture : combien rattachés sur combien disponibles, et pourquoi le reste est écarté.
//
// AUCUN CHEMIN DE REJET N'EST MUET ICI. C'est l'invariant que le test vérifie :
// rattachés + rejetés = disponibles, exactement.
//
// LE SECOND RETOUR est la liste des ORPHELINS reprenables par la porte du véhicule : les rejets
// « sans slot » et « hors fenêtre », ceux dont la signature est celle d'un tireur embarqué. Les
// AMBIGUS n'y sont PAS, et c'est une mesure : l'ambiguïté naît de DEUX slots du même joueur qui
// répliquent tous deux une position à cet instant — donc d'un joueur qui n'est PAS embarqué.
//
// LA VISEE de chaque tir pose vient de viseur (cf. shots_visee.go), qui en tient la couverture :
// la porte du vehicule la partage, et cov.Aim pointe la meme.
func buildShots(tracks map[uint32]slotTrack, events []grammar.FireEvent, clock replayClock,
	occ occupantsDesSlots, viseur *viseurDesTirs) ([]Shot, []orphanShot, LayerCoverage) {
	cov := LayerCoverage{Available: len(events), Aim: viseur.cov}
	if len(tracks) == 0 || len(events) == 0 || occ.vide() {
		cov.NoSlot = len(events) // rien pour rattacher : tout est écarté, et c'est dit
		return nil, nil, cov
	}
	pose := poseurDeTirs{origin: clock.origin, step: clock.step, viseur: viseur}
	var out []Shot
	var orphans []orphanShot
	for _, e := range events {
		// LA REFERENCE 0 D ABORD (lot M4b.4) : le record NOMME l unite qui a tire. Quand c est un
		// bipede qui replique une position a l instant, le tir est le sien — sans passer par
		// l index de tireur, qui n est pas le tireur sur toutes les builds (HI_1_4_1 : un champ
		// `d` constant d un tireur a l autre, mesure sur a521164d).
		if s, ok := tirParLUnite(tracks, e, pose); ok {
			cov.count(reasonAttached)
			cov.ByUnit++
			if pi, connu := occ.indexA(s.Slot, int64(e.TimestampUS)); connu && e.HasShooter && pi != e.FilmIndex {
				cov.UnitOtherIndex++
			}
			out = append(out, s)
			continue
		}
		slot, reason := slotFor(tracks, occ, e.FilmIndex, e.TimestampUS)
		if reason != reasonAttached {
			cov.count(reason)
			if reason == reasonNoSlot {
				orphans = append(orphans, orphanShot{ev: e, reason: reason})
			}
			continue
		}
		p, d := tracks[slot].at(e.TimestampUS)
		if d > shotPosToleranceUS || !p.HasWorld {
			cov.count(reasonOutOfWindow)
			orphans = append(orphans, orphanShot{ev: e, reason: reasonOutOfWindow})
			continue
		}
		cov.count(reasonAttached)
		out = append(out, pose.tir(e, slot, p))
	}
	trierTirs(out)
	return out, orphans, cov
}

// tirParLUnite pose un tir sur le bipede que sa reference 0 designe, quand ce bipede replique une
// position assez pres de l instant du tir.
func tirParLUnite(tracks map[uint32]slotTrack, e grammar.FireEvent, pose poseurDeTirs) (Shot, bool) {
	if !e.Unit.Present {
		return Shot{}, false
	}
	tr, ok := tracks[e.Unit.Slot]
	if !ok {
		return Shot{}, false
	}
	p, d := tr.at(e.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return Shot{}, false
	}
	return pose.tir(e, e.Unit.Slot, p), true
}

// poseurDeTirs construit les tirs publies a pied : la grille du document et le viseur.
type poseurDeTirs struct {
	origin, step uint64
	viseur       *viseurDesTirs
}

// tir construit le tir publie d un evenement, pose sur le slot et la position donnes, avec la
// visee du tireur a l instant du tir (cf. shots_visee.go).
func (b poseurDeTirs) tir(e grammar.FireEvent, slot uint32, p grammar.BipedPosition) Shot {
	s := Shot{
		T:    int((e.TimestampUS - b.origin) / b.step),
		Slot: slot,
		X:    round2(p.X),
		Y:    round2(p.Y),
		tUS:  e.TimestampUS,
	}
	b.viseur.viser(&s, e, slot)
	if e.WeaponID != 0 {
		s.Weapon = formatWeaponID(e.WeaponID)
	}
	return s
}

// indexBySlot regroupe les positions par slot, triées par instant. L index ne COPIE aucune position :
// chaque piste garde les rangs de ses positions dans `pos`, qui doit rester inchangee tant que
// l index sert.
func indexBySlot(pos []grammar.BipedPosition) map[uint32]slotTrack {
	nb := map[uint32]int{}
	for i := range pos {
		nb[pos[i].Slot]++
	}
	// UNE SEULE ALLOCATION de rangs, decoupee par slot : aucune position n est copiee.
	rangs := make([]int32, len(pos))
	out := make(map[uint32]slotTrack, len(nb))
	debut := 0
	for s, n := range nb {
		out[s] = slotTrack{all: pos, idx: rangs[debut : debut : debut+n]}
		debut += n
	}
	for i := range pos {
		t := out[pos[i].Slot]
		t.idx = append(t.idx, int32(i)) //nolint:gosec // un nuage de plus de 2^31 positions ne tient pas en memoire
		out[pos[i].Slot] = t
	}
	parInstant := func(a, b int32) int { return cmp.Compare(pos[a].TimestampUS, pos[b].TimestampUS) }
	for _, t := range out {
		// Les ex aequo (deux positions d un slot au meme horodatage) gardent l ORDRE DU FILM (lot J10.1,
		// 2026-09-27, DT-9) : `at` rend la plus proche, donc l une d elles, et c est elle qu un tir recoit.
		if !slices.IsSortedFunc(t.idx, parInstant) {
			slices.SortStableFunc(t.idx, parInstant)
		}
	}
	return out
}

// La porte de rattachement vit désormais dans coverage.go sous le nom `slotFor`, et elle
// REND SA CAUSE au lieu d'un booléen. L'ancienne `uniqueSlotFor` est supprimée : elle ne
// distinguait pas « slot introuvable » de « slot ambigu », qui désignent pourtant deux
// chantiers opposés, et ses appelants faisaient `continue` sans rien compter.

func absDiffU64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// formatWeaponID rend l'identifiant 64 bits en hexadécimal : un entier 64 bits ne survit
// pas au `number` de JavaScript (53 bits de mantisse).
func formatWeaponID(id uint64) string {
	const hexDigits = "0123456789ABCDEF"
	buf := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[id&0xF]
		id >>= 4
	}
	return "0x" + string(buf)
}
