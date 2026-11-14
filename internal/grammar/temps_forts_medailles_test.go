package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/highlightevent"
	"github.com/JGtm/halo-infinite-film-decoder/medalname"
)

// Les medailles d un morceau reel sont toutes celles des evenements `medal`, dans un ordre total, et
// chacune porte un couple d octets que la table connait.
func TestMedaillesDesTempsForts_MorceauReel(t *testing.T) {
	evs, err := ParseHighlightEvents(realV41Chunk, 41)
	if err != nil {
		t.Fatal(err)
	}
	attendu := 0
	for _, e := range evs {
		if e.EventType == highlightevent.EventTypeMedal {
			attendu++
		}
	}
	got := medaillesDesTempsForts(evs)
	if attendu == 0 || len(got) != attendu {
		t.Fatalf("medailles = %d, evenements medal = %d (le morceau reel en porte)", len(got), attendu)
	}
	for i, m := range got {
		if _, ok := medalname.Lookup(m.TypeHint, m.MedalType); !ok {
			t.Errorf("medaille %d : couple (%d, %d) hors de la table", i, m.TypeHint, m.MedalType)
		}
		if i > 0 && got[i-1].TimeMS > m.TimeMS {
			t.Fatalf("medailles non triees par instant : %d apres %d", m.TimeMS, got[i-1].TimeMS)
		}
	}
}

// Deux medailles de la meme milliseconde se departagent par xuid puis par medaille : l ordre ne
// depend pas de celui du morceau.
func TestMedaillesDesTempsForts_OrdreTotal(t *testing.T) {
	ev := func(xuid uint64, ms, medal int) highlightevent.HighlightEvent {
		return highlightevent.HighlightEvent{XUID: xuid, EventType: highlightevent.EventTypeMedal, TypeHint: 100, TimeMS: ms, MedalType: medal}
	}
	mort := highlightevent.HighlightEvent{XUID: 1, EventType: highlightevent.EventTypeDeath, TimeMS: 5}
	a := medaillesDesTempsForts([]highlightevent.HighlightEvent{ev(9, 10, 3), mort, ev(2, 10, 7), ev(2, 10, 1), ev(5, 4, 0)})
	b := medaillesDesTempsForts([]highlightevent.HighlightEvent{ev(2, 10, 1), ev(5, 4, 0), ev(2, 10, 7), ev(9, 10, 3)})
	if len(a) != 4 || len(b) != 4 {
		t.Fatalf("4 medailles attendues (la mort est ecartee) : %d et %d", len(a), len(b))
	}
	want := [][3]int{{4, 5, 0}, {10, 2, 1}, {10, 2, 7}, {10, 9, 3}}
	for i, w := range want {
		for _, m := range []highlightevent.MedalAward{a[i], b[i]} {
			if got := [3]int{int(m.TimeMS), int(m.XUID), m.MedalType}; got != w {
				t.Errorf("rang %d : %v, attendu %v", i, got, w)
			}
		}
	}
}
