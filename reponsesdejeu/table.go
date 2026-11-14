// Package reponsesdejeu nomme les réponses de score personnel que le script de mode du jeu envoie à un
// joueur, et donne leurs points.
//
// UN MESSAGE DE JEU NE PORTE NI NOM NI POINTS. Il porte un rang, la position de sa réponse dans l'arbre
// des événements des scripts du mode ([lecture.EvenementDeJeu.Rang]) : le même rang désigne une autre
// réponse dans un autre mode. Le groupe de scripts du film (les noms `*.lua` de son chunk 0) fixe l'arbre ;
// cette table dit, pour chaque groupe et chaque rang dont la réponse donne du score, le nom que le script
// lui donne (`killPersonalScore`), ses points et le libellé anglais que le jeu affiche avec eux
// (« Killed %pe »).
//
// ORIGINE : `data/reponses.tsv`, reprise de libfilmshell (MIT, commit ad17521) avec son en-tête de crédit ;
// la licence est à côté (`data/LICENSE-libfilmshell`) et `NOTICE.md` la cite.
//
// UNE RÉPONSE NE S'ATTRIBUE QUE SI SON TEXTE LA CONFIRME QUAND ELLE EN A UN. Un message dont le mot texte
// n'est pas celui de la réponse que le rang désigne est celui d'une voisine : il n'est pas attribué. Un rang
// que l'arbre ne place pas avec certitude (dans une boucle d'un script, ou dans les fichiers propres à
// Firefight) ne s'attribue que si le mot texte du message est celui de la réponse.
package reponsesdejeu

import (
	_ "embed"
	"slices"
	"strconv"
	"strings"
)

//go:embed data/reponses.tsv
var brut string

// Reponse est une réponse de score personnel.
type Reponse struct {
	// Nom est le nom que le script du mode donne à la réponse (`killPersonalScore`).
	Nom string
	// Points est le score personnel qu'elle donne : négatif pour une pénalité.
	Points int
	// Libelle est le texte anglais que le jeu affiche avec les points ; `%pa` et `%pe` désignent des
	// joueurs, que les participants du message nomment.
	Libelle string
	// Textes sont les identifiants de chaîne du libellé propre de la réponse ; vide quand elle n'en a pas.
	Textes []uint32
	// Incertain dit que le rang n'est pas placé avec certitude dans l'arbre du groupe.
	Incertain bool
}

// Groupe est l'arbre des événements d'un mode, réduit à ses réponses qui donnent du score.
type Groupe struct {
	nom     string
	reponse map[uint32]Reponse
}

// Nom rend le groupe : les fichiers d'événements du mode, joints par « + ».
func (g *Groupe) Nom() string { return g.nom }

// Reponse rend la réponse de score que le message de rang `rang` désigne dans ce groupe. `texte` est le mot
// texte du message, valable quand `aTexte`. Faux quand le rang n'est pas celui d'une réponse qui donne du
// score, quand le texte du message est celui d'une autre réponse, ou quand le rang est incertain et que le
// texte ne le confirme pas.
func (g *Groupe) Reponse(rang, texte uint32, aTexte bool) (Reponse, bool) {
	r, ok := g.reponse[rang]
	if !ok {
		return Reponse{}, false
	}
	confirme := aTexte && slices.Contains(r.Textes, texte)
	contredit := aTexte && len(r.Textes) > 0 && !confirme
	if contredit || (r.Incertain && !confirme) {
		return Reponse{}, false
	}
	return r, true
}

// groupes sont les groupes de la table, dans l'ordre du fichier.
var groupes = charger(brut)

// GroupeDesScripts rend le groupe dont les fichiers d'événements sont exactement ceux, parmi les fichiers
// de tous les groupes, que `scripts` (noms de fichier en minuscules) contient. Faux quand aucun groupe ne
// correspond : un film dont le mode est inconnu de la table n'a pas de réponse nommée.
func GroupeDesScripts(scripts []string) (*Groupe, bool) {
	present := map[string]bool{}
	for _, g := range groupes {
		for _, f := range g.fichiers {
			if slices.Contains(scripts, f) {
				present[f] = true
			}
		}
	}
	for _, g := range groupes {
		if len(g.fichiers) == len(present) && !slices.ContainsFunc(g.fichiers, func(f string) bool { return !present[f] }) {
			return &g.Groupe, true
		}
	}
	return nil, false
}

// Taille rend le nombre de lignes de la table et celui de ses groupes.
func Taille() (lignes, nbGroupes int) {
	for _, g := range groupes {
		lignes += len(g.reponse)
	}
	return lignes, len(groupes)
}

type groupeCharge struct {
	Groupe
	fichiers []string
}

// charger lit la table : une ligne par réponse, les lignes de commentaire (`#`) et les lignes vides sont
// ignorées. Une ligne mal formée est une erreur du fichier embarqué, qui ne se corrige que dans le dépôt :
// elle arrête le programme au chargement plutôt que de servir une table tronquée.
func charger(texte string) []*groupeCharge {
	var out []*groupeCharge
	par := map[string]*groupeCharge{}
	for _, l := range strings.Split(texte, "\n") {
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) != colonnes {
			panic("reponsesdejeu : ligne mal formee : " + l)
		}
		rang, errRang := strconv.ParseUint(c[1], 10, 32)
		points, errPts := strconv.Atoi(c[3])
		if errRang != nil || errPts != nil || (c[6] != "0" && c[6] != "1") {
			panic("reponsesdejeu : ligne mal formee : " + l)
		}
		g := par[c[0]]
		if g == nil {
			g = &groupeCharge{Groupe: Groupe{nom: c[0], reponse: map[uint32]Reponse{}}, fichiers: strings.Split(c[0], "+")}
			par[c[0]] = g
			out = append(out, g)
		}
		r := Reponse{Nom: c[2], Points: points, Libelle: c[4], Incertain: c[6] == "1"}
		for _, t := range strings.Split(c[5], ",") {
			if t == "" {
				continue
			}
			v, err := strconv.ParseUint(strings.TrimPrefix(t, "0x"), 16, 32)
			if err != nil {
				panic("reponsesdejeu : mot texte mal forme : " + l)
			}
			r.Textes = append(r.Textes, uint32(v))
		}
		g.reponse[uint32(rang)] = r
	}
	return out
}

// colonnes est le nombre de colonnes d'une ligne de la table.
const colonnes = 7
