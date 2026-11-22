package objectives

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// manchesReelles rend, pour les manches 0..n-1, trois enregistrements de score de mode du slot d'equipe 6 : de quoi
// que chaque manche soit une manche reelle, dans l'ordre du temps (la manche r commence a r x 100 s).
func manchesReelles(n int) []types.StatRecord {
	var out []types.StatRecord
	for r := range n {
		for i := 1; i <= 3; i++ {
			out = append(out, types.StatRecord{TimeMS: r*100_000 + i*1_000, Slot: 6, Round: r,
				Comps: map[int]types.StatValue{0: {A: int64(i)}}})
		}
	}
	return out
}

// emission rend un enregistrement d'un slot de joueur a un instant et une manche.
func emission(t, slot, round int, comps map[int]types.StatValue) types.StatRecord {
	return types.StatRecord{TimeMS: t, Slot: slot, Round: round, Comps: comps}
}

// finaleDe rend la derniere valeur de la serie nommee du slot dans la manche, et si la serie existe.
func finaleDe(t *testing.T, recs []types.StatRecord, nom string, slot, round int, cons *ReplisALaConsultation) (int64, bool) {
	t.Helper()
	pts := SeriesByRound(recs, statNamed(t, nom).Component, false, cons)[slot][round]
	if len(pts) == 0 {
		return 0, false
	}
	return pts[len(pts)-1].Value, true
}

// TestGardeEcarteUnCompteurAuDessusDesEliminations : une emission du composant 8 mal alignee (corps a corps 15,
// grenades 84) pour 12 eliminations gagnait le departage contre l'emission juste (0, 1) ; elle est ecartee en entier,
// et comptee pour chacun des deux compteurs de son composant.
func TestGardeEcarteUnCompteurAuDessusDesEliminations(t *testing.T) {
	recs := append(manchesReelles(1),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 12, B: 3}}),
		emission(20_000, 10, 0, map[int]types.StatValue{8: {A: 15, B: 84}}),
		emission(30_000, 10, 0, map[int]types.StatValue{8: {A: 0, B: 1}}))
	cons := &ReplisALaConsultation{}
	if v, ok := finaleDe(t, recs, StatMeleeKills, 10, 0, cons); !ok || v != 0 {
		t.Errorf("corps a corps = %d (lu %v), attendu 0 : l'emission impossible ne gagne plus", v, ok)
	}
	if v, ok := finaleDe(t, recs, StatGrenadeKills, 10, 0, cons); !ok || v != 1 {
		t.Errorf("grenades = %d (lu %v), attendu 1", v, ok)
	}
	want := map[string]int{StatMeleeKills: 1, StatGrenadeKills: 1}
	if got := cons.EmissionsGardees(); !reflect.DeepEqual(got, want) {
		t.Errorf("emissions gardees = %v, attendu %v", got, want)
	}
}

// TestGardeSansAutreEmissionLaisseLeCompteurAbsent : toutes les emissions ecartees, le compteur est absent, jamais a zero.
func TestGardeSansAutreEmissionLaisseLeCompteurAbsent(t *testing.T) {
	recs := append(manchesReelles(1),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 9, B: 3}}),
		emission(20_000, 10, 0, map[int]types.StatValue{8: {A: 0, B: 16}}))
	cons := &ReplisALaConsultation{}
	if v, ok := finaleDe(t, recs, StatGrenadeKills, 10, 0, cons); ok {
		t.Errorf("grenades = %d, attendu aucune serie", v)
	}
	if got := cons.EmissionsGardees()[StatGrenadeKills]; got != 1 {
		t.Errorf("emissions gardees des grenades = %d, attendu 1", got)
	}
}

// TestGardeBorneLaSerieParLesEliminations : la plus longue serie ne depasse pas les eliminations du joueur.
func TestGardeBorneLaSerieParLesEliminations(t *testing.T) {
	comps := func(b int64) map[int]types.StatValue { return map[int]types.StatValue{5: {A: 429, B: b}} }
	recs := append(manchesReelles(1),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 4, B: 1}}),
		emission(20_000, 10, 0, comps(1)), emission(30_000, 10, 0, comps(2)),
		emission(40_000, 10, 0, comps(3)), emission(50_000, 10, 0, comps(4)),
		emission(60_000, 10, 0, comps(286)))
	cons := &ReplisALaConsultation{}
	if v, ok := finaleDe(t, recs, StatMaxSpree, 10, 0, cons); !ok || v != 4 {
		t.Errorf("serie = %d (lu %v), attendu 4", v, ok)
	}
	if got := cons.EmissionsGardees(); !reflect.DeepEqual(got, map[string]int{StatMaxSpree: 1}) {
		t.Errorf("emissions gardees = %v, attendu une serie", got)
	}
}

