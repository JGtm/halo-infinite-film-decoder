package grammar

// unit_control_scan.go — LE JOUEUR AUX COMMANDES DE CHAQUE BIPEDE, LU PAR LE CANAL DES ETATS DE
// MOUVEMENT.
//
// `consumeUnitControl` consomme la tete d `i18 unit-control` et la publie ([EtatControleUnite]) : le
// canal des etats de mouvement la recoit sur le meme crochet que ses autres etats, sans lire un bit
// de plus. Le record de CREATION d un bipede porte le composant avec le joueur qui le commande ; un
// delta le re-porte quand il change, et la porte de tete se ferme quand le joueur lache le corps.
//
// LE SLOT D UN RECORD NEW N EST LIE QU APRES SA TRAVERSEE : le crochet qui recoit la lecture ne peut
// donc pas dire si le slot est un bipede. La lecture est gardee EN ATTENTE jusqu a la fin de la
// trame ([movementStateScanner.Trame]), ou la table d entites de la marche a lie le slot, et
// seulement alors rangee ou ecartee.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// controleDesUnites est le recueil des lectures d `i18` d une marche.
type controleDesUnites struct {
	// attente : les lectures de la trame en cours, pas encore rapportees a un archetype.
	attente []types.UnitControlRead
	// vus deduplique : meme slot, meme paquet, memes valeurs.
	vus map[types.UnitControlRead]struct{}
	// lus : les lectures retenues, dans l ordre de la marche.
	lus []types.UnitControlRead
}

// recevoir garde une publication d `i18` avec la position du paquet `p`.
func (c *controleDesUnites) recevoir(m *MarcheDistribuee, slot uint32, v []uint64) {
	if len(v) < 2 {
		return
	}
	lu := types.UnitControlRead{Slot: slot, Controller: -1}
	if v[0] != 0 {
		lu.Controller = int(v[1]) //nolint:gosec // R(5), borne a 31
	}
	p := m.Paquet
	lu.TimestampUS, lu.Chunk, lu.PacketIndex = p.TS, p.Chunk, p.Index
	c.attente = append(c.attente, lu)
}

// solder range les lectures en attente de la trame qui vient d etre marchee : celles dont le slot est
// lie a un bipede (`estBipede`) sont gardees, une seule fois.
func (c *controleDesUnites) solder(estBipede func(slot uint32) bool) {
	for _, lu := range c.attente {
		if !estBipede(lu.Slot) {
			continue
		}
		if _, deja := c.vus[lu]; deja {
			continue
		}
		if c.vus == nil {
			c.vus = map[types.UnitControlRead]struct{}{}
		}
		c.vus[lu] = struct{}{}
		c.lus = append(c.lus, lu)
	}
	c.attente = c.attente[:0]
}

// publier rend les lectures gardees, triees sur (instant, slot, controleur) : un ordre total, pour
// que le document ne depende pas du parcours.
func (c *controleDesUnites) publier() []types.UnitControlRead {
	out := slices.Clone(c.lus)
	slices.SortStableFunc(out, func(a, b types.UnitControlRead) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot),
			cmp.Compare(a.Controller, b.Controller))
	})
	return out
}

// slotEstBipede dit que la table d entites de la marche lie le slot a l archetype bipede.
func (sc *movementStateScanner) slotEstBipede(slot uint32) bool {
	ti, ok := sc.archetypeDuSlot(slot)
	return ok && ti == BipedTypeIndex
}

// crochet aiguille une publication du deserialiseur : la tete d `unit-control` vers son recueil, sur tout
// film ; les autres etats vers [movementStateScanner.recevoir], sur un film qui les transmet.
func (sc *movementStateScanner) crochet(comp EtatMouvementComposant, slot uint32, v []uint64) {
	switch {
	case comp == EtatControleUnite:
		sc.controle.recevoir(sc.m, slot, v)
	case sc.transmet:
		sc.recevoir(comp, slot, v)
	}
}
