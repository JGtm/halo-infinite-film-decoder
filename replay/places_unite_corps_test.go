package replay

// places_unite_corps_test.go — UN RECORD DE TIR DONT L UNITE N EST PAS LE CORPS QUE LE REGISTRE NOMME NE LIE PAS SA PLACE.
//
// Les tampons reproduisent le defaut mesure : un tir dont le record porte la place d un joueur B et l unite d un slot que
// le pont rattache a un joueur C, alors que ce slot ne porte le corps de C que BIEN APRES le tir (le slot a ete reutilise :
// la generation de l unite n est pas celle du corps de C). Sans la regle, la place de B est liee a C pour une frame, entre
// deux tenues de B, et le tir passe a C.

import (
	"encoding/json"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// tirDeLUnite : un tir de la place `place` a la frame `fr`, dont l unite est (slot, gen).
func tirDeLUnite(place int, slot, gen uint32, fr int) grammar.FireEvent {
	e := tirDePlace(place, slot, fr)
	e.Unit.Gen = gen
	return e
}

// TestUneUniteQuiNEstPasLeCorpsDuRegistreNeLieRien : le tir de l unite 765 (generation 2) ne lie pas la place 21 a C (joueur 19) :
// le registre ne lit sur ce slot que le corps de generation 1 de C, cree longtemps apres. La place reste tenue par B (joueur 16)
// sur toute la presence ; le record ecarte se compte.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas comparer la generation ; ignorer le record ecarte dans le compte ; comparer
// la generation du corps a l instant du premier record du slot et non a celui du tir.
func TestUneUniteQuiNEstPasLeCorpsDuRegistreNeLieRien(t *testing.T) {
	occ := occupantsDesSlots{
		plat:  map[uint32]int{10: 16, 765: 19},
		corps: corpsParSlot([]grammar.BipedCreation{creationGen(765, 900*100_000, 19, 1)}),
	}
	fire := []grammar.FireEvent{
		tirDeLUnite(21, 10, 0, 100), tirDeLUnite(21, 765, 2, 200), tirDeLUnite(21, 10, 0, 300),
	}
	p := lirePlacesDeControle(fire, occ, vsClock(), map[int][2]int{16: {0, 999}}, true)
	if p.cov.UnitContradicted != 1 || p.cov.Bindings != 2 || p.cov.Handovers != 0 || p.cov.Returns != 0 {
		t.Fatalf("couverture = %+v, attendu 1 record ecarte, 2 liaisons de B, aucun relais", p.cov)
	}
	if j, ok := p.joueurA(21, 200); !ok || j != 16 {
		t.Errorf("la place 21 a la frame 200 est tenue par %d (lu %v), attendu B (16)", j, ok)
	}
	// Le meme tir, dont l unite porte la generation du corps lu, lie bien la place : la regle ne retire que la contradiction.
	fire[1] = tirDeLUnite(21, 765, 1, 950)
	if p := lirePlacesDeControle(fire, occ, vsClock(), nil, true); p.cov.UnitContradicted != 0 || p.cov.Bindings != 3 {
		t.Errorf("couverture = %+v, attendu aucun ecart et 3 liaisons (l unite est le corps lu)", p.cov)
	}
}

// TestUnSlotSansRecordDeCreationNeContredisRien : sans record de creation sur le slot, rien ne contredit l unite ; la liaison
// se fait comme avant.
func TestUnSlotSansRecordDeCreationNeContredisRien(t *testing.T) {
	occ := occupantsPlats(map[uint32]int{765: 19})
	p := lirePlacesDeControle([]grammar.FireEvent{tirDeLUnite(21, 765, 7, 100)}, occ, vsClock(), nil, true)
	if p.cov.UnitContradicted != 0 || p.cov.Bindings != 1 {
		t.Errorf("couverture = %+v, attendu 1 liaison et aucun ecart", p.cov)
	}
}

// TestLeTirDUneUniteContrediteResteAuPiloteDansLeDocument : par le vrai assemblage, le tir de l unite contredite reste a B dans le
// bilan de tirs publie, et l ecart est publie dans `coverage.controlPlaces.unitContradicted`. Sans la regle, C recevrait le tir.
func TestLeTirDUneUniteContrediteResteAuPiloteDansLeDocument(t *testing.T) {
	const arme = 0x00000000AB00CD00
	tir := func(slot, gen uint32, fr int) grammar.FireEvent {
		e := tirDeTest(1, uint8(fr), instantDeLaFrame(fr), arme)
		e.Unit = grammar.UnitRef{Present: true, Slot: slot, Gen: gen}
		return e
	}
	fire := []grammar.FireEvent{tir(10, 0, 10), tir(765, 2, 20), tir(10, 0, 30)}
	a := assemblageDeTouches(true, fire, nil)
	for i := range a.doc.Roster {
		a.doc.Roster[i].Seat = a.doc.Roster[i].FilmIndex
	}
	a.reg = IdentityRegistry{
		own:   OwnerReport{Owner: map[uint32]int{10: 1, 765: 2}},
		corps: corpsParSlot([]grammar.BipedCreation{creationGen(765, instantDeLaFrame(90), 2, 1)}),
	}
	a.opt.Fallbacks = fallback.NouveauCompteur()
	tirs := a.tirsDuCombat()
	a.doc.Coverage.ControlPlaces = &a.placesCov
	doc := documentJSON(t, a.doc)
	cp := doc["coverage"].(map[string]any)["controlPlaces"].(map[string]any)
	if cp["unitContradicted"] != float64(1) || cp["shotsByPlace"] != float64(3) {
		t.Errorf("controlPlaces = %v, attendu unitContradicted 1 et 3 tirs par la place lue", cp)
	}
	for _, f := range tirs {
		if f.FilmIndex != 1 {
			t.Errorf("un tir est rendu a l index %d, attendu 1 (B) pour les trois", f.FilmIndex)
		}
	}
}

// documentJSON rend le document tel qu il se publie : serialise, puis relu sans type.
func documentJSON(t *testing.T, doc ReplayDocument) map[string]any {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
