package signaux

import (
	"encoding/binary"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/datums"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entreeStatborg est l'entree d'un datum vivant qui leve le composant d'entree du statborg.
func entreeStatborg() datums.DatumEntry {
	e := datums.DatumEntry{Drapeaux: datums.DatumAlloue | datums.DatumPublie}
	e.Composants[statborgComposantEntree/64] |= 1 << (statborgComposantEntree % 64)
	return e
}

// tableDe fabrique une table de `n` entrees dont les index de `vivants` sont des entites de statborg.
func tableDe(n int, vivants ...int) []datums.DatumEntry {
	t := make([]datums.DatumEntry, n)
	for _, i := range vivants {
		t[i] = entreeStatborg()
	}
	return t
}

func plage(de, a int) []int {
	var out []int
	for i := de; i <= a; i++ {
		out = append(out, i)
	}
	return out
}

func TestDernierIndexDeBande(t *testing.T) {
	autreArchetype := tableDe(40, plage(3, 12)...)
	autreArchetype[13] = datums.DatumEntry{Drapeaux: datums.DatumAlloue | datums.DatumPublie}
	liberee := tableDe(40, plage(3, 12)...)
	liberee[13] = entreeStatborg()
	liberee[13].Drapeaux |= datums.DatumLibere
	sansComposant := tableDe(40, plage(3, 20)...)
	for i := range sansComposant {
		sansComposant[i].Composants = [4]uint64{}
	}

	cas := []struct {
		nom  string
		en   []datums.DatumEntry
		want int
	}{
		{"table vide", nil, 0},
		{"aucune entite", tableDe(40), 0},
		{"une equipe et huit joueurs", tableDe(40, plage(3, 12)...), 12},
		{"un index vacant isole ne ferme pas la bande", tableDe(40, append(plage(3, 7), plage(9, 20)...)...), 20},
		{"deux index vacants ferment la bande", tableDe(40, append(plage(3, 7), plage(10, 20)...)...), 7},
		{"la bande part de l'index 3", tableDe(40, plage(4, 20)...), 20},
		{"sans l'index 3 ni le 4", tableDe(40, plage(5, 20)...), 0},
		{"un autre archetype ne leve pas le composant d'entree", sansComposant, 0},
		{"un archetype vivant sans le composant ne prolonge pas la bande", autreArchetype, 12},
		{"une entite liberee ne prolonge pas la bande", liberee, 12},
		{"la bande s'arrete au dernier slot lisible", tableDe(120, plage(3, 110)...), StatborgSlotMax / 2},
	}
	for _, c := range cas {
		if got := dernierIndexDeBande(c.en); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.nom, got, c.want)
		}
	}
}

func TestLaBandeDuStatborgFilmAbsent(t *testing.T) {
	got := LaBandeDuStatborg(nil)
	if got.SlotMax != StatborgSlotBandeSure || got.ParDefaut || got.Refuses != 0 {
		t.Errorf("film absent : %+v", got)
	}
}

// bandeEntrees est le nombre d'entrees des blocs de datums synthetiques ; la mise en page est celle
// que `datums.LireBlocDeDatums` relit (79 bits d'entree puis 256 bits de composants par slot).
const (
	bandeEntrees = 40
	bitsEntree   = 79
	bitsBitmap   = 256
)

// blocDeDatums ecrit un bloc de type 1 dont les index de `vivants` sont des entites de statborg.
func blocDeDatums(t *testing.T, vivants []int) []byte {
	t.Helper()
	pay := make([]byte, (bandeEntrees*(bitsEntree+bitsBitmap)+5*32+7)/8)
	for _, i := range vivants {
		pay = setBitsBE(pay, i*bitsEntree, 6, uint64(datums.DatumAlloue|datums.DatumPublie))
		pay = setBitsBE(pay, bandeEntrees*bitsEntree+i*bitsBitmap+statborgComposantEntree, 1, 1)
	}
	bloc, err := datums.LireBlocDeDatums(pay)
	if err != nil {
		t.Fatalf("le bloc synthetique n'est pas relu : %v", err)
	}
	for _, i := range vivants {
		if !bloc.Entrees[i].Vivante() || !bloc.Entrees[i].Composant(statborgComposantEntree) {
			t.Fatalf("l'entree %d du bloc synthetique n'est pas celle qu'on a ecrite", i)
		}
	}
	return pay
}

// filmAvecTable fabrique un film d'un chunk de jeu portant un paquet FRAME (un enregistrement au slot
// 26) et, quand `bloc` n'est pas nil, un paquet de datums.
func filmAvecTable(t *testing.T, bloc []byte) *source.Film {
	t.Helper()
	frame := statVecteurAuSlot(10, 26)
	paquet := func(typ uint16, payload []byte, ts uint64) []byte {
		hdr := make([]byte, 16)
		binary.LittleEndian.PutUint16(hdr[0:], typ)
		binary.LittleEndian.PutUint32(hdr[4:], uint32(len(payload)))
		binary.LittleEndian.PutUint64(hdr[8:], ts)
		return append(hdr, payload...)
	}
	chunk := paquet(typeDeTrame, frame, 0)
	if bloc != nil {
		chunk = append(chunk, paquet(uint16(datums.PacketTypeDatums), bloc, 1000)...)
	}
	film, err := source.Load(source.MemoryChunks{chunk}, []types.ChunkMeta{{Index: 1, ChunkType: 2}})
	if err != nil {
		t.Fatalf("chargement du film : %v", err)
	}
	return film
}

// TestLireLeStatborgSuitLaBandeDesDatums : l'enregistrement du slot 26 n'est lu que si la table de
// datums du film declare treize joueurs ; sans table, ou avec un bloc refuse, la lecture s'arrete au
// slot 24 et le dit. Remplacer la borne de lecture par [StatborgSlotBandeSure] fait echouer le premier cas.
func TestLireLeStatborgSuitLaBandeDesDatums(t *testing.T) {
	cas := []struct {
		nom       string
		bloc      []byte
		lus       int
		parDefaut bool
		refuses   int
		slotMaxLu int
	}{
		{"table de treize joueurs", blocDeDatums(t, plage(3, 15)), 1, false, 0, 30},
		{"table de huit joueurs", blocDeDatums(t, plage(3, 12)), 0, false, 0, StatborgSlotBandeSure},
		{"table absente", nil, 0, true, 0, StatborgSlotBandeSure},
		{"bloc refuse", make([]byte, 9), 0, true, 1, StatborgSlotBandeSure},
		{"table sans entite de statborg", blocDeDatums(t, nil), 0, true, 0, StatborgSlotBandeSure},
	}
	for _, c := range cas {
		film := filmAvecTable(t, c.bloc)
		l := LireLeStatborg(film)
		if len(l.Records) != c.lus || l.BandeParDefaut != c.parDefaut || l.BlocsDeDatumsRefuses != c.refuses {
			t.Errorf("%s : %d enregistrement(s), parDefaut=%v, refuses=%d ; attendu %d, %v, %d",
				c.nom, len(l.Records), l.BandeParDefaut, l.BlocsDeDatumsRefuses, c.lus, c.parDefaut, c.refuses)
		}
		if b := LaBandeDuStatborg(film); b.SlotMax != c.slotMaxLu {
			t.Errorf("%s : slot max %d, attendu %d", c.nom, b.SlotMax, c.slotMaxLu)
		}
	}
}
