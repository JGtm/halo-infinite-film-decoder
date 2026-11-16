package replay

import (
	"encoding/json"
	"strings"
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

// Le delai de retour du drapeau vient de l option de la variante, quand elle la pose et qu elle est numerique.
func TestRegles_DelaiDeRetourDuDrapeau(t *testing.T) {
	avec := func(opts ...profile.OptionDeVariante) *profile.FilmIdentity {
		return &profile.FilmIdentity{InfosDeMatch: profile.InfosDeMatch{VarianteLue: true, Options: opts}}
	}
	tests := []struct {
		nom  string
		id   *profile.FilmIdentity
		want *float64
	}{
		{"option posee", avec(profile.OptionDeVariante{Hachage: 1, Genre: profile.OptionNombre, Valeur: 5},
			profile.OptionDeVariante{Hachage: hachageDelaiDeRetourDuDrapeau, Genre: profile.OptionNombre, Valeur: 24}), new(24.0)},
		{"option absente", avec(profile.OptionDeVariante{Hachage: 1, Genre: profile.OptionNombre, Valeur: 5}), nil},
		{"option booleenne", avec(profile.OptionDeVariante{Hachage: hachageDelaiDeRetourDuDrapeau,
			Genre: profile.OptionBooleenne, Valeur: 1}), nil},
		{"variante non lue", &profile.FilmIdentity{InfosDeMatch: profile.InfosDeMatch{Options: []profile.OptionDeVariante{
			{Hachage: hachageDelaiDeRetourDuDrapeau, Genre: profile.OptionNombre, Valeur: 24}}}}, nil},
	}
	for _, c := range tests {
		r := rulesOf(c.id)
		switch {
		case c.want == nil && r.FlagReturnS != nil:
			t.Errorf("%s : %v, attendu absent", c.nom, *r.FlagReturnS)
		case c.want != nil && (r.FlagReturnS == nil || *r.FlagReturnS != *c.want):
			t.Errorf("%s : %v, attendu %v", c.nom, r.FlagReturnS, *c.want)
		}
	}
}

// La regle de retour du mode ne publie que le delai de la variante du film : sans lui, `resetSeconds` est absent, et
// le delai du manifeste n est jamais publie a sa place.
func TestAttachFlagReturnZone_DelaiDeLaVariante(t *testing.T) {
	manifeste := FlagReturnZone{RadiusM: 1.3, ResetSeconds: 30, SoloSeconds: 3.1}
	carries := []FlagCarry{{}}
	retour := 24.0
	tests := []struct {
		nom   string
		rules *MatchRules
		want  float32
	}{
		{"delai de la variante", &MatchRules{FlagReturnS: &retour}, 24},
		{"variante sans delai", &MatchRules{}, 0},
		{"aucune regle", nil, 0},
	}
	for _, c := range tests {
		doc := &ReplayDocument{Rules: c.rules}
		attachFlagReturnZone(doc, manifeste, carries)
		if doc.FlagReturnZone == nil || doc.FlagReturnZone.ResetSeconds != c.want ||
			doc.FlagReturnZone.SoloSeconds != 3.1 || doc.FlagReturnZone.RadiusM != 1.3 {
			t.Errorf("%s : %+v, attendu un delai de %v", c.nom, doc.FlagReturnZone, c.want)
		}
	}
	b, err := json.Marshal(FlagReturnZone{RadiusM: 1, SoloSeconds: 2})
	if err != nil || strings.Contains(string(b), "resetSeconds") {
		t.Errorf("un delai nul ne se publie pas : %s (%v)", b, err)
	}
	doc := &ReplayDocument{Rules: &MatchRules{FlagReturnS: &retour}}
	attachFlagReturnZone(doc, manifeste, nil)
	if doc.FlagReturnZone != nil {
		t.Error("sans drapeau : aucune regle de retour")
	}
}
