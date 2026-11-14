package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/highlightevent"
)

func medalAward(xuid uint64, ms int64, hint, medal int) highlightevent.MedalAward {
	return highlightevent.MedalAward{XUID: xuid, TimeMS: ms, TypeHint: hint, MedalType: medal}
}

// Une medaille se pose a la frame `(ms - originMs) / pas`, avec son nom quand la table le connait.
func TestBuildMedals_Placement(t *testing.T) {
	origin := int64(3000)
	in := MedalInput{Read: true, Awards: []highlightevent.MedalAward{
		medalAward(2534026679052315, 3000+33*10+5, 100, 0), // Double Kill, frame 10
		medalAward(2534026679052315, 3000, 100, 0),         // frame 0 exactement
		medalAward(2533491259474651, 3000+33*10+20, 100, 85),
	}}
	got, cov := buildMedals(in, &origin, 33, 100)
	if len(got) != 3 || cov.Decoded != 3 || cov.Published != 3 || cov.UnknownName != 0 {
		t.Fatalf("medailles = %+v, couverture = %+v", got, cov)
	}
	if got[0].T != 0 || got[1].T != 10 || got[2].T != 10 {
		t.Errorf("frames = %d, %d, %d ; attendu 0, 10, 10", got[0].T, got[1].T, got[2].T)
	}
	// A frame egale, le xuid decimal le plus petit en tete.
	if got[1].XUID != "2533491259474651" || got[2].XUID != "2534026679052315" {
		t.Errorf("ordre a frame egale : %s puis %s", got[1].XUID, got[2].XUID)
	}
	if got[0].Name != "Double Kill" || got[0].ID != 0 || got[0].MS != 3000 {
		t.Errorf("premiere medaille = %+v", got[0])
	}
}

// Hors de l axe, ou sans origine, une medaille n est pas posee : elle est comptee, jamais clampee.
func TestBuildMedals_Refus(t *testing.T) {
	origin := int64(3000)
	in := MedalInput{Read: true, Awards: []highlightevent.MedalAward{
		medalAward(1, 2999, 100, 0),       // avant la frame 0
		medalAward(1, 3000+33*50, 100, 0), // frame 50 = frameCount : apres la fin
		medalAward(1, 3000+33*49, 100, 0), // derniere frame
		medalAward(1, 4000, 99, 99),       // couple inconnu : publiee sans nom
	}}
	got, cov := buildMedals(in, &origin, 33, 50)
	want := MedalCoverage{Decoded: 4, Published: 2, BeforeOrigin: 1, AfterEnd: 1, UnknownName: 1}
	if cov != want {
		t.Fatalf("couverture = %+v, attendu %+v", cov, want)
	}
	if len(got) != 2 || got[0].Name != "" || got[0].ID != 99 {
		t.Errorf("la medaille au couple inconnu garde son indice sans nom : %+v", got)
	}

	got, cov = buildMedals(in, nil, 33, 50)
	if len(got) != 0 || cov.NoOrigin != 4 || cov.Published != 0 {
		t.Errorf("sans origine rien ne se pose : %+v, %+v", got, cov)
	}
}
