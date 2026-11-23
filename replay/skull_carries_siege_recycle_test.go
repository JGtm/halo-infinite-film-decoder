package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// skull_carries_siege_recycle_test.go — UN SIEGE QUI CHANGE D'OCCUPANT RENDRA LES TICS DES DEUX JOUEURS : le
// partant garde ses trois tics, l'arrivant ses deux, chacun nomme par l'occupant de son instant.
//
// ROUGE OBSERVE AVANT LA COUPURE : les deux suites (1, 2, 3 puis 0, 1, 2) ont meme longueur, la plus longue suite
// du siege garde la derniere et les trois tics du partant disparaissaient.

// siegeRecycleDuCrane : le siege 12 est tenu par "A" (trois tics, trois morts) jusqu'a la retombee de ses
// compteurs a 30 000 ms, puis par "B" (deux tics, trois morts).
func siegeRecycleDuCrane() ([]types.StatRecord, []types.DeathInstant) {
	tick := func(t int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: 12, Comps: map[int]types.StatValue{0: {A: v}}}
	}
	death := func(t int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: 12, Comps: map[int]types.StatValue{2: {B: v}}}
	}
	recs := []types.StatRecord{
		tick(1000, 1), tick(2000, 2), tick(3000, 3),
		death(4500, 1), death(4600, 2), death(4700, 3),
		// La retombee : tous les compteurs du siege repartent de zero, le partant a quitte.
		{TimeMS: 30000, Slot: 12, Comps: map[int]types.StatValue{0: {A: 0}, 2: {A: 0, B: 0}, 3: {A: 0}}},
		tick(40000, 1), tick(41000, 2),
		death(42500, 1), death(42600, 2), death(42700, 3),
	}
	deaths := []types.DeathInstant{
		{XUID: "A", TimeMS: 4500}, {XUID: "A", TimeMS: 4600}, {XUID: "A", TimeMS: 4700},
		{XUID: "B", TimeMS: 42500}, {XUID: "B", TimeMS: 42600}, {XUID: "B", TimeMS: 42700},
	}
	return recs, deaths
}

func TestSkullCarriesSiegeRecycleRendLesTicsDesDeuxOccupants(t *testing.T) {
	recs, deaths := siegeRecycleDuCrane()
	identity := objectives.ResolveRoundIdentity(recs, deaths, nil)
	if occ := identity.Occupations(0, 12); len(occ) != 2 {
		t.Fatalf("occupations du siege 12 : %+v, attendu deux", occ)
	}
	got := skullCarryIntervals(recs, identity, nil)
	want := []skullRawCarry{
		{xuid: "A", round: 0, t0MS: 1000, t1MS: 3000},
		{xuid: "B", round: 0, t0MS: 40000, t1MS: 41000},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("portages = %+v, attendu %+v (un par occupant)", got, want)
	}
}

// TestSkullCarriesSiegeRecycleFermeLeTrainAuChangement : le partant garde le crane jusqu'a son depart et
// l'arrivant le ramasse sans trou de tics (moins de trois secondes) : un seul train serait attribue a un joueur
// pour les tics de l'autre. Le train se ferme au changement d'occupant.
func TestSkullCarriesSiegeRecycleFermeLeTrainAuChangement(t *testing.T) {
	tick := func(t int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: 12, Comps: map[int]types.StatValue{0: {A: v}}}
	}
	death := func(tMS int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: tMS, Slot: 12, Comps: map[int]types.StatValue{2: {B: v}}}
	}
	recs := []types.StatRecord{
		death(500, 1), death(600, 2), death(700, 3),
		tick(1000, 1), tick(2000, 2), tick(3000, 3),
		{TimeMS: 3400, Slot: 12, Comps: map[int]types.StatValue{0: {A: 0}, 2: {A: 0, B: 0}, 3: {A: 0}}},
		tick(4000, 1), tick(5000, 2),
		death(6500, 1), death(6600, 2), death(6700, 3),
	}
	deaths := []types.DeathInstant{
		{XUID: "A", TimeMS: 500}, {XUID: "A", TimeMS: 600}, {XUID: "A", TimeMS: 700},
		{XUID: "B", TimeMS: 6500}, {XUID: "B", TimeMS: 6600}, {XUID: "B", TimeMS: 6700},
	}
	identity := objectives.ResolveRoundIdentity(recs, deaths, nil)
	got := skullCarryIntervals(recs, identity, nil)
	want := []skullRawCarry{
		{xuid: "A", round: 0, t0MS: 1000, t1MS: 3000},
		{xuid: "B", round: 0, t0MS: 4000, t1MS: 5000},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("portages = %+v, attendu %+v", got, want)
	}
}
