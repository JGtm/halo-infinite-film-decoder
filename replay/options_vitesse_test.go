package replay

import (
	"runtime"
	"testing"
)

func TestOptionsVitesseFilsEffectifs(t *testing.T) {
	cas := []struct {
		nom                string
		demande, gomaxproc int
		attendu            int
	}{
		{"zero vaut un fil", 0, 16, 1},
		{"negatif vaut un fil", -3, 16, 1},
		{"dans les bornes", 4, 16, 4},
		{"au-dela de la borne", 100, 64, MaxFils},
		{"egal a la borne", MaxFils, 64, MaxFils},
		{"au-dela de GOMAXPROCS", 6, 2, 2},
		{"GOMAXPROCS a un", 6, 1, 1},
		{"GOMAXPROCS degenere", 6, 0, 1},
	}
	for _, c := range cas {
		if got := filsEffectifs(c.demande, c.gomaxproc); got != c.attendu {
			t.Errorf("%s : filsEffectifs(%d, %d) = %d, attendu %d", c.nom, c.demande, c.gomaxproc, got, c.attendu)
		}
	}
}

func TestOptionsVitesseValeurZeroUnFil(t *testing.T) {
	if got := (OptionsVitesse{}).FilsEffectifs(); got != 1 {
		t.Errorf("valeur zero : %d fils, attendu 1", got)
	}
	if got := (OptionsVitesse{Fils: 1 << 20}).FilsEffectifs(); got > MaxFils || got > runtime.GOMAXPROCS(0) || got < 1 {
		t.Errorf("fils hors bornes : %d", got)
	}
}
