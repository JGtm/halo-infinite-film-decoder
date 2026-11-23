package replay

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

const (
	// tickUS : un tick serveur a 60 Hz, tel que le film l horodate.
	tickUS = 16_667
	// toleranceCapDeg : l ecart toléré entre `Point.H` (ou `Point.P`) et les degres que donnent les
	// quanta bruts de la pleine cadence. `Point.H` est arrondi au dixieme de degre : un point est donc
	// a au plus un demi-dixieme (0,05) de la valeur exacte du quantum, qui est la meme pour les deux
	// puisqu ils lisent le MEME record. Le cap 0 est publie 360 (cf. [headingForJSON]) : l ecart se
	// mesure donc modulo un tour.
	toleranceCapDeg = 0.05 + 1e-4
	// toleranceTempsMS : l ecart toléré entre l instant d un point (microsecondes du paquet) et
	// l echantillon de pleine cadence qui est le meme record. La pleine cadence tronque a la milliseconde
	// ([fullRateTimeUnitUS]) : un point est donc a moins d une milliseconde de son echantillon.
	toleranceTempsMS = 1
)

// positionsA60Hz : une vie qui court en X a 60 Hz pendant 2 s, se tait 6 s (lacune), puis repart
// 1 s. Un record sur trois ne porte pas de visee, comme dans le film (la visee n est repliquee que
// quand elle change). La gigue de quelques dizaines de microsecondes imite l horloge murale du paquet.
func positionsA60Hz() []grammar.BipedPosition {
	var out []grammar.BipedPosition
	poser := func(t uint64, i int) {
		p := grammar.BipedPosition{Slot: 100, TimestampUS: t, X: float32(i) * 0.037, Y: 1.5, Z: -0.25,
			HasWorld: true}
		if i%3 != 0 {
			p.HasYaw, p.YawRaw, p.PitchRaw = true, uint32(4090+i)%4096, uint32(1000+i%7)
		}
		out = append(out, p)
	}
	i := 0
	for t := uint64(1_000_000); t < 3_000_000; t += tickUS {
		poser(t+gigue(i), i)
		i++
	}
	for t := uint64(9_000_000); t < 10_000_000; t += tickUS {
		poser(t+gigue(i), i)
		i++
	}
	return out
}

func gigue(i int) uint64 { return uint64((i * 37) % 80) }

const (
	origineSynthetique = 1_000_000
	pasSynthetique     = 33_000
)

// tracesSynthetiques decime les positions synthetiques sur les vies donnees ; `interne` : comme le document interne du sync,
// qui ne retient pas la pleine cadence (la decimation des points est alors celle d avant l ajout).
func tracesSynthetiques(pos []grammar.BipedPosition, interne bool, vies []lifeSpan) ([]Track, TrackCoverage) {
	return decimateTracks(pos, decoupeDesTraces{
		origin: origineSynthetique, step: pasSynthetique, minPoints: 1, vies: vies,
		fb: fallback.NouveauCompteur(), sansPleineCadence: interne,
	})
}

func uneVie() []lifeSpan {
	return []lifeSpan{{slot: 100, from: 1_000_000, to: 10_000_000, xuid: 111}}
}

// pleinDebitDecode est une piste de pleine cadence reconstruite par somme cumulee.
type pleinDebitDecode struct {
	ms         []int64
	x, y, z    []int32
	aimMS      []int64
	h, p       []int32
	lacunesIdx []int
}

