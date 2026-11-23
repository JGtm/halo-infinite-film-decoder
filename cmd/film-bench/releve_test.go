package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func resultatDeTest() ResultatFilm {
	stats := map[Phase]Stat{}
	for i, ph := range ordreDesPhases {
		stats[ph] = Stat{Mediane: ms10(10 * (i + 1)), Min: ms10(9 * (i + 1)), Max: ms10(12 * (i + 1)), Passes: 5}
	}
	return ResultatFilm{
		Film: Film{Nom: "film01", Mode: "Slayer", Version: "41 / HI_1_13_0", Joueurs: 8, Morceaux: 25},
		Fils: 1, Stats: stats, PicMio: 262.4, EmpreinteDoc: "aa", EmpreinteFaits: "bb",
	}
}

func TestEcrireReleveFormat(t *testing.T) {
	var buf bytes.Buffer
	e := Entete{
		Date: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC), Arbre: "v1.5.0",
		Machine: Machine{Processeur: "CPU test", Coeurs: 16, MemoireMio: 32000, Systeme: "windows/amd64", Go: "go1.26.11", GoMaxProcs: 16},
		Passes:  5, PlafondGiB: 2,
	}
	if err := ecrireReleve(&buf, e, []ResultatFilm{resultatDeTest()}); err != nil {
		t.Fatal(err)
	}
	lignes := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	var donnees []string
	entete := map[string]string{}
	for _, l := range lignes {
		if k, v, ok := strings.Cut(strings.TrimPrefix(l, "# "), "\t"); ok && strings.HasPrefix(l, "#") {
			entete[k] = v
			continue
		}
		if !strings.HasPrefix(l, "#") {
			donnees = append(donnees, l)
		}
	}
	for cle, attendu := range map[string]string{"date": "2026-11-23T10:00:00Z", "arbre": "v1.5.0", "go": "go1.26.11",
		"systeme": "windows/amd64", "processeur": "CPU test", "coeurs": "16", "memoire_mio": "32000", "gomaxprocs": "16"} {
		if entete[cle] != attendu {
			t.Errorf("en-tete %s = %q, attendu %q", cle, entete[cle], attendu)
		}
	}
	if len(donnees) != 1+len(ordreDesPhases) || donnees[0] != colonnesDuReleve {
		t.Fatalf("%d lignes de donnees, colonnes %q", len(donnees), donnees[0])
	}
	nbCol := len(strings.Split(colonnesDuReleve, "\t"))
	for i, ph := range ordreDesPhases {
		cols := strings.Split(donnees[1+i], "\t")
		if len(cols) != nbCol || cols[0] != "film01" || cols[6] != string(ph) || cols[11] != "262" ||
			cols[12] != "aa" || cols[13] != "bb" || cols[5] != "1" || cols[10] != "5" {
			t.Errorf("ligne %d : %q", i, donnees[1+i])
		}
	}
	if cols := strings.Split(donnees[1], "\t"); cols[7] != "10.00" || cols[8] != "9.00" || cols[9] != "12.00" {
		t.Errorf("mediane, min, max de la decompression : %q", donnees[1])
	}
}

func TestEnteteNePorteAucunChemin(t *testing.T) {
	var buf bytes.Buffer
	if err := ecrireReleve(&buf, Entete{Date: time.Now(), Arbre: "inconnu", Machine: releverMachine()}, nil); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(buf.String(), `\`) || strings.Contains(buf.String(), "Users") {
		t.Errorf("l'en-tete laisse passer un chemin : %s", buf.String())
	}
}

func TestEcrireResumeLitLesMedianes(t *testing.T) {
	var buf bytes.Buffer
	if err := ecrireResume(&buf, []ResultatFilm{resultatDeTest()}); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"film01", "50.00", "262"} {
		if !strings.Contains(s, want) {
			t.Errorf("le resume ne contient pas %q :\n%s", want, s)
		}
	}
}
