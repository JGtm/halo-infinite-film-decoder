package grammar

import "testing"

// Le prefixe de version est un bit, suivi de huit bits quand il vaut 1 ; l index de joueur vient apres.
func TestEtatParDefautDeLEntiteJoueurDuMoteur(t *testing.T) {
	// 0 | 010101 -> index 21 ; 1 | 00000011 | 000111 -> index 7.
	br := LecteurSur([]byte{0b00101010, 0b10000001, 0b10001110})
	if got := readPlayerWaypointDefaultState(br); got != 21 {
		t.Fatalf("index %d, attendu 21", got)
	}
	br = LecteurSur([]byte{0b10000001, 0b10001110})
	if got := readPlayerWaypointDefaultState(br); got != 7 {
		t.Fatalf("index %d, attendu 7", got)
	}
}

func TestComptesARebours_RattachementAuSlot(t *testing.T) {
	c := &canalDesHorloges{}
	c.creation = []creationDeJoueur{
		{ts: 100, slot: 5, index: 2},
		{ts: 500, slot: 5, index: 9}, // le slot est repris par un autre joueur
		{ts: 300, slot: 7, index: 4},
	}
	c.lu.Comptes = []LectureCompteARebours{
		{TS: 50, Slot: 5, Joueur: joueurDuMoteurNonRattache},  // avant toute creation : la premiere qui suit
		{TS: 200, Slot: 5, Joueur: joueurDuMoteurNonRattache}, // la creation la plus recente qui precede
		{TS: 900, Slot: 5, Joueur: joueurDuMoteurNonRattache},
		{TS: 100, Slot: 8, Joueur: joueurDuMoteurNonRattache}, // aucune creation
	}
	c.rattacher()
	want := []int{2, 2, 9, joueurDuMoteurNonRattache}
	for i, w := range want {
		if c.lu.Comptes[i].Joueur != w {
			t.Errorf("compte %d : joueur %d, attendu %d", i, c.lu.Comptes[i].Joueur, w)
		}
	}
	if c.lu.Stats.CompteSansJoueur != 1 {
		t.Errorf("sans joueur %d, attendu 1", c.lu.Stats.CompteSansJoueur)
	}
}
