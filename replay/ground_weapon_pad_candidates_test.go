package replay

// ground_weapon_pad_candidates_test.go — LES CANDIDATS DE SOCLE, sur des entrees SYNTHETIQUES.
//
// Trois choses sont verrouillees : une grappe a UNE apparition au repos sort en candidat et pas en
// socle ; la chaine qui rend les candidats rend, pour les socles, EXACTEMENT ce que rendait
// `buildWeaponPads` avant eux ; la couverture ne compte pas un candidat comme un socle.

import (
	"reflect"
	"testing"
)

// gwTestUneApparition rend la voie des armes reduite a sa premiere apparition : une grappe d'une
// seule apparition, au repos, jamais reprise.
func gwTestUneApparition(t *testing.T) (PadScans, PadScans) {
	t.Helper()
	scan, _ := gwTestPadScan(t)
	scan.Creations = scan.Creations[:1]
	scan.Stats.Accepted = 1
	pu := puTestScan(puTestOvershield, 10, 10)
	pu.Creations, pu.Stats.Accepted = pu.Creations[:1], 1
	return PadScans{Weapons: scan}, PadScans{Powerups: pu}
}

func TestPadCandidatsGardeLApparitionUniqueDesArmes(t *testing.T) {
	armes, _ := gwTestUneApparition(t)
	c := buildPadsAndCandidates(armes, gwTestPositions(nil, 10, 10), gwTestClock(), padCatalogs{})
	if len(c.Pads) != 0 || len(c.Picks) != 0 {
		t.Fatalf("aucun socle ni occupation attendus : %+v %+v", c.Pads, c.Picks)
	}
	if len(c.Candidates) != 1 {
		t.Fatalf("1 candidat attendu : %+v", c.Candidates)
	}
	cand := c.Candidates[0]
	if len(cand.Spawns) != 1 || len(cand.Presence) != 1 || cand.Cycle != nil {
		t.Errorf("un candidat a une apparition, une presence et aucun cycle : %+v", cand)
	}
	if cand.Weapon == "" || cand.X < 9 || cand.X > 11 {
		t.Errorf("le candidat porte la famille et la position de son apparition : %+v", cand)
	}
	if c.Cov.Candidates != 1 || c.Cov.PowerupCandidates != 0 {
		t.Errorf("couverture : candidats armes=%d power-ups=%d, attendu 1/0",
			c.Cov.Candidates, c.Cov.PowerupCandidates)
	}
	if c.Cov.Pads != 0 || c.Cov.Occupancies != 0 || !c.Cov.Balanced() {
		t.Errorf("la couverture compte le candidat comme un socle : %+v", c.Cov)
	}
}

func TestPadCandidatsGardeLApparitionUniqueDesPowerups(t *testing.T) {
	_, pu := gwTestUneApparition(t)
	c := buildPadsAndCandidates(pu, gwTestPositions(nil, 0, 0), gwTestClock(), puTestCatalogs())
	if len(c.Pads) != 0 || len(c.Candidates) != 1 {
		t.Fatalf("0 socle et 1 candidat attendus : %+v / %+v", c.Pads, c.Candidates)
	}
	if c.Candidates[0].Weapon != "powerup_overshield" {
		t.Errorf("le candidat de la voie `ti=37` porte la famille du manifeste : %+v", c.Candidates[0])
	}
	if c.Cov.PowerupCandidates != 1 || c.Cov.Candidates != 0 || c.Cov.PowerupPads != 0 {
		t.Errorf("couverture : %+v", c.Cov)
	}
}

// TestPadCandidatsUnSocleRecurrentNEnSortePas : une grappe qui recurre est un socle et n'est pas un
// candidat ; les deux ensembles ne se recouvrent jamais.
func TestPadCandidatsUnSocleRecurrentNEnSortePas(t *testing.T) {
	scan, pos := gwTestPadScan(t)
	c := buildPadsAndCandidates(PadScans{Weapons: scan}, pos, gwTestClock(), padCatalogs{})
	if len(c.Pads) != 1 || len(c.Candidates) != 0 {
		t.Fatalf("1 socle et 0 candidat attendus : %d / %d", len(c.Pads), len(c.Candidates))
	}
}

