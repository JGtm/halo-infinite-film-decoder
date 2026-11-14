package objectives

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func TestJointRunEcarteLEmissionQuiRecule(t *testing.T) {
	pts := []valuePair{{t: 1, a: 1, b: 11}, {t: 2, a: 2, b: 118}, {t: 3, a: 3, b: 120}, {t: 4, a: 1, b: 384}}
	got := jointRun(pts)
	if len(got) != 3 || got[2].b != 120 {
		t.Errorf("suite jointe = %+v, attendu les trois premieres emissions", got)
	}
}

func TestJointRunDepartageVersLeHaut(t *testing.T) {
	got := jointRun([]valuePair{{t: 1, a: 5, b: 5}, {t: 2, a: 1, b: 9}})
	if len(got) != 1 || got[0].b != 9 {
		t.Errorf("a longueur egale la suite qui finit le plus haut l'emporte : %+v", got)
	}
	if jointRun(nil) != nil {
		t.Error("aucune emission : aucune suite")
	}
}

func TestWithoutClears(t *testing.T) {
	pts := []valuePair{{t: 1, a: 0, b: 0}, {t: 2, a: 1, b: 218}, {t: 3, a: 0, b: 0}, {t: 4, a: 2, b: 0}}
	a := withoutClears(pts, false)
	if len(a) != 3 || a[0].t != 1 || a[1].t != 2 || a[2].t != 4 {
		t.Errorf("valeur A : %+v (le zero initial reste, le retour a zero est ecarte, la valeur qui suit est lue)", a)
	}
	b := withoutClears(pts, true)
	if len(b) != 2 {
		t.Errorf("valeur B : %+v", b)
	}
}

// statNamed rend le compteur nomme de la table.
func statNamed(t *testing.T, name string) ScoreboardStat {
	t.Helper()
	for _, st := range ScoreboardStats() {
		if st.Name == name {
			return st
		}
	}
	t.Fatalf("compteur %s absent de la table", name)
	return ScoreboardStat{}
}

// TestSeriesDesCompteursNommesTiennentLeVidage : un compteur emis deux fois, dont la seconde
// emission est le vidage de la ligne, finit sur sa vraie valeur ; l'ancien choix de la suite (la
// derniere emission a longueur egale) finissait sur zero. Le vidage ecarte est note comme repli.
func TestSeriesDesCompteursNommesTiennentLeVidage(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 1_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 1, B: 218}}},
		{TimeMS: 2_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 0, B: 0}}},
	}
	st := statNamed(t, StatHeadshots)
	cons := &ReplisALaConsultation{}
	pts := rawSeries(recs, st.Component, false, cons)[10][0]
	if len(pts) != 1 || pts[0].Value != 1 {
		t.Errorf("tirs a la tete = %+v, attendu 1", pts)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 1 (le vidage)", got)
	}
	plain := rawSeriesByRound(recs, statSlotKey{Comp: 6, Side: sideA}, false, nil)[10][0]
	if kept := longestRun(plain, false); kept[len(kept)-1].Value != 0 {
		t.Errorf("le temoin (suite sans la regle) ne finit plus sur le vidage : %+v", kept)
	}
}

// TestSerieMaximaleADeuxEmissions : le maximum emis deux fois, puis vide, garde sa derniere valeur,
// et une emission a valeur negative est jetee et notee.
func TestSerieMaximaleADeuxEmissions(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 1_000, Slot: 10, Comps: map[int]types.StatValue{5: {A: 126, B: 1}}},
		{TimeMS: 2_000, Slot: 10, Comps: map[int]types.StatValue{5: {A: 271, B: 2}}},
		{TimeMS: 3_000, Slot: 10, Comps: map[int]types.StatValue{5: {A: 0, B: 0}}},
		{TimeMS: 4_000, Slot: 10, Comps: map[int]types.StatValue{5: {A: -124, B: 97}}},
	}
	cons := &ReplisALaConsultation{}
	pts := rawSeries(recs, statNamed(t, StatMaxSpree).Component, false, cons)[10][0]
	if len(pts) != 2 || pts[1].Value != 2 {
		t.Errorf("serie maximale = %+v, attendu 1 puis 2", pts)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 2 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 2 (le vidage et la valeur negative)", got)
	}
}

// TestSuiteJointeNoteLesEmissionsEcartees : une emission dont la valeur A recule est ecartee de la
// suite jointe et notee.
func TestSuiteJointeNoteLesEmissionsEcartees(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 1_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 1, B: 11}}},
		{TimeMS: 2_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 2, B: 118}}},
		{TimeMS: 3_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 3, B: 120}}},
		{TimeMS: 4_000, Slot: 10, Comps: map[int]types.StatValue{6: {A: 1, B: 384}}},
	}
	cons := &ReplisALaConsultation{}
	pts := rawSeries(recs, statNamed(t, StatHeadshots).Component, false, cons)[10][0]
	if len(pts) != 3 || pts[2].Value != 3 {
		t.Errorf("tirs a la tete = %+v, attendu 1, 2, 3", pts)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 1", got)
	}
}

// TestScoreboardStatsNeSeChevauchentPas : un nom, un emplacement, et aucun des quatre compteurs de
// base (composants 0 a 3) ni le score de mode.
func TestScoreboardStatsNeSeChevauchentPas(t *testing.T) {
	names, slots := map[string]bool{}, map[[2]int]bool{}
	for _, st := range ScoreboardStats() {
		side := 0
		if st.Component.SideB {
			side = 1
		}
		k := [2]int{st.Component.Comp, side}
		if names[st.Name] || slots[k] {
			t.Errorf("%s : nom ou emplacement deja pris", st.Name)
		}
		names[st.Name], slots[k] = true, true
		if st.Component.Comp <= coreAssistsComp {
			t.Errorf("%s : composant %d deja publie par les quatre compteurs de base", st.Name, st.Component.Comp)
		}
		if !st.Component.HoldOnClear {
			t.Errorf("%s : lu sans les regles des compteurs nommes", st.Name)
		}
	}
}
