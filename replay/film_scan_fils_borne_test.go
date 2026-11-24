package replay

// film_scan_fils_borne_test.go — LA BORNE REELLE DES FILS QUI CALCULENT EN MEME TEMPS.
//
// La promesse de [OptionsVitesse.Fils] : a `Fils` = 1, une seule goroutine calcule (aucune n'est lancee) ; a
// `Fils` >= 2, au plus [filsMaxSimultanes] goroutines calculent en meme temps pendant une cuisson, chaine
// principale comprise, et la sortie ne change pas. Le compte vient du
// suivi que les trois points de lancement du decodeur alimentent ([grammar.FilmContext.SuivreLeCalcul]) : la chaine
// principale compte pour un, un appelant qui attend la jointure de ses fils ne compte pas.

import (
	"bytes"
	"runtime"
	"sync/atomic"
	"testing"
)

// mesurerLeCalcul cuit la bobine a `fils` fils et rend le maximum de fils qui calculaient en meme temps et
// le nombre de goroutines lancees.
func mesurerLeCalcul(t *testing.T, dossier string, fils int) (maxi, lances int64, doc, faits []byte) {
	t.Helper()
	var actifs, haut, total atomic.Int64
	actifs.Store(1) // la chaine principale
	haut.Store(1)
	suivi := func(delta int) {
		n := actifs.Add(int64(delta))
		if delta > 0 {
			total.Add(1)
		}
		for h := haut.Load(); n > h && !haut.CompareAndSwap(h, n); h = haut.Load() {
		}
	}
	doc, faits = cuireLaBobineSuivie(t, dossier, fils, suivi)
	return haut.Load(), total.Load(), doc, faits
}

// filsMaxSimultanes est la borne ecrite dans le commentaire de [OptionsVitesse.Fils] : a `Fils` >= 2, le
// pre-passage des images-cles lance `Fils` ouvriers (la chaine principale attend) pendant que la tache de fond
// et les deux voies calculent.
func filsMaxSimultanes(fils int) int64 {
	if fils <= 1 {
		return 1
	}
	return int64(fils) + 3
}

// TestFilsBorneReelleDesFilsQuiCalculent : le maximum mesure est au plus [filsMaxSimultanes], a un fil
// aucune goroutine n est lancee, et le document et les faits sont ceux d un fil.
func TestFilsBorneReelleDesFilsQuiCalculent(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(max(runtime.GOMAXPROCS(0), MaxFils)))
	for _, dossier := range bobinesCuitesDeBoutEnBout {
		_, lances, doc1, faits1 := mesurerLeCalcul(t, dossier, 1)
		if lances != 0 {
			t.Errorf("%s : %d goroutines lancees a un fil, attendu aucune", dossier, lances)
		}
		for _, fils := range []int{2, 3, 4, 8} {
			maxi, lances, doc, faits := mesurerLeCalcul(t, dossier, fils)
			t.Logf("%s : fils=%d maximum simultane=%d goroutines lancees=%d", dossier, fils, maxi, lances)
			if lances == 0 {
				t.Errorf("%s : aucune goroutine lancee a %d fils (la tache de fond au moins)", dossier, fils)
			}
			if maxi > filsMaxSimultanes(fils) {
				t.Errorf("%s : %d fils calculent en meme temps a Fils=%d, borne %d", dossier, maxi, fils, filsMaxSimultanes(fils))
			}
			if !bytes.Equal(doc, doc1) || !bytes.Equal(faits, faits1) {
				t.Errorf("%s : la sortie a %d fils differe de celle a 1 fil", dossier, fils)
			}
		}
	}
}
