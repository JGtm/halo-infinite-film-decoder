package replay

// zone_lettres_controle.go — LA LETTRE LUE SUR LE MARQUEUR D'ECRAN, COMME CONTROLE DU RANG PUBLIE.
//
// # CE QUE LE FILM DIT, ET CE QU'IL NE DIT PAS
//
// Chaque zone de Bases porte, a l'ecran, un marqueur dont le groupe d'etats « control_point » a pour
// etat la lettre de la zone (A, B, C). Le film replique donc la lettre. Il ne replique PAS le lien du
// marqueur a la zone : les references de proprietes du marqueur sont vides. Le seul lien lisible est
// une CONCORDANCE : le marqueur change de proprietaire aux memes instants que la zone qu'il designe.
//
// # POURQUOI C'EST UN CONTROLE ET JAMAIS UNE SOURCE
//
// Une concordance qui decide d'une valeur publiee decide d'une sortie sur une correlation (regle R-2
// du plan). Ce fichier ne change donc JAMAIS `ZoneState.LetterRank` : il compte, dans
// [zoneLettersControl], combien de lettres le film a lues et combien s'accordent avec le rang que le
// rejeu a publie. Un desaccord se voit au compte ; il ne corrige rien.
//
// # LA CONCORDANCE
//
// Les horloges du marqueur et du rejeu different d'un decalage unique par film : il se lit, il ne se
// suppose pas. Le decalage retenu est le MODE des ecarts entre un changement de proprietaire d'un
// marqueur et un changement de proprietaire d'une zone, a equipe egale, classes par pas de
// [zoneLetterToleranceMS] ; a effectif egal le plus petit ecart en valeur absolue (ordre total, donc
// meme sortie a un fil et a plusieurs). Un changement de marqueur concorde avec une zone quand un
// changement de cette zone, de meme equipe, tombe a moins de [zoneLetterToleranceMS] de lui, une fois
// le decalage retranche.
//
// Un marqueur n'est rattache a une zone que si cette zone est SON seul meilleur score (strictement
// superieur a tous les autres, et non nul) ET qu'aucun autre marqueur n'a la meme zone pour meilleure :
// la correspondance exigee est une bijection. Un marqueur sans rattachement univoque est lu (sa lettre
// existe) mais n'est pas compte en accord, et `Unmatched` le dit.

import (
	"cmp"
	"slices"
)

// zoneLetterToleranceMS est la tolerance de concordance entre le changement de proprietaire d'un
// marqueur et celui d'une zone, apres decalage d'horloge. C'est celle de la mesure du 2026-11-22
// (1,5 s), sous laquelle les 231 zones des 77 films de Bases s'accordent avec le rang publie.
const zoneLetterToleranceMS = 1500

// ZoneLetterMarker est un marqueur d'ecran porteur d'une lettre, tel que le film le replique.
type ZoneLetterMarker struct {
	// Letter est le rang de la lettre lue (0 = A, 1 = B, 2 = C). Un rang negatif dit que le marqueur
	// n'a pas de lettre lisible : il ne se compte pas.
	Letter int
	// Changes sont les changements de proprietaire du marqueur, dans l'ordre du temps.
	Changes []ZoneLetterChange
}

// ZoneLetterChange est un changement de proprietaire : l'instant (ms, horloge du marqueur) et le camp
// qui tient desormais l'objet (indice d'equipe, jamais le neutre).
type ZoneLetterChange struct {
	TimeMS int
	Team   int
}

// zoneLettersControl est le resultat du controle, pour le film entier.
type zoneLettersControl struct {
	// Read : les lettres distinctes lues sur les marqueurs.
	Read int
	// Agreed : les marqueurs rattaches a une zone dont le rang publie est leur lettre.
	Agreed int
	// Unmatched : les marqueurs lus sans rattachement univoque (aucune concordance, egalite, ou zone
	// reclamee par deux marqueurs).
	Unmatched int
}

// zoneLettersControlOf confronte les lettres lues aux rangs publies. `zones` sont les etats publies
// (leurs intervalles donnent les changements de proprietaire), `frameMS` la duree d'une image du
// rejeu : les intervalles sont en images, les marqueurs en millisecondes.
func zoneLettersControlOf(markers []ZoneLetterMarker, zones []ZoneState, frameMS int) zoneLettersControl {
	var out zoneLettersControl
	lus := markersWithLetter(markers)
	out.Read = distinctLetters(lus)
	if len(lus) == 0 || len(zones) == 0 || frameMS <= 0 {
		out.Unmatched = len(lus)
		return out
	}
	changes := zoneChangesOf(zones, frameMS)
	offset := zoneLetterClockOffset(lus, changes)
	best, unique := zoneLetterBestZones(lus, changes, offset)
	claimed := map[int]int{}
	for _, z := range best {
		if z >= 0 {
			claimed[z]++
		}
	}
	for i, m := range lus {
		z := best[i]
		if !unique[i] || z < 0 || claimed[z] != 1 {
			out.Unmatched++
			continue
		}
		if r := zones[z].LetterRank; r != nil && *r == m.Letter {
			out.Agreed++
		}
	}
	return out
}

