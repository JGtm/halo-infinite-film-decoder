package profile

import "testing"

// TestEcritureDuBuildPourSuitLaTable : un champ de [EcritureDuBuild] est vrai exactement pour les builds dont la
// table du profil porte une ligne de ce champ, et faux pour tout autre build, un build vide ou inconnu compris.
// MUTATIONS — un build ajoute a [EcritureDuBuildPour] sans ligne de table, ou une ligne retiree : ROUGE.
func TestEcritureDuBuildPourSuitLaTable(t *testing.T) {
	champs := map[string]func(EcritureDuBuild) bool{
		champChargesDeVueA:    func(e EcritureDuBuild) bool { return e.ChargesDeVueAMesurees },
		champCreationParPorte: func(e EcritureDuBuild) bool { return e.CreationParPorte },
		champProprieteGeree:   func(e EcritureDuBuild) bool { return e.ProprieteGereeComplete },
		champInterruption:     func(e EcritureDuBuild) bool { return e.InterruptionDuMoteurLue },
		champBascule:          func(e EcritureDuBuild) bool { return e.BasculeDeControleLue },
		champCapacite:         func(e EcritureDuBuild) bool { return e.ActivationDeCapaciteLue },
		champTeleportation:    func(e EcritureDuBuild) bool { return e.TeleportationDuTranslocateurLue },
	}
	lignes := map[string]bool{}
	for _, l := range tableProfilEcriture() {
		if _, ok := champs[l.Champ]; !ok {
			t.Errorf("ligne %s / %s : champ inconnu de EcritureDuBuild", l.Cle, l.Champ)
		}
		lignes[l.Cle+"|"+l.Champ] = true
	}
	builds := append(buildsDeLaTable(), "", "HI_1_14_0", "HI_1_1_0")
	for champ, lire := range champs {
		for _, build := range builds {
			if got, want := lire(EcritureDuBuildPour(build)), lignes["build="+build+"|"+champ]; got != want {
				t.Errorf("build %q, champ %s : %v, la table dit %v", build, champ, got, want)
			}
		}
	}
	ouverts := map[string][4]bool{ // genres 13, 27, 93, 117
		"HI_1_11_0": {true, true, true, false},
		"HI_1_12_0": {false, false, true, true},
		"HI_1_13_0": {false, true, true, true},
	}
	for build, want := range ouverts {
		e := EcritureDuBuildPour(build)
		if got := [4]bool{e.InterruptionDuMoteurLue, e.BasculeDeControleLue, e.ActivationDeCapaciteLue, e.TeleportationDuTranslocateurLue}; got != want {
			t.Errorf("build %s : charges courtes %v, attendu %v", build, got, want)
		}
	}
	for _, build := range []string{"HI_1_11_0", "HI_1_12_0", "HI_1_13_0"} {
		if e := EcritureDuBuildPour(build); !e.ChargesDeVueAMesurees || !e.CreationParPorte || !e.ProprieteGereeComplete {
			t.Errorf("build %s : %+v, attendu les trois ecritures", build, e)
		}
	}
	for _, build := range []string{"HI_1_10_0", "HI_1_9_0", "HI_1_8_0", "HI_1_4_1", ""} {
		if e := EcritureDuBuildPour(build); e != (EcritureDuBuild{}) {
			t.Errorf("build %q : %+v, attendu aucune ecriture (non prouvee)", build, e)
		}
	}
}

// TestLeProfilResoutLEcritureDeSonBuild : le profil d un film porte l ecriture de son build, et celle d un film
// sans section d identification est nulle.
func TestLeProfilResoutLEcritureDeSonBuild(t *testing.T) {
	avec := Resoudre(ClesDuFilm{RegistrePresent: true, Format: formatConnu, IdentiteLue: true,
		Identite: FilmIdentity{Build: "HI_1_12_0"}}, nil)
	if avec.Ecriture() != EcritureDuBuildPour("HI_1_12_0") || !avec.Ecriture().CreationParPorte {
		t.Errorf("film HI_1_12_0 : %+v", avec.Ecriture())
	}
	if sans := Resoudre(ClesDuFilm{RegistrePresent: true, Format: formatConnu}, nil); sans.Ecriture() != (EcritureDuBuild{}) {
		t.Errorf("film sans section d identification : %+v, attendu aucune ecriture", sans.Ecriture())
	}
}
