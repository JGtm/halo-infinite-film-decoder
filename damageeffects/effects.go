// Package damageeffects dit à quoi appartient un effet de dégât du jeu (une arme, un corps à corps, une grenade, un
// véhicule, le décor...) et combien de points il retire à un Spartan.
//
// Un message `damage_aftermath` du film nomme l'effet qui touche une unité par l'identifiant de son tag (`jpt!`).
// Seuls les effets d'arme font une touche ([EstUneTouche]) ; les montants disent ce qu'un coup direct d'un effet à
// montant fixe retire du bouclier et du corps ([Effet.Repartir]).
//
// ORIGINE : `data/effects.tsv`, reprise de libfilmshell (MIT, commit ad17521) avec son en-tête de crédit ; la méthode de
// répartition d'un coup est celle de `hit_damage.rs` du même dépôt, écrite ici dans les termes de ce paquet. La
// licence est à côté (`data/LICENSE-libfilmshell`) et `NOTICE.md` la cite.
package damageeffects

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed data/effects.tsv
var brut string

// Cause est ce à quoi un effet de dégât appartient, dans les mots du fichier.
type Cause string

// Les causes que le fichier écrit.
const (
	// CauseArme : un projectile d'arme (ou un projectile sans arme au catalogue).
	CauseArme Cause = "weapon"
	// CauseCorpsACorps : un coup de mêlée, épée ou marteau compris.
	CauseCorpsACorps Cause = "melee"
	// CauseGrenade : l'effet d'une grenade.
	CauseGrenade Cause = "grenade"
	// CauseEquipement : l'effet d'un équipement.
	CauseEquipement Cause = "equipment"
	// CauseVehicule : l'effet propre d'un véhicule.
	CauseVehicule Cause = "vehicle"
	// CauseCollision : un modèle physique : une chose en a heurté une autre.
	CauseCollision Cause = "collision"
	// CauseDecor : les globaux du jeu : une chute, un volume mortel.
	CauseDecor Cause = "world"
	// CauseExplosion : un modèle qui explose.
	CauseExplosion Cause = "explosion"
	// CauseInconnue : le catalogue ne l'attribue à rien.
	CauseInconnue Cause = "unknown"
)

// Les drapeaux d'un effet, tels que le tag les range.
const (
	drapeauTirALaTete       = 1 << 1
	drapeauIgnoreLeBouclier = 1 << 9
	drapeauSansDebordement  = 1 << 23
)

// Effet est un effet de dégât du catalogue.
type Effet struct {
	Cause Cause
	// APoints dit que l'effet retire des points : faux pour un effet que le jeu connaît sans lui en donner.
	APoints bool
	// Bas est la borne basse, Haut la plage de la borne haute, en points ; CoeffBouclier et CoeffCorps les
	// multiplicateurs du bouclier et du corps du Spartan ; Drapeaux ceux du tag ; Rayon celui de la zone, en unités
	// du moteur (nul pour un coup direct).
	Bas                       float64
	Haut                      [2]float64
	CoeffBouclier, CoeffCorps float64
	Drapeaux                  uint64
	Rayon                     float64
}

// MontantFixe dit que l'effet retire le même montant quelle que soit la distance ou le tirage : un coup direct
// dont les deux bornes sont égales.
func (e Effet) MontantFixe() bool { return e.Rayon == 0 && e.Bas == e.Haut[0] && e.Bas == e.Haut[1] }

// PeutTirerALaTete dit que l'effet peut faire un tir à la tête.
func (e Effet) PeutTirerALaTete() bool { return e.Drapeaux&drapeauTirALaTete != 0 }

