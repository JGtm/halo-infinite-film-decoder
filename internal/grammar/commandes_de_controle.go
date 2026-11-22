package grammar

// commandes_de_controle.go — LA COMMANDE DE DEPLACEMENT TENUE DE L ENTREE DE CONTROLE, pliee par le meme
// collecteur que le tir continu ([collecteurTirContinu]).
//
// L entree de controle d un participant (`FUN_1406cd860`) porte, hors de son bloc d action, le couple
// analogique du deplacement : deux scalaires de six bits, `0x1f` a zero exact ([LargeurScalaireAnalogique]).
// Un etat de commande persiste jusqu a l entree lue suivante, exactement comme une gachette : la regle des
// debuts, des fins et des trous est celle de `tir_continu.go`, et une entree sans bloc ne dit rien. Le
// document publie l intervalle ([types.ControlHold]), jamais la serie des valeurs.
//
// Le champ facultatif de cinq bits de l entree (les drapeaux `+0x14`, [largeurDrapeaux14]) n est pas publie :
// son bit de poids 4, qui parait etre le bouton de saut, ne s accorde pas assez avec le saut derive de la
// vitesse verticale d un build a l autre (FORMAT.md, section de la vue de controle).

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// scalaireAnalogiqueNul est le code du zero exact d un scalaire analogique (`FUN_1406d6ef4`).
const scalaireAnalogiqueNul = 0x1f

// deplacementNonNul dit que l un des deux scalaires analogiques n est pas au zero exact.
func (c ChampsDeControle) deplacementNonNul() bool {
	for _, v := range c.Analogique {
		if v != ChampDeControleAbsent && v != scalaireAnalogiqueNul {
			return true
		}
	}
	return false
}

// commande applique UNE entree lue qui porte le bloc : l etat de sa commande de deplacement.
func (c *collecteurTirContinu) commande(e EntreeDeControle) {
	if e.Index < 0 || e.Index >= joueursDeControle {
		return
	}
	c.commandeStats.Entries++
	c.bit(cleBitDeTir{joueur: e.Index, nature: natureDeplacement}, e.Champs.deplacementNonNul(), 0)
}

// clorerLaCommande rend la commande tenue d un bit, finie a `fin` pour la cause `borne`, et libere le bit.
func (c *collecteurTirContinu) clorerLaCommande(k cleBitDeTir, s *etatBitDeTir, fin uint64, borne string) {
	h := types.ControlHold{FilmIndex: k.joueur, StartUS: s.debut, EndUS: fin, StartBound: s.debutBorne,
		EndBound: borne, Entries: s.entrees, Holes: s.trous}
	c.commandes = append(c.commandes, h)
	st := &c.commandeStats
	st.Holds++
	st.InnerHoles += len(h.Holes)
	if h.StartBound == types.ContinuousFireBoundHole || h.EndBound == types.ContinuousFireBoundHole || len(h.Holes) > 0 {
		st.WithHole++
	}
	delete(c.tenus, k)
}

// sortControlHolds ordonne les commandes tenues sur un ordre TOTAL (debut, joueur).
func sortControlHolds(out []types.ControlHold) {
	slices.SortStableFunc(out, func(a, b types.ControlHold) int {
		return cmp.Or(cmp.Compare(a.StartUS, b.StartUS), cmp.Compare(a.FilmIndex, b.FilmIndex))
	})
}
