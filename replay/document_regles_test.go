package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestRegles(t *testing.T) {
	if !rulesOf(nil).vide() {
		t.Fatal("sans identite : aucune regle")
	}
	if !rulesOf(&profile.FilmIdentity{}).vide() {
		t.Fatal("variante non lue : aucune regle")
	}
	delai, tir := int64(8), false
	id := &profile.FilmIdentity{InfosDeMatch: profile.InfosDeMatch{VarianteLue: true,
		DelaiDeReapparitionSecondes: &delai, TirsAllies: &tir}}
	r := rulesOf(id)
	if r.MinRespawnS == nil || *r.MinRespawnS != 8 || r.FriendlyFire == nil || *r.FriendlyFire ||
		r.BetrayalPenaltyS != nil || r.SuicidePenaltyS != nil {
		t.Fatalf("regles %+v", r)
	}
}
