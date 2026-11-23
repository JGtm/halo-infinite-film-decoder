package grammar

// fils.go — L'UNIQUE SOURCE DE GOROUTINES DU DECODEUR (garde-rail : depottest.TestUneSeuleSourceDeGoroutines).
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
func executerEnParallele(n, nbTaches int, fn func(i int)) {
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
	jointure.Wait()
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
