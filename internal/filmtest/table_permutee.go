// Package filmtest : DES FILMS DE TEST FABRIQUES EN MEMOIRE A PARTIR D'UNE MINI-BOBINE VERSIONNEE. Aide de test : aucun
// fichier de production de la brique ne l'importe.
package filmtest

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// PermuterDeuxLignesDeLaTable echange en place, dans le registre du film, les octets des deux premieres lignes
// occupees de la table des joueurs (de meme longueur) : l'occupant de chaque ligne est range au rang de l'autre, alors
// que les paquets de type 8 gardent leurs sieges. C'est un film dont le rang de la table n'est pas le siege liste, sans
// un octet de fixture de plus. Rend les deux xuids permutes. Le film doit venir de [source.LoadDir] sur une bobine qui
// porte son `chunk_00` et dont la table a au moins deux occupants de meme longueur d'enregistrement.
func PermuterDeuxLignesDeLaTable(t testing.TB, film *source.Film) (a, b uint64) {
	t.Helper()
	chunk0, ok := grammar.FilmRegistryChunk(film)
	if !ok {
		t.Fatal("film sans registre")
	}
	ident, err := grammar.ReadFilmIdentity(chunk0)
	if err != nil {
		t.Fatal(err)
	}
	slots, _, err := grammar.ReadPlayerTable(chunk0, ident)
	if err != nil {
		t.Fatal(err)
	}
	var l []types.PlayerSlot
	for _, s := range slots {
		if s.XUID != 0 {
			l = append(l, s)
		}
	}
	if len(l) < 2 || l[0].TotalBits != l[1].TotalBits {
		t.Fatalf("deux lignes de meme longueur attendues, %d occupees", len(l))
	}
	bit := func(pos int) byte { return chunk0[pos>>3] >> (7 - pos&7) & 1 }
	pose := func(pos int, v byte) {
		m := byte(1) << (7 - pos&7)
		if v == 0 {
			chunk0[pos>>3] &^= m
		} else {
			chunk0[pos>>3] |= m
		}
	}
	for i := range l[0].TotalBits {
		x, y := bit(l[0].Bit+i), bit(l[1].Bit+i)
		pose(l[0].Bit+i, y)
		pose(l[1].Bit+i, x)
	}
	return l[0].XUID, l[1].XUID
}
