package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// deuxViesDuSlot : le slot 100 a deux vies jointives, les bornes etant incluses ; le slot 200 n en a aucune.
func deuxViesDuSlot() []lifeSpan {
	return []lifeSpan{
		{slot: 100, from: 1_000_000, to: 5_000_000, xuid: 111},
		{slot: 100, from: 5_000_001, to: 10_000_000, xuid: 222},
	}
}

func viseeSeule(slot uint32, tUS uint64, yaw, pitch uint32) grammar.BipedAim {
	return grammar.BipedAim{Slot: slot, TimestampUS: tUS, YawRaw: yaw, PitchRaw: pitch}
}

// tracesAvecViseesSeules decime les positions synthetiques avec des visees sans position, au seuil donne.
func tracesAvecViseesSeules(pos []grammar.BipedPosition, seules []grammar.BipedAim, interne bool, vies []lifeSpan,
	minPoints int) ([]Track, TrackCoverage) {
	return decimateTracks(pos, decoupeDesTraces{
		origin: origineSynthetique, step: pasSynthetique, minPoints: minPoints, vies: vies,
		fb: fallback.NouveauCompteur(), sansPleineCadence: interne, viseesSeules: seules,
	})
}

// TestViseesSeulesSePosentSurLaVieQuiCouvreLeurInstant : chaque visee va a la vie de son slot dont les bornes
// (incluses) couvrent son instant ; une visee avant l origine de la grille, apres la derniere vie ou d un slot
// sans vie n est posee nulle part, et la premiere se compte a part.
func TestViseesSeulesSePosentSurLaVieQuiCouvreLeurInstant(t *testing.T) {
	visees := []grammar.BipedAim{
		viseeSeule(100, 2_000_000, 1, 1),  // vie 0
		viseeSeule(100, 5_000_000, 2, 2),  // borne haute de la vie 0, incluse
		viseeSeule(100, 5_000_001, 3, 3),  // borne basse de la vie 1, incluse
		viseeSeule(100, 10_000_001, 4, 4), // apres la derniere vie
		viseeSeule(100, 999_999, 5, 5),    // avant l origine de la grille
		viseeSeule(200, 2_000_000, 6, 6),  // slot sans vie
	}
	var cov ViseesSeulesCoverage
	bornes := map[uint32][]lifeSpan{100: deuxViesDuSlot()}
	par := repartirLesViseesSeules(visees, bornes, 1_000_000, &cov)
	if len(par) != 2 || len(par[vieIndexee{100, 0}]) != 2 || len(par[vieIndexee{100, 1}]) != 1 {
		t.Fatalf("repartition %v, attendu deux visees sur la vie 0 et une sur la vie 1", par)
	}
	if par[vieIndexee{100, 0}][1].YawRaw != 2 || par[vieIndexee{100, 1}][0].YawRaw != 3 {
		t.Errorf("les visees ne sont pas dans leur ordre d entree : %v", par)
	}
	if cov.AimOnlyBeforeOrigin != 1 {
		t.Errorf("avant l origine : %d, attendu 1", cov.AimOnlyBeforeOrigin)
	}
}

