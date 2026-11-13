package replay

// seuils_ms_cadence_test.go — LES SEUILS TEMPORELS SONT DES DUREES, PAS DES NOMBRES D'IMAGES.
//
// Les tests « ...SurDesInstantsMultiplesDuPas » rejouent la MEME situation (les memes ecarts en
// millisecondes) a 100 ms et a 33 ms par image, avec des instants multiples EXACTS des deux pas : la
// decision y est identique. Un seuil ecrit en images changerait de sens avec la cadence (un ecart de
// 150 ms serait refuse a 33 ms par un seuil « 2 images »).
//
// CE QU'ILS NE DISENT PAS : hors de cette phase, la decision n'est identique qu'A UNE IMAGE PRES. Le
// document ne porte que des frames (`Shot.T`, `Projectile.T0`, `VehicleRide.T1`, `Tracked.T0`) : au
// point de decision, les deux instants en millisecondes ne sont plus disponibles, et deux instants
// separes de 100 ms tombent a 1 frame d'ecart a 100 ms mais jusqu'a 4 a 33 ms.
// TestFenetresEnMsEgalesAUneImagePres ecrit cette propriete.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// cadencesDeTest : le pas historique et une cadence triple.
var cadencesDeTest = []int{100, 33}

func TestFramesOfMS(t *testing.T) {
	for _, c := range []struct {
		nom          string
		ms, interval int
		attendu      int
	}{
		{"200 ms a 100 ms", 200, 100, 2},
		{"100 ms a 100 ms", 100, 100, 1},
		{"300 ms a 100 ms", 300, 100, 3},
		{"200 ms a 33 ms (arrondi vers le bas)", 200, 33, 6},
		{"100 ms a 33 ms", 100, 33, 3},
		{"100 ms a 40 ms (jamais plus large que la duree)", 100, 40, 2},
		{"duree nulle", 0, 100, 0},
		{"duree negative", -50, 100, 0},
		{"pas absent = pas historique (document sans frameIntervalMs)", 200, 0, 200 / HistoricalFrameIntervalMS},
		{"pas negatif = pas historique", 200, -1, 200 / HistoricalFrameIntervalMS},
		{"pas plus grand que la duree", 100, 250, 0},
	} {
		if got := framesOfMS(c.ms, c.interval); got != c.attendu {
			t.Errorf("%s : framesOfMS(%d, %d) = %d, attendu %d", c.nom, c.ms, c.interval, got, c.attendu)
		}
	}
}

// TestPasHistoriqueDistinctDuDefautDeConstruction : deux constantes, deux roles. Le pas historique
// est celui des documents publies sans `frameIntervalMs` ; le defaut est celui de la construction.
func TestPasHistoriqueDistinctDuDefautDeConstruction(t *testing.T) {
	if HistoricalFrameIntervalMS != 100 {
		t.Errorf("HistoricalFrameIntervalMS = %d, attendu 100 (documents publies sans frameIntervalMs)",
			HistoricalFrameIntervalMS)
	}
	if DefaultFrameIntervalMS != 33 {
		t.Errorf("DefaultFrameIntervalMS = %d, attendu 33 (30,3 images par seconde, schema 111)", DefaultFrameIntervalMS)
	}
}

// TestToleranceDePositionDeZoneSurDesInstantsMultiplesDuPas : un echantillon de position a 150 ms de
// l'action est retenu, un echantillon a 250 ms est refuse, a toute cadence.
func TestToleranceDePositionDeZoneSurDesInstantsMultiplesDuPas(t *testing.T) {
	zones := []Zone{zoneAt(0, 101, 0, 0, 0)}
	for _, interval := range cadencesDeTest {
		for _, c := range []struct {
			nom       string
			echantMS  int
			attribuee bool
		}{
			{"150 ms tolere", 150, true},
			{"250 ms refuse", 50, false},
		} {
			const actionMS = 300
			act := ObjectiveAction{T: actionMS / interval, XUID: "2533", Stat: "zone_secures", TimeMS: actionMS}
			tracks := []Track{track("2533", pointAt(c.echantMS/interval, 1, 0, 0))}
			res, cov := AttributeZones([]ObjectiveAction{act}, tracks, zones, nil,
				AttributeOptions{FrameIntervalMS: interval})
			checkInvariant(t, cov)
			if res[0].Attributed != c.attribuee {
				t.Errorf("pas %d ms, %s : attribuee = %v, attendu %v", interval, c.nom, res[0].Attributed, c.attribuee)
			}
		}
	}
}

