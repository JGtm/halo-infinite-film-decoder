//go:build research

package divergence

// versions_research_test.go : LA TABLE DES VERSIONS D EVENEMENTS QUE LE FILM PORTE (`chunk_00 + 0xCB208`,
// 123 u32) contre la table native du jeu (`DAT_14474cd90`), film par film (plan marche complete, A2.V).
//
//	VERSIONS_RACINE=<film_chunks> VERSIONS_FILMS=<liste> go test -tags=research ./research/divergence/ -run TestVersions -v

import (
	"fmt"
	"os"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// versionsNatives : la colonne version de `DAT_14474cd90`, recopiee de grammar/vue_a_versions.go.
const versionsNatives = "" +
	"1111111111" + "1111111121" + "1111111111" + "3111143111" + "2111112121" +
	"1111112111" + "2611111112" + "1111111111" + "1515111112" + "3312111411" +
	"1111111311" + "1111111111" + "111"

func TestVersions(t *testing.T) {
	racine, liste := os.Getenv("VERSIONS_RACINE"), os.Getenv("VERSIONS_FILMS")
	if racine == "" || liste == "" {
		t.Skip("VERSIONS_RACINE et VERSIONS_FILMS requis")
	}
	films, err := lireLesFilms(liste)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range films {
		fc, err := ouvrirFilm(racine, f.id, f.niveau, nil)
		if err != nil {
			t.Fatalf("%s : %v", f.id, err)
		}
		p := fc.Profile()
		maj := p.Highlight()
		if !p.IdentityRead() {
			fmt.Printf("%s %s major=%#x(lue=%v) identite non lue\n", f.id, f.build, maj.MajorVersion, maj.Lue)
			continue
		}
		tv := p.Identity().TypeVersions
		diffs := ""
		for i, v := range tv {
			if i < len(versionsNatives) && v != uint32(versionsNatives[i]-'0') {
				diffs += fmt.Sprintf(" g%d=%d(natif %c)", i, v, versionsNatives[i])
			}
		}
		fmt.Printf("%s %s build=%q major=%#x(lue=%v) genres=%d diffs_vs_natif=[%s]\n", f.id, f.build, p.Build(), maj.MajorVersion, maj.Lue, len(tv), diffs)
	}
	_ = grammar.GenresVueA
}
