package replay

// pad_pickup_dating_cycle.go — UNE SEULE PRISE DE SOCLE PAR RÉAPPARITION DE L'ARME (repli
// `repli_prise_de_socle_premiere_du_cycle`).
//
// # LA RÈGLE DE JEU
//
// Une arme réapparue sur son socle n'en part qu'une fois : la PREMIÈRE prise faite au socle après
// sa réapparition est la prise du socle, et les ramassages suivants de la même famille dans le
// même cycle sont ceux d'une autre arme — le plus souvent celle que le preneur lâche en mourant
// près du socle, reprise au sol. C'est l'utilisateur qui la pose (2026-10-09) : « deux prises sur
// socle dans ce laps de temps me paraît impossible. Au sol oui, mais pas sur socle. »
//
// # CE QUE LE FILM ÉCRIT, ET CE QUI RESTE DÉCIDÉ
//
// Le film écrit chaque ramassage (`biped_pickup`, daté, ramasseur porté), l'apparition de chaque
// arme sur le socle (record de création) et la position du ramasseur à l'instant du ramassage. Il
// n'écrit pas de quel objet vient un ramassage : c'est ce que la règle décide, et seulement quand
// la lecture (un seul ramassage dans la fenêtre, revendiqué par une seule occupation) ne l'a pas
// fait. Trois conditions, toutes sur des faits lus :
//
//   - le CYCLE : un ramassage appartient à la dernière apparition du socle qui le précède. Un
//     ramassage fait après l'apparition suivante de l'arme n'est plus celui de cette occupation ;
//   - AU SOCLE : le ramasseur se tient à moins de [originDropMaxDist] de la position du socle (le
//     rayon où un joueur prend un objet). Une prise faite ailleurs sur la carte est celle d'une
//     autre arme de la même famille ;
//   - LA PREMIÈRE : parmi les ramassages du cycle faits au socle, le plus tôt. Un ramassage plus
//     tôt dont le ramasseur n'est pas localisé interdit de conclure : l'occupation s'abstient.
//
// Un ramassage qu'une lecture a daté n'est jamais repris — quand c'est lui, le premier au socle du
// cycle, l'occupation s'abstient au lieu de prendre une reprise au sol plus tardive —, et un
// ramassage que le repli désigne pour deux occupations n'en date aucune : la règle « un ramassage
// date au plus une occupation » (pad_pickup_dating.go) tient des deux côtés.

import "math"

// localiserRamasseur rend la position du ramasseur d'un ramassage natif publié, à l'instant du
// ramassage, quand le film la porte.
type localiserRamasseur func(Pickup) ([3]float32, bool)

// entreesDuCycle : ce que le repli consomme, déjà lu ailleurs.
type entreesDuCycle struct {
	pads      []WeaponPad
	picks     []PadPickup
	pickups   []Pickup
	localiser localiserRamasseur
}

// premieresPrisesDuCycle applique le repli aux occupations que la lecture n'a pas datées et pose
// dans `retenu` le ramassage qu'il leur désigne. Rend le nombre d'occupations ainsi datées.
func premieresPrisesDuCycle(in entreesDuCycle, fenetres [][]int, retenu []int) int {
	if in.localiser == nil {
		return 0
	}
	pris := map[int]bool{}
	for _, c := range retenu {
		if c >= 0 {
			pris[c] = true
		}
	}
	designe := make([]int, len(in.picks))
	revendications := map[int]int{}
	for i := range in.picks {
		designe[i] = candidatNonDate
		if retenu[i] >= 0 || len(fenetres[i]) == 0 {
			continue
		}
		designe[i] = premiereAuSocle(in, i, fenetres[i], pris)
		if designe[i] >= 0 {
			revendications[designe[i]]++
		}
	}
	n := 0
	for i, c := range designe {
		if c >= 0 && revendications[c] == 1 {
			retenu[i] = c
			n++
		}
	}
	return n
}

// premiereAuSocle rend le premier ramassage du cycle de l'occupation `i` fait au socle, parmi ses
// candidats triés par instant ; [candidatNonDate] quand le cycle n'est pas borné, qu'aucun n'est
// au socle, qu'un candidat plus tôt n'est pas localisé, ou que le premier ramassage au socle est
// déjà daté par une lecture : il est alors la prise du cycle, les suivants sont des reprises au sol,
// et un ramassage qu'une lecture a daté n'est jamais repris.
func premiereAuSocle(in entreesDuCycle, i int, candidats []int, pris map[int]bool) int {
	k := in.picks[i]
	pad := in.pads[k.Pad]
	fin, ok := finDuCycle(pad, k)
	if !ok {
		return candidatNonDate
	}
	socle := [3]float32{pad.X, pad.Y, pad.Z}
	for _, j := range candidats {
		if in.pickups[j].tUS >= fin {
			break // ramassage d'une réapparition suivante
		}
		pos, ok := in.localiser(in.pickups[j])
		if !ok {
			return candidatNonDate
		}
		if dist3(pos, socle) >= originDropMaxDist {
			continue // prise d'une autre arme de la famille, ailleurs sur la carte
		}
		if pris[j] {
			return candidatNonDate
		}
		return j
	}
	return candidatNonDate
}

// finDuCycle rend l'instant, en microsecondes, de l'apparition du socle qui SUIT celle de l'occupation
// `k` (l'infini à défaut), et faux quand l'apparition de `k` ne se retrouve pas sans ambiguïté
// dans `pad.Presence` — fenêtre absente ou partagée par deux apparitions.
func finDuCycle(pad WeaponPad, k PadPickup) (uint64, bool) {
	at := -1
	for p, pr := range pad.Presence {
		if pr.tLowUS != k.tLowUS || pr.tHighUS != k.tHighUS {
			continue
		}
		if at >= 0 {
			return 0, false
		}
		at = p
	}
	if at < 0 {
		return 0, false
	}
	t0 := pad.Presence[at].tUS
	fin := uint64(math.MaxUint64)
	for _, pr := range pad.Presence {
		if pr.tUS > t0 && pr.tUS < fin {
			fin = pr.tUS
		}
	}
	return fin, true
}

// localiserParLeCanalNatif rend le localisateur des ramasseurs : la position du ramasseur à l'instant EXACT du
// ramassage ([pickupOriginJudge.positionDe]). Le ramassage publié porte son horodatage ; un instant antérieur à
// l'origine n'est jamais publié.
func localiserParLeCanalNatif(
	position func(slot uint32, tsUS uint64) (x, y, z float32, ok bool),
) localiserRamasseur {
	return func(p Pickup) ([3]float32, bool) {
		x, y, z, ok := position(p.Slot, p.tUS)
		return [3]float32{x, y, z}, ok
	}
}
