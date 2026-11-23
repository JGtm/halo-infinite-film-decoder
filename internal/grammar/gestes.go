package grammar

// gestes.go — LES GESTES DU FILM : LE LANCER AMORCE (genre 39) ET LE COUP DE MELEE (genre 40) DE LA VUE A.
//
// Les deux messages (`biped_throw_initiate`, `FUN_140c6a58c` ; `biped_melee_initiate`, `FUN_140ff8d70`)
// finissent sur l index de joueur de l auteur (`FUN_1407f2058`) : R(1), puis R(5) si la garde est a zero.
// Leur charge propre (tete, arme, variante) n est pas gardee : le canal rend l instant de la trame, l auteur
// et le geste, rien de plus. Un message ne dit pas ce que son geste devient : le coup part qu il touche ou
// non, l amorce n est pas la grenade.
//
// Le canal rend un geste par (trame, genre, auteur, references), comme [canalDesRechargements] : le jeu peut
// ecrire le meme message plusieurs fois dans la meme trame, et ces copies ne sont pas des gestes de plus
// (elles se comptent, [types.GestureStats]). Les messages derriere une vue A que la lecture n atteint pas ne
// se lisent pas : les trames arretees se comptent ([types.GestureStats.Stopped]).

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Les genres de la vue A que le canal des gestes lit.
const (
	// GenreLancerInitie est le genre `biped_throw_initiate`.
	GenreLancerInitie = 39
	// GenreCorpsACorps est le genre `biped_melee_initiate`.
	GenreCorpsACorps = 40
)

// lireJoueurDuMessage lit l index de joueur de l auteur que ferment les messages de la vue A
// (`FUN_1407f2058` : R(1) ; si 0 : R(5)) ; [lecture.RefAbsente] quand la garde est a un.
func lireJoueurDuMessage(br *Lecteur) int8 {
	if br.ReadBit() {
		return lecture.RefAbsente
	}
	return int8(br.ReadBits(5)) //nolint:gosec // R(5)
}

// recueillirUnGeste range, dans `out`, le message de genre 39 ou 40 que [lireLaVueA] vient de lire a la
// position `debut` ; il rend vrai pour ces deux genres, pour que l appelant n en fasse rien d autre.
func recueillirUnGeste(br *Lecteur, out *FluxVueA, genre, debut int) bool {
	if genre != GenreLancerInitie && genre != GenreCorpsACorps {
		return false
	}
	m := br.gesteLu
	m.Debut, m.Genre = uint32(debut), uint8(genre) //nolint:gosec // position dans un payload, genre R(7)
	for i, d := range [3]uint8{lecture.GesteRef0, lecture.GesteRef1, lecture.GesteRef2} {
		if r := br.refsDuMessage[i]; r.presente {
			m.Ref[i], m.Gen[i] = uint16(r.index), uint8(r.gen) //nolint:gosec // index de domaine, generation 2 bits
			m.Drapeaux |= d
		}
	}
	out.Gestes = append(out.Gestes, m)
	return true
}

// identiteDeGeste est ce qui fait de deux messages d une trame un seul geste : le genre, l auteur et les
// references de l en-tete.
type identiteDeGeste struct {
	genre    uint8
	joueur   int8
	drapeaux uint8
	ref      [3]uint16
	gen      [3]uint8
}

// canalDesGestes recueille les gestes des trames delta d un film.
type canalDesGestes struct {
	out   []types.GestureRead
	stats types.GestureStats
	// vus : les identites deja rendues par la trame en cours (au plus quelques unes).
	vus []identiteDeGeste
}

func nouveauCanalDesGestes() *canalDesGestes { return &canalDesGestes{} }

// Interets : le canal ne lit que la vue A rangee de chaque trame.
func (c *canalDesGestes) Interets() []Interet { return nil }

// Clore : rien, le canal se lit par son champ.
func (c *canalDesGestes) Clore(BilanDeMarche) {}

// Tete range les gestes de la vue A de la trame `p`.
func (c *canalDesGestes) Tete(p *lecture.Paquet) {
	st := &c.stats
	st.Frames++
	switch p.VueA.Etat {
	case lecture.VueTerminee:
		st.Closed++
	case lecture.VueArretee:
		st.Stopped++
	}
	c.vus = c.vus[:0]
	for _, m := range p.VueA.Gestes {
		id := identiteDeGeste{genre: m.Genre, joueur: m.Joueur, drapeaux: m.Drapeaux, ref: m.Ref, gen: m.Gen}
		kind, compte := types.GestureMelee, &st.Melee
		if m.Genre == GenreLancerInitie {
			kind, compte = types.GestureThrowStart, &st.Throw
		}
		if c.dejaVu(id) {
			compte.Copies++
			continue
		}
		c.vus = append(c.vus, id)
		c.out = append(c.out, types.GestureRead{TimestampUS: p.TS, Player: m.Joueur, Kind: kind})
		compte.Messages++
	}
}

// dejaVu dit que la trame en cours a deja rendu un geste de cette identite.
func (c *canalDesGestes) dejaVu(id identiteDeGeste) bool {
	for _, v := range c.vus {
		if v == id {
			return true
		}
	}
	return false
}
