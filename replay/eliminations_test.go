package replay

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// eliminations_test.go — LE CALQUE DES ÉLIMINATIONS : tout est synthétique, les identifiants sont des étiquettes.

// totalOf pose une courbe cumulée de points (image, valeur).
func totalOf(points ...[2]int) ScoreSeries {
	var s ScoreSeries
	for _, p := range points {
		s.Total = append(s.Total, ScoreTick{T: p[0], V: p[1]})
	}
	return s
}

// joueurDeTest : un joueur dont les éliminations montent aux points `kills` et les tirs à la tête aux points `heads`.
// `heads` nil : aucun compteur de tirs à la tête.
func joueurDeTest(xuid string, kills, heads []([2]int)) PlayerScore {
	p := PlayerScore{XUID: xuid, Kills: totalOf(kills...)}
	if heads != nil {
		p.Stats = map[string]ScoreSeries{objectives.StatHeadshots: totalOf(heads...)}
	}
	return p
}

func killAction(t int, killer, victim string) ScoreAction {
	return ScoreAction{T: t, XUID: killer, Name: killScoreActionName, Points: 100, Target: victim}
}

func perfectMedal(t int, xuid string) Medal {
	return Medal{T: t, XUID: xuid, ID: 109, Name: medalPerfectName}
}

// marque rend "v", "f" ou "?" selon l'état d'un pointeur de marque.
func marque(b *bool) string {
	switch {
	case b == nil:
		return "?"
	case *b:
		return "v"
	}
	return "f"
}

// lignesDesEliminations rend, pour chaque élimination, "image:tête:parfait:victime".
func lignesDesEliminations(es []Elimination) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, strconv.Itoa(e.T)+":"+marque(e.Headshot)+":"+marque(e.Perfect)+":"+e.Victim)
	}
	return out
}

// Cas nominal : trois éliminations, la deuxième est un tir à la tête, la troisième un frag parfait ; les victimes se
// joignent à une image près.
func TestEliminationsNominal(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{joueurDeTest("k", [][2]int{{10, 1}, {20, 2}, {30, 3}}, [][2]int{{20, 1}})},
		Actions: []ScoreAction{killAction(10, "k", "va"), killAction(21, "k", "vb"), killAction(30, "k", "vc"), killAction(30, "autre", "vz")},
		Medals:  []Medal{perfectMedal(31, "k"), {T: 10, XUID: "k", Name: "Double Kill"}},
	}
	got, cov := buildEliminations(in)
	want := []string{"10:f:?:va", "20:v:?:vb", "30:f:v:vc"}
	if !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	wantCov := EliminationCoverage{Eliminations: 3, WithVictim: 3, HeadYes: 1, HeadNo: 2, PerfectYes: 1, PerfectUnknown: 2}
	if cov != wantCov {
		t.Errorf("couverture = %+v, attendu %+v", cov, wantCov)
	}
}