func decoderPleinDebit(t *testing.T, f *FullRateTrack) pleinDebitDecode {
	t.Helper()
	if len(f.X) != len(f.T) || len(f.Y) != len(f.T) || len(f.Z) != len(f.T) {
		t.Fatalf("colonnes de position de longueurs differentes : %d %d %d %d", len(f.T), len(f.X), len(f.Y), len(f.Z))
	}
	if len(f.H) != len(f.AimT) || len(f.P) != len(f.AimT) {
		t.Fatalf("colonnes de visee de longueurs differentes : %d %d %d", len(f.AimT), len(f.H), len(f.P))
	}
	var d pleinDebitDecode
	var ms int64
	var x, y, z int32
	for i := range f.T {
		ms, x, y, z = ms+f.T[i], x+f.X[i], y+f.Y[i], z+f.Z[i]
		d.ms, d.x, d.y, d.z = append(d.ms, ms), append(d.x, x), append(d.y, y), append(d.z, z)
	}
	var a int64
	var h, p int32
	for i := range f.AimT {
		a, h, p = a+f.AimT[i], h+f.H[i], p+f.P[i]
		d.aimMS, d.h, d.p = append(d.aimMS, a), append(d.h, h), append(d.p, p)
	}
	d.lacunesIdx = f.Gaps
	return d
}

// ecartCapDeg rend l ecart entre deux angles en degres, modulo un tour.
func ecartCapDeg(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	return math.Min(d, 360-d)
}

// verifierLaCoherenceAvecLesPoints : chaque point de la grille est le PREMIER echantillon de sa
// frame, et il retrouve cet echantillon a la tolerance nommee ci-dessus. `pasUS` est le pas de grille ;
// `origineUS`, l origine de la grille, ou -1 quand l appelant ne la connait pas (le temps n est alors pas confronte).
func verifierLaCoherenceAvecLesPoints(t *testing.T, tr Track, pasUS, origineUS int64) {
	t.Helper()
	d := decoderPleinDebit(t, tr.FullRate)
	frame := func(ms int64) int { return int(ms * fullRateTimeUnitUS / pasUS) }
	j := 0
	for k, pt := range tr.Points {
		for j < len(d.ms) && frame(d.ms[j]) < pt.T {
			j++ // les echantillons suivants de la frame precedente
		}
		if j >= len(d.ms) || frame(d.ms[j]) != pt.T {
			t.Fatalf("point %d (frame %d) sans echantillon de pleine cadence dans sa frame", k, pt.T)
		}
		if k > 0 && frame(d.ms[j-1]) == pt.T {
			t.Fatalf("point %d : l echantillon qui le precede est dans la meme frame %d", k, pt.T)
		}
		if int32(math.Round(float64(pt.X)*coordScale)) != d.x[j] || int32(math.Round(float64(pt.Y)*coordScale)) != d.y[j] ||
			int32(math.Round(float64(pt.Z)*coordScale)) != d.z[j] {
			t.Fatalf("point %d : (%v, %v, %v) contre l echantillon (%d, %d, %d) centiemes", k, pt.X, pt.Y, pt.Z, d.x[j], d.y[j], d.z[j])
		}
		if origineUS >= 0 && math.Abs(float64(d.ms[j])-float64(int64(pt.us)-origineUS)/1000) > toleranceTempsMS {
			t.Fatalf("point %d : instant %d us, echantillon a %d ms de l origine", k, pt.us, d.ms[j])
		}
		// DEUX RECORDS A LA MEME MILLISECONDE (le film en ecrit quelquefois) : la visee d un echantillon ne se
		// rattache plus a sa position par l instant, le test ne tranche pas.
		if (j+1 < len(d.ms) && d.ms[j+1] == d.ms[j]) || (j > 0 && d.ms[j-1] == d.ms[j]) {
			continue
		}
		verifierLaViseeDuPoint(t, k, pt, d, j)
	}
}

// verifierLaViseeDuPoint : le point porte un cap quand l echantillon de sa frame porte une visee A
// LA MEME MILLISECONDE, et les deux disent la meme chose a la tolerance nommee.
func verifierLaViseeDuPoint(t *testing.T, k int, pt Point, d pleinDebitDecode, j int) {
	t.Helper()
	i, trouve := slices.BinarySearch(d.aimMS, d.ms[j])
	if !trouve {
		if pt.H != 0 || pt.P != 0 {
			t.Fatalf("point %d porte une visee (%v, %v) que la pleine cadence ne porte pas a son instant", k, pt.H, pt.P)
		}
		return
	}
	var lu grammar.BipedPosition
	lu.HasYaw, lu.YawRaw, lu.PitchRaw = true, uint32(d.h[i]), uint32(d.p[i])
	cap, _ := lu.AimHeadingDeg()
	elev, _ := lu.AimPitchDeg()
	if pt.H == 0 {
		t.Fatalf("point %d sans cap alors que la pleine cadence porte une visee a son instant", k)
	}
	if ecartCapDeg(float64(pt.H), float64(cap)) > toleranceCapDeg || math.Abs(float64(pt.P)-float64(elev)) > toleranceCapDeg {
		t.Fatalf("point %d : visee (%v, %v) contre (%v, %v) lus dans les quanta", k, pt.H, pt.P, cap, elev)
	}
}

