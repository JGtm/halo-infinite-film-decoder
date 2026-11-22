package replay

// t0_horloge_test.go — LE COUP D'ENVOI LU SUR L'HORLOGE DE MANCHE, SUR FIXTURES SYNTHETIQUES.
//
// Chaque cas isole une propriete de la regle (cf. t0_horloge.go). Les lectures sont ecrites en quanta de 16
// bits, tels que le canal les rend : `limiteQ` quanta de limite, `limiteQ - k` de temps restant, soit
// k quanta ecoules, un quantum valant 36000/65534 s (0,549 s). Zero seconde ecoulee s'ecrit `reste == limite`.

import (
	"context"
	"math"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

const (
	// t0hOrigine est l'horodatage du premier paquet de position (microsecondes), t0hOrigineMs l'instant de la
	// frame 0 sur l'horloge du fil.
	t0hOrigine   = 10_000_000
	t0hOrigineMs = 5_000
	// t0hLimiteQ : 1311 quanta, soit 719,9 s (une limite de douze minutes).
	t0hLimiteQ = 1311
	// t0hQuantumUS : la duree d'un quantum de la minuterie, en microsecondes.
	t0hQuantumUS = 36000.0 / 65534.0 * 1e6
)

// t0hLecture est une lecture de l'horloge a `ts` microsecondes du film.
func t0hLecture(ts uint64, limite, reste uint16, etat uint8) grammar.LectureHorloge {
	return grammar.LectureHorloge{TS: ts, Limite: limite, Reste: reste, Etat: etat}
}

// t0hAttendu rend la marge (millisecondes, arrondie vers le bas) d'une lecture prise a `ts` apres `quanta`
// quanta ecoules. Le jeu dequantifie le quantum zero a zero exactement et le quantum q > 0 a (q - 0,5) quanta : un
// ecoule lu depuis zero (minuterie qui monte) vaut donc q - 0,5 quanta.
func t0hAttendu(ts uint64, quanta float64) int64 {
	depart := float64(ts) - quanta*t0hQuantumUS - t0hOrigine
	return int64(math.Floor(depart / 1000))
}

// t0hVerifie confronte le resultat a la marge attendue, a une milliseconde pres (la minuterie est en flottant
// simple precision).
func t0hVerifie(t *testing.T, t0 *int64, cov *T0FilmCoverage, ts uint64, quanta float64, lu bool) {
	t.Helper()
	if t0 == nil || !cov.Detected || cov.Reason != "" {
		t.Fatalf("coup d'envoi absent alors qu'une lecture court : t0=%v verdict=%+v", t0, cov)
	}
	want := t0hAttendu(ts, quanta)
	if d := cov.MarginMs - want; d < -1 || d > 1 {
		t.Errorf("marge = %d ms, attendu %d ms (a 1 ms pres)", cov.MarginMs, want)
	}
	if *t0 != t0hOrigineMs+cov.MarginMs {
		t.Errorf("coup d'envoi = %d ms, attendu origine + marge = %d ms", *t0, t0hOrigineMs+cov.MarginMs)
	}
	if cov.Source != t0FilmSourceRoundClock || cov.StartRead != lu {
		t.Errorf("verdict = %+v, attendu source %q et depart lu %t", cov, t0FilmSourceRoundClock, lu)
	}
}

func TestCoupDEnvoiHorlogeArreteePuisQuiDescend(t *testing.T) {
	// Arretee a sa limite, puis la premiere lecture qui descend arrive 1 s apres la derniere lecture arretee,
	// deux quanta (1,1 s) entames : le depart tombe 98 ms AVANT la frame 0, apres la derniere lecture arretee.
	run := uint64(t0hOrigine + 1_000_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(t0hOrigine-1_000_000, t0hLimiteQ, t0hLimiteQ, 1),
		t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ, 1),
		t0hLecture(run, t0hLimiteQ, t0hLimiteQ-2, 6),
		t0hLecture(run+1_000_000, t0hLimiteQ, t0hLimiteQ-4, 6),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run, 2, true)
	if cov.MarginMs >= 0 {
		t.Errorf("marge = %d ms : le depart precede la frame 0, la marge est negative", cov.MarginMs)
	}
}