// Deux éliminations dans la même image, une seule montée : aucune des deux n'est marquée, et c'est compté.
func TestEliminationsDeuxEliminationsUneMontee(t *testing.T) {
	in := eliminationInput{Players: []PlayerScore{joueurDeTest("k", [][2]int{{10, 2}}, [][2]int{{10, 1}})}}
	got, cov := buildEliminations(in)
	if want := []string{"10:?:?:", "10:?:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadUnknown != 2 || cov.HeadAmbiguous != 2 || cov.HeadYes != 0 || cov.HeadNo != 0 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Deux éliminations dans la même image, deux montées : les deux sont à la tête.
func TestEliminationsDeuxEliminationsDeuxMontees(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{joueurDeTest("k", [][2]int{{10, 2}}, [][2]int{{10, 2}})},
		Actions: []ScoreAction{killAction(10, "k", "va"), killAction(10, "k", "vb")},
	}
	got, cov := buildEliminations(in)
	if want := []string{"10:v:?:va", "10:v:?:vb"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadYes != 2 || cov.HeadUnknown != 0 || cov.WithVictim != 2 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Sans compteur de tirs à la tête, la marque est inconnue, jamais fausse : joueur sans `stats`, joueur dont `stats` ne
// porte pas ce compteur. Un compteur publié et resté à zéro prouve le faux.
func TestEliminationsJoueurSansCompteur(t *testing.T) {
	sansStats := joueurDeTest("a", [][2]int{{10, 1}}, nil)
	autreCompteur := joueurDeTest("b", [][2]int{{10, 1}}, nil)
	autreCompteur.Stats = map[string]ScoreSeries{objectives.StatMeleeKills: totalOf([2]int{10, 1})}
	zero := joueurDeTest("c", [][2]int{{10, 1}}, [][2]int{{0, 0}})
	got, cov := buildEliminations(eliminationInput{Players: []PlayerScore{sansStats, autreCompteur, zero}})
	if want := []string{"10:?:?:", "10:?:?:", "10:f:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadNoCounter != 2 || cov.HeadNo != 1 || cov.HeadUnknown != 2 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Une médaille se rattache à un pas à deux images au plus ; à trois, elle est orpheline ; sans élimination du joueur, aussi ;
// entre deux pas qui la réclament, elle est ambiguë et ne marque rien.
func TestEliminationsMedailles(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{
			joueurDeTest("a", [][2]int{{10, 1}}, nil),
			joueurDeTest("b", [][2]int{{10, 1}}, nil),
			joueurDeTest("c", [][2]int{{10, 1}, {12, 2}}, nil),
			joueurDeTest("d", [][2]int{{10, 1}}, nil),
		},
		Medals: []Medal{
			perfectMedal(12, "a"), // +2 : rattachée
			perfectMedal(13, "b"), // +3 : orpheline
			perfectMedal(11, "c"), // entre les pas 10 et 12 : ambiguë
			perfectMedal(5, "z"),  // joueur sans élimination : orpheline
			perfectMedal(8, "d"),  // -2 : rattachée
		},
	}
	got, cov := buildEliminations(in)
	if want := []string{"10:?:v:", "10:?:?:", "10:?:?:", "10:?:v:", "12:?:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.PerfectYes != 2 || cov.PerfectOrphans != 2 || cov.PerfectAmbiguous != 1 || cov.PerfectUnknown != 3 {
		t.Errorf("couverture = %+v", cov)
	}
	for _, e := range got {
		if e.Perfect != nil && !*e.Perfect {
			t.Errorf("`perfect` ne vaut jamais faux : %+v", e)
		}
	}
}

// Un pas de deux éliminations qui porte une seule médaille ne marque rien ; avec deux médailles, les deux sont parfaites.
func TestEliminationsMedaillesSurUnPasMultiple(t *testing.T) {
	p := joueurDeTest("k", [][2]int{{10, 2}}, nil)
	_, une := buildEliminations(eliminationInput{Players: []PlayerScore{p}, Medals: []Medal{perfectMedal(10, "k")}})
	if une.PerfectYes != 0 || une.PerfectAmbiguous != 1 {
		t.Errorf("une médaille pour deux éliminations : %+v", une)
	}
	_, deux := buildEliminations(eliminationInput{Players: []PlayerScore{p}, Medals: []Medal{perfectMedal(10, "k"), perfectMedal(11, "k")}})
	if deux.PerfectYes != 2 || deux.PerfectAmbiguous != 0 {
		t.Errorf("deux médailles pour deux éliminations : %+v", deux)
	}
}

// Les montées sans élimination sont comptées : à une image sans pas, ou au-delà des éliminations du pas.
func TestEliminationsMonteesOrphelines(t *testing.T) {
	in := eliminationInput{Players: []PlayerScore{
		joueurDeTest("a", [][2]int{{10, 1}}, [][2]int{{11, 1}}),
		joueurDeTest("b", [][2]int{{10, 1}}, [][2]int{{10, 3}}),
	}}
	got, cov := buildEliminations(in)
	if want := []string{"10:f:?:", "10:?:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadOrphanRises != 1+2 || cov.HeadAmbiguous != 1 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Quand la garde a écarté des émissions du compteur de tirs à la tête, le faux n'est plus prouvé ; le vrai l'est encore.
func TestEliminationsGardeDesCompteurs(t *testing.T) {
	in := eliminationInput{
		Players:          []PlayerScore{joueurDeTest("k", [][2]int{{10, 1}, {20, 2}}, [][2]int{{20, 1}})},
		HeadshotsGuarded: true,
	}
	got, cov := buildEliminations(in)
	if want := []string{"10:?:?:", "20:v:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadGuarded != 1 || cov.HeadYes != 1 || cov.HeadNo != 0 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Sans action de score, ou avec une action trop éloignée, la victime est absente ; une action ne sert qu'une fois.
func TestEliminationsVictimes(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{joueurDeTest("k", [][2]int{{10, 1}, {11, 2}, {50, 3}}, nil)},
		// Une seule action à l'image 11 : la deuxième élimination la prend (même image), la première ne l'a pas.
		Actions: []ScoreAction{killAction(11, "k", "va"), killAction(53, "k", "loin"), {T: 50, XUID: "k", Name: "assistPersonalScore", Target: "vx"}},
	}
	got, cov := buildEliminations(in)
	if want := []string{"10:?:?:", "11:?:?:va", "50:?:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.WithVictim != 1 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Films à plusieurs manches : la courbe cumulée continue d'une manche à l'autre, et les pas se lisent sur elle.
func TestEliminationsMancheMultiples(t *testing.T) {
	p := PlayerScore{
		XUID: "k",
		Kills: ScoreSeries{
			Rounds: []ScoreRound{
				{Round: 0, Points: []ScoreTick{{T: 10, V: 1}, {T: 20, V: 2}}},
				{Round: 1, Points: []ScoreTick{{T: 510, V: 1}}},
			},
			Total: []ScoreTick{{T: 10, V: 1}, {T: 20, V: 2}, {T: 510, V: 3}},
		},
		Stats: map[string]ScoreSeries{objectives.StatHeadshots: {
			Rounds: []ScoreRound{{Round: 1, Points: []ScoreTick{{T: 510, V: 1}}}},
			Total:  []ScoreTick{{T: 510, V: 1}},
		}},
	}
	got, _ := buildEliminations(eliminationInput{Players: []PlayerScore{p}, Medals: []Medal{perfectMedal(20, "k")}})
	if want := []string{"10:f:?:", "20:f:v:", "510:v:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
}

// Les lignes sont triées par image, tueur, victime ; les invariants de la couverture tiennent.
func TestEliminationsOrdreEtInvariants(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{
			joueurDeTest("b", [][2]int{{5, 1}, {30, 2}}, [][2]int{{30, 1}}),
			joueurDeTest("a", [][2]int{{5, 1}}, nil),
		},
		Medals: []Medal{perfectMedal(5, "a")},
	}
	got, cov := buildEliminations(in)
	if want := []string{"5:?:v:", "5:f:?:", "30:v:?:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) || got[0].XUID != "a" {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.Eliminations != cov.HeadYes+cov.HeadNo+cov.HeadUnknown || cov.Eliminations != cov.PerfectYes+cov.PerfectUnknown ||
		cov.HeadUnknown != cov.HeadNoCounter+cov.HeadAmbiguous+cov.HeadGuarded {
		t.Errorf("invariants de la couverture rompus : %+v", cov)
	}
}

// De l'enregistrement du tableau jusqu'à la marque : le compteur d'éliminations et celui des tirs à la tête sont écrits
// par le même enregistrement, l'image est celle de l'élimination, et la courbe publiée ne perd pas la montée.
func TestEliminationsDepuisLesEnregistrementsDuTableau(t *testing.T) {
	recs := []types.StatRecord{
		statRec(2_000, 10, 0, map[int]types.StatValue{2: {A: 1}}),
		statRec(3_000, 10, 0, map[int]types.StatValue{2: {A: 2}, 6: {A: 1}}),
		statRec(4_000, 10, 0, map[int]types.StatValue{2: {A: 3}}),
		statRec(5_000, 10, 0, map[int]types.StatValue{2: {A: 4}, 6: {A: 2}}),
	}
	in := &ScoreInput{Records: recs, Lines: []types.PlayerLine{{XUID: "x10", Kills: 4}}}
	tl, _ := buildScoreTimeline(in, nil, testClock(), nil)
	if tl == nil {
		t.Fatal("aucun tableau publié")
	}
	got, cov := buildEliminations(eliminationInput{Players: tl.Players, Medals: []Medal{perfectMedal(41, "x10")}})
	if want := []string{"10:f:?:", "20:v:?:", "30:f:?:", "40:v:v:"}; !reflect.DeepEqual(lignesDesEliminations(got), want) {
		t.Fatalf("éliminations = %v, attendu %v", lignesDesEliminations(got), want)
	}
	if cov.HeadYes != 2 || cov.HeadNo != 2 || cov.PerfectYes != 1 {
		t.Errorf("couverture = %+v", cov)
	}
}

// Le calque est câblé dans l'assemblage : avec un tableau des scores le document porte les éliminations et leur couverture,
// sans tableau il ne porte ni l'un ni l'autre.
func TestDocumentPorteLesEliminationsAvecLeTableau(t *testing.T) {
	recs := []types.StatRecord{
		statRec(1_000, 10, 0, map[int]types.StatValue{2: {A: 1}}),
		statRec(1_100, 10, 0, map[int]types.StatValue{2: {A: 2}, 6: {A: 1}}),
		statRec(1_200, 10, 0, map[int]types.StatValue{2: {A: 3}}),
	}
	in := &ScoreInput{Records: recs, Lines: []types.PlayerLine{{XUID: "x10", Kills: 3}}}
	pos := positionsPourOrigine()
	avec := BuildFromPositions(context.Background(), "m", "halo_infinite", pos, nil, Options{FilmClockOriginUS: 1_000_000, Score: in})
	if len(avec.Eliminations) != 3 || avec.Coverage == nil || avec.Coverage.Eliminations == nil {
		t.Fatalf("éliminations = %+v, couverture = %+v", avec.Eliminations, avec.Coverage)
	}
	if c := avec.Coverage.Eliminations; c.Eliminations != 3 || c.HeadYes != 1 || c.HeadNo != 2 {
		t.Errorf("couverture = %+v", c)
	}
	sans := BuildFromPositions(context.Background(), "m", "halo_infinite", pos, nil, Options{FilmClockOriginUS: 1_000_000})
	if sans.Eliminations != nil || (sans.Coverage != nil && sans.Coverage.Eliminations != nil) {
		t.Errorf("calque publié sans tableau des scores : %+v", sans.Eliminations)
	}
}
