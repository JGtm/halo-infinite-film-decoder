package grammar

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// dossierDeLaMiniBobine : un prefixe de vrais chunks (registre compris) dont la table range chaque occupant a la ligne
// du siege que les paquets de type 8 lui donnent.
const dossierDeLaMiniBobine = "../facts/killsource/testdata/minibobine_000d5950"

func bitDe(buf []byte, pos int) byte { return buf[pos>>3] >> (7 - pos&7) & 1 }

func poserBit(buf []byte, pos int, v byte) {
	m := byte(1) << (7 - pos&7)
	if v == 0 {
		buf[pos>>3] &^= m
		return
	}
	buf[pos>>3] |= m
}

// echangerLesLignes echange en place, dans le registre, les octets de deux lignes de la table de meme longueur.
func echangerLesLignes(buf []byte, a, b types.PlayerSlot) {
	for i := range a.TotalBits {
		x, y := bitDe(buf, a.Bit+i), bitDe(buf, b.Bit+i)
		poserBit(buf, a.Bit+i, y)
		poserBit(buf, b.Bit+i, x)
	}
}

// bobineAuRangPermute charge la mini-bobine et permute dans son registre deux lignes de la table : l'occupant de
// chaque ligne est range a la place de l'autre, alors que les paquets de type 8 gardent leurs sieges. C'est un film
// dont le rang de la table n'est pas le siege liste, sans aucun octet de fixture de plus. Rend le film et les deux
// xuids permutes.
func bobineAuRangPermute(t *testing.T) (film *source.Film, xa, xb uint64) {
	t.Helper()
	film, err := source.LoadDir(dossierDeLaMiniBobine, nil)
	if err != nil {
		t.Fatal(err)
	}
	chunk0, ok := FilmRegistryChunk(film)
	if !ok {
		t.Fatal("mini-bobine sans registre")
	}
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		t.Fatal(err)
	}
	slots, _, err := ReadPlayerTable(chunk0, ident)
	if err != nil {
		t.Fatal(err)
	}
	occupes := slots[:0:0]
	for _, s := range slots {
		if s.XUID != 0 {
			occupes = append(occupes, s)
		}
	}
	if len(occupes) < 2 || occupes[0].TotalBits != occupes[1].TotalBits {
		t.Fatalf("deux lignes de meme longueur attendues, %d occupees", len(occupes))
	}
	echangerLesLignes(chunk0, occupes[0], occupes[1])
	return film, occupes[0].XUID, occupes[1].XUID
}

// rangsParXUID rend, par xuid, l'index que la table donne a son occupant.
func rangsParXUID(t FilmPlayerTable) map[uint64]int {
	out := map[uint64]int{}
	for _, s := range t.Seats {
		out[s.XUID] = s.FilmIndex
	}
	return out
}

// TestLaTableLueAuxListagesRendLeSiegeListe : sur un film dont deux lignes de la table sont permutees, la table
// brute donne a chaque occupant le rang de l'autre ; la lecture que tous les lecteurs emploient lui rend le siege que
// les paquets de type 8 listent, compte les deux sieges et ne laisse aucun vacant a departager.
// MUTATIONS — la correction retiree de la lecture, le rang gardé : ROUGES.
func TestLaTableLueAuxListagesRendLeSiegeListe(t *testing.T) {
	film, xa, xb := bobineAuRangPermute(t)
	brute, err := ScanFilmPlayerTable(film)
	if err != nil {
		t.Fatal(err)
	}
	lu, err := ScanFilmPlayerTableAuxListages(film)
	if err != nil {
		t.Fatal(err)
	}
	listes := map[uint64]int{}
	for _, l := range lu.Listages {
		listes[l.XUID] = l.Slot
	}
	rb, rl := rangsParXUID(brute), rangsParXUID(lu.Table)
	if rb[xa] != listes[xb] || rb[xb] != listes[xa] {
		t.Fatalf("table brute %v, listes %v : la permutation de test n'a pas pris", rb, listes)
	}
	if lu.Corriges != 2 || !lu.Table.Lue() || len(lu.Table.Seats) != len(brute.Seats) {
		t.Fatalf("corriges %d, lue %v, %d sieges : deux sieges corriges, la table se lit", lu.Corriges, lu.Table.Lue(), len(lu.Table.Seats))
	}
	for x, siege := range rl {
		if siege != listes[x] {
			t.Fatalf("xuid %d : siege %d, la liste dit %d", x, siege, listes[x])
		}
	}
	if !slices.IsSortedFunc(lu.Table.Seats, func(a, b FilmPlayerSeat) int { return a.FilmIndex - b.FilmIndex }) {
		t.Fatal("les sieges ne sont plus ranges par index")
	}
}

// TestAucunLecteurNeLitLaTableSansLesListages : la table de `chunk_00` se lit par UN point, qui la rend aux sieges des
// listes ([ScanFilmPlayerTableAuxListages]). Un lecteur qui appelle [ScanFilmPlayerTable] seul garde le rang d'une
// ligne que les listes donnent a un autre siege : la cuisson, les bots non epingles et le kill-feed ne liraient plus la
// meme table. Seuls ce fichier et la lecture brute elle-meme l'appellent.
// MUTATIONS — un lecteur de production qui appelle la lecture brute : ROUGE.
func TestAucunLecteurNeLitLaTableSansLesListages(t *testing.T) {
	racine := "../.."
	autorises := map[string]bool{"internal/grammar/film_player_table.go": true, "internal/grammar/film_player_table_listages.go": true}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(racine, chemin)
		rel = filepath.ToSlash(rel)
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || autorises[rel] {
			return nil
		}
		octets, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		for i, ligne := range strings.Split(string(octets), "\n") {
			code, _, _ := strings.Cut(ligne, "//")
			if strings.Contains(code, "ScanFilmPlayerTable(") {
				t.Errorf("%s:%d appelle ScanFilmPlayerTable : passer par ScanFilmPlayerTableAuxListages", rel, i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
