package grammar

// movement_states_occurrences.go — LES OCCURRENCES D UN COMPOSANT DANS LE REGISTRE, que le canal des
// etats de mouvement compte par record (sorti de `movement_states.go`, deplacement pur).

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// occurrencesDuComposant : par archetype du registre (son rang), le masque des index d iteration ou
// il declare un composant.
type occurrencesDuComposant map[int16]uint64

// occurrencesDe releve, dans le registre, les index ou chaque archetype declare le composant `nom`.
func occurrencesDe(reg *Registry, nom string) occurrencesDuComposant {
	out := occurrencesDuComposant{}
	for ti, a := range reg.Archetypes {
		for _, i := range a.indicesOf(nom) {
			if i < 64 {
				out[int16(ti)] |= 1 << uint(i) //nolint:gosec // rang d un archetype, sur 6 bits
			}
		}
	}
	return out
}

// lu rend 1 quand le record `r` du paquet `p` porte une occurrence du composant — traversee ou
// infranchissable : une lecture du composant au sens de la trace de la marche —, 0 sinon.
func (o occurrencesDuComposant) lu(p *lecture.Paquet, r lecture.Record) int {
	masque, ok := o[r.TI]
	if !ok {
		return 0
	}
	for _, c := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if masque>>c.Index&1 == 1 {
			return 1
		}
	}
	return 0
}
