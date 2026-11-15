package grammar

// suppression_bipede.go — LA SUPPRESSION D UN BIPEDE, LUE DANS LES RECORDS DE LA MARCHE.
//
// Un record DEL retire un slot de la table d entites ([World.Unbind]) sans dire l archetype qu il
// retirait : le canal tient donc, depuis les records de chaque trame, les slots que la marche a
// vus porter un bipede avec leur generation (les deux bits de tete de l identifiant, celle que la
// table d entites compare). Un DEL de la meme generation est la suppression du corps ; un DEL d une
// generation ancienne, ou d un slot que la marche n a pas vu porter un bipede, n en est pas une.
//
// UN DEL D UNE MARCHE NON PROUVEE N EST PAS PUBLIE : le monde ne le prend pas non plus
// ([Lecteur.delier]), il se compte. Le canal ne relit aucun bit, il lit les records que la marche a
// deja ranges.
//
// Methode reprise de libfilmshell (Dend, OpenSpartan, licence MIT, `src/motion.rs` et `src/lives.rs`) :
// une vie finit par une mort quand son joueur lache la commande du corps, par la suppression du
// corps quand une manche la termine.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// suppressionsDeBipedes est le recueil des suppressions de bipede d une marche.
type suppressionsDeBipedes struct {
	// suivis : le slot -> la generation du bipede que la marche y a vu en dernier.
	suivis map[uint32]uint32
	lues   []types.BipedDeletion
	stats  types.BipedDeletionStats
}

// trame range les suppressions de bipede de la trame `p`, dans l ordre de ses records : un record
// qui engendre ou porte un bipede le suit, un record qui engendre un autre archetype libere le slot,
// un DEL d un bipede suivi le supprime.
func (s *suppressionsDeBipedes) trame(p *lecture.Paquet) {
	prouve := p.Debut != lecture.DebutParFermetureAuBit
	for _, r := range p.Records {
		slot, gen := r.Vie.Slot, r.Vie.Gen
		switch r.Genre {
		case lecture.GenreNeuf:
			s.creer(r, prouve)
		case lecture.GenreDelta:
			if r.TI == BipedTypeIndex {
				s.suivre(slot, gen, prouve)
			}
		case lecture.GenreSuppression:
			s.supprimer(p, slot, gen, prouve)
		}
	}
}

// creer range un record NEW : un bipede lu jusqu au bout est suivi, un autre archetype lu jusqu au bout
// libere le slot qu il lie. Un NEW desynchronise ou d une marche non prouvee ne lie rien.
func (s *suppressionsDeBipedes) creer(r lecture.Record, prouve bool) {
	if !prouve || r.Desync != lecture.SansDesynchronisation {
		return
	}
	if r.TI == BipedTypeIndex {
		s.suivre(r.Vie.Slot, r.Vie.Gen, prouve)
		return
	}
	delete(s.suivis, r.Vie.Slot)
}

// suivre retient que le slot porte un bipede de la generation `gen`. Une marche non prouvee ne lie
// rien au monde : elle ne suit rien non plus.
func (s *suppressionsDeBipedes) suivre(slot, gen uint32, prouve bool) {
	if !prouve {
		return
	}
	if s.suivis == nil {
		s.suivis = map[uint32]uint32{}
	}
	s.suivis[slot] = gen
}

// supprimer publie la suppression du bipede suivi du slot quand la generation du record est la
// sienne.
func (s *suppressionsDeBipedes) supprimer(p *lecture.Paquet, slot, gen uint32, prouve bool) {
	vue, ok := s.suivis[slot]
	if !ok || vue != gen {
		return
	}
	if !prouve {
		s.stats.DeletionsUnproven++
		return
	}
	delete(s.suivis, slot)
	s.stats.Deletions++
	s.lues = append(s.lues, types.BipedDeletion{Slot: slot, TimestampUS: p.TS, Chunk: p.Chunk, PacketIndex: p.Index})
}

// publier rend les suppressions gardees, triees sur (instant, slot) : un ordre total, pour que le
// document ne depende pas du parcours.
func (s *suppressionsDeBipedes) publier() []types.BipedDeletion {
	out := slices.Clone(s.lues)
	slices.SortStableFunc(out, func(a, b types.BipedDeletion) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot))
	})
	return out
}
