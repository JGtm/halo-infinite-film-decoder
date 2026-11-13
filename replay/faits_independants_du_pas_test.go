package replay

// faits_independants_du_pas_test.go — UN FAIT DU MATCH NE DEPEND PAS DU PAS DE PUBLICATION.
//
// Chaque test construit le MEME scenario (memes instants en microsecondes ou en millisecondes) sur
// une grille de 100, 66, 50, 33, 20 et 10 ms, et exige le meme fait. Aucun film, aucune identite reelle :
// des instants et des positions litteraux. Une fenetre, une tolerance ou un ordre qui se decide en
// nombre d'images de la grille donne a ces scenarios un verdict different d'un pas a l'autre — c'est
// ce que chaque test attrape, et la mutation qui le fait rougir est decrite en tete de chacun.

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// pasDeTest sont les pas de grille que les tests de ce fichier rejouent.
var pasDeTest = []int{100, 66, 50, 33, 20, 10}

// TestPointDuPorteurIndependantDuPas : le point publie qui localise un joueur est le DERNIER a l'instant ou
// avant, a au plus `flagCarrierPointToleranceMS`, a tout pas. Mutation : `bd <= 1` (une image) : a 33 ms un
// point a 150 ms n'est plus retrouve ; « le plus proche » : un point posterieur plus proche l'emporte sur le
// dernier avant.
func TestPointDuPorteurIndependantDuPas(t *testing.T) {
	for _, pas := range pasDeTest {
		pointAu := func(ms int, x float32) Point { return Point{T: ms / pas, X: x} }
		tracks := []Track{{Points: []Point{pointAu(1000, 1), pointAu(1300, 2)}}}
		tol := framesOfMS(flagCarrierPointToleranceMS, pas)
		for _, c := range []struct {
			nom string
			ms  int
			x   float32
			ok  bool
		}{
			{"a l'instant du point", 1000, 1, true},
			{"180 ms apres un corps immobile", 1180, 1, true}, // le plus proche serait 1300 : le dernier AVANT l'emporte
			{"le point suivant, plus tard", 1300, 2, true},
			{"avant tout point, a moins de 200 ms", 900, 1, true}, // a defaut de point avant, le suivant
			{"loin de tout point", 1700, 0, false},
			{"avant tout point, loin", 600, 0, false},
		} {
			p, ok := pointOfXUIDAt(tracks, c.ms/pas, tol)
			if ok != c.ok || (ok && p.X != c.x) {
				t.Errorf("pas %d ms, %s : (x=%v, ok=%v), attendu (x=%v, ok=%v)", pas, c.nom, p.X, ok, c.x, c.ok)
			}
		}
	}
}

// TestLacherVolontaireDateEnInstantsExacts : l'instant d'une vie libre se compare en millisecondes EXACTES
// au portage qu'elle ferme, jamais arrondi au debut de la frame. Mutation : `matchMSOfFrame(frameOf(..))` :
// une vie nee 10 ms apres la fin du portage est refusee a 33 ms (l'instant arrondi tombe apres) et acceptee a
// 100 ms (l'instant arrondi tombe avant).
func TestLacherVolontaireDateEnInstantsExacts(t *testing.T) {
	const originUS, finMS = uint64(1_000_000), 4000
	for _, pas := range pasDeTest {
		for _, c := range []struct {
			nom     string
			decalMS int
			ferme   bool
		}{
			{"10 ms avant la fin", -10, true},
			{"3 ms avant la fin", -3, true},
			{"10 ms apres la fin", 10, false},
			{"exactement a la fin", 0, false},
		} {
			ctx := flagCarryCtx{matchClock: matchClock{origin: originUS, step: uint64(pas) * 1000, frames: 1000}}
			naissanceUS := uint64(int64(finMS+c.decalMS) * 1000)
			scan := FlagCarryScan{Free: []flagFreeLife{{T0US: naissanceUS, T1US: naissanceUS,
				Pts: []flagFreeSample{{TUS: naissanceUS, X: 5, Y: 5}}}}}
			r := flagCarryRaw{xuid: "1", t0: 0, t1: finMS}
			f := int((naissanceUS - originUS) / (uint64(pas) * 1000))
			tracks := []Track{{Points: []Point{{T: f, X: 5, Y: 5}}}}
			_, ok := flagFreeDropInside(r, ctx, tracks, scan)
			if ok != c.ferme {
				t.Errorf("pas %d ms, vie libre %s : ferme = %v, attendu %v", pas, c.nom, ok, c.ferme)
			}
		}
	}
}

