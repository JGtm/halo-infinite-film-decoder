package replay

// ground_weapon_pad_candidates.go — LES CANDIDATS DE SOCLE : ce que la règle « un socle est une
// récurrence » met de côté.
//
// CE QUE C'EST. Une grappe d'apparitions au repos qui n'a qu'UNE apparition ne récurre pas, donc
// n'est pas un socle ([gwPadsKeep]). Elle n'est pourtant pas du bruit : un objet que la carte a posé
// au repos et que personne n'a pris ne réapparaît jamais, et sa grappe n'a qu'une apparition. Le
// document la publie sous `padCandidates`, dans la forme d'un socle (`WeaponPad`) dont `spawns` et
// `presence` n'ont qu'une ligne et dont `cycle` est absent.
//
// CE QUE LE CALQUE NE DIT PAS. Un candidat n'est PAS un socle : rien dans le film ne prouve qu'une
// seconde arme viendra s'y poser. Seul un lecteur qui connaît la carte (un point d'apparition ou un
// socle de la variante à moins d'un mètre) peut le promouvoir ; le décodeur ne lit aucun fichier de
// carte et ne promeut rien. Aucune occupation n'est publiée pour un candidat (`padPickups` ne les
// porte pas) : la présence bornée suffit à dire dans quel état il est.
//
// POURQUOI UN CALQUE À PART ET PAS UN DRAPEAU SUR `weaponPads`. Les consommateurs de `weaponPads`
// lisent chaque ligne comme un socle prouvé par la récurrence et indexent `padPickups[].pad` sur
// ce tableau : y mêler des candidats changerait le sens d'une ligne qui existe déjà.
//
// UN FAIT PUBLIÉ UNE FOIS. Les armes au sol que publie `groundWeapons` sont celles qui ont BOUGÉ
// (document_ground_weapon_items.go) ; une arme apparue au repos est écartée de ce calque parce
// qu'elle appartient aux socles. Les candidats comblent le seul cas que ni l'un ni l'autre ne
// publiait : l'objet au repos vu une seule fois.

// gwPadCandidates rend, pour les grappes `aside` (celles de [gwPadsSetAside]), un candidat de socle
// par grappe. `src` donne l'index d'avant filtre de chaque grappe, celui sur lequel `members` est
// indexé. Les compteurs de la couverture ne bougent pas : un candidat se construit sur la même
// fonction que le socle ([gwBuildPad]), qui compte ses occupations dans la couverture qu'on lui
// passe, et celle-ci est jetée.
func gwPadCandidates(
	aside []gwPadCluster, src []int, objs []gwPickupObject, members map[int][]int, clock replayClock,
) []WeaponPad {
	out := make([]WeaponPad, 0, len(aside))
	for i, c := range aside {
		cand, _ := gwBuildPad(c, objs, members[src[i]], clock, &GroundWeaponCoverage{})
		out = append(out, cand)
	}
	return out
}
