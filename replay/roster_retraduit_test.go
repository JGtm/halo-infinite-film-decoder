package replay

import (
	"slices"
	"testing"
)

// TestRetraduireLeRosterRendLesOriginaux : un fictif que la table connait redevient son original, un
// xuid inconnu reste tel quel, et l ordre de la feuille est garde.
func TestRetraduireLeRosterRendLesOriginaux(t *testing.T) {
	table := map[uint64]uint64{9990000000000001: 9990000000000101, 9990000000000002: 9990000000000102}
	got := retraduireLeRoster([]uint64{9990000000000102, 9990000000000999, 9990000000000101}, table)
	want := []uint64{9990000000000002, 9990000000000999, 9990000000000001}
	if !slices.Equal(got, want) {
		t.Fatalf("roster retraduit %v, attendu %v", got, want)
	}
}
