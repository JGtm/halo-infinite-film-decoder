package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestListagesEtControlesFontLAllerRetour : les deux faits se relisent a l identique, listes vides
// comprises (nil), et le flux est entierement consomme.
func TestListagesEtControlesFontLAllerRetour(t *testing.T) {
	ls := []types.RosterListing{
		{XUID: 2533491259474651, Slot: 4, FromUS: 100},
		{XUID: 2534026679052315, Slot: -1, FromUS: 100, ToUS: 90_000_000},
		{XUID: 2533491259474651, Slot: 4, FromUS: 4_000_000, ToUS: 4_000_001},
	}
	lst := types.RosterListingStats{Scanned: true, Packets: 9, Distinct: 3, Refused: 1, Entries: 12}
	cs := []types.UnitControlRead{
		{Slot: 513, TimestampUS: 10, Chunk: 1, PacketIndex: 2, Controller: 7},
		{Slot: 514, TimestampUS: 10, Chunk: 1, PacketIndex: 3, Controller: -1},
	}
	cst := types.UnitControlStats{Scanned: true}
	for _, tc := range []struct {
		nom string
		ls  []types.RosterListing
		cs  []types.UnitControlRead
	}{{"pleins", ls, cs}, {"vides", nil, nil}} {
		w := &gwriter{}
		encodeListagesDeRoster(w, tc.ls, lst)
		encodeControlesDeBipede(w, tc.cs, cst)
		r := &greader{b: w.b}
		gl, gls := decodeListagesDeRoster(r)
		gc, gcs := decodeControlesDeBipede(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("%s : relecture invalide (err %v, %d/%d octets)", tc.nom, r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(gl, tc.ls) || gls != lst || !reflect.DeepEqual(gc, tc.cs) || gcs != cst {
			t.Errorf("%s : aller-retour different\n listages %+v / %+v\n controles %+v / %+v", tc.nom, gl, tc.ls, gc, tc.cs)
		}
	}
}