// TestOrigineDUnRamassageIndependanteDuPas : l'ordre entre un ramassage et une pose se decide en
// microsecondes, et l'echange (la pose du ramasseur lui-meme, d'une autre famille) n'est pas une origine.
// Mutation : `d.t > frame` : une pose nee 10 ms (a 33 ms) ou 40 ms (a 100 ms) apres le ramassage tombe dans
// sa frame et sort `ground`.
func TestOrigineDUnRamassageIndependanteDuPas(t *testing.T) {
	const ramasseur, autre = uint32(1), uint32(2)
	const ramassageUS = uint64(10_000_000)
	for _, pas := range pasDeTest {
		for _, c := range []struct {
			nom     string
			poseUS  uint64
			poseur  int
			famille string
			attendu string
		}{
			{"pose posee 10 ms apres le ramassage", ramassageUS + 10_000, int(autre), "wall", ""},
			{"pose posee 40 ms apres le ramassage", ramassageUS + 40_000, int(autre), "wall", ""},
			{"pose posee 2 s avant par un autre", ramassageUS - 2_000_000, int(autre), "wall", PickupOriginGround},
			{"pose a l'instant meme du ramassage", ramassageUS, int(autre), "wall", PickupOriginGround},
			{"echange : pose du ramasseur, autre famille", ramassageUS - 50_000, int(ramasseur), "wall", ""},
			{"pose du ramasseur, meme famille", ramassageUS - 50_000, int(ramasseur), "grenade_frag", PickupOriginGround},
			{"pose du ramasseur, famille non etablie", ramassageUS - 50_000, int(ramasseur), "", PickupOriginGround},
			{"pose d'un autre, autre famille", ramassageUS - 50_000, int(autre), "wall", PickupOriginGround},
		} {
			dropped := []droppedSpot{{t0US: c.poseUS, jusquaUS: -1, x: 10, y: 10, poseur: c.poseur, famille: c.famille}}
			pos := []grammar.BipedPosition{poPos(ramasseur, ramassageUS, 10, 10, 0)}
			clock := replayClock{origin: 0, step: uint64(pas) * 1000, frames: 100_000,
				families: map[uint32]string{0xbcabbe43: "grenade_frag"}}
			got, _ := buildPickups([]types.BipedPickup{poRamassage(ramasseur, ramassageUS, 2)}, clock,
				pickupInputs{judge: poJuge(nil, SpawnPointsEstablished, pos, dropped)})
			if len(got) != 1 || got[0].Origin != c.attendu {
				t.Errorf("pas %d ms, %s : %+v, origine attendue %q", pas, c.nom, got, c.attendu)
			}
		}
	}
}

// TestCycleDeVehiculeIndependantDeLOrdreDesJumeaux : deux vies nees au meme lieu et au meme instant (un jumeau
// sans fin datee et le vehicule reel) donnent les memes ecarts quel que soit leur rang et l'ordre du tableau,
// et la naissance suivante s'apparie a la mort datee la plus recente. Mutation : l'appariement au voisin
// immediat, ordonne par frame : les jumeaux d'une meme frame echangent leur rang et la mort n'est plus
// le voisin de la naissance suivante.
func TestCycleDeVehiculeIndependantDeLOrdreDesJumeaux(t *testing.T) {
	const s = uint64(1_000_000)
	avant := vehicleCycleLife{bornUS: 0, endUS: 100 * s}
	suivante := vehicleCycleLife{bornUS: 230 * s}
	for _, c := range []struct {
		nom        string
		reel, fant vehicleCycleLife
	}{
		{"le reel nait en premier", vehicleCycleLife{bornUS: 130 * s, endUS: 200 * s}, vehicleCycleLife{bornUS: 130*s + 33_000}},
		{"le reel nait en second", vehicleCycleLife{bornUS: 130*s + 33_000, endUS: 200 * s}, vehicleCycleLife{bornUS: 130 * s}},
	} {
		for _, ordre := range [][]vehicleCycleLife{
			{avant, c.reel, c.fant, suivante}, {suivante, c.fant, c.reel, avant}, {c.fant, avant, suivante, c.reel},
		} {
			gaps, manques := vehicleCycleGaps(append([]vehicleCycleLife(nil), ordre...))
			if len(gaps) != 2 || gaps[0] != 30 || gaps[1] != 30 || manques != 1 {
				t.Errorf("%s : ecarts %v, manques %d, attendu [30 30] et 1", c.nom, gaps, manques)
			}
		}
	}
}