func TestCoupDEnvoiHorlogeDeLaPremiereLectureSansEcoule(t *testing.T) {
	// La premiere lecture qui descend vaut encore sa limite : le depart est l'instant de la lecture.
	run := uint64(t0hOrigine + 3_000_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ, 1),
		t0hLecture(run, t0hLimiteQ, t0hLimiteQ, 6),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run, 0, true)
	if *t0 != t0hOrigineMs+3_000 {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms (lecture a 3 s de la frame 0)", *t0, t0hOrigineMs+3_000)
	}
}

func TestCoupDEnvoiHorlogeDejaEnMarcheALaPremiereLecture(t *testing.T) {
	// Le film commence horloge en marche : aucune lecture arretee, le depart se deduit du temps ecoule
	// (38 quanta, 20,9 s) lu sur la premiere lecture.
	run := uint64(t0hOrigine + 500_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(run, t0hLimiteQ, t0hLimiteQ-38, 6),
		t0hLecture(run+1_000_000, t0hLimiteQ, t0hLimiteQ-40, 6),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run, 38, false)
	if cov.MarginMs > -20_000 {
		t.Errorf("marge = %d ms, attendu au-dela de -20 s (20,9 s ecoulees avant la lecture)", cov.MarginMs)
	}
}

func TestCoupDEnvoiHorlogeQuiMonte(t *testing.T) {
	// Sans limite de temps la minuterie monte : le depart est l'instant ou elle valait zero, soit le temps de la
	// minuterie (ici le quantum 38, 37,5 quanta) avant la lecture. Une lecture arretee (zero) la precede.
	run := uint64(t0hOrigine + 2_000_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(t0hOrigine, 0, 0, 1),
		t0hLecture(run, 0, 38, 19),
		t0hLecture(run+1_000_000, 0, 40, 19),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run, 37.5, true)
}

func TestCoupDEnvoiHorlogeSeulLaPremiereMancheDate(t *testing.T) {
	// Deux manches : arretee puis en marche, puis la seconde manche (la minuterie revient a sa limite, arretee,
	// puis repart). Seul le depart de la premiere date le coup d'envoi.
	run1 := uint64(t0hOrigine + 1_000_000)
	run2 := uint64(t0hOrigine + 400_000_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ, 1),
		t0hLecture(run1, t0hLimiteQ, t0hLimiteQ, 6),
		t0hLecture(run1+1_000_000, t0hLimiteQ, t0hLimiteQ-2, 6),
		t0hLecture(run2-5_000_000, t0hLimiteQ, t0hLimiteQ, 1),
		t0hLecture(run2, t0hLimiteQ, t0hLimiteQ, 6),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run1, 0, true)
	if *t0 != t0hOrigineMs+1_000 {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms (premiere manche)", *t0, t0hOrigineMs+1_000)
	}
}

func TestCoupDEnvoiHorlogeIgnoreLesLecturesQueLaMesureEcarte(t *testing.T) {
	// Une lecture impossible (temps au-dessus de la limite, etat qui descend) et une lecture d'un etat non mesure
	// ne datent rien ; la premiere lecture RETENUE qui court date, et sans lecture arretee retenue avant elle le
	// depart n'est pas lu.
	run := uint64(t0hOrigine + 2_000_000)
	reads := []grammar.LectureHorloge{
		t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ+7, 6),
		t0hLecture(t0hOrigine+500_000, t0hLimiteQ, t0hLimiteQ-3, 9),
		t0hLecture(run, t0hLimiteQ, t0hLimiteQ-4, 6),
	}
	t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
	t0hVerifie(t, t0, cov, run, 4, false)
}

