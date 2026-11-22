package objectives

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// named_modes_test.go — les compteurs de mode nommes par emplacement : eliminations de ceux
// qui ramenent le drapeau, eliminations offensives et defensives des bases, eliminations et
// assistances de mode du VIP. Tests synthetiques : aucun film.

// emplacementDeMode : un emplacement de mode, son cote, la famille qui le nomme et le nom attendu.
type emplacementDeMode struct {
	famille string
	comp    int
	side    string
	stat    string
}

var emplacementsDeMode = []emplacementDeMode{
	{ObjectiveTypeFlag, 22, sideB, StatFlagReturnersKilled},
	{ObjectiveTypeFlag, 25, sideA, StatFlagKillsAsReturner},
	{ObjectiveTypeFlag, 24, sideB, StatFlagKillsAsCarrier},
	{ObjectiveTypeZone, 21, sideB, StatZoneOffensiveKills},
	{ObjectiveTypeZone, 22, sideA, StatZoneDefensiveKills},
	{ObjectiveTypeVip, 20, sideB, StatVipKills},
	{ObjectiveTypeVip, 21, sideA, StatVipAssists},
	{ObjectiveTypeVip, 21, sideB, StatVipKillsAsVip},
}

// enregistrementsDe rend trois emissions du composant `comp` : la valeur de l'emplacement monte de 0 a 1 puis a 3.
func enregistrementsDe(comp int, side string, slot int) []types.StatRecord {
	val := func(n int64) map[int]types.StatValue {
		if side == sideB {
			return map[int]types.StatValue{comp: {B: n}}
		}
		return map[int]types.StatValue{comp: {A: n}}
	}
	return []types.StatRecord{
		{TimeMS: 1000, Slot: slot, Comps: val(0)},
		{TimeMS: 2000, Slot: slot, Comps: val(1)},
		{TimeMS: 3500, Slot: slot, Comps: val(3)},
	}
}

// Chaque emplacement nomme emet une unite par elimination, datee a l'emission, sous le nom de sa famille.
func TestCompteursDeModeUneUniteParEvenement(t *testing.T) {
	for _, e := range emplacementsDeMode {
		got := NamedEventsFrom(enregistrementsDe(e.comp, e.side, 12), e.famille, nil)
		if len(got) != 3 {
			t.Errorf("%s %d%s : %d evenements, attendu 3", e.famille, e.comp, e.side, len(got))
			continue
		}
		wantTimes := []int{2000, 3500, 3500}
		for i, ev := range got {
			if ev.Stat != e.stat || ev.Slot != 12 || ev.TimeMS != wantTimes[i] || ev.Comp != e.comp || ev.Side != e.side {
				t.Errorf("%s %d%s : evenement %d = %+v, attendu %s a t=%d", e.famille, e.comp, e.side, i, ev, e.stat, wantTimes[i])
			}
		}
	}
}

// Un meme emplacement ne vaut que pour sa famille : `22 A` est une prise de drapeau en CTF et une
// elimination defensive des bases ; `21 B` est un porteur tue en CTF et une elimination offensive.
func TestCompteursDeModeLeSensDependDeLaFamille(t *testing.T) {
	for _, e := range emplacementsDeMode {
		for _, autre := range []string{ObjectiveTypeFlag, ObjectiveTypeZone, ObjectiveTypeVip} {
			if autre == e.famille {
				continue
			}
			for _, ev := range NamedEventsFrom(enregistrementsDe(e.comp, e.side, 12), autre, nil) {
				if ev.Stat == e.stat {
					t.Errorf("%s publie %s, qui n'appartient qu'a %s", autre, e.stat, e.famille)
				}
			}
		}
	}
}

// Les compteurs de duree et la meilleure serie ne se nomment pas : un pas de dixieme de seconde
// n'est pas un evenement. Le VIP porte `LongestTimeAsVip` en 23 A et `TimeAsVip` en 23 B, les
// bases leur tic de score en 23 A.
func TestCompteursDeDureeNeSontPasDesEvenements(t *testing.T) {
	var recs []types.StatRecord
	for i := range 40 {
		recs = append(recs, types.StatRecord{TimeMS: 1000 + 100*i, Slot: 12,
			Comps: map[int]types.StatValue{23: {A: int64(i), B: int64(i)}, 22: {B: int64(i)}}})
	}
	if got := NamedEventsFrom(recs, ObjectiveTypeVip, nil); len(got) != 0 {
		t.Errorf("VIP : %d evenements pour des compteurs de duree, attendu 0 (%+v)", len(got), got[0])
	}
	for _, ev := range NamedEventsFrom(recs, ObjectiveTypeZone, nil) {
		if ev.Comp == 23 {
			t.Errorf("zones : le tic de score (23 A) ne se nomme pas, evenement %+v", ev)
		}
	}
}

// Un saut aberrant d'un compteur de mode (75 d'un coup, constate sur une emission mal alignee) n'est
// pas deroule en soixante-quinze evenements : la borne par pas le rejette.
func TestCompteursDeModeSautAberrantRejete(t *testing.T) {
	recs := []types.StatRecord{
		{TimeMS: 1000, Slot: 12, Comps: map[int]types.StatValue{25: {A: 0}}},
		{TimeMS: 2000, Slot: 12, Comps: map[int]types.StatValue{25: {A: 75}}},
	}
	if got := NamedEventsFrom(recs, ObjectiveTypeFlag, nil); len(got) != 0 {
		t.Errorf("%d evenements d'un saut de 75, attendu 0", len(got))
	}
}

// Les inventaires par famille portent les nouveaux noms, et seulement les leurs.
func TestInventaireDesCompteursDeMode(t *testing.T) {
	for _, e := range emplacementsDeMode {
		if !KnownStats(e.famille)[e.stat] {
			t.Errorf("%s : %s absent de l'inventaire", e.famille, e.stat)
		}
	}
	if KnownStats(ObjectiveTypeHill)[StatZoneOffensiveKills] {
		t.Error("colline : aucun compteur de mode n'est replique, la table des bases ne vaut pas pour elle")
	}
}
