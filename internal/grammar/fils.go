package grammar

// fils.go — L'ASSISTANT DE FILS DE LA GRAMMAIRE, L'UN DES TROIS POINTS DE LANCEMENT DE GOROUTINES DU DECODEUR
// (les deux autres, dans `replay`, sont la tache de fond et les voies du balayage ; garde-rail :
// depottest.TestUneSeuleSourceDeGoroutines, qui nomme les trois fichiers et refuse tout autre).
//
// executerEnParallele distribue des taches independantes sur au plus n fils, sans rien partager
// d'autre qu'un compteur : chaque tache s'ecrit dans un emplacement que l'appelant a prevu pour son
// seul indice, et l'appelant lit les resultats apres le retour. A n <= 1, ou pour une seule tache,
// tout s'execute dans l'appelant : aucune goroutine n'est lancee et le comportement est celui d'une
// boucle ordinaire.

import (
	"sync"
	"sync/atomic"
)

// executerEnParallele appelle fn(i) une fois pour chaque i de 0 a nbTaches-1.
//
// Avec n <= 1 ou nbTaches <= 1, les appels se font dans l'appelant, dans l'ordre des indices.
// Sinon min(n, nbTaches) fils tirent chacun l'indice suivant par un compteur atomique : l'ordre
// d'execution n'est pas defini, la jointure l'est (tous les appels sont termines au retour).
//
// Une panique dans fn est capturee ; apres la jointure, celle de la tache d'indice le plus petit
// est rejouee sur l'appelant. Une panique n'interrompt pas les autres taches deja tirees.
func executerEnParallele(suivi func(delta int), n, nbTaches int, fn func(i int)) {
	if nbTaches <= 0 {
		return
	}
	if n <= 1 || nbTaches == 1 {
		for i := 0; i < nbTaches; i++ {
			fn(i)
		}
		return
	}
	if n > nbTaches {
		n = nbTaches
	}
	var (
		suivant  atomic.Int64
		jointure sync.WaitGroup
		verrou   sync.Mutex
		panique  any
		indiceP  = -1
	)
	jointure.Add(n)
	for f := 0; f < n; f++ {
		go func() {
			defer jointure.Done()
			// Le fil cesse de compter AVANT la jointure : l'appelant peut alors rendre ses jetons.
			noterLeCalcul(suivi, 1)
			defer noterLeCalcul(suivi, -1)
			for {
				i := int(suivant.Add(1)) - 1
				if i >= nbTaches {
					return
				}
				executerUneTache(i, fn, func(v any) {
					verrou.Lock()
					if indiceP < 0 || i < indiceP {
						indiceP, panique = i, v
					}
					verrou.Unlock()
				})
			}
		}()
	}
	// L'appelant attend : il ne calcule pas pendant que ses fils le font.
	noterLeCalcul(suivi, -1)
	jointure.Wait()
	noterLeCalcul(suivi, 1)
	if indiceP >= 0 {
		panic(panique)
	}
}

// executerUneTache appelle fn(i) et remet une eventuelle panique a signaler.
func executerUneTache(i int, fn func(i int), signaler func(v any)) {
	defer func() {
		if v := recover(); v != nil {
			signaler(v)
		}
	}()
	fn(i)
}

// noterLeCalcul appelle le suivi du contexte, s'il y en a un.
func noterLeCalcul(suivi func(delta int), delta int) {
	if suivi != nil {
		suivi(delta)
	}
}
