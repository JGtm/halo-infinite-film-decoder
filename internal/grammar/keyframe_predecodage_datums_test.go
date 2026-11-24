package grammar

// keyframe_predecodage_datums_test.go — les deux taches du pre-passage qui rangent la table des datums
// et les en-tetes exacts de joueur : meme resultat a tout nombre de fils, memo plein seulement a plus
// d'un fil.

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// contexteDeBobineEquipes rend un contexte neuf, regle a `n` fils, de la mini-bobine d'equipes `film`.
func contexteDeBobineEquipes(t *testing.T, film string, n int) *FilmContext {
	t.Helper()
	f, err := source.LoadDir(filepath.Join("..", "..", "replay", "testdata", "minifilm_"+film), nil)
	if err != nil {
		t.Fatalf("bobine %s : %v", film, err)
	}
	fc := NewFilmContext(f)
	fc.PoserFils(n)
	return fc
}

// TestFilsDatumsIdentiques : la table des datums rendue par le chemin normal est la meme a 1, 2, 4 et 8
// fils ; le memo des tables est plein apres le premier acces aux images-cles a plus d'un fil seulement.
func TestFilsDatumsIdentiques(t *testing.T) {
	for _, nom := range bobinesAEmpreinte {
		film := chargerUneBobine(t, nom)
		un := contexteAFils(film, 1)
		payloads := un.payloadsDesImagesCles()
		if len(payloads) == 0 {
			t.Fatalf("%s : aucune image-cle, le temoin ne prouve rien", nom)
		}
		type lecture struct {
			table   map[uint32]EnteteDeTable
			ambigus int
		}
		ref := make([]lecture, len(payloads))
		for i, pay := range payloads {
			ref[i].table, ref[i].ambigus = un.tableDeDatumsDe(pay)
		}
		for _, n := range []int{1, 2, 4, 8} {
			fc := contexteAFils(film, n)
			fc.MarcheDImageCle()
			if plein := len(fc.recup.tables) > 0; plein != (n > 1) {
				t.Errorf("%s, %d fils : memo des tables plein apres le premier acces = %v", nom, n, plein)
			}
			for i, pay := range fc.payloadsDesImagesCles() {
				table, amb := fc.tableDeDatumsDe(pay)
				if amb != ref[i].ambigus || !reflect.DeepEqual(table, ref[i].table) {
					t.Errorf("%s, %d fils, payload %d : table differente de celle d'un fil", nom, n, i)
				}
			}
		}
	}
}

// TestFilsEntetesExactsIdentiques : les en-tetes exacts rendus par le chemin normal sont les memes a 1, 2, 4
// et 8 fils, et le resultat complet de [ScanPlayerTeams] aussi ; le memo des en-tetes est plein seulement
// a plus d'un fil.
func TestFilsEntetesExactsIdentiques(t *testing.T) {
	for _, b := range bobinesEquipes() {
		un := contexteDeBobineEquipes(t, b.film, 1)
		payloads := un.payloadsDesImagesCles()
		if len(payloads) == 0 {
			t.Fatalf("%s : aucune image-cle, le temoin ne prouve rien", b.film)
		}
		teamsRef, repRef, entsRef := ScanPlayerTeams(un)
		if len(un.recup.entetes) != 0 {
			t.Fatalf("%s : a un fil le memo des en-tetes doit rester vide", b.film)
		}
		for _, n := range []int{1, 2, 4, 8} {
			fc := contexteDeBobineEquipes(t, b.film, n)
			fc.MarcheDImageCle()
			if plein := len(fc.recup.entetes) > 0; plein != (n > 1) {
				t.Errorf("%s, %d fils : memo des en-tetes plein apres le premier acces = %v", b.film, n, plein)
			}
			for i, pay := range fc.payloadsDesImagesCles() {
				if got, want := fc.entetesExactsDe(pay), slotsDEntetesExacts(payloads[i]); !reflect.DeepEqual(got, want) {
					t.Errorf("%s, %d fils, payload %d : en-tetes %v, attendu %v", b.film, n, i, got, want)
				}
			}
			teams, rep, ents := ScanPlayerTeams(fc)
			if !reflect.DeepEqual(teams, teamsRef) || !reflect.DeepEqual(rep, repRef) || !reflect.DeepEqual(ents, entsRef) {
				t.Errorf("%s, %d fils : ScanPlayerTeams differe de celui d'un fil", b.film, n)
			}
		}
	}
}
