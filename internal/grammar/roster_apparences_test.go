package grammar

// roster_apparences_test.go — LA LECTURE DE L'APPARENCE, SUR DES OCTETS FABRIQUES : la forme du bloc (paires ou brut),
// la queue de zeros, la regle du dernier bloc non nul et les compteurs. Aucun bloc d'un vrai joueur.

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// blocSynth fabrique un bloc de `largeur` octets : la tete (n, 0), n paires (cle i+1, valeur base+i), puis `suite`.
func blocSynth(largeur, n int, base uint32, suite []byte) []byte {
	b := make([]byte, largeur)
	binary.LittleEndian.PutUint32(b, uint32(n)) //nolint:gosec // petit compteur de test
	for i := range n {
		binary.LittleEndian.PutUint32(b[8+8*i:], uint32(i+1))     //nolint:gosec // cle de test
		binary.LittleEndian.PutUint32(b[12+8*i:], base+uint32(i)) //nolint:gosec // valeur de test
	}
	copy(b[8+8*n:], suite)
	return b
}

func TestDecouperLeBlocPairesEtQueueDeZeros(t *testing.T) {
	bloc := blocSynth(64, 3, 0xA0, []byte{1, 2, 3, 0, 0, 4, 0, 0})
	pa := decouperLeBloc(bloc)
	if pa.Size != 64 || len(pa.Pairs) != 3 || pa.Pairs[2] != (types.AppearancePair{Key: 3, Value: 0xA2}) {
		t.Fatalf("paires mal lues : %+v", pa)
	}
	// la queue de zeros est omise, les zeros du milieu sont gardes.
	if want := []byte{1, 2, 3, 0, 0, 4}; !slices.Equal(pa.Rest, want) {
		t.Fatalf("rest = %v, attendu %v", pa.Rest, want)
	}
	// chaque octet est range une fois : tete + paires + rest + queue omise = largeur.
	if got := 8 + 8*len(pa.Pairs) + len(pa.Rest); got > pa.Size {
		t.Fatalf("octets publies %d > largeur %d", got, pa.Size)
	}
}

func TestDecouperLeBlocSansReste(t *testing.T) {
	pa := decouperLeBloc(blocSynth(64, 2, 5, nil))
	if len(pa.Pairs) != 2 || pa.Rest != nil {
		t.Fatalf("un bloc sans reste ne porte pas de rest : %+v", pa)
	}
}

func TestDecouperLeBlocBrutQuandLaTeteNeSeLitPas(t *testing.T) {
	cas := map[string][]byte{
		"compteur nul":        blocSynth(64, 0, 0, []byte{9, 9}),
		"compteur trop grand": blocSynth(64, 3, 0, nil),
		"mot non nul":         blocSynth(64, 3, 0, nil),
		"paires hors bloc":    blocSynth(64, 3, 0, nil),
	}
	binary.LittleEndian.PutUint32(cas["compteur trop grand"], apparencePairesMax+1)
	binary.LittleEndian.PutUint32(cas["mot non nul"][4:], 1)
	binary.LittleEndian.PutUint32(cas["paires hors bloc"], 20)
	for nom, bloc := range cas {
		pa := decouperLeBloc(bloc)
		if len(pa.Pairs) != 0 {
			t.Errorf("%s : des paires ont ete lues", nom)
		}
		// brut : tout le bloc, a la queue de zeros pres.
		if want := slices.Clone(bloc); !slices.Equal(pa.Rest, trimZeros(want)) {
			t.Errorf("%s : rest = %v, attendu %v", nom, pa.Rest, trimZeros(want))
		}
	}
}

func trimZeros(b []byte) []byte {
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	return b
}

func TestOctetsDuFluxNonAligne(t *testing.T) {
	// 0xAB 0xCD 0xEF lus a partir du bit 4 : 0xBC 0xDE.
	got, ok := octetsDuFlux([]byte{0xAB, 0xCD, 0xEF}, 4, 2)
	if !ok || !slices.Equal(got, []byte{0xBC, 0xDE}) {
		t.Fatalf("lu %x ok=%v", got, ok)
	}
	if _, ok := octetsDuFlux([]byte{1, 2}, 4, 2); ok {
		t.Fatal("une plage hors du paquet doit etre refusee")
	}
}

