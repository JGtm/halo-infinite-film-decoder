package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestEmpreinteDeValeurDeterministe(t *testing.T) {
	type v struct {
		A []int
		M map[string]int
	}
	a := empreinteDeValeur(v{A: []int{1, 2}, M: map[string]int{"b": 2, "a": 1}})
	b := empreinteDeValeur(v{A: []int{1, 2}, M: map[string]int{"a": 1, "b": 2}})
	c := empreinteDeValeur(v{A: []int{2, 1}, M: map[string]int{"a": 1, "b": 2}})
	if !a.Serialisee || a.Empreinte != b.Empreinte || a.Empreinte == c.Empreinte || len(a.Empreinte) != 64 {
		t.Errorf("empreintes : %+v %+v %+v", a, b, c)
	}
	e := empreinteDeValeur(func() {})
	if e.Serialisee || !strings.HasPrefix(e.Empreinte, "erreur:") {
		t.Errorf("valeur non serialisable : %+v", e)
	}
}

func TestEcrireSuiteGardeLOrdre(t *testing.T) {
	var buf bytes.Buffer
	suite := []Etape{{Nom: "zeta", Empreinte: "11"}, {Nom: "alpha", Empreinte: "22"}, {Nom: "zeta", Empreinte: "33"}}
	if err := ecrireSuite(&buf, "film01", suite); err != nil {
		t.Fatal(err)
	}
	want := "film01\t1\tzeta\t11\nfilm01\t2\talpha\t22\nfilm01\t3\tzeta\t33\n"
	if buf.String() != want {
		t.Errorf("suite :\n%s\nattendu :\n%s", buf.String(), want)
	}
}

func TestLireOptionsFilsEtEtapes(t *testing.T) {
	base := []string{"-correspondance", "c.tsv", "-catalogue", "cat.json"}
	o, err := lireOptions(base, io.Discard)
	if err != nil || o.fils != 1 || o.etapes != "" || o.vitesse().Fils != 1 {
		t.Fatalf("defauts : %v %+v", err, o)
	}
	o, err = lireOptions(append(append([]string{}, base...), "-fils", "4", "-etapes", "e.tsv"), io.Discard)
	if err != nil || o.fils != 4 || o.etapes != "e.tsv" || o.vitesse().Fils != 4 {
		t.Fatalf("valeurs : %v %+v", err, o)
	}
	refus := map[string][]string{
		"zero fil":         {"-fils", "0"},
		"fils negatif":     {"-fils", "-2"},
		"etapes et fils":   {"-etapes", "e.tsv", "-enfant"},
		"etapes et profil": {"-etapes", "e.tsv", "-profil-cpu", "p", "-seulement", "a"},
	}
	for nom, extra := range refus {
		if _, err := lireOptions(append(append([]string{}, base...), extra...), io.Discard); err == nil {
			t.Errorf("%s : une erreur etait attendue", nom)
		}
	}
}
