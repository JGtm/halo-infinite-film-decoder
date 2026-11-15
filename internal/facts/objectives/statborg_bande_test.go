package objectives

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/signaux"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLaBandeParDefautSeCompteEtSeDit : un film dont aucune table de datums ne porte la bande des
// joueurs compte UN repli et leve un diagnostic ; un film dont la table la porte, aucun.
func TestLaBandeParDefautSeCompteEtSeDit(t *testing.T) {
	par := signaux.LectureDuStatborg{BandeParDefaut: true, BlocsDeDatumsRefuses: 2, ChunksDatables: 1}
	if got := comptesDuStatborg(par).BandesParDefaut; got != 1 {
		t.Errorf("repli compte %d fois, attendu 1", got)
	}
	diags := diagnosticsDuStatborg(par, nil, "m")
	if len(diags) != 1 || diags[0].Code != DiagStatborgBandeParDefaut {
		t.Fatalf("diagnostics = %+v, attendu un %s", diags, DiagStatborgBandeParDefaut)
	}
	sain := signaux.LectureDuStatborg{ChunksDatables: 1}
	if got := comptesDuStatborg(sain).BandesParDefaut; got != 0 {
		t.Errorf("repli compte %d fois sur un film dont la table porte la bande", got)
	}
	if d := diagnosticsDuStatborg(sain, nil, "m"); len(d) != 0 {
		t.Errorf("diagnostics = %+v sur un film sain", d)
	}
}

func TestRecordsDeLaBandeSure(t *testing.T) {
	recs := []types.StatRecord{{Slot: 6}, {Slot: 8}, {Slot: 10}, {Slot: 24}, {Slot: 26}, {Slot: 58}}
	got := RecordsDeLaBandeSure(recs)
	if len(got) != 4 || got[3].Slot != 24 {
		t.Errorf("bande sure = %+v, attendu les slots 6, 8, 10 et 24", got)
	}
	if len(RecordsDeLaBandeSure(nil)) != 0 {
		t.Error("aucun enregistrement : aucun rendu")
	}
}

func TestSiegesAuDelaDeLaBandeSure(t *testing.T) {
	recs := []types.StatRecord{{Slot: 8}, {Slot: 24}, {Slot: 26}, {Slot: 26}, {Slot: 58}}
	if got := SiegesAuDelaDeLaBandeSure(recs); got != 2 {
		t.Errorf("sieges au-dela de la bande = %d, attendu 2 (les slots 26 et 58, comptes une fois)", got)
	}
	if SiegesAuDelaDeLaBandeSure(nil) != 0 {
		t.Error("aucun enregistrement : aucun siege")
	}
}
