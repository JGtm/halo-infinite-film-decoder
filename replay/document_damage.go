package replay

// document_damage.go — LES DEGATS EN POINTS, PAR JOUEUR, DANS LE TEMPS ET PAR ARME : UNE ESTIMATION.
//
// Chaque message `damage_aftermath` qui touche un joueur ou un bot devient une ligne `damage[]` : quand,
// qui a causé le dégât, qui l'a pris, par quel effet et avec quelle arme, combien de points du bouclier et du corps
// ([calculerLesPointsDeDegats] : la table des effets, et la vie réelle de l'unité blessée relue dans ses composants).
// Les totaux par joueur sont dans `shotStats[]` (`damageDealt`, `damageTaken`), sous le MÊME verdict de suffisance que les
// touches : faux, le consommateur n'affiche aucun chiffre de dégât.
//
// LES LIGNES A ZERO POINT SE RECONNAISSENT A LEUR VALEUR : `shield` et `body` valent zéro. Le film écrit des messages qui ne retirent rien
// (un dégât qu'on se fait à soi-même, un doublon d'un message à points dans la même trame, une victime déjà morte) ; ils sont publiés
// pour que le client voie qu'un effet a touché une unité, mais ils ne sont NI un dégât NI une touche : aucun total de `shotStats[]`, aucun
// compte de `coverage.hits` (`damages`, `damagesMeasured`, `damagesOffUnit`, `selfDamages`) ne les voit, ils ont leurs propres compteurs
// (`damagesZero` et suivants). Tout lecteur qui COMPTE des dégâts à partir de `damage[]` ignore donc les lignes dont les deux valeurs sont nulles.
//
// `slot` ET `by` DESIGNENT LES PISTES. `slot` est le slot de la piste de la victime (celle de l'unité blessée), `by` celui de la piste de
// l'unité responsable quand elle en a une à cette frame : ils tiennent là où `victim` et `attacker` manquent (un bot n'a pas de xuid), et
// désignent la piste exacte quand un même joueur en a plusieurs. `by` est absent quand le message ne nomme pas de responsable ou que son
// unité n'a pas de piste ; il vaut `slot` sur un dégât qu'une unité s'inflige. Ce sont des lectures des références du message, pas des déductions.
//
// LES TOTAUX COUVRENT TOUT LE FILM, LES LIGNES SEULEMENT L'AXE. Un message dont l'instant tombe avant la frame 0 ou après la dernière
// compte dans `damageDealt` et `damageTaken` sans avoir de ligne dans `damage[]` : la somme des lignes d'un joueur peut donc être
// inférieure à son total.
//
// UNE UNITE SANS PISTE N'EST PAS UN JOUEUR. Un message dont l'unité blessée n'a pas de piste à cette frame (un véhicule, un
// objet) n'est ni publié ni compté : il n'est pas du dégât fait à un joueur. Une victime dont la piste n'a pas de xuid est un
// bot : la ligne n'a pas de `victim`, et le dégât compte pour l'auteur mais pour personne côté subi. Un auteur que le roster
// ne résout pas (le décor, un bot) n'a pas de `attacker`. Un joueur qui se blesse lui-même ne se compte pas en infligé.

import (
	"cmp"
	"math"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Damage est UNE ligne de dégât : quand, de qui à qui, par quel effet, avec quelle arme, pour combien de points. `damage[]` ne porte que
// les messages dont l'instant tient dans l'axe de frames ; les totaux de `shotStats[]` comptent tout le film.
type Damage struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Attacker est l'auteur, Victim la victime, en décimal — même forme que Track.XUID. Absents : le décor ou un bot
	// pour l'auteur, un bot pour la victime.
	Attacker string `json:"attacker,omitempty"`
	Victim   string `json:"victim,omitempty"`
	// Effect est l'identifiant de l'effet de dégât, Weapon l'arme du dernier tir de l'auteur dans les deux secondes (cf. [Hit]).
	Effect string `json:"effect"`
	Weapon string `json:"w,omitempty"`
	// Shield et Body sont les points retirés au bouclier et au corps, au dixième près.
	Shield float64 `json:"shield"`
	Body   float64 `json:"body"`
	// Slot est le slot de la piste de la victime, By celui de la piste du responsable (cf. l'en-tête). Un slot de piste vaut au moins
	// [decalageDesSlotsDUnite] : `omitempty` ne peut pas effacer un slot réel.
	Slot uint32 `json:"slot,omitempty"`
	By   uint32 `json:"by,omitempty"`
	// Measured : les points viennent de la vie lue autour du coup (bons à deux points de bouclier près), pas de la table.
	Measured bool `json:"measured,omitempty"`
}

// messageEtPoints est un message de dégât avec ce qu'il retire.
type messageEtPoints struct {
	d grammar.DegatLu
	p PointsDeDegat
}

