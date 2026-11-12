package decfilm_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// chunk0DeLaBobine lit le `chunk_00` de la bobine d un film du module.
func chunk0DeLaBobine(t *testing.T, film string) []byte {
	t.Helper()
	d, err := decfilm.ReadFilmChunk("../replay/testdata/minifilm_"+film, 0)
	if err != nil {
		t.Fatalf("bobine %s : %v", film, err)
	}
	return d
}

// TestReadMatchInfoRendLesInfosDuFilm : la facade rend les infos de match d un `chunk_00` lu.
// MUTATIONS — un champ de l identite non rendu : ROUGE.
func TestReadMatchInfoRendLesInfosDuFilm(t *testing.T) {
	infos, err := decfilm.ReadMatchInfo(chunk0DeLaBobine(t, "fb1a1a72"))
	if err != nil {
		t.Fatal(err)
	}
	if !infos.VarianteLue || infos.Nom == nil || *infos.Nom != "Capture the Flag" ||
		infos.ScoreDeVictoire == nil || *infos.ScoreDeVictoire != 3 ||
		infos.DureeSecondes == nil || *infos.DureeSecondes != 720 ||
		infos.DebutUnix == nil || *infos.DebutUnix != 1784839686 || infos.CarteAssetID == nil {
		t.Errorf("infos de match : %+v", infos)
	}
}

// TestReadMatchInfoRendDesErreursTypees : un `chunk_00` coupe, ou sans section d identification, rend une
// erreur typee et aucune info ; jamais une panique.
// MUTATIONS — l erreur avalee, une valeur rendue avec l erreur : ROUGES.
func TestReadMatchInfoRendDesErreursTypees(t *testing.T) {
	d := chunk0DeLaBobine(t, "fb1a1a72")
	infos, err := decfilm.ReadMatchInfo(d[:100])
	if !errors.Is(err, decfilm.ErrChunk00Truncated) || infos.VarianteLue || infos.DebutUnix != nil {
		t.Errorf("chunk coupe : %v %+v", err, infos)
	}
	sans := bytes.Clone(d)
	at := bytes.Index(sans, []byte("HI_1_13_0"))
	if at < 0 {
		t.Fatal("la bobine ne porte pas sa chaine de build")
	}
	clear(sans[at : at+len("HI_1_13_0")])
	infos, err = decfilm.ReadMatchInfo(sans)
	if !errors.Is(err, decfilm.ErrNoFilmIdentity) || infos.VarianteLue || infos.DebutUnix != nil {
		t.Errorf("sans identification : %v %+v", err, infos)
	}
}
