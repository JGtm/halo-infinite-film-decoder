package replay

import (
	"cmp"
	"reflect"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// pistes_partagees_test.go : l'index par slot partage par l'assemblage rend, pour chaque slot, la
// MEME suite de positions que la construction d'origine, gardee ici comme reference.

// indexParSlotReference est la construction d'origine de [indexBySlot] : une copie des positions
// de chaque slot, triee par instant, les ex aequo gardant l'ordre du film.
func indexParSlotReference(pos []grammar.BipedPosition) map[uint32][]grammar.BipedPosition {
	m := map[uint32][]grammar.BipedPosition{}
	for _, p := range pos {
		m[p.Slot] = append(m[p.Slot], p)
	}
	for s := range m {
		slices.SortStableFunc(m[s], func(a, b grammar.BipedPosition) int {
			return cmp.Compare(a.TimestampUS, b.TimestampUS)
		})
	}
	return m
}

// positionsAvecEgalites est un nuage desordonne, a plusieurs slots, avec des ex aequo
// distinguables par leur abscisse.
func positionsAvecEgalites() []grammar.BipedPosition {
	return []grammar.BipedPosition{
		posAt(512, 3_000_000, 1, 0, 0), posAt(513, 1_000_000, 2, 0, 0), posAt(512, 1_000_000, 3, 0, 0),
		posAt(512, 1_000_000, 4, 0, 0), posAt(513, 1_000_000, 5, 0, 0), posAt(512, 2_000_000, 6, 0, 0),
		posAt(514, 5_000_000, 7, 0, 0),
	}
}

func verifierIndexContreReference(t *testing.T, nom string, pos []grammar.BipedPosition,
	tracks map[uint32]slotTrack) {
	t.Helper()
	ref := indexParSlotReference(pos)
	if len(tracks) != len(ref) {
		t.Fatalf("%s : %d slots indexes, la reference en compte %d", nom, len(tracks), len(ref))
	}
	for s, want := range ref {
		if got := tracks[s].pts(); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s : la suite du slot %d differe de la reference (%d positions contre %d)",
				nom, s, len(got), len(want))
		}
	}
}

func TestIndexBySlotRendLaSuiteDeReference(t *testing.T) {
	pos := positionsAvecEgalites()
	verifierIndexContreReference(t, "nuage synthetique", pos, indexBySlot(pos))
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		verifierIndexContreReference(t, b.Short8, g.Positions, indexBySlot(g.Positions))
	}
}

// TestPistesPartageesParLAssemblage : l'index de `a.sorted` est construit une fois, repris tel quel
// tant que la tranche ne change pas, rebati quand elle change, et reste egal a la reference.
func TestPistesPartageesParLAssemblage(t *testing.T) {
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		a := &assemblage{sorted: append([]grammar.BipedPosition(nil), g.Positions...)}
		slices.SortStableFunc(a.sorted, func(p, q grammar.BipedPosition) int {
			return cmp.Compare(p.TimestampUS, q.TimestampUS)
		})
		premier := a.pistes()
		verifierIndexContreReference(t, b.Short8, a.sorted, premier)
		if reflect.ValueOf(a.pistes()).Pointer() != reflect.ValueOf(premier).Pointer() {
			t.Fatalf("%s : l'index est reconstruit alors que la tranche n'a pas change", b.Short8)
		}
		if len(a.sorted) < 2 {
			continue
		}
		// L'ecart d'un slot remplace la tranche (cf. construireLeRegistre) : l'index la suit.
		ecartes := map[uint32]struct{}{a.sorted[0].Slot: {}}
		a.sorted, _ = sansLesSlots(a.sorted, ecartes)
		verifierIndexContreReference(t, b.Short8+" apres ecart", a.sorted, a.pistes())
	}
}

// TestPistesNeModifientPasLeNuage : indexer ne trie ni ne modifie la tranche d'origine, qui est
// aussi celle que les autres passes lisent.
func TestPistesNeModifientPasLeNuage(t *testing.T) {
	pos := positionsAvecEgalites()
	avant := append([]grammar.BipedPosition(nil), pos...)
	_ = indexBySlot(pos)
	if !reflect.DeepEqual(pos, avant) {
		t.Fatal("indexBySlot a modifie le nuage d'origine")
	}
}

// pts materialise la piste en tranche : les mesures de recherche et les references de test la
// lisent comme une suite de positions. La production ne le fait jamais (elle lit par rang).
func (s slotTrack) pts() []grammar.BipedPosition {
	out := make([]grammar.BipedPosition, s.n())
	for i := range out {
		out[i] = s.pt(i)
	}
	return out
}