// TestPadCandidatsLaChaineDesSoclesNeChangePas : avec ou sans candidat dans le film, les socles, les
// occupations et leurs compteurs sont ceux que rendait `buildWeaponPads` seul.
func TestPadCandidatsLaChaineDesSoclesNeChangePas(t *testing.T) {
	scan, pos := gwTestPadScan(t)
	// Une cinquieme creation, isolee, d'une autre famille : un candidat de plus dans le film.
	solo := gwTestCreation(14, 0, 5_000_000, gwTestFamily(t, 1), 50, 50)
	scan.Creations = append(scan.Creations, solo)
	scan.Stats.Accepted++
	seul := PadScans{Weapons: scan}
	pads, picks, cov, objs := buildWeaponPads(seul, pos, gwTestClock(), padCatalogs{})
	c := buildPadsAndCandidates(seul, pos, gwTestClock(), padCatalogs{})
	if !reflect.DeepEqual(pads, c.Pads) || !reflect.DeepEqual(picks, c.Picks) ||
		!reflect.DeepEqual(objs, c.Objs) || !reflect.DeepEqual(cov, c.Cov) {
		t.Fatalf("buildWeaponPads n'est plus la projection de la chaine complete")
	}
	if len(pads) != 1 || len(c.Candidates) != 1 {
		t.Fatalf("1 socle et 1 candidat attendus : %d / %d", len(pads), len(c.Candidates))
	}
	// Retirer la creation isolee ne change aucun socle : le candidat ne prete rien aux socles.
	sans, _ := gwTestPadScan(t)
	pads2, picks2, _, _ := buildWeaponPads(PadScans{Weapons: sans}, pos, gwTestClock(), padCatalogs{})
	if !reflect.DeepEqual(pads, pads2) || !reflect.DeepEqual(picks, picks2) {
		t.Errorf("la presence d'un candidat change les socles publies")
	}
}

// TestPadCandidatsArmesAvantPowerups : l'ordre est celui des socles, armes d'abord puis power-ups.
func TestPadCandidatsArmesAvantPowerups(t *testing.T) {
	armes, pu := gwTestUneApparition(t)
	c := buildPadsAndCandidates(PadScans{Weapons: armes.Weapons, Powerups: pu.Powerups},
		gwTestPositions(nil, 10, 10), gwTestClock(), puTestCatalogs())
	if len(c.Candidates) != 2 || c.Candidates[1].Weapon != "powerup_overshield" ||
		c.Candidates[0].Weapon == "powerup_overshield" {
		t.Fatalf("armes puis power-ups attendus : %+v", c.Candidates)
	}
}

// TestPadCandidatsSansHorlogeNeRendRien : sans axe de temps, aucune voie ne se lit, donc aucun
// candidat — comme aucun socle.
func TestPadCandidatsSansHorlogeNeRendRien(t *testing.T) {
	armes, _ := gwTestUneApparition(t)
	c := buildPadsAndCandidates(armes, nil, replayClock{}, padCatalogs{})
	if c.Candidates != nil || c.Pads != nil || c.Cov == nil {
		t.Fatalf("rien attendu sans horloge : %+v", c)
	}
}

// TestGwPadsSetAsideEstLeComplementDeKeep : toute grappe est dans exactement l'un des deux jeux, et
// les index d'origine se partitionnent.
func TestGwPadsSetAsideEstLeComplementDeKeep(t *testing.T) {
	in := []gwPadCluster{
		{TS: []uint64{1}}, {TS: []uint64{1, 2}}, {TS: []uint64{1, 2, 3}}, {TS: nil}, {TS: []uint64{9}},
	}
	keep, keepSrc := gwPadsKeep(in)
	aside, asideSrc := gwPadsSetAside(in)
	if len(keep)+len(aside) != len(in) {
		t.Fatalf("%d + %d != %d", len(keep), len(aside), len(in))
	}
	seen := map[int]bool{}
	for _, i := range append(append([]int(nil), keepSrc...), asideSrc...) {
		if seen[i] {
			t.Fatalf("index %d dans les deux jeux", i)
		}
		seen[i] = true
	}
	if !reflect.DeepEqual(keepSrc, []int{1, 2}) || !reflect.DeepEqual(asideSrc, []int{0, 3, 4}) {
		t.Errorf("index d'origine : socles %v, candidats %v", keepSrc, asideSrc)
	}
}
