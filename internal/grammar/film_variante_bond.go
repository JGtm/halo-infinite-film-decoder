package grammar

// film_variante_bond.go — LA LECTURE D UN ARBRE BOND PAR CHEMINS, POUR LES INFOS DE MATCH.
//
// Le corps de `chunk_00` porte la variante de partie en Microsoft Bond CompactBinary de version 2
// (les ecrivains sont cites dans `film_variante_de_partie.go`). Cette lecture parcourt un message
// de bout en bout et ne rend que les valeurs dont le chemin est demande ; le reste est enjambe par sa
// longueur (une structure), son compte (une liste d octets) ou lu sans etre rendu. Elle partage les
// primitives du curseur [lecteurDuCorps] (entete de champ, entier variable) avec la marche des trois
// champs de [lireLaVarianteDePartie], dont elle ne reprend pas les regles d arret : une valeur d un
// type inconnu, une longueur de structure qui ne tombe pas sur son octet de fin, un compte qui ne tient
// pas dans le tampon ou un message trop profond font tomber le curseur (`ok` faux) et rien n est rendu.
//
// UN CHEMIN est la suite des id de champ depuis la racine ; l element d une liste ajoute son rang a
// la suite. Les types Bond lus sont ceux de la specification : booleen 2, entiers non signes 3 a 6,
// flottants 7 et 8, chaine 9, structure 10, liste 11, ensemble 12, table 13, entiers signes 14 a 17,
// chaine large 18. Une liste d octets (element de type 3) est rendue d un bloc.

// Les types Bond que la lecture par chemins reconnait, en plus de ceux de `film_variante_de_partie.go`.
const (
	bondOctet         = 0x3
	bondEntierNS16    = 0x4
	bondEntierNS64    = 0x6
	bondFlottant32    = 0x7
	bondFlottant64    = 0x8
	bondChaine        = 0x9
	bondEnsemble      = 0xC
	bondTable         = 0xD
	bondOctetSigne    = 0xE
	bondEntierS16     = 0xF
	bondEntierS64     = 0x11
	bondChaineLarge   = 0x12
	profondeurBondMax = 32 // les variantes mesurees emboitent quelques niveaux
	octetsParEntier64 = 10 // un entier variable de 64 bits tient sur dix octets au plus
)

// visiteBond dit a la lecture ce qui l interesse et ou rendre les valeurs. Les chemins passes aux
// fonctions ne sont valables que le temps de l appel.
type visiteBond struct {
	// suivre est appelee une fois par valeur rencontree : vrai pour la lire (une structure ou une liste :
	// en descendre), faux pour l enjamber.
	suivre func(chemin []int) bool
	// feuille recoit une valeur suivie : l entier d un booleen, d un octet ou d un entier (signe decode),
	// ou les octets d une chaine ou d une liste d octets. Un flottant n est pas rendu.
	feuille func(chemin []int, entier int64, octets []byte)
}

// visiteQuiIgnore ne suit aucune valeur : ce qu elle parcourt est lu pour avancer le curseur, rien de plus.
func visiteQuiIgnore() visiteBond {
	return visiteBond{suivre: func([]int) bool { return false }, feuille: func([]int, int64, []byte) {}}
}

// varint lit un entier variable de Bond de `max` octets au plus (sept bits par octet, poids faible
// d abord, 0x80 = suite) ; un entier plus long fait tomber le curseur.
func (l *lecteurDuCorps) varint(max int) uint64 {
	var v uint64
	for i := range max {
		o := l.lire(8)
		v |= (o & 0x7F) << (7 * uint(i))
		if o&0x80 == 0 {
			return v
		}
	}
	l.ok = false
	return 0
}

// octetsRestants rend le nombre d octets entiers encore dans le tampon ; zero des qu une lecture est sortie
// du tampon, pour qu aucune longueur lue dans le flux ne passe la borne apres un debordement.
func (l *lecteurDuCorps) octetsRestants() uint64 {
	reste := l.br.Remaining()
	if reste < 0 {
		return 0
	}
	return uint64(reste) / 8
}

// octetsDe lit `n` octets, ou fait tomber le curseur quand ils ne tiennent pas dans le tampon.
func (l *lecteurDuCorps) octetsDe(n uint64) []byte {
	if !l.ok || n > l.octetsRestants() {
		l.ok = false
		return nil
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(l.lire(8))
	}
	return b
}

// enjamberLesOctets avance de `n` octets sans les copier.
func (l *lecteurDuCorps) enjamberLesOctets(n uint64) {
	if !l.ok || n > l.octetsRestants() {
		l.ok = false
		return
	}
	l.br.Skip(int(n) * 8)
}

// zigzag64 decode un entier signe ecrit en zigzag.
func zigzag64(u uint64) int64 { return int64(u>>1) ^ -int64(u&1) }