func TestCoupDEnvoiHorlogeSansLectureQuiCourtNeDateRien(t *testing.T) {
	for nom, reads := range map[string][]grammar.LectureHorloge{
		"aucune lecture": nil,
		"seulement arretee": {
			t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ, 1),
			t0hLecture(t0hOrigine+1_000_000, t0hLimiteQ, t0hLimiteQ, 1),
		},
		"seulement des lectures ecartees": {
			t0hLecture(t0hOrigine, t0hLimiteQ, t0hLimiteQ+3, 6),
			t0hLecture(t0hOrigine+1_000_000, t0hLimiteQ, t0hLimiteQ-1, 12),
		},
	} {
		t.Run(nom, func(t *testing.T) {
			t0, cov := coupDEnvoiParHorloge(reads, t0hOrigine, t0hOrigineMs)
			if t0 != nil {
				t.Fatalf("coup d'envoi date (%d ms) sans aucune lecture qui court : aucun repli permis", *t0)
			}
			if cov.Detected || cov.Reason != t0FilmReasonNoRunningClock || cov.Source != t0FilmSourceRoundClock {
				t.Errorf("verdict = %+v, attendu non date, raison %q, source %q", cov, t0FilmReasonNoRunningClock,
					t0FilmSourceRoundClock)
			}
		})
	}
}

func t0hPtr(v int64) *int64 { return &v }

// t0hPositions rend des pistes de deux joueurs qui partent ensemble tres tard : le detecteur de premier
// mouvement daterait ce film, et le coup d'envoi ne doit pas en tenir compte.
func t0hPositions() []grammar.BipedPosition {
	var out []grammar.BipedPosition
	for _, slot := range []uint32{1, 2} {
		for i := range 6 {
			out = append(out, grammar.BipedPosition{Slot: slot, TimestampUS: 2_000_000 + 100_000*uint64(i),
				X: float32(i), Y: 0, Z: 1, HasWorld: true})
		}
	}
	return out
}

// TestPoserLeCoupDEnvoiDansLeDocument : la passe pose le champ, son verdict, et l'etat « non lu » selon ses
// gardes ; le mouvement des pistes n'y entre pour rien.
func TestPoserLeCoupDEnvoiDansLeDocument(t *testing.T) {
	horloges := func(trames int, reads ...grammar.LectureHorloge) grammar.HorlogesLues {
		return grammar.HorlogesLues{Horloges: reads, Stats: grammar.StatsDesHorloges{Trames: trames}}
	}
	cas := []struct {
		nom      string
		clockUS  uint64
		h        grammar.HorlogesLues
		veutT0   *int64
		veutCov  bool
		veutRais string
	}{
		{"lu, depart lu", 1_000_000, horloges(3,
			t0hLecture(2_000_000, t0hLimiteQ, t0hLimiteQ, 1),
			t0hLecture(3_000_000, t0hLimiteQ, t0hLimiteQ, 6)), t0hPtr(2_000), true, ""},
		{"aucune lecture qui court", 1_000_000, horloges(3,
			t0hLecture(2_000_000, t0hLimiteQ, t0hLimiteQ, 1)), nil, true, t0FilmReasonNoRunningClock},
		{"marche des trames sans lecture", 1_000_000, horloges(0), nil, false, ""},
		{"origine non etablie", 0, horloges(3,
			t0hLecture(3_000_000, t0hLimiteQ, t0hLimiteQ, 6)), nil, false, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			doc := BuildFromPositions(context.Background(), "m", "halo_infinite", t0hPositions(), nil,
				Options{FilmClockOriginUS: c.clockUS, Horloges: c.h})
			switch {
			case c.veutT0 == nil && doc.T0FilmMs != nil:
				t.Errorf("t0FilmMs = %d, attendu absent", *doc.T0FilmMs)
			case c.veutT0 != nil && (doc.T0FilmMs == nil || *doc.T0FilmMs != *c.veutT0):
				t.Errorf("t0FilmMs = %v, attendu %d", doc.T0FilmMs, *c.veutT0)
			}
			got := doc.Coverage.T0Film
			if (got != nil) != c.veutCov {
				t.Fatalf("coverage.t0Film = %+v, presence attendue %t", got, c.veutCov)
			}
			if got != nil && got.Reason != c.veutRais {
				t.Errorf("raison = %q, attendu %q", got.Reason, c.veutRais)
			}
			if got != nil && (got.Tracks != 0 || got.Moving != 0 || got.Burst != 0) {
				t.Errorf("verdict = %+v : les comptes du detecteur de mouvement restent a zero", got)
			}
		})
	}
}