// TestToleranceDePositionDeZoneParDefautVaut200ms : au pas de construction, le defaut vaut les 200 ms
// de [DefaultMaxGapMS] converties a ce pas, et `DefaultMaxGapFrames` (cmd de mesure) en reste
// l'expression a ce pas. Le nombre de frames se DERIVE du pas : aucun nombre n'est fige ici.
func TestToleranceDePositionDeZoneParDefautVaut200ms(t *testing.T) {
	if DefaultMaxGapMS != 200 {
		t.Errorf("DefaultMaxGapMS = %d, attendu 200", DefaultMaxGapMS)
	}
	attendu := DefaultMaxGapMS / DefaultFrameIntervalMS
	if DefaultMaxGapFrames != attendu {
		t.Errorf("DefaultMaxGapFrames = %d, attendu %d (200 ms au pas de %d ms)",
			DefaultMaxGapFrames, attendu, DefaultFrameIntervalMS)
	}
	if got := (AttributeOptions{FrameIntervalMS: DefaultFrameIntervalMS}).maxGapFrames(); got != attendu {
		t.Errorf("tolerance au pas de construction = %d frames, attendu %d", got, attendu)
	}
	// Un pas absent est celui d'un document sans frameIntervalMs : 100 ms, jamais le defaut de construction.
	if got, want := (AttributeOptions{}).maxGapFrames(), DefaultMaxGapMS/HistoricalFrameIntervalMS; got != want {
		t.Errorf("tolerance sans pas = %d frames, attendu %d (pas historique)", got, want)
	}
	if got := (AttributeOptions{FrameIntervalMS: 100}).maxGapFrames(); got != 2 {
		t.Errorf("tolerance par defaut a 100 ms = %d frames, attendu 2", got)
	}
	if got := (AttributeOptions{FrameIntervalMS: 33}).maxGapFrames(); got != 6 {
		t.Errorf("tolerance par defaut a 33 ms = %d frames, attendu 6", got)
	}
	if got := (AttributeOptions{MaxGapFrames: 4, FrameIntervalMS: 33}).maxGapFrames(); got != 4 {
		t.Errorf("surcharge explicite en frames = %d, attendu 4", got)
	}
}

// TestFenetreTirProjectileSurDesInstantsMultiplesDuPas : un tir a 100 ms de la naissance du projectile
// le lie, un tir a 200 ms non, a toute cadence.
func TestFenetreTirProjectileSurDesInstantsMultiplesDuPas(t *testing.T) {
	for _, interval := range cadencesDeTest {
		const naissanceMS = 1000
		for _, c := range []struct {
			nom  string
			tirM int
			lie  bool
		}{
			{"tir 100 ms avant", naissanceMS - 100, true},
			{"tir 100 ms apres", naissanceMS + 100, true},
			{"tir 200 ms avant", naissanceMS - 200, false},
			{"tir 200 ms apres", naissanceMS + 200, false},
		} {
			projs := []Projectile{projectileNeAuMS(naissanceMS, interval, 0, 0)}
			shots := []Shot{tirAuMS(c.tirM, interval, 1, "A", 0, 0)}
			linked, _ := linkShotsToProjectiles(projs, shots, nil, interval)
			if (linked == 1) != c.lie {
				t.Errorf("pas %d ms, %s : lie = %v, attendu %v", interval, c.nom, linked == 1, c.lie)
			}
		}
	}
}

// TestNaissanceCommuneDeTourelleSurDesInstantsMultiplesDuPas : un porteur ne 100 ms apres la piece est
// ne avec elle, un porteur ne 200 ms apres non, a toute cadence.
func TestNaissanceCommuneDeTourelleSurDesInstantsMultiplesDuPas(t *testing.T) {
	for _, interval := range cadencesDeTest {
		for _, c := range []struct {
			nom      string
			ecartMS  int
			ensemble bool
		}{
			{"ne 100 ms apres", 100, true},
			{"ne 200 ms apres", 200, false},
		} {
			porteur := vtChassis(101, familleWarthog)
			porteur.T0 = c.ecartMS / interval
			porteur.Spawn = nil
			tracks := []VehicleTrack{vtPiece(100, vtLAAG), porteur}
			anchors := vehicleBoardingAnchors{clock: replayClock{step: uint64(interval) * 1000}}
			tally := poseTurretsOnCarriers(tracks, anchors, fallback.NouveauCompteur())
			if got := tracks[0].Carrier != nil; got != c.ensemble {
				t.Errorf("pas %d ms, %s : porteur pose = %v, attendu %v", interval, c.nom, got, c.ensemble)
			}
			if !c.ensemble && tally.birthMismatch != 1 {
				t.Errorf("pas %d ms, %s : refus compte = %d, attendu 1", interval, c.nom, tally.birthMismatch)
			}
		}
	}
}

