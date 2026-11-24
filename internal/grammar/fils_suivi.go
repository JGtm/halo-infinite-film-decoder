package grammar

// fils_suivi.go — LE SUIVI DU NOMBRE DE FILS QUI CALCULENT, POUR LES TESTS.
//
// Chaque point de lancement du decodeur (l'assistant de [executerEnParallele], la tache de fond et les
// voies du balayage de `replay`) appelle [SuivreLeCalcul] : +1 quand un fil commence a calculer, -1 quand
// il s'arrete ; l'appelant d'une jointure se compte -1 pendant qu'il attend. En production personne ne
// pose de suivi : le cout est la lecture d'un pointeur atomique.

import "sync/atomic"

var suiviDuCalcul atomic.Pointer[func(delta int)]

// SuivreLeCalcul signale au suivi, s'il y en a un, qu'un fil de calcul commence (delta = 1) ou finit (-1).
func SuivreLeCalcul(delta int) {
	if f := suiviDuCalcul.Load(); f != nil {
		(*f)(delta)
	}
}

// PoserLeSuiviDuCalcul installe `f` comme suivi (nil le retire) et rend de quoi remettre le suivi
// precedent. Reserve aux tests : un seul suivi a la fois, pose avant les cuissons qu'il observe.
func PoserLeSuiviDuCalcul(f func(delta int)) (remettre func()) {
	var p *func(delta int)
	if f != nil {
		p = &f
	}
	avant := suiviDuCalcul.Swap(p)
	return func() { suiviDuCalcul.Store(avant) }
}
