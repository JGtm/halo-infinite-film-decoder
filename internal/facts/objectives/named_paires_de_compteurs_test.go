package objectives

// named_paires_de_compteurs_test.go — UNE EMISSION EST JETEE EN ENTIER QUAND LA TABLE DU MODE NOMME SES DEUX CANAUX
// COMME COMPTEURS ET QU'UN DES DEUX SORT DU DOMAINE D'UN COMPTEUR ([emissionHorsDomaine], [namedStatSlots]).
//
// Un canal frere que la table ne nomme pas n'est jamais juge : en zones `22 B` est une duree (470, 555, ... 3670), et
// l'ecarter ferait perdre les valeurs justes de `22 A` (`zone_defensive_kills`). Le composant `0` garde sa regle
// propre. Le controle de borne ne s'applique que si la table nomme les DEUX canaux du composant : un composant dont
// la table ne nomme qu'un canal (`2 A`, `22 A`) n'est juge que sur la valeur negative du canal lu. Chaque test est un
// garde-fou : retirer le controle des deux canaux nommes de [emissionHorsDomaine] fait rougir les cas « jetee » ;
// juger les canaux d'office, ou un canal nomme sans son frere, fait rougir les cas « gardee ».

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestEmissionJeteeSelonLaTableDuMode : pour chaque emission et chaque canal lu, la table dit si elle est jetee.
func TestEmissionJeteeSelonLaTableDuMode(t *testing.T) {
	joueur, equipe := 10, 6
	cas := []struct {
		nom     string
		famille string // "" : lecture hors de toute famille (table nil)
		comp    int
		v       types.StatValue
		slot    int
		jetee   bool
	}{
		{"Drapeau 24, vol et eliminations aberrantes", ObjectiveTypeFlag, 24, types.StatValue{A: 12, B: 7034}, joueur, true},
		{"Drapeau 24, canal A aberrant", ObjectiveTypeFlag, 24, types.StatValue{A: 12489, B: 2}, joueur, true},
		{"Drapeau 24, saine", ObjectiveTypeFlag, 24, types.StatValue{A: 1, B: 0}, joueur, false},
		{"Drapeau 24, hautes mais saines", ObjectiveTypeFlag, 24, types.StatValue{A: 12, B: 66}, joueur, false},
		{"Drapeau 24, bornes incluses", ObjectiveTypeFlag, 24, types.StatValue{A: statMaxModeScore, B: statMaxModeScore}, joueur, false},
		{"Drapeau 24, juste au-dela de la borne", ObjectiveTypeFlag, 24, types.StatValue{A: statMaxModeScore + 1, B: 0}, joueur, true},
		{"Drapeau 22, retours aberrants", ObjectiveTypeFlag, 22, types.StatValue{A: 3, B: 9000}, joueur, true},
		{"Drapeau 3, assistances : canal B non nomme", ObjectiveTypeFlag, 3, types.StatValue{A: 4, B: 7034}, joueur, false},
		{"Drapeau 25, canal B non nomme", ObjectiveTypeFlag, 25, types.StatValue{A: 4, B: 7034}, joueur, false},
		{"Drapeau 12, emplacements redondants nommes", ObjectiveTypeFlag, 12, types.StatValue{A: 5, B: 9000}, joueur, true},
		{"Drapeau 2, canal des morts non nomme", ObjectiveTypeFlag, 2, types.StatValue{A: 5, B: 9000}, joueur, false},
		{"Drapeau 2, A nomme seul au-dela de la borne : garde", ObjectiveTypeFlag, 2, types.StatValue{A: statMaxModeScore + 1, B: 0}, joueur, false},
		{"zones 2, A nomme seul au-dela de la borne : garde", ObjectiveTypeZone, 2, types.StatValue{A: statMaxModeScore + 1, B: 0}, joueur, false},
		{"zones 22, A nomme seul, B grand : garde", ObjectiveTypeZone, 22, types.StatValue{A: 1, B: 7034}, joueur, false},
		{"zones 22, duree au canal B : gardee", ObjectiveTypeZone, 22, types.StatValue{A: 1, B: 3670}, joueur, false},
		{"zones 21, deux canaux nommes", ObjectiveTypeZone, 21, types.StatValue{A: 5, B: 9000}, joueur, true},
		{"zones 20, canal A non nomme", ObjectiveTypeZone, 20, types.StatValue{A: 9000, B: 2}, joueur, false},
		{"VIP 22, canal B non nomme : garde", ObjectiveTypeVip, 22, types.StatValue{A: 4, B: 7034}, joueur, false},
		{"VIP 23, aucun canal nomme : garde", ObjectiveTypeVip, 23, types.StatValue{A: 900, B: 7034}, joueur, false},
		{"hors famille : le canal frere n'est pas juge", "", 24, types.StatValue{A: 12, B: 7034}, joueur, false},
		{"hors famille, eliminations et morts", "", 2, types.StatValue{A: 5, B: 9000}, joueur, false},
		{"score de mode, canal B aberrant", ObjectiveTypeFlag, modeScoreComp, types.StatValue{A: 66, B: 16635}, joueur, true},
		{"score de mode sain", ObjectiveTypeFlag, modeScoreComp, types.StatValue{A: 66, B: 0}, joueur, false},
		{"score de mode d'equipe, canal B non nul", ObjectiveTypeFlag, modeScoreComp, types.StatValue{A: 3, B: 5}, equipe, true},
		{"score de mode hors famille, canal B aberrant", "", modeScoreComp, types.StatValue{A: 66, B: 16635}, joueur, true},
	}
	for _, c := range cas {
		for _, cote := range []string{sideA, sideB} {
			// Un canal negatif se juge seul, quelle que soit la table : on ne teste ici que la regle du canal frere.
			if canalDe(c.v, cote) < 0 {
				continue
			}
			got := emissionHorsDomaine(statSlotKey{Comp: c.comp, Side: cote}, c.slot, c.v, namedStatSlots[c.famille])
			if got != c.jetee {
				t.Errorf("%s (composant %d, canal %s lu) : jetee=%v, attendu %v", c.nom, c.comp, cote, got, c.jetee)
			}
		}
	}
}

