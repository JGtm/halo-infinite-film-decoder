package grammar

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// keyframe_datums.go — LA TABLE D ENTITES QU UNE IMAGE-CLE PORTE : SES EN-TETES, RETROUVES PAR LEURS
// INVARIANTS.
//
// # CE QUE LE JEU LIT
//
// Un paquet d image-cle ouvre le morceau par la table d entites du decodeur partage, que le jeu lit
// d un bout a l autre par `FUN_142e2bfd0` (appele par `FUN_1428e2a9c`) : apres un bit d en-tete
// (versions de format superieures a 7), une entree par emplacement, DANS L ORDRE DES EMPLACEMENTS, vides
// compris, sans index ecrit dans le flux :
//
//	eid       32 bits   generation dans les deux bits de poids fort, emplacement dans les autres
//	type      32 bits   0xffffffff : emplacement vide, rien ne suit
//	poignee   32 bits   0xffffffff : aucune
//	drapeaux   4 bits   (`FUN_142e29cf8`)
//	octet      8 bits
//	si non vide : n 32 bits puis la structure de creation du type si n > 0, m 32 bits puis l etat
//
// Ces 108 bits sont l en-tete d une entree ; l entree garde son eid et son type apres la suppression
// de l entite, jusqu a la reallocation de l emplacement (un delta qui porte l ancien eid passe encore la
// garde de `FUN_1406cbaa0`, qui compare l eid ENTIER a `*(uint *)(slot * 200 + t)` et lit l archetype en
// `+4`). `FUN_142f2e174` et `FUN_142f30610`, que ce fichier citait pour cette table, ecrivent les
// ENREGISTREMENTS de la vue B (creation, suppression, mise a jour) ; ils ne l ecrivent pas.
//
// # CE QUE LE DECODEUR EN LIT : UNE RECHERCHE D EN-TETES, NOMMEE ET COMPTEE
//
// La creation et l etat d une entree n ont pas de longueur dans le flux : `n` et `m` sont des tailles de
// structure, pas des nombres de bits. L en-tete suivant ne se deduit donc pas, et lire la table comme le
// jeu demanderait de porter la creation et l etat de chaque type (hors perimetre : les donnees de creation
// et d etat des entrees ne se lisent pas ici). Le decodeur RECHERCHE les en-tetes : [EntetesDeTable]. C est
// une recuperation, pas une lecture de la marche : la liaison qu elle pose porte la provenance
// [lecture.LiaisonDatum], comptee par `DatumBindings`, et les en-tetes valides qu elle ecarte
// par `DatumAmbiguous`.
//
// # LES INVARIANTS D UN EN-TETE (mesures contre un decodeur tiers qui lit la table de bout en bout)
//
//   - emplacement sous [kfTableCap] et type sous [typeMaxDeTable] : les dix-sept bits de l eid au-dessus
//     de l emplacement sont nuls ;
//   - l octet de l en-tete n a pas ses trois bits de poids fort poses ;
//   - drapeaux a 0 : aucune poignee, ni creation ni etat (n et m nuls) ; drapeaux a 1 ou 3 : n est la
//     taille de la structure de creation, au plus [tailleMaxDeCreation] ; tout autre drapeau est refuse ;
//   - les emplacements croissent strictement le long de la table.
//
// # DEUX PASSES
//
// La premiere passe retient les entrees de generation NON NULLE : la generation 0 est la valeur que
// l allocateur pose aussi a la quatrieme allocation d un emplacement (`gen = (gen + 1) & 3`), et la
// laisser entrer partout fait entrer des coincidences de bits dans les donnees des autres entrees. La
// seconde passe ne la retient qu ENTRE DEUX ENTREES RECONNUES, pour un emplacement strictement compris
// entre leurs deux emplacements.
//
// MESURE de la recherche contre la table du decodeur tiers (5 films, 102 157 entrees) : 102 155 entrees
// justes, 2 manquees, aucune fausse. La taille de creation n est pas comparee a une taille par type : cette
// table, propre au decodeur tiers, varie d un build a l autre et n apporte rien a la mesure.
//
// HORS LIGNE — jamais depuis un chemin de requete.

// EnteteDeTable est l en-tete d une entree non vide de la table d entites d une image-cle.
type EnteteDeTable struct {
	// ID : l eid complet, la generation dans les deux bits de poids fort.
	ID uint32
	// TI : le type de l entite.
	TI uint32
}

// masqueDeSlot isole l emplacement dans un eid.
const masqueDeSlot = 0x3fffffff

