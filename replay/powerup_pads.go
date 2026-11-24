package replay

// powerup_pads.go — LES SOCLES DE POWER-UP : la voie `ti=37` de la chaîne des socles.
//
// CE QUE CETTE VOIE AJOUTE, ET CE QU'ELLE NE REMPLACE PAS. La chaîne `ti=37` de production
// (`grammar.ScanFilmEquipmentPlacements` -> `equipmentPlacements`) ne retient un record de
// création que si sa position retombe sur le PREMIER POINT d'une vie décodée des paquets DELTA
// (`confirmPlacements` -> `MatchEquipmentLife`). Cet oracle reste, et il doit rester : c'est lui
// qui rend publiables les POSES d'un joueur (mur, capteur, grenades lâchées). Mais il est
// AVEUGLE aux objets de socle PAR CONSTRUCTION — un objet qui ne bouge jamais n'émet aucune
// position delta, donc n'a aucune vie sur laquelle retomber. Cette voie-ci lui est ADJOINTE,
// jamais substituée, et elle emploie le filtre que les armes au sol emploient déjà : L'IDENTITÉ.
//
// LA MESURE QUI LA FONDE (phase 3,
// 2026-08-18). Sur les deux films KOTH de Catalyst, le balayage des créations `ti=37` SANS
// l'oracle de vie delta rend NEUF (`01e1f945`) et SEPT (`75f1188f`) créations de
// `powerup_overshield` (`0xb781197a`) à la MÊME position au centimètre — (0,257 ; -0,003 ;
// 21,36) —, à 0,19 m du point que la phase 1 avait localisé par le croisement des trajectoires
// des quatre porteurs, sans lire un seul bit de record de création. Deux mesures indépendantes,
// aucun code partagé, 0,19 m d'écart. Les deux films CTF de la même carte n'en portent AUCUN :
// c'est le SOUS-MODE qui arme le socle, exactement ce que l'utilisateur décrivait.
//
// LES DEUX GARDE-FOUS QUE LA MESURE IMPOSE, et qui ne sont pas négociables :
//
//	LA GRAPPE      Le témoin FANTÔME du balayage de créations (une bande de slots jamais vus
//	               porter l'archétype, même cardinalité, même code) retient jusqu'à 5,9 créations
//	               sur `64e8adfa` : l'identité seule ne suffit donc pas partout. Ce qui a fait la
//	               preuve, c'est la RÉCURRENCE au même point (9 créations au même centimètre) —
//	               d'où le seuil `gwPadMinHits`, partagé avec les armes.
//	LA VIE DELTA   Un power-up LÂCHÉ à une mort bouge (il tombe), donc il a une vie delta. Il
//	               reste publié par `equipmentPlacements` avec son origine `dropped`, et il est
//	               écarté d'ici — un socle est un objet qui n'a JAMAIS bougé.
//
// CE QUE CETTE VOIE NE FAIT PAS : lire `t1`. Un objet sans vie delta n'en a pas. La présence se
// borne par le RECENSEMENT des images-clés, comme celle des armes (`PadPresence`).
//
// PUR (aucune I/O) : le décodage vit dans `build_powerup_pads.go`, l'assemblage dans
// `ground_weapon_pads.go`.

import "strings"

// padPowerupPrefix est le préfixe des familles d'équipement du manifeste du titre qui désignent
// un POWER-UP (`powerup_overshield`, `powerup_camo` — `replay_labels.toml`).
//
// C'EST LE MÊME LITTÉRAL QUE LA TABLE DE TAILLE DU CLIENT (`POWER_PAD_KEYS`,
// `apps/web/src/features/match-replay/weaponPadFamilies.ts`), et ce n'est pas une coïncidence :
// la FAMILLE est ce que le socle publie dans `weaponPads[].weapon`, donc c'est elle que le
// calque lit pour décider de la taille de l'icône. Un garde-rail interdit toute seconde écriture
// du littéral dans ce paquet (`powerup_pads_guard_test.go`).
const padPowerupPrefix = "powerup_"

