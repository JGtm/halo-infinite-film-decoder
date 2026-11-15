package objectives

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func TestSplitUnplayedRounds(t *testing.T) {
	recs := []types.StatRecord{{Round: 0}, {Round: 1}, {Round: 2}, {Round: 7}, {Round: 2}}
	cas := []struct {
		nom      string
		decided  uint32
		kept     int
		unplayed []int
	}{
		{"aucune issue : la manche 0 seule, plus la manche en cours", 0, 1, []int{1, 2, 7}},
		{"manche 0 jouee : la suivante est en cours", 0b1, 2, []int{2, 7}},
		{"trois manches jouees", 0b111, 4, []int{7}},
		{"issue de la manche 7 : tout est joue", 1 << 7, 5, nil},
		{"issues sautees : le dernier rang prime", 0b101, 4, []int{7}},
	}
	for _, c := range cas {
		kept, unplayed := SplitUnplayedRounds(recs, c.decided)
		if len(kept) != c.kept || !slices.Equal(unplayed, c.unplayed) {
			t.Errorf("%s : %d garde(s), %v non jouee(s) ; attendu %d, %v", c.nom, len(kept), unplayed, c.kept, c.unplayed)
		}
	}
	if k, u := SplitUnplayedRounds(nil, 0); len(k) != 0 || u != nil {
		t.Errorf("aucun enregistrement : %v %v", k, u)
	}
}
