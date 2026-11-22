package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entrees de test : une frame dure 1000 µs, l origine est zero.
func entreeDesCommandes(holds ...types.ControlHold) controlStanceInputs {
	return controlStanceInputs{
		holds: holds,
		tracks: []Track{
			{Slot: 513, StartFrame: 0, EndFrame: 10},
			{Slot: 514, StartFrame: 12, EndFrame: 30},
			{Slot: 520, StartFrame: 0, EndFrame: 30},
		},
		occ:    occupantsPlats(map[uint32]int{513: 3, 514: 3, 520: 7}),
		places: tireursParPlace{},
		clock:  replayClock{origin: 0, step: 1000, frames: 100},
	}
}

func holdDeTest(place int, debut, fin uint64, trous ...types.ContinuousFireHole) types.ControlHold {
	return types.ControlHold{FilmIndex: place, StartUS: debut, EndUS: fin, Holes: trous}
}

// TestUneCommandeSeCoupeAuxViesDeLOccupant : un intervalle qui traverse deux vies du joueur donne un intervalle
// par vie, borne a la vie ; la vie d un autre joueur ne le porte pas. MUTATIONS : une vie d un autre joueur
// retenue, l intervalle non borne a la vie -> ROUGE.
func TestUneCommandeSeCoupeAuxViesDeLOccupant(t *testing.T) {
	in := entreeDesCommandes(holdDeTest(3, 5_000, 20_000))
	got, cov := buildControlStances(in)
	want := []Stance{{Slot: 513, Kind: "moveCommand", T0: 5, T1: 10}, {Slot: 514, Kind: "moveCommand", T0: 12, T1: 20}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("intervalles %+v, attendu %+v", got, want)
	}
	if cov.Holds != 0 || cov.Published != 1 || cov.Intervals != 2 || cov.NoTrack != 0 || cov.NoPlayer != 0 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestUneCommandeSansVieDeSonOccupantSeCompte : aucune vie ne recouvre l intervalle, ou les vies qui le
// recouvrent ne sont pas celles de l occupant : rien n est publie, et chaque cas a son compteur.
func TestUneCommandeSansVieDeSonOccupantSeCompte(t *testing.T) {
	in := entreeDesCommandes(
		holdDeTest(3, 40_000, 50_000), // aucune vie apres la frame 30
		holdDeTest(9, 1_000, 5_000),   // la place 9 n a aucun occupant lu
	)
	in.stats = types.ControlHoldStats{Holds: 2}
	got, cov := buildControlStances(in)
	if len(got) != 0 || cov.NoTrack != 1 || cov.NoPlayer != 1 || cov.Published != 0 ||
		cov.Holds != cov.Published+cov.NoTrack+cov.NoPlayer {
		t.Errorf("intervalles %+v, couverture %+v", got, cov)
	}
}

// TestLesTrousDeLaLectureSeTaisentSurLesFramesQuIlsCouvrentEntieres : un trou qui couvre des frames entieres les
// coupe de l intervalle ; un trou plus court qu une frame ne coupe rien. MUTATION : arrondir le debut du trou
// a la frame precedente -> ROUGE.
func TestLesTrousDeLaLectureSeTaisentSurLesFramesQuIlsCouvrentEntieres(t *testing.T) {
	long := types.ContinuousFireHole{StartUS: 14_000, EndUS: 17_000} // frames 14, 15 et 16 muettes
	court := types.ContinuousFireHole{StartUS: 20_100, EndUS: 20_300}
	in := entreeDesCommandes(types.ControlHold{FilmIndex: 7, StartUS: 12_000, EndUS: 25_000,
		Holes: []types.ContinuousFireHole{long, court}})
	got, _ := buildControlStances(in)
	want := []Stance{{Slot: 520, Kind: "moveCommand", T0: 12, T1: 13}, {Slot: 520, Kind: "moveCommand", T0: 17, T1: 25}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("intervalles %+v, attendu %+v", got, want)
	}
}

// TestLesCommandesFontLAllerRetourDesFaits : chaque champ d une commande, ses trous et les compteurs se relisent a
// l identique ; une liste vide se relit nil.
func TestLesCommandesFontLAllerRetourDesFaits(t *testing.T) {
	cas := []struct {
		hs []types.ControlHold
		st types.ControlHoldStats
	}{
		{nil, types.ControlHoldStats{}},
		{[]types.ControlHold{
			{FilmIndex: 3, StartUS: 4_500_000, EndUS: 4_900_000, StartBound: "pressed",
				EndBound: "hole", Entries: 4, Holes: []types.ContinuousFireHole{{StartUS: 4_600_000, EndUS: 4_700_000}}},
			{FilmIndex: 31, StartUS: 4_500_000, EndUS: 4_500_000, StartBound: "hole", EndBound: "filmEnd", Entries: 1},
		}, types.ControlHoldStats{Holds: 2, WithHole: 2, InnerHoles: 1, Entries: 9}},
	}
	for _, c := range cas {
		w := &gwriter{}
		encodeCommandes(w, c.hs, c.st)
		r := &greader{b: w.b}
		hs, st := decodeCommandes(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(hs, c.hs) || st != c.st {
			t.Errorf("relu %+v %+v, attendu %+v %+v", hs, st, c.hs, c.st)
		}
	}
}
