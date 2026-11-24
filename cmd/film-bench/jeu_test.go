package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ecrire(t *testing.T, dir, nom, contenu string) string {
	t.Helper()
	p := filepath.Join(dir, nom)
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLireJeuIgnoreEnteteEtCommentaires(t *testing.T) {
	p := ecrire(t, t.TempDir(), "films.tsv",
		"# commentaire\nnom\tmode\tversion\tjoueurs\tmorceaux\n\nfilm01\tSlayer\t41 / HI_1_13_0\t8\t25\nfilm02\tBases\t40\t12\t30\n")
	jeu, err := lireJeu(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(jeu) != 2 || jeu[0].Nom != "film01" || jeu[0].Joueurs != 8 || jeu[1].Morceaux != 30 || jeu[1].Mode != "Bases" {
		t.Errorf("jeu lu : %+v", jeu)
	}
}

func TestLireJeuRefuseLesLignesMalFormees(t *testing.T) {
	cas := map[string]string{
		"colonnes":  "film01\tSlayer\t41\t8\n",
		"joueurs":   "film01\tSlayer\t41\thuit\t25\n",
		"morceaux":  "film01\tSlayer\t41\t8\t0\n",
		"doublon":   "film01\tSlayer\t41\t8\t25\nfilm01\tBases\t41\t8\t25\n",
		"aucunfilm": "# rien\n",
	}
	for nom, contenu := range cas {
		p := ecrire(t, t.TempDir(), "films.tsv", contenu)
		if _, err := lireJeu(p); err == nil {
			t.Errorf("%s : une erreur etait attendue", nom)
		}
	}
	if _, err := lireJeu(filepath.Join(t.TempDir(), "absent.tsv")); err == nil {
		t.Error("fichier absent : une erreur etait attendue")
	}
}

func TestLireCorrespondanceResoutLesCheminsRelatifs(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "ailleurs", "m.json")
	p := ecrire(t, dir, "corr.tsv",
		"# local\nfilm01\tdossiers/a\tmanifestes/a.json\tFortress\nfilm02\t"+abs+"\t"+abs+"\tsgh_streets\n")
	cors, err := lireCorrespondance(p)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cors["film01"].Dossier, filepath.Join(dir, "dossiers", "a"); got != want {
		t.Errorf("dossier relatif : %q, attendu %q", got, want)
	}
	if got := cors["film02"].Manifeste; got != filepath.Clean(abs) {
		t.Errorf("chemin absolu change : %q", got)
	}
	if cors["film01"].Carte != "Fortress" {
		t.Errorf("carte : %q", cors["film01"].Carte)
	}
}

func TestLireCorrespondanceRefuseLesLignesMalFormees(t *testing.T) {
	cas := map[string]string{
		"colonnes": "film01\ta\tb\n",
		"carte":    "film01\ta\tb\t \n",
		"doublon":  "film01\ta\tb\tc\nfilm01\ta\tb\tc\n",
	}
	for nom, contenu := range cas {
		if _, err := lireCorrespondance(ecrire(t, t.TempDir(), "c.tsv", contenu)); err == nil {
			t.Errorf("%s : une erreur etait attendue", nom)
		}
	}
}

func TestVerifierCheminsRefuseUnCheminManquant(t *testing.T) {
	dir := t.TempDir()
	manifeste := ecrire(t, dir, "m.json", "{}")
	ok := Correspondance{Nom: "film01", Dossier: dir, Manifeste: manifeste}
	if err := verifierChemins(ok); err != nil {
		t.Fatalf("chemins valides refuses : %v", err)
	}
	for nom, c := range map[string]Correspondance{
		"dossier":   {Nom: "film01", Dossier: filepath.Join(dir, "absent"), Manifeste: manifeste},
		"manifeste": {Nom: "film01", Dossier: dir, Manifeste: filepath.Join(dir, "absent.json")},
		"fichier":   {Nom: "film01", Dossier: manifeste, Manifeste: manifeste},
	} {
		err := verifierChemins(c)
		if err == nil {
			t.Errorf("%s manquant : une erreur etait attendue", nom)
			continue
		}
		if strings.Contains(err.Error(), dir) {
			t.Errorf("%s : le message ne doit pas porter de chemin : %v", nom, err)
		}
	}
}

func TestRetenirFiltreDansLOrdreDuJeu(t *testing.T) {
	jeu := []Film{{Nom: "film01"}, {Nom: "film02"}, {Nom: "film03"}}
	got, err := retenir(jeu, "film03, film01")
	if err != nil || len(got) != 2 || got[0].Nom != "film01" || got[1].Nom != "film03" {
		t.Errorf("retenu : %+v, %v", got, err)
	}
	if all, _ := retenir(jeu, ""); len(all) != 3 {
		t.Errorf("sans filtre : %d films", len(all))
	}
	if _, err := retenir(jeu, "film09"); err == nil {
		t.Error("film inconnu : une erreur etait attendue")
	}
}

func TestLireManifesteGardeIndexTypeEtDebut(t *testing.T) {
	p := ecrire(t, t.TempDir(), "m.json",
		`{"chunks":[{"index":0,"chunk_type":1,"start_ms":0},{"index":3,"chunk_type":2,"start_ms":19984}],"autre":1}`)
	meta, err := lireManifeste(p)
	if err != nil || len(meta) != 2 || meta[1].Index != 3 || meta[1].ChunkType != 2 || meta[1].StartMS != 19984 {
		t.Errorf("manifeste : %+v, %v", meta, err)
	}
	if _, err := lireManifeste(ecrire(t, t.TempDir(), "x.json", "pas du json")); err == nil {
		t.Error("manifeste invalide : une erreur etait attendue")
	}
	if _, err := lireManifeste(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Error("manifeste absent : une erreur etait attendue")
	}
}
