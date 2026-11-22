package objectives

// named_paires_de_compteurs_test.go — UNE EMISSION D'UN COMPOSANT A DEUX COMPTEURS EST JETEE EN ENTIER DES QU'UN
// DE SES DEUX CANAUX SORT DU DOMAINE D'UN COMPTEUR ([pairedCounterComps], [emissionHorsDomaine]).
//
// Les composants `3` et `25` n'en font pas partie (leur canal B n'est pas un compteur), et le composant `0` garde
// sa regle propre. Chaque test est un garde-fou de la regle : retirer la ligne `pairedCounterComps[key.Comp] && ...`
// de [emissionHorsDomaine] les fait rougir (le premier sur ses lignes « jetee », le second sur le nombre
// d'actions publiees).

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestEmissionDeDeuxCompteursJeteeEnEntier : la table dit, pour chaque emission, si elle est jetee, lue sur l'un
// ou l'autre de ses deux canaux.
func TestEmissionDeDeuxCompteursJeteeEnEntier(t *testing.T) {
	joueur, equipe := 10, 6
	cas := []struct {
		nom   string
		comp  int
		v     types.StatValue
		slot  int
		jetee bool
	}{
		{"vol et eliminations aberrantes", 24, types.StatValue{A: 12, B: 7034}, joueur, true},
		{"canal B negatif", 24, types.StatValue{A: 3, B: -79}, joueur, true},
		{"canal A aberrant, canal B sain", 24, types.StatValue{A: 12489, B: 2}, joueur, true},
		{"emission saine", 24, types.StatValue{A: 1, B: 0}, joueur, false},
		{"valeurs hautes mais saines", 24, types.StatValue{A: 12, B: 66}, joueur, false},
		{"bornes incluses", 24, types.StatValue{A: statMaxModeScore, B: statMaxModeScore}, joueur, false},
		{"juste au-dela de la borne", 24, types.StatValue{A: statMaxModeScore + 1, B: 0}, joueur, true},
		{"eliminations", 2, types.StatValue{A: 5, B: 9000}, joueur, true},
		{"eliminations redondantes", 12, types.StatValue{A: 5, B: 9000}, joueur, true},
		{"assistances de capture", 20, types.StatValue{A: 0, B: 9000}, joueur, true},
		{"captures", 21, types.StatValue{A: 5, B: -1}, joueur, true},
		{"prises", 22, types.StatValue{A: 9000, B: 0}, joueur, true},
		{"retours", 23, types.StatValue{A: 5, B: 9000}, joueur, true},
		{"assistances : canal B libre", 3, types.StatValue{A: 4, B: 7034}, joueur, false},
		{"canal B du composant 25 libre", 25, types.StatValue{A: 4, B: 7034}, joueur, false},
		{"score de mode, canal B aberrant", modeScoreComp, types.StatValue{A: 66, B: 16635}, joueur, true},
		{"score de mode sain", modeScoreComp, types.StatValue{A: 66, B: 0}, joueur, false},
		{"score de mode d'equipe, canal B non nul", modeScoreComp, types.StatValue{A: 3, B: 5}, equipe, true},
	}
	for _, c := range cas {
		for _, cote := range []string{sideA, sideB} {
			val := c.v.A
			if cote == sideB {
				val = c.v.B
			}
			// Un canal negatif se juge seul, quel que soit le composant : on ne teste ici que la regle de la paire.
			if val < 0 {
				continue
			}
			if got := emissionHorsDomaine(statSlotKey{Comp: c.comp, Side: cote}, c.slot, c.v, val); got != c.jetee {
				t.Errorf("%s (composant %d, canal %s lu) : jetee=%v, attendu %v", c.nom, c.comp, cote, got, c.jetee)
			}
		}
	}
}

// TestEmissionDeDeuxCompteursAbsenteDeLaSerieEtDesActions : une emission `24 = 12 / 7034` n'ouvre aucune serie de
// vols (donc aucune action), alors qu'une emission `24 = 1 / 0` en ouvre une ; l'emission jetee est comptee une fois
// par serie.
func TestEmissionDeDeuxCompteursAbsenteDeLaSerieEtDesActions(t *testing.T) {
	base := onePassCorpus()
	vols := func(recs []types.StatRecord) (n int) {
		for _, e := range NamedEventsFrom(recs, ObjectiveTypeFlag, nil) {
			if e.Stat == StatFlagSteals && e.Slot == 10 {
				n++
			}
		}
		return n
	}
	avecVol := func(a, b int64) []types.StatRecord {
		return append(append([]types.StatRecord{}, base...),
			types.StatRecord{TimeMS: 4500, Slot: 10, Round: 0, Comps: map[int]types.StatValue{24: {A: a, B: b}}})
	}
	if got := vols(base); got != 0 {
		t.Fatalf("le corpus de base porte deja %d vols : le test serait vacant", got)
	}
	if got := vols(avecVol(1, 0)); got != 1 {
		t.Fatalf("emission saine `24 = 1 / 0` : %d vols, attendu 1", got)
	}
	if got := vols(avecVol(12, 7034)); got != 0 {
		t.Fatalf("emission `24 = 12 / 7034` : %d vols publies, attendu 0 (emission jetee en entier)", got)
	}

	cons := &ReplisALaConsultation{}
	table := map[statSlotKey]statSlot{{24, sideA}: {Stat: StatFlagSteals}}
	rawSeriesByKey(avecVol(12, 7034), table, cons)
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Fatalf("emission jetee comptee %d fois, attendu 1", got)
	}
}
