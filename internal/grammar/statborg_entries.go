package grammar

// statborg_entries.go — DE QUI EST LA LIGNE DU TABLEAU DES SCORES, LUE DANS L'IMAGE-CLE.
//
// Chaque entite du statborg porte un composant d'entree (`statborg-entry-index-and-type`, i57) qui dit
// de qui elle est : `[valeur:32][genre:8]`. Genre 0, une ligne de joueur : les seize bits de poids fort
// de la valeur sont l'identifiant moteur du joueur, le siege que les paquets de type 8 listent
// ([ScanRosterListings]) ; le joueur qui le tient a l'instant d'une valeur se lit dans ces plages.
// Genre 1, une ligne d'equipe : les seize bits de poids faible sont son numero.
//
// L'entree s'ecrit a la creation de l'entite et ne se re-emet pas dans les trames : elle se lit aux
// images-cles, un enregistrement par entite et par image-cle, sans balayage.
//
// Le meme record porte les ISSUES des manches (`statborg-round-outcomes`, deux bits par manche, zero tant que la manche
// n'est pas jouee) : l'union des manches qui en portent une ([types.StatborgEntryStats.DecidedRounds]) dit les manches
// que le film a jouees, ce que ne dit aucun enregistrement de valeur fortuit.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	// statborgTypeIndex est l'archetype du tableau des scores.
	statborgTypeIndex = 6
	// statborgEntryComponent est le composant d'entree de ses entites.
	statborgEntryComponent = "statborg-entry-index-and-type-component"
	// statborgEntryBits est la largeur du composant : la valeur et le genre.
	statborgEntryBits = 32 + 8
	// statborgOutcomesComponent est le composant des issues des manches : trente-deux champs de deux bits, un par manche
	// (zero tant que la manche n'est pas jouee).
	statborgOutcomesComponent = "statborg-round-outcomes-component"
	// statborgOutcomesBits est sa largeur.
	statborgOutcomesBits = 64
)

// ScanStatborgEntries lit l'entree de chaque entite du tableau des scores a chaque image-cle porteuse
// et rend les lectures dans l'ordre du flux. Un film sans registre, ou dont l'archetype du tableau n'a
// pas ce composant, rend `Scanned` faux.
//
// HORS LIGNE (parcourt les images-cles du film) ; un appel par cuisson.
func ScanStatborgEntries(fc *FilmContext) ([]types.StatborgEntry, types.StatborgEntryStats) {
	var st types.StatborgEntryStats
	reg, err := fc.Registry()
	if err != nil {
		return nil, st
	}
	arch, ok := reg.Archetype(statborgTypeIndex)
	if !ok || !contientLeComposant(arch.Components, statborgEntryComponent) {
		return nil, st
	}
	c := &canalDesEntreesDuStatborg{arch: arch, st: types.StatborgEntryStats{Scanned: true}}
	marche := fc.MarcheDImageCle()
	distribuerLaDemande(fc, demandeDImagesCles{marche: &marche}, []Canal{c})
	return c.lus, c.st
}

func contientLeComposant(noms []string, nom string) bool {
	for _, n := range noms {
		if n == nom {
			return true
		}
	}
	return false
}

// canalDesEntreesDuStatborg lit le composant d'entree de chaque record du tableau des scores d'une
// image-cle.
type canalDesEntreesDuStatborg struct {
	arch Archetype
	st   types.StatborgEntryStats
	lus  []types.StatborgEntry
}

func (*canalDesEntreesDuStatborg) Interets() []Interet {
	return []Interet{
		{Phase: PhaseImagesCles, TI: statborgTypeIndex, Composant: statborgEntryComponent},
		{Phase: PhaseImagesCles, TI: statborgTypeIndex, Composant: statborgOutcomesComponent},
	}
}

func (*canalDesEntreesDuStatborg) Clore(BilanDeMarche) {}

// ImageCle lit les records du tableau des scores d'UN paquet d'image-cle.
func (c *canalDesEntreesDuStatborg) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != statborgTypeIndex {
			continue
		}
		c.st.Records++
		c.st.DecidedRounds |= c.lireIssues(p, r)
		e, lu, ok := c.lireEntree(p, r)
		switch {
		case !ok:
			c.st.Unreached++
		case !lu:
			c.st.Vacant++
		default:
			c.st.Read++
			c.lus = append(c.lus, e)
		}
	}
}

// lireIssues rend le masque des manches dont le record `r` porte une issue : le bit de la manche `m` est leve quand son
// champ de deux bits n'est pas nul. Zero quand le composant n'est pas atteint.
func (c *canalDesEntreesDuStatborg) lireIssues(p *lecture.Paquet, r *lecture.Record) uint32 {
	for k := r.Comps[0]; k < r.Comps[1]; k++ {
		comp := p.Comps[k]
		if int(comp.Index) >= len(c.arch.Components) || c.arch.Components[comp.Index] != statborgOutcomesComponent {
			continue
		}
		if comp.Etat == lecture.EtatInfranchissable || comp.Etat == lecture.EtatArrete ||
			int(comp.Bits) < statborgOutcomesBits || int(comp.Debut)+statborgOutcomesBits > len(p.Payload)*8 {
			return 0
		}
		var masque uint32
		for m := range 32 {
			if source.BitsBourres(p.Payload, int(comp.Debut)+2*m, 2) != 0 {
				masque |= 1 << m
			}
		}
		return masque
	}
	return 0
}

// lireEntree rend l'entree du record `r` du paquet `p`. `ok` faux : le composant n'a pas ete atteint ;
// `lu` faux : l'entite n'est la ligne de personne (genre inconnu).
func (c *canalDesEntreesDuStatborg) lireEntree(p *lecture.Paquet, r *lecture.Record) (e types.StatborgEntry, lu, ok bool) {
	for k := r.Comps[0]; k < r.Comps[1]; k++ {
		comp := p.Comps[k]
		if int(comp.Index) >= len(c.arch.Components) || c.arch.Components[comp.Index] != statborgEntryComponent {
			continue
		}
		if comp.Etat == lecture.EtatInfranchissable || comp.Etat == lecture.EtatArrete ||
			int(comp.Bits) < statborgEntryBits || int(comp.Debut)+statborgEntryBits > len(p.Payload)*8 {
			return e, false, false
		}
		valeur := source.BitsBourres(p.Payload, int(comp.Debut), 32)
		e = types.StatborgEntry{Slot: int(r.Vie.Slot), TS: p.TS, Kind: int(source.BitsBourres(p.Payload, int(comp.Debut)+32, 8))}
		switch e.Kind {
		case types.StatborgEntryPlayer:
			e.ID = int(valeur >> 16)
		case types.StatborgEntryTeam:
			e.ID = int(valeur & 0xffff)
		default:
			return e, false, true
		}
		return e, true, true
	}
	return e, false, false
}