// TestTirContinuIndependantDuPas : le compte des coups d'une rafale (coups a la cadence, trous en
// microsecondes) est le meme a tout pas, y compris pour un trou plus court que la frame. Mutation : le trou
// projete sur les frames ENTIERES qu'il couvre : un trou de 20 ms ne rend muette aucune frame a 100 ms et une
// frame a 10 ms.
func TestTirContinuIndependantDuPas(t *testing.T) {
	doc := &ReplayDocument{
		Tracks:   []Track{{Slot: 11, Points: []Point{{T: 0}, {T: 90_000}}}},
		Loadouts: []Loadout{{T: 0, Slot: 11, W: []string{"0xC3542946", "0xA0955E9E"}}},
	}
	r := types.ContinuousFireBurst{FilmIndex: 4, Weapon: 1, StartUS: 1_234_567, EndUS: 2_000_000,
		StartBound: types.ContinuousFireBoundPressed, EndBound: types.ContinuousFireBoundReleased,
		Holes: []types.ContinuousFireHole{{StartUS: 1_500_000, EndUS: 1_520_000}, {StartUS: 1_800_100, EndUS: 1_800_350},
			{StartUS: 1_900_000, EndUS: 1_950_000}}}
	var ref int
	for i, pas := range pasDeTest {
		clock := replayClock{origin: 0, step: uint64(pas) * 1000, frames: 100_000}
		_, cov := buildFireBursts(context.Background(), doc, []types.ContinuousFireBurst{r},
			types.ContinuousFireStats{Scanned: true}, occupantsPlats(map[uint32]int{11: 4}), clock)
		if cov.Published != 1 {
			t.Fatalf("pas %d ms : %d rafale(s) publiee(s), attendu 1 (%+v)", pas, cov.Published, cov)
		}
		if i == 0 {
			ref = cov.Shots
			// 60 coups/s de 1 234 567 us a 2 000 000 us : 46 coups (k/60 s, k = 0..45) ; les trous en retirent
			// 2 (k = 16, 17), 0 et 3 (k = 40, 41, 42) : 41.
			if ref != 41 {
				t.Fatalf("coups = %d, attendu 41", ref)
			}
		}
		if cov.Shots != ref {
			t.Errorf("pas %d ms : %d coups, %d a 100 ms", pas, cov.Shots, ref)
		}
	}
}

// TestTroncatureDUnVolIndependanteDuPas : un saut impossible entre deux enregistrements consecutifs du film
// coupe le vol, que les deux tombent dans la meme frame ou dans deux frames. Mutation : le test de vitesse
// entre echantillons RETENUS (un par frame) : a 100 ms le saut tombe dans une frame et n'est jamais teste.
func TestTroncatureDUnVolIndependanteDuPas(t *testing.T) {
	const origin, recordUS = uint64(1_000_000), uint64(16_700)
	var pts []types.ProjectileSample
	for k := 0; k < 14; k++ {
		x := float32(k) * 0.5
		if k >= 7 {
			x += 9 // saut de 9 m en un enregistrement : 540 m/s
		}
		pts = append(pts, types.ProjectileSample{TimestampUS: origin + uint64(k)*recordUS, X: x})
	}
	for _, pas := range pasDeTest {
		_, _, tronquees := buildProjectiles([]types.ProjectileTrack{{Slot: 1, Pts: pts}}, origin, uint64(pas)*1000, nil, nil)
		if tronquees != 1 {
			t.Errorf("pas %d ms : %d vol(s) coupe(s), attendu 1", pas, tronquees)
		}
	}
}

// TestDatationDUneOccupationIndependanteDuPas : un ramassage 50 ms apres l'image-cle qui borne deux occupations
// successives d'un socle n'est candidat que de la seconde, a tout pas ; pose a l'instant meme de l'image-cle, il
// reste candidat des deux. Mutation : la comparaison en frames : a 100 ms le ramassage tombe dans la frame de
// l'image-cle et devient candidat des deux occupations, qui deviennent ambigues.
func TestDatationDUneOccupationIndependanteDuPas(t *testing.T) {
	const famille = 0x11223344
	for _, pas := range pasDeTest {
		pasUS := uint64(pas) * 1000
		frame := func(us uint64) int { return int(us / pasUS) }
		const cle = uint64(37_700_000)
		picks := []PadPickup{
			{Pad: 0, tLowUS: 30_000_000, tHighUS: cle},
			{Pad: 0, tLowUS: cle, tHighUS: 50_000_000},
		}
		for i := range picks {
			picks[i].TLow, picks[i].THigh = frame(picks[i].tLowUS), frame(picks[i].tHighUS)
		}
		for _, c := range []struct {
			nom     string
			us      uint64
			attendu [][]int
		}{
			{"50 ms apres l'image-cle", cle + 50_000, [][]int{nil, {0}}},
			{"50 ms avant l'image-cle", cle - 50_000, [][]int{{0}, nil}},
			{"a l'instant de l'image-cle", cle, [][]int{{0}, {0}}},
		} {
			pickups := []Pickup{{T: frame(c.us), tUS: c.us, W: pickupWeaponForm(famille), Kind: PickupWeapon}}
			var st PadDatingStats
			got := candidatsDeDatation([]WeaponPad{{Weapon: padWeaponForm(famille)}}, picks, pickups, &st)
			for i := range c.attendu {
				if len(got[i]) != len(c.attendu[i]) {
					t.Errorf("pas %d ms, %s : occupation %d, candidats %v, attendu %v", pas, c.nom, i, got[i], c.attendu[i])
				}
			}
		}
	}
}
