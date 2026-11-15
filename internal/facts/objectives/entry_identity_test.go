package objectives

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Les identifiants de ces tests sont fictifs.
const (
	xuidA = 1001
	xuidB = 1002
	xuidC = 1003
)

func entree(slot int, tsMS int, id int) types.StatborgEntry {
	return types.StatborgEntry{Slot: slot, TS: uint64(tsMS) * 1000, Kind: types.StatborgEntryPlayer, ID: id}
}

func plage(xuid uint64, id int, fromMS, toMS int) types.RosterListing {
	return types.RosterListing{XUID: xuid, Slot: id, FromUS: uint64(fromMS) * 1000, ToUS: uint64(toMS) * 1000}
}

func rec(slot, round, tMS int) types.StatRecord {
	return types.StatRecord{Slot: slot, Round: round, TimeMS: tMS, Comps: map[int]types.StatValue{2: {A: 1}}}
}

func TestEntryResolverNommeLeJoueurDeLaLigne(t *testing.T) {
	// L'entite 5 est la ligne de l'identifiant 3 ; A tient l'identifiant 3 jusqu'a 50 s, puis B le prend.
	r := NewEntryResolver(
		[]types.StatborgEntry{entree(5, 1_000, 3), entree(5, 21_000, 3), entree(5, 41_000, 3), entree(5, 61_000, 3)},
		[]types.RosterListing{plage(xuidA, 3, 0, 50_000), plage(xuidB, 3, 50_000, 0)}, 0)
	cas := []struct {
		slot, tMS int
		want      uint64
		ok        bool
	}{
		{10, 5_000, xuidA, true},
		{10, 49_999, xuidA, true},
		{10, 50_000, xuidB, true},
		{10, 120_000, xuidB, true},
		{12, 5_000, 0, false}, // l'entite 6 n'a pas d'entree
	}
	for _, c := range cas {
		got, ok := r.Holder(c.slot, c.tMS, 0)
		if ok != c.ok || got != c.want {
			t.Errorf("slot %d a %d ms : %d, %v ; attendu %d, %v", c.slot, c.tMS, got, ok, c.want, c.ok)
		}
	}
}

// TestEntryResolverUneMancheReattribueLesLignes : l'entite 5 est la ligne de l'identifiant 3 (A) en manche 0, puis de
// l'identifiant 4 (B) en manche 1, qui commence a 30 s ; l'image-cle de 21 s porte encore A. Un enregistrement de la
// manche 1 a 31 s ne prend pas l'entree d'avant la manche.
func TestEntryResolverUneMancheReattribueLesLignes(t *testing.T) {
	r := NewEntryResolver(
		[]types.StatborgEntry{entree(5, 1_000, 3), entree(5, 21_000, 3), entree(5, 41_000, 4)},
		[]types.RosterListing{plage(xuidA, 3, 0, 0), plage(xuidB, 4, 0, 0)}, 0)
	if got, _ := r.Holder(10, 25_000, 0); got != xuidA {
		t.Errorf("manche 0 : %d, attendu A", got)
	}
	if got, _ := r.Holder(10, 31_000, 30_000); got != xuidB {
		t.Errorf("manche 1 avant l'image-cle suivante : %d, attendu B (jamais l'entree d'avant la manche)", got)
	}
	if got, _ := r.Holder(10, 45_000, 30_000); got != xuidB {
		t.Errorf("manche 1 : %d, attendu B", got)
	}
}

func TestEntryResolverSplitDonneUnSlotVirtuelParJoueur(t *testing.T) {
	r := NewEntryResolver(
		[]types.StatborgEntry{entree(5, 1_000, 3), {Slot: 3, TS: 1_000_000, Kind: types.StatborgEntryTeam, ID: 0}},
		[]types.RosterListing{plage(xuidA, 3, 0, 50_000), plage(xuidC, 3, 50_000, 0)}, 0)
	recs := []types.StatRecord{rec(6, 0, 1_000), rec(10, 0, 10_000), rec(10, 0, 60_000), rec(12, 0, 10_000), rec(10, 0, 70_000)}
	out, identite := r.Split(recs)
	if out[0].Slot != 6 {
		t.Errorf("slot d'equipe change : %d", out[0].Slot)
	}
	if out[3].Slot != 12 {
		t.Errorf("slot sans entree change : %d", out[3].Slot)
	}
	if out[1].Slot < slotVirtuelMin || out[2].Slot < slotVirtuelMin || out[1].Slot == out[2].Slot || out[2].Slot != out[4].Slot {
		t.Errorf("slots virtuels : %d %d %d", out[1].Slot, out[2].Slot, out[4].Slot)
	}
	if identite[out[1].Slot] != "1001" || identite[out[2].Slot] != "1003" || len(identite) != 2 {
		t.Errorf("identite : %v", identite)
	}
	if recs[1].Slot != 10 {
		t.Error("les enregistrements d'entree sont modifies")
	}
}

func TestEntryResolverSansEntreeNeNommePersonne(t *testing.T) {
	r := NewEntryResolver(nil, []types.RosterListing{plage(xuidA, 3, 0, 0)}, 0)
	out, identite := r.Split([]types.StatRecord{rec(10, 0, 1_000)})
	if out[0].Slot != 10 || len(identite) != 0 {
		t.Errorf("%v %v", out, identite)
	}
}

// TestEntriesProveLayout : le tableau s'ouvre sur ses deux lignes d'equipe ; des entrees qui n'en portent pas deux de numeros
// distincts, ou qui sont toutes nulles, ne prouvent pas la disposition.
func TestEntriesProveLayout(t *testing.T) {
	equipe := func(id int) types.StatborgEntry {
		return types.StatborgEntry{Slot: 3 + id, TS: 1_000_000, Kind: types.StatborgEntryTeam, ID: id}
	}
	cas := []struct {
		nom     string
		entries []types.StatborgEntry
		want    bool
	}{
		{"deux lignes d'equipe et des joueurs", []types.StatborgEntry{equipe(0), equipe(1), entree(5, 1_000, 3)}, true},
		{"deux lignes d'equipe lues a plusieurs images-cles", []types.StatborgEntry{equipe(1), equipe(0), equipe(1), equipe(0)}, true},
		{"numeros d'equipe distincts quelconques", []types.StatborgEntry{equipe(2), equipe(5)}, true},
		{"une seule ligne d'equipe", []types.StatborgEntry{equipe(0), entree(5, 1_000, 3)}, false},
		{"une seule ligne d'equipe lue plusieurs fois", []types.StatborgEntry{equipe(1), equipe(1), equipe(1)}, false},
		{"joueurs distincts sans ligne d'equipe", []types.StatborgEntry{entree(5, 1_000, 3), entree(6, 1_000, 4)}, false},
		{"entrees toutes nulles", []types.StatborgEntry{entree(5, 1_000, 0), entree(6, 1_000, 0), entree(7, 1_000, 0)}, false},
		{"aucune entree", nil, false},
	}
	for _, c := range cas {
		if got := EntriesProveLayout(c.entries); got != c.want {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.want)
		}
	}
}
