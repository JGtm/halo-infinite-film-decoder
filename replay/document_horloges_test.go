package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

const (
	testOrigine = 10_000_000
	testPas     = 33_000
)

func TestSerieDeLHorloge(t *testing.T) {
	reads := []grammar.LectureHorloge{
		{TS: testOrigine - 1_000_000, Limite: 1311, Reste: 1311, Etat: 1},
		{TS: testOrigine, Limite: 1311, Reste: 1311, Etat: 1}, // identique a la precedente : non republiee
		{TS: testOrigine + 1_000_000, Limite: 1311, Reste: 1311, Etat: 6},
		{TS: testOrigine + 2_000_000, Limite: 1311, Reste: 1309, Etat: 6},
		{TS: testOrigine + 3_000_000, Limite: 8968, Reste: 35423, Etat: 6}, // temps au-dessus de la limite : impossible
		{TS: testOrigine + 3_500_000, Limite: 1311, Reste: 1300, Etat: 9},  // etat que la mesure ne soutient pas
		{TS: testOrigine + 4_000_000, Limite: 0, Reste: 40, Etat: 19},      // sans limite, la minuterie monte
	}
	var cov ClocksCoverage
	got := serieDeLHorloge(reads, testOrigine, testPas, &cov)
	wantT := []int{-31, 30, 60, 121}
	wantEtat := []int{1, 6, 6, 19}
	if len(got) != len(wantT) {
		t.Fatalf("serie %+v", got)
	}
	for i := range got {
		if got[i].T != wantT[i] || got[i].State != wantEtat[i] {
			t.Errorf("echantillon %d : %+v, attendu t %d etat %d", i, got[i], wantT[i], wantEtat[i])
		}
	}
	if got[1].Left != got[1].Limit || got[2].Left >= got[2].Limit || got[3].Limit != 0 {
		t.Errorf("secondes : %+v", got)
	}
	if cov.ClockReads != 7 || cov.ClockPublished != 4 || cov.ClockImpossible != 1 || cov.ClockUnmeasuredState != 1 {
		t.Errorf("couverture %+v", cov)
	}
}

// compte rend un compte a rebours de `total` secondes : une lecture par seconde de `debut` a `fin` restantes.
func compte(joueur int, ts0 uint64, total, debut, fin uint16) []grammar.LectureCompteARebours {
	var out []grammar.LectureCompteARebours
	for r := debut; ; r-- {
		out = append(out, grammar.LectureCompteARebours{TS: ts0 + uint64(debut-r)*1_000_000, Joueur: joueur,
			Reste: r, Total: total})
		if r == fin {
			return out
		}
	}
}

