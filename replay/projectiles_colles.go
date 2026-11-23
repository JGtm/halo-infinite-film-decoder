package replay

// projectiles_colles.go : LE PROJECTILE QUI SE COLLE, PUBLIE (`projectiles[].attach`).
//
// Un projectile (grenade, trait d arme, arc de grenade...) qui se fixe sur une entite l ecrit dans son etat de parente. Son record
// nomme alors ce sur quoi il est colle ([types.ProjectileAttachRead]) : le PARENT (dans le meme espace de
// slots que les pistes), un NOEUD de ce parent, et trois valeurs de 16 bits qui placent le projectile dans le
// repere de ce noeud. Le document publie cette lecture, rien de plus.
//
// LA POSE SE PUBLIE BRUTE. Les trois valeurs couvrent toute la plage de 0 a 65 535 et ne forment pas un vecteur
// unitaire : le film ne donne ni leur plage ni leur sens, et aucune constante tiree d un client ne s y
// substitue. Un lecteur qui connait le squelette du parent les interprete.
//
// LE PARENT SE RETROUVE PAR LES PISTES DEJA PUBLIEES, pas par une seconde voie : `parentSlot` est le slot d une
// piste de joueur (qui tient la place a la frame `fromFrame`, comme `damage[].slot`) ou d un vehicule
// (`vehicles[].slot`, meme generation, meme intervalle de vie). Un parent qui n est ni l un ni l autre (une arme
// au sol, un objet de Forge, un equipement) est publie quand meme, et compte a part.
//
// LE COLLAGE EST POSE SUR LE VOL QUI L A PRECEDE : le projectile publie dont la cle (slot, generation) est celle de la
// lecture, que la lecture suit apres son premier echantillon et au plus tard a sa detonation. Une cle se reutilise (la
// generation tient sur deux bits) : la fenetre de vie du vol departage. Un projectile qui nait colle, a bout portant,
// n a replique aucun vol : il n a pas de projectile publie pour porter son collage, et se compte.

