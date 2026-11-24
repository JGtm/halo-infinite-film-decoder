package replay

import (
	"context"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// lectureDeGestes est le jeu d entrees des tests des gestes : un coup de mêlée et une amorce publiables, un
// message sans joueur, un avant l origine, un après la dernière frame, un geste inconnu.
func lectureDeGestes() ([]types.GestureRead, axeDeFrames) {
	axe := axeDeFrames{origin: 1_000_000, step: 40_000, frames: 100}
	return []types.GestureRead{
		{TimestampUS: 1_000_000 + 40_000*7 + 12_500, Player: 5, Kind: types.GestureMelee},
		{TimestampUS: 1_000_000 + 40_000*7, Player: 2, Kind: types.GestureThrowStart},
		{TimestampUS: 1_000_000 + 40_000*3, Player: types.GestureNoPlayer, Kind: types.GestureMelee},
		{TimestampUS: 999_999, Player: 1, Kind: types.GestureThrowStart},
		{TimestampUS: 1_000_000 + 40_000*100, Player: 1, Kind: types.GestureMelee},
		{TimestampUS: 1_000_000 + 40_000*99, Player: 31, Kind: types.GestureMelee},
		{TimestampUS: 1_000_000 + 40_000*50, Player: 4, Kind: types.GestureKind(9)},
	}, axe
}

// TestLesGestesSePosentSurLAxeEtSeComptent : un geste dans l axe est publié à sa frame et à sa milliseconde, sous
// l index de joueur du message et son genre ; un message sans index, avant l origine ou après la dernière frame
// est écarté et compté ; les copies du canal entrent dans le lu ; l identité de la couverture tient.
// MUTATIONS : une borne d axe décalée, un message sans joueur publié, la milliseconde prise depuis le début du
// film, les deux genres échangés, une cause écartée non comptée -> ROUGE.
func TestLesGestesSePosentSurLAxeEtSeComptent(t *testing.T) {
	lus, axe := lectureDeGestes()
	in := GestureInput{Reads: lus, Stats: types.GestureStats{Frames: 50, Closed: 40, Stopped: 10,
		Melee: types.GestureCounts{Messages: 4, Copies: 2}, Throw: types.GestureCounts{Messages: 2, Copies: 1}}}
	got, cov := buildGestures(in, axe, true)
	want := []Gesture{
		{T: 7, Ms: 280, Place: 2, Kind: GestureKindThrowStart},
		{T: 7, Ms: 292, Place: 5, Kind: GestureKindMelee},
		{T: 99, Ms: 3960, Place: 31, Kind: GestureKindMelee},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gestes %+v, attendu %+v", got, want)
	}
	wantCov := GestureCoverage{Frames: 50, ClosedFrames: 40, StoppedFrames: 10, Read: 9, Copies: 3, WithoutPlayer: 1,
		BeforeOrigin: 1, AfterEnd: 1, Published: 3}
	if cov != wantCov {
		t.Errorf("couverture %+v, attendu %+v", cov, wantCov)
	}
	if cov.Read != cov.Copies+cov.Gated+cov.WithoutPlayer+cov.BeforeOrigin+cov.AfterEnd+cov.Published {
		t.Errorf("l identité de la couverture ne tient pas : %+v", cov)
	}
}

// TestLaPorteDeCadenceRetientLesGestes : porte fermée, aucun geste ne se publie, tous ceux que le canal rend se
// comptent en `gated` et l identité de la couverture tient ; la porte ouverte publie.
// MUTATIONS : la porte ignorée, un geste retenu non compté -> ROUGE.
func TestLaPorteDeCadenceRetientLesGestes(t *testing.T) {
	lus, axe := lectureDeGestes()
	in := GestureInput{Reads: lus, Stats: types.GestureStats{Frames: 1, Melee: types.GestureCounts{Messages: 4, Copies: 0},
		Throw: types.GestureCounts{Messages: 2}}}
	got, cov := buildGestures(in, axe, false)
	if len(got) != 0 || cov.Published != 0 {
		t.Errorf("porte fermée : %d geste(s) publié(s)", len(got))
	}
	if cov.Gated != 6 || cov.Read != 6 {
		t.Errorf("porte fermée : couverture %+v, attendu 6 retenus sur 6 lus", cov)
	}
	if cov.Read != cov.Copies+cov.Gated+cov.WithoutPlayer+cov.BeforeOrigin+cov.AfterEnd+cov.Published {
		t.Errorf("l identité de la couverture ne tient pas : %+v", cov)
	}
}

// TestLaPoseDesGestesSuitLaPorteEtLesTrames : sans trame lue le calque et sa couverture sont absents ; avec
// des trames, la porte ouverte (le pas des entrées de contrôle mesuré à pleine cadence) publie et la porte
// fermée (pas non mesuré ou lent) ne publie que la couverture.
// MUTATIONS : la pose sans garde de trames, la porte non lue sur les statistiques des commandes -> ROUGE.
func TestLaPoseDesGestesSuitLaPorteEtLesTrames(t *testing.T) {
	lus, axe := lectureDeGestes()
	pleine := types.ControlHoldStats{StepGaps: types.EcartsDuPasMin, StepUS: 16_650}
	lente := types.ControlHoldStats{StepGaps: types.EcartsDuPasMin, StepUS: 33_364}
	poser := func(in GestureInput, commandes types.ControlHoldStats) *assemblage {
		a := &assemblage{ctx: context.Background(), origin: axe.origin, step: axe.step}
		a.doc.FrameCount = axe.frames
		a.doc.Coverage = &Coverage{}
		a.opt.Gestures = in
		a.opt.ControlHoldStats = commandes
		a.poserLesGestes()
		return a
	}
	if a := poser(GestureInput{Reads: lus}, pleine); a.doc.Gestures != nil || a.doc.Coverage.Gestures != nil {
		t.Error("aucune trame lue : le calque et sa couverture doivent être absents")
	}
	stats := types.GestureStats{Frames: 10}
	if a := poser(GestureInput{Reads: lus, Stats: stats}, pleine); len(a.doc.Gestures) != 3 || a.doc.Coverage.Gestures.Gated != 0 {
		t.Errorf("porte ouverte : %d geste(s), couverture %+v", len(a.doc.Gestures), a.doc.Coverage.Gestures)
	}
	for nom, commandes := range map[string]types.ControlHoldStats{"pas lent": lente, "pas non mesuré": {}} {
		a := poser(GestureInput{Reads: lus, Stats: stats}, commandes)
		if len(a.doc.Gestures) != 0 || a.doc.Coverage.Gestures == nil || a.doc.Coverage.Gestures.Gated != 6 {
			t.Errorf("porte fermée (%s) : %d geste(s), couverture %+v", nom, len(a.doc.Gestures), a.doc.Coverage.Gestures)
		}
	}
}

// TestLesGestesFontLAllerRetourDesFaits : les comptes du canal et chaque geste (horodatage, index de joueur dont
// l absent, genre) se relisent à l identique ; une liste vide se relit nil.
func TestLesGestesFontLAllerRetourDesFaits(t *testing.T) {
	cas := []struct {
		gs []types.GestureRead
		st types.GestureStats
	}{
		{nil, types.GestureStats{}},
		{nil, types.GestureStats{Frames: 3, Closed: 3}},
		{[]types.GestureRead{
			{TimestampUS: 4_517_903_000, Player: 4, Kind: types.GestureMelee},
			{TimestampUS: 4_517_903_000, Player: 31, Kind: types.GestureThrowStart},
			{TimestampUS: 4_517_999_000, Player: types.GestureNoPlayer, Kind: types.GestureMelee},
		}, types.GestureStats{Frames: 9000, Closed: 8990, Stopped: 10,
			Melee: types.GestureCounts{Messages: 2, Copies: 1}, Throw: types.GestureCounts{Messages: 1, Copies: 4}}},
	}
	for _, c := range cas {
		w := &gwriter{}
		encodeGestes(w, c.gs, c.st)
		r := &greader{b: w.b}
		gs, st := decodeGestes(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(gs, c.gs) || st != c.st {
			t.Errorf("relu %+v %+v, attendu %+v %+v", gs, st, c.gs, c.st)
		}
	}
}
