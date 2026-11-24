package replay

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// document_stances_boutons_test.go — LES COMMANDES D ACCROUPI ET DE SAUT PUBLIEES : la porte de cadence, le debut
// du saut, le vehicule, les millisecondes et la couverture. Scenarios synthetiques, sans film.

// pleineCadence et sousLaCadence sont les compteurs de pas de deux films : 60 Hz et 30 Hz.
var (
	pleineCadence  = types.ControlHoldStats{StepUS: 16_600, StepGaps: 50_000, Gaps: 60_000}
	sousLaCadence  = types.ControlHoldStats{StepUS: 33_400, StepGaps: 50_000, Gaps: 60_000}
	sansPasMesure  = types.ControlHoldStats{StepUS: 16_600, StepGaps: types.EcartsDuPasMin - 1}
	troisGenresLus = []string{types.MovementMoveCommand, types.MovementCrouchCommand, types.MovementJumpCommand}
)

// holdDuGenre rend une commande tenue de la place 3 (occupee par les vies 513 et 514 de entreeDesCommandes), posee
// puis relachee sur les entrees lues.
func holdDuGenre(kind string, debut, fin uint64) types.ControlHold {
	return types.ControlHold{Kind: kind, FilmIndex: 3, StartUS: debut, EndUS: fin, StartBound: types.ContinuousFireBoundPressed,
		EndBound: types.ContinuousFireBoundReleased, Entries: 2}
}

// troisCommandes rend une commande de chaque genre sur [5 ms, 8 ms) de la place 3.
func troisCommandes() []types.ControlHold {
	var hs []types.ControlHold
	for _, k := range troisGenresLus {
		hs = append(hs, holdDuGenre(k, 5_000, 8_000))
	}
	return hs
}

// TestLesGenresNeufsNeSePublientQueAPleineCadence : a pleine cadence, l accroupi et le saut se publient avec leurs
// millisecondes ; sous la cadence, ou sans pas mesurable, ils ne se publient pas et se comptent (`gated`), et le
// deplacement n est pas touche. MUTATIONS : la porte retiree, la porte appliquee au deplacement, une commande
// ecartee sans compte -> ROUGE.
func TestLesGenresNeufsNeSePublientQueAPleineCadence(t *testing.T) {
	cas := []struct {
		nom     string
		stats   types.ControlHoldStats
		publies int
	}{{"60 Hz", pleineCadence, 3}, {"30 Hz", sousLaCadence, 1}, {"pas non mesurable", sansPasMesure, 1}}
	for _, c := range cas {
		in := entreeDesCommandes(troisCommandes()...)
		in.stats = c.stats
		got, cov := buildControlStances(in)
		if len(got) != c.publies {
			t.Errorf("%s : %d intervalle(s) %+v, attendu %d", c.nom, len(got), got, c.publies)
		}
		ouvert := c.publies == 3
		if cov.FullCadence != ouvert || cov.StepUS != c.stats.StepUS {
			t.Errorf("%s : couverture %+v", c.nom, cov)
		}
		gated := 0
		if !ouvert {
			gated = 1
		}
		if cov.CrouchCommand.Gated != gated || cov.JumpCommand.Gated != gated || cov.Published != 1 {
			t.Errorf("%s : couverture %+v, attendu %d commande(s) ecartee(s) par genre et le deplacement publie", c.nom, cov, gated)
		}
	}
}

