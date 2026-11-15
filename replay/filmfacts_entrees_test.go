package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestEntreesDuStatborgFontLAllerRetour : les entrees se relisent a l'identique, listes vides comprises (nil), et le
// flux est entierement consomme.
func TestEntreesDuStatborgFontLAllerRetour(t *testing.T) {
	es := []types.StatborgEntry{
		{Slot: 3, TS: 4_517_903_407, Kind: types.StatborgEntryTeam, ID: 0},
		{Slot: 5, TS: 4_537_898_226, Kind: types.StatborgEntryPlayer, ID: 0},
		{Slot: 12, TS: 4_537_898_226, Kind: types.StatborgEntryPlayer, ID: 31},
		{Slot: 5, TS: 4_557_001_000, Kind: types.StatborgEntryPlayer, ID: 9},
	}
	st := types.StatborgEntryStats{Scanned: true, Records: 1248, Read: 4, Vacant: 1244, DecidedRounds: 0b101}
	for _, tc := range []struct {
		nom string
		es  []types.StatborgEntry
	}{{"pleines", es}, {"vides", nil}} {
		w := &gwriter{}
		encodeEntreesDuStatborg(w, tc.es, st)
		r := &greader{b: w.b}
		got, gst := decodeEntreesDuStatborg(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("%s : relecture invalide (err %v, %d/%d octets)", tc.nom, r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(got, tc.es) || gst != st {
			t.Errorf("%s : aller-retour different\n %+v / %+v", tc.nom, got, tc.es)
		}
	}
}
