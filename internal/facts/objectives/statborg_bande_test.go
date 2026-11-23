package objectives

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/signaux"
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