// TestToleranceDeNaissanceDesObjetsEngendresSurDesInstantsMultiplesDuPas : une trajectoire nee 300 ms
// apres la pile qui nomme l'objet le situe, une nee 400 ms apres non, a toute cadence.
func TestToleranceDeNaissanceDesObjetsEngendresSurDesInstantsMultiplesDuPas(t *testing.T) {
	if toleranceDeNaissanceMS != 300 {
		t.Errorf("toleranceDeNaissanceMS = %d, attendu 300", toleranceDeNaissanceMS)
	}
	for _, interval := range cadencesDeTest {
		const t0MS = 30_000
		for _, c := range []struct {
			nom     string
			ecartMS int
			situe   bool
		}{
			{"300 ms apres", 300, true},
			{"300 ms avant", -300, true},
			{"400 ms apres", 400, false},
			{"400 ms avant", -400, false},
		} {
			cands := []reposEngendre{cand((t0MS+c.ecartMS)/interval, 7)}
			if _, ok := reposNeAPeuPres(cands, t0MS/interval, interval); ok != c.situe {
				t.Errorf("pas %d ms, %s : situe = %v, attendu %v", interval, c.nom, ok, c.situe)
			}
		}
	}
}

// TestPasImpossibleDeProjectileEnVitesseIndependantDeLaCadence : un projectile a 60 m/s se lit, un
// projectile rapide du jeu (120 m/s) se lit aussi, un saut a 400 m/s est coupe, a toute cadence.
func TestPasImpossibleDeProjectileEnVitesseIndependantDeLaCadence(t *testing.T) {
	if projectileMaxStepM(100_000) != 20 {
		t.Errorf("a 100 ms le pas maximal vaut %v m, attendu 20", projectileMaxStepM(100_000))
	}
	if got, want := projectileMaxStepM(1_000), projectileMaxStepM(projectileMinElapsedUS); got != want {
		t.Errorf("a 1 ms le pas maximal vaut %v m, attendu celui de la duree plancher (%v m)", got, want)
	}
	for _, interval := range cadencesDeTest {
		pasUS := uint64(interval) * 1000
		for _, c := range []struct {
			nom     string
			vitesse float64 // m/s
			coupe   bool
		}{
			{"60 m/s", 60, false},
			{"120 m/s (projectile rapide du jeu)", 120, false},
			{"400 m/s", 400, true},
		} {
			pasM := float32(c.vitesse * float64(interval) / 1000)
			pts := make([]types.ProjectileSample, 6)
			for i := range pts {
				pts[i] = types.ProjectileSample{TimestampUS: 1_000_000 + uint64(i)*pasUS, X: float32(i) * pasM}
			}
			pts[5].AtRest = true
			tracks := []types.ProjectileTrack{{Slot: 4096, Gen: 1, Pts: pts}}
			out, _, tronquees := buildProjectiles(tracks, 1_000_000, pasUS, nil, nil)
			if (tronquees == 1) != c.coupe {
				t.Errorf("pas %d ms, %s : coupe = %v, attendu %v", interval, c.nom, tronquees == 1, c.coupe)
			}
			if !c.coupe && (len(out) != 1 || len(out[0].P) != 6) {
				t.Errorf("pas %d ms, %s : vol incomplet : %+v", interval, c.nom, out)
			}
		}
	}
}

// TestVolRapideSeLitAToutePhaseDeLaGrille : un vol a 120 m/s, echantillonne comme le film le fait (un
// record toutes les 16,7 ms, 2 m par record), n'est jamais coupe, quelle que soit la phase de la grille
// ni son pas ; un vol qui saute de 11 m en un record l'est. La borne porte sur la duree REELLE entre
// deux echantillons retenus : a 33 ms ils sont separes de 1 a 2 records, a 100 ms de 4 a 7.
func TestVolRapideSeLitAToutePhaseDeLaGrille(t *testing.T) {
	const recordUS = 16_667
	for _, interval := range cadencesDeTest {
		pasUS := uint64(interval) * 1000
		for phase := uint64(0); phase < pasUS; phase += 2_500 {
			origine := uint64(1_000_000)
			pts := make([]types.ProjectileSample, 14)
			for i := range pts {
				pts[i] = types.ProjectileSample{TimestampUS: origine + phase + uint64(i)*recordUS, X: float32(i) * 2}
			}
			pts[13].AtRest = true
			_, _, tronquees := buildProjectiles([]types.ProjectileTrack{{Slot: 4096, Gen: 1, Pts: pts}}, origine, pasUS, nil, nil)
			if tronquees != 0 {
				t.Errorf("pas %d ms, phase %d us : un vol a 120 m/s est coupe", interval, phase)
			}
			pts[7].X += 11
			_, _, tronquees = buildProjectiles([]types.ProjectileTrack{{Slot: 4096, Gen: 1, Pts: pts}}, origine, pasUS, nil, nil)
			// le saut n'est vu que s'il tombe sur un echantillon retenu (le premier de sa frame)
			if retenu := (pts[7].TimestampUS-origine)/pasUS != (pts[6].TimestampUS-origine)/pasUS; retenu && tronquees != 1 {
				t.Errorf("pas %d ms, phase %d us : le saut de 11 m n'est pas coupe", interval, phase)
			}
		}
	}
}

