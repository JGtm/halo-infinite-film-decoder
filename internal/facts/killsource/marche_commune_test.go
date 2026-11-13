package killsource

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// TestLaMarcheCommuneRendCeQueRendentLesDeuxMarches : sur la mini-bobine, un decodage a qui l appelant
// donne le contexte qu il partagera avec la cuisson rend le MEME resultat qu un decodage qui ouvre le
// sien ; et ce contexte, une fois le profil de la cuisson pose, rend a la cuisson la marche que rend un
// contexte neuf sous ce profil, avec et sans les morts d objet. Une seconde demande marche de nouveau,
// et rend la meme chose.
func TestLaMarcheCommuneRendCeQueRendentLesDeuxMarches(t *testing.T) {
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine : %v", err)
	}
	seul, err := Decode(t.Context(), miniBobineFilm, src, optionsDeLaBobine(t))
	if err != nil {
		t.Fatalf("decodage seul : %v", err)
	}
	for _, lectures := range []grammar.LecturesDeLaMarche{{}, {Morts: true}, {MortsSiVehicules: true}} {
		opt := *optionsDeLaBobine(t)
		opt.Contexte = grammar.NewFilmContextForMap(src, opt.Carte, nil)
		commun, err := Decode(t.Context(), miniBobineFilm, src, &opt)
		if err != nil {
			t.Fatalf("decodage a contexte partage : %v", err)
		}
		if !reflect.DeepEqual(seul, commun) {
			t.Fatalf("lectures %+v : le resultat du decodage a contexte partage differe de celui du decodage seul", lectures)
		}
		// Le contexte de la cuisson, tel qu elle le pose : le profil calibre, puis la carte et le decoupage.
		neuf := grammar.NewFilmContextForMap(src, opt.Carte, nil)
		for _, fc := range []*grammar.FilmContext{opt.Contexte, neuf} {
			fc.PoserProfilDeBalayage(commun.ProfilCalibre)
			fc.PoserLaCarteEtLeDecoupage()
		}
		if !opt.Contexte.GardeUneMarchePourLaCuisson() || neuf.GardeUneMarchePourLaCuisson() {
			t.Fatalf("lectures %+v : le contexte partage doit garder une marche sous le profil de la cuisson, le neuf aucune", lectures)
		}
		attendue, errNeuf := grammar.ScanMarcheDesTramesAvec(neuf, lectures)
		gardee, errGardee := grammar.ScanMarcheDesTramesAvec(opt.Contexte, lectures)
		if (errNeuf == nil) != (errGardee == nil) || !reflect.DeepEqual(attendue, gardee) {
			t.Fatalf("lectures %+v : la marche gardee differe de celle d un contexte neuf (erreurs %v / %v)",
				lectures, errGardee, errNeuf)
		}
		if !reflect.DeepEqual(neuf.ComptesDesReplis(), opt.Contexte.ComptesDesReplis()) {
			t.Fatalf("lectures %+v : rapport de replis du contexte partage %+v, d un contexte neuf %+v",
				lectures, opt.Contexte.ComptesDesReplis(), neuf.ComptesDesReplis())
		}
		if opt.Contexte.GardeUneMarchePourLaCuisson() {
			t.Fatalf("lectures %+v : la marche gardee se rend une seule fois", lectures)
		}
		encore, _ := grammar.ScanMarcheDesTramesAvec(opt.Contexte, lectures)
		if !reflect.DeepEqual(attendue, encore) {
			t.Fatalf("lectures %+v : une seconde demande au contexte partage rend une autre marche", lectures)
		}
	}
}
