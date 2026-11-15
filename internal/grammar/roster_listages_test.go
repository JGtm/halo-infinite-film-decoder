package grammar

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// listagesDe rejoue des paquets (instant, entrees) dans le suivi et rend les plages.
func listagesDe(paquets []struct {
	ts      uint64
	entrees []RosterEntry
}) []types.RosterListing {
	s := &suiviDesListages{ouvertes: map[cleDeListage]int{}}
	for _, p := range paquets {
		s.listage(p.ts, p.entrees)
	}
	return s.plages()
}

func entree(xuid uint64, slot int) RosterEntry { return RosterEntry{Identite: xuid, Option: slot} }

type paquetDeTest = struct {
	ts      uint64
	entrees []RosterEntry
}

// TestListagesArriveeDepartEtRetour : un joueur du premier paquet est la des le debut ; un arrivant
// ouvre une plage au premier paquet qui le nomme ; un partant ferme la sienne au premier paquet qui ne
// le nomme plus ; un retour ouvre une seconde plage.
func TestListagesArriveeDepartEtRetour(t *testing.T) {
	got := listagesDe([]paquetDeTest{
		{10, []RosterEntry{entree(1, 0), entree(2, 1)}},
		{50, []RosterEntry{entree(1, 0), entree(2, 1), entree(3, 2)}},
		{80, []RosterEntry{entree(1, 0), entree(3, 2)}},
		{120, []RosterEntry{entree(1, 0), entree(2, 1), entree(3, 2)}},
	})
	want := []types.RosterListing{
		{XUID: 1, Slot: 0, FromUS: 10},
		{XUID: 2, Slot: 1, FromUS: 10, ToUS: 80},
		{XUID: 3, Slot: 2, FromUS: 50},
		{XUID: 2, Slot: 1, FromUS: 120},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("plages = %+v, attendu %+v", got, want)
	}
}

// TestListagesUnSiegeRepris : un arrivant qui prend le siege d un partant ouvre une plage a lui, la
// plage du partant se ferme au meme paquet ; un humain qui change de siege ouvre une nouvelle plage.
func TestListagesUnSiegeRepris(t *testing.T) {
	got := listagesDe([]paquetDeTest{
		{10, []RosterEntry{entree(1, 0), entree(2, 1)}},
		{60, []RosterEntry{entree(1, 0), entree(9, 1)}},
		{90, []RosterEntry{entree(1, 4), entree(9, 1)}},
	})
	want := []types.RosterListing{
		{XUID: 1, Slot: 0, FromUS: 10, ToUS: 90},
		{XUID: 2, Slot: 1, FromUS: 10, ToUS: 60},
		{XUID: 9, Slot: 1, FromUS: 60},
		{XUID: 1, Slot: 4, FromUS: 90},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("plages = %+v, attendu %+v", got, want)
	}
}

// TestListagesIdentiquesNeFontRien : deux paquets qui listent les memes joueurs ne ferment rien.
func TestListagesIdentiquesNeFontRien(t *testing.T) {
	got := listagesDe([]paquetDeTest{
		{10, []RosterEntry{entree(1, 0)}},
		{20, []RosterEntry{entree(1, 0)}},
	})
	if want := []types.RosterListing{{XUID: 1, Slot: 0, FromUS: 10}}; !slices.Equal(got, want) {
		t.Fatalf("plages = %+v, attendu %+v", got, want)
	}
}