// markersWithLetter rend les marqueurs dont la lettre est lisible, dans un ordre total (lettre, puis
// premier changement) : la sortie ne depend pas de l'ordre d'arrivee.
func markersWithLetter(markers []ZoneLetterMarker) []ZoneLetterMarker {
	var out []ZoneLetterMarker
	for _, m := range markers {
		if m.Letter >= 0 {
			out = append(out, m)
		}
	}
	slices.SortStableFunc(out, func(a, b ZoneLetterMarker) int {
		return cmp.Or(cmp.Compare(a.Letter, b.Letter), cmp.Compare(firstChangeMS(a), firstChangeMS(b)))
	})
	return out
}

func firstChangeMS(m ZoneLetterMarker) int {
	if len(m.Changes) == 0 {
		return 0
	}
	return m.Changes[0].TimeMS
}

func distinctLetters(ms []ZoneLetterMarker) int {
	seen := map[int]bool{}
	for _, m := range ms {
		seen[m.Letter] = true
	}
	return len(seen)
}

// zoneChangesOf rend, par zone (dans l'ordre de `zones`), les changements de proprietaire publies :
// le debut de chaque intervalle tenu par un camp, converti en millisecondes.
func zoneChangesOf(zones []ZoneState, frameMS int) [][]ZoneLetterChange {
	out := make([][]ZoneLetterChange, len(zones))
	for i, z := range zones {
		for _, s := range z.Spans {
			if s.Owner == nil {
				continue
			}
			out[i] = append(out[i], ZoneLetterChange{TimeMS: s.T0 * frameMS, Team: *s.Owner})
		}
	}
	return out
}

// zoneLetterClockOffset rend le mode des ecarts (marqueur moins zone) entre changements de meme equipe,
// classes par pas de [zoneLetterToleranceMS]. Aucun couple : 0.
func zoneLetterClockOffset(markers []ZoneLetterMarker, zones [][]ZoneLetterChange) int {
	bins := map[int]int{}
	for _, m := range markers {
		for _, c := range m.Changes {
			for _, zc := range zones {
				for _, y := range zc {
					if y.Team == c.Team {
						bins[floorDiv(c.TimeMS-y.TimeMS, zoneLetterToleranceMS)]++
					}
				}
			}
		}
	}
	keys := make([]int, 0, len(bins))
	for k := range bins {
		keys = append(keys, k)
	}
	// Effectif decroissant, puis plus petit |ecart|, puis ecart : ordre total.
	slices.SortFunc(keys, func(a, b int) int {
		return cmp.Or(cmp.Compare(bins[b], bins[a]), cmp.Compare(abs(a), abs(b)), cmp.Compare(a, b))
	})
	if len(keys) == 0 {
		return 0
	}
	return keys[0]*zoneLetterToleranceMS + zoneLetterToleranceMS/2
}

// zoneLetterBestZones rend, par marqueur, la zone de meilleur score et si ce meilleur score est unique
// et non nul ; -1 quand aucune zone ne concorde.
func zoneLetterBestZones(markers []ZoneLetterMarker, zones [][]ZoneLetterChange, offset int) (best []int, unique []bool) {
	best, unique = make([]int, len(markers)), make([]bool, len(markers))
	for i, m := range markers {
		best[i] = -1
		top, ties := 0, 0
		for z, zc := range zones {
			s := zoneLetterScore(m.Changes, zc, offset)
			switch {
			case s > top:
				best[i], top, ties = z, s, 1
			case s == top && s > 0:
				ties++
			}
		}
		unique[i] = top > 0 && ties == 1
	}
	return best, unique
}

// zoneLetterScore compte les changements du marqueur qui concordent avec un changement de la zone, de
// meme equipe, a la tolerance pres une fois le decalage retranche. Un changement de zone sert une
// fois au plus.
func zoneLetterScore(marker, zone []ZoneLetterChange, offset int) int {
	used := make([]bool, len(zone))
	n := 0
	for _, c := range marker {
		for j, y := range zone {
			if used[j] || y.Team != c.Team || abs(c.TimeMS-offset-y.TimeMS) > zoneLetterToleranceMS {
				continue
			}
			used[j] = true
			n++
			break
		}
	}
	return n
}

// floorDiv est la division entiere vers le bas : le classement par pas doit etre le meme de part et
// d'autre de zero.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
