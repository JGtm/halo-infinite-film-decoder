package replay

// document_apparence_test.go — L'APPARENCE DANS LE FICHIER DE FAITS ET DANS LE ROSTER, SUR DES OCTETS FABRIQUES : l'aller-retour du
// codec (listes vides comprises), le refus d'un flux tronque, la pose par xuid, la forme du JSON et les compteurs.

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func apparencesSynth() ([]types.PlayerAppearance, types.AppearanceStats) {
	as := []types.PlayerAppearance{
		{XUID: 9001, Size: 64, Pairs: []types.AppearancePair{{Key: 1, Value: 0xFFFFFFFF}, {Key: 2, Value: 3}}, Rest: []byte{1, 0, 2}},
		{XUID: 9002, Size: 64, Rest: []byte{9, 9}},
		{XUID: 9003, Size: 64, Pairs: []types.AppearancePair{{Key: 5, Value: 6}}},
	}
	return as, types.AppearanceStats{Scanned: true, Packets: 5, Distinct: 3, Refused: 1, NoXUID: 2, Pairs: 2, Raw: 1, Unloaded: 4, Changed: 1}
}

func TestApparencesFontLAllerRetour(t *testing.T) {
	as, st := apparencesSynth()
	for _, tc := range []struct {
		nom string
		as  []types.PlayerAppearance
	}{{"pleines", as}, {"vides", nil}} {
		w := &gwriter{}
		encodeApparences(w, tc.as, st)
		r := &greader{b: w.b}
		got, gst := decodeApparences(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("%s : relecture invalide (err %v, %d/%d octets)", tc.nom, r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(got, tc.as) || gst != st {
			t.Errorf("%s : aller-retour different\n %+v / %+v\n %+v / %+v", tc.nom, got, tc.as, gst, st)
		}
	}
}

func TestApparencesRefusentUnFluxTronque(t *testing.T) {
	as, st := apparencesSynth()
	w := &gwriter{}
	encodeApparences(w, as, st)
	for n := range len(w.b) {
		r := &greader{b: w.b[:n]}
		decodeApparences(r)
		if r.err == nil {
			t.Fatalf("un flux tronque a %d octet(s) sur %d doit echouer", n, len(w.b))
		}
	}
}

func TestPoserLesApparencesParXUID(t *testing.T) {
	as, st := apparencesSynth()
	a := &assemblage{ctx: context.Background()}
	a.doc.Coverage = new(Coverage)
	a.opt.Appearances, a.opt.AppearanceStats = as, st
	a.doc.Roster = []RosterEntry{{XUID: "9002"}, {XUID: ""}, {XUID: "9001"}}
	a.poserLesApparences()
	if a.doc.Roster[1].Appearance != nil {
		t.Error("un bot (xuid vide) ne porte pas d'apparence")
	}
	got := a.doc.Roster[2].Appearance
	if got == nil || got.Size != 64 || len(got.Pairs) != 2 || got.Pairs[0] != [2]uint32{1, 0xFFFFFFFF} || !reflect.DeepEqual(got.Rest, []byte{1, 0, 2}) {
		t.Errorf("apparence posee : %+v", got)
	}
	if raw := a.doc.Roster[0].Appearance; raw == nil || len(raw.Pairs) != 0 || !reflect.DeepEqual(raw.Rest, []byte{9, 9}) {
		t.Errorf("apparence brute : %+v", raw)
	}
	cov := a.apparenceCov
	if cov == nil || cov.Pairs != 2 || cov.Raw != 1 || cov.Unloaded != 4 || cov.Changed != 1 || cov.Unmatched != 1 {
		t.Errorf("couverture : %+v", cov)
	}
}

func TestPoserLesApparencesSansLectureNePoseRien(t *testing.T) {
	a := &assemblage{ctx: context.Background()}
	a.doc.Coverage = new(Coverage)
	a.doc.Roster = []RosterEntry{{XUID: "9001"}}
	a.poserLesApparences()
	if a.doc.Roster[0].Appearance != nil || a.apparenceCov != nil {
		t.Error("sans lecture, ni apparence ni couverture")
	}
}

func TestApparenceJSON(t *testing.T) {
	b, err := json.Marshal(RosterEntry{XUID: "1", Appearance: appearanceDuDocument(types.PlayerAppearance{
		Size: 64, Pairs: []types.AppearancePair{{Key: 7, Value: 8}}, Rest: []byte{1, 2, 3}})})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"appearance":{"size":64,"pairs":[[7,8]],"rest":"AQID"}`) {
		t.Errorf("forme du JSON : %s", b)
	}
	b, _ = json.Marshal(RosterEntry{XUID: "1"})
	if strings.Contains(string(b), "appearance") {
		t.Errorf("une apparence absente ne se publie pas : %s", b)
	}
}

// TestApparenceDepuisLesFaitsEgaleCelleDuFilm : le rejeu reconstruit depuis les faits (codec ecrit puis relu) pose EXACTEMENT
// la meme apparence et la meme couverture que le rejeu depuis le film.
func TestApparenceDepuisLesFaitsEgaleCelleDuFilm(t *testing.T) {
	as, st := apparencesSynth()
	w := &gwriter{}
	encodeApparences(w, as, st)
	r := &greader{b: w.b}
	asFaits, stFaits := decodeApparences(r)
	poser := func(as []types.PlayerAppearance, st types.AppearanceStats) ([]RosterEntry, *AppearanceCoverage) {
		a := &assemblage{ctx: context.Background()}
		a.doc.Coverage = new(Coverage)
		a.opt.Appearances, a.opt.AppearanceStats = as, st
		a.doc.Roster = []RosterEntry{{XUID: "9001"}, {XUID: "9002"}, {XUID: "9003"}, {XUID: ""}}
		a.poserLesApparences()
		return a.doc.Roster, a.apparenceCov
	}
	rf, cf := poser(as, st)
	rh, ch := poser(asFaits, stFaits)
	if !reflect.DeepEqual(rf, rh) || !reflect.DeepEqual(cf, ch) {
		t.Errorf("l apparence rejouee depuis les faits differe de celle du film :\n%+v / %+v\n%+v / %+v", rh, ch, rf, cf)
	}
}

// TestLaCouvertureDeLApparenceSePoseParLAssemblageComplet : par le chemin REEL de l'assemblage (`doc.Coverage` n'existe qu'a la
// composition de la couverture), la lecture faite pose `coverage.appearance` ; sans lecture, rien. Sans cette preuve, un test
// qui prepare lui-meme `doc.Coverage` laisserait passer une pose trop tot (panique sur un film reel).
func TestLaCouvertureDeLApparenceSePoseParLAssemblageComplet(t *testing.T) {
	as, st := apparencesSynth()
	doc := BuildFromPositions(context.Background(), "m", "halo_infinite", positionsPourVersion(), nil, Options{
		FrameIntervalMS: 100, Appearances: as, AppearanceStats: st})
	if doc.Coverage.Appearance == nil || doc.Coverage.Appearance.Pairs != 2 || doc.Coverage.Appearance.Unmatched != 3 {
		t.Errorf("couverture de l apparence : %+v", doc.Coverage.Appearance)
	}
	doc = BuildFromPositions(context.Background(), "m", "halo_infinite", positionsPourVersion(), nil, Options{FrameIntervalMS: 100})
	if doc.Coverage.Appearance != nil {
		t.Errorf("sans lecture, pas de couverture : %+v", doc.Coverage.Appearance)
	}
}

// TestAppearanceCoverageJSON : la couverture se publie en camelCase comme les autres, jamais sous le nom du champ Go.
func TestAppearanceCoverageJSON(t *testing.T) {
	b, err := json.Marshal(AppearanceCoverage{Pairs: 1, Unmatched: 2, UnknownBuild: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, cle := range []string{`"pairs":1`, `"raw":0`, `"unloaded":0`, `"changed":0`, `"unknownBuild":true`, `"packets":0`,
		`"distinct":0`, `"refused":0`, `"noXUID":0`, `"unmatched":2`} {
		if !strings.Contains(string(b), cle) {
			t.Errorf("la couverture publiee ne porte pas %s : %s", cle, b)
		}
	}
}
