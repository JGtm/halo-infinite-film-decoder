package replay

// degats_en_points.go — CE QUE CHAQUE MESSAGE DE DEGAT RETIRE A SON UNITE BLESSEE, EN POINTS.
//
// Un message `damage_aftermath` nomme un effet ; la table des effets ([damageeffects]) dit ce qu'un coup direct de cet
// effet retire, et la vie réelle de l'unité blessée ([registreDesVies], relue dans ses composants de vitalité) dit ce
// qu'elle perd vraiment. MÉTHODE reprise de libfilmshell (OpenSpartan, MIT, `src/game/hit_damage.rs`) : on confronte
// la table à la vie lue ; le code de ce fichier est celui de ce dépôt.
//
// UNE SCENE est l'ensemble des messages d'une même unité blessée (slot et génération) à un même instant de trame.
// Ils partagent une seule perte de vie observable : de la vie juste avant l'instant à la vie à l'instant. La scène se
// joue en deux passes :
//
//  1. chaque coup est classé. CALCULE : un effet à montant fixe, d'échelle pleine ; il retire ce que dit la table,
//     depuis ce qui reste après les coups calculés qui le précèdent dans le film. NUL : un effet que la table connaît
//     sans lui donner de points ; il ne retire rien. MESURE : tout autre coup (zone, plage de montants, échelle
//     réduite, effet inconnu) ; la table ne dit pas son montant ;
//  2. la perte observée que les coups calculés n'expliquent pas est partagée à parts égales entre les coups mesurés,
//     sans qu'un coup retire plus que ce qui reste à l'unité.
//
// Un tir à la tête qui achève l'unité retire tout ce qui lui reste de corps : la table donne le coup, pas la mort. Une
// vie commence pleine et rien ne régénère par délai : le bouclier ne remonte que quand le film l'écrit.