// valeurBond lit la valeur de type `ty` au curseur, au chemin `chemin`, sous la visite `v`.
func (l *lecteurDuCorps) valeurBond(ty int, chemin []int, v visiteBond, prof int) {
	if !l.ok {
		return
	}
	if prof > profondeurBondMax {
		l.ok = false
		return
	}
	suivie := v.suivre(chemin)
	switch ty {
	case bondStructure:
		l.structureBond(suivie, chemin, v, prof)
	case bondListe, bondEnsemble:
		l.listeBond(suivie, chemin, v, prof)
	case bondTable:
		l.tableBond(prof)
	case bondChaine:
		l.chaineBond(suivie, chemin, v)
	case bondChaineLarge:
		l.enjamberLesOctets(2 * l.varint(5))
	default:
		l.scalaireBond(ty, suivie, chemin, v)
	}
}

// scalaireBond lit un booleen, un entier ou un flottant ; un autre type fait tomber le curseur.
func (l *lecteurDuCorps) scalaireBond(ty int, suivie bool, chemin []int, v visiteBond) {
	var e int64
	switch {
	case ty == bondBooleen || ty == bondOctet:
		e = int64(l.lire(8))
	case ty == bondOctetSigne:
		e = int64(int8(l.lire(8)))
	case ty >= bondEntierNS16 && ty <= bondEntierNS64:
		e = int64(l.varint(octetsParEntier64))
	case ty >= bondEntierS16 && ty <= bondEntierS64:
		e = zigzag64(l.varint(octetsParEntier64))
	case ty == bondFlottant32:
		l.lire(32)
		return
	case ty == bondFlottant64:
		l.lire(64)
		return
	default:
		l.ok = false
		return
	}
	if suivie && l.ok {
		v.feuille(chemin, e, nil)
	}
}

// chaineBond lit une chaine : sa longueur (entier variable de 32 bits) puis ses octets.
func (l *lecteurDuCorps) chaineBond(suivie bool, chemin []int, v visiteBond) {
	n := l.varint(5)
	if !suivie {
		l.enjamberLesOctets(n)
		return
	}
	if b := l.octetsDe(n); l.ok {
		v.feuille(chemin, 0, b)
	}
}

// structureBond lit une structure : sa longueur en octets, ses champs, son octet de fin. Une
// structure non suivie est enjambee par sa longueur, dont le dernier octet doit etre la fin.
func (l *lecteurDuCorps) structureBond(suivie bool, chemin []int, v visiteBond, prof int) {
	n := l.varint(5)
	if n > l.octetsRestants() {
		l.ok = false
		return
	}
	fin := l.br.BitPos() + 8*int(n)
	if !suivie {
		l.allerA(fin - 8)
		l.ok = l.ok && l.lire(8) == bondFin
		return
	}
	for l.ok && l.br.BitPos() < fin {
		id, ty := l.enTete()
		switch ty {
		case bondFin:
			l.ok = l.ok && l.br.BitPos() == fin
			return
		case bondFinDeBase:
		default:
			l.valeurBond(ty, append(chemin, id), v, prof+1)
		}
	}
	l.ok = false
}

// listeBond lit une liste ou un ensemble : un octet `type | (n + 1) << 5` quand n < 7, sinon le type
// puis n en entier variable. Une liste d octets est rendue d un bloc au chemin de la liste ; les
// elements des autres listes ont pour chemin celui de la liste suivi de leur rang.
func (l *lecteurDuCorps) listeBond(suivie bool, chemin []int, v visiteBond, prof int) {
	x := l.lire(8)
	n, et := x>>5, int(x&0x1F)
	if n == 0 {
		n = l.varint(5)
	} else {
		n--
	}
	if !l.ok || n > l.octetsRestants() {
		l.ok = false
		return
	}
	if !suivie {
		v = visiteQuiIgnore() // les elements d une liste non suivie ne sont jamais soumis a la visite
	}
	if et == bondOctet {
		if !suivie {
			l.enjamberLesOctets(n)
		} else if b := l.octetsDe(n); l.ok {
			v.feuille(chemin, 0, b)
		}
		return
	}
	for i := range int(n) {
		l.valeurBond(et, append(chemin, i), v, prof+1)
	}
}

// tableBond enjambe une table : les types de sa cle et de sa valeur, son compte, ses paires. Aucun
// chemin suivi ne traverse une table.
func (l *lecteurDuCorps) tableBond(prof int) {
	kt, vt := int(l.lire(8)), int(l.lire(8))
	n := l.varint(5)
	if !l.ok || n > l.octetsRestants() {
		l.ok = false
		return
	}
	for range int(n) {
		l.valeurBond(kt, nil, visiteQuiIgnore(), prof+1)
		l.valeurBond(vt, nil, visiteQuiIgnore(), prof+1)
	}
}

// cheminAvecJokers dit si `chemin` est `modele` ou son debut ; un element `joker` du modele accepte
// tout id ou tout rang.
func cheminAvecJokers(chemin, modele []int) bool {
	if len(chemin) > len(modele) {
		return false
	}
	for i, c := range chemin {
		if modele[i] != jokerDeChemin && modele[i] != c {
			return false
		}
	}
	return true
}

// cheminEgal dit si `chemin` est exactement `modele` (jokers compris).
func cheminEgal(chemin, modele []int) bool {
	return len(chemin) == len(modele) && cheminAvecJokers(chemin, modele)
}

// jokerDeChemin tient la place d un id ou d un rang dans un modele de chemin.
const jokerDeChemin = -1