// poserBloc ecrit `bloc` dans `buf` a partir du bit `debut` (poids fort d'abord).
func poserBloc(buf []byte, debut int, bloc []byte) {
	for i, c := range bloc {
		for k := range 8 {
			bit := debut + 8*i + k
			if c&(0x80>>uint(k)) != 0 {
				buf[bit/8] |= 0x80 >> uint(bit%8)
			} else {
				buf[bit/8] &^= 0x80 >> uint(bit%8)
			}
		}
	}
}

// paquetSynth fabrique un paquet de type 8 dont chaque entree porte le bloc donne (meme ordre que `ids`).
func paquetSynth(t *testing.T, ts uint64, ids []uint64, blocs [][]byte) types.Packet {
	t.Helper()
	var entrees []rosterSynthEntree
	for _, id := range ids {
		entrees = append(entrees, rosterSynthEntree{porte: true, identite: id, etiquette: []uint16{'x'}})
	}
	pay, _ := rosterEcrire(entrees, true, rosterPersoSynth*8)
	lus, bilan := DecodeRoster(pay, rosterOptGateVersion, rosterPersoSynth*8)
	if bilan.Entrees != len(ids) {
		t.Fatalf("paquet synthetique mal forme : %+v", bilan)
	}
	for i, e := range lus {
		fin := e.Joueur.Bit + e.Joueur.TotalBits
		poserBloc(pay, fin-blocDeQueueBits-rosterPersoSynth*8, blocs[i])
	}
	return types.Packet{Type: int(PacketTypeRoster), TS: ts, Payload: pay}
}

const (
	xuidSynthA = uint64(0x0009_0000_0000_0001)
	xuidSynthB = uint64(0x0009_0000_0000_0002)
	xuidSynthC = uint64(0x0009_0000_0000_0003)
)

func TestLireLesApparencesRegleDuDernierBlocNonNul(t *testing.T) {
	zero := make([]byte, rosterPersoSynth)
	a1 := blocSynth(rosterPersoSynth, 2, 0x10, nil)
	a2 := blocSynth(rosterPersoSynth, 2, 0x20, []byte{7})
	b := blocSynth(rosterPersoSynth, 1, 0x30, nil)
	brut := blocSynth(rosterPersoSynth, 0, 0, []byte{5, 5})
	ids := []uint64{xuidSynthA, xuidSynthB, xuidSynthC}
	pkts := []types.Packet{
		// A : charge, puis un bloc nul (n'efface rien), puis un autre bloc (change). B : constant. C : toujours nul.
		paquetSynth(t, 1, ids, [][]byte{a1, b, zero}),
		paquetSynth(t, 2, ids, [][]byte{zero, b, zero}),
		paquetSynth(t, 3, ids, [][]byte{a2, b, zero}),
		// un paquet identique au precedent n'est pas relu.
		paquetSynth(t, 4, ids, [][]byte{a2, b, zero}),
	}
	pkts = append(pkts, types.Packet{Type: 0, Payload: []byte{1}}) // pas un paquet de roster
	out, st := lireLesApparences(pkts, rosterOptGateVersion, rosterPersoSynth, types.AppearanceStats{Scanned: true})
	if len(out) != 2 || out[0].XUID != xuidSynthA || out[1].XUID != xuidSynthB {
		t.Fatalf("apparences %+v", out)
	}
	if got := out[0].Pairs[0].Value; got != 0x20 {
		t.Errorf("le dernier bloc non nul doit faire foi : valeur %#x", got)
	}
	if !slices.Equal(out[0].Rest, []byte{7}) {
		t.Errorf("rest de A = %v", out[0].Rest)
	}
	want := types.AppearanceStats{Scanned: true, Packets: 4, Distinct: 3, Pairs: 2, Unloaded: 1, Changed: 1}
	if st != want {
		t.Errorf("compteurs %+v, attendu %+v", st, want)
	}

	// brut : un bloc dont la tete ne se lit pas est rendu entier et compte « brut ».
	out, st = lireLesApparences([]types.Packet{paquetSynth(t, 1, ids[:1], [][]byte{brut})}, rosterOptGateVersion,
		rosterPersoSynth, types.AppearanceStats{Scanned: true})
	if len(out) != 1 || len(out[0].Pairs) != 0 || !slices.Equal(out[0].Rest, []byte{0, 0, 0, 0, 0, 0, 0, 0, 5, 5}) || st.Raw != 1 || st.Pairs != 0 {
		t.Errorf("brut : %+v %+v", out, st)
	}
}

