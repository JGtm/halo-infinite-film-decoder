//go:build research

package divergence

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// snapshots_bits_research_test.go : lectures brutes de la charge utile d un paquet d image-cle, a cote de
// la comparaison des tables d entites de `snapshots_research_test.go` : bits d une plage, table que la
// production y lit, positions d un mot donne.

// TestSnapBits : les bits d une plage de la charge utile du paquet d image-cle (type 2) d un chunk.
//
//	DIVERGENCE_RACINE=<film_chunks> BITS_FILM=<id> BITS_CHUNK=<n> BITS_DE=<bit> BITS_A=<bit> \
//	  go test -tags=research ./research/divergence/ -run TestSnapBits -v
func TestSnapBits(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	data, pks, ok := fc.ChunkAt(envInt("BITS_CHUNK", -1))
	if !ok {
		t.Fatal("chunk absent")
	}
	for _, pk := range pks {
		if pk.Type != grammar.PacketTypeKeyframe {
			continue
		}
		pay := pk.Payload(data)
		de, a := envInt("BITS_DE", 0), envInt("BITS_A", 0)
		fmt.Printf("paquet %d (type 2) : %d bits ; bits [%d,%d) : %s\n", pk.Index, len(pay)*8, de, a, formeBits(pay, de, a))
	}
}

// dernierOp : la derniere operation du temoin sur le slot avant le chunk c.
func dernierOp(h []opHist, c int) string {
	var last *opHist
	for i := range h {
		if h[i].chunk < c {
			last = &h[i]
		}
	}
	switch {
	case last == nil:
		return "jamais_cree_vu_par_le_temoin"
	case last.del:
		return "supprime_vu_par_le_temoin"
	}
	return "cree_vu_par_le_temoin_sans_suppression"
}

func u32Bits(pay []byte, at int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		b := at + i
		if b>>3 >= len(pay) {
			return v << (32 - i)
		}
		v = v<<1 | uint32(pay[b>>3]>>(7-uint(b&7))&1)
	}
	return v
}

// TestSnapScan : la table d entites que la PRODUCTION lit dans l image-cle (`grammar.EntetesDeTable`, la recherche d en-tetes a
// deux passes) contre la table que le temoin lit de bout en bout (`<id>_snap.tsv`) : entrees justes (emplacement, generation et type
// egaux), de type ou de generation autre, manquees, en plus.
func TestSnapScan(t *testing.T) {
	racine, sdir, liste := os.Getenv("SNAP_RACINE"), os.Getenv("SNAP_SNAP"), os.Getenv("SNAP_FILMS")
	if racine == "" || sdir == "" || liste == "" {
		t.Skip("SNAP_RACINE, SNAP_SNAP, SNAP_FILMS requis")
	}
	films, err := lireLesFilms(liste)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range films {
		fc, err := ouvrirFilm(racine, f.id, f.niveau, nil)
		if err != nil {
			t.Fatal(err)
		}
		snaps, err := lireSnap(filepath.Join(sdir, f.id+"_snap.tsv"))
		if err != nil {
			t.Fatal(err)
		}
		var w, trouvees, justes, autretype, autregen, manquees, enplus, ecartes int
		for c, W := range snaps {
			data, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != grammar.PacketTypeKeyframe {
					continue
				}
				entrees, ec := grammar.EntetesDeTable(pk.Payload(data))
				ecartes += ec
				vus := map[int]grammar.EnteteDeTable{}
				for _, e := range entrees {
					vus[int(e.Slot())] = e
				}
				w += len(W)
				trouvees += len(entrees)
				for s, we := range W {
					e, ok := vus[s]
					switch {
					case !ok:
						manquees++
					case int(e.TI) != we.ty:
						autretype++
					case int(e.Gen()) != we.gen:
						autregen++
					default:
						justes++
					}
				}
				for s := range vus {
					if _, ok := W[s]; !ok {
						enplus++
					}
				}
			}
		}
		fmt.Printf("%s TABLE DE PRODUCTION : temoin %d ; table %d ; justes %d ; autre type %d ; autre generation %d ; manquees %d ; en plus %d ; slots ecartes %d\n",
			f.id, w, trouvees, justes, autretype, autregen, manquees, enplus, ecartes)
	}
}

// TestSnapFind : les positions (bits) ou un mot de 64 bits `[id:32][type:32]` apparait dans la charge utile du
// paquet d image-cle d un chunk, a tout decalage. BITS_FILM, BITS_CHUNK, FIND_ID, FIND_TYPE (decimaux).
func TestSnapFind(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	data, pks, ok := fc.ChunkAt(envInt("BITS_CHUNK", -1))
	if !ok {
		t.Fatal("chunk absent")
	}
	wid, wty := uint32(envInt("FIND_ID", 0)), uint32(envInt("FIND_TYPE", 0)) //nolint:gosec // operateur
	for _, pk := range pks {
		if pk.Type != grammar.PacketTypeKeyframe {
			continue
		}
		pay := pk.Payload(data)
		for at := 0; at+64 <= len(pay)*8; at++ {
			if u32Bits(pay, at) == wid && u32Bits(pay, at+32) == wty {
				fmt.Printf("trouve au bit %d\n", at)
			}
		}
	}
}
