package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entreesCroisees pose, sur le film croise de [reassignFixture] (slot 22 = A puis B, slot 20 = B puis A), les entrees et les
// plages qui disent la meme chose par lecture : l'entite 11 (slot 22) est la ligne de l'identifiant 1 (A) en manche 0, puis de
// l'identifiant 2 (B) en manche 1 ; l'entite 10 (slot 20) l'inverse.
func entreesCroisees() *objectives.EntryResolver {
	entree := func(slot, tsMS, id int) types.StatborgEntry {
		return types.StatborgEntry{Slot: slot, TS: uint64(tsMS) * 1000, Kind: types.StatborgEntryPlayer, ID: id}
	}
	plage := func(xuid uint64, id int) types.RosterListing { return types.RosterListing{XUID: xuid, Slot: id} }
	return objectives.NewEntryResolver(
		[]types.StatborgEntry{entree(11, 1_000, 1), entree(10, 1_000, 2), entree(11, 11_500, 2), entree(10, 11_500, 1)},
		[]types.RosterListing{plage(1001, 1), plage(1002, 2)}, 0)
}

// TestScoreTimelineNommeParLesEntreesUnFilmAPlusieursManches : un film a plusieurs manches dont les lignes changent de joueur
// se nomme par les entrees, publie les compteurs nommes de chaque joueur et ne prive personne.
func TestScoreTimelineNommeParLesEntreesUnFilmAPlusieursManches(t *testing.T) {
	recs, deaths := reassignFixture()
	for _, slot := range []int{20, 22} {
		for round, t0 := range []int{1_000, 11_000} {
			for i := range 3 {
				recs = append(recs, statRec(t0+i*1_000, slot, round, map[int]types.StatValue{6: {A: int64(i + 1)}}))
			}
		}
	}
	tl, cov, _ := assembleScoreTimeline(&ScoreInput{Records: recs}, deaths, entreesCroisees(), multiRoundClock(), nil)
	if tl == nil || len(tl.Players) != 2 {
		t.Fatalf("joueurs publies : %+v", tl)
	}
	a, b := playerByXUID(tl.Players, "1001"), playerByXUID(tl.Players, "1002")
	if a == nil || b == nil {
		t.Fatalf("A ou B absent : %+v", tl.Players)
	}
	if got := lastValue(a.Score); got != 560 {
		t.Errorf("score personnel de A = %d, attendu 560 (slot 22 manche 0, puis slot 20 manche 1)", got)
	}
	if got := lastValue(b.Score); got != 370 {
		t.Errorf("score personnel de B = %d, attendu 370 (slot 20 manche 0, puis slot 22 manche 1)", got)
	}
	for _, p := range []*PlayerScore{a, b} {
		if got := lastValue(p.Stats[objectives.StatHeadshots]); got != 6 {
			t.Errorf("%s : tirs a la tete = %d, attendu 6 (3 par manche)", p.XUID, got)
		}
	}
	if cov.StatsWithheld != 0 {
		t.Errorf("statsWithheld = %d, attendu 0 : les entrees nomment", cov.StatsWithheld)
	}
}

// TestEntryResolverOfExigeLesEntreesEtLesPlages : sans entree lue, sans plage lue ou sans ligne d'equipe qui prouve la disposition
// du tableau, il se nomme par le pont d'identite des slots.
func TestEntryResolverOfExigeLesEntreesEtLesPlages(t *testing.T) {
	equipes := []types.StatborgEntry{{Slot: 3, Kind: types.StatborgEntryTeam, ID: 0}, {Slot: 4, Kind: types.StatborgEntryTeam, ID: 1}}
	joueur := types.StatborgEntry{Slot: 5, Kind: types.StatborgEntryPlayer, ID: 1}
	es := append([]types.StatborgEntry{joueur}, equipes...)
	lu := types.StatborgEntryStats{Scanned: true}
	cas := []struct {
		nom string
		opt Options
		ok  bool
	}{
		{"tout lu", Options{StatborgEntries: es, StatborgEntryStats: lu, Presences: LecturesDePresence{ListingStats: types.RosterListingStats{Scanned: true}}}, true},
		{"entrees non lues", Options{StatborgEntries: es, Presences: LecturesDePresence{ListingStats: types.RosterListingStats{Scanned: true}}}, false},
		{"plages non lues", Options{StatborgEntries: es, StatborgEntryStats: lu}, false},
		{"une seule ligne d'equipe", Options{StatborgEntries: []types.StatborgEntry{joueur, equipes[1]}, StatborgEntryStats: lu, Presences: LecturesDePresence{ListingStats: types.RosterListingStats{Scanned: true}}}, false},
		{"disposition non prouvee", Options{StatborgEntries: []types.StatborgEntry{joueur}, StatborgEntryStats: lu, Presences: LecturesDePresence{ListingStats: types.RosterListingStats{Scanned: true}}}, false},
		{"aucune ligne de joueur", Options{StatborgEntryStats: lu, Presences: LecturesDePresence{ListingStats: types.RosterListingStats{Scanned: true}}}, false},
	}
	for _, c := range cas {
		if got := entryResolverOf(c.opt) != nil; got != c.ok {
			t.Errorf("%s : resolveur %v, attendu %v", c.nom, got, c.ok)
		}
	}
}