// TestDocumentInterneNeRetientPasLaPleineCadence : le document interne du sync n est pas publie : aucune piste ne
// retient de pleine cadence, aucun compteur n est rempli, et ses pistes ne portent pas la balise.
func TestDocumentInterneNeRetientPasLaPleineCadence(t *testing.T) {
	pos := positionsA60Hz()
	pos = append(pos, grammar.BipedPosition{Slot: 100, TimestampUS: 9_500_000, HasWorld: false}) // un quantum sans bornes de carte
	slices.SortStableFunc(pos, func(a, b grammar.BipedPosition) int { return int(int64(a.TimestampUS) - int64(b.TimestampUS)) })
	tracks, cov := tracesSynthetiques(pos, true, uneVie())
	if len(tracks) != 1 || tracks[0].FullRate != nil {
		t.Fatalf("%d piste(s), pleine cadence presente=%v ; attendu 1 piste qui n en porte pas", len(tracks),
			len(tracks) == 1 && tracks[0].FullRate != nil)
	}
	if cov.FullRateSamples != 0 || cov.FullRateAimSamples != 0 {
		t.Errorf("compteurs de la pleine cadence remplis sur le document interne : %+v", cov)
	}
	blob, err := json.Marshal(tracks)
	if err != nil {
		t.Fatal(err)
	}
	for _, nom := range []string{"fullRate", `"aimT"`} {
		if strings.Contains(string(blob), nom) {
			t.Errorf("la serialisation du document interne contient %q", nom)
		}
	}
}

// TestPleineCadenceNeChangePasLesPoints : les pistes, leurs points, leurs bornes et la
// couverture d avant sont identiques avec ou sans la pleine cadence (celle du document interne) ; seuls la pleine cadence et ses
// compteurs s ajoutent.
func TestPleineCadenceNeChangePasLesPoints(t *testing.T) {
	pos := positionsA60Hz()
	eteint, covEteinte := tracesSynthetiques(pos, true, uneVie())
	allume, covAllumee := tracesSynthetiques(pos, false, uneVie())
	if len(allume) != len(eteint) {
		t.Fatalf("%d pistes avec la pleine cadence, %d sans", len(allume), len(eteint))
	}
	for i := range allume {
		if allume[i].FullRate == nil {
			t.Fatalf("piste %d sans pleine cadence", i)
		}
		allume[i].FullRate = nil
	}
	if !reflect.DeepEqual(allume, eteint) {
		t.Errorf("les pistes different hors pleine cadence : points, bornes ou fin de vie ont bouge")
	}
	covAllumee.FullRateSamples, covAllumee.FullRateAimSamples = 0, 0
	if covAllumee != covEteinte {
		t.Errorf("la couverture d avant a bouge : %+v contre %+v", covAllumee, covEteinte)
	}
}