// powerupPadRule est la règle de la voie `ti=37` : l'identité résolue dans les familles `eqip` du
// manifeste du titre, et retenue si cette famille peut se trouver sur un socle.
//
// UN SOCLE D'ÉQUIPEMENT N'EST PAS QUE LE POWER-UP. Le même archétype `ti=37` porte les objets
// d'équipement que la carte pose sur ses socles (répulseur, détecteur, mur, grappin, propulseur,
// écran occultant...). Le filtre par le préfixe `powerup_` les écartait, et le calque des socles ne
// montrait jamais un socle d'équipement. La règle retient donc toute famille du manifeste sauf
// celles qui ne se posent jamais sur un socle (cf. [padItemFamily]) et les FORMES DÉPLOYÉES que
// l'hôte désigne ([padCatalogs.DeployedEquipment]) : un mur posé par un joueur est au repos, et
// deux murs posés au même endroit feraient un faux socle.
//
// MESURE QUI LA FONDE (43 films, 28 cartes ; un lieu est juste s'il tombe à moins d'un mètre d'un
// point d'apparition ou d'un socle de la carte) : 117 socles de plus qu'avec le seul préfixe
// `powerup_`, tous justes. Quatre ne sont pas sur un point d'apparition mais sur un socle de la
// carte de type power-up : des objets de translocateur qui réapparaissent en cycle sur ce socle. Le
// film les écrit créés ATTACHÉS À L'ENTITÉ DU SOCLE (un seul parent, qui n'est pas un bipède), comme
// tout objet de socle : rien ne les distingue d'un équipement ramassé sur un autre socle, et rien ne
// permet de les écarter sans écarter les autres.
//
// AUCUNE TABLE D'OBJETS D'OBJECTIF ICI, et c'est mesuré, pas oublié : les objets qui font le
// mode (le drapeau) vivent dans l'archétype des armes au sol (`ti=42`) et portent des
// identifiants du catalogue `weap` ; le catalogue `eqip` n'en contient aucun.
func powerupPadRule(cat padCatalogs) padRule {
	return padRule{
		Kind: gwPadKindPowerup,
		Family: func(id uint32) (string, bool) {
			f := cat.EquipmentFamilies[id]
			if !padItemFamily(f) || cat.DeployedEquipment[id] {
				return "", false
			}
			return f, true
		},
	}
}

// padItemFamily dit qu'une famille `eqip` du manifeste peut se trouver sur un socle : toute famille
// nommée, sauf la grenade (`grenade_*`), qui gît au sol à un point d'apparition de grenade et non
// sur un socle.
func padItemFamily(f string) bool {
	return f != "" && !strings.HasPrefix(f, padGrenadePrefix)
}

// padGrenadePrefix est le préfixe des familles de grenade du manifeste (`grenade_frag`, ...).
const padGrenadePrefix = "grenade_"

// padCatalogs porte les tables d'identité que la chaîne des socles consomme, une par voie. Elles
// voyagent ensemble parce que l'assemblage les prend ensemble (règle des 5 paramètres).
type padCatalogs struct {
	// ObjectiveObjects est la table GlobalID `weap` -> NOM du drapeau : les objets d'objectif que la
	// voie des ARMES doit reconnaître pour NE PAS les publier comme des socles.
	ObjectiveObjects map[uint32]Label
	// EquipmentFamilies est la table GlobalID `eqip` -> FAMILLE du manifeste. C'est elle qui
	// fait toute la sélectivité de la voie `ti=37` : sans elle, aucune création ne se résout, et le
	// calque publie zéro socle de ce genre en silence (le compteur `powerupKept` de la couverture
	// est le témoin qui rompt ce silence).
	EquipmentFamilies map[uint32]string
	// DeployedEquipment est l'ensemble des GlobalID `eqip` qui sont la forme POSÉE d'un objet (un
	// panneau de mur déployé) : ils ne sont jamais sur un socle. Fourni par l'hôte, comme
	// EquipmentFamilies. Vide = aucune forme déployée connue.
	DeployedEquipment map[uint32]bool
}

// PadScans porte les DEUX lectures du film que la chaîne des socles consomme : les ARMES AU SOL
// (`ti=42`) et les objets d'ÉQUIPEMENT du monde (`ti=37`, d'où sortent les power-ups de socle).
//
// UNE ENTRÉE DE DONNÉES, comme les poses : `Scanned` faux d'un côté n'empêche pas l'autre de
// publier. Les deux voies aboutissent au MÊME tableau `weaponPads` — le socle est le même objet
// de jeu, et un second calque aurait dupliqué trois types publiés pour la même grandeur.
type PadScans struct {
	Weapons  WorldObjectScan
	Powerups WorldObjectScan
}