// TestScoreTimelineNommeParLesEntreesIgnoreLeVidageDUnJoueurParti : un joueur parti en fin de match, nomme par les entrees,
// garde sa derniere mort (la ligne videe qui suit n'est pas une valeur).
func TestScoreTimelineNommeParLesEntreesIgnoreLeVidageDUnJoueurParti(t *testing.T) {
	r := objectives.NewEntryResolver(
		[]types.StatborgEntry{{Slot: 5, TS: 1_000_000, Kind: types.StatborgEntryPlayer, ID: 1}},
		[]types.RosterListing{{XUID: 1001, Slot: 1}}, 0)
	recs := append(modeRamp(6, 0, 2_000, 1_000, 1, 2, 3),
		statRec(3_000, 10, 0, map[int]types.StatValue{2: {A: 4, B: 1}}),
		statRec(5_000, 10, 0, map[int]types.StatValue{2: {A: 0, B: 0}}))
	tl, _, _ := assembleScoreTimeline(&ScoreInput{Records: recs}, nil, r, testClock(), nil)
	p := playerByXUID(tl.Players, "1001")
	if p == nil || lastValue(p.Deaths) != 1 || lastValue(p.Kills) != 4 {
		t.Errorf("joueur parti : %+v", p)
	}
}

// TestScoreInputOfPlayedRoundsEcarteLesManchesFantomes : un film dont les issues ne montrent que la manche 0 perd les
// enregistrements d'une « manche 2 » fortuite, et la couverture le dit ; sans issues lues, l'entree est celle de l'appelant.
func TestScoreInputOfPlayedRoundsEcarteLesManchesFantomes(t *testing.T) {
	recs := []types.StatRecord{statRec(1_000, 10, 0, nil), statRec(2_000, 10, 2, nil), statRec(3_000, 10, 2, nil)}
	lu := types.StatborgEntryStats{Scanned: true, Read: 8, DecidedRounds: 0b1}
	in, unplayed := scoreInputOfPlayedRounds(Options{Score: &ScoreInput{Records: recs}, StatborgEntryStats: lu})
	if len(in.Records) != 1 || len(unplayed) != 1 || unplayed[0] != 2 {
		t.Errorf("%d enregistrement(s) gardes, manches non jouees %v", len(in.Records), unplayed)
	}
	if len(recs) != 3 {
		t.Error("l'entree de l'appelant est modifiee")
	}
	for _, opt := range []Options{
		{Score: &ScoreInput{Records: recs}},
		{Score: &ScoreInput{Records: recs}, StatborgEntryStats: types.StatborgEntryStats{Scanned: true}},
	} {
		if in, unplayed := scoreInputOfPlayedRounds(opt); len(in.Records) != 3 || unplayed != nil {
			t.Errorf("issues non lues : %d enregistrement(s), %v", len(in.Records), unplayed)
		}
	}
	if in, unplayed := scoreInputOfPlayedRounds(Options{}); in != nil || unplayed != nil {
		t.Errorf("aucune entree : %v %v", in, unplayed)
	}
}

