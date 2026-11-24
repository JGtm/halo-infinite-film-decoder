package replay

// places_de_controle_test.go — LA PLACE DE CONTROLE LUE DANS LES TIRS (`places_de_controle.go`) :
// chaque regle P1 a P4 et le repli sur le siege, sur des tirs construits.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// tirDePlace rend un tir de la place `place` par l unite `slot` a la frame `fr` (pas de 100 ms).
func tirDePlace(place int, slot uint32, fr int) grammar.FireEvent {
	return grammar.FireEvent{TimestampUS: uint64(fr) * 100_000, FilmIndex: place, HasShooter: true,
		Unit: grammar.UnitRef{Present: true, Slot: slot}}
}

// tirSansUnite rend un tir de la place `place` a la frame `fr` dont le record ne nomme aucune unite.
func tirSansUnite(place, fr int) grammar.FireEvent {
	return grammar.FireEvent{TimestampUS: uint64(fr) * 100_000, FilmIndex: place, HasShooter: true}
}

// lecturePlaces lit les places de `fire` pour des slots a pied dont `owner` donne l index de joueur.
func lecturePlaces(fire []grammar.FireEvent, owner map[uint32]int, vies map[int][2]int) placesDeControle {
	return lirePlacesDeControle(fire, occupantsPlats(owner), vsClock(), vies, true)
}

// attendu : la place `place` est tenue par `joueur` a la frame `fr` (`tenue` faux : par personne).
type attenduDePlace struct {
	place, fr, joueur int
	tenue             bool
}

func verifierLesTenues(t *testing.T, p placesDeControle, cas []attenduDePlace) {
	t.Helper()
	for _, c := range cas {
		j, ok := p.joueurA(c.place, c.fr)
		if ok != c.tenue || (ok && j != c.joueur) {
			t.Errorf("place %d frame %d : joueur %d (tenue %v), attendu %d (%v)", c.place, c.fr, j, ok, c.joueur, c.tenue)
		}
	}
}

// TestLaPlaceLueEstTenueSurToutePresence — P2 puis P3 : un joueur qui n a qu une place la tient
// pendant sa presence entiere, au-dela de sa premiere et de sa derniere liaison.
func TestLaPlaceLueEstTenueSurToutePresence(t *testing.T) {
	fire := []grammar.FireEvent{tirDePlace(3, 10, 100), tirDePlace(3, 10, 300), tirDePlace(3, 10, 200)}
	p := lecturePlaces(fire, map[uint32]int{10: 3}, map[int][2]int{3: {20, 500}})
	if p.cov.Bindings != 3 || p.cov.Players != 1 || p.cov.MultiPlace != 0 || p.cov.Handovers != 0 || p.cov.Contested != 0 {
		t.Fatalf("couverture = %+v", p.cov)
	}
	verifierLesTenues(t, p, []attenduDePlace{
		{3, 19, 0, false}, {3, 20, 3, true}, {3, 100, 3, true}, {3, 499, 3, true}, {3, 500, 3, true}, {3, 501, 0, false},
		{4, 100, 0, false}, // une autre place : personne
	})
}

// TestBindingsCompteDesLiaisonsDistinctesEtPasDesTirs : `coverage.controlPlaces.bindings` compte les
// liaisons (place, frame, joueur) distinctes. Deux tirs du meme joueur, a la meme place, dans la
// meme frame, n en font qu une ; un tir a la frame suivante en fait une seconde.
func TestBindingsCompteDesLiaisonsDistinctesEtPasDesTirs(t *testing.T) {
	fire := []grammar.FireEvent{tirDePlace(3, 10, 100), tirDePlace(3, 10, 100), tirDePlace(3, 10, 101)}
	p := lecturePlaces(fire, map[uint32]int{10: 3}, nil)
	if p.cov.Bindings != 2 {
		t.Errorf("bindings = %d pour trois tirs dont deux dans la meme frame, attendu 2 liaisons distinctes", p.cov.Bindings)
	}
	if p.cov.Players != 1 || p.cov.Contested != 0 {
		t.Errorf("couverture = %+v : un seul joueur, rien de conteste", p.cov)
	}
}

// TestLaPlaceLueSansPresenceNEtendPas — P2 seul : sans presence connue du joueur, la tenue est celle
// de ses liaisons.
func TestLaPlaceLueSansPresenceNEtendPas(t *testing.T) {
	fire := []grammar.FireEvent{tirDePlace(3, 10, 100), tirDePlace(3, 10, 300)}
	p := lecturePlaces(fire, map[uint32]int{10: 3}, nil)
	verifierLesTenues(t, p, []attenduDePlace{{3, 99, 0, false}, {3, 100, 3, true}, {3, 300, 3, true}, {3, 301, 0, false}})
}