// Slot rend l emplacement de l entree.
func (e EnteteDeTable) Slot() uint32 { return e.ID & masqueDeSlot }

// Gen rend la generation de l entree.
func (e EnteteDeTable) Gen() uint32 { return e.ID >> 30 }

// Les dimensions d un en-tete et de la recherche.
const (
	// teteDEntreeBits : eid, type, poignee (32 bits chacun), drapeaux (4), octet (8).
	teteDEntreeBits = 108
	// reserveDeFinBits : la recherche s arrete cette longueur avant la fin du payload, de quoi lire n et m
	// derriere le dernier en-tete essaye.
	reserveDeFinBits = 128
	// debutDeLaTableBit : le bit qui precede la premiere entree (versions de format superieures a 7).
	debutDeLaTableBit = 1
	// typeMaxDeTable : un type tient sur six bits.
	typeMaxDeTable = 64
	// poigneeAbsente : la poignee d une entree sans objet.
	poigneeAbsente = 0xffffffff
	// tailleMaxDeCreation : la taille de creation que `n` annonce ne depasse pas cette borne. MESUREE : 212 octets au
	// plus (types 41 et 44) sur les films de la mesure, de HI_1_8_0 a HI_1_13_0 ; le decodeur tiers compare `n` a une
	// taille par type, constante par type dans HI_1_12_0 et HI_1_13_0 et plus etroite de quatre octets pour sept types
	// dans HI_1_11_0 et avant : la borne ne depend d aucun build, et ne change aucun resultat sur les films de la mesure.
	tailleMaxDeCreation = 256
	// bitsHautsDeLOctet : les trois bits de poids fort de l octet de l en-tete, nuls.
	bitsHautsDeLOctet = 0xe0
)

// Les drapeaux (4 bits) que la recherche accepte.
const (
	drapeauxSansObjet    = 0
	drapeauxAvecCreation = 1
	drapeauxAvecEtat     = 3
)

// entreeSituee est un en-tete retenu et son bit.
type entreeSituee struct {
	EnteteDeTable
	bit int
}

// balayageDeTable est la recherche d en-tetes d un payload. `ecartes` recoit les emplacements dont un en-tete
// valide a ete refuse par la croissance des emplacements.
type balayageDeTable struct {
	pay     []byte
	total   int // longueur du payload en bits
	fin     int // premier decalage non essaye
	ecartes map[uint32]bool
}

func nouveauBalayageDeTable(pay []byte) *balayageDeTable {
	total := len(pay) * 8
	return &balayageDeTable{pay: pay, total: total, fin: total - teteDEntreeBits - reserveDeFinBits,
		ecartes: map[uint32]bool{}}
}

// enteteA lit l en-tete candidat au bit `at` et dit si ses invariants tiennent. `accepteGen0` : la
// generation 0 est admise (seconde passe).
func (b *balayageDeTable) enteteA(at int, accepteGen0 bool) (EnteteDeTable, bool) {
	id := uint32(source.BitsAt(b.pay, at, 32))
	if id&masqueDeSlot >= kfTableCap || (id>>30 == 0 && !accepteGen0) {
		return EnteteDeTable{}, false
	}
	ty := uint32(source.BitsAt(b.pay, at+32, 32))
	drapeaux := uint32(source.BitsAt(b.pay, at+96, 12)) // drapeaux (4 bits) puis octet (8 bits)
	if ty >= typeMaxDeTable || drapeaux&bitsHautsDeLOctet != 0 {
		return EnteteDeTable{}, false
	}
	n := uint32(source.BitsAt(b.pay, at+teteDEntreeBits, 32))
	switch drapeaux >> 8 {
	case drapeauxSansObjet:
		m := uint32(source.BitsAt(b.pay, at+teteDEntreeBits+32, 32))
		ok := uint32(source.BitsAt(b.pay, at+64, 32)) == poigneeAbsente && n == 0 && m == 0
		return EnteteDeTable{ID: id, TI: ty}, ok
	case drapeauxAvecCreation, drapeauxAvecEtat:
		return EnteteDeTable{ID: id, TI: ty}, n <= tailleMaxDeCreation
	}
	return EnteteDeTable{}, false
}

// premierePasse retient les entrees de generation non nulle, a emplacements strictement croissants.
func (b *balayageDeTable) premierePasse() []entreeSituee {
	var out []entreeSituee
	suivant := uint32(0)
	for at := debutDeLaTableBit; at < b.fin; {
		e, ok := b.enteteA(at, false)
		switch {
		case !ok:
			at++
		case e.Slot() < suivant:
			b.ecartes[e.Slot()] = true
			at++
		default:
			out = append(out, entreeSituee{e, at})
			suivant = e.Slot() + 1
			at += teteDEntreeBits
		}
	}
	return out
}

