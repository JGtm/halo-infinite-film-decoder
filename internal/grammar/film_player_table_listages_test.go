package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func tableDeTest(interleaved bool, seats ...FilmPlayerSeat) FilmPlayerTable {
	return FilmPlayerTable{Seats: seats, Occupied: len(seats), InterleavedVacant: interleaved}
}

func listeAuDepart(slots ...[2]uint64) []types.RosterListing {
	out := make([]types.RosterListing, 0, len(slots))
	for _, s := range slots {
		out = append(out, types.RosterListing{XUID: s[0], Slot: int(s[1]), FromUS: 1000})
	}
	return out
}

var lecturesFaites = types.RosterListingStats{Scanned: true}

// Un occupant range a la ligne 2 de la table et liste au siege 8 prend le siege 8 ; la table n'a plus de
// vacant intercale a departager.
func TestSiegesParLesListagesRemplaceLeRang(t *testing.T) {
	table := tableDeTest(true,
		FilmPlayerSeat{FilmIndex: 0, XUID: 10}, FilmPlayerSeat{FilmIndex: 1, XUID: 11},
		FilmPlayerSeat{FilmIndex: 2, XUID: 12}, FilmPlayerSeat{FilmIndex: 4, XUID: 13})
	liste := listeAuDepart([2]uint64{10, 0}, [2]uint64{11, 1}, [2]uint64{12, 8}, [2]uint64{13, 4})
	out, n := SiegesParLesListages(table, liste, lecturesFaites)
	if n != 1 || out.InterleavedVacant || !out.Lue() {
		t.Fatalf("corriges %d, intercale %v, lue %v : un siege change, la table se lit", n, out.InterleavedVacant, out.Lue())
	}
	var rangs []int
	for _, s := range out.Seats {
		rangs = append(rangs, s.FilmIndex)
	}
	if len(rangs) != 4 || rangs[0] != 0 || rangs[1] != 1 || rangs[2] != 4 || rangs[3] != 8 || out.Seats[3].XUID != 12 {
		t.Fatalf("sieges %v, attendu 0 1 4 8 avec l'occupant 12 au siege 8", out.Seats)
	}
	if table.Seats[2].FilmIndex != 2 {
		t.Fatal("la table d'entree est modifiee")
	}
}

// Tous les rangs sont deja les sieges listes : rien n'est corrige, le vacant intercale reste.
func TestSiegesParLesListagesSansEcartNeChangeRien(t *testing.T) {
	table := tableDeTest(true, FilmPlayerSeat{FilmIndex: 0, XUID: 10}, FilmPlayerSeat{FilmIndex: 2, XUID: 11})
	out, n := SiegesParLesListages(table, listeAuDepart([2]uint64{10, 0}, [2]uint64{11, 2}), lecturesFaites)
	if n != 0 || !out.InterleavedVacant {
		t.Fatalf("corriges %d, intercale %v : rien a corriger", n, out.InterleavedVacant)
	}
}

// Chaque garde laisse la table intacte : table refusee, listes non lues, occupant non liste, plage
// sans siege, occupant liste apres le premier paquet, siege liste deux fois, occupant liste sous deux sieges au premier paquet.
func TestSiegesParLesListagesGardesLaTableIntacte(t *testing.T) {
	table := tableDeTest(false, FilmPlayerSeat{FilmIndex: 0, XUID: 10}, FilmPlayerSeat{FilmIndex: 1, XUID: 11})
	ecart := listeAuDepart([2]uint64{10, 0}, [2]uint64{11, 5})
	refusee := table
	refusee.Refusal = FilmTableNotFound
	tardif := append(listeAuDepart([2]uint64{10, 0}), types.RosterListing{XUID: 11, Slot: 5, FromUS: 2000})
	sansSiege := append(listeAuDepart([2]uint64{10, 0}), types.RosterListing{XUID: 11, Slot: -1, FromUS: 1000})
	cas := map[string]struct {
		t  FilmPlayerTable
		l  []types.RosterListing
		st types.RosterListingStats
	}{
		"table refusee":         {refusee, ecart, lecturesFaites},
		"listes non lues":       {table, ecart, types.RosterListingStats{}},
		"aucune liste":          {table, nil, lecturesFaites},
		"occupant non liste":    {table, listeAuDepart([2]uint64{10, 0}), lecturesFaites},
		"plage sans siege":      {table, sansSiege, lecturesFaites},
		"liste tardive":         {table, tardif, lecturesFaites},
		"siege en double":       {table, listeAuDepart([2]uint64{10, 5}, [2]uint64{11, 5}), lecturesFaites},
		"xuid sous deux sieges": {table, listeAuDepart([2]uint64{10, 0}, [2]uint64{10, 5}, [2]uint64{11, 1}), lecturesFaites},
	}
	for nom, c := range cas {
		out, n := SiegesParLesListages(c.t, c.l, c.st)
		if n != 0 || out.Seats[1].FilmIndex != 1 || out.InterleavedVacant != c.t.InterleavedVacant {
			t.Errorf("%s : corriges %d, sieges %v : la table devait rester telle quelle", nom, n, out.Seats)
		}
	}
}