// TestUnArrivantReprendLaPlaceDuPartant — P2 et P4 : le partant 3 puis l arrivant 7 se succedent a la
// place 3. Chacun la tient sur sa presence ; la part que les deux presences se disputent est a
// personne, et un ecart sans presence l est aussi.
func TestUnArrivantReprendLaPlaceDuPartant(t *testing.T) {
	fire := []grammar.FireEvent{
		tirDePlace(3, 10, 100), tirDePlace(3, 10, 300), // joueur 3
		tirDePlace(3, 20, 600), tirDePlace(3, 20, 800), // joueur 7, arrive a la place 3
		tirDePlace(5, 30, 50), // un joueur seul a sa place
	}
	owner := map[uint32]int{10: 3, 20: 7, 30: 5}
	t.Run("presences qui se recouvrent", func(t *testing.T) {
		p := lecturePlaces(fire, owner, map[int][2]int{3: {0, 500}, 7: {400, 999}, 5: {0, 999}})
		if p.cov.Handovers != 1 || p.cov.Returns != 0 || p.cov.Players != 3 {
			t.Fatalf("couverture = %+v", p.cov)
		}
		verifierLesTenues(t, p, []attenduDePlace{
			{3, 0, 3, true}, {3, 399, 3, true},
			{3, 400, 0, false}, {3, 500, 0, false}, // [400, 500] : les deux presences se recouvrent
			{3, 501, 7, true}, {3, 999, 7, true},
			{5, 999, 5, true},
		})
	})
	t.Run("presences separees par un ecart", func(t *testing.T) {
		p := lecturePlaces(fire, owner, map[int][2]int{3: {0, 320}, 7: {450, 999}})
		verifierLesTenues(t, p, []attenduDePlace{
			{3, 320, 3, true}, {3, 321, 0, false}, {3, 449, 0, false}, {3, 450, 7, true},
		})
	})
	t.Run("sans presence", func(t *testing.T) {
		p := lecturePlaces(fire, owner, nil)
		verifierLesTenues(t, p, []attenduDePlace{
			{3, 300, 3, true}, {3, 301, 0, false}, {3, 599, 0, false}, {3, 600, 7, true}, {3, 800, 7, true},
		})
	})
}

// TestLaTenueNePasseJamaisLaTenueVoisine — P3 : l etendue d un joueur s arrete avant la tenue lue d un
// autre, meme quand sa presence la recouvre.
func TestLaTenueNePasseJamaisLaTenueVoisine(t *testing.T) {
	fire := []grammar.FireEvent{tirDePlace(3, 10, 100), tirDePlace(3, 20, 300)}
	p := lecturePlaces(fire, map[uint32]int{10: 3, 20: 7}, map[int][2]int{3: {0, 900}, 7: {0, 900}})
	verifierLesTenues(t, p, []attenduDePlace{
		{3, 0, 3, true}, {3, 100, 3, true}, {3, 101, 0, false}, {3, 299, 0, false}, {3, 300, 7, true}, {3, 900, 7, true},
	})
}

// TestUnJoueurLuAPlusieursPlacesNEstPasEtendu — P2 : un joueur qui revient sur une AUTRE place ouvre
// une nouvelle tenue ; P3 ne l etend pas, rien ne dit laquelle il tenait hors de ses liaisons.
func TestUnJoueurLuAPlusieursPlacesNEstPasEtendu(t *testing.T) {
	fire := []grammar.FireEvent{
		tirDePlace(2, 10, 100), tirDePlace(2, 10, 200), // premiere fiche
		tirDePlace(6, 10, 700), tirDePlace(6, 10, 800), // revient sur une autre
	}
	p := lecturePlaces(fire, map[uint32]int{10: 3}, map[int][2]int{3: {0, 999}})
	if p.cov.MultiPlace != 1 || p.cov.Players != 1 {
		t.Fatalf("couverture = %+v", p.cov)
	}
	verifierLesTenues(t, p, []attenduDePlace{
		{2, 99, 0, false}, {2, 150, 3, true}, {2, 201, 0, false}, {6, 699, 0, false}, {6, 750, 3, true}, {6, 801, 0, false},
	})
}