// TestPleineCadenceFusionneLesViseesSeules : les visees sans position entrent dans les colonnes de visee, dans
// l ordre du temps, la visee du record de position d abord a la meme milliseconde ; les positions, les lacunes
// et les points ne bougent pas, et la couverture rend chaque visee lue.
func TestPleineCadenceFusionneLesViseesSeules(t *testing.T) {
	pos := positionsA60Hz()
	seules := []grammar.BipedAim{
		viseeSeule(100, 4_000_000, 7, 8),   // dans le silence de replication : la vie le couvre
		viseeSeule(100, 1_016_950, 11, 12), // meme milliseconde que la visee du record de position (t=16 ms)
		viseeSeule(100, 1_500_000, 21, 22),
		viseeSeule(100, 10_500_000, 31, 32), // apres la vie
		viseeSeule(100, 900_000, 41, 42),    // avant l origine
		viseeSeule(300, 1_500_000, 51, 52),  // slot sans vie
	}
	sans, covSans := tracesSynthetiques(pos, false, uneVie())
	avec, cov := tracesAvecViseesSeules(pos, seules, false, uneVie(), 1)
	if len(avec) != 1 {
		t.Fatalf("%d pistes, attendu 1", len(avec))
	}
	a, b := decoderPleinDebit(t, sans[0].FullRate), decoderPleinDebit(t, avec[0].FullRate)
	if len(b.aimMS) != len(a.aimMS)+3 {
		t.Fatalf("%d visees, attendu %d (les %d des records de position + 3 posees)", len(b.aimMS), len(a.aimMS)+3, len(a.aimMS))
	}
	if !slices.IsSorted(b.aimMS) {
		t.Errorf("les colonnes de visee ne sont pas dans l ordre du temps : %v", b.aimMS[:20])
	}
	i := slices.Index(b.aimMS, 16)
	if i < 0 || b.aimMS[i+1] != 16 || b.h[i] != 4091 || b.h[i+1] != 11 || b.p[i+1] != 12 {
		t.Errorf("a t=16 ms : caps %v ; attendu le quantum du record de position (4091) puis la visee sans position (11)",
			b.h[i:i+2])
	}
	if j := slices.Index(b.aimMS, 3000); j < 0 || b.h[j] != 7 || b.p[j] != 8 {
		t.Errorf("la visee du silence de replication (t=3000 ms) manque : %v", b.aimMS)
	}
	if !slices.Equal(a.ms, b.ms) || !slices.Equal(a.x, b.x) || !slices.Equal(a.y, b.y) || !slices.Equal(a.z, b.z) ||
		!slices.Equal(a.lacunesIdx, b.lacunesIdx) {
		t.Error("les positions et les lacunes de la pleine cadence ont change")
	}
	if !slices.Equal(sans[0].Points, avec[0].Points) {
		t.Error("les points de la grille ont change")
	}
	if cov.FullRateAimOnlySamples != 3 || cov.AimOnlyBeforeOrigin != 1 || cov.AimOnlyNoLife != 2 {
		t.Errorf("couverture %+v, attendu 3 posees, 1 avant l origine, 2 sans vie", cov.ViseesSeulesCoverage)
	}
	if got := cov.FullRateAimSamples; got != covSans.FullRateAimSamples+3 {
		t.Errorf("fullRateAimSamples %d, attendu %d (celles des records de position, plus les 3 posees)", got, covSans.FullRateAimSamples+3)
	}
}

// TestViseesSeulesSurUneVieRefuseeSontSansVie : une visee dont la vie n est pas publiee (refusee par le seuil)
// se compte parmi celles sans vie, jamais posee sur la vie voisine.
func TestViseesSeulesSurUneVieRefuseeSontSansVie(t *testing.T) {
	pos := positionsA60Hz()
	pos = pos[:120:120] // la premiere vie : tout ce qui precede 5 s
	pos = append(pos, grammar.BipedPosition{Slot: 100, TimestampUS: 6_000_000, HasWorld: true})
	seules := []grammar.BipedAim{viseeSeule(100, 2_000_000, 1, 1), viseeSeule(100, 6_000_000, 2, 2)}
	tracks, cov := tracesAvecViseesSeules(pos, seules, false, deuxViesDuSlot(), 2)
	if len(tracks) != 1 || cov.RefusedMinPoints != 1 {
		t.Fatalf("%d pistes publiees, %d vies refusees ; attendu 1 et 1", len(tracks), cov.RefusedMinPoints)
	}
	if cov.FullRateAimOnlySamples != 1 || cov.AimOnlyNoLife != 1 || cov.AimOnlyBeforeOrigin != 0 {
		t.Errorf("couverture %+v, attendu 1 posee et 1 sans vie", cov.ViseesSeulesCoverage)
	}
}

