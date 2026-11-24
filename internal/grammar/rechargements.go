package grammar

// rechargements.go — LES RECHARGEMENTS D ARME DU FILM (genre 38 de la vue A).
//
// Un message `weapon_reload` porte trois references d en-tete, quatre booleens et l index de joueur de
// l auteur ([chargeRechargement]). La premiere reference est l unite qui recharge ; aucun message ne
// nomme l arme. Le canal rend, dans l ordre du flux, un rechargement par (trame, auteur, references) :
// le jeu ecrit souvent le meme message deux fois dans la meme trame, et ces copies ne sont pas des
// rechargements de plus : elles se comptent, [types.ReloadStats.Copies], et chaque rechargement rendu porte le
// nombre de copies de son message ([types.ReloadRead.Copies]). Deux messages d une meme trame
// qui ne different que par leurs booleens sont une seule identite : le message ne dit pas laquelle des
// deux copies vaut mieux, et aucun ne decrit l arme.
//
// Les messages derriere une vue A que la lecture n atteint pas ne se lisent pas : les trames arretees se
// comptent ([StatsRechargements.Arretees]).

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// GenreRechargement est le genre `weapon_reload` de la vue A.
const GenreRechargement = 38

// messageDeRechargement range le message de rechargement que [chargeRechargement] vient de lire a la
// position `debut`, avec les references de son en-tete.
func messageDeRechargement(br *Lecteur, debut int) lecture.MessageDeRechargement {
	m := br.rechargementLu
	m.Debut = uint32(debut) //nolint:gosec // position dans un payload
	for i, d := range [3]uint8{lecture.RechargementRef0, lecture.RechargementRef1, lecture.RechargementRef2} {
		if r := br.refsDuMessage[i]; r.presente {
			m.Ref[i], m.Gen[i] = uint16(r.index), uint8(r.gen) //nolint:gosec // index de domaine, generation 2 bits
			m.Drapeaux |= d
		}
	}
	return m
}

// identiteDeRechargement est ce qui fait de deux messages d une trame un seul rechargement : l auteur et
// les references de l en-tete, a l exclusion des booleens.
type identiteDeRechargement struct {
	joueur   int8
	drapeaux uint8
	ref      [3]uint16
	gen      [3]uint8
}

// canalDesRechargements recueille les rechargements des trames delta d un film.
type canalDesRechargements struct {
	out   []types.ReloadRead
	stats types.ReloadStats
	// vus : les identites deja rendues par la trame en cours (au plus quelques unes), et rangs la position
	// dans `out` du rechargement rendu pour chacune.
	vus   []identiteDeRechargement
	rangs []int
}

func nouveauCanalDesRechargements() *canalDesRechargements { return &canalDesRechargements{} }

// Interets : le canal ne lit que la vue A rangee de chaque trame.
func (c *canalDesRechargements) Interets() []Interet { return nil }

// Clore : rien, le canal se lit par son champ.
func (c *canalDesRechargements) Clore(BilanDeMarche) {}

// Tete range les rechargements de la vue A de la trame `p`.
func (c *canalDesRechargements) Tete(p *lecture.Paquet) {
	st := &c.stats
	st.Frames++
	switch p.VueA.Etat {
	case lecture.VueTerminee:
		st.Closed++
	case lecture.VueArretee:
		st.Stopped++
	}
	c.vus, c.rangs = c.vus[:0], c.rangs[:0]
	for _, m := range p.VueA.Rechargements {
		id := identiteDeRechargement{joueur: m.Joueur, drapeaux: m.Drapeaux, ref: m.Ref, gen: m.Gen}
		if k := c.dejaVu(id); k >= 0 {
			st.Copies++
			c.out[c.rangs[k]].Copies++
			continue
		}
		c.vus, c.rangs = append(c.vus, id), append(c.rangs, len(c.out))
		c.out = append(c.out, types.ReloadRead{TimestampUS: p.TS, Player: m.Joueur})
		st.Messages++
	}
}

// dejaVu rend le rang, dans `vus`, de l identite que la trame en cours a deja rendue, ou -1.
func (c *canalDesRechargements) dejaVu(id identiteDeRechargement) int {
	for k, v := range c.vus {
		if v == id {
			return k
		}
	}
	return -1
}
