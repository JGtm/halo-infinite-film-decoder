package replay

// Tests — LA FORME DES AXES ET DE L'ECHELLE D'UN EMPLACEMENT.
//
// Forward, Up et Scale sont renseignes par l'hote : le calque servi sans eux garde exactement la
// forme d'avant (aucune cle neuve), et avec eux ils voyagent tels quels.

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMapWeaponPadDTO_AxesAbsentsNeChangentPasLaForme : sans axes ni echelle, aucune des trois cles
// n'apparait dans le JSON.
// MUTATIONS — `omitempty` retire d'un des trois champs : ROUGE.
func TestMapWeaponPadDTO_AxesAbsentsNeChangentPasLaForme(t *testing.T) {
	got := BuildMapWeaponPads(carteTemoin(), []WeaponPad{filmPad(-9.74, 0, 22.40)})
	if got == nil {
		t.Fatal("calque nil")
	}
	brut, err := json.Marshal(got.Pads[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, cle := range []string{`"forward"`, `"up"`, `"scale"`} {
		if strings.Contains(string(brut), cle) {
			t.Errorf("cle %s presente sans valeur : %s", cle, brut)
		}
	}
}

// TestMapWeaponPadDTO_AxesVoyagent : l'hote pose les axes et l'echelle, le JSON les porte sous
// leurs trois noms, et un aller-retour les rend identiques.
func TestMapWeaponPadDTO_AxesVoyagent(t *testing.T) {
	dto := MapWeaponPadDTO{X: 1, Y: 2, Pad: 3, Family: "rack",
		Forward: &[3]float32{0, 1, 0}, Up: &[3]float32{0, 0, 1}, Scale: &[3]float32{1, 1, 2}}
	brut, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	for _, cle := range []string{`"forward":[0,1,0]`, `"up":[0,0,1]`, `"scale":[1,1,2]`} {
		if !strings.Contains(string(brut), cle) {
			t.Errorf("%s absent : %s", cle, brut)
		}
	}
	var relu MapWeaponPadDTO
	if err := json.Unmarshal(brut, &relu); err != nil {
		t.Fatal(err)
	}
	if relu.Forward == nil || relu.Up == nil || relu.Scale == nil || relu.Up[2] != 1 || relu.Scale[2] != 2 {
		t.Errorf("aller-retour : %+v", relu)
	}
}