// messagesParInstant joint chaque message de `in` à ses points et les range par instant (l'ordre du film à instant égal).
func messagesParInstant(in HitInput) []messageEtPoints {
	out := make([]messageEtPoints, len(in.Messages))
	for i, d := range in.Messages {
		out[i].d = d
		if i < len(in.Points) {
			out[i].p = in.Points[i]
		}
	}
	slices.SortStableFunc(out, func(a, b messageEtPoints) int { return cmp.Compare(a.d.TS, b.d.TS) })
	return out
}

// dixiemes rend un nombre de points en dixièmes comme un flottant au dixième près.
func dixiemes(v uint16) float64 { return math.Round(float64(v)) / 10 }

// poserLesDegats pose les lignes de dégât, ajoute les totaux au bilan de chaque joueur et complète la couverture.
//
// Un message qui retire des points suit la voie historique : ligne, totaux, compteurs `damages`, `damagesMeasured`, `damagesOffUnit` et
// `selfDamages`. Un message à zéro point ne fait que sa ligne (et ses compteurs `damagesZero*`) : il n'entre ni dans le bilan ni dans les
// compteurs historiques, qui comptent des dégâts.
func poserLesDegats(msgs []messageEtPoints, r resolveurDeTouches, armes armesDesTirs, bilan bilanDesTirs, cov *HitCoverage) []Damage {
	var out []Damage
	for _, m := range msgs {
		d := m.d
		if !d.AVictime {
			continue
		}
		nul := m.p.Bouclier+m.p.Corps == 0
		t := r.frameBornee(d.TS)
		victime, piste := r.victime(d.Victime, t)
		if !piste {
			if nul {
				cov.DamagesZeroOffUnit++
			} else {
				cov.DamagesOffUnit++
			}
			continue
		}
		attaquant := auteurDuDegat(d, r, t)
		if !nul {
			compterLeDegat(m.p, attaquant, victime, bilan, cov)
		}
		ligne, ok := ligneDeDegat(m, attaquant, victime, r, armes)
		if !ok {
			if nul {
				cov.DamagesZeroOutOfAxis++
			}
			continue // hors de l'axe : un message à points compte aux totaux, sans ligne
		}
		out = append(out, ligne)
		compterLaLigne(ligne, nul, cov)
	}
	slices.SortStableFunc(out, func(a, b Damage) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Attacker, b.Attacker), cmp.Compare(a.Victim, b.Victim), cmp.Compare(a.Effect, b.Effect))
	})
	return out
}

// auteurDuDegat rend le xuid de l'auteur du message `d` à la frame `t`, vide pour le décor, un bot ou un auteur que le roster ne résout pas.
func auteurDuDegat(d grammar.DegatLu, r resolveurDeTouches, t int) string {
	if d.Joueur < 0 {
		return ""
	}
	if x, res := r.auteurDuMessage(d, t); res == destinataireResolu {
		return x
	}
	return ""
}

// compterLaLigne compte une ligne posée : `damagesZero` pour une ligne à zéro point, `damages` et `damagesMeasured` pour les autres.
func compterLaLigne(l Damage, nul bool, cov *HitCoverage) {
	switch {
	case nul:
		cov.DamagesZero++
	case l.Measured:
		cov.Damages++
		cov.DamagesMeasured++
	default:
		cov.Damages++
	}
}

// compterLeDegat ajoute les points d'un message qui en retire au bilan : infligé à l'auteur résolu, subi à la victime humaine ; le dégât qu'un
// joueur se fait à lui-même ne se compte pas en infligé et se compte à part.
func compterLeDegat(p PointsDeDegat, attaquant, victime string, bilan bilanDesTirs, cov *HitCoverage) {
	points := dixiemes(p.Bouclier) + dixiemes(p.Corps)
	if attaquant == victime && attaquant != "" {
		cov.SelfDamages++
	} else if attaquant != "" {
		bilan.de(attaquant).DamageDealt += points
	}
	if victime != "" {
		bilan.de(victime).DamageTaken += points
	}
}

// ligneDeDegat pose le message `m` sur l'axe de frames, ou faux quand son instant n'y tient pas. L'unité blessée a une piste (l'appelant l'a
// vérifié) : `slot` est son slot ; `by` celui de l'unité responsable quand le message en nomme une qui a une piste à cette frame.
func ligneDeDegat(m messageEtPoints, attaquant, victime string, r resolveurDeTouches, armes armesDesTirs) (Damage, bool) {
	t, refus := frameDeLAction(m.d.TS, r.axe)
	if refus != refusAucun {
		return Damage{}, false
	}
	l := Damage{T: t, Attacker: attaquant, Victim: victime, Effect: formaterLEffet(m.d.Tag),
		Shield: dixiemes(m.p.Bouclier), Body: dixiemes(m.p.Corps), Measured: m.p.Mesure,
		Slot: uint32(m.d.Victime) + decalageDesSlotsDUnite}
	if m.d.AResponsable {
		if _, piste := r.victime(m.d.Responsable, t); piste {
			l.By = uint32(m.d.Responsable) + decalageDesSlotsDUnite
		}
	}
	if m.d.Joueur >= 0 {
		l.Weapon = armes.derniereArme(attaquant, int(m.d.Joueur), m.d.TS)
	}
	return l, true
}
