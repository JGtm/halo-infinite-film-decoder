package grammar

// commandes_de_controle.go — LES COMMANDES TENUES DE L ENTREE DE CONTROLE (deplacement, accroupi, saut), pliees
// par le meme collecteur que le tir continu ([collecteurTirContinu]).
//
// L entree de controle d un participant (`FUN_1406cd860`) porte, hors de son bloc d action, trois commandes :
//
//	le couple analogique du deplacement   deux scalaires de six bits, `0x1f` a zero exact
//	                                      ([LargeurScalaireAnalogique]) ; la commande est tenue des qu un scalaire
//	                                      quitte le zero exact ;
//	le bouton d accroupi                  le bit de valeur 1 du champ facultatif de cinq bits (les drapeaux `+0x14`,
//	                                      [largeurDrapeaux14]) ;
//	le bouton de saut                     le bit de valeur 4 du meme champ.
//
// Un etat de commande persiste jusqu a l entree lue suivante, exactement comme une gachette : la regle des
// debuts, des fins et des trous est celle de `tir_continu.go`, et une entree sans bloc ne dit rien. Le document
// publie l intervalle ([types.ControlHold]), jamais la serie des valeurs.
//
// LES DEUX BITS DES DRAPEAUX. Leur lecture : bit de valeur 1 = bouton d accroupi, bit de valeur 4 = bouton de
// saut. Le premier bit n est pas l accroupi « du corps » : le jeu n a qu un bouton pour l accroupi et la glissade.
// Le second est un bouton enfonce, pas un saut effectue (il peut l etre en l air ou contre un obstacle). Les
// autres valeurs du champ ne sont pas lues : aucune mesure ne les rattache a un geste.
// Le champ est FACULTATIF : absent du flux ([ChampDeControleAbsent], qui vaut -1), il dit « aucun bouton
// tenu », et c est l absence qu on teste AVANT le masque, parce que `-1 & 4` n est pas nul.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// scalaireAnalogiqueNul est le code du zero exact d un scalaire analogique (`FUN_1406d6ef4`).
const scalaireAnalogiqueNul = 0x1f

// Les bits lus des drapeaux `+0x14` de l entree de controle, par leur VALEUR.
const (
	drapeauAccroupi = 1 << 0
	drapeauSaut     = 1 << 2
)

// deplacementNonNul dit que l un des deux scalaires analogiques n est pas au zero exact.
func (c ChampsDeControle) deplacementNonNul() bool {
	for _, v := range c.Analogique {
		if v != ChampDeControleAbsent && v != scalaireAnalogiqueNul {
			return true
		}
	}
	return false
}

// drapeauPose dit que le bit de valeur `bit` des drapeaux est pose. Un champ absent ne pose rien.
func (c ChampsDeControle) drapeauPose(bit int) bool {
	return c.Drapeaux != ChampDeControleAbsent && c.Drapeaux&bit != 0
}

// genreDeNature rend le genre publie d une nature de bit qui n est pas un bit de tir.
func genreDeNature(n natureDeBit) string {
	switch n {
	case natureAccroupi:
		return types.MovementCrouchCommand
	case natureSaut:
		return types.MovementJumpCommand
	default:
		return types.MovementMoveCommand
	}
}

// commande applique UNE entree lue qui porte le bloc : l etat de ses trois commandes.
func (c *collecteurTirContinu) commande(e EntreeDeControle) {
	if e.Index < 0 || e.Index >= joueursDeControle {
		return
	}
	c.commandeStats.Entries++
	c.pas.noter(e.Index, c.ts)
	c.bit(cleBitDeTir{joueur: e.Index, nature: natureDeplacement}, e.Champs.deplacementNonNul(), 0)
	c.bit(cleBitDeTir{joueur: e.Index, nature: natureAccroupi}, e.Champs.drapeauPose(drapeauAccroupi), 0)
	c.bit(cleBitDeTir{joueur: e.Index, nature: natureSaut}, e.Champs.drapeauPose(drapeauSaut), 0)
}

// compteDuGenre rend le compte des commandes tenues du genre `kind`.
func (c *collecteurTirContinu) compteDuGenre(kind string) *types.ControlHoldCount {
	switch kind {
	case types.MovementCrouchCommand:
		return &c.commandeStats.Crouch
	case types.MovementJumpCommand:
		return &c.commandeStats.Jump
	default:
		return &c.commandeStats.Move
	}
}

// clorerLaCommande rend la commande tenue d un bit, finie a `fin` pour la cause `borne`, et libere le bit.
func (c *collecteurTirContinu) clorerLaCommande(k cleBitDeTir, s *etatBitDeTir, fin uint64, borne string) {
	h := types.ControlHold{Kind: genreDeNature(k.nature), FilmIndex: k.joueur, StartUS: s.debut, EndUS: fin,
		StartBound: s.debutBorne, EndBound: borne, Entries: s.entrees, Holes: s.trous}
	c.commandes = append(c.commandes, h)
	st := c.compteDuGenre(h.Kind)
	st.Holds++
	st.InnerHoles += len(h.Holes)
	if h.StartBound == types.ContinuousFireBoundHole || h.EndBound == types.ContinuousFireBoundHole || len(h.Holes) > 0 {
		st.WithHole++
	}
	delete(c.tenus, k)
}

// sortControlHolds ordonne les commandes tenues sur un ordre TOTAL (debut, joueur, genre).
func sortControlHolds(out []types.ControlHold) {
	slices.SortStableFunc(out, func(a, b types.ControlHold) int {
		return cmp.Or(cmp.Compare(a.StartUS, b.StartUS), cmp.Compare(a.FilmIndex, b.FilmIndex),
			cmp.Compare(a.Kind, b.Kind))
	})
}