import (
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ProjectileAttach est ou se tient un projectile colle, a partir de FromFrame et jusqu a sa detonation.
type ProjectileAttach struct {
	// FromFrame est la frame de la lecture, sur le meme axe que Point.T.
	FromFrame int `json:"fromFrame"`
	// ParentSlot est le slot de l objet sur lequel le projectile est colle : la piste d un joueur (`tracks[].slot`)
	// ou un vehicule (`vehicles[].slot`) quand l un d eux tient ce slot a FromFrame ; une arme, un objet ou un
	// equipement sinon.
	ParentSlot uint32 `json:"parentSlot"`
	// Node est le noeud du parent, brut : l index d un noeud de son modele de rendu.
	Node uint32 `json:"node"`
	// Pose est la position du projectile dans le repere de ce noeud : trois valeurs de 16 bits, BRUTES, dans l ordre du film.
	// Le film ne dit ni leur plage ni leur unite.
	Pose [3]uint32 `json:"pose"`
}

// ProjectileAttachInput est ce que l assemblage recoit des projectiles colles : les lectures du film et ce que leur canal a vu.
type ProjectileAttachInput struct {
	Reads []types.ProjectileAttachRead
	Stats types.ProjectileAttachStats
}

// ProjectileAttachCoverage dit ce que le calque a lu, pose et ecarte. Chaque lecture compte dans UNE de ces trois classes : hors du
// match, posee sur un vol publie (`projectiles[].attach`), ou publiee dans `attachments[]` (`outOfMatch + onFlight + listed == read`) ;
// une lecture posee ou publiee compte dans UNE des trois cibles (`player + vehicle + unresolved == onFlight + listed`).
type ProjectileAttachCoverage struct {
	// Records : les records de projectile dont l etat de parente a ete lu jusqu au bout ; Read : ceux qui ouvrent la porte
	// « attache », donc les lectures ; Created : parmi elles, celles d un projectile qui nait colle.
	Records int `json:"records"`
	Read    int `json:"read"`
	Created int `json:"created"`
	// Desynced : les records dont l etat de parente a ete lu avant un desalignement, ecartes. UnreadFrames : les trames dont la liste
	// de records est annoncee et reste illisible.
	Desynced     int `json:"desynced"`
	UnreadFrames int `json:"unreadFrames"`
	// WideLocations : les listes de records que la marche n avait pas localisees et que le canal a recuperees a largeur libre.
	WideLocations int `json:"wideLocations"`
	// OutOfMatch : les lectures avant l origine de l axe ou apres sa derniere frame.
	OutOfMatch int `json:"outOfMatch"`
	// OnFlight : les lectures posees sur un projectile publie, dont le vol se termine par elles. Listed : les lectures qu aucun projectile
	// publie ne porte (le projectile nait colle, son vol n est pas publie, ou la lecture tombe hors de sa fenetre de vie), publiees dans
	// `attachments[]`.
	OnFlight int `json:"onFlight"`
	Listed   int `json:"listed"`
	// Player, Vehicle, Unresolved : parmi les lectures posees ou publiees, celles dont le parent est une piste de joueur ou un vehicule a la
	// frame de la lecture, et celles dont il n est ni l un ni l autre (sans cible resolue).
	Player     int `json:"player"`
	Vehicle    int `json:"vehicle"`
	Unresolved int `json:"unresolved"`
	// Tagged : parmi les publiees dans `attachments[]`, celles dont le type du projectile est lu. Authored : celles dont le createur est lu et
	// tient une piste a cette frame (`by`) ; AuthorNoTrack : celles dont le createur est lu et ne tient aucune piste ; AuthorNone : celles dont
	// la creation ne nomme pas de createur (ou n est pas lue).
	Tagged        int `json:"tagged"`
	Authored      int `json:"authored"`
	AuthorNoTrack int `json:"authorNoTrack"`
	AuthorNone    int `json:"authorNone"`
}

// cibleDuCollage est ce que le parent d un collage est dans le document.
type cibleDuCollage uint8

const (
	cibleNonResolue cibleDuCollage = iota
	cibleJoueur
	cibleVehicule
)

// cibleDuParent dit ce qu est le parent `p` a la frame `t` : une piste de joueur, par la meme resolution que les degats
// ([resolveurDeTouches.victime]), un vehicule de meme vie qui est la a cette frame, ou rien de publie.
func cibleDuParent(p types.LifeKey, t int, pistes []Track, vehicules []VehicleTrack) cibleDuCollage {
	if unite := p.Slot - decalageDesSlotsDUnite; p.Slot >= decalageDesSlotsDUnite && unite <= 0xFFFF {
		if _, piste := (resolveurDeTouches{tracks: pistes}).victime(uint16(unite), t); piste {
			return cibleJoueur
		}
	}
	for _, v := range vehicules {
		if v.Slot == p.Slot && v.Gen == p.Gen && t >= v.T0 && t <= v.T1Max {
			return cibleVehicule
		}
	}
	return cibleNonResolue
}

// lecturesParVie range les indices des lectures par vie de projectile, dans l ordre du flux.
func lecturesParVie(lectures []types.ProjectileAttachRead) map[types.LifeKey][]int {
	out := make(map[types.LifeKey][]int, len(lectures))
	for i, l := range lectures {
		out[l.Life] = append(out[l.Life], i)
	}
	return out
}

// premiereLectureDuVol rend l indice de la premiere lecture `libre` de la vie du vol `tr` qui suit son premier echantillon et tombe au
// plus tard a la frame de sa detonation `detoT`, ou -1. `frames[i]` est la frame de la lecture `i`. Une lecture dont la creation est lue
// n appartient au vol que si celui-ci commence apres cette creation : une cle se reutilise.
func premiereLectureDuVol(tr types.ProjectileTrack, detoT int, indices []int, lectures []types.ProjectileAttachRead,
	frames []int, prises []bool) int {
	debut := tr.Pts[0].TimestampUS
	for _, i := range indices {
		l := lectures[i]
		if !prises[i] && l.TimestampUS >= debut && (!l.HasBirth || l.BornUS <= debut) && frames[i] <= detoT {
			return i
		}
	}
	return -1
}

// compterLaCible ajoute une lecture de cible `cible` au compte de sa cible dans la couverture.
func (cov *ProjectileAttachCoverage) compterLaCible(cible cibleDuCollage) {
	switch cible {
	case cibleJoueur:
		cov.Player++
	case cibleVehicule:
		cov.Vehicle++
	default:
		cov.Unresolved++
	}
}

// poserLesProjectilesColles pose `attach` sur les projectiles publies dont le vol se termine par un collage, publie les autres lectures
// dans `attachments[]`, et pose la couverture du calque. Sans lecture des projectiles colles (marche des trames non jouee, archetype sans
// `i10`) ou sans piste de projectile, rien ne se pose et la couverture reste absente.
func (a *assemblage) poserLesProjectilesColles() {
	in := a.opt.ProjectileAttaches
	if a.projCov == nil || !in.Stats.Scanned || in.Stats.Absent {
		return
	}
	st := in.Stats
	cov := ProjectileAttachCoverage{Records: st.Records, Read: st.Read, Created: st.Created,
		Desynced: st.Desynced, UnreadFrames: st.UnreadFrames, WideLocations: st.WideLocations}
	cadre := cadreDesCollages{pistes: a.doc.Tracks, vehicules: a.doc.Vehicles}
	axe := axeDeFrames{a.origin, a.step, a.doc.FrameCount}
	frames := make([]int, len(in.Reads))
	prises := make([]bool, len(in.Reads))
	for i, l := range in.Reads {
		t, refus := frameDeLAction(l.TimestampUS, axe)
		frames[i] = t
		if refus != refusAucun {
			prises[i] = true // hors de l axe : aucun vol ne la prend, aucune frame ne la porte
			cov.OutOfMatch++
		}
	}
	parVie := lecturesParVie(in.Reads)
	for brut, tr := range a.opt.Projectiles {
		rang, publie := a.projParPiste[brut]
		if !publie || len(tr.Pts) == 0 {
			continue
		}
		p := &a.doc.Projectiles[rang]
		i := premiereLectureDuVol(tr, p.DetoT, parVie[types.LifeKey{Slot: tr.Slot, Gen: tr.Gen}], in.Reads, frames, prises)
		if i < 0 {
			continue
		}
		prises[i] = true
		l := in.Reads[i]
		p.Attach = &ProjectileAttach{FromFrame: frames[i], ParentSlot: l.Parent.Slot, Node: l.Node, Pose: l.Pose}
		cov.OnFlight++
		cov.compterLaCible(cibleDuParent(l.Parent, frames[i], cadre.pistes, cadre.vehicules))
	}
	a.doc.Attachments = publierLesCollagesSansVol(in.Reads, frames, prises, cadre, &cov)
	a.projCov.Attach = &cov
}
