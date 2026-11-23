package grammar

// pas_des_commandes.go — LE PAS DES ENTREES DE CONTROLE : a quelle cadence le film enregistre les commandes d un
// joueur.
//
// # CE QUE LE FILM ECRIT
//
// Un film enregistre l entree de controle d un joueur une fois par tick du jeu (16,7 ms a 60 Hz), ou une fois
// tous les deux ticks (33,3 ms). Une commande tenue qui dure un seul tick — l appui d un bouton de saut — n est
// vue qu une fois sur deux a la seconde cadence : sa lecture est alors incomplete, et la cadence est le seul fait
// du film qui separe les films ou elle est juste de ceux ou elle ne l est pas (ni la build, ni la version du film
// ne le font : deux films de meme build et de meme version portent les deux cadences).
//
// # LA MESURE
//
// Le pas d un film est l ecart le PLUS FREQUENT entre deux entrees lues, portant le bloc de commande, d une meme
// place, sans paquet non lu entre elles : un trou peut cacher des entrees, et l ecart qui le traverse mesure le
// trou, pas la cadence. Les instants des paquets portent une gigue d environ une milliseconde (un tick de 16,7 ms
// se lit 15, 16 ou 17 ms) : les ecarts se rangent donc par milliseconde, et le pas est la
// moyenne des ecarts de la FENETRE de trois millisecondes la plus peuplee, ce qui rend le tick (16 667 ou
// 33 333 µs) sans que la gigue partage le rang en deux. Un ecart nul (deux entrees d une place dans le meme
// paquet) ne mesure rien. Les ecarts d au moins [ecartsDuPasMaxMS] millisecondes sont comptes mais jamais
// modaux : ils disent une place qu on n a pas lue pendant un moment, pas une cadence.
//
// Le verdict (« ce pas est-il la pleine cadence ? ») est celui de [types.ControlHoldStats.PleineCadence], pas de
// la lecture.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// ecartsDuPasMaxMS est la plage des ecarts rangeables : un ecart de cette duree ou plus n entre dans aucun rang.
const ecartsDuPasMaxMS = 128

// pasDesEntrees accumule les ecarts entre entrees lues d une meme place.
type pasDesEntrees struct {
	// derniere : l instant de la derniere entree lue de chaque place ; vue : la place a une derniere entree sans
	// trou depuis.
	derniere [joueursDeControle]uint64
	vue      [joueursDeControle]bool
	// rang : le nombre d ecarts par milliseconde ; somme : leur somme en microsecondes ; total : tous les ecarts
	// mesures, hors plage compris.
	rang  [ecartsDuPasMaxMS]int
	somme [ecartsDuPasMaxMS]uint64
	total int
}

// noter range l ecart entre l entree lue de la place `index` a l instant `ts` et la precedente de la meme place,
// quand aucun trou ne les separe.
func (p *pasDesEntrees) noter(index int, ts uint64) {
	if index < 0 || index >= joueursDeControle {
		return
	}
	if p.vue[index] && ts > p.derniere[index] {
		ecart := ts - p.derniere[index]
		p.total++
		if ms := ecart / 1000; ms < ecartsDuPasMaxMS {
			p.rang[ms]++
			p.somme[ms] += ecart
		}
	}
	p.derniere[index], p.vue[index] = ts, true
}

// rompre dit qu un paquet non lu vient de passer : il peut cacher des entrees de toutes les places, et l ecart
// qui le traverserait ne mesure pas la cadence.
func (p *pasDesEntrees) rompre() {
	p.vue = [joueursDeControle]bool{}
}

// mesurer rend le bilan de la mesure dans les compteurs de la lecture : le pas, et le nombre d ecarts qui le
// portent (ceux de la fenetre modale). Egalite des fenetres : la plus courte l emporte, pour que le verdict ne
// depende pas de l ordre.
func (p *pasDesEntrees) mesurer(st *types.ControlHoldStats) {
	st.Gaps = p.total
	best, bestN := -1, 0
	for ms := range p.rang {
		if n, _ := p.fenetre(ms); n > bestN {
			best, bestN = ms, n
		}
	}
	if best < 0 {
		return
	}
	n, somme := p.fenetre(best)
	st.StepGaps = n
	st.StepUS = int64(somme) / int64(n) //nolint:gosec // somme d ecarts d un film, bien sous 2^63 µs
}

// fenetre rend le nombre d ecarts et leur somme, en microsecondes, sur la milliseconde `ms` et ses deux voisines.
func (p *pasDesEntrees) fenetre(ms int) (n int, somme uint64) {
	for k := max(ms-1, 0); k <= min(ms+1, ecartsDuPasMaxMS-1); k++ {
		n += p.rang[k]
		somme += p.somme[k]
	}
	return n, somme
}
