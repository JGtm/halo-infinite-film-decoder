package grammar

// canal_des_projectiles_colles.go : LE PROJECTILE QUI SE COLLE, UN CANAL DE LA MARCHE DES TRAMES.
//
// # CE QUE LE FILM ECRIT
//
// Un projectile (grenade, trait d arme, arc de grenade...) qui se fixe sur une entite l ecrit dans
// `object-parent-state-component` (i10), porte « attache » ouverte : le record capture d un collage (les tests de
// ce canal) porte i10 et `projectile-at-rest-state` (i18), sans position (i0). La branche « attache » de i10 nomme le
// PARENT (le handle de la categorie 1, de base [parentHandleBase] : celui que lisent deja l occupation des
// vehicules et l equipement porte), un noeud de ce parent, et trois valeurs de 16 bits qui placent le
// projectile dans le repere de ce noeud. Un projectile lance a bout portant peut aussi NAITRE colle : son record de
// creation ouvre alors la meme porte. Le film peut ecrire d autres records de la meme vie apres un collage (un i10
// libre, un detachement) : ce canal rend chaque collage lu, il ne suit pas la vie.
//
// # CE QUE CE CANAL REND, ET CE QU IL NE DIT PAS
//
// Les lectures BRUTES, dans l ordre du flux : le parent, le noeud et les trois valeurs, jamais interpretees.
// Aucune plage n est connue pour ces trois valeurs (cf. [types.ProjectileAttachRead]) ; ce qu elles valent en
// unites du monde est le travail d un lecteur qui connait le squelette du parent, pas celui du decodeur.
// Chaque lecture porte aussi ce que la CREATION de sa vie dit du projectile : son instant, son type (le mot de
// 32 bits de son tag) et son createur ([canal_des_projectiles_colles_naissance.go]). Le canal ne dit pas quel
// tir ou quelle arme a produit le projectile : le film ne l ecrit pas.
//
// # LES LARGEURS SONT CELLES DE i10, ET LE FILM LES PROUVE
//
// La lecture de la branche « attache » est celle que le deserialiseur de i10 applique a tout archetype
// ([decodeObjectParentState]) ; ce canal n en ajoute aucune. Il ne retient qu un record parcouru JUSQU AU BOUT
// (`DesyncAt == -1`) : un record lu a la bonne largeur laisse la suite de la trame se lire, et celui qui
// s est desaligne avant sa fin est ecarte et compte ([types.ProjectileAttachStats.Desynced]).
//
// Le canal ne marche rien lui-meme, ne lance aucun fil et ne tient qu une table des vies deja vues : la
// sortie ne depend ni du nombre de fils ni des voies.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// canalDesProjectilesColles est le canal des projectiles colles de la marche des trames ([CanalDesTrames]).
type canalDesProjectilesColles struct {
	m    *MarcheDistribuee
	st   types.ProjectileAttachStats
	out  []types.ProjectileAttachRead
	vues map[types.LifeKey]struct{}
	// id relit l etat de creation de chaque projectile ; nil quand le profil du film ne le deroule pas. naissances tient la creation
	// courante de chaque cle (slot, generation) : la derniere lue.
	id         *lecteurDeNaissance
	naissances map[types.LifeKey]naissanceDeProjectile
}

// nouveauCanalDesProjectilesColles prepare le canal du film `fc` dont le registre est `reg` : un archetype
// projectile qui ne declare pas `i10` ne transmet aucun collage, ce que `Absent` publie.
func nouveauCanalDesProjectilesColles(fc *FilmContext, reg *Registry) *canalDesProjectilesColles {
	c := &canalDesProjectilesColles{vues: map[types.LifeKey]struct{}{}, naissances: map[types.LifeKey]naissanceDeProjectile{},
		id: nouveauLecteurDeNaissance(fc)}
	arch, ok := reg.Archetype(ProjectileTypeIndex)
	c.st.Absent = !ok || len(arch.indicesOf(compObjectParentState)) == 0
	return c
}