// TestGardeSansElimination : un joueur sans emission d'eliminations dans la manche en a zero, et ne peut avoir ni serie
// ni corps a corps.
func TestGardeSansElimination(t *testing.T) {
	recs := append(manchesReelles(1),
		emission(20_000, 10, 0, map[int]types.StatValue{5: {A: 0, B: 1}}))
	cons := &ReplisALaConsultation{}
	if v, ok := finaleDe(t, recs, StatMaxSpree, 10, 0, cons); ok {
		t.Errorf("serie = %d, attendu aucune serie", v)
	}
	if got := cons.EmissionsGardees()[StatMaxSpree]; got != 1 {
		t.Errorf("emissions gardees = %d, attendu 1", got)
	}
}

// TestGardeCasNominalInchange : des compteurs qui tiennent dans les eliminations ressortent tels quels, sans rien compter.
func TestGardeCasNominalInchange(t *testing.T) {
	recs := append(manchesReelles(1),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 12, B: 3}}),
		emission(20_000, 10, 0, map[int]types.StatValue{8: {A: 1, B: 0}, 6: {A: 2}}),
		emission(30_000, 10, 0, map[int]types.StatValue{8: {A: 2, B: 1}, 5: {A: 9, B: 12}}))
	cons := &ReplisALaConsultation{}
	for nom, want := range map[string]int64{StatMeleeKills: 2, StatGrenadeKills: 1, StatHeadshots: 2, StatMaxSpree: 12} {
		if v, ok := finaleDe(t, recs, nom, 10, 0, cons); !ok || v != want {
			t.Errorf("%s = %d (lu %v), attendu %d", nom, v, ok, want)
		}
	}
	if got := cons.EmissionsGardees(); got != nil {
		t.Errorf("emissions gardees = %v, attendu aucune", got)
	}
}

// TestGardePlusieursManches : la borne est celle de la manche, pas celle du match : 15 corps a corps passent la ou le
// joueur a 15 eliminations, et sont ecartes dans la manche ou il n'en a que 5.
func TestGardePlusieursManches(t *testing.T) {
	recs := append(manchesReelles(2),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 15, B: 3}}),
		emission(20_000, 10, 0, map[int]types.StatValue{8: {A: 15, B: 2}}),
		emission(110_000, 10, 1, map[int]types.StatValue{2: {A: 5, B: 3}}),
		emission(120_000, 10, 1, map[int]types.StatValue{8: {A: 15, B: 1}}),
		emission(130_000, 10, 1, map[int]types.StatValue{8: {A: 2, B: 1}}))
	cons := &ReplisALaConsultation{}
	if v, ok := finaleDe(t, recs, StatMeleeKills, 10, 0, cons); !ok || v != 15 {
		t.Errorf("manche 0 : corps a corps = %d (lu %v), attendu 15", v, ok)
	}
	if v, ok := finaleDe(t, recs, StatMeleeKills, 10, 1, cons); !ok || v != 2 {
		t.Errorf("manche 1 : corps a corps = %d (lu %v), attendu 2", v, ok)
	}
	if v, ok := finaleDe(t, recs, StatGrenadeKills, 10, 1, cons); !ok || v != 1 {
		t.Errorf("manche 1 : grenades = %d (lu %v), attendu 1", v, ok)
	}
	if got := cons.EmissionsGardees(); !reflect.DeepEqual(got, map[string]int{StatMeleeKills: 1, StatGrenadeKills: 1}) {
		t.Errorf("emissions gardees = %v, attendu une emission du composant 8 (manche 1)", got)
	}
}

// TestGardeNeToucheQueLesCompteursBornes : les suicides, les trahisons et les reapparitions ne comptent pas d'eliminations.
func TestGardeNeToucheQueLesCompteursBornes(t *testing.T) {
	recs := append(manchesReelles(1),
		emission(10_000, 10, 0, map[int]types.StatValue{2: {A: 1, B: 3}}),
		emission(20_000, 10, 0, map[int]types.StatValue{4: {A: 5, B: 3}, 10: {B: 7}}))
	cons := &ReplisALaConsultation{}
	for nom, want := range map[string]int64{StatSuicides: 5, StatBetrayals: 3, StatSpawns: 7} {
		if v, ok := finaleDe(t, recs, nom, 10, 0, cons); !ok || v != want {
			t.Errorf("%s = %d (lu %v), attendu %d", nom, v, ok, want)
		}
	}
	if got := cons.EmissionsGardees(); got != nil {
		t.Errorf("emissions gardees = %v, attendu aucune", got)
	}
}

// TestEmissionsGardeesNil : un enregistreur nil ne compte rien et ne panique pas.
func TestEmissionsGardeesNil(t *testing.T) {
	var cons *ReplisALaConsultation
	cons.noterEmissionGardee(statSlotKey{}, 10, 0, 1)
	if cons.EmissionsGardees() != nil {
		t.Error("un enregistreur nil ne compte rien")
	}
}
