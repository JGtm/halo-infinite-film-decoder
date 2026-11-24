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
		{TimeMS: 500, Slot: 10, Comps: map[int]types.StatValue{2: {A: 5}}}, // les eliminations qui bornent les compteurs nommes
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
	plain := rawSeriesByRound(recs, statSlotKey{Comp: 6, Side: sideA}, nil)[10][0]
	if kept := longestRun(plain, false); kept[len(kept)-1].Value != 0 {
		t.Errorf("le temoin (suite sans la regle) ne finit plus sur le vidage : %+v", kept)
	}
}

// TestSerieMaximaleADeuxEmissions : le maximum emis deux fois, puis vide, garde sa derniere valeur,
// et une emission a valeur negative est jetee et notee.
func TestSerieMaximaleADeuxEmissions(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 500, Slot: 10, Comps: map[int]types.StatValue{2: {A: 5}}}, // les eliminations qui bornent les compteurs nommes
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
		{TimeMS: 500, Slot: 10, Comps: map[int]types.StatValue{2: {A: 5}}}, // les eliminations qui bornent les compteurs nommes
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

// TestDropClearsEcarteLeVidageDUnCompteurDeBase : un compteur de base dont la ligne est videe apres sa derniere valeur finit sur
// cette valeur quand l'emplacement ecarte les vidages, et sur zero sinon ; le vidage ecarte est note comme repli.
func TestDropClearsEcarteLeVidageDUnCompteurDeBase(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 1_000, Slot: 10, Comps: map[int]types.StatValue{2: {A: 0, B: 1}}},
		{TimeMS: 2_000, Slot: 10, Comps: map[int]types.StatValue{2: {A: 0, B: 0}}},
	}
	cons := &ReplisALaConsultation{}
	garde := SeriesTotal(recs, DeathsComponent().ClearsDropped(), false, cons)[10]
	if len(garde) != 1 || garde[0].Value != 1 {
		t.Errorf("morts avec la regle = %+v, attendu 1", garde)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 1", got)
	}
	sans := SeriesTotal(recs, DeathsComponent(), false, nil)[10]
	if len(sans) == 0 || sans[len(sans)-1].Value != 0 {
		t.Errorf("morts sans la regle = %+v, attendu un vidage final a 0", sans)
	}
}

// Des enregistrements fortuits dont le canal des frags est negatif portent un canal des morts valide en apparence (16 pour 7 reelles) :
// l'emission entiere est mal alignee, et la serie des morts ne doit pas etre prolongee par elle. La regle ne vaut que pour les compteurs
// lus avec [StatComponent.ClearsDropped] : la serie brute des evenements garde son domaine.
func TestUnCanalNegatifEcarteLEmissionDesCompteursDeBase(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 100_000, Slot: 10, Comps: map[int]types.StatValue{2: {A: 3, B: 1}}},
		{TimeMS: 516_178, Slot: 10, Comps: map[int]types.StatValue{2: {A: 5, B: 7}}},
		{TimeMS: 526_885, Slot: 10, Comps: map[int]types.StatValue{2: {A: -31744, B: 16}}},
		{TimeMS: 557_751, Slot: 10, Comps: map[int]types.StatValue{2: {A: -31744, B: 16}}},
	}
	cons := &ReplisALaConsultation{}
	morts := SeriesTotal(recs, DeathsComponent().ClearsDropped(), false, cons)[10]
	if len(morts) == 0 || morts[len(morts)-1].Value != 7 {
		t.Errorf("morts = %+v, attendu une serie qui finit a 7", morts)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 2 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 2", got)
	}
	frags := SeriesTotal(recs, KillsComponent().ClearsDropped(), false, nil)[10]
	if len(frags) == 0 || frags[len(frags)-1].Value != 5 {
		t.Errorf("frags = %+v, attendu une serie qui finit a 5", frags)
	}
}

// Sans la regle du tableau des scores, la serie brute des morts garde son domaine : le pont d'identite, le drapeau et le crane lisent
// les memes emplacements et leurs sorties ne bougent pas.
func TestUnCanalNegatifNeChangePasLaSerieBrute(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 516_178, Slot: 10, Comps: map[int]types.StatValue{2: {A: 5, B: 7}}},
		{TimeMS: 526_885, Slot: 10, Comps: map[int]types.StatValue{2: {A: -31744, B: 16}}},
	}
	morts := SeriesTotal(recs, DeathsComponent(), false, nil)[10]
	if len(morts) == 0 || morts[len(morts)-1].Value != 16 {
		t.Errorf("serie brute des morts = %+v, attendu 16 (domaine inchange)", morts)
	}
}

