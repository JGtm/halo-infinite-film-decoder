package grammar

// canal_des_proprietes_gerees_trames.go — LES VALEURS DES PROPRIETES RESEAU DE ti=13 DANS LES TRAMES DELTA,
// UN CANAL DE LA MARCHE DES TRAMES (ADR 0037 IR-2).
//
// # CE QUE LE FILM ECRIT
//
// L archetype 13 est un sac de proprietes nommees attachees a un objet gere par le mode (cf.
// components_managed_property.go). Une trame delta n emet une propriete qu a son changement : une valeur qui
// ne bouge pas n est que dans les images-cles ([ManagedPropertyScan.KeyReads]).
//
// # CE QUE CE CANAL REND, ET CE QU IL NE DIT PAS
//
// Les valeurs BRUTES — le tag et le quantum — des records de ti=13 que la marche lit dans la liste des
// entites de chaque trame (vue B), rendues par la couche de capture : jamais un octet. Une lecture porte le
// slot de l objet, l horloge du paquet, le champ (valeur scalaire ou valeur par joueur) et l index de joueur du
// film pour une valeur par joueur. Ce que le tag veut dire est affaire de la publication.
//
// LA MARCHE LIT LA LISTE DES ENTITES DE LA TRAME (vue B). Une trame dont la liste n est pas localisee n en rend
// rien ([StatsDesProprietesGerees.TramesNonLues]), et la passe qui ancre les records sur la bande de slots
// ([ScanManagedProperties]) retrouve des changements d etat de zone que la marche ne lit pas : c est pourquoi les
// zones restent sur la passe, et seule la jauge de retour du drapeau lit ce canal.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// StatsDesProprietesGerees sont les denominateurs du canal.
type StatsDesProprietesGerees struct {
	// Trames : les trames delta dont la liste d entites a ete lue ; TramesNonLues : celles dont la liste n a pas
	// ete localisee, et dont aucune valeur n a donc ete lue.
	Trames, TramesNonLues int
	// Records : les records de ti=13 qui portent au moins une valeur.
	Records int
}

// ProprietesGereesLues est ce que le canal rend d un film.
type ProprietesGereesLues struct {
	// Lectures sont les valeurs lues dans les trames delta, dans l ordre du flux. Leur `Chained` dit que le
	// record qui les porte est clos sans desynchronisation ; elles ne portent pas de nom ([ManagedPropertyRead.Named]
	// est faux : `i0` n est ecrit qu a la creation).
	Lectures []ManagedPropertyRead
	Stats    StatsDesProprietesGerees
}

// canalDesProprietesGereesTrames est le canal des valeurs de propriete des trames delta ([CanalDesTrames]).
type canalDesProprietesGereesTrames struct {
	m  *MarcheDistribuee
	lu ProprietesGereesLues
}

func nouveauCanalDesProprietesGereesTrames() *canalDesProprietesGereesTrames {
	return &canalDesProprietesGereesTrames{}
}

// Interets : les deux composants de valeur de ti=13, dans les trames.
func (*canalDesProprietesGereesTrames) Interets() []Interet {
	return []Interet{
		{Phase: PhaseTrames, TI: ManagedPropertyTypeIndex, Composant: compManagedObjectProperty},
		{Phase: PhaseTrames, TI: ManagedPropertyTypeIndex, Composant: compManagedObjectPlayerMaskedProperty},
	}
}

// Brancher garde ce que la marche expose ; le canal ne pose aucun crochet.
func (c *canalDesProprietesGereesTrames) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Clore : rien, le canal se lit par [canalDesProprietesGereesTrames.resultat].
func (*canalDesProprietesGereesTrames) Clore(BilanDeMarche) {}

// Trame range les valeurs de propriete des records de la trame.
func (c *canalDesProprietesGereesTrames) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		if listeAnnoncee(&p.VueA) {
			c.lu.Stats.TramesNonLues++
		}
		return
	}
	c.lu.Stats.Trames++
	for i := range recs {
		if c.ranger(p.TS, &recs[i]) {
			c.lu.Stats.Records++
		}
	}
}

// ranger verse les valeurs de propriete du record `r` et dit s il en portait une.
func (c *canalDesProprietesGereesTrames) ranger(ts uint64, r *FrameRecord) bool {
	vu := false
	for _, comp := range r.Trace.Comps {
		if !comp.Ported {
			continue
		}
		v, ok := comp.ManagedPropertyOf()
		if !ok {
			continue
		}
		vu = true
		rd := ManagedPropertyRead{Slot: r.Slot, TimestampUS: ts, Tag: v.Tag, Value: v.Value, HasValue: v.HasValue,
			Chained: r.DesyncAt == -1}
		if comp.Name == compManagedObjectProperty {
			rd.Field, rd.FilmIndex = ManagedPropertyScalar, -1
		} else {
			rd.Field, rd.FilmIndex = ManagedPropertyPerPlayer, ManagedPropertyFilmIndex(comp.Index)
		}
		c.lu.Lectures = append(c.lu.Lectures, rd)
	}
	return vu
}

// resultat rend ce que le canal a lu.
func (c *canalDesProprietesGereesTrames) resultat() ProprietesGereesLues { return c.lu }
