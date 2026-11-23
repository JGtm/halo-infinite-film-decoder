package objectives

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// score_penalties_test.go — LA PENALITE D'UN SUICIDE OU D'UNE TRAHISON ENTRE DANS LA COURBE DU SCORE PERSONNEL.
//
// ROUGE OBSERVE AVANT LA LECTURE : la plus longue suite non decroissante prenait la baisse de cent points pour du
// bruit, la courbe ne reculait jamais, et le total d'un joueur dont la penalite est le dernier mouvement gardait la
// valeur d'avant la penalite. Une penalite qui fait tomber le score sous zero etait jetee comme un ancrage parasite,
// une penalite qui le ramene a zero comme le vidage d'une ligne.

// recScore : un enregistrement de siege (compteurs de base croissants avec `n`, qui garde le score de mode croissant et
// la manche reelle) dont le score personnel vaut `personnel` ; `suicides` et `trahisons` portent le composant 4 quand
// ils sont >= 0 (-1 : l'emission ne le porte pas).
func recScore(timeMS, slot, round, n int, personnel int64, suicides, trahisons int64) types.StatRecord {
	r := recKDA(timeMS, slot, round, n, 0, 0)
	r.Comps[personalScoreComp] = types.StatValue{B: personnel}
	if suicides >= 0 {
		r.Comps[suicideBetrayalComp] = types.StatValue{A: suicides, B: trahisons}
	}
	return r
}

// valeurs rend les valeurs d'une suite.
func valeurs(pts []types.ScorePoint) []int64 {
	out := make([]int64, len(pts))
	for i, p := range pts {
		out[i] = p.Value
	}
	return out
}

// courbe rend la courbe du score personnel d'un siege dans la manche 0, telle que le tableau la publie.
func courbe(recs []types.StatRecord, c StatComponent) []int64 {
	return valeurs(SeriesByRound(recs, c, false, nil)[12][0])
}

// total rend le total du score personnel d'un siege sur le match.
func total(recs []types.StatRecord, c StatComponent) []int64 {
	return valeurs(SeriesTotal(recs, c, false, nil)[12])
}

// troisPoints : trois actions de cent points, avant une penalite.
func troisPoints() []types.StatRecord {
	return []types.StatRecord{
		recScore(10000, 12, 0, 1, 100, -1, 0), recScore(20000, 12, 0, 2, 200, -1, 0), recScore(30000, 12, 0, 3, 300, -1, 0),
	}
}

// TestPenaliteDeSuicideRecule : le compteur de suicides monte dans l'emission dont le score tombe de cent : la courbe
// recule, puis reprend.
func TestPenaliteDeSuicideRecule(t *testing.T) {
	recs := append(troisPoints(), recScore(40000, 12, 0, 4, 200, 1, 0), recScore(50000, 12, 0, 5, 300, 1, 0))
	want := []int64{100, 200, 300, 200, 300}
	for nom, c := range map[string]StatComponent{"brut": PersonalScoreComponent(), "vidage ecarte": PersonalScoreComponent().ClearsDropped()} {
		if got := courbe(recs, c); !slices.Equal(got, want) {
			t.Errorf("%s : courbe = %v, attendu %v (la penalite de suicide recule)", nom, got, want)
		}
	}
	if got, want := total(recs, PersonalScoreComponent()), want; !slices.Equal(got, want) {
		t.Errorf("total = %v, attendu %v", got, want)
	}
}

// TestPenaliteDeTrahisonEstLaDerniereDuJoueur : la penalite est le dernier mouvement du joueur : son total est celui
// d'apres la penalite (avant, le total gardait les trois cents points).
func TestPenaliteDeTrahisonEstLaDerniereDuJoueur(t *testing.T) {
	recs := append(troisPoints(), recScore(40000, 12, 0, 4, 200, 0, 1))
	got := total(recs, PersonalScoreComponent())
	if want := []int64{100, 200, 300, 200}; !slices.Equal(got, want) {
		t.Fatalf("total = %v, attendu %v : la trahison retire cent points du dernier total", got, want)
	}
}

