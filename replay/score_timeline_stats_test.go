package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// statsInput est fragsIdentityInput, plus des compteurs nommes pour le slot 10 (x10) : les tirs a la
// tete (composant 6, valeur A) et les reapparitions (composant 10, valeur B) a chaque vie, la serie
// maximale (composant 5, valeur B), et le vidage de la ligne en fin de film.
func statsInput() *ScoreInput {
	in := fragsIdentityInput()
	in.Records = append(in.Records,
		statRec(4_000, 10, 0, map[int]types.StatValue{6: {A: 1}, 10: {B: 1}, 5: {A: 10, B: 1}}),
		statRec(5_000, 10, 0, map[int]types.StatValue{6: {A: 2}, 10: {B: 2}, 5: {A: 10, B: 2}}),
		statRec(6_000, 10, 0, map[int]types.StatValue{6: {A: 3}, 10: {B: 3}}),
		// vidage : la ligne repasse a zero quand le joueur part ; une emission dont une valeur est
		// negative est mal alignee, une dont la valeur A du composant 6 recule l'est aussi.
		statRec(7_000, 10, 0, map[int]types.StatValue{6: {A: 0}, 10: {B: 0}, 5: {A: 0, B: 0}}),
		statRec(7_500, 10, 0, map[int]types.StatValue{5: {A: -124, B: 97}}),
		statRec(7_600, 10, 0, map[int]types.StatValue{6: {A: 1}, 10: {B: 40}}),
	)
	return in
}

func playerOf(t *testing.T, tl *ScoreTimeline, xuid string) PlayerScore {
	t.Helper()
	for _, p := range tl.Players {
		if p.XUID == xuid {
			return p
		}
	}
	t.Fatalf("le joueur %s n'est pas publie", xuid)
	return PlayerScore{}
}

// TestScoreTimelinePublieLesCompteursNommes : chaque compteur nomme est publie sous son nom, sa
// derniere valeur ignore le vidage de fin de film, et le maximum n'a pas de total.
func TestScoreTimelinePublieLesCompteursNommes(t *testing.T) {
	tl, _ := buildScoreTimeline(statsInput(), nil, testClock(), nil)
	p := playerOf(t, tl, "x10")
	if got := lastValue(p.Stats[objectives.StatHeadshots]); got != 3 {
		t.Errorf("tirs a la tete = %d, attendu 3 (le vidage a zero n'est pas une valeur)", got)
	}
	if got := lastValue(p.Stats[objectives.StatSpawns]); got != 3 {
		t.Errorf("reapparitions = %d, attendu 3 (l'emission dont les tirs a la tete reculent est ecartee)", got)
	}
	spree, ok := p.Stats[objectives.StatMaxSpree]
	if !ok || len(spree.Rounds) != 1 {
		t.Fatalf("serie maximale absente ou mal decoupee : %+v", spree)
	}
	if len(spree.Total) != 0 {
		t.Errorf("la serie maximale porte un total : %+v", spree.Total)
	}
	pts := spree.Rounds[0].Points
	if last := pts[len(pts)-1].V; last != 2 {
		t.Errorf("serie maximale = %d, attendu 2 (l'emission a valeur negative est ecartee)", last)
	}
}

// TestScoreTimelineSansCompteursNommes : un joueur dont le film ne remplit aucun compteur nomme
// n'a pas de `stats`, et sa ligne de base reste publiee.
func TestScoreTimelineSansCompteursNommes(t *testing.T) {
	tl, _ := buildScoreTimeline(statsInput(), nil, testClock(), nil)
	p := playerOf(t, tl, "x12")
	if p.Stats != nil {
		t.Errorf("stats publiees pour un joueur sans compteur nomme : %+v", p.Stats)
	}
	if lastValue(p.Kills) != 2 {
		t.Errorf("frags de x12 = %d, attendu 2", lastValue(p.Kills))
	}
}

// TestScoreTimelineFilmAPlusieursMancheSansCompteursNommes : sur un film a plusieurs manches, les
// quatre compteurs de base se publient comme avant, les compteurs nommes non, et la couverture compte
// les joueurs que cette regle en prive.
func TestScoreTimelineFilmAPlusieursMancheSansCompteursNommes(t *testing.T) {
	recs, deaths := reassignFixture()
	for _, slot := range []int{20, 22} {
		for round, t0 := range []int{1_000, 11_000} {
			for i := range 3 {
				recs = append(recs, statRec(t0+i*1_000, slot, round, map[int]types.StatValue{6: {A: int64(i + 1)}}))
			}
		}
	}
	tl, cov := buildScoreTimeline(&ScoreInput{Records: recs}, deaths, multiRoundClock(), nil)
	if tl == nil || len(tl.Players) != 2 {
		t.Fatalf("joueurs publies : %+v", tl)
	}
	for _, p := range tl.Players {
		if p.Stats != nil {
			t.Errorf("%s : compteurs nommes publies sur un film a plusieurs manches : %+v", p.XUID, p.Stats)
		}
		if lastValue(p.Kills) != 6 {
			t.Errorf("%s : frags %d, attendu 6 (les quatre compteurs de base ne changent pas)", p.XUID, lastValue(p.Kills))
		}
	}
	if cov.StatsWithheld != 2 {
		t.Errorf("statsWithheld = %d, attendu 2", cov.StatsWithheld)
	}
	if _, cov1 := buildScoreTimeline(statsInput(), nil, testClock(), nil); cov1.StatsWithheld != 0 {
		t.Errorf("statsWithheld = %d sur un film d'une manche, attendu 0", cov1.StatsWithheld)
	}
}

// TestEntreeDuDrapeauGardeLaBandeSure : le calque du drapeau ne recoit que les equipes et les huit
// premiers joueurs, quel que soit le dernier siege que le statborg porte.
func TestEntreeDuDrapeauGardeLaBandeSure(t *testing.T) {
	recs := []types.StatRecord{statRec(1, 8, 0, nil), statRec(2, 24, 0, nil), statRec(3, 26, 0, nil), statRec(4, 58, 0, nil)}
	in := EntreeDuDrapeau(recs, nil, &PontParManche{})
	if len(in.Records) != 2 || in.Records[1].Slot != 24 {
		t.Errorf("enregistrements du drapeau = %+v, attendu les slots 8 et 24", in.Records)
	}
}