// TestPleineCadencePublieChaqueRecordASonInstant : la piste porte TOUS les records retenus, a leur
// milliseconde, quand `Points` n en garde qu un par frame. Le test rougit si la publication retombe
// sur la decimation par frame.
func TestPleineCadencePublieChaqueRecordASonInstant(t *testing.T) {
	pos := positionsA60Hz()
	tracks, cov := tracesSynthetiques(pos, false, uneVie())
	if len(tracks) != 1 || tracks[0].FullRate == nil {
		t.Fatalf("%d piste(s), pleine cadence presente=%v ; attendu 1 piste qui la porte", len(tracks),
			len(tracks) == 1 && tracks[0].FullRate != nil)
	}
	f := tracks[0].FullRate
	if len(f.T) != len(pos) || cov.FullRateSamples != len(pos) {
		t.Fatalf("%d echantillons (couverture %d), attendu %d : la publication est retombee sur la decimation (%d points)",
			len(f.T), cov.FullRateSamples, len(pos), len(tracks[0].Points))
	}
	if cov.FullRateSamples < 1+(pasSynthetique/tickUS)*cov.PublishedPoints/2 {
		t.Errorf("%d echantillons pour %d points : a 60 Hz sur une grille de 33 ms, attendu environ deux fois plus",
			cov.FullRateSamples, cov.PublishedPoints)
	}
	d := decoderPleinDebit(t, f)
	for i, p := range pos {
		if want := int64((p.TimestampUS - origineSynthetique) / fullRateTimeUnitUS); d.ms[i] != want {
			t.Fatalf("echantillon %d : t=%d ms, attendu %d", i, d.ms[i], want)
		}
		if d.x[i] != centiemes(p.X) || d.y[i] != centiemes(p.Y) || d.z[i] != centiemes(p.Z) {
			t.Fatalf("echantillon %d : (%d, %d, %d) centiemes, attendu (%d, %d, %d)", i, d.x[i], d.y[i], d.z[i],
				centiemes(p.X), centiemes(p.Y), centiemes(p.Z))
		}
	}
	verifierLesViseesDuFilm(t, f, pos)
	if len(f.Gaps) != 1 || f.Gaps[0] != 120 {
		t.Errorf("lacunes %v, attendu [120] : l indice du record qui ROUVRE la vie apres 6 s de silence", f.Gaps)
	}
	// LA MEME LACUNE QUE `Point.G` : le seul point qui la porte est celui de la frame du record qui rouvre la vie.
	wantG := int((pos[120].TimestampUS - pos[119].TimestampUS) / 1000)
	for _, pt := range tracks[0].Points {
		if pt.G != 0 && (pt.G != wantG || pt.T != int(d.ms[120]*fullRateTimeUnitUS/pasSynthetique)) {
			t.Errorf("lacune du point de la frame %d : %d ms, attendu %d ms a la frame %d", pt.T, pt.G, wantG, d.ms[120]*fullRateTimeUnitUS/pasSynthetique)
		}
	}
	if tracks[0].Points[0].G != 0 {
		t.Errorf("la premiere vie s ouvre sur une lacune")
	}
	verifierLaCoherenceAvecLesPoints(t, tracks[0], pasSynthetique, origineSynthetique)
}

// verifierLesViseesDuFilm : une visee par record qui la porte, a son instant, quantum exact, et les
// quanta rendent les degres de l accesseur du film.
func verifierLesViseesDuFilm(t *testing.T, f *FullRateTrack, pos []grammar.BipedPosition) {
	t.Helper()
	d := decoderPleinDebit(t, f)
	j := 0
	for _, p := range pos {
		if !p.HasYaw {
			continue
		}
		if j >= len(d.aimMS) {
			t.Fatalf("%d visees publiees, le film en porte davantage", len(d.aimMS))
		}
		want := int64((p.TimestampUS - origineSynthetique) / fullRateTimeUnitUS)
		if d.aimMS[j] != want || d.h[j] != int32(p.YawRaw) || d.p[j] != int32(p.PitchRaw) {
			t.Fatalf("visee %d : (t=%d, h=%d, p=%d), attendu (%d, %d, %d)", j, d.aimMS[j], d.h[j], d.p[j], want, p.YawRaw, p.PitchRaw)
		}
		j++
	}
	if j != len(d.aimMS) {
		t.Fatalf("%d visees publiees, %d dans le film", len(d.aimMS), j)
	}
}