func TestLireLesApparencesPaquetRefuseEtSansXUID(t *testing.T) {
	bon := blocSynth(rosterPersoSynth, 1, 1, nil)
	pkts := []types.Packet{
		{Type: int(PacketTypeRoster), TS: 1, Payload: []byte{0xFF, 0xFF, 0xFF, 0xFF}}, // annonce plus d'entrees qu'il n'en porte
		paquetSynth(t, 2, []uint64{0, xuidSynthA}, [][]byte{bon, bon}),
	}
	out, st := lireLesApparences(pkts, rosterOptGateVersion, rosterPersoSynth, types.AppearanceStats{Scanned: true})
	if st.Refused != 1 || st.NoXUID != 1 || len(out) != 1 || out[0].XUID != xuidSynthA {
		t.Errorf("%+v %+v", out, st)
	}
}

// TestApparencesBornesSurOctetsFabriques : aucun accès au bloc ne sort de sa largeur, quelle que soit la tete : compteur
// absurde, paires qui debordent, bloc plus court que la tete, largeur de profil plus grande que les octets du paquet,
// paquet tronque a chaque longueur. Rien ne panique ; ce qui ne se lit pas est refuse ou rendu brut.
func TestApparencesBornesSurOctetsFabriques(t *testing.T) {
	for _, n := range []uint32{0, 257, 1 << 31, 0xFFFFFFFF} {
		bloc := make([]byte, 64)
		binary.LittleEndian.PutUint32(bloc, n)
		bloc[8] = 1
		pa := decouperLeBloc(bloc)
		if len(pa.Pairs) != 0 || len(pa.Rest) == 0 {
			t.Errorf("compteur %d : le bloc doit etre rendu brut, obtenu %+v", n, pa)
		}
	}
	if pa := decouperLeBloc(blocSynth(80, 8, 1, nil)[:64]); len(pa.Pairs) != 0 { // 8 + 8*8 > 64 : les paires debordent
		t.Errorf("paires hors du bloc : %+v", pa)
	}
	for l := range 9 { // bloc plus court que la tete
		b := bytes.Repeat([]byte{1}, l)
		if pa := decouperLeBloc(b); len(pa.Pairs) != 0 || len(pa.Rest) != l {
			t.Errorf("bloc de %d octets : %+v", l, pa)
		}
	}
	// chaque octet publie une fois : tete + paires + reste, completes de zeros jusqu'a Size, redonnent le bloc.
	for _, bloc := range [][]byte{blocSynth(64, 3, 9, []byte{1, 0, 2}), blocSynth(64, 0, 0, []byte{5}), blocSynth(64, 7, 2, nil)} {
		pa := decouperLeBloc(bloc)
		var re []byte
		if len(pa.Pairs) > 0 {
			re = append(re, bloc[:apparenceTeteOctets]...)
			for _, p := range pa.Pairs {
				re = binary.LittleEndian.AppendUint32(re, p.Key)
				re = binary.LittleEndian.AppendUint32(re, p.Value)
			}
		}
		re = append(re, pa.Rest...)
		re = append(re, make([]byte, pa.Size-len(re))...)
		if !bytes.Equal(re, bloc) {
			t.Errorf("le bloc ne se retrouve pas par size : %x / %x", re, bloc)
		}
	}
	pkt := paquetSynth(t, 1, []uint64{xuidSynthA}, [][]byte{blocSynth(rosterPersoSynth, 2, 1, nil)})
	st0 := types.AppearanceStats{Scanned: true}
	// largeur de profil plus grande que le paquet : refus ou rien, jamais de panique.
	for _, largeur := range []int{rosterPersoSynth + 1, 4096, 1 << 20} {
		lireLesApparences([]types.Packet{pkt}, rosterOptGateVersion, largeur, st0)
	}
	for n := range len(pkt.Payload) {
		cp := pkt
		cp.Payload = pkt.Payload[:n]
		out, st := lireLesApparences([]types.Packet{cp}, rosterOptGateVersion, rosterPersoSynth, st0)
		if len(out) != 0 && n < len(pkt.Payload)/2 {
			t.Errorf("paquet tronque a %d octets : une apparence est rendue (%+v, %+v)", n, out, st)
		}
	}
}