// TestCanalNegatifJugeSeulQuelleQueSoitLaTable : un canal lu negatif est un ancrage parasite, avec ou sans table ; le
// canal frere sain d'une emission dont l'autre canal est negatif n'est pas juge hors famille.
func TestCanalNegatifJugeSeulQuelleQueSoitLaTable(t *testing.T) {
	v := types.StatValue{A: 3, B: -79}
	for _, table := range []map[statSlotKey]statSlot{nil, namedStatSlots[ObjectiveTypeFlag], namedStatSlots[ObjectiveTypeZone]} {
		if !emissionHorsDomaine(statSlotKey{Comp: 24, Side: sideB}, 10, v, table) {
			t.Errorf("canal B negatif lu garde (table de %d ligne(s))", len(table))
		}
	}
	if emissionHorsDomaine(statSlotKey{Comp: 24, Side: sideA}, 10, v, nil) {
		t.Error("hors famille, le canal A sain d'une emission dont B est negatif n'est pas juge")
	}
}

// TestEmissionAbsenteDeLaSerieSelonLaTableDuMode : en Drapeau, `24 = 12 / 7034` n'ouvre ni vols ni eliminations en
// portant, alors que `24 = 1 / 0` ouvre un vol ; en zones, les emissions `22 = n / duree` gardent la serie de
// `zone_defensive_kills` intacte ; l'emission jetee est comptee une fois ; et sans le frere nomme par la table (preuve
// d'echec de la regle) l'emission `24 = 12 / 7034` reste dans la serie.
func TestEmissionAbsenteDeLaSerieSelonLaTableDuMode(t *testing.T) {
	base := onePassCorpus()
	avec := func(extra ...types.StatRecord) []types.StatRecord {
		return append(append([]types.StatRecord{}, base...), extra...)
	}
	emission := func(ms, comp int, a, b int64) types.StatRecord {
		return types.StatRecord{TimeMS: ms, Slot: 31, Round: 0, Comps: map[int]types.StatValue{comp: {A: a, B: b}}}
	}
	compte := func(recs []types.StatRecord, famille, stat string) (n int) {
		for _, e := range NamedEventsFrom(recs, famille, nil) {
			if e.Stat == stat && e.Slot == 31 {
				n++
			}
		}
		return n
	}
	if got := compte(base, ObjectiveTypeFlag, StatFlagSteals); got != 0 {
		t.Fatalf("le corpus de base porte deja %d vols : le test serait vacant", got)
	}
	if got := compte(avec(emission(4500, 24, 1, 0)), ObjectiveTypeFlag, StatFlagSteals); got != 1 {
		t.Fatalf("emission saine `24 = 1 / 0` : %d vols, attendu 1", got)
	}
	aberrante := avec(emission(4500, 24, 12, 7034))
	if got := compte(aberrante, ObjectiveTypeFlag, StatFlagSteals); got != 0 {
		t.Fatalf("emission `24 = 12 / 7034` : %d vols publies, attendu 0 (jetee en entier)", got)
	}
	if got := compte(aberrante, ObjectiveTypeFlag, StatFlagKillsAsCarrier); got != 0 {
		t.Fatalf("emission `24 = 12 / 7034` : %d eliminations en portant, attendu 0 (jetee sur les deux cotes)", got)
	}
	jetees := func(recs []types.StatRecord, famille string) int {
		cons := &ReplisALaConsultation{}
		rawSeriesByKey(recs, namedStatSlots[famille], cons)
		return cons.ComptesDesReplis().EmissionsHorsDomaineJetees
	}
	// Le corpus porte deja ses jetees (negatif, score de mode) : l'emission aberrante en ajoute une par serie, `24 A` et `24 B`.
	if avant, apres := jetees(base, ObjectiveTypeFlag), jetees(aberrante, ObjectiveTypeFlag); apres-avant != 2 {
		t.Fatalf("emission aberrante comptee %d fois, attendu 2 (une par serie nommee)", apres-avant)
	}
	// Preuve d'echec : sans `24 B` nomme, le canal frere n'est pas juge et l'emission reste dans la serie.
	sansFrere := map[statSlotKey]statSlot{{24, sideA}: {Stat: StatFlagSteals}}
	if got := len(rawSeriesByKey(aberrante, sansFrere, nil)[statSlotKey{24, sideA}][31][0]); got != 1 {
		t.Fatalf("sans canal frere nomme, l'emission `24 = 12 / 7034` doit rester dans la serie : %d point(s)", got)
	}

	// Zones : `22 B` est une duree que la table ne nomme pas ; la serie de `22 A` reste intacte.
	zones := avec(emission(4000, 22, 1, 470), emission(5000, 22, 2, 555), emission(6000, 22, 3, 3670))
	if got := compte(zones, ObjectiveTypeZone, StatZoneDefensiveKills); got != 3 {
		t.Fatalf("zones, `22 = n / duree` : %d defenses, attendu 3 (la serie de `22 A` est intacte)", got)
	}
	if avant, apres := jetees(base, ObjectiveTypeZone), jetees(zones, ObjectiveTypeZone); apres != avant {
		t.Fatalf("zones : %d emission(s) jetee(s) de plus, attendu 0", apres-avant)
	}
}