// TestPenaliteSousZeroEstUneValeur : un joueur penalise avant d'avoir marque tombe a -100 : la valeur n'est pas un
// ancrage parasite, et le score remonte ensuite de ce qu'il marque. Une emission negative que rien ne confirme reste
// jetee, comme avant.
func TestPenaliteSousZeroEstUneValeur(t *testing.T) {
	recs := []types.StatRecord{
		recScore(10000, 12, 0, 1, 0, 0, 0), recScore(20000, 12, 0, 2, -100, 0, 1), recScore(30000, 12, 0, 3, 0, 0, 1),
	}
	if got, want := courbe(recs, PersonalScoreComponent()), []int64{0, -100, 0}; !slices.Equal(got, want) {
		t.Fatalf("courbe = %v, attendu %v", got, want)
	}
	seul := []types.StatRecord{recScore(10000, 12, 0, 1, 0, 0, 0), recScore(20000, 12, 0, 2, -100, 0, 1)}
	if got := total(seul, PersonalScoreComponent()); got[len(got)-1] != -100 {
		t.Fatalf("total = %v, attendu -100 au dernier point", got)
	}
	parasite := []types.StatRecord{recScore(10000, 12, 0, 1, 0, 0, 0), recScore(20000, 12, 0, 2, -100, 0, 0), recScore(30000, 12, 0, 3, 100, 0, 0)}
	if got, want := courbe(parasite, PersonalScoreComponent()), []int64{0, 100}; !slices.Equal(got, want) {
		t.Fatalf("emission negative sans compteur : courbe = %v, attendu %v (jetee)", got, want)
	}
}

// TestPenaliteJusquAZeroNEstPasUnVidage : le score tombe a zero parce que le joueur se suicide, compteur monte : ce
// n'est pas le vidage de sa ligne, que la lecture qui l'ecarte laissait pourtant partir.
func TestPenaliteJusquAZeroNEstPasUnVidage(t *testing.T) {
	recs := []types.StatRecord{recScore(10000, 12, 0, 1, 100, 0, 0), recScore(20000, 12, 0, 2, 0, 1, 0), recScore(30000, 12, 0, 3, 100, 1, 0)}
	if got, want := courbe(recs, PersonalScoreComponent().ClearsDropped()), []int64{100, 0, 100}; !slices.Equal(got, want) {
		t.Fatalf("courbe = %v, attendu %v", got, want)
	}
}

// TestBaisseSansCompteurResteEcartee : le vidage de la ligne d'un joueur qui part (le score retombe a zero, aucun
// compteur ne monte) et toute baisse que le film n'explique pas restent ce qu'elles etaient : ecartees.
func TestBaisseSansCompteurResteEcartee(t *testing.T) {
	recs := []types.StatRecord{
		recScore(10000, 12, 0, 1, 100, 0, 0), recScore(20000, 12, 0, 2, 200, 0, 0), recScore(30000, 12, 0, 3, 300, 0, 0),
		recScore(40000, 12, 0, 4, 200, 0, 0), recScore(50000, 12, 0, 5, 400, 0, 0),
	}
	brut := []types.ScorePoint{{TimeMS: 10000, Value: 100}, {TimeMS: 20000, Value: 200}, {TimeMS: 30000, Value: 300},
		{TimeMS: 40000, Value: 200}, {TimeMS: 50000, Value: 400}}
	if got, want := courbe(recs, PersonalScoreComponent()), valeurs(longestRun(brut, false)); !slices.Equal(got, want) {
		t.Errorf("baisse de cent sans compteur : courbe = %v, attendu %v (la plus longue suite non decroissante)", got, want)
	}
	vidage := []types.StatRecord{
		recScore(10000, 12, 0, 1, 100, 0, 0), recScore(20000, 12, 0, 2, 200, 0, 0), recScore(30000, 12, 0, 3, 0, 0, 0),
	}
	if got, want := courbe(vidage, PersonalScoreComponent().ClearsDropped()), []int64{100, 200}; !slices.Equal(got, want) {
		t.Errorf("vidage de ligne : courbe = %v, attendu %v (ecarte)", got, want)
	}
}

// TestBaisseDifferenteDeLaPenaliteResteEcartee : le compteur monte mais la baisse n'est pas celle de ses unites (une
// emission mal alignee, ou un tir qui rapporte en meme temps) : rien n'est confirme.
func TestBaisseDifferenteDeLaPenaliteResteEcartee(t *testing.T) {
	recs := append(troisPoints(), recScore(40000, 12, 0, 4, 250, 1, 0), recScore(50000, 12, 0, 5, 400, 1, 0))
	sans := append(troisPoints(), recScore(40000, 12, 0, 4, 250, -1, 0), recScore(50000, 12, 0, 5, 400, -1, 0))
	if got, want := courbe(recs, PersonalScoreComponent()), courbe(sans, PersonalScoreComponent()); !slices.Equal(got, want) {
		t.Fatalf("courbe = %v, attendu %v (celle d'avant) : une baisse de 50 avec un suicide n'est pas la penalite de 100", got, want)
	}
	if m := penaltyMarksOf(recs); len(m) != 0 {
		t.Fatalf("penalites confirmees = %v, attendu aucune", m)
	}
}

