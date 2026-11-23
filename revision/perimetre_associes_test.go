package revision_test

// perimetre_associes_test.go — UNE TABLE DE DONNEES CONSOMMEE PAR LA PUBLICATION ENTRE DANS LA REVISION DE LA COUCHE
// QUI FAIT LIRE CE QU'ELLE NOMME ([revision.Couche.Associes]).
//
// Le cas : la table des reponses de jeu (`reponsesdejeu`) nomme et chiffre les actions de score que `grammar` fait lire.
// Seul le paquet de publication l'importe, si bien que la fermeture des imports de `grammar` ne la rencontrait pas :
// une ligne de plus ne changeait aucune revision, et aucun rejeu n'etait a redecoder.

import (
	"errors"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/revision"
)

// moduleAvecTable : le module de fixture, plus un paquet `table` (une source qui embarque un fichier de donnees) que la
// couche `a` n'importe pas. Les couches rendues sont celles de la fixture, `a` portant `associes`.
func moduleAvecTable(t *testing.T, donnees string, associes []string) (revision.Module, []revision.Couche) {
	t.Helper()
	m, couches := moduleDeFixture(t, map[string]string{
		"table/t.go":         "package table\n\nimport _ \"embed\"\n\n//go:embed data/t.tsv\nvar Brut string\n",
		"table/data/t.tsv":   donnees,
		"table/autre/aux.go": "package autre\n\nconst X = 1\n",
	})
	couches[0].Associes = associes
	return m, couches
}

func TestAssocies_LesDonneesDuPaquetAssocieEntrentDansLEmpreinte(t *testing.T) {
	empreinte := func(donnees string, associes []string) string {
		m, couches := moduleAvecTable(t, donnees, associes)
		res, _, err := m.CalculerCouche("a", couches, nil, map[string]string{"z": "z-1"})
		if err != nil {
			t.Fatalf("empreinte de a : %v", err)
		}
		return res.Empreinte
	}
	avec := empreinte("1\ta\n", []string{"table"})
	if got := empreinte("1\ta\n2\tb\n", []string{"table"}); got == avec {
		t.Error("une ligne s'ajoute a la table du paquet associe et l'empreinte de la couche ne bouge pas : " +
			"le rejeu ne serait jamais a redecoder")
	}
	if got := empreinte("1\ta\n", nil); got == avec {
		t.Error("declarer le paquet associe ne change pas l'empreinte : il n'entre pas dans le hachage")
	}
	if got := empreinte("1\ta\r\n", []string{"table"}); got != avec {
		t.Error("la meme table en CRLF rend une autre empreinte")
	}
}

func TestAssocies_LePaquetEntreDansLeCalculDeLaFermeture(t *testing.T) {
	m, couches := moduleAvecTable(t, "1\ta\n", []string{"table"})
	p, err := m.Fermeture("a", couches)
	if err != nil {
		t.Fatalf("fermeture : %v", err)
	}
	attendu := revision.Perimetre{
		Paquets: []string{"b", "c", "couches/a", "couches/a/sous", "table"},
		Amonts:  []string{"z"},
	}
	if !p.Egal(attendu) {
		t.Fatalf("fermeture de a = %+v, attendu %+v (le paquet associe, pas son sous-dossier que personne n'importe)", p, attendu)
	}
}

func TestAssocies_UneDeclarationFausseEchoueBruyamment(t *testing.T) {
	for quoi, associes := range map[string][]string{
		"un paquet d'une couche revisee": {"couches/z/zz"},
		"un dossier sans source":         {"table/data"},
		"un dossier absent":              {"fantome"},
	} {
		m, couches := moduleAvecTable(t, "1\ta\n", associes)
		_, err := m.Fermeture("a", couches)
		if !errors.Is(err, revision.ErrPerimetre) && !errors.Is(err, revision.ErrRacineSansSource) {
			t.Errorf("%s : err = %v, attendu une faute de declaration", quoi, err)
		}
	}
}

// TestAssocies_GrammarHacheLaTableDesReponsesDeJeu : dans le module reel, la fermeture de `grammar` contient le paquet de
// la table des reponses de jeu, et aucune autre couche ne le porte.
func TestAssocies_GrammarHacheLaTableDesReponsesDeJeu(t *testing.T) {
	m := moduleDeLaBrique(t)
	const table = "internal/games/halo_infinite/film/reponsesdejeu"
	for _, c := range revision.CouchesRevisees() {
		p, err := m.Fermeture(c.Nom, revision.CouchesRevisees())
		if err != nil {
			t.Fatalf("fermeture de %s : %v", c.Nom, err)
		}
		if got, want := slices.Contains(p.Paquets, table), c.Nom == "grammar"; got != want {
			t.Errorf("couche %s : la table des reponses de jeu dans sa fermeture = %v, attendu %v", c.Nom, got, want)
		}
	}
}
