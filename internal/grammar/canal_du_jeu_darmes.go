package grammar

// canal_du_jeu_darmes.go : LE JEU D ARMES DESIRE DU BIPEDE DANS LES TRAMES DELTA, recueilli par slot
// sur la marche des trames.
//
// Un delta ne porte `i42` que quand le jeu change : chaque lecture est un changement d arme, a la
// trame pres. Rien n est lu de plus que ce que le deserialiseur publiait deja : le crochet recoit le
// slot du record avec les trois champs de [JeuDArmes]. Le canal ne marche rien lui-meme, ne lance
// aucun fil et ne tient qu une table de deduplication par film : la sortie ne depend ni du nombre
// de fils ni des voies.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// canalDuJeuDArmes est le canal du jeu d armes desire de la marche des trames ([CanalDesTrames]).
type canalDuJeuDArmes struct {
	st   types.DrawnWeaponStats
	m    *MarcheDistribuee
	vues map[jeuDArmesCle]types.DrawnWeaponRead
	out  []types.DrawnWeaponRead
}

// jeuDArmesCle deduplique une lecture : une seconde traversee du meme record (chaine de transitoires
// de [decodeInferLoop]) re-publie la MEME lecture.
type jeuDArmesCle struct {
	slot uint32
	ts   uint64
}

// nouveauCanalDuJeuDArmes prepare le canal d un film d archetype bipede `arch` : un archetype qui ne
// declare pas `i42` ne transmet pas le jeu d armes, ce que `Absent` publie.
func nouveauCanalDuJeuDArmes(arch Archetype) *canalDuJeuDArmes {
	c := &canalDuJeuDArmes{vues: map[jeuDArmesCle]types.DrawnWeaponRead{}}
	c.st.Absent = componentIndexOfAny(arch, compBipedDesiredWeaponSet) < 0
	return c
}

// Interets : `i42` du bipede, dans les trames ; rien pour un film dont l archetype ne le declare pas.
func (c *canalDuJeuDArmes) Interets() []Interet {
	if c.st.Absent {
		return nil
	}
	return []Interet{{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: compBipedDesiredWeaponSet}}
}

// Brancher retient la marche et pose le crochet du jeu d armes desire.
func (c *canalDuJeuDArmes) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.m = m
	if !c.st.Absent {
		obs.DesiredWeaponSetHook = c.recevoir
	}
}

// Trame : rien a compter par trame, le crochet recoit les lectures pendant la marche.
func (*canalDuJeuDArmes) Trame(*lecture.Paquet) {}

// Clore verse la table de deduplication dans la sortie, triee par (instant, slot).
func (c *canalDuJeuDArmes) Clore(BilanDeMarche) {
	c.st.Scanned = true
	c.out = make([]types.DrawnWeaponRead, 0, len(c.vues))
	for _, r := range c.vues {
		c.out = append(c.out, r)
	}
	slices.SortFunc(c.out, func(a, b types.DrawnWeaponRead) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot))
	})
	c.vues = nil
}

// recevoir capte UNE lecture d `i42`. Le filtre est celui des etats de mouvement
// ([movementStateScanner.recevoir]) : la lecture ne vaut que sur un slot lie au bipede dans la table
// d entites de la marche ; un NEW sur un slot neuf, lie apres sa traversee, est ecarte et compte.
func (c *canalDuJeuDArmes) recevoir(slot uint32, j JeuDArmes) {
	if e, ok := c.m.Entites.Entite(slot); !ok || e.TI != BipedTypeIndex {
		c.st.SlotUnbound++
		return
	}
	k := jeuDArmesCle{slot: slot, ts: c.m.Paquet.TS}
	if _, deja := c.vues[k]; deja {
		c.st.Duplicates++
		return
	}
	c.vues[k] = types.DrawnWeaponRead{TimestampUS: k.ts, Slot: slot,
		Principal: int8(j.Principal), Second: int8(j.Second)} //nolint:gosec // emplacements -1 a 3
	c.st.Read++
}