// TestPleineCadenceSuitLaDecoupeDesVies : deux vies ecrites sur un meme slot donnent deux pistes,
// chacune avec SA pleine cadence, bornee a sa vie : aucun echantillon d une vie ne passe dans l autre,
// et la seconde vie ne s ouvre pas sur une lacune.
func TestPleineCadenceSuitLaDecoupeDesVies(t *testing.T) {
	pos := positionsA60Hz()
	vies := []lifeSpan{{slot: 100, from: 1_000_000, to: 5_000_000, xuid: 111},
		{slot: 100, from: 5_000_001, to: 10_000_000, xuid: 222}}
	tracks, cov := tracesSynthetiques(pos, false, vies)
	if len(tracks) != 2 {
		t.Fatalf("%d pistes, attendu 2", len(tracks))
	}
	a, b := tracks[0].FullRate, tracks[1].FullRate
	if len(a.T) != 120 || len(b.T) != len(pos)-120 {
		t.Errorf("pleine cadence %d + %d, attendu 120 + %d", len(a.T), len(b.T), len(pos)-120)
	}
	if b.T[0] != 8_000 || len(b.Gaps) != 0 {
		t.Errorf("la seconde vie s ouvre a t=%d ms avec les lacunes %v ; attendu 8000 et aucune (une vie neuve n est pas une lacune)",
			b.T[0], b.Gaps)
	}
	if cov.FullRateSamples != len(pos) {
		t.Errorf("couverture %d, attendu %d", cov.FullRateSamples, len(pos))
	}
	da, db := decoderPleinDebit(t, a), decoderPleinDebit(t, b)
	if da.ms[len(da.ms)-1] >= db.ms[0] {
		t.Errorf("les vies se chevauchent : la premiere finit a %d ms, la seconde commence a %d ms", da.ms[len(da.ms)-1], db.ms[0])
	}
	for i, tr := range tracks {
		d := decoderPleinDebit(t, tr.FullRate)
		premiere, derniere := int(d.ms[0]*1000/pasSynthetique), int(d.ms[len(d.ms)-1]*1000/pasSynthetique)
		if premiere != tr.StartFrame || derniere > tr.EndFrame {
			t.Errorf("piste %d : echantillons des frames %d a %d, vie de %d a %d", i, premiere, derniere, tr.StartFrame, tr.EndFrame)
		}
		verifierLaCoherenceAvecLesPoints(t, tr, pasSynthetique, origineSynthetique)
	}
}

// TestPleineCadenceCompteLesPositionsSansBornesDeCarte : une position sans bornes de carte n est
// pas une coordonnee, ni les points ni la pleine cadence ne la publient ; elle se compte.
func TestPleineCadenceCompteLesPositionsSansBornesDeCarte(t *testing.T) {
	pos := positionsA60Hz()
	pos = append(pos, grammar.BipedPosition{Slot: 100, TimestampUS: 9_990_000, HasWorld: false, Q: [3]uint32{1, 2, 3}})
	tracks, cov := tracesSynthetiques(pos, false, uneVie())
	if cov.NoWorld != 1 || cov.FullRateSamples != len(pos)-1 || len(tracks[0].FullRate.T) != len(pos)-1 {
		t.Errorf("sans bornes : %d, echantillons %d (colonne %d), attendu 1 et %d", cov.NoWorld, cov.FullRateSamples,
			len(tracks[0].FullRate.T), len(pos)-1)
	}
}

// TestPleineCadenceUneVieSansVisee : une vie dont aucun record ne porte de visee n a ni aimT, ni h, ni p.
func TestPleineCadenceUneVieSansVisee(t *testing.T) {
	pos := positionsA60Hz()
	for i := range pos {
		pos[i].HasYaw = false
	}
	tracks, cov := tracesSynthetiques(pos, false, uneVie())
	f := tracks[0].FullRate
	if f.AimT != nil || f.H != nil || f.P != nil || cov.FullRateAimSamples != 0 {
		t.Errorf("visee publiee sans visee dans le film : %d lignes, couverture %d", len(f.AimT), cov.FullRateAimSamples)
	}
}