// TestBorneDeVolPorteSurLaDureeEntreEchantillonsRetenus : un vol a 120 m/s dont les echantillons lus sont
// espaces de 100 ms (12 m) sur une grille de 33 ms n'est pas coupe. Une borne appliquee au pas de la
// grille (6,6 m a 33 ms) le couperait : c'est la duree REELLE entre deux echantillons retenus qui compte.
func TestBorneDeVolPorteSurLaDureeEntreEchantillonsRetenus(t *testing.T) {
	const (
		pasUS      = 33_000
		ecartUS    = 100_000
		origine    = 1_000_000
		metresParS = 120.0
	)
	pts := make([]types.ProjectileSample, 5)
	for i := range pts {
		pts[i] = types.ProjectileSample{
			TimestampUS: origine + uint64(i)*ecartUS,
			X:           float32(float64(i) * metresParS * ecartUS / 1e6),
		}
	}
	pts[len(pts)-1].AtRest = true
	out, _, tronquees := buildProjectiles([]types.ProjectileTrack{{Slot: 4096, Gen: 1, Pts: pts}}, origine, pasUS, nil, nil)
	if tronquees != 0 {
		t.Fatalf("un vol a 120 m/s lu tous les 100 ms est coupe sur une grille de 33 ms (tronquees = %d)", tronquees)
	}
	if len(out) != 1 || len(out[0].P) != len(pts) {
		t.Errorf("vol incomplet : %+v", out)
	}
}

// TestArriveeDuGrappinBorneeParUnInstant : la fenetre de recherche de l'arrivee se ferme 2,5 s apres
// l'accroche REELLE, a toute cadence. Une trajectoire qui se rapproche de l'ancre sans jamais l'atteindre
// a son point le plus proche au bout de la fenetre : l'arrivee est la derniere frame dont le debut
// tient dans la fenetre, a une frame pres de l'accroche + 2,5 s.
func TestArriveeDuGrappinBorneeParUnInstant(t *testing.T) {
	for _, interval := range cadencesDeTest {
		pasUS := uint64(interval) * 1000
		for _, attachUS := range []uint64{650_000, 667_000, 683_000, 699_000} {
			pts := []Point{}
			for i := 0; uint64(i)*pasUS <= 5_000_000; i++ {
				v := float32(10 + float64(uint64(i)*pasUS)/1e6*5) // 5 m/s : loin de l'ancre (100), distance decroissante
				pts = append(pts, Point{T: i, X: v, Y: v})
			}
			track := Track{Slot: 5, StartFrame: 0, EndFrame: pts[len(pts)-1].T, Points: pts}
			lines, _ := buildGrappleLines([]types.GrappleRead{grappleRead(5, attachUS, true)}, grappleEntry(), 0, pasUS, []Track{track})
			if len(lines) != 1 {
				t.Fatalf("pas %d ms : %d traction(s), attendu 1", interval, len(lines))
			}
			finUS, t1US := attachUS+grapplePullCapUS, uint64(lines[0].T1)*pasUS
			if t1US > finUS || finUS-t1US >= pasUS {
				t.Errorf("pas %d ms, accroche a %d us : arrivee a %d us, attendu dans ]%d, %d] us (la fenetre se ferme a l'accroche + 2,5 s)",
					interval, attachUS, t1US, finUS-pasUS, finUS)
			}
		}
	}
}