// TestDocumentInterneNeRetientPasLesViseesSeules : le document interne du sync ne retient pas la pleine cadence,
// donc ne pose ni ne compte aucune visee sans position, et ses points ne changent pas.
func TestDocumentInterneNeRetientPasLesViseesSeules(t *testing.T) {
	pos := positionsA60Hz()
	seules := []grammar.BipedAim{viseeSeule(100, 1_500_000, 21, 22), viseeSeule(100, 900_000, 41, 42)}
	sans, _ := tracesSynthetiques(pos, true, uneVie())
	avec, cov := tracesAvecViseesSeules(pos, seules, true, uneVie(), 1)
	if avec[0].FullRate != nil || cov.ViseesSeulesCoverage != (ViseesSeulesCoverage{}) {
		t.Errorf("pleine cadence %v, couverture %+v ; attendu aucune des deux", avec[0].FullRate, cov.ViseesSeulesCoverage)
	}
	if !slices.Equal(sans[0].Points, avec[0].Points) {
		t.Error("les points du document interne ont change")
	}
}

// TestViseesSeulesSurUneVieSansVisee : une vie dont aucun record de position ne porte de visee recoit les
// visees sans position comme premieres colonnes de visee.
func TestViseesSeulesSurUneVieSansVisee(t *testing.T) {
	pos := positionsA60Hz()
	for i := range pos {
		pos[i].HasYaw = false
	}
	seules := []grammar.BipedAim{viseeSeule(100, 2_000_000, 100, 200), viseeSeule(100, 1_500_000, 90, 190)}
	tracks, cov := tracesAvecViseesSeules(pos, seules, false, uneVie(), 1)
	d := decoderPleinDebit(t, tracks[0].FullRate)
	if !slices.Equal(d.aimMS, []int64{500, 1000}) || !slices.Equal(d.h, []int32{90, 100}) || !slices.Equal(d.p, []int32{190, 200}) {
		t.Errorf("colonnes %v %v %v, attendu les deux visees triees", d.aimMS, d.h, d.p)
	}
	if cov.FullRateAimSamples != 2 || cov.FullRateAimOnlySamples != 2 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestLesViseesSansPositionAccompagnentLeCalqueQuelleQueSoitSonIssue : bande vide, creations ou nuage illisibles,
// lecture aboutie : les visees lues sortent dans les quatre cas, sans declarer vehicules un film qui n en a pas ;
// une lecture de visees en echec ne rend rien et n arrete pas le calque.
func TestLesViseesSansPositionAccompagnentLeCalqueQuelleQueSoitSonIssue(t *testing.T) {
	aims := []grammar.BipedAim{viseeSeule(100, 2_000_000, 1, 1)}
	kf := grammar.WorldObjectKeyframes{TimesUS: []uint64{2_000_000}}
	cas := []struct {
		nom    string
		lu     lectureVehicules
		balaye bool
	}{
		{"bande vide", lectureVehicules{issue: lectureBandeVide, kf: kf}, true},
		{"creations illisibles", lectureVehicules{issue: lectureCreationsIllisibles}, false},
		{"nuage illisible", lectureVehicules{issue: lectureNuageIllisible}, false},
		{"lecture aboutie", lectureVehicules{issue: lectureComplete, kf: kf}, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			c.lu.visees = aims
			out := journaliserVehicules(t.Context(), nil, "film", c.lu)
			if out.Scanned != c.balaye || len(out.Aims) != 1 {
				t.Errorf("balaye=%v, %d visees ; attendu %v et 1", out.Scanned, len(out.Aims), c.balaye)
			}
		})
	}
	echec := lectureVehicules{issue: lectureBandeVide, kf: kf, errVisees: grammar.ErrNoFilmChunk, visees: aims}
	if out := journaliserVehicules(t.Context(), nil, "film", echec); out.Aims != nil || !out.Scanned {
		t.Errorf("visees illisibles : %d visees, balaye=%v ; attendu aucune et un calque balaye", len(out.Aims), out.Scanned)
	}
}
