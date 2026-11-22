package replay

import "testing"

// TestFrozenRoundsCoverage : la couverture des valeurs figees compte les manches lues, les manches attendues (les
// manches retenues moins la derniere) et les joueurs sans manche figee ; elle est absente sur un match a une manche.
func TestFrozenRoundsCoverage(t *testing.T) {
	players := []PlayerScore{
		{XUID: "1", FrozenRounds: []FrozenRound{{Round: 0}, {Round: 1}}},
		{XUID: "2", FrozenRounds: []FrozenRound{{Round: 0}}},
		{XUID: "3"},
	}
	if cov := frozenRoundsCoverage(players, 1, 0); cov != nil {
		t.Errorf("couverture sur un match a une manche : %+v", cov)
	}
	got := frozenRoundsCoverage(players, 3, 2)
	if want := (FrozenRoundsCoverage{Read: 2, Expected: 2, PlayersWithout: 1, Discarded: 2}); got == nil || *got != want {
		t.Errorf("couverture %+v, attendu %+v", got, want)
	}
}

// TestAttachFrozenRounds : chaque joueur publie recoit ses manches figees, et seulement les siennes.
func TestAttachFrozenRounds(t *testing.T) {
	players := []PlayerScore{{XUID: "1"}, {XUID: "2"}}
	attachFrozenRounds(players, map[string][]FrozenRound{"2": {{Round: 0, Values: map[string]int{"kills": 4}}}})
	if players[0].FrozenRounds != nil || len(players[1].FrozenRounds) != 1 || players[1].FrozenRounds[0].Values["kills"] != 4 {
		t.Errorf("manches figees posees : %+v", players)
	}
}
