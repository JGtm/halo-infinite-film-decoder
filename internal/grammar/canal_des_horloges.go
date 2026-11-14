package grammar

// canal_des_horloges.go — L HORLOGE DE MANCHE ET LES COMPTES A REBOURS DE REAPPARITION, UN CANAL DE LA
// MARCHE DES TRAMES (ADR 0037 IR-2).
//
// # CE QUE LE FILM ECRIT
//
// L entite moteur (`game-engine-round-timer-component`) porte la limite de la manche, le temps de la minuterie
// et son etat sur 5 bits ; le jeu ne la reecrit que quand l un d eux change — la minuterie court ensuite
// d elle-meme, au rythme que l etat designe. Chaque entite joueur du moteur
// (`player-respawn-timer-component`) porte un drapeau, les secondes restantes et les secondes totales du
// compte a rebours de son joueur, reecrit a chaque seconde tant qu il court.
//
// # CE QUE CE CANAL REND, ET CE QU IL NE DIT PAS
//
// Les lectures BRUTES des deux composants, dans l ordre du flux, rendues par la couche de capture : jamais un
// octet. Une lecture de compte a rebours porte l INDEX DE JOUEUR de son entite — celui que l etat de creation
// de l entite ecrit (R(6) apres le prefixe de version), lu a la creation dans une trame delta ou, pour une
// entite deja la au debut du film, dans l etat complet d une image-cle. Les lectures d une entite qu aucun
// des deux ne nomme gardent `Joueur` a -1 : elles se comptent, elles ne se rattachent a personne.
//
// L interpretation — les periodes de compte a rebours, la remise a zero de la minuterie aux manches — est
// celle de la publication, qui rejoue depuis ces lectures.

import (
	"cmp"
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// Les etiquettes de registre des deux composants lus.
const (
	compPlayerWaypoint        = "player-waypoint-component"
	joueurDuMoteurNonRattache = -1
)

// LectureHorloge est une lecture de l horloge de manche (ti=0 i5).
type LectureHorloge struct {
	// TS est l horloge du film du paquet qui la porte, en microsecondes.
	TS uint64
	// Limite et Reste sont les deux quanta de 16 bits que le jeu ecrit sur 0 a 36000 s ([RoundTimer]) : la
	// limite de la manche et le temps de la minuterie.
	Limite, Reste uint16
	// Etat est l etat de 5 bits : l indice de la table des vitesses de la minuterie du jeu.
	Etat uint8
}

// LectureCompteARebours est une lecture du compte a rebours de reapparition d une entite joueur (ti=5 i1).
type LectureCompteARebours struct {
	// TS est l horloge du film du paquet qui la porte, en microsecondes.
	TS uint64
	// Slot est le slot de replication de l entite joueur du moteur.
	Slot uint32
	// Joueur est l index de joueur de l entite, [joueurDuMoteurNonRattache] quand aucune creation ne la nomme.
	Joueur int
	// Actif, Reste et Total sont les trois champs du composant : le drapeau, les secondes restantes, les
	// secondes totales (entiers bruts : l unite est la seconde, mesuree sur le film).
	Actif        bool
	Reste, Total uint16
}

// StatsDesHorloges sont les denominateurs du canal.
type StatsDesHorloges struct {
	// Trames : les trames delta dont la liste d evenements a ete lue ; TramesNonLues : celles dont la liste
	// d evenements n a pas ete localisee, et dont aucune lecture n a donc ete faite.
	Trames, TramesNonLues int
	// Horloges, Comptes : les lectures rendues. CompteSansJoueur : celles d une entite que rien ne nomme.
	Horloges, Comptes, CompteSansJoueur int
	// Creations : les index de joueur lus, dans une trame delta ou une image-cle.
	Creations int
}

// HorlogesLues est ce que le canal rend d un film.
type HorlogesLues struct {
	Horloges []LectureHorloge
	Comptes  []LectureCompteARebours
	Stats    StatsDesHorloges
}

// creationDeJoueur est l index de joueur qu un record de creation ou une image-cle nomme pour un slot.
type creationDeJoueur struct {
	ts    uint64
	slot  uint32
	index int
}

// canalDesHorloges est le canal de l horloge de manche et des comptes a rebours ([CanalDesTrames], et
// [CanalDImageCle] pour le nom des entites joueur).
type canalDesHorloges struct {
	reg      *Registry
	ctx      ContexteDeLecture
	m        *MarcheDistribuee
	moteur   []int
	joueurs  []int
	arch     Archetype
	archOk   bool
	creation []creationDeJoueur
	lu       HorlogesLues
}

// nouveauCanalDesHorloges prepare le canal des horloges d un film de registre `reg`, lu sous `ctx`.
func nouveauCanalDesHorloges(reg *Registry, ctx ContexteDeLecture) *canalDesHorloges {
	c := &canalDesHorloges{reg: reg, ctx: ctx}
	for ti, a := range reg.Archetypes {
		if componentIndexOfAny(a, compGameEngineRoundTimer) >= 0 {
			c.moteur = append(c.moteur, ti)
		}
		if componentIndexOfAny(a, compPlayerRespawnTimer) >= 0 {
			c.joueurs = append(c.joueurs, ti)
		}
	}
	if a, ok := reg.Archetype(archetypeJoueurDuMoteur); ok && componentIndexOfAny(a, compPlayerWaypoint) == 0 {
		c.arch, c.archOk = a, true
	}
	return c
}

// Interets : l horloge de l entite moteur et le compte a rebours de l entite joueur dans les trames, le
// premier composant de l entite joueur dans les images-cles (son etat de creation y est relu).
func (c *canalDesHorloges) Interets() []Interet {
	var out []Interet
	for _, ti := range c.moteur {
		out = append(out, Interet{Phase: PhaseTrames, TI: ti, Composant: compGameEngineRoundTimer})
	}
	for _, ti := range c.joueurs {
		out = append(out, Interet{Phase: PhaseTrames, TI: ti, Composant: compPlayerRespawnTimer})
	}
	if c.archOk {
		out = append(out, Interet{Phase: PhaseImagesCles, TI: archetypeJoueurDuMoteur, Composant: compPlayerWaypoint})
	}
	return out
}

// Brancher garde ce que la marche expose ; le canal ne pose aucun crochet.
func (c *canalDesHorloges) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// ImageCle relit l index de joueur des entites joueur du moteur que l image-cle porte.
func (c *canalDesHorloges) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	if !c.archOk {
		return
	}
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != archetypeJoueurDuMoteur {
			continue
		}
		if idx, ok := c.indexALEtendue(p, r); ok {
			c.noterLaCreation(p.TS, r.Vie.Slot, idx)
		}
	}
}