// TestPoseNeeApresLaFinDeLAxeEstEcartee : une pose nee apres la DERNIERE POSITION du film est hors de l'axe, quel que
// soit le pas de la grille ; une pose nee a la derniere position, ou avant, est publiee. La fin de la grille tombe
// 0 a 99 ms (100 ms) ou 0 a 32 ms (33 ms) apres la derniere position : une pose nee dans cet intervalle ne doit pas
// se publier a un pas et pas a l'autre.
func TestPoseNeeApresLaFinDeLAxeEstEcartee(t *testing.T) {
	life := types.LifeKey{Slot: 900, Gen: 1}
	const origin, dernierePosition = uint64(1_000_000), uint64(10_950_000)
	for _, apresMS := range []uint64{1, 5, 17, 45, 120, 1000} {
		for _, interval := range cadencesDeTest {
			step := uint64(interval) * 1000
			clock := replayClock{origin: origin, step: step, frames: int((dernierePosition-origin)/step) + 1}
			raw := []types.EquipmentPlacement{
				pePose(life, 2_000_000),                     // en milieu de film
				pePose(life, dernierePosition),              // a la derniere position
				pePose(life, dernierePosition+apresMS*1000), // apres la derniere position
			}
			in := equipmentInputs{Raw: raw, Stats: grammar.EquipmentPlacementStats{Scanned: true}, LastUS: dernierePosition}
			out, _ := buildEquipmentPlacements(in, clock)
			if len(out) != 2 {
				t.Errorf("pas %d ms, %d ms apres la derniere position : %d pose(s) publiee(s), attendu 2",
					interval, apresMS, len(out))
				continue
			}
			if dernier := out[len(out)-1]; dernier.T0 != clock.frames-1 {
				t.Errorf("pas %d ms, %d ms apres : derniere pose a t0=%d, attendu la derniere frame %d",
					interval, apresMS, dernier.T0, clock.frames-1)
			}
		}
	}
}

// TestFenetresEnMsEgalesAUneImagePres : pour les fenetres de rattachement dont la decision se prend sur
// des frames (tir / naissance de projectile, naissance d'un objet engendre, marge a bord d'une
// destruction), la decision a 100 et a 33 ms est celle de l'ecart en millisecondes a UNE IMAGE PRES : sure
// pour un ecart d'au plus `limite - pas`, refusee pour un ecart de plus de `limite + pas`, et libre entre
// les deux, ou la phase de la grille decide. Les instants ne sont pas disponibles au point de decision
// (le document ne porte que des frames) ; la propriete exacte est donc celle-ci.
func TestFenetresEnMsEgalesAUneImagePres(t *testing.T) {
	type regle struct {
		nom      string
		limiteMS int
		decide   func(pas, aMS, bMS int) bool // a puis b >= a
	}
	regles := []regle{
		{"tir / naissance du projectile", projShotWindowMS, func(pas, a, b int) bool {
			n, _ := linkShotsToProjectiles([]Projectile{projectileNeAuMS(b, pas, 0, 0)}, []Shot{tirAuMS(a, pas, 1, "A", 0, 0)}, nil, pas)
			return n == 1
		}},
		{"naissance d'un objet engendre", toleranceDeNaissanceMS, func(pas, a, b int) bool {
			_, ok := reposNeAPeuPres([]reposEngendre{cand(b/pas, 7)}, a/pas, pas)
			return ok
		}},
		{"marge a bord d'une destruction", vehicleDestructionAboardTolMS, func(pas, a, b int) bool {
			ride := []VehicleRide{{T0: 0, T1: a / pas}}
			return len(ridesAboardAt(ride, b/pas, framesOfMS(vehicleDestructionAboardTolMS, pas))) == 1
		}},
	}
	for _, r := range regles {
		for _, pas := range cadencesDeTest {
			for _, phase := range []int{0, 13, 29, 32, 57, 90, 99} {
				for ecart := 0; ecart <= r.limiteMS+2*pas; ecart += 3 {
					a := 10_000 + phase
					got := r.decide(pas, a, a+ecart)
					switch {
					case ecart <= r.limiteMS-pas && !got:
						t.Errorf("%s, pas %d ms, phase %d, ecart %d ms : refuse, attendu accepte (ecart <= limite - pas)", r.nom, pas, phase, ecart)
					case ecart > r.limiteMS+pas && got:
						t.Errorf("%s, pas %d ms, phase %d, ecart %d ms : accepte, attendu refuse (ecart > limite + pas)", r.nom, pas, phase, ecart)
					}
				}
			}
		}
	}
	// Le cas hors phase qui limitait la regle quand elle se jugeait en frames : un tir a 32 ms, une naissance a
	// 132 ms (100 ms d'ecart, pile a la limite) tombaient a une frame d'ecart a 100 ms et a quatre a 33 ms. Les
	// instants exacts les lient a tout pas.
	if !regles[0].decide(100, 32, 132) || !regles[0].decide(33, 32, 132) {
		t.Errorf("tir a 32 ms, naissance a 132 ms : attendu lie a 100 ms ET a 33 ms (100 ms d'ecart exact)")
	}
}
