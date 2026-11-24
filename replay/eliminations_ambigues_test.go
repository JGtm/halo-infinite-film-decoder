package replay

import (
	"reflect"
	"testing"
)

// eliminations_ambigues_test.go — LA VICTIME D'UNE ÉLIMINATION QUAND PLUSIEURS CANDIDATES RESTENT : jamais appariée au plus
// proche, absente et comptée. Tout est synthétique.

// victimesDe rend, pour un tueur `k` dont le compteur monte aux images `frames` (une unité chacune), l'image:victime de chaque
// élimination et la couverture, pour les actions de score données.
func victimesDe(frames []int, actions ...ScoreAction) ([]string, EliminationCoverage) {
	var pts [][2]int
	for i, f := range frames {
		pts = append(pts, [2]int{f, i + 1})
	}
	got, cov := buildEliminations(eliminationInput{Players: []PlayerScore{joueurDeTest("k", pts, nil)}, Actions: actions})
	out := make([]string, 0, len(got))
	for _, e := range got {
		out = append(out, lignesDesEliminations([]Elimination{e})[0][:2]+"/"+e.Victim)
	}
	return out, cov
}

// Une action à l'image 11 que les éliminations des images 10 et 12 réclament toutes deux : elle n'est donnée à aucune, et
// les deux lignes sont comptées ambiguës.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : revenir à « la première qui la réclame la prend » ; ne pas écarter la candidate
// d'une ligne ambiguë ; ne pas compter l'ambiguïté.
func TestVictimeActionReclameeParDeuxImages(t *testing.T) {
	got, cov := victimesDe([]int{10, 12}, killAction(11, "k", "va"))
	if want := []string{"10/", "12/"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lignes %v, attendu %v", got, want)
	}
	if cov.VictimAmbiguous != 2 || cov.WithVictim != 0 {
		t.Errorf("couverture %+v : 2 ambiguës, 0 avec victime attendues", cov)
	}
}

// Une élimination dont deux actions voisines nomment deux victimes différentes n'en reçoit aucune ; deux actions voisines qui
// nomment la MÊME victime ne sont pas une ambiguïté.
func TestVictimeCandidatesDeNomsDifferents(t *testing.T) {
	got, cov := victimesDe([]int{10}, killAction(9, "k", "va"), killAction(11, "k", "vb"))
	if want := []string{"10/"}; !reflect.DeepEqual(got, want) || cov.VictimAmbiguous != 1 {
		t.Fatalf("lignes %v couverture %+v, attendu une ligne sans victime et une ambiguë", got, cov)
	}
	got, cov = victimesDe([]int{10}, killAction(9, "k", "va"), killAction(11, "k", "va"))
	if want := []string{"10/va"}; !reflect.DeepEqual(got, want) || cov.VictimAmbiguous != 0 {
		t.Fatalf("lignes %v couverture %+v, attendu va et aucune ambiguïté", got, cov)
	}
}

// Une candidate unique à l'image voisine reste appariée, et deux éliminations de la même image se partagent leurs deux
// actions sans ambiguïté (les lignes sont interchangeables).
func TestVictimeSansAmbiguite(t *testing.T) {
	got, cov := victimesDe([]int{10, 20, 20}, killAction(11, "k", "va"), killAction(20, "k", "vb"), killAction(20, "k", "vc"))
	if want := []string{"10/va", "20/vb", "20/vc"}; !reflect.DeepEqual(got, want) || cov.VictimAmbiguous != 0 || cov.WithVictim != 3 {
		t.Fatalf("lignes %v couverture %+v, attendu va vb vc et aucune ambiguïté", got, cov)
	}
}

// Plus d'actions candidates que d'éliminations à la même image, de victimes différentes : l'élimination n'en reçoit aucune.
func TestVictimeTropDeCandidatesALaMemeImage(t *testing.T) {
	got, cov := victimesDe([]int{10}, killAction(10, "k", "va"), killAction(10, "k", "vb"))
	if want := []string{"10/"}; !reflect.DeepEqual(got, want) || cov.VictimAmbiguous != 1 {
		t.Fatalf("lignes %v couverture %+v, attendu une ligne ambiguë", got, cov)
	}
}
