package replay

// projectiles_colles_calque.go : LES COLLAGES QUE NUL VOL NE PORTE, PUBLIES (`attachments[]`).
//
// Un projectile qui se colle n'est pas toujours un vol publie : il peut naitre colle, a bout portant, ou voler trop peu pour que sa
// trajectoire se dessine (moins de trois echantillons, moins de deux points de grille). Sa lecture de collage n'a alors aucun
// `projectiles[]` pour la porter. Elle se publie ici, une fois : chaque collage est dans `projectiles[].attach` OU dans `attachments[]`,
// jamais dans les deux (`coverage.projectiles.attach` dit combien dans chacun).
//
// CE QUE LE FILM DONNE, ET QUI SE PUBLIE : l'instant, le parent (resolu en joueur ou en vehicule par les pistes deja publiees, comme
// `projectiles[].attach`), le noeud et la pose BRUTE ; le TYPE de l'objet, mot de 32 bits de sa creation (le tag `proj`, en hexadecimal) ; son
// CREATEUR, la reference de categorie 1 de l'etat de creation du projectile, resolue en piste comme l'auteur d'un degat (`damage[].by`,
// `damage[].attacker`). Le createur est un fait du film, pas une deduction : il se lit dans la creation du projectile, et ce qu'il nomme se
// recoupe avec les tirs et les lancers (le tireur du tir de meme instant, le lanceur de la grenade qui precede).
//
// CE QUI NE SE PUBLIE PAS : le tir ou l'arme du collage. Le film ne relie pas un projectile a un tir par un champ : il ne le fait que par la
// coincidence des instants du tireur, que `projectiles[].shot` fait deja pour les vols publies ; ici, aucune coincidence n'est declaree
// fait. Le nom de l'objet vient de son type, que l'hote nomme.

import (
	"fmt"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Attachment est un collage qu'aucun vol publie ne porte : ou un projectile se tient, sur quoi, de quel type, et qui l'a cree.
type Attachment struct {
	// T est la frame de la lecture, sur le meme axe que Point.T.
	T int `json:"t"`
	// ParentSlot est le slot de l'objet sur lequel le projectile est colle, comme `projectiles[].attach.parentSlot`.
	ParentSlot uint32 `json:"parentSlot"`
	// Target dit ce que ce parent est dans le document a la frame T : `player` (la piste de `tracks[]` qui tient le slot) ou `vehicle` (le
	// vehicule de `vehicles[]` de meme slot et de meme vie). Absent : ni l'un ni l'autre (une arme au sol, un objet, un equipement, un vehicule
	// que `vehicles[]` ne publie pas sous ce slot).
	Target string `json:"target,omitempty"`
	// Victim est le joueur cible, en decimal comme `Track.XUID`, quand le parent est une piste de joueur humain ; absent pour un bot.
	Victim string `json:"victim,omitempty"`
	// Node est le noeud du parent, brut ; Pose, la position du projectile dans le repere de ce noeud : trois valeurs de 16 bits BRUTES, dans
	// l'ordre du film, sans plage (cf. [ProjectileAttach]).
	Node uint32    `json:"node"`
	Pose [3]uint32 `json:"pose"`
	// Tag est le type du projectile : le mot de 32 bits de sa creation (le tag `proj`), huit chiffres hexadecimaux. Absent quand la creation
	// n'est pas lue.
	Tag string `json:"tag,omitempty"`
	// By est le slot de la piste du createur a la frame T, Attacker son xuid (absent pour un bot), comme `damage[].by` et `damage[].attacker`.
	// Absents quand la creation ne nomme pas de createur ou que son unite ne tient aucune piste a cette frame.
	By       uint32 `json:"by,omitempty"`
	Attacker string `json:"attacker,omitempty"`
	// Born : le projectile nait colle (a bout portant). Absent : un projectile qui avait vole.
	Born bool `json:"born,omitempty"`
}

// cadreDesCollages est ce que la resolution d'un collage lit du document : les pistes de joueur et les vehicules.
type cadreDesCollages struct {
	pistes    []Track
	vehicules []VehicleTrack
}

// nomCibleJoueur est le mot publie d'une cible resolue en joueur.
const nomCibleJoueur = "player"

// nomsDesCibles donne le mot publie de chaque cible resolue ; la cible non resolue n'a pas de mot.
var nomsDesCibles = map[cibleDuCollage]string{cibleJoueur: nomCibleJoueur, cibleVehicule: "vehicle"}

// unitePortee rend le xuid du joueur dont la piste tient le slot `slot` a la frame `t`, et si une piste le tient : la resolution des degats
// ([resolveurDeTouches.victime]).
func unitePortee(slot uint32, t int, pistes []Track) (xuid string, piste bool) {
	if unite := slot - decalageDesSlotsDUnite; slot >= decalageDesSlotsDUnite && unite <= 0xFFFF {
		return (resolveurDeTouches{tracks: pistes}).victime(uint16(unite), t)
	}
	return "", false
}

// attachementDeLaLecture rend le collage publie de la lecture `l` a la frame `t`, la cible de son parent, et dit si son createur se resout.
func attachementDeLaLecture(l types.ProjectileAttachRead, t int, c cadreDesCollages) (Attachment, cibleDuCollage) {
	cible := cibleDuParent(l.Parent, t, c.pistes, c.vehicules)
	at := Attachment{T: t, ParentSlot: l.Parent.Slot, Target: nomsDesCibles[cible], Node: l.Node, Pose: l.Pose, Born: l.Created}
	if cible == cibleJoueur {
		at.Victim, _ = unitePortee(l.Parent.Slot, t, c.pistes)
	}
	if l.HasTag {
		at.Tag = fmt.Sprintf("%08x", l.Tag)
	}
	if l.HasCreator {
		if xuid, piste := unitePortee(l.Creator.Slot, t, c.pistes); piste {
			at.By, at.Attacker = l.Creator.Slot, xuid
		}
	}
	return at, cible
}

// publierLesCollagesSansVol rend les collages des lectures que ni la frame (`prises`, dont `frames`) ni un vol n'ont prises, dans l'ordre du
// flux, et compte dans `cov` leurs cibles, leurs types et leurs createurs. Nil sans collage.
func publierLesCollagesSansVol(lectures []types.ProjectileAttachRead, frames []int, prises []bool, c cadreDesCollages,
	cov *ProjectileAttachCoverage) []Attachment {
	var out []Attachment
	for i, l := range lectures {
		if prises[i] {
			continue
		}
		at, cible := attachementDeLaLecture(l, frames[i], c)
		cov.Listed++
		cov.compterLaCible(cible)
		cov.Tagged += unSiVrai(l.HasTag)
		switch {
		case at.By != 0:
			cov.Authored++
		case l.HasCreator:
			cov.AuthorNoTrack++
		default:
			cov.AuthorNone++
		}
		out = append(out, at)
	}
	return out
}
