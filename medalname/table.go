// Package medalname — IDENTITE DES MEDAILLES LUES DANS LE FILM HALO INFINITE.
//
// CE QUE CE PAQUET RESOUT. Le chunk highlight du film porte, pour chaque medaille,
// un couple d octets : `type_hint` (b[47] du bloc event) et `medal_type` (b[59]) —
// jamais le nom. `type_hint` est le poids de tri de la medaille, `medal_type` son indice
// dans la liste des medailles du jeu. Le nom anglais est la CLE DE REFERENTIEL du reste
// de la chaine : `highlight_events.raw_json.medal_name` -> `medal_definitions` -> label
// localise, image, medal_name_id (cf. service/match_view_killfeed_medals.go). Sans lui,
// un event medal n a pas d identite et le fil des eliminations ne l affiche pas.
//
// CE N EST PAS UN LIBELLE D INTERFACE. Le nom rendu ici est une clef anglaise. La
// locale reste resolue a la LECTURE par `medal_definitions` — rien de traduit ici.
//
// D OU VIENT LA TABLE. `data/medals.tsv` reprend la liste des medailles du jeu de
// libfilmshell (MIT, `src/game/medals_data.rs`, generee par ses mainteneurs a partir des
// fichiers du jeu) : un couple (type_hint, medal_type) et un nom par medaille, 157 au
// total, avec son entete de credit ; la licence est a cote (`data/LICENSE-libfilmshell`)
// et `NOTICE.md` la cite. La table connait donc aussi les medailles des modes que
// personne n a joue sur nos films (Infection, Attrition...). Le fichier se regenere
// depuis la source, il ne s edite pas a la main.
//
// LE CORPUS MESURE RESTE LE TEMOIN. `testdata/corpus_medailles_2026-09-02.tsv` fige 124
// couples releves sur des films reels (44 568 events medal, un nom par couple, zero
// couple ambigu) : le test rejoue chaque ligne contre la table, qui doit rendre exactement
// le nom mesure. C est ce qui prouve que la source et les films disent la meme chose.
//
// DEGRADATION. Un couple absent de la table ne rend AUCUN nom (jamais un voisin) :
// l event medal est persiste sans `raw_json` et le collecteur compte le trou.
package medalname

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed data/medals.tsv
var brut string

// medalKey est le couple d octets qui identifie une medaille dans le bloc event du
// chunk highlight : type_hint = b[47], medalType = b[59] (cf.
// highlightevent.HighlightEvent.TypeHint / .MedalType).
type medalKey struct {
	typeHint  int
	medalType int
}

// Colonnes d une ligne du fichier.
const (
	colTypeHint = iota
	colMedalType
	colNom
	colonnes
)

// table : couple du film -> nom anglais de la medaille (clef de referentiel).
var table = charger(brut)

// charger lit la table : une ligne par medaille, les lignes de commentaire (`#`) et les
// lignes vides sont ignorees. Une ligne mal formee, un nom vide ou un couple en double
// est une erreur du fichier embarque, qui ne se corrige que dans le depot : elle arrete
// le programme au chargement plutot que de servir une table tronquee ou ecrasee.
func charger(texte string) map[medalKey]string {
	out := map[medalKey]string{}
	for _, l := range strings.Split(texte, "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) != colonnes || c[colNom] == "" {
			panic("medalname : ligne mal formee : " + l)
		}
		th, errTh := strconv.Atoi(c[colTypeHint])
		mt, errMt := strconv.Atoi(c[colMedalType])
		if errTh != nil || errMt != nil {
			panic("medalname : couple mal forme : " + l)
		}
		k := medalKey{typeHint: th, medalType: mt}
		if _, deja := out[k]; deja {
			panic("medalname : couple en double : " + l)
		}
		out[k] = c[colNom]
	}
	return out
}

// Lookup rend le nom anglais de la medaille designee par le couple du film, et
// `false` si le couple est inconnu de la table. Un couple inconnu N EST PAS une erreur :
// c est une medaille que la source ne liste pas (medaille posterieure a la table). L
// appelant persiste alors l event sans identite plutot que d inventer un nom voisin.
func Lookup(typeHint, medalType int) (string, bool) {
	name, ok := table[medalKey{typeHint: typeHint, medalType: medalType}]
	return name, ok
}

// Len rend le nombre de couples connus. Sert au test de completude et aux compteurs
// de diagnostic.
func Len() int { return len(table) }