// TestUnSautSeDateALaMilliseconde : un saut d un tick tient dans une frame de 100 ms ; `ms0` et `ms1` le datent sans
// deborder de l intervalle lu, meme coupe par la fin d une vie. MUTATION : ms deduits des frames -> ROUGE.
func TestUnSautSeDateALaMilliseconde(t *testing.T) {
	in := entreeDesCommandes(holdDuGenre(types.MovementJumpCommand, 6_250_000, 6_267_000)) // frame 52 de la grille
	in.pont.clock = replayClock{origin: 1_000_000, step: 100_000, frames: 100}
	in.tracks = []Track{{Slot: 513, StartFrame: 0, EndFrame: 60}}
	in.stats = pleineCadence
	got, cov := buildControlStances(in)
	if len(got) != 1 || got[0].Kind != types.MovementJumpCommand || got[0].T0 != 52 || got[0].T1 != 52 ||
		got[0].Ms0 == nil || got[0].Ms1 == nil || *got[0].Ms0 != 5250 || *got[0].Ms1 != 5267 {
		t.Fatalf("intervalles %+v, attendu un saut de la frame 52, de 5250 a 5267 ms", got)
	}
	if cov.JumpCommand.Holds != 0 || cov.JumpCommand.Published != 1 || cov.JumpCommand.Intervals != 1 {
		t.Errorf("couverture %+v", cov.JumpCommand)
	}
	// coupe par la fin de la vie : la borne est celle de la frame, pas celle du saut lu.
	in = entreeDesCommandes(holdDuGenre(types.MovementCrouchCommand, 5_000, 20_000))
	in.stats = pleineCadence
	got, _ = buildControlStances(in)
	if len(got) != 2 || *got[0].Ms0 != 5 || *got[0].Ms1 != 11 || *got[1].Ms0 != 12 || *got[1].Ms1 != 20 {
		t.Errorf("intervalles %+v, attendu [5, 11) puis [12, 20) ms", got)
	}
}

// TestLeDeplacementNePorteNiMsNiPorte : la commande de deplacement garde sa forme : pas de millisecondes, pas de
// porte, pas d exclusion en vehicule.
func TestLeDeplacementNePorteNiMsNiPorte(t *testing.T) {
	in := entreeDesCommandes(holdDeTest(3, 5_000, 8_000))
	in.stats = sousLaCadence
	in.vehicles = []VehicleTrack{{Rides: []VehicleRide{{Slot: 513, T0: 0, T1: 20}}}}
	got, _ := buildControlStances(in)
	want := []Stance{{Slot: 513, Kind: types.MovementMoveCommand, T0: 5, T1: 8}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("intervalles %+v, attendu %+v", got, want)
	}
}