// Un joueur qui tient deux lignes dans la meme manche (il change de place) : ses segments se mettent bout a bout dans l'ordre du temps,
// et pour un compteur cumulatif le second repart du total du premier. Sans ce decalage, la serie recule.
func TestJoinSegmentsDecaleLeSecondSegmentDUnCompteurCumulatif(t *testing.T) {
	premier := []types.ScorePoint{{TimeMS: 1_000, Slot: 10, Value: 1}, {TimeMS: 2_000, Slot: 10, Value: 3}}
	second := []types.ScorePoint{{TimeMS: 5_000, Slot: 1010, Value: 1}, {TimeMS: 6_000, Slot: 1010, Value: 2}}
	// Donnes dans le desordre : l'ordre est celui du premier point de chaque segment.
	got := joinSegments([][]types.ScorePoint{second, premier}, true)
	want := []int64{1, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("%d points, attendu %d : %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Value != w {
			t.Errorf("point %d = %d, attendu %d (%+v)", i, got[i].Value, w, got)
		}
	}
	// Un compteur non cumulatif (un pic) ne se decale pas.
	brut := joinSegments([][]types.ScorePoint{premier, second}, false)
	if brut[2].Value != 1 || brut[3].Value != 2 {
		t.Errorf("segments non cumulatifs decales : %+v", brut)
	}
}

// optionsDesEntrees pose, sur le film croise de [reassignFixture], les entrees `entrees` et les plages que [entreesCroisees] lit.
func optionsDesEntrees(entrees []types.StatborgEntry) Options {
	return Options{
		StatborgEntries:    entrees,
		StatborgEntryStats: types.StatborgEntryStats{Scanned: true},
		Presences: LecturesDePresence{
			ListingStats: types.RosterListingStats{Scanned: true},
			Listings:     []types.RosterListing{{XUID: 1001, Slot: 1}, {XUID: 1002, Slot: 2}},
		},
	}
}

// TestScoreTimelineBasculeSurLePontSansDispositionProuvee : de bout en bout, des entrees qui prouvent la disposition nomment les
// joueurs et leurs compteurs nommes ; sans cette preuve, le tableau se nomme par le pont d'identite des slots, qui prive d'abord de
// `stats` un film a plusieurs manches et le dit dans `statsWithheld`.
func TestScoreTimelineBasculeSurLePontSansDispositionProuvee(t *testing.T) {
	recs, deaths := reassignFixture()
	for _, slot := range []int{20, 22} {
		for round, t0 := range []int{1_000, 11_000} {
			for i := range 3 {
				recs = append(recs, statRec(t0+i*1_000, slot, round, map[int]types.StatValue{6: {A: int64(i + 1)}}))
			}
		}
	}
	entree := func(slot, tsMS, id int) types.StatborgEntry {
		return types.StatborgEntry{Slot: slot, TS: uint64(tsMS) * 1000, Kind: types.StatborgEntryPlayer, ID: id}
	}
	equipe := func(id int) types.StatborgEntry {
		return types.StatborgEntry{Slot: 3 + id, TS: 1_000_000, Kind: types.StatborgEntryTeam, ID: id}
	}
	joueurs := []types.StatborgEntry{entree(11, 1_000, 1), entree(10, 1_000, 2), entree(11, 11_500, 2), entree(10, 11_500, 1)}
	cas := []struct {
		nom         string
		entrees     []types.StatborgEntry
		stats       bool
		statsWithld int
	}{
		{"deux lignes d'equipe", append([]types.StatborgEntry{equipe(0), equipe(1)}, joueurs...), true, 0},
		{"une seule ligne d'equipe", append([]types.StatborgEntry{equipe(1)}, joueurs...), false, 2},
		{"aucune ligne d'equipe", joueurs, false, 2},
	}
	for _, c := range cas {
		tl, cov, _ := assembleScoreTimeline(&ScoreInput{Records: recs}, deaths, entryResolverOf(optionsDesEntrees(c.entrees)), multiRoundClock(), nil)
		if tl == nil || len(tl.Players) != 2 {
			t.Fatalf("%s : joueurs publies : %+v", c.nom, tl)
		}
		for _, p := range tl.Players {
			if got := p.Stats != nil; got != c.stats {
				t.Errorf("%s : %s publie ses compteurs nommes : %v, attendu %v", c.nom, p.XUID, got, c.stats)
			}
			if lastValue(p.Kills) != 6 {
				t.Errorf("%s : %s frags %d, attendu 6", c.nom, p.XUID, lastValue(p.Kills))
			}
		}
		if cov.StatsWithheld != c.statsWithld {
			t.Errorf("%s : statsWithheld = %d, attendu %d", c.nom, cov.StatsWithheld, c.statsWithld)
		}
	}
}