// TestPleineCadenceDesBuildsFiges : sur les huit builds figes, la pleine cadence ne change RIEN au
// document hors pleine cadence et compteurs (meme JSON, octet pour octet, que celui du document interne qui ne la retient pas), la somme des colonnes est le
// compte de la couverture, ce compte n est jamais inferieur a celui des points, et chaque point retrouve
// son echantillon.
func TestPleineCadenceDesBuildsFiges(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			assembler := func(interne, avecViseesSeules bool) ReplayDocument {
				opt := g.options()
				opt.Bots = botsDuFixture(t, b.Short8)
				opt.Labels, opt.EquipmentEnergy = goldenCatalog(t), goldenEnergyCatalog(t)
				opt.MapQuant, opt.documentInterne = &entry, interne
				if !avecViseesSeules {
					opt.Vehicles.Aims = nil
				}
				return BuildFromPositions(context.Background(), b.Short8, "halo_infinite", g.Positions, g.Fire, opt)
			}
			eteint, allume, sansSeules := assembler(true, true), assembler(false, true), assembler(false, false)
			if eteint.Coverage.Tracks.FullRateSamples != 0 {
				t.Errorf("compteurs remplis sur le document interne : %+v", eteint.Coverage.Tracks)
			}
			c, somme, visees := allume.Coverage.Tracks, 0, 0
			for i := range allume.Tracks {
				tr := allume.Tracks[i]
				if tr.FullRate == nil {
					t.Fatalf("piste slot %d sans pleine cadence", tr.Slot)
				}
				somme, visees = somme+len(tr.FullRate.T), visees+len(tr.FullRate.AimT)
				allume.Tracks[i].FullRate = nil
			}
			// La coherence point par echantillon se juge sans les visees sans position : un point n en porte jamais, mais l echantillon
			// de sa frame peut en porter une (cf. tracks_visees_seules.go).
			for _, tr := range sansSeules.Tracks {
				verifierLaCoherenceAvecLesPoints(t, tr, int64(allume.FrameIntervalMS)*1000, -1)
			}
			verifierLeSortDesViseesSeules(t, &allume)
			if somme != c.FullRateSamples || visees != c.FullRateAimSamples {
				t.Errorf("colonnes %d / %d, couverture %d / %d", somme, visees, c.FullRateSamples, c.FullRateAimSamples)
			}
			// Jamais moins d echantillons que de points : chaque point est un echantillon. Le rapport depend du film : environ
			// deux a 60 Hz sur une grille de 33 ms, un seul quand le film replique deja a la cadence de la grille (builds anciens).
			if somme < c.PublishedPoints {
				t.Errorf("%d echantillons pour %d points : la publication perd des records", somme, c.PublishedPoints)
			}
			c.FullRateSamples, c.FullRateAimSamples, c.ViseesSeulesCoverage = 0, 0, ViseesSeulesCoverage{}
			verifierLesVolsDuDocument(t, &allume)
			a, e := mustJSON(t, allume), mustJSON(t, eteint)
			if a != e {
				t.Errorf("le document, prive de la pleine cadence et de ses compteurs, differe du document interne")
			}
		})
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// verifierLeSortDesViseesSeules : chaque visee sans position lue est posee, ecartee avant l origine de la grille ou
// ecartee faute de vie ; la somme des trois est les lectures du film, et les posees sont comprises dans les visees
// de la pleine cadence.
func verifierLeSortDesViseesSeules(t *testing.T, doc *ReplayDocument) {
	t.Helper()
	c := doc.Coverage.Tracks
	if doc.Coverage.Vehicles == nil {
		t.Fatal("coverage.vehicles absente")
	}
	if somme := c.FullRateAimOnlySamples + c.AimOnlyBeforeOrigin + c.AimOnlyNoLife; somme != doc.Coverage.Vehicles.AimReads {
		t.Errorf("visees sans position : %d posees + %d avant l origine + %d sans vie = %d, lectures du film %d",
			c.FullRateAimOnlySamples, c.AimOnlyBeforeOrigin, c.AimOnlyNoLife, somme, doc.Coverage.Vehicles.AimReads)
	}
	if c.FullRateAimOnlySamples > c.FullRateAimSamples {
		t.Errorf("%d visees sans position posees pour %d visees de pleine cadence", c.FullRateAimOnlySamples, c.FullRateAimSamples)
	}
}