// indexALEtendue rend l index de joueur du record ti=5 `r` d une image-cle dont la phase a parcouru le
// corps : l etat par defaut se relit au premier bit du corps, et doit finir exactement ou le premier
// composant commence.
func (c *canalDesHorloges) indexALEtendue(p *lecture.Paquet, r *lecture.Record) (int, bool) {
	if r.Comps[1] == r.Comps[0] {
		return 0, false
	}
	c0 := p.Comps[r.Comps[0]]
	if c0.Index != 0 || c0.Etat == lecture.EtatInfranchissable || c0.Etat == lecture.EtatArrete {
		return 0, false
	}
	br := relecteurSur(p.Payload, c.ctx)
	cadre := br.cadre()
	br.SetBitPos(int(r.Debut) + cadre.EnTeteBits + cadre.MotDeTailleBits)
	idx := readPlayerWaypointDefaultState(br)
	attendu := br.BitPos() + cadre.MotDeTailleBits
	if c.ctx.Profil.Grammaire.ControleDeCorruption {
		attendu += cadre.MotDeTailleBits
	}
	return idx, attendu == int(c0.Debut)
}

// noterLaCreation inscrit l index de joueur d un slot a l instant `ts`.
func (c *canalDesHorloges) noterLaCreation(ts uint64, slot uint32, index int) {
	c.creation = append(c.creation, creationDeJoueur{ts: ts, slot: slot, index: index})
	c.lu.Stats.Creations++
}

// Trame lit les records de la trame : les creations d entite joueur, les horloges, les comptes a rebours.
func (c *canalDesHorloges) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		if listeAnnoncee(&p.VueA) {
			c.lu.Stats.TramesNonLues++
		}
		return
	}
	c.lu.Stats.Trames++
	for i := range recs {
		r := &recs[i]
		if r.Type == recNew && r.Trace.JoueurDuMoteurLu {
			c.noterLaCreation(p.TS, r.Slot, int(r.Trace.JoueurDuMoteur))
		}
		for _, comp := range r.Trace.Comps {
			if !comp.Ported {
				continue
			}
			if rt, ok := comp.RoundTimerOf(); ok {
				c.lu.Horloges = append(c.lu.Horloges, LectureHorloge{TS: p.TS, Limite: rt.QA, Reste: rt.QB, Etat: rt.Tail})
			}
			if rs, ok := comp.RespawnOf(); ok {
				c.lu.Comptes = append(c.lu.Comptes, LectureCompteARebours{TS: p.TS, Slot: r.Slot,
					Joueur: joueurDuMoteurNonRattache, Actif: rs.Active, Reste: rs.T0, Total: rs.T1})
			}
		}
	}
}

// Clore rattache les comptes a rebours a leur joueur et range les denominateurs.
func (c *canalDesHorloges) Clore(BilanDeMarche) {
	c.rattacher()
	c.lu.Stats.Horloges = len(c.lu.Horloges)
	c.lu.Stats.Comptes = len(c.lu.Comptes)
}

// rattacher donne a chaque lecture de compte a rebours l'index de joueur de son slot : la creation la plus
// recente a l'instant de la lecture, sinon la premiere qui la suit (une entite nee dans un chunk n'est nommee que
// par l'image-cle du chunk suivant).
func (c *canalDesHorloges) rattacher() {
	parSlot := map[uint32][]creationDeJoueur{}
	for _, cr := range c.creation {
		parSlot[cr.slot] = append(parSlot[cr.slot], cr)
	}
	for _, l := range parSlot {
		slices.SortStableFunc(l, func(a, b creationDeJoueur) int { return cmp.Compare(a.ts, b.ts) })
	}
	for i := range c.lu.Comptes {
		r := &c.lu.Comptes[i]
		if idx, ok := joueurDuSlot(parSlot[r.Slot], r.TS); ok {
			r.Joueur = idx
		} else {
			c.lu.Stats.CompteSansJoueur++
		}
	}
}

// joueurDuSlot rend l'index de joueur que les creations `l` d'un slot (triees par instant) donnent a l'instant
// `ts` : la plus recente qui le precede, sinon la premiere qui le suit.
func joueurDuSlot(l []creationDeJoueur, ts uint64) (int, bool) {
	if len(l) == 0 {
		return 0, false
	}
	k := sort.Search(len(l), func(j int) bool { return l[j].ts > ts })
	return l[max(k-1, 0)].index, true
}

// resultat rend ce que le canal a lu.
func (c *canalDesHorloges) resultat() HorlogesLues { return c.lu }
