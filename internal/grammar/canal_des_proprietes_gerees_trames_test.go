package grammar

import (
	"reflect"
	"testing"
)

// Le variant se lit comme le jeu : l etiquette sur 4 bits, puis la charge que la moitie du mode choisit.
func TestDecodeManagedProperty(t *testing.T) {
	tests := []struct {
		nom   string
		bits  []byte
		modeA bool
		want  ManagedPropertyValue
	}{
		// 0011 (tag 3, quantifie) puis R(24) = 0x800000.
		{"scalaire quantifie", []byte{0b00111000, 0b00000000, 0b00000000, 0b00000000}, true,
			ManagedPropertyValue{Tag: ManagedPropertyTagQuant, Value: 0x800000, HasValue: true}},
		// 0010 (tag 2, booleen) puis R(1) = 1.
		{"scalaire booleen", []byte{0b00101000}, true,
			ManagedPropertyValue{Tag: ManagedPropertyTagBool, Value: 1, HasValue: true}},
		// 0000 : aucune charge.
		{"etiquette vide", []byte{0b00000000}, true, ManagedPropertyValue{Tag: ManagedPropertyTagVide}},
		// 0111 (tag 7, quantifie par joueur) puis R(24) = 1.
		{"par joueur quantifie", []byte{0b01110000, 0b00000000, 0b00000000, 0b00010000}, false,
			ManagedPropertyValue{Tag: ManagedPropertyTagQuantJ, Value: 1, HasValue: true}},
		// 0011 en mode par joueur : l etiquette du mode A est muette.
		{"etiquette de l autre mode", []byte{0b00110000}, false, ManagedPropertyValue{Tag: ManagedPropertyTagQuant}},
	}
	for _, c := range tests {
		br := LecteurSur(c.bits)
		if got := decodeManagedProperty(br, c.modeA); got != c.want {
			t.Errorf("%s : %+v, attendu %+v", c.nom, got, c.want)
		}
	}
}

// ranger verse les valeurs de ti=13 d un record, avec leur champ et leur index de joueur, et ignore le reste.
func TestCanalDesProprietesGereesTrames_Ranger(t *testing.T) {
	val := func(tag int, v uint64) ManagedPropertyValue {
		return ManagedPropertyValue{Tag: tag, Value: v, HasValue: true}
	}
	rec := &FrameRecord{Slot: 1439, DesyncAt: -1, Trace: EntityTrace{Comps: []CompResult{
		{Index: 0, Name: compManagedObjectPropName, Ported: true},                                 // pas une valeur
		{Index: 1, Name: compManagedObjectProperty, Ported: true, Payload: val(3, 42)},            // scalaire
		{Index: 5, Name: compManagedObjectPlayerMaskedProperty, Ported: true, Payload: val(7, 9)}, // joueur 3
		{Index: 6, Name: compManagedObjectPlayerMaskedProperty, Ported: false, Payload: val(7, 1)},
	}}}
	c := nouveauCanalDesProprietesGereesTrames()
	if !c.ranger(1234, rec) {
		t.Fatal("le record porte des valeurs")
	}
	want := []ManagedPropertyRead{
		{Slot: 1439, TimestampUS: 1234, Field: ManagedPropertyScalar, FilmIndex: -1, Tag: 3, Value: 42, HasValue: true, Chained: true},
		{Slot: 1439, TimestampUS: 1234, Field: ManagedPropertyPerPlayer, FilmIndex: 3, Tag: 7, Value: 9, HasValue: true, Chained: true},
	}
	if !reflect.DeepEqual(c.lu.Lectures, want) {
		t.Errorf("lectures %+v, attendu %+v", c.lu.Lectures, want)
	}

	// Un record desynchronise rend ses valeurs, non chainees ; un record sans valeur n en rend aucune.
	c = nouveauCanalDesProprietesGereesTrames()
	des := &FrameRecord{Slot: 7, DesyncAt: 4, Trace: EntityTrace{Comps: []CompResult{
		{Index: 1, Name: compManagedObjectProperty, Ported: true, Payload: val(3, 1)}}}}
	if !c.ranger(1, des) || c.lu.Lectures[0].Chained {
		t.Errorf("record desynchronise : %+v", c.lu.Lectures)
	}
	if c.ranger(2, &FrameRecord{Slot: 8, DesyncAt: -1, Trace: EntityTrace{Comps: []CompResult{{Index: 0, Name: "high-frequency", Ported: true}}}}) {
		t.Error("un record sans valeur de propriete ne compte pas")
	}
}
