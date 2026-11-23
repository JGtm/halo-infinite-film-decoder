package types_test

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestLaPorteDeCadenceSeLitSurLePasMesure : la pleine cadence demande un pas mesurable (assez d ecarts) et au plus
// `PasDeCommande60HzMaxUS` ; les pas mesures du corpus de reference tombent de chaque cote de la porte.
func TestLaPorteDeCadenceSeLitSurLePasMesure(t *testing.T) {
	cas := []struct {
		nom  string
		st   types.ControlHoldStats
		want bool
	}{
		{"60 Hz, plus grand pas mesure", types.ControlHoldStats{StepUS: 16_635, StepGaps: 200_000}, true},
		{"30 Hz, plus petit pas mesure", types.ControlHoldStats{StepUS: 33_438, StepGaps: 60_000}, false},
		{"au plafond", types.ControlHoldStats{StepUS: types.PasDeCommande60HzMaxUS, StepGaps: types.EcartsDuPasMin}, true},
		{"au-dessus du plafond", types.ControlHoldStats{StepUS: types.PasDeCommande60HzMaxUS + 1, StepGaps: 100_000}, false},
		{"trop peu d ecarts", types.ControlHoldStats{StepUS: 16_600, StepGaps: types.EcartsDuPasMin - 1}, false},
		{"aucun pas", types.ControlHoldStats{}, false},
	}
	for _, c := range cas {
		if got := c.st.PleineCadence(); got != c.want {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.want)
		}
	}
}
