package replay

import "testing"

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
			got := n.saut(x.numero, x.unite)
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
