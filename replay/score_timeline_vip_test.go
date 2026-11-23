package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// score_timeline_vip_test.go — LES COMPTEURS DU TABLEAU PROPRES AU VIP : temps en VIP, plus long temps, meilleure serie. Tout est
// synthetique.

// vipStatsInput : le joueur x10 est VIP deux fois. Le composant 23 porte le plus long temps (A) et le temps total (B) en dixiemes de
// seconde, le composant 22 la meilleure serie en VIP (B) ; la ligne se vide en fin de film.
func vipStatsInput() *ScoreInput {
	in := fragsIdentityInput()
	in.Records = append(in.Records,
		statRec(4_000, 10, 0, map[int]types.StatValue{22: {A: 1, B: 1}, 23: {A: 100, B: 100}}),
		statRec(5_000, 10, 0, map[int]types.StatValue{22: {A: 1, B: 2}, 23: {A: 200, B: 200}}),
		statRec(6_000, 10, 0, map[int]types.StatValue{22: {A: 2, B: 2}, 23: {A: 200, B: 250}}),
		statRec(7_000, 10, 0, map[int]types.StatValue{22: {A: 0, B: 0}, 23: {A: 0, B: 0}}),
	)
	return in
}

// Un match VIP publie le temps en VIP (avec son total), le plus long temps et la meilleure serie (courbes par manche seulement) ; la
// ligne videe en fin de film n'est pas une valeur.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : lire le composant 23 sans la famille ; publier un maximum avec un total ; inverser les cotes
// A et B du composant 23 ; compter le vidage de fin de film comme une valeur.
func TestScoreTimelinePublieLesCompteursDuVip(t *testing.T) {
	c := testClock()
	c.family = objectives.ObjectiveTypeVip
	tl, _ := buildScoreTimeline(vipStatsInput(), nil, c, nil)
	p := playerOf(t, tl, "x10")
	if got := lastValue(p.Stats[objectives.StatTimeAsVip]); got != 250 {
		t.Errorf("temps en VIP = %d, attendu 250 dixiemes (le vidage a zero n'est pas une valeur)", got)
	}
	for name, want := range map[string]int{objectives.StatLongestTimeAsVip: 200, objectives.StatMaxSpreeAsVip: 2} {
		s, ok := p.Stats[name]
		if !ok || len(s.Rounds) != 1 || len(s.Total) != 0 {
			t.Fatalf("%s : absent, mal decoupe ou porteur d'un total : %+v", name, s)
		}
		pts := s.Rounds[0].Points
		if got := pts[len(pts)-1].V; got != want {
			t.Errorf("%s = %d, attendu %d", name, got, want)
		}
	}
}

// Hors VIP, les composants 22 et 23 ne se lisent pas comme des compteurs du VIP : la meme ligne n'y publie rien de plus.
func TestScoreTimelineSansFamilleVipNePublieAucunCompteurDeMode(t *testing.T) {
	tl, _ := buildScoreTimeline(vipStatsInput(), nil, testClock(), nil)
	for _, name := range []string{objectives.StatTimeAsVip, objectives.StatLongestTimeAsVip, objectives.StatMaxSpreeAsVip} {
		if _, ok := playerOf(t, tl, "x10").Stats[name]; ok {
			t.Errorf("%s publie sans que l'hote reconnaisse un VIP", name)
		}
	}
}
