package grammar

// vue_a_charges_mesurees.go — LES TROIS CHARGES DE LA VUE A DONT LA DISPOSITION EST MESUREE, ET NON
// LUE DANS LE JEU.
//
// Les lecteurs de `biped_melee_clang` (genre 12, `FUN_142ef8e08`), de `PlayEffectOnObject` (genre
// 14, `FUN_142eebf08`) et de `ObjectCollisionDamage` (genre 106, `FUN_14112134c`) sont ceux que la
// table des descripteurs designe ([tableDesGenresVueA], `vue_a_genres_test.go`) ; aucun n a ete
// relu dans l executable. Leur disposition vient du decodeur tiers libfilmshell (`events.rs`) et
// elle est CONFIRMEE PAR LA LARGEUR MESUREE du message entier — du bit de continuation au bit qui
// precede le message suivant — contre le decodeur tiers, sur des films HI_1_12_0 :
//
//	genre  message               largeur  disposition (apres la tete et ses references)
//	12     biped_melee_clang     112      joueur, joueur, tag, position a precision 12, direction R(19)
//	14     PlayEffectOnObject    53       R(32)
//	106    ObjectCollisionDamage 82       position a precision 12, R(10) direction, R(9) index
//
// La position est lue par le portage unique de `FUN_14076e494` ([lireE494Sur]) au niveau
// [niveauDeChoc] : l immediat 0xc est celui que la largeur mesuree impose (les largeurs de la
// region jouee, moins quatre bits par axe), pas un immediat releve a un site du jeu — la table des
// sites de position (`lecteur_position_ratchet_test.go`) le dit. Une charge relue dans le jeu
// remplace sa ligne ici et dans cette table.

// niveauDeChoc est le niveau de precision de la position des genres 12 et 106 : 0xc, MESURE.
const niveauDeChoc = 0xc

// Les largeurs de la charge de `ObjectCollisionDamage` (genre 106) et de `PlayEffectOnObject`
// (genre 14).
const (
	largeurDirectionDeCollision = 10
	largeurIndexDeCollision     = 9
	largeurEffetSurObjet        = 32
)

// chargeMesuree rend le lecteur de charge d un genre de disposition mesuree ; nil sinon.
func chargeMesuree(genre int) func(*Lecteur) bool {
	switch genre {
	case 12:
		return chargeChocDeMelee
	case 14:
		return chargeEffetSurObjet
	case 106:
		return chargeCollisionDObjet
	}
	return nil
}

// chargeChocDeMelee porte `biped_melee_clang` (genre 12) : deux joueurs (R(1) a 0 puis R(5)), un
// tag (R(1) puis R(32) si 1), une position, la direction R(19).
func chargeChocDeMelee(br *Lecteur) bool {
	consumeGate0R(br, 5)
	consumeGate0R(br, 5)
	consumeGateR(br, 32)
	if _, ok := lireE494Sur(br, niveauDeChoc, br.vueA.positions); !ok {
		return false
	}
	br.Skip(largeurDirection)
	return true
}

// chargeEffetSurObjet porte `PlayEffectOnObject` (genre 14) : R(32).
func chargeEffetSurObjet(br *Lecteur) bool {
	br.Skip(largeurEffetSurObjet)
	return true
}

// chargeCollisionDObjet porte `ObjectCollisionDamage` (genre 106) : une position, une direction
// R(10), un index R(9).
func chargeCollisionDObjet(br *Lecteur) bool {
	if _, ok := lireE494Sur(br, niveauDeChoc, br.vueA.positions); !ok {
		return false
	}
	br.Skip(largeurDirectionDeCollision + largeurIndexDeCollision)
	return true
}
