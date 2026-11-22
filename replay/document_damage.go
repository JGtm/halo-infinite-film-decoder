package replay

// document_damage.go — LES DEGATS EN POINTS, PAR JOUEUR, DANS LE TEMPS ET PAR ARME : UNE ESTIMATION.
//
// Chaque message `damage_aftermath` qui retire des points à un joueur ou à un bot devient une ligne `damage[]` : quand,
// qui a causé le dégât, qui l'a pris, par quel effet et avec quelle arme, combien de points du bouclier et du corps
// ([calculerLesPointsDeDegats] : la table des effets, et la vie réelle de l'unité blessée relue dans ses composants).
// Les totaux par joueur sont dans `shotStats[]` (`damageDealt`, `damageTaken`), sous le MÊME verdict de suffisance que les
// touches : faux, le consommateur n'affiche aucun chiffre de dégât.
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
func poserLesDegats(msgs []messageEtPoints, r resolveurDeTouches, armes armesDesTirs, bilan bilanDesTirs, cov *HitCoverage) []Damage {
	var out []Damage
	for _, m := range msgs {
		d := m.d
		if !d.AVictime || m.p.Bouclier+m.p.Corps == 0 {
			continue
		}
		t := r.frameBornee(d.TS)
		victime, piste := r.victime(d.Victime, t)
		if !piste {
			cov.DamagesOffUnit++
			continue
		}
		attaquant := ""
		if d.Joueur >= 0 {
			if x, res := r.auteurDuMessage(d, t); res == destinataireResolu {
				attaquant = x
			}
		}
		points := dixiemes(m.p.Bouclier) + dixiemes(m.p.Corps)
		if attaquant == victime && attaquant != "" {
			cov.SelfDamages++
		} else if attaquant != "" {
			bilan.de(attaquant).DamageDealt += points
		}
		if victime != "" {
			bilan.de(victime).DamageTaken += points
		}
		if ligne, ok := ligneDeDegat(m, attaquant, victime, r, armes); ok {
			out = append(out, ligne)
			if ligne.Measured {
				cov.DamagesMeasured++
			}
		}
	}
	cov.Damages = len(out)
	slices.SortStableFunc(out, func(a, b Damage) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Attacker, b.Attacker), cmp.Compare(a.Victim, b.Victim), cmp.Compare(a.Effect, b.Effect))
	})
	return out
}

// ligneDeDegat pose le message `m` sur l'axe de frames, ou faux quand son instant n'y tient pas.
func ligneDeDegat(m messageEtPoints, attaquant, victime string, r resolveurDeTouches, armes armesDesTirs) (Damage, bool) {
	t, refus := frameDeLAction(m.d.TS, r.axe)
	if refus != refusAucun {
		return Damage{}, false
	}
	l := Damage{T: t, Attacker: attaquant, Victim: victime, Effect: formaterLEffet(m.d.Tag),
		Shield: dixiemes(m.p.Bouclier), Body: dixiemes(m.p.Corps), Measured: m.p.Mesure}
	if m.d.Joueur >= 0 {
		l.Weapon = armes.derniereArme(int(m.d.Joueur), m.d.TS)
	}
	return l, true
}