import (
	"math"

	"github.com/JGtm/halo-infinite-film-decoder/damageeffects"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// echelleDUnCoupPlein est l'échelle d'un coup direct plein.
const echelleDUnCoupPlein = 15

// PointsDeDegat est ce qu'un message retire à son unité blessée, en DIXIEMES de point (arrondi à la persistance : le
// calcul rejoue à l'identique depuis les faits).
type PointsDeDegat struct {
	Bouclier, Corps uint16
	// Mesure : le montant vient de la vie lue autour du coup, pas de la table (bon à deux points de bouclier près).
	Mesure bool
}

// cleDeScene identifie une scène : l'unité blessée, sa génération et l'instant de la trame.
type cleDeScene struct {
	ts         uint64
	unite      uint16
	generation uint8
}

// scenesDeDegats range les messages qui nomment une unité blessée par scène ; chaque scène garde les index de ses
// messages dans l'ordre du film. Un message sans unité blessée n'appartient à aucune scène : il ne retire rien, et ne
// se mêle pas à un message de l'unité 0, génération 0, au même instant.
func scenesDeDegats(degats []grammar.DegatLu) map[cleDeScene][]int {
	scenes := map[cleDeScene][]int{}
	for i, d := range degats {
		if !d.AVictime {
			continue
		}
		k := cleDeScene{d.TS, d.Victime, d.GenVictime}
		scenes[k] = append(scenes[k], i)
	}
	return scenes
}

// calculerLesPointsDeDegats rend, pour chaque message de `degats` (dans le même ordre), ce qu'il retire à son unité
// blessée.
func calculerLesPointsDeDegats(degats []grammar.DegatLu, vitales grammar.VitalitesLues) []PointsDeDegat {
	out := make([]PointsDeDegat, len(degats))
	vies := registreDesViesDe(vitales.Lectures)
	for k, indices := range scenesDeDegats(degats) {
		jouerLaScene(degats, indices, vies.vieDeLUnite(k.unite, k.generation), k.ts, out)
	}
	return out
}

// uniteEnPoints est l'état d'une unité pendant qu'une scène se joue, en points.
type uniteEnPoints struct {
	// bouclier et corps : ce qui reste. perteBouclier et perteCorps : ce que la vie lue a perdu et qu'aucun coup
	// calculé n'a encore expliqué. achevee : la vie lue à l'instant n'a plus de corps.
	bouclier, corps           float64
	perteBouclier, perteCorps float64
	achevee                   bool
}

// uniteDeLaScene pose l'état de départ d'une scène depuis la vie juste avant l'instant et la vie à l'instant.
func uniteDeLaScene(vie vieDUnBiped, ts uint64) uniteEnPoints {
	santeAvant, bouclierAvant := vie.avant(ts)
	santeA, bouclierA := vie.a(ts)
	return uniteEnPoints{
		bouclier:      bouclierAvant * damageeffects.BouclierMaximal,
		corps:         santeAvant * damageeffects.CorpsMaximal,
		perteBouclier: math.Max((bouclierAvant-bouclierA)*damageeffects.BouclierMaximal, 0),
		perteCorps:    math.Max((santeAvant-santeA)*damageeffects.CorpsMaximal, 0),
		achevee:       santeA <= 0,
	}
}

// genreDeCoup dit comment un coup se chiffre.
type genreDeCoup int

const (
	coupCalcule genreDeCoup = iota
	coupNul
	coupMesure
)

// classer dit comment le coup `d` se chiffre et, s'il est calculé, ce que la table lui fait retirer à l'unité telle
// qu'elle est.
func (u *uniteEnPoints) classer(d grammar.DegatLu) (genreDeCoup, [2]float64) {
	e, connu := damageeffects.Lookup(d.Tag)
	switch {
	case !d.ATag || !connu:
		return coupMesure, [2]float64{}
	case !e.APoints:
		return coupNul, [2]float64{}
	case !e.MontantFixe() || d.Echelle != echelleDUnCoupPlein:
		return coupMesure, [2]float64{}
	}
	surBouclier, surCorps := e.Repartir(e.Bas, u.bouclier, u.corps)
	if u.acheveALaTete(e, surBouclier, surCorps) {
		surCorps = u.corps
	}
	return coupCalcule, [2]float64{surBouclier, surCorps}
}

// acheveALaTete dit que le coup calculé est un tir à la tête qui achève l'unité : l'effet peut tirer à la tête, la vie
// lue n'a plus de corps, elle a perdu au corps plus que ce que le coup calculé lui retire (à un point près), et le
// bouclier n'a pas pu absorber le coup.
func (u *uniteEnPoints) acheveALaTete(e damageeffects.Effet, surBouclier, surCorps float64) bool {
	return u.achevee && e.PeutTirerALaTete() && u.perteCorps > surCorps+1 && u.bouclier-surBouclier < 1
}

// imputer retire un coup calculé de ce qui reste et de la perte encore inexpliquée.
func (u *uniteEnPoints) imputer(p [2]float64) {
	u.bouclier -= p[0]
	u.corps -= p[1]
	u.perteBouclier = math.Max(u.perteBouclier-p[0], 0)
	u.perteCorps = math.Max(u.perteCorps-p[1], 0)
}

// partDeLaPerte rend la part de la perte inexpliquée d'un des `n` coups mesurés, sans dépasser ce qui reste.
func (u *uniteEnPoints) partDeLaPerte(n int) [2]float64 {
	return [2]float64{
		math.Min(u.perteBouclier/float64(n), math.Max(u.bouclier, 0)),
		math.Min(u.perteCorps/float64(n), math.Max(u.corps, 0)),
	}
}

// jouerLaScene chiffre les messages `indices` d'une unité à l'instant `ts` dans `out`, depuis sa vie.
func jouerLaScene(degats []grammar.DegatLu, indices []int, vie vieDUnBiped, ts uint64, out []PointsDeDegat) {
	u := uniteDeLaScene(vie, ts)
	var mesures []int
	for _, i := range indices {
		genre, p := u.classer(degats[i])
		switch genre {
		case coupMesure:
			mesures = append(mesures, i)
		case coupCalcule:
			u.imputer(p)
			out[i] = PointsDeDegat{Bouclier: enDixiemes(p[0]), Corps: enDixiemes(p[1])}
		}
	}
	if len(mesures) == 0 {
		return
	}
	part := u.partDeLaPerte(len(mesures))
	for _, i := range mesures {
		out[i] = PointsDeDegat{Bouclier: enDixiemes(part[0]), Corps: enDixiemes(part[1]), Mesure: true}
	}
}

// enDixiemes arrondit un nombre de points au dixième, en dixièmes.
func enDixiemes(p float64) uint16 {
	return uint16(math.Round(math.Min(math.Max(p, 0), 6553) * 10))
}
