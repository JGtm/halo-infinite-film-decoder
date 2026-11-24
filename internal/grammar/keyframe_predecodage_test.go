package grammar

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// marchesDeToutesLesImagesCles marche, par le contexte, chaque payload d'image-cle du film et rend
// les marches dans l'ordre du flux.
func marchesDeToutesLesImagesCles(fc *FilmContext) []MarcheDePayload {
	m := fc.MarcheDImageCle()
	var out []MarcheDePayload
	for _, pay := range fc.payloadsDesImagesCles() {
		out = append(out, m.Marcher(pay))
	}
	return out
}

// contexteAFils rend un contexte neuf de la bobine, regle a `n` fils.
func contexteAFils(film *source.Film, n int) *FilmContext {
	fc := contexteDeBobine(film)
	fc.PoserFils(n)
	return fc
}

// TestFilsAncresIdentiquesATousLesFils : sur chaque bobine, toutes les marches rendues par le contexte
// (records, ecartes, decisions) sont egales a 1, 2, 4 et 8 fils ; le pre-passage a rempli le memo
// avant le premier acces rendu, et une marche rend toujours une copie.
func TestFilsAncresIdentiquesATousLesFils(t *testing.T) {
	for _, nom := range bobinesAEmpreinte {
		film := chargerUneBobine(t, nom)
		ref := marchesDeToutesLesImagesCles(contexteAFils(film, 1))
		if len(ref) == 0 {
			t.Fatalf("%s : aucune image-cle, le temoin ne prouve rien", nom)
		}
		for _, n := range []int{1, 2, 4, 8} {
			fc := contexteAFils(film, n)
			m := fc.MarcheDImageCle()
			distincts := map[clePayload]bool{}
			for _, pay := range fc.payloadsDesImagesCles() {
				distincts[clePayload{debut: &pay[0], n: len(pay)}] = true
			}
			if plein := len(fc.marches.parPayload) == len(distincts); plein != (n > 1) {
				t.Errorf("%s, %d fils : memo plein apres le premier acces = %v (%d entrees pour %d payloads)",
					nom, n, plein, len(fc.marches.parPayload), len(distincts))
			}
			for i, pay := range fc.payloadsDesImagesCles() {
				mp := m.Marcher(pay)
				if !reflect.DeepEqual(mp, ref[i]) {
					t.Errorf("%s, %d fils, payload %d : marche differente de celle d'un fil\n  %+v\n  %+v", nom, n, i, mp, ref[i])
					continue
				}
				// Une marche rend une copie : abimer la tranche rendue ne touche pas le memo.
				if len(mp.Records) > 0 {
					mp.Records[0].Slot = -12345
					if again := m.Marcher(pay); !reflect.DeepEqual(again, ref[i]) {
						t.Errorf("%s, %d fils, payload %d : le memo partage sa tranche avec l'appelant", nom, n, i)
					}
				}
			}
		}
	}
}

// TestFilsAUnFilAucuneGoroutine : a un fil (ou moins), le premier acces ne pre-calcule rien et ne
// lance rien ; a plusieurs fils, le meme acces pre-calcule le memo.
func TestFilsAUnFilAucuneGoroutine(t *testing.T) {
	film := chargerUneBobine(t, bobinesAEmpreinte[0])
	for _, n := range []int{-1, 0, 1} {
		fc := contexteAFils(film, n)
		avant := runtime.NumGoroutine()
		m := fc.MarcheDImageCle()
		if got := len(fc.marches.parPayload); got != 0 {
			t.Fatalf("%d fil : %d marches pre-calculees, aucune attendue", n, got)
		}
		for _, pay := range fc.payloadsDesImagesCles() {
			m.Marcher(pay)
			if g := runtime.NumGoroutine(); g > avant {
				t.Fatalf("%d fil : %d goroutines apres une marche, %d avant", n, g, avant)
			}
		}
	}
	fc := contexteAFils(film, 4)
	fc.MarcheDImageCle()
	if len(fc.marches.parPayload) == 0 {
		t.Fatal("4 fils : le pre-passage n'a rien range, le temoin ne discrimine rien")
	}
}

// TestFilsLaPreuveEstFigeeAuPremierAcces : le pre-passage ne part ni a la pose des fils ni avant le
// premier acces ; la preuve qu'il emploie est celle du premier acces, la meme qu'a un fil, et un
// reglage du contexte posterieur ne change plus aucune marche.
func TestFilsLaPreuveEstFigeeAuPremierAcces(t *testing.T) {
	film := chargerUneBobine(t, bobinesAEmpreinte[1])
	un, huit := contexteAFils(film, 1), contexteAFils(film, 8)
	if huit.marches != nil {
		t.Fatal("PoserFils ne doit pas creer le memo : le pre-passage part au premier acces")
	}
	mUn, mHuit := un.MarcheDImageCle(), huit.MarcheDImageCle()
	if mUn.preuve == nil || mHuit.preuve == nil {
		t.Fatal("la bobine n'a pas de preuve : le temoin ne prouve rien")
	}
	if !reflect.DeepEqual(mUn.preuve.ctx, mHuit.preuve.ctx) {
		t.Fatal("la preuve a 8 fils n'est pas celle d'un fil")
	}
	if huit.MarcheDImageCle().preuve != mHuit.preuve {
		t.Fatal("la preuve change entre deux acces")
	}
	// Apres le premier acces, regler le contexte (MPP) ne change ni la preuve ni les marches.
	for _, fc := range []*FilmContext{un, huit} {
		fc.PoserMPP(profile.MPPWidths{})
	}
	a, b := marchesDeToutesLesImagesCles(un), marchesDeToutesLesImagesCles(huit)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("les marches d'un fil et de 8 fils different apres un reglage du contexte")
	}
}