// TestDeuxPenalitesDansUneEmission : deux unites dans la meme emission retirent deux fois cent points.
func TestDeuxPenalitesDansUneEmission(t *testing.T) {
	recs := append(troisPoints(), recScore(40000, 12, 0, 4, 100, 1, 1))
	if got, want := total(recs, PersonalScoreComponent()), []int64{100, 200, 300, 100}; !slices.Equal(got, want) {
		t.Fatalf("total = %v, attendu %v", got, want)
	}
}

// TestPenaliteSansPrecedentNeConfirmeRien : la premiere emission d'un siege n'a pas de score precedent.
func TestPenaliteSansPrecedentNeConfirmeRien(t *testing.T) {
	if m := penaltyMarksOf([]types.StatRecord{recScore(10000, 12, 0, 1, -100, 1, 0)}); len(m) != 0 {
		t.Fatalf("penalites confirmees = %v, attendu aucune", m)
	}
}

// TestPenaliteNeFranchitPasLaManche : la penalite de la manche 0 est le dernier mouvement du joueur dans sa manche ; la manche
// 1 repart de zero et le total est la somme des fins de manche.
func TestPenaliteNeFranchitPasLaManche(t *testing.T) {
	recs := append(troisPoints(), recScore(40000, 12, 0, 4, 200, 1, 0),
		recScore(100000, 12, 1, 1, 0, 0, 0), recScore(110000, 12, 1, 2, 100, 0, 0), recScore(120000, 12, 1, 3, 200, 0, 0))
	got := total(recs, PersonalScoreComponent())
	if len(got) == 0 || got[len(got)-1] != 400 {
		t.Fatalf("total = %v, attendu 400 au dernier point (200 de la manche 0, 200 de la manche 1)", got)
	}
}

// TestPenaliteDuPartantNeSeReportePasSurLArrivant : un siege change d'occupant ; la penalite du partant reste dans sa
// suite, la retombee des compteurs n'est pas une penalite (aucun compteur de suicide ne monte), et l'arrivant repart
// de la derniere valeur du partant comme sans penalite.
func TestPenaliteDuPartantNeSeReportePasSurLArrivant(t *testing.T) {
	recs := []types.StatRecord{
		recScore(10000, 12, 0, 1, 100, 0, 0), recScore(20000, 12, 0, 2, 200, 0, 0), recScore(30000, 12, 0, 3, 100, 1, 0),
		recScore(40000, 12, 0, 0, 0, 0, 0), // retombee : le partant quitte
		recScore(50000, 12, 0, 1, 100, 0, 0), recScore(60000, 12, 0, 2, 200, 0, 0),
	}
	if cuts := seatCutsOf(recs).of(0, 12); !slices.Equal(cuts, []int{40000}) {
		t.Fatalf("frontieres = %v, attendu [40000] : le scenario doit etre un changement d'occupant", cuts)
	}
	got := valeurs(SeriesByRoundBySeat(recs, PersonalScoreComponent(), false, nil)[12][0])
	if want := []int64{100, 200, 100, 100, 200, 300}; !slices.Equal(got, want) {
		t.Fatalf("courbe = %v, attendu %v : la penalite du partant (200 -> 100) est publiee, l'arrivant repart de 100", got, want)
	}
}

// TestSansPenaliteLaSuiteEstCelleDEnAvant : sans penalite confirmee, la suite est [longestRun], valeur pour valeur.
func TestSansPenaliteLaSuiteEstCelleDEnAvant(t *testing.T) {
	pts := []types.ScorePoint{
		{TimeMS: 1, Slot: 12, Value: 1}, {TimeMS: 2, Slot: 12, Value: 2}, {TimeMS: 3, Slot: 12, Value: 0}, {TimeMS: 4, Slot: 12, Value: 1},
	}
	if got, want := penalisedRun(pts, false, nil), longestRun(pts, false); !slices.Equal(got, want) {
		t.Fatalf("sans penalite = %v, attendu %v", got, want)
	}
	if got, want := penalisedRun(nil, false, map[int]int{1: 1}), []types.ScorePoint(nil); !slices.Equal(got, want) {
		t.Fatalf("suite vide = %v", got)
	}
}
