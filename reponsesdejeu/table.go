// Package reponsesdejeu nomme les réponses que le script de mode du jeu envoie à un joueur, et donne les
// points de celles qui donnent du score personnel.
//
// UN MESSAGE DE JEU NE PORTE NI NOM NI POINTS. Il porte un rang, la position de sa réponse dans l'arbre
// des événements des scripts du mode ([lecture.EvenementDeJeu.Rang]) : le même rang désigne une autre
// réponse dans un autre mode. Le groupe de scripts du film (les noms `*.lua` de son chunk 0) fixe l'arbre ;
// cette table dit, pour chaque groupe et chaque rang de l'arbre, le nom que le script donne à la réponse
// (`killPersonalScore`), son genre (score personnel, médaille, annonce, ligne du fil des éliminations...)
// et, pour un score personnel, ses points et le libellé anglais que le jeu affiche avec eux (« Killed %pe »).
//
// QUATRE ISSUES POUR UN MESSAGE ([Nature]) : une réponse qui donne du score ([NatureScore]), une réponse que
// l'arbre nomme et qui n'en donne pas ([NatureSansScore] : annonce, médaille, ligne du fil), un rang de l'arbre
// que le message ne confirme pas ([NatureNonConfirmee]), et un rang hors de l'arbre ([NatureHorsArbre]). L'arbre ne
// liste que les événements qui sont des réponses : un rang hors de lui est le plus souvent un événement qui n'en est
// pas une, et la table n'en dit rien de plus.
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

// Genres d'une réponse, tels que le fichier les écrit.
const (
	// GenreScore est le genre d'une réponse qui donne du score personnel.
	GenreScore      = "score"
	genreMedaille   = "medal"
	genreAnnonce    = "announcer"
	genreFil        = "killfeed"
	genreAutre      = "other"
	valeurIncertain = "1"
)

// Colonnes d'une ligne du fichier.
const (
	colGroupe = iota
	colRang
	colNom
	colGenre
	colPoints
	colLibelle
	colTextes
	colIncertain
	colonnes
)

// Reponse est une réponse de l'arbre d'un mode.
type Reponse struct {
	// Nom est le nom que le script du mode donne à la réponse (`killPersonalScore`) ; vide pour une réponse
	// que le script ne nomme pas.
	Nom string
	// Genre est GenreScore quand la réponse donne du score personnel ; sinon la médaille, l'annonce, la
	// ligne du fil ou le reste, dans les mots du fichier.
	Genre string
	// Points est le score personnel qu'elle donne : négatif pour une pénalité, nul hors GenreScore et pour
	// une réponse que le jeu nomme sans lui donner de points.
	Points int
	// Libelle est le texte anglais que le jeu affiche avec les points ; `%pa` et `%pe` désignent des
	// joueurs, que les participants du message nomment. Vide hors GenreScore.
	Libelle string
	// Textes sont les identifiants de chaîne du libellé propre de la réponse ; vide quand elle n'en a pas.
	Textes []uint32
	// Incertain dit que le rang n'est pas placé avec certitude dans l'arbre du groupe.
	Incertain bool
}

// Nature dit ce que la table sait d'un message.
type Nature int

const (
	// NatureHorsArbre : le rang n'est pas dans l'arbre du groupe. L'arbre ne liste que les événements qui sont des
	// réponses : un rang hors de lui est un événement qui n'en est pas une (la majorité des messages d'un film), ou
	// une réponse que la table ne connaît pas (un mode que ses auteurs ont écrit eux-mêmes).
	NatureHorsArbre Nature = iota
	// NatureNonConfirmee : l'arbre place une réponse à ce rang, mais le message n'en est pas la confirmation (son mot
	// texte est celui d'une autre réponse, ou le rang est incertain et le texte ne le confirme pas).
	NatureNonConfirmee
	// NatureSansScore : le rang désigne une réponse que l'arbre nomme et qui ne donne pas de score.
	NatureSansScore
	// NatureScore : le rang désigne une réponse qui donne du score personnel (ses points peuvent être nuls).
	NatureScore
)

// Groupe est l'arbre des événements d'un mode.
type Groupe struct {
	nom     string
	reponse map[uint32]Reponse
}

// Nom rend le groupe : les fichiers d'événements du mode, joints par « + ».
func (g *Groupe) Nom() string { return g.nom }

// Reponse rend la réponse que le message de rang `rang` désigne dans ce groupe, et ce que la table en sait.
// `texte` est le mot texte du message, valable quand `aTexte`. [NatureHorsArbre] quand le rang n'est pas dans l'arbre ; [NatureNonConfirmee]
// (et une réponse vide) quand le texte du message est celui d'une autre réponse, ou quand le rang
// est incertain et que le texte ne le confirme pas.
func (g *Groupe) Reponse(rang, texte uint32, aTexte bool) (Reponse, Nature) {
	r, ok := g.reponse[rang]
	if !ok {
		return Reponse{}, NatureHorsArbre
	}
	confirme := aTexte && slices.Contains(r.Textes, texte)
	contredit := aTexte && len(r.Textes) > 0 && !confirme
	switch {
	case contredit || (r.Incertain && !confirme):
		return Reponse{}, NatureNonConfirmee
	case r.Genre == GenreScore:
		return r, NatureScore
	}
	return r, NatureSansScore
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

// Taille rend le nombre de rangs de la table, celui des rangs qui donnent du score personnel, et celui des groupes.
func Taille() (rangs, scores, nbGroupes int) {
	for _, g := range groupes {
		rangs += len(g.reponse)
		for _, r := range g.reponse {
			if r.Genre == GenreScore {
				scores++
			}
		}
	}
	return rangs, scores, len(groupes)
}

type groupeCharge struct {
	Groupe
	fichiers []string
}

// genreValide dit que le fichier n'écrit que les genres que le code connaît.
func genreValide(g string) bool {
	switch g {
	case GenreScore, genreMedaille, genreAnnonce, genreFil, genreAutre:
		return true
	}
	return false
}

// charger lit la table : une ligne par rang, les lignes de commentaire (`#`) et les lignes vides sont
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
		rang, errRang := strconv.ParseUint(c[colRang], 10, 32)
		if errRang != nil || !genreValide(c[colGenre]) || (c[colIncertain] != "0" && c[colIncertain] != valeurIncertain) {
			panic("reponsesdejeu : ligne mal formee : " + l)
		}
		points := 0
		if c[colGenre] == GenreScore {
			var err error
			if points, err = strconv.Atoi(c[colPoints]); err != nil {
				panic("reponsesdejeu : points mal formes : " + l)
			}
		}
		g := par[c[colGroupe]]
		if g == nil {
			g = &groupeCharge{Groupe: Groupe{nom: c[colGroupe], reponse: map[uint32]Reponse{}}, fichiers: strings.Split(c[colGroupe], "+")}
			par[c[colGroupe]] = g
			out = append(out, g)
		}
		r := Reponse{Nom: c[colNom], Genre: c[colGenre], Points: points, Libelle: c[colLibelle], Incertain: c[colIncertain] == valeurIncertain}
		for _, t := range strings.Split(c[colTextes], ",") {
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
