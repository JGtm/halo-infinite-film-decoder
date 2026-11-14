package grammar

// film_scripts.go — LES SCRIPTS DE MODE QUE LE FILM DECLARE.
//
// Le chunk 0 du film porte, parmi ses structures de variante, les chemins des scripts du jeu que la partie
// a chargés (`tags\multiplayer\lua\weaponconstants.lua`, `...\slayerevents.lua`) : des chaînes écrites octet
// par octet dans le flux de bits, trois bits après le début d'un octet du chunk. Les fichiers d'événements
// (`slayerevents.lua`, `arenaflagevents.lua` + `flageventstubs.lua`...) désignent le mode, donc l'arbre des
// événements dont le rang d'un message de jeu ([lecture.EvenementDeJeu.Rang]) est une position.
//
// La lecture porte sur le chunk 0 seul, décalé de [decalageDesNomsDeScript] bits, et sur ses seules chaînes
// qui finissent par `.lua` après un séparateur de chemin : elle ne parcourt aucun paquet.

import (
	"bytes"
	"slices"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// decalageDesNomsDeScript est le décalage, en bits, des chaînes de noms de script dans le chunk 0.
const decalageDesNomsDeScript = 3

// suffixeDeScript termine le nom d'un fichier de script ; les deux séparateurs de chemin que le chunk écrit
// devant lui sont l'antislash (92) et l'oblique (47).
const (
	suffixeDeScript     = ".lua"
	separateurAntislash = 92
	separateurOblique   = 47
)

// ScriptsDuFilm rend les noms de fichier (sans dossier, en minuscules, triés, sans doublon) des scripts que
// le chunk 0 de `f` déclare, ou nil quand le film n'a pas de chunk 0.
func ScriptsDuFilm(f *source.Film) []string {
	chunk, ok := FilmRegistryChunk(f)
	if !ok {
		return nil
	}
	return nomsDeScriptDuFlux(fluxDesNomsDeScript(chunk))
}

// fluxDesNomsDeScript rend `chunk` décalé de [decalageDesNomsDeScript] bits vers la gauche : l'octet i du
// flux porte les bits de l'octet i et ceux de l'octet i+1.
func fluxDesNomsDeScript(chunk []byte) []byte {
	if len(chunk) < 2 {
		return nil
	}
	flux := make([]byte, len(chunk)-1)
	for i := range flux {
		flux[i] = chunk[i]<<decalageDesNomsDeScript | chunk[i+1]>>(8-decalageDesNomsDeScript)
	}
	return flux
}

// nomsDeScriptDuFlux rend les noms de fichier de script que `flux` écrit, triés et sans doublon.
func nomsDeScriptDuFlux(flux []byte) []string {
	vus := map[string]bool{}
	for debut := 0; ; {
		i := bytes.Index(flux[debut:], []byte(suffixeDeScript))
		if i < 0 {
			break
		}
		fin := debut + i + len(suffixeDeScript)
		debut = fin
		if fin < len(flux) && estCaractereDeNom(flux[fin]) {
			continue // `.luac` : le suffixe n'est pas celui du nom
		}
		if nom, ok := nomDeScriptAvant(flux, fin); ok {
			vus[nom] = true
		}
	}
	if len(vus) == 0 {
		return nil
	}
	noms := make([]string, 0, len(vus))
	for n := range vus {
		noms = append(noms, n)
	}
	slices.Sort(noms)
	return noms
}

// nomDeScriptAvant rend le nom de fichier qui finit à `fin` dans `flux` : les caractères de nom qui
// précèdent le suffixe, jusqu'à un séparateur de chemin. Faux quand aucun séparateur ne précède un nom non
// vide.
func nomDeScriptAvant(flux []byte, fin int) (string, bool) {
	racine := fin - len(suffixeDeScript)
	debut := racine
	for debut > 0 && estCaractereDeNom(flux[debut-1]) {
		debut--
	}
	if debut == 0 || debut == racine || (flux[debut-1] != separateurAntislash && flux[debut-1] != separateurOblique) {
		return "", false
	}
	return strings.ToLower(string(flux[debut:fin])), true
}

// estCaractereDeNom dit que `b` peut figurer dans un nom de fichier de script.
func estCaractereDeNom(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == ' '
}
