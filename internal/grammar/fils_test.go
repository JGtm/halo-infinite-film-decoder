package grammar

import (
	"runtime"
	"sync/atomic"
	"testing"
)

func TestExecuterEnParallele_AUnFilAucuneGoroutine(t *testing.T) {
	for _, n := range []int{-3, 0, 1} {
		avant := runtime.NumGoroutine()
		maxVu := 0
		appelant := 0
		executerEnParallele(nil, n, 50, func(i int) {
			appelant++
			if g := runtime.NumGoroutine(); g > maxVu {
				maxVu = g
			}
			if i != appelant-1 {
				t.Fatalf("n=%d : ordre rompu : indice %d au appel %d", n, i, appelant)
			}
		})
		if appelant != 50 {
			t.Fatalf("n=%d : %d appels, 50 attendus", n, appelant)
		}
		if maxVu > avant {
			t.Fatalf("n=%d : %d goroutines pendant l'appel, %d avant", n, maxVu, avant)
		}
	}
	// Une seule tache : dans l'appelant meme avec plusieurs fils permis.
	avant := runtime.NumGoroutine()
	executerEnParallele(nil, 8, 1, func(int) {
		if runtime.NumGoroutine() > avant {
			t.Fatal("une seule tache ne doit lancer aucune goroutine")
		}
	})
}

func TestExecuterEnParallele_ChaqueTacheUneFois(t *testing.T) {
	for _, n := range []int{2, 4, 8, 64} {
		const nb = 1000
		compte := make([]atomic.Int32, nb)
		executerEnParallele(nil, n, nb, func(i int) { compte[i].Add(1) })
		for i := range compte {
			if v := compte[i].Load(); v != 1 {
				t.Fatalf("n=%d : tache %d executee %d fois", n, i, v)
			}
		}
	}
}

func TestExecuterEnParallele_ZeroTache(t *testing.T) {
	executerEnParallele(nil, 4, 0, func(int) { t.Fatal("aucune tache attendue") })
}

func TestExecuterEnParallele_ResultatsParIndice(t *testing.T) {
	const nb = 500
	ref := make([]int, nb)
	executerEnParallele(nil, 1, nb, func(i int) { ref[i] = i*i + 7 })
	for _, n := range []int{2, 4, 8} {
		got := make([]int, nb)
		executerEnParallele(nil, n, nb, func(i int) { got[i] = i*i + 7 })
		for i := range got {
			if got[i] != ref[i] {
				t.Fatalf("n=%d : resultat %d = %d, attendu %d", n, i, got[i], ref[i])
			}
		}
	}
}

func TestExecuterEnParallele_PaniqueRejoueeSurLAppelant(t *testing.T) {
	for _, n := range []int{1, 4} {
		var fait atomic.Int32
		func() {
			defer func() {
				v := recover()
				if v != "tache-3" {
					t.Fatalf("n=%d : panique rejouee = %v, attendu tache-3 (plus petit indice)", n, v)
				}
			}()
			executerEnParallele(nil, n, 20, func(i int) {
				fait.Add(1)
				if i == 3 || i == 11 {
					panic("tache-" + map[int]string{3: "3", 11: "11"}[i])
				}
			})
			t.Fatalf("n=%d : la panique devait remonter", n)
		}()
		if n > 1 && fait.Load() != 20 {
			t.Fatalf("n=%d : %d taches executees, 20 attendues (jointure complete)", n, fait.Load())
		}
	}
}
