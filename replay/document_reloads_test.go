package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLesRechargementsSePosentSurLAxeEtSeComptent : un rechargement dans l axe est publie a sa frame, sous
// l index de joueur du message ; un message sans index, avant l origine ou apres la derniere frame est ecarte
// et compte ; l identite de la couverture tient. MUTATIONS : une borne d axe decalee, un message sans joueur
// publie, une cause ecartee non comptee -> ROUGE.
func TestLesRechargementsSePosentSurLAxeEtSeComptent(t *testing.T) {
	axe := axeDeFrames{origin: 1_000_000, step: 40_000, frames: 100}
	st := types.ReloadStats{Frames: 50, Closed: 40, Stopped: 10, Messages: 6, Copies: 3}
	in := []types.ReloadRead{
		{TimestampUS: 1_000_000 + 40_000*7, Player: 5},
		{TimestampUS: 1_000_000 + 40_000*7 + 39_999, Player: 2, Copies: 2},
		{TimestampUS: 1_000_000 + 40_000*3, Player: types.ReloadNoPlayer},
		{TimestampUS: 999_999, Player: 1},
		{TimestampUS: 1_000_000 + 40_000*100, Player: 1},
		{TimestampUS: 1_000_000 + 40_000*99, Player: 31, Copies: 1},
	}
	got, cov := buildReloads(in, st, axe)
	want := []Reload{{T: 7, Place: 2, Ms: 319, Copies: 2}, {T: 7, Place: 5, Ms: 280}, {T: 99, Place: 31, Ms: 3960, Copies: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rechargements %+v, attendu %+v", got, want)
	}
	wantCov := ReloadCoverage{Frames: 50, ClosedFrames: 40, StoppedFrames: 10, Read: 9, Copies: 3, WithoutPlayer: 1,
		BeforeOrigin: 1, AfterEnd: 1, Published: 3}
	if cov != wantCov {
		t.Errorf("couverture %+v, attendu %+v", cov, wantCov)
	}
	if cov.Read != cov.Copies+cov.WithoutPlayer+cov.BeforeOrigin+cov.AfterEnd+cov.Published {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
}

// TestLesRechargementsFontLAllerRetourDesFaits : les comptes du canal et chaque rechargement (horodatage,
// index de joueur dont l absent) se relisent a l identique ; une liste vide se relit nil.
func TestLesRechargementsFontLAllerRetourDesFaits(t *testing.T) {
	cas := []struct {
		rs []types.ReloadRead
		st types.ReloadStats
	}{
		{nil, types.ReloadStats{}},
		{nil, types.ReloadStats{Frames: 3, Closed: 3}},
		{[]types.ReloadRead{
			{TimestampUS: 4_517_903_000, Player: 4, Copies: 1}, {TimestampUS: 4_517_903_000, Player: 31},
			{TimestampUS: 4_517_999_000, Player: types.ReloadNoPlayer, Copies: 7},
		}, types.ReloadStats{Frames: 9000, Closed: 8990, Stopped: 10, Messages: 3, Copies: 1}},
	}
	for _, c := range cas {
		w := &gwriter{}
		encodeRechargements(w, c.rs, c.st)
		r := &greader{b: w.b}
		rs, st := decodeRechargements(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(rs, c.rs) || st != c.st {
			t.Errorf("relu %+v %+v, attendu %+v %+v", rs, st, c.rs, c.st)
		}
	}
}
