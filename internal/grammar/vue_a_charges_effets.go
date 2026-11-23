package grammar

// vue_a_charges_effets.go — QUATRE CHARGES COURTES DE LA VUE A QUI ARRETAIENT LA LECTURE D UNE TRAME :
// `motor_system_interruption` (13), `controlToggleResponse` (27), `activate_spartan_ability` (93) et
// `EquipmentTranslocatorTeleportEffects` (117).
//
// Avant ces charges, le premier message de l un de ces genres dans une trame arretait la lecture de la
// vue A : les messages suivants de la trame etaient perdus et le rejeu se declarait incomplet. Les
// valeurs lues se TRAVERSENT : aucun champ n en est publie, aucun lecteur ne les attend.
//
// Les largeurs des genres 13, 27 et 117 sont celles de la bibliotheque libfilmshell de Den (OpenSpartan,
// MIT), TEMOIN de cette lecture : `src/events.rs`, au `match` du type d evenement. Le temoin ne lit pas le
// genre 93 : seule sa charge inhibee, la seule que les films montrent, se lit. Dans tous les cas la preuve
// est le film : une charge a la bonne largeur laisse la trame se poursuivre jusqu a sa fin (la trame se
// ferme, ses vues B et C se lisent jusqu au bout).

// largeurInterruptionLarge et largeurInterruptionCourt : le R(7) du cas large de `motor_system_interruption`, et le R(2) du cas court.
const (
	largeurInterruptionLarge = 7
	largeurInterruptionCourt = 2
)

// chargeInterruptionDuMoteur porte `motor_system_interruption` (genre 13, lecteur `0x142ef8f44`, qui
// appelle `0x142ef47c8`) : R(1) « large » ; si 1 : R(7), sinon R(2) ; puis un R(1). Les films ne montrent que
// le cas large (R(7) et R(1) bornes par la mesure) ; le cas court est celui du temoin, jamais vu sur film.
func chargeInterruptionDuMoteur(br *Lecteur) bool {
	if br.ReadBit() {
		br.Skip(largeurInterruptionLarge)
	} else {
		br.Skip(largeurInterruptionCourt)
	}
	br.Skip(1)
	return true
}

// largeurBasculeDeControle est le R(2) de `controlToggleResponse`.
const largeurBasculeDeControle = 2

// chargeReponseDeBascule porte `controlToggleResponse` (genre 27, lecteur `0x142f16de8`) : R(2).
func chargeReponseDeBascule(br *Lecteur) bool {
	br.Skip(largeurBasculeDeControle)
	return true
}

// largeurCapaciteEnTete est la largeur des bits qui precedent le bit d inhibition de
// `activate_spartan_ability` (genre 93, lecteur `0x142ef8d94`).
const largeurCapaciteEnTete = 15

// chargeActivationDeCapacite porte `activate_spartan_ability` (genre 93) : quinze bits, puis un R(1)
// d inhibition. Seule la charge INHIBEE se lit : c est la seule que les films montrent (33 messages sur
// 7 films, tous inhibes ; la largeur de l en-tete est bornee par la mesure : quatorze bits arretent des
// trames). Sans inhibition le lecteur lit un R(2) de branche puis un bloc (`FUN_142f262d4` pour l une
// d elles) que la grammaire n a vu sur aucun film : la charge rend faux, et la lecture s arrete comme
// avant sur ce seul cas, sans rien deviner.
func chargeActivationDeCapacite(br *Lecteur) bool {
	br.Skip(largeurCapaciteEnTete)
	return br.ReadBit()
}

// chargeTeleportationDuTranslocateur porte `EquipmentTranslocatorTeleportEffects` (genre 117, lecteur
// `0x140f04fb8`) : l identifiant d effet garde (`FUN_14080d69c` : R(1), puis R(32)), puis les deux
// positions du va-et-vient, chacune sous la garde de pleine precision (`FUN_14076e494`, niveau 0x10, CALLs
// `140f04ff0` et `140f05023`). Les films ne montrent que l identifiant d effet present ; son absence est la
// porte ordinaire de la primitive. Une position illisible (index de region hors des tables de la carte) rend
// faux. `transloc_events.go` lit la meme charge depuis la tete d un paquet, avec les bornes de la carte.
func chargeTeleportationDuTranslocateur(br *Lecteur) bool {
	consumeGateR(br, 32)
	for range 2 {
		if _, ok := lireE494Sur(br, niveauPosition, br.vueA.positions); !ok {
			return false
		}
	}
	return true
}
