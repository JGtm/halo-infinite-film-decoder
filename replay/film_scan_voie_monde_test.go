package replay

import "testing"

// TestVoieMondeLitLesSoclesAuxLargeursDesPoses (VOIE-MONDE) : les largeurs MPP que la voie du monde donne aux
// socles et aux vehicules sont celles que la calibration des poses a mesurees, c est-a-dire celles que le fil
// principal lit dans les statistiques publiees (`s.in.PlacementStats`) quand il lit en serie. Sur les bobines
// le format est connu et les largeurs du jeu l emportent : seule cette egalite garde le fil des largeurs.
// Mutation vue rouge : la voie ne retient pas les largeurs de ses poses. Limite : sur les bobines le format est connu, donc des largeurs nulles passees aux socles ne changeraient pas leur lecture ; les films reels le verifient.
func TestVoieMondeLitLesSoclesAuxLargeursDesPoses(t *testing.T) {
	avecHuitProcesseurs(t)
	valides := 0
	for _, dossier := range bobinesDuFond {
		var sortie sortieDesVoies
		s := nouveauBalayage(t, dossier, 4, &sortie)
		s.lireLaMarcheDesTrames()
		s.voies = s.demarrerLesVoies()
		if s.voies == nil {
			t.Fatalf("%s : aucune voie", dossier)
		}
		s.publierPontDIdentite(s.lirePontDIdentite())
		s.balayerMonde()
		got, want := s.voies.monde.largeurs, s.in.PlacementStats.Calibration.Widths
		if got != want {
			t.Errorf("%s : largeurs de la voie %v, publiees %v", dossier, got, want)
		}
		valides += unSiVrai(want.Valid())
		s.voies.joindre()
		s.fond.joindre()
	}
	if valides == 0 {
		t.Error("aucune bobine ne porte de largeurs valides : le test ne prouverait rien")
	}
}