// borneDeTable est une entree reconnue qui borne un intervalle de la seconde passe : son emplacement (-1 avant la
// premiere entree, kfTableCap apres la derniere) et son bit.
type borneDeTable struct{ slot, at int }

// secondePasse rend toutes les entrees : celles de la premiere passe, et, entre deux voisines, les entrees de
// generation 0 dont l emplacement est strictement compris entre les leurs.
func (b *balayageDeTable) secondePasse(voisines []entreeSituee) []EnteteDeTable {
	out := make([]EnteteDeTable, 0, len(voisines)+16)
	prec := borneDeTable{slot: -1, at: debutDeLaTableBit - teteDEntreeBits}
	for i := 0; i <= len(voisines); i++ {
		suiv := borneDeTable{slot: kfTableCap, at: b.total}
		if i < len(voisines) {
			suiv = borneDeTable{slot: int(voisines[i].Slot()), at: voisines[i].bit}
		}
		out = append(out, b.generationZeroEntre(prec, suiv)...)
		if i < len(voisines) {
			out = append(out, voisines[i].EnteteDeTable)
			prec = suiv
		}
	}
	return out
}

// generationZeroEntre rend les entrees de generation 0 dont l emplacement est strictement compris entre ceux de
// `a` et de `c`, a emplacements croissants, dans les bits qui separent les deux voisines.
func (b *balayageDeTable) generationZeroEntre(a, c borneDeTable) []EnteteDeTable {
	var out []EnteteDeTable
	suivant := a.slot + 1
	for at := a.at + teteDEntreeBits; at < c.at-teteDEntreeBits && at < b.fin; {
		e, ok := b.enteteA(at, true)
		if !ok || e.Gen() != 0 {
			at++
			continue
		}
		if s := int(e.Slot()); s < suivant || s >= c.slot {
			b.ecartes[e.Slot()] = true
			at++
			continue
		}
		out = append(out, e)
		suivant = int(e.Slot()) + 1
		at += teteDEntreeBits
	}
	return out
}

// EntetesDeTable rend les entrees non vides de la table d entites que porte le payload d une image-cle, a
// emplacements croissants, et le nombre d emplacements dont un en-tete valide a ete ecarte par la croissance
// (cf. l en-tete du fichier).
func EntetesDeTable(pay []byte) (entrees []EnteteDeTable, ecartes int) {
	b := nouveauBalayageDeTable(pay)
	entrees = b.secondePasse(b.premierePasse())
	return entrees, len(b.ecartes)
}

// TableDeDatums rend la table `emplacement -> en-tete` que porte le payload d un paquet d image-cle, et le
// nombre d emplacements dont un en-tete valide a ete ecarte ([EntetesDeTable]).
func TableDeDatums(pay []byte) (table map[uint32]EnteteDeTable, ambigus int) {
	entrees, ambigus := EntetesDeTable(pay)
	table = make(map[uint32]EnteteDeTable, len(entrees))
	for _, e := range entrees {
		table[e.Slot()] = e
	}
	return table, ambigus
}

// LierTableDeDatums pose dans `w` les liaisons de la table d entites des paquets d image-cle d un chunk. Rend le
// nombre de liaisons posees et le nombre d emplacements ecartes (cf. [EntetesDeTable]).
//
// C est le geste qui donne au monde hors ligne le modele de la branche vive : l archetype d un slot que la chaine de
// l image-cle n a pas atteint et qu aucun NEW n a declare.
func LierTableDeDatums(w *World, data []byte, pks []FilmPacket) (posees, ambigus int) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		table, amb := TableDeDatums(pk.Payload(data))
		ambigus += amb
		posees += lierLesDatums(w, table)
	}
	return posees, ambigus
}

// lierLesDatums pose les liaisons d UNE table d entites, et rend le nombre de liaisons posees. L entree dit l eid
// et le type : une liaison deja posee qui repond de meme pour cet eid (type et generation,
// [World.ArchetypeForEID]) est gardee, elle porte davantage (vue, position) ; toute autre est remplacee.
func lierLesDatums(w *World, table map[uint32]EnteteDeTable) int {
	posees := 0
	for _, e := range table {
		if ti, lie := w.ArchetypeForEID(e.ID); lie && ti == e.TI {
			continue
		}
		w.BindEntreeDeTable(e.ID, e.TI)
		posees++
	}
	return posees
}