// Le canal B du composant des assistances est un signe legitime (negatif avec chaque assistance) : l'ecarter perdrait les assistances.
func TestLeCanalNegatifDesAssistancesNEstPasEcarte(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 83_153, Slot: 10, Comps: map[int]types.StatValue{3: {A: 1, B: 33}}},
		{TimeMS: 233_006, Slot: 10, Comps: map[int]types.StatValue{3: {A: 3, B: -100}}},
		{TimeMS: 440_685, Slot: 10, Comps: map[int]types.StatValue{3: {A: 5, B: -533}}},
	}
	assists := SeriesTotal(recs, AssistsComponent().ClearsDropped(), false, nil)[10]
	if len(assists) == 0 || assists[len(assists)-1].Value != 5 {
		t.Errorf("assistances = %+v, attendu une serie qui finit a 5", assists)
	}
}

// Une emission des assistances dont le canal C est negatif est mal alignee : elle ne prolonge pas la serie des assistances (8 pour 4),
// alors que les canaux B et D negatifs des emissions vraies restent des valeurs.
func TestUnCanalCNegatifEcarteLEmissionDesAssistances(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 282_228, Slot: 10, Comps: map[int]types.StatValue{3: {A: 1, B: 133, D: 133, HasD: true}}},
		{TimeMS: 414_828, Slot: 10, Comps: map[int]types.StatValue{3: {A: 3, B: -100, D: -100, HasD: true}}},
		{TimeMS: 469_882, Slot: 10, Comps: map[int]types.StatValue{3: {A: 4, B: -266, D: -266, HasD: true}}},
		{TimeMS: 552_222, Slot: 10, Comps: map[int]types.StatValue{3: {A: 8, B: 8, C: -9695, HasC: true}}},
	}
	cons := &ReplisALaConsultation{}
	assists := SeriesTotal(recs, AssistsComponent().ClearsDropped(), false, cons)[10]
	if len(assists) == 0 || assists[len(assists)-1].Value != 4 {
		t.Errorf("assistances = %+v, attendu une serie qui finit a 4", assists)
	}
	if got := cons.ComptesDesReplis().EmissionsHorsDomaineJetees; got != 1 {
		t.Errorf("%d emission(s) jetee(s) notee(s), attendu 1", got)
	}
	brut := SeriesTotal(recs, AssistsComponent(), false, nil)[10]
	if len(brut) == 0 || brut[len(brut)-1].Value != 8 {
		t.Errorf("serie brute des assistances = %+v, attendu 8 (domaine inchange)", brut)
	}
}

func TestNegativeCounter(t *testing.T) {
	cas := []struct {
		nom  string
		comp int
		v    types.StatValue
		want bool
	}{
		{"frags et morts valides", coreKillsComp, types.StatValue{A: 5, B: 7}, false},
		{"frags negatifs", coreKillsComp, types.StatValue{A: -1, B: 7}, true},
		{"morts negatives", coreKillsComp, types.StatValue{A: 5, B: -1}, true},
		{"assistances et points signes", coreAssistsComp, types.StatValue{A: 1, B: -66, D: -300, HasD: true}, false},
		{"assistances negatives", coreAssistsComp, types.StatValue{A: -1}, true},
		{"canal C des assistances negatif", coreAssistsComp, types.StatValue{A: 8, B: 8, C: -9695, HasC: true}, true},
		{"canal C des assistances positif", coreAssistsComp, types.StatValue{A: 2, C: 28, HasC: true}, false},
		{"autre composant", 4, types.StatValue{A: -1, B: -1, C: -1}, false},
	}
	for _, c := range cas {
		if got := negativeCounter(c.comp, c.v); got != c.want {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.want)
		}
	}
}

// TestCompteursDeModeDuTableau : le VIP ajoute trois compteurs, qui ne reprennent ni un nom ni un emplacement de la table de base ;
// une autre famille, ou aucune, rend exactement la table de base.
func TestCompteursDeModeDuTableau(t *testing.T) {
	base := ScoreboardStats()
	vip := ScoreboardStatsOf(ObjectiveTypeVip)
	if len(vip) != len(base)+3 {
		t.Fatalf("VIP : %d compteurs, attendu %d", len(vip), len(base)+3)
	}
	prisN, slots := map[string]bool{}, map[[2]int]bool{}
	for _, st := range base {
		prisN[st.Name] = true
		slots[[2]int{st.Component.Comp, b2i(st.Component.SideB)}] = true
	}
	for _, st := range vip[len(base):] {
		k := [2]int{st.Component.Comp, b2i(st.Component.SideB)}
		if prisN[st.Name] || slots[k] || !st.Component.HoldOnClear {
			t.Errorf("%s : nom ou emplacement deja pris, ou lu sans les regles des compteurs nommes", st.Name)
		}
	}
	for _, famille := range []string{"", ObjectiveTypeFlag, ObjectiveTypeZone} {
		if got := ScoreboardStatsOf(famille); len(got) != len(base) {
			t.Errorf("famille %q : %d compteurs, attendu la table de base (%d)", famille, len(got), len(base))
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
