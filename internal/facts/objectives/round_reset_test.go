package objectives

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// round_reset_test.go — L'ADMISSION D'UNE MANCHE COURTE PAR LA REMISE A ZERO SIMULTANEE.

// mancheCourteAvecRemise fabrique une manche courte : par joueur, un enregistrement de remise a zero
// (score de mode et score personnel a zero) a `debutMS + decalage[i]`, puis deux enregistrements sans
// score. `decalage` a un element par joueur ; un decalage negatif dit « ce joueur n'a pas de remise »
// (son premier enregistrement porte un seul composant).
func mancheCourteAvecRemise(round, debutMS int, decalage []int) []types.StatRecord {
	var out []types.StatRecord
	for i, d := range decalage {
		slot := 10 + 2*i
		if d < 0 {
			out = append(out, types.StatRecord{TimeMS: debutMS, Slot: slot, Round: round,
				Comps: map[int]types.StatValue{modeScoreComp: {A: 0}}})
		} else {
			out = append(out, types.StatRecord{TimeMS: debutMS + d, Slot: slot, Round: round,
				Comps: map[int]types.StatValue{modeScoreComp: {}, personalScoreComp: {}}})
		}
		out = append(out,
			types.StatRecord{TimeMS: debutMS + 5_000, Slot: slot, Round: round},
			types.StatRecord{TimeMS: debutMS + 20_000, Slot: slot, Round: round})
	}
	return out
}

// filmDeuxManches : une manche 0 pleine (admise par la matiere), puis la manche courte donnee.
func filmDeuxManches(courte []types.StatRecord) []types.StatRecord {
	return append(joueurSerie(0, 1_000, 900, 200), courte...)
}

func zeros(n int) []int { return make([]int, n) }

func TestResetSimultaneAdmetUneMancheCourte(t *testing.T) {
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, zeros(8))))
	if want := []int{0, 1}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : la manche courte dont les huit joueurs "+
			"sont remis a zero a la meme seconde doit etre admise", d.Real, want)
	}
	if len(d.Contradicted) != 0 {
		t.Errorf("contradictions = %v, attendu aucune", d.Contradicted)
	}
}

func TestResetSimultaneToleLaSecondeEntiere(t *testing.T) {
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, []int{0, 0, 300, 0, 900, 0, 1000, 0})))
	if want := []int{0, 1}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : des remises a moins d'une seconde "+
			"d'ecart sont simultanees", d.Real, want)
	}
}

func TestResetSimultaneRefuseUnJoueurPresentNonRemisAZero(t *testing.T) {
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, []int{0, 0, 0, 0, 0, 0, 0, -1})))
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : un joueur present sans remise a zero "+
			"refuse la manche", d.Real, want)
	}
}

func TestResetSimultaneRefuseDesRemisesEtalees(t *testing.T) {
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, []int{0, 3_000, 6_000, 9_000, 0, 3_000, 6_000, 9_000})))
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : des remises etalees sur plusieurs "+
			"secondes ne sont pas simultanees", d.Real, want)
	}
}

func TestResetSimultaneRefuseUneRemiseEnCoursDeManche(t *testing.T) {
	courte := mancheCourteAvecRemise(1, 803_000, zeros(8))
	// La remise de chaque joueur tombe 3 s apres un premier enregistrement de joueur : ce n'est
	// pas le debut de la manche.
	courte = append(courte, types.StatRecord{TimeMS: 800_000, Slot: 10, Round: 1})
	d := ResolveRounds(filmDeuxManches(courte))
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v", d.Real, want)
	}
}

func TestResetSimultaneRefuseUnFragmentIsole(t *testing.T) {
	// Deux slots seulement sur huit : un fragment, pas une manche entiere.
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, zeros(2))))
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : 2 joueurs sur 8 ne tiennent pas pour la "+
			"manche entiere", d.Real, want)
	}
	// Un seul joueur : sous le minimum, meme dans un film d'un seul joueur.
	if got := simultaneousResetRounds(mancheCourteAvecRemise(1, 803_000, zeros(1))); len(got) != 0 {
		t.Errorf("un seul joueur remis a zero admet %v : un enregistrement nul ne prouve rien", got)
	}
}

func TestResetSimultaneIgnoreLaMancheZero(t *testing.T) {
	if got := simultaneousResetRounds(mancheCourteAvecRemise(0, 1_000, zeros(8))); len(got) != 0 {
		t.Errorf("manche 0 admise par la remise : %v", got)
	}
}

// TestResetSimultaneNeChangeAucuneMancheDejaDecidee : sans remise, les manches admises ou refusees
// par les deux premiers criteres restent exactement celles qu'ils decidaient.
func TestResetSimultaneNeChangeAucuneMancheDejaDecidee(t *testing.T) {
	recs := fixtureManche2SansManche1()
	if got := simultaneousResetRounds(recs); len(got) != 0 {
		t.Errorf("remise vue dans une fixture sans remise : %v", got)
	}
	d := ResolveRounds(recs)
	if !reflect.DeepEqual(d.Real, []int{0}) || !reflect.DeepEqual(d.Contradicted, []int{2}) {
		t.Errorf("verdict change : retenues %v, contredites %v", d.Real, d.Contradicted)
	}
}

// La manche que seule la remise admet est comptee a part ; la manche pleine, admise par la suite ou la matiere,
// n'y figure pas.
func TestResetSimultaneEstComptee(t *testing.T) {
	d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, zeros(8))))
	if want := []int{1}; !reflect.DeepEqual(d.AdmittedByReset, want) {
		t.Errorf("manches admises par la remise = %v, attendu %v", d.AdmittedByReset, want)
	}
	if want := []int{0}; !reflect.DeepEqual(d.AdmittedByShare, want) { // la manche 0 du film fabrique n a que la matiere
		t.Errorf("manches admises par la matiere seule = %v, attendu %v", d.AdmittedByShare, want)
	}
	if d := ResolveRounds(filmDeuxManches(mancheCourteAvecRemise(1, 803_000, []int{0, 0, 5000, 0, 0, 0, 0, 0}))); len(d.AdmittedByReset) != 0 {
		t.Errorf("manche refusee : admise par la remise = %v", d.AdmittedByReset)
	}
}
