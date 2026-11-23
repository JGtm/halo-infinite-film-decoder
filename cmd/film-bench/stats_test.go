package main

import (
	"slices"
	"testing"
	"time"
)

func ms10(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestTournerDecaleLOrdreEtNeModifiePasLEntree(t *testing.T) {
	noms := []string{"a", "b", "c"}
	cas := map[int][]string{
		0: {"a", "b", "c"},
		1: {"b", "c", "a"},
		2: {"c", "a", "b"},
		3: {"a", "b", "c"},
		4: {"b", "c", "a"},
	}
	for passe, attendu := range cas {
		if got := tourner(noms, passe); !slices.Equal(got, attendu) {
			t.Errorf("passe %d : %v, attendu %v", passe, got, attendu)
		}
	}
	if !slices.Equal(noms, []string{"a", "b", "c"}) {
		t.Errorf("l'entree a ete modifiee : %v", noms)
	}
	if got := tourner(nil, 3); got != nil {
		t.Errorf("liste vide : %v", got)
	}
}

func TestTournerDonneATousLesFilmsToutesLesPositions(t *testing.T) {
	noms := []string{"a", "b", "c", "d"}
	positions := map[string]map[int]bool{}
	for p := range len(noms) {
		for i, n := range tourner(noms, p) {
			if positions[n] == nil {
				positions[n] = map[int]bool{}
			}
			positions[n][i] = true
		}
	}
	for n, pos := range positions {
		if len(pos) != len(noms) {
			t.Errorf("film %s : %d positions distinctes sur %d", n, len(pos), len(noms))
		}
	}
}

func TestResumerMedianeMinMax(t *testing.T) {
	impair := resumer([]time.Duration{ms10(30), ms10(10), ms10(20)})
	if impair.Mediane != ms10(20) || impair.Min != ms10(10) || impair.Max != ms10(30) || impair.Passes != 3 {
		t.Errorf("impair : %+v", impair)
	}
	pair := resumer([]time.Duration{ms10(40), ms10(10), ms10(20), ms10(30)})
	if pair.Mediane != ms10(25) || pair.Passes != 4 {
		t.Errorf("pair : %+v", pair)
	}
	if (resumer(nil) != Stat{}) {
		t.Error("liste vide : valeur zero attendue")
	}
}

func TestResumerNeTriePasLEntree(t *testing.T) {
	v := []time.Duration{ms10(3), ms10(1), ms10(2)}
	resumer(v)
	if v[0] != ms10(3) || v[1] != ms10(1) {
		t.Errorf("l'entree a ete triee : %v", v)
	}
}

func TestResumerPassesCouvreChaquePhase(t *testing.T) {
	passes := []Passe{
		{Decompression: ms10(1), Chargement: ms10(2), Decodage: ms10(10), Publication: ms10(3), Total: ms10(15)},
		{Decompression: ms10(3), Chargement: ms10(4), Decodage: ms10(30), Publication: ms10(5), Total: ms10(39)},
	}
	st := resumerPasses(passes)
	attendu := map[Phase]time.Duration{
		PhaseDecompression: ms10(2), PhaseChargement: ms10(3), PhaseDecodage: ms10(20),
		PhasePublication: ms10(4), PhaseTotal: ms10(27),
	}
	for ph, med := range attendu {
		if st[ph].Mediane != med || st[ph].Passes != 2 {
			t.Errorf("phase %s : %+v, mediane attendue %v", ph, st[ph], med)
		}
	}
	if len(st) != len(ordreDesPhases) {
		t.Errorf("%d phases rendues, %d attendues", len(st), len(ordreDesPhases))
	}
}