// Interets : l `i10` du projectile, dans les trames ; rien pour un film dont l archetype ne le declare pas.
func (c *canalDesProjectilesColles) Interets() []Interet {
	if c.st.Absent {
		return nil
	}
	return []Interet{{Phase: PhaseTrames, TI: ProjectileTypeIndex, Composant: compObjectParentState}}
}

// Brancher garde ce que la marche expose ; le canal ne pose aucun crochet.
func (c *canalDesProjectilesColles) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Clore trie les lectures par instant puis par projectile : l ordre du flux, rendu total.
func (c *canalDesProjectilesColles) Clore(BilanDeMarche) {
	c.st.Scanned = true
	slices.SortStableFunc(c.out, func(a, b types.ProjectileAttachRead) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Life.Slot, b.Life.Slot),
			cmp.Compare(a.Life.Gen, b.Life.Gen))
	})
	c.vues, c.naissances = nil, nil
}

// Trame lit les records delta de projectile de la trame `p`. Une liste que la marche n a pas localisee se
// relit comme le fait le canal des morts ([MarcheDistribuee.recupererLaListe]) ; si elle reste illisible, la
// trame ne dit rien des projectiles et se compte.
func (c *canalDesProjectilesColles) Trame(p *lecture.Paquet) {
	if c.st.Absent {
		return
	}
	annoncee := listeAnnoncee(&p.VueA)
	recs, lus := c.m.recordsDeLaTrame()
	if !lus && annoncee {
		var libre bool
		recs, lus, libre = c.m.recupererLaListe()
		c.st.WideLocations += unSi(libre)
	}
	if !lus {
		if annoncee && p.VueA.Etat != lecture.VueArretee {
			c.st.UnreadFrames++
		}
		return
	}
	c.st.Frames++
	for i := range recs {
		c.lire(p, &recs[i])
	}
}

// lire range la lecture du record `r` s il est la creation ou le delta d un projectile dont i10 ouvre la porte
// « attache ». Une creation qui s ouvre sur cette porte est un projectile qui nait colle, a bout portant. Toute
// creation lue jusqu au bout est retenue, collee ou non : c est elle qui dit le type et le createur de la vie.
func (c *canalDesProjectilesColles) lire(p *lecture.Paquet, r *FrameRecord) {
	if r.TypeIndex != ProjectileTypeIndex || (r.Type != recDelta && r.Type != recNew) {
		return
	}
	vie := types.LifeKey{Slot: r.Slot, Gen: r.ID >> 30}
	if r.Type == recNew && r.DesyncAt == -1 {
		c.naissances[vie] = c.id.lire(p, r)
	}
	st, ok := parentDuRecord(r)
	if !ok {
		return
	}
	if r.DesyncAt != -1 {
		c.st.Desynced++
		return
	}
	c.st.Records++
	if !st.Attached {
		return
	}
	if _, deja := c.vues[vie]; deja {
		c.st.Repeats++
	}
	c.vues[vie] = struct{}{}
	c.st.Read++
	c.st.Created += unSi(r.Type == recNew)
	l := types.ProjectileAttachRead{TimestampUS: p.TS, Life: vie, Created: r.Type == recNew,
		Parent: types.LifeKey{Slot: (st.Quant16 & parentHandleValueMask) + parentHandleBase, Gen: st.Quant16 >> 30},
		Node:   st.Word16, Pose: st.Mtx}
	if n, ok := c.naissances[vie]; ok {
		l.BornUS, l.HasBirth = n.us, true
		l.Tag, l.HasTag = n.tag, n.aTag
		l.Creator, l.HasCreator = n.createur, n.aCreateur
	}
	c.st.NoBirth += unSi(!l.HasBirth)
	c.st.NoTag += unSi(!l.HasTag)
	c.st.NoCreator += unSi(!l.HasCreator)
	c.out = append(c.out, l)
}
