// Package chattername — NOM COURT DES REPLIQUES DES SPARTANS LUES DANS LE FILM HALO INFINITE.
//
// CE QUE LE FILM PORTE. Un message de replique (`AIDialog`) ecrit un identifiant de son (32 bits, UN PAR VOIX ET PAR
// VARIANTE) et un indice de replique de douze bits. C est L INDICE qui designe le nom : il numerote une entree des
// tables de vocalisation du jeu, commune a toutes les voix ; le son, lui, change avec la voix et ne nomme rien. Les
// tables du jeu nomment chaque entree par le hachage de son nom, jamais par le nom.
//
// CE QUE LA TABLE PROUVE. `data/names.tsv` rend, pour un indice, le nom dont le hachage (MurmurHash3 x86 32 bits,
// graine 0, sur les octets ASCII) EGALE le hachage enregistre pour cette entree des tables du jeu. Les hachages enregistres
// (deuxieme colonne) ont ete relus par le mainteneur dans les tags de dialogue de l installation du jeu, en lecture seule ;
// le test ne peut pas les relire (aucun fichier du jeu n est livre) : il prouve que chaque nom donne bien son hachage, la
// lecture du jeu reste le fait du mainteneur. Le hachage est dans le fichier :[TestChaqueNomEstProuveParSonHachage] le recalcule. Un nom n entre dans la table qu avec cette egalite ET
// des mots qui se lisent ; une ressemblance de texte ne prouve rien et n est pas utilisee.
//
// CE QUE LA TABLE NE DIT PAS. (1) L indice n a de sens que pour la disposition des tables du build qui a ecrit le film :
// les builds anterieurs ont insere un bloc de 205 entrees avant le quatrieme, et l indice y vaut 205 de plus. La table ne
// sert donc qu aux builds de [layoutsConnus], ceux dont la disposition a ete mesuree sur des films reels (la priorite
// ecrite par le film egale celle de la table pour 99 % des repliques). Un autre build, ou un film sans build lu, ne recoit
// AUCUN nom. (2) Une entree dont le nom n est pas prouve n est pas dans la table : la replique se publie sans nom.
// (3) Seules les entrees 0 a 914 et 2732 a 2956 sont nommees : l ordre des deuxieme et troisieme tables n est pas etabli,
// et avec lui l indice de toutes celles qui les suivent jusqu a 2731.
//
// AUCUN FICHIER DU JEU N EST REPRODUIT, ni aucun texte prononce : la table est le resultat d une recherche, un nom court
// par indice.
package chattername

import (
	_ "embed"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

//go:embed data/names.tsv
var brut string

// layoutsConnus sont les builds dont la disposition des tables de vocalisation est celle que la table suppose : 1.12 et
// 1.13, mesures sur trois films (priorite du film egale a celle de la table : 162 sur 165, 363 sur 365, 414 sur 418 ; les
// ecarts sont tous la meme entree, `knockback`, dont la priorite est reecrite). Les builds 1.10 et 1.11 ont l autre
// disposition (un bloc de 205 entrees insere), les builds plus anciens et les plus recents sont non mesures.
var layoutsConnus = map[string]bool{"HI_1_12_0": true, "HI_1_13_0": true}

// Colonnes d une ligne du fichier.
const (
	colIndice = iota
	colHachage
	colNom
	colonnes
)

// entree est une ligne de la table : le hachage enregistre et le nom.
type entree struct {
	hachage uint32
	nom     string
}

// table : indice de replique -> entree.
var table = charger(brut)

// charger lit la table : une ligne par entree, lignes de commentaire (`#`) et vides ignorees. Une ligne mal formee, un
// indice en double, un indice hors des douze bits ou un nom vide est une erreur du fichier embarque : elle arrete le
// programme au chargement plutot que de servir une table tronquee ou ecrasee.
func charger(texte string) map[int]entree {
	out := map[int]entree{}
	for _, l := range strings.Split(texte, "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) != colonnes || c[colNom] == "" {
			panic("chattername : ligne mal formee : " + l)
		}
		i, errI := strconv.Atoi(c[colIndice])
		h, errH := strconv.ParseUint(c[colHachage], 16, 32)
		if errI != nil || errH != nil || i < 0 || i > 0xFFF {
			panic("chattername : indice ou hachage mal forme : " + l)
		}
		if _, deja := out[i]; deja {
			panic("chattername : indice en double : " + l)
		}
		out[i] = entree{hachage: uint32(h), nom: c[colNom]}
	}
	return out
}

// Lookup rend le nom court de la replique d indice `line` pour un film du build `build`, et `false` quand le build n est
// pas de ceux dont la disposition est mesuree ([layoutsConnus]), ou que l indice n a pas de nom prouve. Un refus N EST PAS
// une erreur : la replique se publie sans nom, et le document la compte.
func Lookup(build string, line int) (string, bool) {
	if !layoutsConnus[build] {
		return "", false
	}
	e, ok := table[line]
	return e.nom, ok
}

// Len rend le nombre d indices nommes. Sert au test de completude.
func Len() int { return len(table) }

// Hash rend le hachage d un nom de vocalisation : MurmurHash3 x86 32 bits, graine 0, sur ses octets. C est la fonction dont
// le jeu se sert pour nommer une entree de ses tables ; elle sert au seul test de preuve.
func Hash(nom string) uint32 {
	const c1, c2 = 0xcc9e2d51, 0x1b873593
	b := []byte(nom)
	var h uint32
	n := len(b) / 4 * 4
	for i := 0; i < n; i += 4 {
		k := uint32(b[i]) | uint32(b[i+1])<<8 | uint32(b[i+2])<<16 | uint32(b[i+3])<<24
		k = bits.RotateLeft32(k*c1, 15) * c2
		h ^= k
		h = bits.RotateLeft32(h, 13)*5 + 0xe6546b64
	}
	var k uint32
	for i, x := range b[n:] {
		k |= uint32(x) << (8 * uint(i))
	}
	if len(b) > n {
		h ^= bits.RotateLeft32(k*c1, 15) * c2
	}
	h ^= uint32(len(b)) //nolint:gosec // longueur d un nom court
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

// verifier rend une erreur pour la premiere ligne dont le hachage enregistre n est pas celui de son nom, ou nil.
func verifier() error {
	for i, e := range table {
		if Hash(e.nom) != e.hachage {
			return fmt.Errorf("indice %d : le hachage de %q est %08x, le fichier porte %08x", i, e.nom, Hash(e.nom), e.hachage)
		}
	}
	return nil
}
