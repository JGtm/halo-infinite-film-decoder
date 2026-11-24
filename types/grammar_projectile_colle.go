package types

// grammar_projectile_colle.go : LE PROJECTILE QUI SE COLLE, lu dans les trames delta (composant
// `i10 object-parent-state`, branche « attache »).
//
// Un projectile (grenade, trait d arme, arc de grenade...) qui se fixe sur une entite l ecrit dans
// son etat de parente : le record nomme ce sur quoi il est colle (le parent), un noeud de ce parent
// et trois valeurs de 16 bits qui le placent dans le repere de ce noeud. Ces trois valeurs sont rendues BRUTES : leur plage n est pas connue (elles couvrent toute
// la plage de 0 a 65 535 dans les films et ne forment pas un vecteur unitaire).

// ProjectileAttachRead est UNE lecture de collage : le record d un projectile (creation ou delta) dont l etat de
// parente ouvre la porte « attache ».
type ProjectileAttachRead struct {
	// TimestampUS : l horodatage du paquet porteur, sur la meme horloge que ProjectileSample.TimestampUS.
	TimestampUS uint64
	// Life : le projectile (slot, generation), la cle de sa ProjectileTrack.
	Life LifeKey
	// BornUS : l instant de la creation de cette vie du projectile, le dernier record de creation de la cle (slot, generation) que le
	// canal a lu avant la lecture ; egal a TimestampUS quand la lecture est celle de la creation. Zero sans creation lue (HasBirth faux).
	BornUS   uint64
	HasBirth bool
	// Tag est le mot de 32 bits du bloc de proprietes de la creation du projectile (l identifiant de son tag `proj`, avant le decalage d un bit
	// qui en fait un identifiant global) : le TYPE de l objet, tel que le film l ecrit. HasTag est faux sans creation lue, ou quand le film
	// n a pas de deserialiseur d etat de creation du projectile.
	Tag    uint32
	HasTag bool
	// Creator : l unite qui a cree le projectile, la reference de categorie 1 de son etat de creation (meme espace de slots que Parent). Le
	// film la porte derriere une porte : HasCreator faux quand elle est fermee.
	Creator    LifeKey
	HasCreator bool
	// Created : le record est la CREATION du projectile (un record NEW) : il nait deja colle, a bout portant, sans
	// avoir replique de vol. Faux : un record delta, le collage d un projectile qui avait vole.
	Created bool
	// Parent : l objet sur lequel il est colle. Le slot est celui du handle plus la base de sa
	// categorie, donc comparable a BipedPosition.Slot ; la generation distingue les vies du slot.
	Parent LifeKey
	// Node : le R(16) qui suit le handle du parent, brut. C est l index d un noeud du modele de rendu
	// du parent ; le film ne dit pas lequel.
	Node uint32
	// Pose : les trois R(16) de la branche « attache », bruts, dans l ordre ou le film les ecrit.
	Pose [3]uint32
}

// ProjectileAttachStats sont les denominateurs de la lecture des projectiles colles.
type ProjectileAttachStats struct {
	// Scanned : la marche des trames a ete jouee et son canal clos ; faux = aucune lecture n a eu lieu,
	// ce qui n est pas « zero collage ».
	Scanned bool
	// Absent : l archetype projectile du film ne declare pas `i10` ; rien n a pu etre lu.
	Absent bool
	// Frames : les trames delta dont les records ont ete lus ; UnreadFrames : celles dont la liste est
	// annoncee et dont les records restent illisibles.
	Frames, UnreadFrames int
	// Records : les records de projectile (creation ou delta) parcourus jusqu au bout dont l etat de parente a ete lu ;
	// Read : parmi eux, ceux dont la porte « attache » est ouverte, donc les lectures retenues ; Created : parmi ces
	// dernieres, celles d un record de creation.
	Records, Read, Created int
	// Desynced : les records de projectile dont l etat de parente a ete lu avant un desalignement du
	// record, et que la lecture ecarte : rien ne prouve l alignement de ce qu ils portent.
	Desynced int
	// Repeats : les records « attache » d une vie qui avait deja un collage lu ; le canal ne rend que le premier collage de chaque vie, une vie
	// finissant a la suppression de son slot ou a la creation suivante sous ce slot. Ils ne comptent pas dans Read.
	Repeats int
	// NoBirth : les lectures dont la VIE n avait aucun record de creation lu (la creation precede le film, ou n a pas ete lue jusqu au bout) ;
	// NoTag : celles dont le tag du projectile n est pas lu (creation absente, ou film sans deserialiseur d etat de creation) ; NoCreator :
	// celles dont la creation ne nomme pas de createur.
	NoBirth, NoTag, NoCreator int
	// WideLocations : les listes que la marche n avait pas localisees et que le repli a largeur libre a
	// recuperees pour ce canal.
	WideLocations int
}
