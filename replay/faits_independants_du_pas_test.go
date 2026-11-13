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
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// pasDeTest sont les pas de grille que les tests de ce fichier rejouent.
var pasDeTest = []int{100, 66, 50, 33, 20, 10}

// TestPointDuPorteurIndependantDuPas : le point publie qui localise un joueur est le plus proche en temps EXACT, a au
// plus `flagCarrierPointToleranceMS`, a tout pas ; a egalite celui d'avant ; jamais celui d'une autre vie. Les
// points portent leur instant exact : la frame ne sert qu'a la publication. Mutations : `bd <= 1` (une image) ;
// tolerance convertie en images (`framesOfMS`, 299 ms a 100 ms, 230 ms a 33 ms : 250 ms accepte a 100 et 66 ms,
// refuse a 50 ms et en dessous) ; « dernier avant » au lieu du plus proche ; sans garde de vie.
func TestPointDuPorteurIndependantDuPas(t *testing.T) {
	for _, pas := range pasDeTest {
		pointAu := func(ms int, x float32) Point { return Point{T: ms / pas, us: uint64(ms) * 1000, X: x} }
		piste := func(pts ...Point) []Track { return []Track{{Points: pts}} }
		deuxVies := func(a, b Point) []Track { return []Track{{Points: []Point{a}}, {Points: []Point{b}}} }
		for _, c := range []struct {
			nom    string
			tracks []Track
			ms     int
			x      float32
			ok     bool
		}{
			{"a l'instant du point", piste(pointAu(1000, 1)), 1000, 1, true},
			{"199 ms apres", piste(pointAu(1000, 1)), 1199, 1, true},
			{"200 ms apres (borne incluse)", piste(pointAu(1000, 1)), 1200, 1, true},
			{"201 ms apres", piste(pointAu(1000, 1)), 1201, 0, false},
			{"250 ms apres", piste(pointAu(1000, 1)), 1250, 0, false},
			{"299 ms apres", piste(pointAu(1000, 1)), 1299, 0, false},
			{"199 ms avant", piste(pointAu(1000, 1)), 801, 1, true},
			{"201 ms avant", piste(pointAu(1000, 1)), 799, 0, false},
			{"le plus proche : le suivant (120 ms) plutot que le precedent (180 ms)", piste(pointAu(1000, 1), pointAu(1300, 2)), 1180, 2, true},
			{"le plus proche : le precedent (80 ms) plutot que le suivant (220 ms)", piste(pointAu(1000, 1), pointAu(1300, 2)), 1080, 1, true},
			{"egalite : celui d'avant", piste(pointAu(1000, 1), pointAu(1100, 2)), 1050, 1, true},
			{"point seulement apres, a 150 ms", piste(pointAu(1150, 2)), 1000, 2, true},
			{"point seulement apres, a 250 ms", piste(pointAu(1250, 2)), 1000, 0, false},
			{"mort puis reapparition : la vie d'avant, a 150 ms, est ecartee", deuxVies(pointAu(850, 1), pointAu(950, 2)), 1000, 2, true},
			{"autre vie avant (100 ms), bonne vie apres (50 ms)", deuxVies(pointAu(900, 1), pointAu(1050, 2)), 1000, 2, true},
			{"la vie suivante nait apres l'instant, hors tolerance : le point de la vie en cours reste", deuxVies(pointAu(850, 1), pointAu(1250, 2)), 1000, 1, true},
			{"deux vies, le point d'avant est de la vie en cours", deuxVies(pointAu(900, 1), pointAu(1100, 2)), 1000, 1, true},
		} {
			p, ok := pointOfXUIDAt(c.tracks, uint64(c.ms)*1000)
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
			tracks := []Track{{Points: []Point{{T: f, us: naissanceUS, X: 5, Y: 5}}}}
			_, ok := flagFreeDropInside(r, ctx, tracks, scan)
			if ok != c.ferme {
				t.Errorf("pas %d ms, vie libre %s : ferme = %v, attendu %v", pas, c.nom, ok, c.ferme)
			}
		}
	}
}

// TestOrigineDUnRamassageIndependanteDuPas : l'ordre entre un ramassage et une pose se decide en
// microsecondes, et l'echange (la pose du ramasseur lui-meme, d'une autre famille) n'est pas une origine. Le juge ne
// recoit que des instants ; la boucle sur les pas rejoue le cablage de `buildPickups` (qui publie la frame du pas) pour
// prouver que celle-ci ne rentre jamais dans la decision.
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

// TestCycleDeVehiculeIndependantDeLOrdreDesJumeaux : deux vies nees au meme lieu a 33 ms (un jumeau sans fin datee et
// le vehicule reel) sont UN occupant, quel que soit leur rang et l'ordre du tableau : la naissance suivante
// s'apparie a la fin datee du reel. Mutation : l'appariement au voisin immediat ordonne par frame, qui laisse
// l'ordre du tableau trancher les jumeaux d'une meme frame (et perd l'ecart quand le jumeau passe en dernier).
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
			if !slices.Equal(gaps, []float64{30, 30}) || manques != 0 {
				t.Errorf("%s : ecarts %v, manques %d, attendu [30 30] et 0", c.nom, gaps, manques)
			}
		}
	}
	// Deux naissances a plus de 100 ms ne sont plus jumelles : le premier occupant n'a pas de fin datee, manque.
	lointain := vehicleCycleLife{bornUS: 130*s + 300_000}
	if gaps, manques := vehicleCycleGaps([]vehicleCycleLife{avant, {bornUS: 130 * s, endUS: 200 * s}, lointain, suivante}); manques != 2 || len(gaps) != 1 {
		t.Errorf("non jumeles : ecarts %v, manques %d, attendu 1 ecart et 2 manques", gaps, manques)
	}
}

// TestCycleDeVehiculeNApparieQuALaVieQuiOccupaitLEmplacement : A nait a 0 s et meurt a 100 s ; B renait dans l'amas
// a 50 s alors que A vit encore ; C nait a 130 s. C repond a la disparition de B (jamais datee), pas a celle de A : aucun
// ecart de 30 s ne se fabrique. Mutation : l'appariement a la derniere mort datee de l'amas, quelle que soit la vie.
func TestCycleDeVehiculeNApparieQuALaVieQuiOccupaitLEmplacement(t *testing.T) {
	const s = uint64(1_000_000)
	a := vehicleCycleLife{bornUS: 0, endUS: 100 * s}
	b := vehicleCycleLife{bornUS: 50 * s}
	c := vehicleCycleLife{bornUS: 130 * s}
	gaps, manques := vehicleCycleGaps([]vehicleCycleLife{a, b, c})
	if len(gaps) != 0 || manques != 2 {
		t.Errorf("ecarts %v, manques %d, attendu aucun ecart et 2 manques", gaps, manques)
	}
	if gaps, manques = vehicleCycleGaps([]vehicleCycleLife{a, c}); !slices.Equal(gaps, []float64{30}) || manques != 0 {
		t.Errorf("sans B : ecarts %v, manques %d, attendu [30] et 0", gaps, manques)
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