// TestUnRetourSurLaMemePlaceSeCompte — P2 : un joueur reprend la place qu un autre a tenue entre-temps ;
// les trois tenues sont lues, et le retour est compte.
func TestUnRetourSurLaMemePlaceSeCompte(t *testing.T) {
	fire := []grammar.FireEvent{
		tirDePlace(4, 10, 100), tirDePlace(4, 10, 200),
		tirDePlace(4, 20, 300), tirDePlace(4, 20, 400),
		tirDePlace(4, 10, 500), tirDePlace(4, 10, 600),
	}
	p := lecturePlaces(fire, map[uint32]int{10: 1, 20: 2}, nil)
	if p.cov.Handovers != 2 || p.cov.Returns != 1 {
		t.Fatalf("couverture = %+v", p.cov)
	}
	verifierLesTenues(t, p, []attenduDePlace{
		{4, 150, 1, true}, {4, 250, 0, false}, {4, 350, 2, true}, {4, 450, 0, false}, {4, 550, 1, true},
	})
}

// TestDeuxJoueursALaMemePlaceALaMemeFrameSeContredisent — P1 : la frame n est a personne, et les deux
// liaisons sont comptees.
func TestDeuxJoueursALaMemePlaceALaMemeFrameSeContredisent(t *testing.T) {
	fire := []grammar.FireEvent{
		tirDePlace(4, 10, 100), tirDePlace(4, 10, 150), tirDePlace(4, 10, 150), tirDePlace(4, 20, 150), tirDePlace(4, 10, 200),
	}
	p := lecturePlaces(fire, map[uint32]int{10: 1, 20: 2}, nil)
	if p.cov.Contested != 2 || p.cov.Bindings != 2 || p.cov.Handovers != 0 {
		t.Fatalf("couverture = %+v, attendu 2 liaisons contestees, 2 retenues, aucun relais", p.cov)
	}
	// La frame contestee n est a personne ; sans presence connue, rien ne s etend de part et d autre.
	verifierLesTenues(t, p, []attenduDePlace{{4, 100, 1, true}, {4, 120, 0, false}, {4, 150, 0, false}, {4, 200, 1, true}})
	avecPresence := lecturePlaces(fire, map[uint32]int{10: 1, 20: 2}, map[int][2]int{1: {0, 300}})
	verifierLesTenues(t, avecPresence, []attenduDePlace{{4, 0, 1, true}, {4, 149, 1, true}, {4, 150, 0, false}, {4, 151, 1, true}, {4, 300, 1, true}})
}

// TestUnTirDeVehiculeOuSansUniteNeLieRien — le tir dont l unite n est pas un bipede du registre
// (vehicule) ou que le record ne nomme pas ne lie aucune place ; la place qu il porte est lue des
// autres tirs de son joueur.
func TestUnTirDeVehiculeOuSansUniteNeLieRien(t *testing.T) {
	const vehicule = 700 // un slot que le registre des occupants ne connait pas
	fire := []grammar.FireEvent{
		tirDePlace(5, 10, 100), tirDePlace(5, 10, 300),
		tirDePlace(5, vehicule, 200), // tir du canon d un vehicule : aucun occupant de bipede
		tirSansUnite(5, 250),
		{TimestampUS: 1, FilmIndex: 9}, // sans tireur
	}
	p := lecturePlaces(fire, map[uint32]int{10: 2}, nil)
	if p.cov.Bindings != 2 || p.cov.Players != 1 {
		t.Fatalf("couverture = %+v, attendu 2 liaisons (les deux tirs a pied)", p.cov)
	}
	verifierLesTenues(t, p, []attenduDePlace{{5, 200, 2, true}, {5, 250, 2, true}, {9, 0, 0, false}})
	t2 := tireursDesPlaces{lues: p}
	if idx, source := t2.tireur(5, 200); idx != 2 || source != tireurParLaPlaceLue {
		t.Errorf("tir du vehicule : tireur %d (source %d), attendu 2 par la place lue", idx, source)
	}
}

// TestSurUnIndexQuiNEstPasLaPlaceRienNEstLu — `actif` faux : aucune place n est lue.
func TestSurUnIndexQuiNEstPasLaPlaceRienNEstLu(t *testing.T) {
	p := lirePlacesDeControle([]grammar.FireEvent{tirDePlace(4, 10, 100)}, occupantsPlats(map[uint32]int{10: 1}), vsClock(), nil, false)
	if p.cov.Read || p.cov.Bindings != 0 {
		t.Fatalf("couverture = %+v", p.cov)
	}
	verifierLesTenues(t, p, []attenduDePlace{{4, 100, 0, false}})
}

