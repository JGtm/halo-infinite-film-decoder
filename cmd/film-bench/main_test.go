package main

import (
	"io"
	"strings"
	"testing"
)

func TestLireOptionsDefautsEtRefus(t *testing.T) {
	o, err := lireOptions([]string{"-correspondance", "c.tsv", "-catalogue", "cat.json"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if o.passes != 5 || o.plafondGiB != 2 || o.films != "bench/films.tsv" || o.enfant {
		t.Errorf("defauts : %+v", o)
	}
	refus := map[string][]string{
		"sans correspondance": {"-catalogue", "cat.json"},
		"sans catalogue":      {"-correspondance", "c.tsv"},
		"zero passe":          {"-correspondance", "c", "-catalogue", "k", "-passes", "0"},
		"plafond nul":         {"-correspondance", "c", "-catalogue", "k", "-plafond-gio", "0"},
		"fils multi-films":    {"-correspondance", "c", "-catalogue", "k", "-enfant", "-seulement", "a,b"},
		"drapeau inconnu":     {"-correspondance", "c", "-catalogue", "k", "-threads", "4"},
	}
	for nom, args := range refus {
		if _, err := lireOptions(args, io.Discard); err == nil {
			t.Errorf("%s : une erreur etait attendue", nom)
		}
	}
}

func TestRefuserSousData(t *testing.T) {
	for _, ok := range []string{"bench/releves/v1.5.0.tsv", "sortie/essai.tsv", "database/x.tsv"} {
		if err := refuserSousData(ok); err != nil {
			t.Errorf("%s refuse : %v", ok, err)
		}
	}
	for _, ko := range []string{"data/x.tsv", "a/Data/b/x.tsv", `a\data\x.tsv`} {
		if err := refuserSousData(ko); err == nil {
			t.Errorf("%s accepte", ko)
		}
	}
}

func TestLireRapportPrendLaDerniereLigne(t *testing.T) {
	r, err := lireRapport([]byte("bruit\n{\"pic_octets\":42}\n"))
	if err != nil || r.PicOctets != 42 {
		t.Fatalf("la derniere ligne fait foi : %+v, %v", r, err)
	}
	r, err = lireRapport([]byte("{\"pic_octets\":42,\"empreinte_document\":\"aa\",\"empreinte_faits\":\"bb\"}\n"))
	if err != nil || r.PicOctets != 42 || r.EmpreinteDoc != "aa" || r.EmpreinteFaits != "bb" || r.Depasse {
		t.Errorf("rapport : %+v, %v", r, err)
	}
	r, err = lireRapport([]byte("{\"pic_octets\":7,\"depasse\":true}"))
	if err != nil || !r.Depasse || r.PicOctets != 7 {
		t.Errorf("depassement : %+v, %v", r, err)
	}
	for _, vide := range []string{"", "\n", "pas du json"} {
		if _, err := lireRapport([]byte(vide)); err == nil {
			t.Errorf("%q : une erreur etait attendue", vide)
		}
	}
}

func TestDernieresLignes(t *testing.T) {
	got := dernieresLignes("a\n\nb\nc\nd\n", 2)
	if got != "c\nd" || strings.Contains(got, "a") {
		t.Errorf("dernieres lignes : %q", got)
	}
}

func TestLireOptionsProfil(t *testing.T) {
	base := []string{"-correspondance", "c.tsv", "-catalogue", "cat.json"}
	o, err := lireOptions(append(append([]string{}, base...), "-seulement", "film01", "-profil-cpu", "p.prof"), io.Discard)
	if err != nil || !o.profilDemande() || o.profilCPU != "p.prof" {
		t.Fatalf("profil valide refuse : %v %+v", err, o)
	}
	refus := map[string][]string{
		"sans film":    {"-profil-tas", "t.prof"},
		"plusieurs":    {"-profil-cpu", "p", "-seulement", "a,b"},
		"avec le fils": {"-profil-cpu", "p", "-seulement", "a", "-enfant"},
	}
	for nom, extra := range refus {
		if _, err := lireOptions(append(append([]string{}, base...), extra...), io.Discard); err == nil {
			t.Errorf("%s : une erreur etait attendue", nom)
		}
	}
	if o, _ := lireOptions(base, io.Discard); o.profilDemande() {
		t.Error("profil demande sans option")
	}
}
