package reponsesdejeu

// decalages.go : LES RANGS DONT L'ARBRE D'UN BUILD N'EST PAS CELUI DE LA TABLE.
//
// La table (`data/reponses.tsv`) donne l'arbre des événements d'un groupe de scripts tel que le jeu actuel l'écrit.
// Sur certains builds, l'arbre que le film a suivi est décalé : le rang `r` du film désigne la réponse que la table
// place au rang `r + cran`. Un décalage ne change ni le nom ni les points d'une réponse : il dit quelle réponse un rang
// désigne. Il ne vaut que pour la ligne qui le déclare (un build, un groupe, un intervalle de rangs du film) :
// hors des intervalles déclarés et pour tout autre build ou groupe, le rang se lit tel quel. Aucun rang n'est
// deviné : une ligne n'entre ici qu'avec la preuve qui la fonde, écrite à côté d'elle.
//
// Un rang décalé qui tombe sur un rang que la table ne contient pas est HORS DE L'ARBRE, comme tout rang absent.

// decalage est une ligne déclarée : sur `build`, dans le groupe `groupe`, les messages dont le rang du film est
// compris entre `premier` et `dernier` (bornes incluses) désignent la réponse de la table au rang `rang + cran`.
type decalage struct {
	build   string
	groupe  string
	premier uint32
	dernier uint32
	cran    uint32
}

// groupeGrandeBataille est le groupe de la Grande bataille en Slayer.
const groupeGrandeBataille = "bigteambattle.lua+btbslayerevents.lua"

// buildDecale est le build dont la table porte des décalages déclarés.
const buildDecale = "HI_1_11_0"

// decalages sont les décalages déclarés.
//
// HI_1_11_0, Grande bataille, rangs 179 à 263 : un cran. La preuve est double et indépendante. Les réponses qui
// portent un mot texte (les médailles de série de victimes et leurs lignes du fil) ne se confirment par leur texte
// qu'au rang de la table plus un sur les rangs 179 à 205 ; aucun film du build n'a de message non confirmé une fois
// le cran appliqué. Et le score du tableau du film, lu à chaque message isolé d'un joueur, donne à chaque rang de
// 227 à 263 les points de la réponse de la table un rang plus loin, sans aucune lecture non nulle contraire : le
// rang 227 est l'aide par IEM (50), 231 la destruction d'un Ghost (50), 240 celle d'un Wraith (100), 250 le
// détournement d'un Banshee (25), 263 un rang hors de l'arbre (0). Les points restent ceux de la table : seul le rang
// change de réponse.
//
// HI_1_11_0, Grande bataille, rangs 447 à 460 : deux crans. Les médailles à mot texte de ces rangs ne se confirment
// qu'au rang de la table plus deux ; aucun film du build n'a de message non confirmé une fois le cran appliqué.
var decalages = []decalage{
	{build: buildDecale, groupe: groupeGrandeBataille, premier: 179, dernier: 263, cran: 1},
	{build: buildDecale, groupe: groupeGrandeBataille, premier: 447, dernier: 460, cran: 2},
}

// decalagesDe rend les lignes déclarées pour ce build et ce groupe.
func decalagesDe(build, groupe string) []decalage {
	var out []decalage
	for _, d := range decalages {
		if d.build == build && d.groupe == groupe {
			out = append(out, d)
		}
	}
	return out
}

// Decalage rend le nombre de rangs dont le rang `rang` du film est décalé dans la table pour le build du groupe :
// zéro hors des intervalles déclarés.
func (g *Groupe) Decalage(rang uint32) uint32 {
	for _, d := range g.decalages {
		if rang >= d.premier && rang <= d.dernier {
			return d.cran
		}
	}
	return 0
}
