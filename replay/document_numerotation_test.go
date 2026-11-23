package replay

import "testing"

// dixSecondes sépare les événements des cas : assez pour qu'un tour de 256 coups tienne à la cadence maximale des armes.
const dixSecondes = 10_000_000

// TestLaNumerotationNeCompteJamaisUnRetard : un numéro en retard remplit un trou que l'unité a sauté ; il ne compte rien et ne
// déplace pas la tête. Un numéro derrière la tête qui n'est dans aucun trou, déjà vu comme événement ou déjà rempli, ou dont le trou
// est celui d'une autre unité, est un saut avant.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter le retard comme un saut ; déplacer la tête sur lui ; remplir un trou deux fois ;
// ignorer l'unité du trou ; accepter un retard de plus d'un demi-tour.
func TestLaNumerotationNeCompteJamaisUnRetard(t *testing.T) {
	type tir struct {
		numero int
		unite  uint32
	}
	cas := []struct {
		nom    string
		tirs   []tir
		sauts  []int
		retard int // le nombre de tirs que l'état doit compter au total
	}{
		{"un retard dans un trou", []tir{{116, 1}, {118, 1}, {121, 1}, {117, 1}, {123, 1}}, []int{1, 2, 3, 0, 2}, 8},
		{"un numéro déjà vu derrière la tête est un tour", []tir{{10, 1}, {11, 1}, {12, 1}, {11, 1}}, []int{1, 1, 1, 255}, 258},
		{"un trou ne se remplit qu'une fois", []tir{{10, 1}, {14, 1}, {12, 1}, {12, 1}}, []int{1, 4, 0, 254}, 259},
		{"le trou d'une autre unité ne se remplit pas", []tir{{10, 1}, {14, 1}, {12, 2}}, []int{1, 4, 254}, 259},
		{"un saut entre deux unités ne laisse aucun trou", []tir{{10, 1}, {14, 2}, {12, 2}}, []int{1, 4, 254}, 259},
		{"au-delà d'un demi-tour c'est un saut avant", []tir{{10, 1}, {200, 1}, {70, 1}}, []int{1, 190, 126}, 317},
		{"un retard de l'autre côté du retour à zéro", []tir{{254, 1}, {2, 1}, {255, 1}}, []int{1, 4, 0}, 5},
	}
	for _, c := range cas {
		var n numerotationDuTireur
		total := 0
		for i, x := range c.tirs {
			got := n.saut(x.numero, x.unite, uint64(i)*dixSecondes)
			total += got
			if got != c.sauts[i] {
				t.Errorf("%s : le tir %d (numéro %d) ajoute %d, attendu %d", c.nom, i, x.numero, got, c.sauts[i])
			}
		}
		if total != c.retard {
			t.Errorf("%s : %d tirs comptés, attendu %d", c.nom, total, c.retard)
		}
	}
}

// TestLaNumerotationNeCompteJamaisUneRelecture : le film réécrit un événement déjà écrit, mêlé à des tirs neufs (mesuré : 3, 2, 4, 3 dans un
// même paquet). Un numéro que la même unité a déjà écrit, derrière la tête, revenu plus vite que la cadence maximale ne laisse faire un
// tour, est une relecture : il n'ajoute rien et ne déplace pas la tête. Revenu assez tard pour un tour, ou écrit par une autre unité, ou
// jamais écrit, il garde sa règle.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter la relecture comme un tour ; la déplacer sur la tête ; ignorer l'unité ; ignorer le temps ;
// relire un numéro que personne n'a écrit.
func TestLaNumerotationNeCompteJamaisUneRelecture(t *testing.T) {
	type tir struct {
		numero  int
		unite   uint32
		instant uint64
	}
	const ms = 1000
	cas := []struct {
		nom   string
		tirs  []tir
		sauts []int
	}{
		{"deux suites entrelacées d'une seule unité (relectures dans le paquet)",
			[]tir{{2, 7, 0}, {3, 7, 500 * ms}, {2, 7, 500 * ms}, {4, 7, 500 * ms}, {3, 7, 500 * ms}, {2, 7, 633 * ms}, {5, 7, 633 * ms}, {4, 7, 633 * ms}},
			[]int{1, 1, 0, 1, 0, 0, 1, 0}},
		{"une seule suite : rien ne change", []tir{{10, 1, 0}, {11, 1, 100 * ms}, {12, 1, 200 * ms}, {13, 1, 300 * ms}}, []int{1, 1, 1, 1}},
		{"la suite repart après 255 : un saut avant de un", []tir{{254, 1, 0}, {255, 1, 100 * ms}, {0, 1, 200 * ms}, {1, 1, 300 * ms}}, []int{1, 1, 1, 1}},
		{"le même numéro revenu après un tour possible est un tour", []tir{{10, 1, 0}, {11, 1, 100 * ms}, {12, 1, 200 * ms}, {11, 1, 200*ms + dixSecondes}}, []int{1, 1, 1, 255}},
		{"un numéro écrit par une autre unité n'est pas relu", []tir{{10, 1, 0}, {11, 1, 100 * ms}, {12, 1, 200 * ms}, {11, 2, 300 * ms}}, []int{1, 1, 1, 255}},
		{"un numéro jamais écrit n'est pas une relecture", []tir{{10, 1, 0}, {14, 1, 100 * ms}, {13, 2, 200 * ms}}, []int{1, 4, 255}},
		{"une relecture ne déplace pas la tête", []tir{{10, 1, 0}, {11, 1, 100 * ms}, {10, 1, 150 * ms}, {12, 1, 200 * ms}}, []int{1, 1, 0, 1}},
	}
	for _, c := range cas {
		var n numerotationDuTireur
		for i, x := range c.tirs {
			if got := n.saut(x.numero, x.unite, x.instant); got != c.sauts[i] {
				t.Errorf("%s : le tir %d (numéro %d) ajoute %d, attendu %d", c.nom, i, x.numero, got, c.sauts[i])
			}
		}
	}
}

// TestLaCadenceMaximaleEstCelleDeLaTable : la borne d'une relecture vient de la table des armes, pas d'un nombre posé ici.
func TestLaCadenceMaximaleEstCelleDeLaTable(t *testing.T) {
	if c := cadenceMaximaleDesArmes(); c != 60 {
		t.Errorf("cadence maximale = %v, attendu 60 (Rayon de Sentinelle)", c)
	}
}
