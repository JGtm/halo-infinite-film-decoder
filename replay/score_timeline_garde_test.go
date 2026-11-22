package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestScoreTimelinePubliePourChaqueCompteurLesEmissionsGardees : un joueur de 3 eliminations dont le film porte une
// emission a 40 tirs a la tete et une serie de 9 garde ses vraies valeurs, et la couverture dit ce que la garde a
// ecarte, par compteur.
func TestScoreTimelinePubliePourChaqueCompteurLesEmissionsGardees(t *testing.T) {
	in := statsInput()
	in.Records = append(in.Records,
		statRec(3_500, 10, 0, map[int]types.StatValue{6: {A: 40}}),
		statRec(5_500, 10, 0, map[int]types.StatValue{5: {A: 10, B: 9}}),
	)
	c := testClock()
	c.cons = &objectives.ReplisALaConsultation{}
	tl, cov := buildScoreTimeline(in, nil, c, nil)
	p := playerOf(t, tl, "x10")
	if got := lastValue(p.Stats[objectives.StatHeadshots]); got != 3 {
		t.Errorf("tirs a la tete = %d, attendu 3", got)
	}
	pts := p.Stats[objectives.StatMaxSpree].Rounds[0].Points
	if last := pts[len(pts)-1].V; last != 2 {
		t.Errorf("serie maximale = %d, attendu 2", last)
	}
	want := map[string]int{objectives.StatHeadshots: 1, objectives.StatMaxSpree: 1}
	if !reflect.DeepEqual(cov.StatsGuarded, want) {
		t.Errorf("emissions gardees = %v, attendu %v", cov.StatsGuarded, want)
	}
}

// TestScoreTimelineSansEmissionGardeeNePublieRien : le cas nominal ne porte aucun compte de garde.
func TestScoreTimelineSansEmissionGardeeNePublieRien(t *testing.T) {
	c := testClock()
	c.cons = &objectives.ReplisALaConsultation{}
	_, cov := buildScoreTimeline(statsInput(), nil, c, nil)
	if cov.StatsGuarded != nil {
		t.Errorf("emissions gardees = %v, attendu aucune", cov.StatsGuarded)
	}
}