// Repartir rend ce qu'un coup brut de `degat` points retire à un Spartan qui garde `bouclier` points de bouclier et
// `corps` points de corps, dans cet ordre : ce que le bouclier encaisse, puis ce que le corps encaisse.
//
// Le bouclier voit le coup au multiplicateur du bouclier. S'il le tient en entier, le corps n'en voit rien ; sinon il
// s'épuise, et la fraction du coup qu'il n'a pas arrêtée passe au corps, au multiplicateur du corps, sauf pour un
// effet qui ne déborde pas (le corps n'en voit alors rien). Sans bouclier, ou pour un effet qui l'ignore, le coup
// entier passe au corps. Un coup ne retire jamais plus que ce qui reste.
//
// MÉTHODE reprise de libfilmshell (OpenSpartan, MIT, `src/game/hit_damage.rs`) : les trois drapeaux et les
// multiplicateurs sont ceux de la table ; la formulation par fraction du coup est celle de ce paquet.
func (e Effet) Repartir(degat, bouclier, corps float64) (surLeBouclier, surLeCorps float64) {
	passe := 1.0 // fraction du coup que le bouclier laisse passer
	if bouclier > 0 && e.Drapeaux&drapeauIgnoreLeBouclier == 0 {
		vu := degat * e.CoeffBouclier
		switch {
		case vu <= bouclier:
			surLeBouclier, passe = vu, 0
		case e.Drapeaux&drapeauSansDebordement != 0:
			surLeBouclier, passe = bouclier, 0
		default:
			surLeBouclier, passe = bouclier, 1-bouclier/vu
		}
	}
	return surLeBouclier, min(max(degat*passe*e.CoeffCorps, 0), max(corps, 0))
}

// Corps et bouclier maximaux d'un Spartan du multijoueur, en points.
const (
	CorpsMaximal    = 90.0
	BouclierMaximal = 140.0
)

// effets sont les effets par tag, chargés du fichier embarqué.
var effets = charger(brut)

// Lookup rend l'effet de dégât `tag`, ou faux quand la table ne le connaît pas.
func Lookup(tag uint32) (Effet, bool) {
	e, ok := effets[tag]
	return e, ok
}

// EstUneTouche dit que l'effet `tag` est un effet d'arme : un message de dégât qui le nomme est une touche.
func EstUneTouche(tag uint32) bool { return effets[tag].Cause == CauseArme }

// Taille rend le nombre d'effets de la table, celui des effets d'arme et celui des effets qui retirent des points.
func Taille() (effetsConnus, effetsDArme, effetsAPoints int) {
	for _, e := range effets {
		if e.Cause == CauseArme {
			effetsDArme++
		}
		if e.APoints {
			effetsAPoints++
		}
	}
	return len(effets), effetsDArme, effetsAPoints
}

const colonnes = 9

// charger lit la table : une ligne par effet, les lignes de commentaire (`#`) et les lignes vides sont
// ignorées. Une ligne mal formée est une erreur du fichier embarqué, qui ne se corrige que dans le dépôt : elle
// arrête le programme au chargement plutôt que de servir une table tronquée.
func charger(texte string) map[uint32]Effet {
	out := map[uint32]Effet{}
	for _, l := range strings.Split(texte, "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) != colonnes {
			panic("damageeffects : ligne mal formee : " + l)
		}
		tag, err := strconv.ParseUint(c[0], 16, 32)
		if err != nil || c[1] == "" {
			panic("damageeffects : ligne mal formee : " + l)
		}
		e := Effet{Cause: Cause(c[1]), APoints: c[2] != ""}
		if e.APoints {
			e = lireLesMontants(e, c, l)
		}
		out[uint32(tag)] = e
	}
	return out
}

// lireLesMontants complète `e` des sept colonnes de montants de la ligne `c`.
func lireLesMontants(e Effet, c []string, ligne string) Effet {
	var f [6]float64
	for i, s := range []string{c[2], c[3], c[4], c[5], c[6], c[8]} {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			panic("damageeffects : montant mal forme : " + ligne)
		}
		f[i] = v
	}
	drapeaux, err := strconv.ParseUint(c[7], 16, 32)
	if err != nil {
		panic("damageeffects : drapeaux mal formes : " + ligne)
	}
	e.Bas, e.Haut, e.CoeffBouclier, e.CoeffCorps, e.Rayon = f[0], [2]float64{f[1], f[2]}, f[3], f[4], f[5]
	e.Drapeaux = drapeaux
	return e
}