func TestPeriodesDeCompte(t *testing.T) {
	t0 := uint64(testOrigine + 100_000_000)
	var reads []grammar.LectureCompteARebours
	reads = append(reads, compte(3, t0, 8, 8, 0)...)              // complet
	reads = append(reads, compte(3, t0+50_000_000, 8, 6, 1)...)   // debut manque, fin avant zero
	reads = append(reads, compte(5, t0+10_000_000, 10, 10, 0)...) // autre joueur, meme intervalle
	reads = append(reads, compte(6, t0+200_000_000, 8, 8, 8)...)  // une lecture seule qui commence : une periode
	reads = append(reads,
		grammar.LectureCompteARebours{TS: t0, Joueur: 3, Reste: 8, Total: 8},                // repetition de la premiere
		grammar.LectureCompteARebours{TS: t0, Joueur: 3, Reste: 965, Total: 20},             // impossible
		grammar.LectureCompteARebours{TS: t0, Joueur: 3, Reste: 0, Total: 0},                // total nul
		grammar.LectureCompteARebours{TS: t0, Joueur: -1, Reste: 4, Total: 8},               // entite sans joueur
		grammar.LectureCompteARebours{TS: t0 + 80_000_000, Joueur: 7, Reste: 5, Total: 906}, // lecture isolee
	)
	var cov ClocksCoverage
	got := periodesDeCompte(reads, testOrigine, testPas, rosterDesComptes(), &cov)
	want := []RespawnCountdown{
		{FilmIndex: 3, From: 3030, To: 3272, Seconds: 8, StartRead: true, EndRead: true},
		{FilmIndex: 5, From: 3333, To: 3636, Seconds: 10, StartRead: true, EndRead: true},
		// le debut est entame de deux secondes et la fin a une seconde de zero : les deux se deduisent
		{FilmIndex: 3, From: 4484, To: 4727, Seconds: 8},
		// une lecture seule qui commence : la fin se deduit
		{FilmIndex: 6, From: 9090, To: 9333, Seconds: 8, StartRead: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("periodes %+v, attendu %+v", got, want)
	}
	if cov.CountdownReads != len(reads) || cov.CountdownImpossible != 2 || cov.CountdownNoPlayer != 1 || cov.CountdownIsolated != 1 ||
		cov.Countdowns != 4 || cov.StartDeduced != 1 || cov.EndDeduced != 2 {
		t.Errorf("couverture %+v", cov)
	}
}

// rosterDesComptes : un occupant par index 3, 5, 6 et 7, present pendant tout le film.
func rosterDesComptes() []RosterEntry {
	present := []PresenceInterval{{From: 0, To: 100_000}}
	return []RosterEntry{{FilmIndex: 3, Presence: present}, {FilmIndex: 5, Presence: present},
		{FilmIndex: 6, Presence: present}, {FilmIndex: 7, Presence: present}}
}

// Une lecture manquante au milieu d un compte ne le coupe pas en deux periodes identiques.
func TestPeriodesDeCompte_UneLectureManquanteNeCoupePasLeCompte(t *testing.T) {
	t0 := uint64(testOrigine + 100_000_000)
	reads := append(compte(3, t0, 8, 8, 7), compte(3, t0+3_000_000, 8, 5, 0)...) // les secondes 6 manque
	var cov ClocksCoverage
	got := periodesDeCompte(reads, testOrigine, testPas, rosterDesComptes(), &cov)
	want := []RespawnCountdown{{FilmIndex: 3, From: 3030, To: 3272, Seconds: 8, StartRead: true, EndRead: true}}
	if !reflect.DeepEqual(got, want) || cov.Countdowns != 1 {
		t.Fatalf("periodes %+v (couverture %+v), attendu %+v", got, cov, want)
	}
}

// Deux periodes de meme (joueur, debut, fin) n en font qu une, quel que soit l ordre des lectures.
func TestPeriodesDeCompte_LesDoublonsSontFusionnes(t *testing.T) {
	t0 := uint64(testOrigine + 100_000_000)
	a, b := compte(3, t0, 8, 8, 0), compte(3, t0, 8, 8, 0)
	for i := range b {
		b[i].TS += 400_000 // la meme periode lue deux fois, decalee de moins d une seconde
	}
	var cov ClocksCoverage
	got := periodesDeCompte(append(a, b...), testOrigine, testPas, rosterDesComptes(), &cov)
	if len(got) != 1 {
		t.Fatalf("periodes %+v, attendu une", got)
	}
}

func TestOccupantUnique(t *testing.T) {
	absent, fin := 2000, 1999
	roster := []RosterEntry{
		{FilmIndex: 1, Presence: []PresenceInterval{{From: 0, To: 5000}}},
		{FilmIndex: 2, Presence: []PresenceInterval{{From: 0, To: fin, ToMax: &absent}}},
		{FilmIndex: 3}, {FilmIndex: 4}, {FilmIndex: 4}, // l index 4 a deux occupants
		{FilmIndex: 5, Presence: []PresenceInterval{{From: 0, To: 100}, {From: 3000, To: 4000}}},
	}
	tests := []struct {
		nom   string
		index int
		from  int
		want  bool
		champ func(ClocksCoverage) int
	}{
		{"occupant unique present", 1, 4000, true, nil},
		{"depart avant le compte", 1, 6000, false, func(c ClocksCoverage) int { return c.WithheldOutsidePresence }},
		{"present jusqu a la borne haute", 2, 2000, true, nil},
		{"presence vide : jamais presente", 3, 9000, false, func(c ClocksCoverage) int { return c.WithheldOutsidePresence }},
		{"index repris", 4, 10, false, func(c ClocksCoverage) int { return c.WithheldReusedIndex }},
		{"index absent du roster", 9, 10, false, func(c ClocksCoverage) int { return c.WithheldUnknownIndex }},
		{"deuxieme intervalle de presence", 5, 3500, true, nil},
		{"entre deux presences", 5, 2000, false, func(c ClocksCoverage) int { return c.WithheldOutsidePresence }},
	}
	for _, c := range tests {
		var cov ClocksCoverage
		got := occupantUnique(roster, RespawnCountdown{FilmIndex: c.index, From: c.from}, &cov)
		if got != c.want || (c.champ != nil && c.champ(cov) != 1) {
			t.Errorf("%s : %v, couverture %+v", c.nom, got, cov)
		}
	}
}
