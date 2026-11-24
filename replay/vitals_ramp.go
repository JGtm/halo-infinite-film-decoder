package replay

// vitals_ramp.go : LA REGLE DE RAMPE DU CALQUE DES VITALITES.
//
// LE PROBLEME. Le jeu ecrit la recharge du bouclier cran par cran, a chaque trame (un cran = 1/64 du bouclier plein,
// environ 17 ms) : publiee telle quelle, une recharge complete pese des dizaines de lignes qu un lecteur retrouve par
// une droite.
//
// LA REGLE, « RAMPE A L ERREUR D UN CRAN ». Une ligne d une vie est ECARTEE quand elle est a l interieur d une rampe : la
// suite de lignes consecutives de la vie ou la sante ne bouge pas et ou le bouclier MONTE a chaque ligne, et quand elle
// s ecarte de la droite tracee entre les deux lignes qui bornent la rampe d au plus un cran de bouclier (la resolution du
// film), a l instant de la ligne. Un lecteur qui interpole entre la ligne de debut et la ligne de fin retrouve donc chaque
// valeur ecartee a un cran pres, et chaque instant ecarte est un instant que la droite date. Le glissement est glouton : la
// rampe s etend tant que toutes ses lignes interieures tiennent dans l erreur, puis la ligne qui la clot ouvre la suivante.
//
// CE QU ELLE NE TOUCHE JAMAIS (lignes PROTEGEES, ecartees par aucune rampe) :
//   - la ligne a l instant d un message de degat de la vie, et la derniere ligne STRICTEMENT avant cet instant : l etat
//     juste avant et l etat a l instant du coup, que le calcul des points de degat compare, sont donc exacts ;
//   - la ligne de la mort (la premiere a sante nulle) et la derniere ligne de la vie : la derniere valeur avant la mort ;
//   - toute ligne ou la sante change, et toute ligne ou le bouclier baisse ou se tient (hors rampe montante) : seule la
//     montee du bouclier est reguliere, ce qui n est pas une montee reste publie.
//
// La regle ne depend ni de l ordre d entree ni du nombre de fils : elle se joue vie par vie sur des lignes deja triees.

import (
	"math"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// crandDeBouclier est un cran de la valeur de bouclier, en fraction du bouclier plein : le bouclier est ecrit sur 8 bits et
// le plein vaut 64 crans ([grammar.OvershieldFullQ]).
const crandDeBouclier = 1.0 / grammar.OvershieldFullQ

// instantsDeCoups range, par vie, les instants des messages de degat qui la nomment, tries.
func instantsDeCoups(degats []grammar.DegatLu) map[cleDeVie][]uint64 {
	out := map[cleDeVie][]uint64{}
	for _, d := range degats {
		if !d.AVictime {
			continue
		}
		k := cleDeVie{uint32(d.Victime) + decalageDesSlotsDUnite, d.GenVictime}
		out[k] = append(out[k], d.TS)
	}
	for _, ts := range out {
		slices.Sort(ts)
	}
	return out
}

// proteger marque, parmi les lignes d une vie, celles que la regle de rampe ne peut jamais ecarter : voir l en-tete.
func proteger(lignes []ligneVitale, coups []uint64) {
	if len(lignes) == 0 {
		return
	}
	lignes[len(lignes)-1].protegee = true
	for _, ts := range coups {
		// premiere ligne a l instant `ts` ou apres : la ligne a `ts` s il y en a une, sinon celle qui suit.
		i, _ := slices.BinarySearchFunc(lignes, ts, func(l ligneVitale, ts uint64) int {
			switch {
			case l.us < ts:
				return -1
			case l.us > ts:
				return 1
			}
			return 0
		})
		for j := i; j < len(lignes) && lignes[j].us == ts; j++ {
			lignes[j].protegee = true
		}
		if i > 0 {
			lignes[i-1].protegee = true
		}
	}
}

// montePure dit que `b` suit `a` dans une rampe : meme sante, bouclier plus haut, `a` n etant pas la mort.
func montePure(a, b ligneVitale) bool {
	return !a.v.Dead && a.v.Hp == b.v.Hp && b.v.Sh > a.v.Sh && b.us > a.us
}

// surLaDroite dit que la ligne `l` est a au plus un cran de bouclier de la droite entre `debut` et `fin`, a son instant.
func surLaDroite(debut, fin, l ligneVitale) bool {
	f := float64(l.us-debut.us) / float64(fin.us-debut.us)
	valeur := float64(debut.v.Sh) + f*float64(fin.v.Sh-debut.v.Sh)
	return math.Abs(valeur-float64(l.v.Sh)) <= crandDeBouclier
}

// finDeRampe rend l index de la ligne qui clot la plus longue rampe commencee a la ligne `i`, ou `i+1` quand aucune ligne
// n est a ecarter. Les lignes interieures ne sont ni protegees ni hors de la droite.
func finDeRampe(lignes []ligneVitale, i int) int {
	meilleure := i + 1
	for e := i + 1; e < len(lignes); e++ {
		if !montePure(lignes[e-1], lignes[e]) {
			break
		}
		if e-1 > i && lignes[e-1].protegee {
			break
		}
		ok := true
		for k := i + 1; k < e && ok; k++ {
			ok = surLaDroite(lignes[i], lignes[e], lignes[k])
		}
		if !ok {
			break
		}
		meilleure = e
	}
	return meilleure
}

// ecarterLesRampes rend les lignes d une vie sans l interieur de ses rampes, et le nombre de lignes ecartees.
func ecarterLesRampes(lignes []ligneVitale) ([]ligneVitale, int) {
	if len(lignes) < 3 {
		return lignes, 0
	}
	gardees := make([]ligneVitale, 0, len(lignes))
	gardees = append(gardees, lignes[0])
	i := 0
	for i < len(lignes)-1 {
		e := finDeRampe(lignes, i)
		gardees = append(gardees, lignes[e])
		i = e
	}
	return gardees, len(lignes) - len(gardees)
}
