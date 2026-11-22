package objectives

// frozen_rounds_test.go — LES MANCHES FIGEES PAR JOUEUR ([FrozenRoundsByOwner]), sur des enregistrements
// synthetiques : des manches materielles ([joueurSerie]) et, au debut de chaque manche suivante, la ligne
// figee d'un joueur.

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ligneFigee fabrique l'enregistrement du slot `slot` a l'instant `t`, de la manche `round`, qui porte les
// valeurs figees `figees` (index du compteur -> manches).
func ligneFigee(t, slot, round int, figees map[int][]types.StatFrozen) types.StatRecord {
	return types.StatRecord{TimeMS: t, Slot: slot, Round: round,
		Comps: map[int]types.StatValue{coreKillsComp: {}}, Frozen: figees}
}

// manchesMaterielles rend `n` manches retenues, la manche r commencant a r x 100 s.
func manchesMaterielles(n int) []types.StatRecord {
	var recs []types.StatRecord
	for round := range n {
		recs = append(recs, joueurSerie(round, round*100_000, 40, 0)...)
	}
	return recs
}

// proprietaire nomme le joueur d'un slot de joueur ; le slot 99 n'a personne.
func proprietaire(r types.StatRecord) string {
	if r.Slot == 99 {
		return ""
	}
	return map[int]string{10: "a", 12: "b"}[r.Slot]
}

// TestDeuxManchesUneManchefigee : la manche 0 se fige au debut de la manche 1 ; la derniere manche n'a rien.
// La remise a zero qui precede la ligne figee ne l'efface pas.
func TestDeuxManchesUneManchefigee(t *testing.T) {
	recs := manchesMaterielles(2)
	recs = append(recs,
		ligneFigee(100_000, 10, 1, map[int][]types.StatFrozen{2: {{Round: 0}}, 7: {{Round: 0}}}),
		ligneFigee(100_050, 10, 1, map[int][]types.StatFrozen{2: {{Round: 0, A: 7, B: 9}}, 7: {{Round: 0, A: 45, B: 6164}}}))
	got, ecartes := FrozenRoundsByOwner(recs, proprietaire)
	if ecartes != 0 || len(got["a"]) != 1 || got["a"][0].Round != 0 {
		t.Fatalf("manches figees %v (ecartes %d), attendu la seule manche 0 du joueur a", got, ecartes)
	}
	v := got["a"][0].Values
	for name, want := range map[string]int64{FrozenKills: 7, FrozenDeaths: 9, FrozenShotsHit: 45, FrozenAccuracy: 6164, FrozenAssists: 0} {
		if v[name] != want {
			t.Errorf("%s = %d, attendu %d", name, v[name], want)
		}
	}
	if len(v) != len(FrozenStats()) {
		t.Errorf("%d valeurs publiees, attendu une par emplacement (%d)", len(v), len(FrozenStats()))
	}
}

// TestTroisManchesDeuxManchesFigees : au debut de la manche 2, la ligne porte les manches 0 et 1 ; la valeur
// de la manche 0 deja lue au debut de la manche 1 est la meme.
func TestTroisManchesDeuxManchesFigees(t *testing.T) {
	recs := manchesMaterielles(3)
	recs = append(recs,
		ligneFigee(100_050, 12, 1, map[int][]types.StatFrozen{2: {{Round: 0, A: 4, B: 11}}}),
		ligneFigee(200_050, 12, 2, map[int][]types.StatFrozen{2: {{Round: 0, A: 4, B: 11}, {Round: 1, A: 3, B: 4}}}))
	got, _ := FrozenRoundsByOwner(recs, proprietaire)
	var kills []int64
	for _, fr := range got["b"] {
		kills = append(kills, fr.Values[FrozenKills])
	}
	if !reflect.DeepEqual(kills, []int64{4, 3}) {
		t.Errorf("eliminations figees par manche %v, attendu [4 3]", kills)
	}
}

// TestUneMancheRienDeFige : un match a une manche n'a aucune manche figee ; une valeur lue pour une manche qui
// n'est suivie d'aucune autre n'est pas publiee, et elle est comptee.
func TestUneMancheRienDeFige(t *testing.T) {
	recs := append(manchesMaterielles(1), ligneFigee(50_000, 10, 0, map[int][]types.StatFrozen{2: {{Round: 0, A: 7}}}))
	got, ecartes := FrozenRoundsByOwner(recs, proprietaire)
	if len(got) != 0 || ecartes != 1 {
		t.Errorf("manches figees %v, ecartes %d : attendu aucune et un couple ecarte", got, ecartes)
	}
}

// TestUneLigneToutANullNEstPasFigee : une ligne dont toutes les valeurs sont nulles (remise a zero, joueur
// absent de la manche) n'est pas une manche figee ; une ligne sans joueur et une ligne d'equipe non plus.
func TestUneLigneToutANullNEstPasFigee(t *testing.T) {
	recs := manchesMaterielles(2)
	recs = append(recs,
		ligneFigee(100_000, 10, 1, map[int][]types.StatFrozen{2: {{Round: 0}}}),
		ligneFigee(100_050, 99, 1, map[int][]types.StatFrozen{2: {{Round: 0, A: 5}}}),
		ligneFigee(100_050, 6, 1, map[int][]types.StatFrozen{2: {{Round: 0, A: 5}}}))
	if got, _ := FrozenRoundsByOwner(recs, func(r types.StatRecord) string {
		if r.Slot == 6 {
			return "equipe"
		}
		return proprietaire(r)
	}); len(got) != 0 {
		t.Errorf("manches figees %v, attendu aucune", got)
	}
}

// TestFrozenStatsNeSeChevauchentPas : un emplacement n'est publie qu'une fois, sous un seul nom.
func TestFrozenStatsNeSeChevauchentPas(t *testing.T) {
	vus, noms := map[StatComponent]bool{}, map[string]bool{}
	for _, s := range FrozenStats() {
		if vus[s.Component] || noms[s.Name] {
			t.Errorf("emplacement %+v ou nom %q publie deux fois", s.Component, s.Name)
		}
		vus[s.Component], noms[s.Name] = true, true
	}
}