// TestUnSautApresUnTrouSeCompteEtPasUnAccroupi : un saut dont le debut suit un trou de lecture n a pas de debut
// connu : il se compte (`afterHole`) ; un accroupi dans le meme cas se publie, son trou reste muet.
func TestUnSautApresUnTrouSeCompteEtPasUnAccroupi(t *testing.T) {
	saut, accroupi := holdDuGenre(types.MovementJumpCommand, 5_000, 8_000), holdDuGenre(types.MovementCrouchCommand, 5_000, 8_000)
	saut.StartBound, accroupi.StartBound = types.ContinuousFireBoundHole, types.ContinuousFireBoundHole
	in := entreeDesCommandes(saut, accroupi)
	in.stats = pleineCadence
	got, cov := buildControlStances(in)
	if len(got) != 1 || got[0].Kind != types.MovementCrouchCommand {
		t.Fatalf("intervalles %+v, attendu l accroupi seul", got)
	}
	if cov.JumpCommand.AfterHole != 1 || cov.JumpCommand.Published != 0 || cov.CrouchCommand.AfterHole != 0 ||
		cov.CrouchCommand.Published != 1 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestLesGenresNeufsSeTaisentEnVehicule : rien ne se pose pendant un episode de vehicule de l occupant ; le reste de
// l intervalle se publie, et une commande tout entiere a bord se compte (`inVehicule`). MUTATION : episodes
// ignores -> ROUGE.
func TestLesGenresNeufsSeTaisentEnVehicule(t *testing.T) {
	in := entreeDesCommandes(holdDuGenre(types.MovementCrouchCommand, 2_000, 9_000),
		holdDuGenre(types.MovementJumpCommand, 4_500, 5_500))
	in.stats = pleineCadence
	in.vehicles = []VehicleTrack{{Rides: []VehicleRide{{Slot: 513, T0: 4, T1: 6}, {Slot: 520, T0: 0, T1: 99}}}}
	got, cov := buildControlStances(in)
	want := []Stance{{Slot: 513, Kind: types.MovementCrouchCommand, T0: 2, T1: 3}, {Slot: 513, Kind: types.MovementCrouchCommand, T0: 7, T1: 9}}
	if len(got) != 2 || got[0].T0 != want[0].T0 || got[0].T1 != want[0].T1 || got[1].T0 != want[1].T0 || got[1].T1 != want[1].T1 {
		t.Errorf("intervalles %+v, attendu l accroupi sur les frames 2 a 3 puis 7 a 9 (a bord de 4 a 6)", got)
	}
	if cov.CrouchCommand.Published != 1 || cov.CrouchCommand.Intervals != 2 || cov.JumpCommand.InVehicle != 1 ||
		cov.JumpCommand.Published != 0 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestChaqueCommandeLueAUnSortExclusif : pour le saut, lus = publies + sans vie + sans joueur + en
// vehicule + apres un trou + sous la cadence, sur un melange de cas.
func TestChaqueCommandeLueAUnSortExclusif(t *testing.T) {
	apresTrou := holdDuGenre(types.MovementJumpCommand, 5_000, 6_000)
	apresTrou.StartBound = types.ContinuousFireBoundHole
	hs := []types.ControlHold{
		holdDuGenre(types.MovementJumpCommand, 5_000, 8_000),   // publiee
		holdDuGenre(types.MovementJumpCommand, 50_000, 60_000), // aucune vie
		apresTrou, // debut apres un trou
		{Kind: types.MovementJumpCommand, FilmIndex: 9, StartUS: 1_000, EndUS: 5_000}, // place sans occupant
		holdDuGenre(types.MovementJumpCommand, 4_500, 5_500),                          // a bord
	}
	in := entreeDesCommandes(hs...)
	in.stats = pleineCadence
	in.stats.Jump = types.ControlHoldCount{Holds: len(hs)}
	in.vehicles = []VehicleTrack{{Rides: []VehicleRide{{Slot: 513, T0: 4, T1: 6}}}}
	_, cov := buildControlStances(in)
	j := cov.JumpCommand
	if j.Published != 1 || j.NoTrack != 1 || j.AfterHole != 1 || j.NoPlayer != 1 || j.InVehicle != 1 || j.Gated != 0 ||
		j.Holds != j.Published+j.NoTrack+j.NoPlayer+j.InVehicle+j.AfterHole+j.Gated {
		t.Errorf("couverture du saut %+v", j)
	}
}

// TestSansLesRidesRetireLesEpisodes : les passages a pied d un intervalle, episodes bornes compris.
func TestSansLesRidesRetireLesEpisodes(t *testing.T) {
	rides := [][2]int{{3, 4}, {4, 6}, {9, 12}, {20, 30}}
	cas := []struct {
		t0, t1 int
		want   [][2]int
	}{
		{0, 2, [][2]int{{0, 2}}},
		{0, 10, [][2]int{{0, 2}, {7, 8}}},
		{5, 6, nil},
		{7, 8, [][2]int{{7, 8}}},
		{10, 25, [][2]int{{13, 19}}},
		{0, 40, [][2]int{{0, 2}, {7, 8}, {13, 19}, {31, 40}}},
	}
	for _, c := range cas {
		if got := sansLesRides(c.t0, c.t1, rides); !reflect.DeepEqual(got, c.want) {
			t.Errorf("[%d, %d] : %v, attendu %v", c.t0, c.t1, got, c.want)
		}
	}
}

// TestLaVieDeLaPlaceSeDecideEnUnSeulPoint : le pont d une place vers la vie de son occupant ne se recopie pas. Le
// fichier qui l ecrit est le seul de production a lire l occupant d une place ET le joueur d un slot ; le troisieme
// appel qui viendrait recopier la paire echoue ici. MUTATION : la paire recopiee dans un autre fichier -> ROUGE.
func TestLaVieDeLaPlaceSeDecideEnUnSeulPoint(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("balayage des fichiers du paquet : %v", err)
	}
	lus := 0
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") || f == "vie_de_la_place.go" {
			continue
		}
		src, err := os.ReadFile(f) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		lus++
		if s := string(src); strings.Contains(s, ".places.tireur(") && strings.Contains(s, ".indexA(") {
			t.Errorf("%s decide a son tour quelle vie porte une place : passer par pontDesPlaces.vieDeLaPlace", f)
		}
	}
	if lus < 20 {
		t.Fatalf("balayage muet : %d fichiers de production lus", lus)
	}
}
