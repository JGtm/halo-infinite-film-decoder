package grammar

// canal_des_vitalites.go — LA VIE DES BIPEDES : SANTE ET BOUCLIER, UN CANAL DE LA MARCHE DES TRAMES.
//
// # CE QUE LE FILM ECRIT
//
// Le bipede porte deux composants de vitalite, `object-body-vitality` (i4, la sante, de -1 a +1) et
// `object-shield-vitality` (i5, le bouclier, de 0 a 4 ; 1 est le bouclier plein), que le jeu ne reecrit que
// quand l un d eux change. Un joueur immobile qui encaisse un degat n ecrit aucune position : ses vitalites ne
// se lisent que dans le composant lui-meme, pas dans les lectures de position des bipedes.
//
// # CE QUE CE CANAL REND, ET CE QU IL NE DIT PAS
//
// Les lectures BRUTES, dans l ordre du flux, par vie de bipede (le slot et sa generation) : jamais un octet.
// La creation d un bipede est une lecture a part (une vie commence pleine : ni creation ni mise a jour n ecrit
// ses vitalites a sa naissance). Ce que vaut une lecture pour un coup, c est le calcul des degats qui le dit.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// LectureVitale est une lecture de la vie d un bipede : sa creation, ou une mise a jour de sa sante et/ou de son
// bouclier.
type LectureVitale struct {
	// TS est l horloge du film du paquet qui la porte, en microsecondes.
	TS uint64
	// Slot et Gen identifient la vie : la paire, comme [types.LifeKey].
	Slot uint32
	Gen  uint8
	// Creation : la vie commence ici (sante et bouclier pleins), aucune vitalite n est lue.
	Creation bool
	// ASante et ABouclier disent quelles vitalites la lecture porte ; Sante est dans [-1, +1], Bouclier dans [0, 4].
	ASante, ABouclier bool
	Sante, Bouclier   float32
}

// StatsVitalites sont les denominateurs du canal.
type StatsVitalites struct {
	// Trames : les trames delta dont les records ont ete lus ; TramesNonLues : celles dont la liste est annoncee et dont les records restent
	// illisibles, hors celles que la vue A arrete deja ([StatsDegats.Arretees]).
	Trames, TramesNonLues int
	// Lectures : les lectures rendues ; Creations : celles qui sont des creations.
	Lectures, Creations int
	// LocalisationsALargeurLibre : les listes que la marche n avait pas localisees et que le repli a largeur libre a recuperees
	// (`repli_localisation_largeur_libre`, comme le canal des morts) ; la cuisson les verse au rapport des replis du contexte.
	LocalisationsALargeurLibre int
}

// VitalitesLues est ce que le canal rend d un film.
type VitalitesLues struct {
	Lectures []LectureVitale
	Stats    StatsVitalites
}

// canalDesVitalites est le canal de la vie des bipedes ([CanalDesTrames]).
type canalDesVitalites struct {
	m  *MarcheDistribuee
	lu VitalitesLues
}

func nouveauCanalDesVitalites() *canalDesVitalites { return &canalDesVitalites{} }

// Interets : la sante et le bouclier du bipede, dans les trames.
func (c *canalDesVitalites) Interets() []Interet {
	return []Interet{
		{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: compObjectBodyVitality},
		{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: compObjectShieldVitality},
	}
}

// Brancher garde ce que la marche expose ; le canal ne pose aucun crochet.
func (c *canalDesVitalites) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Clore : rien — le canal se lit par son champ.
func (c *canalDesVitalites) Clore(BilanDeMarche) {}

// Trame lit les records bipèdes de la trame : les créations et les mises à jour des vitalités. Une liste que la marche n'a
// pas localisée se relit comme le fait le canal des morts ([MarcheDistribuee.recupererLaListe]) ; si elle reste illisible, la
// trame ne dit rien de la vie des bipèdes et se compte, sauf quand la vue A s'y arrête déjà (la trame est alors perdue pour
// les messages de dégât, et son compte est celui de [StatsDegats.Arretees]).
func (c *canalDesVitalites) Trame(p *lecture.Paquet) {
	annoncee := listeAnnoncee(&p.VueA)
	recs, lus := c.m.recordsDeLaTrame()
	if !lus && annoncee {
		var libre bool
		recs, lus, libre = c.m.recupererLaListe()
		c.lu.Stats.LocalisationsALargeurLibre += unSi(libre)
	}
	if !lus {
		if annoncee && p.VueA.Etat != lecture.VueArretee {
			c.lu.Stats.TramesNonLues++
		}
		return
	}
	c.lu.Stats.Trames++
	c.lire(p.TS, recs)
}

// lire range les lectures des records `recs` d'une trame de l'instant `ts`.
func (c *canalDesVitalites) lire(ts uint64, recs []FrameRecord) {
	for i := range recs {
		r := &recs[i]
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		l := LectureVitale{TS: ts, Slot: r.Slot, Gen: uint8(r.ID >> 30)} //nolint:gosec // generation sur 2 bits
		if r.Type == recNew {
			l.Creation = true
			c.lu.Stats.Creations++
			c.lu.Lectures = append(c.lu.Lectures, l)
			continue
		}
		if r.Type != recDelta {
			continue
		}
		for _, comp := range r.Trace.Comps {
			if !comp.Ported {
				continue
			}
			if b, ok := comp.BodyOf(); ok {
				l.ASante, l.Sante = true, b.Health
			}
			if s, ok := comp.ShieldOf(); ok {
				l.ABouclier, l.Bouclier = true, s.Shield
			}
		}
		if l.ASante || l.ABouclier {
			c.lu.Lectures = append(c.lu.Lectures, l)
		}
	}
	c.lu.Stats.Lectures = len(c.lu.Lectures)
}

// resultat rend ce que le canal a lu.
func (c *canalDesVitalites) resultat() VitalitesLues { return c.lu }