// TestLaLectureNeDependPasDeLOrdreDesTirs — memes liaisons dans un autre ordre, memes tenues.
func TestLaLectureNeDependPasDeLOrdreDesTirs(t *testing.T) {
	fire := []grammar.FireEvent{
		tirDePlace(3, 10, 100), tirDePlace(3, 20, 600), tirDePlace(3, 10, 300), tirDePlace(3, 20, 800), tirDePlace(5, 30, 50),
	}
	owner := map[uint32]int{10: 3, 20: 7, 30: 5}
	vies := map[int][2]int{3: {0, 500}, 7: {400, 999}}
	avant := lecturePlaces(fire, owner, vies)
	renverse := make([]grammar.FireEvent, len(fire))
	for i, e := range fire {
		renverse[len(fire)-1-i] = e
	}
	apres := lecturePlaces(renverse, owner, vies)
	for _, place := range []int{3, 5} {
		for fr := 0; fr < 1000; fr += 7 {
			j1, ok1 := avant.joueurA(place, fr)
			j2, ok2 := apres.joueurA(place, fr)
			if j1 != j2 || ok1 != ok2 {
				t.Fatalf("place %d frame %d : %d (%v) puis %d (%v)", place, fr, j1, ok1, j2, ok2)
			}
		}
	}
}

// TestLaPresenceVientDuRosterEtDesVies — la presence d un joueur (P3) est celle de ses intervalles de
// roster, jusqu a leur borne d affichage, etendue a ses vies publiees.
func TestLaPresenceVientDuRosterEtDesVies(t *testing.T) {
	fin := 40
	roster := []RosterEntry{{FilmIndex: 3, Presence: []PresenceInterval{{From: 10, To: 30, ToMax: &fin}, {From: 60, To: 70}}}}
	tracks := []Track{{Slot: 10, StartFrame: 5, EndFrame: 20}}
	vies := viesParJoueur(tracks, roster, occupantsPlats(map[uint32]int{10: 3}), vsClock())
	if v := vies[3]; v != [2]int{5, 70} {
		t.Errorf("presence = %v, attendu [5 70]", v)
	}
	if _, ok := vies[4]; ok {
		t.Error("joueur 4 n a aucune presence")
	}
}

// TestLeSiegeRendLeTireurQuandLaPlaceNEstPasLue — la lecture tient la place 5 (joueur 7) ; la place 6,
// qu aucun tir a unite n a liee, revient a l occupant de son siege, compte comme repli ; la place 8,
// sans siege non plus, reste l index du record.
func TestLeSiegeRendLeTireurQuandLaPlaceNEstPasLue(t *testing.T) {
	horloge := vsClock()
	horloge.fb = fallback.NouveauCompteur()
	lues := lirePlacesDeControle([]grammar.FireEvent{tirDePlace(5, 20, 100), tirDePlace(5, 20, 300)},
		occupantsPlats(map[uint32]int{20: 7}), horloge, map[int][2]int{7: {0, 999}}, true)
	sieges := nouveauxTireursParPlace([]RosterEntry{
		{XUID: "1", FilmIndex: 5, Seat: 5, Presence: []PresenceInterval{{From: 0, To: 999}}},
		{XUID: "2", FilmIndex: 8, Seat: 6, Presence: []PresenceInterval{{From: 0, To: 999}}},
	})
	cov := lues.cov
	fire := []grammar.FireEvent{
		{TimestampUS: 20_000_000, FilmIndex: 5, HasShooter: true},
		{TimestampUS: 20_000_000, FilmIndex: 6, HasShooter: true},
		{TimestampUS: 20_000_000, FilmIndex: 9, HasShooter: true},
		{TimestampUS: 20_000_000, FilmIndex: 5}, // sans tireur : ni compte ni touche
	}
	out, autres := tireursDesPlaces{lues: lues, sieges: sieges}.tirsParPlace(fire, horloge, &cov)
	if autres != 2 || out[0].FilmIndex != 7 || out[1].FilmIndex != 8 || out[2].FilmIndex != 9 || out[3].FilmIndex != 5 {
		t.Errorf("index = %d, %d, %d, %d (%d rendus), attendu 7, 8, 9, 5 (2)", out[0].FilmIndex, out[1].FilmIndex,
			out[2].FilmIndex, out[3].FilmIndex, autres)
	}
	if cov.ShotsByPlace != 1 || cov.ShotsBySeat != 1 || cov.ShotsUnheld != 1 {
		t.Errorf("comptes = %+v : attendu 1 tir par la place lue, 1 par le siege, 1 sans tenant", cov)
	}
	if n := horloge.fb.Compte(fallback.NomTireurParLeSiegeDeLaPlace); n != 2 {
		t.Errorf("repli compte %d fois, attendu 2 (le siege et le sans tenant)", n)
	}
	if fire[0].FilmIndex != 5 {
		t.Error("les evenements de l appelant ont ete modifies")
	}
}
