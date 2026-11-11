//go:build research

package divergence

// snapshots_research_test.go : LA TABLE D ENTITES A L OUVERTURE D UN MORCEAU, la notre contre celle du
// temoin (plan marche complete, A2.H, mesure B1).
//
//	SNAP_RACINE=<film_chunks> SNAP_FILMS=<liste> SNAP_TEMOIN=<dossier des TSV oracle> SNAP_SNAP=<dossier des
//	<id>_snap.tsv du binaire `snapshots`> SNAP_CARTE=<catalogue> SNAP_SORTIE=<dossier> [SNAP_ID=<id>] \
//	  go test -tags=research ./research/divergence/ -run TestSnap -v
//
// Notre table d ouverture du morceau c = la table d entites de la marche rendue apres le PREMIER paquet delta
// du morceau (les slots que ce paquet cree ou supprime sont ecartes de la comparaison). Notre table de fin
// du morceau c-1 = celle rendue apres son DERNIER paquet delta. La table « gardee » K(c) simule la regle
// « garder la table d un morceau au suivant tant qu aucune suppression ne la retire » : l ouverture
// courante, plus les liaisons de la fin du morceau precedent que l ouverture n a pas reportees (ce que la
// marche oublie aujourd hui, `OublierLesSlotsNonPortes`).
//
// Le temoin : `<id>_snap.tsv` (lignes S : index generation type bit source). Ses creations et suppressions
// par paquet viennent du TSV oracle.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

const slotsSnap = 8192

type liaisonNotre struct {
	ti   int
	gen  int // -1 : generation inconnue
	prov lecture.Liaison
}

type entiteTemoin struct {
	gen, ty, bit int
	src          string
}

// instantane copie la table d entites de la marche (valide le temps d un tour).
func instantane(e lecture.Entites) map[int]liaisonNotre {
	out := make(map[int]liaisonNotre, e.Liees())
	for s := 0; s < slotsSnap; s++ {
		en, ok := e.Entite(uint32(s)) //nolint:gosec // borne
		if !ok {
			continue
		}
		g := -1
		if en.GenerationConnue {
			g = int(en.EID >> 30)
		}
		out[s] = liaisonNotre{ti: int(en.TI), gen: g, prov: en.Liaison}
	}
	return out
}

// lireSnap lit `<id>_snap.tsv` : chunk -> slot -> entite (lignes S).
func lireSnap(chemin string) (map[int]map[int]entiteTemoin, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin de l operateur
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[int]map[int]entiteTemoin{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) < 9 || c[1] != "S" {
			if len(c) >= 3 && c[1] == "W" {
				if _, ok := out[atoi(c[2])]; !ok {
					out[atoi(c[2])] = map[int]entiteTemoin{}
				}
			}
			continue
		}
		ch := atoi(c[2])
		if out[ch] == nil {
			out[ch] = map[int]entiteTemoin{}
		}
		out[ch][atoi(c[3])] = entiteTemoin{gen: atoi(c[4]), ty: atoi(c[5]), bit: atoi(c[6]), src: c[7]}
	}
	return out, sc.Err()
}

type opHist struct {
	chunk, paquet, gen, ty int
	del                    bool
}

type opCompact struct {
	slot, gen int
	kind      string
}

type paquetOps struct {
	paquet int
	ops    []opCompact
}

// etatDans dit ce que la table T donne pour l entite w du slot s : ok, absente, type_autre, generation_autre.
func etatDans(T map[int]liaisonNotre, s int, we entiteTemoin) string {
	l, ok := T[s]
	switch {
	case !ok:
		return "absente"
	case l.ti != we.ty:
		return "type_autre"
	case l.gen >= 0 && l.gen != we.gen:
		return "generation_autre"
	}
	return "ok"
}

// impactDesPaquets : pour chaque paquet du chunk (hors le premier), les mises a jour du temoin sur une
// entite du snapshot que le paquet n a pas (re)creee, jugees sous la table d ouverture O puis sous la table
// gardee K. Un paquet compte une fois, sous le pire etat de ses mises a jour (absente avant type_autre).
func impactDesPaquets(pq []paquetOps, W map[int]entiteTemoin, O, K map[int]liaisonNotre, tc map[int]bool, cO, cK *compteurs, chunk int, w *bufio.Writer) {
	touchee := map[int]bool{}
	for i, p := range pq {
		pireO, pireK := "ok", "ok"
		rang := map[string]int{"ok": 0, "generation_autre": 1, "type_autre": 2, "absente": 3}
		for _, o := range p.ops {
			if o.kind != "update" {
				touchee[o.slot] = true
				continue
			}
			we, ok := W[o.slot]
			if !ok || touchee[o.slot] || tc[o.slot] || i == 0 {
				continue
			}
			if eo, ek := etatDans(O, o.slot, we), etatDans(K, o.slot, we); eo != "ok" || ek != "ok" {
				fmt.Fprintf(w, "IMPACT\t%d\t%d\t%d\t%d\t%d\t%s\t%s\n", chunk, p.paquet, o.slot, we.gen, we.ty, eo, ek)
			}
			if e := etatDans(O, o.slot, we); rang[e] > rang[pireO] {
				pireO = e
			}
			if e := etatDans(K, o.slot, we); rang[e] > rang[pireK] {
				pireK = e
			}
		}
		cO.ajouter("paquets_mise_a_jour_chunk_" + pireO)
		cK.ajouter("paquets_mise_a_jour_chunk_" + pireK)
	}
}

type compteurs struct{ m map[string]int }

func (c *compteurs) ajouter(k string) {
	if c.m == nil {
		c.m = map[string]int{}
	}
	c.m[k]++
}

func TestSnap(t *testing.T) {
	racine, tdir, sdir, sortie, liste := os.Getenv("SNAP_RACINE"), os.Getenv("SNAP_TEMOIN"), os.Getenv("SNAP_SNAP"), os.Getenv("SNAP_SORTIE"), os.Getenv("SNAP_FILMS")
	if racine == "" || tdir == "" || sdir == "" || sortie == "" || liste == "" {
		t.Skip("SNAP_RACINE, SNAP_FILMS, SNAP_TEMOIN, SNAP_SNAP, SNAP_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	var cat *profile.MapQuantCatalog
	if c := os.Getenv("SNAP_CARTE"); c != "" {
		var err error
		if cat, err = profile.LoadMapQuantCatalog(c); err != nil {
			t.Fatal(err)
		}
	}
	films, err := lireLesFilms(liste)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(sortie, 0o755) //nolint:gosec // sortie
	for _, f := range films {
		if seul := os.Getenv("SNAP_ID"); seul != "" && seul != f.id {
			continue
		}
		if err := snapUnFilm(racine, tdir, sdir, sortie, f, cat); err != nil {
			t.Fatalf("film %s : %v", f.id, err)
		}
	}
}

func snapUnFilm(racine, tdir, sdir, sortie string, f filmDeTest, cat *profile.MapQuantCatalog) error {
	fc, err := ouvrirFilm(racine, f.id, f.niveau, cat)
	if err != nil {
		return err
	}
	verdicts, err := lireLesVerdicts(fc)
	if err != nil {
		return err
	}
	snaps, err := lireSnap(filepath.Join(sdir, f.id+"_snap.tsv"))
	if err != nil {
		return err
	}
	// Dernier paquet delta de chaque chunk.
	dernier := map[int]int{}
	for _, c := range fc.ChunkNumbers() {
		_, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type == grammar.PacketTypeDelta {
				dernier[c] = max(dernier[c], pk.Index)
			}
		}
	}
	// Histoire des creations/suppressions du temoin, par slot, dans l ordre du flux.
	tem, err := ouvrirTemoin(filepath.Join(tdir, f.id+".tsv"))
	if err != nil {
		return err
	}
	hist := map[int][]opHist{}
	parChunk := map[int][]paquetOps{}
	for {
		p, err := tem.suivant()
		if err != nil {
			tem.fermer()
			return err
		}
		if p == nil {
			break
		}
		po := paquetOps{paquet: p.paquet}
		for _, o := range p.ops {
			po.ops = append(po.ops, opCompact{slot: o.index, gen: o.gen, kind: o.op})
			switch o.op {
			case "create":
				hist[o.index] = append(hist[o.index], opHist{chunk: p.chunk, paquet: p.paquet, gen: o.gen, ty: atoi(o.typeEntite)})
			case "delete":
				hist[o.index] = append(hist[o.index], opHist{chunk: p.chunk, paquet: p.paquet, gen: o.gen, del: true})
			}
		}
		if len(po.ops) > 0 {
			parChunk[p.chunk] = append(parChunk[p.chunk], po)
		}
	}
	tem.fermer()

	// Notre marche : tables d ouverture et de fin de chunk.
	ouverture := map[int]map[int]liaisonNotre{}
	touches := map[int]map[int]bool{}
	fin := map[int]map[int]liaisonNotre{}
	vu := map[int]bool{}
	for p, err := range fc.Trames(grammar.NouvelleObservation()) {
		if err != nil {
			return err
		}
		if !vu[p.Chunk] {
			vu[p.Chunk] = true
			ouverture[p.Chunk] = instantane(p.Entites)
			tc := map[int]bool{}
			for _, r := range p.Records {
				if r.Genre == lecture.GenreNeuf || r.Genre == lecture.GenreSuppression {
					tc[int(r.Vie.Slot)] = true
				}
			}
			touches[p.Chunk] = tc
		}
		if d, ok := dernier[p.Chunk]; ok && p.Index == d {
			fin[p.Chunk] = instantane(p.Entites)
		}
	}

	out, err := os.Create(filepath.Join(sortie, f.id+"_snap_cmp.tsv")) //nolint:gosec // sortie
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	w := bufio.NewWriter(out)
	defer func() { _ = w.Flush() }()
	fmt.Fprint(w, "chunk\tslot\tw_gen\tw_type\tw_src\to\to_ti\to_gen\to_prov\tk\tk_ti\tk_prov\tcause_absence_K\n")

	var cO, cK, cKs compteurs
	var prevKs map[int]liaisonNotre
	discard := bufio.NewWriter(io.Discard)
	chunks := make([]int, 0, len(snaps))
	for c := range snaps {
		chunks = append(chunks, c)
	}
	sort.Ints(chunks)
	for _, c := range chunks {
		W, O := snaps[c], ouverture[c]
		if O == nil {
			continue
		}
		prec := fin[c-1] // peut etre nil (premier chunk)
		K := make(map[int]liaisonNotre, len(O))
		for s, l := range prec {
			K[s] = l
		}
		for s, l := range O {
			K[s] = l // l ouverture courante prime sur la fin du chunk precedent
		}
		// Ks : report ITERATIF (optimiste) : la fin du chunk precedent et le report du chunk d avant, sauf les slots que le
		// temoin a vu supprimer, puis l ouverture courante par-dessus.
		Ks := make(map[int]liaisonNotre, len(O))
		for s, l := range prevKs {
			if dernierOp(hist[s], c) != "supprime_vu_par_le_temoin" {
				Ks[s] = l
			}
		}
		for s, l := range prec {
			if dernierOp(hist[s], c) != "supprime_vu_par_le_temoin" {
				Ks[s] = l
			}
		}
		for s, l := range O {
			Ks[s] = l
		}
		prevKs = Ks
		tc := touches[c]
		for s, we := range W {
			if !tc[s] {
				cKs.ajouter("Ks_" + etatDans(Ks, s, we))
			}
			cO.ajouter("entites_temoin")
			if tc[s] {
				cO.ajouter("ecartees_touchees_par_le_1er_paquet")
				continue
			}
			cO.ajouter("entites_temoin_comparees")
			st := func(T map[int]liaisonNotre) (string, liaisonNotre) {
				l, ok := T[s]
				switch {
				case !ok:
					return "absente", l
				case l.ti != we.ty:
					return "type_autre", l
				case l.gen >= 0 && l.gen != we.gen:
					return "generation_autre", l
				}
				return "ok", l
			}
			so, lo := st(O)
			sk, lk := st(K)
			cO.ajouter("O_" + so)
			cK.ajouter("K_" + sk)
			cause := ""
			if sk == "absente" {
				cause = causeAbsence(hist[s], c, we.gen, verdicts)
				cK.ajouter("K_absente_cause_" + cause)
			}
			if sk == "type_autre" || sk == "generation_autre" {
				cK.ajouter(fmt.Sprintf("K_%s_prov_%d", sk, lk.prov))
			}
			if so == "type_autre" || so == "generation_autre" {
				cO.ajouter(fmt.Sprintf("O_%s_prov_%d", so, lo.prov))
			}
			if so != "ok" || sk != "ok" {
				fmt.Fprintf(w, "%d\t%d\t%d\t%d\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\n", c, s, we.gen, we.ty, we.src, so, lo.ti, lo.gen, lo.prov, sk, lk.ti, lk.prov, cause)
			}
		}
		impactDesPaquets(parChunk[c], W, O, K, tc, &cO, &cK, c, w)
		var poubelle compteurs
		impactDesPaquets(parChunk[c], W, O, Ks, tc, &poubelle, &cKs, c, discard)
		for s := range O {
			if _, ok := W[s]; !ok && !tc[s] {
				cO.ajouter("O_en_plus")
				cO.ajouter(fmt.Sprintf("O_en_plus_prov_%d", O[s].prov))
			}
		}
		for s := range K {
			if _, ok := W[s]; !ok && !tc[s] {
				cK.ajouter("K_en_plus")
				cK.ajouter(fmt.Sprintf("K_en_plus_prov_%d", K[s].prov))
				cK.ajouter("K_en_plus_" + dernierOp(hist[s], c))
			}
		}
	}
	var sb strings.Builder
	for _, cm := range []compteurs{cO, cK, cKs} {
		ks := make([]string, 0, len(cm.m))
		for k := range cm.m {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			fmt.Fprintf(&sb, "%s\t%d\n", k, cm.m[k])
		}
	}
	return os.WriteFile(filepath.Join(sortie, f.id+"_snap_resume.txt"), []byte(sb.String()), 0o644) //nolint:gosec // sortie
}

// causeAbsence dit pourquoi une entite que le temoin porte a l ouverture du chunk c n est pas dans notre
// table : la derniere creation que le temoin a vue du slot avant le chunk c.
func causeAbsence(h []opHist, c, genTemoin int, verdicts map[cleDePaquet]verdictNous) string {
	var last *opHist
	for i := range h {
		if h[i].chunk < c {
			last = &h[i]
		}
	}
	switch {
	case last == nil:
		return "aucune_creation_vue_par_le_temoin"
	case last.del:
		return "temoin_a_supprime_puis_realloue"
	case last.gen != genTemoin:
		return "emplacement_realloue"
	case !verdicts[cleDePaquet{last.chunk, last.paquet}].fermee:
		if last.chunk == c-1 {
			return "creee_au_chunk_precedent_paquet_non_ferme"
		}
		return "creee_plus_tot_paquet_non_ferme"
	case last.chunk == c-1:
		return "creee_au_chunk_precedent_paquet_ferme"
	}
	return "creee_plus_tot_paquet_ferme"
}

var _ = strconv.Itoa

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

// creationSizeTemoin : la taille de la structure de creation par type (`snapshot.rs::creation_size`, du
// temoin) ; 0 pour les types sans structure.
func creationSizeTemoin(ty uint32) uint32 {
	switch ty {
	case 3, 9, 22:
		return 12
	case 5:
		return 2
	case 6, 14, 17, 21, 24, 28, 47, 48:
		return 4
	case 7, 10:
		return 8
	case 11, 12, 20, 23, 29, 46, 49:
		return 1
	case 13:
		return 136
	case 35:
		return 152
	case 36, 43:
		return 96
	case 37, 38, 39:
		return 104
	case 40:
		return 176
	case 41, 44:
		return 212
	case 42:
		return 168
	}
	return 0
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

type enteteVu struct{ index, gen, ty uint32 }

// balayageDEntetes est `snapshot::entity_table` du temoin : les entrees vivantes retrouvees par leur seul
// en-tete (108 bits) et ses invariants, sans lire creation ni etat. `accepteGen0` : la variante qui garde
// aussi la generation 0.
func balayageDEntetes(pay []byte, accepteGen0 bool) []enteteVu {
	total := len(pay) * 8
	var out []enteteVu
	next := uint32(0)
	for at := 1; at < total-108-128; {
		id := u32Bits(pay, at)
		gen, index := id>>30, id&0x3fffffff
		if (gen == 0 && !accepteGen0) || index >= 8192 || index < next {
			at++
			continue
		}
		ty := u32Bits(pay, at+32)
		flags := u32Bits(pay, at+96) >> 20
		if ty >= 64 || flags&0xe0 != 0 {
			at++
			continue
		}
		n := u32Bits(pay, at+108)
		var ok bool
		switch flags >> 8 {
		case 0:
			ok = u32Bits(pay, at+64) == 0xffffffff && n == 0 && u32Bits(pay, at+140) == 0
		case 1, 3:
			ok = n == creationSizeTemoin(ty)
		}
		if !ok {
			at++
			continue
		}
		out = append(out, enteteVu{index, gen, ty})
		next = index + 1
		at += 108
	}
	return out
}

// TestSnapScan : precision et rappel du balayage d en-tetes (seul, sans lecture de creation ni d etat) contre
// la table que le temoin lit de bout en bout (`<id>_snap.tsv`, lignes `walk`).
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
		{ // NOTRE table de datums (TableDeDatums), seule : sans la chaine des records, sans les NEW lus.
			var w, trouvees, justes, autretype, manquees, enplus int
			for c, W := range snaps {
				data, pks, ok := fc.ChunkAt(c)
				if !ok {
					continue
				}
				for _, pk := range pks {
					if pk.Type != grammar.PacketTypeKeyframe {
						continue
					}
					table, _ := grammar.TableDeDatums(pk.Payload(data))
					w += len(W)
					trouvees += len(table)
					for s, we := range W {
						ti, ok := table[uint32(s)] //nolint:gosec // borne
						switch {
						case !ok:
							manquees++
						case int(ti) == we.ty:
							justes++
						default:
							autretype++
						}
					}
					for s := range table {
						if _, ok := W[int(s)]; !ok {
							enplus++
						}
					}
				}
			}
			fmt.Printf("%s NOTRE TableDeDatums : temoin %d ; table %d ; type juste %d ; autre type %d ; manquees %d ; en plus %d\n", f.id, w, trouvees, justes, autretype, manquees, enplus)
		}
		for _, variante := range []bool{false} {
			var w, trouvees, justes, fausses, manquees int
			for c, W := range snaps {
				data, pks, ok := fc.ChunkAt(c)
				if !ok {
					continue
				}
				for _, pk := range pks {
					if pk.Type != grammar.PacketTypeKeyframe {
						continue
					}
					got := balayageDEntetes(pk.Payload(data), variante)
					vus := map[uint32]enteteVu{}
					for _, e := range got {
						vus[e.index] = e
					}
					w += len(W)
					trouvees += len(got)
					for s, we := range W {
						if g, ok := vus[uint32(s)]; ok && int(g.gen) == we.gen && int(g.ty) == we.ty { //nolint:gosec // borne
							justes++
						} else {
							manquees++
						}
					}
					fausses += len(got) - justesDe(got, W)
				}
			}
			fmt.Printf("%s gen0=%v : temoin %d ; balayage %d ; justes %d ; manquees %d ; fausses %d\n", f.id, variante, w, trouvees, justes, manquees, fausses)
		}
	}
}

func justesDe(got []enteteVu, W map[int]entiteTemoin) int {
	n := 0
	for _, g := range got {
		if we, ok := W[int(g.index)]; ok && int(g.gen) == we.gen && int(g.ty) == we.ty {
			n++
		}
	}
	return n
}

type enteteVuAt struct {
	enteteVu
	at int
}

// enteteSurLePlan lit l en-tete candidat a `at` selon les invariants du temoin ; ok faux sinon.
func enteteSurLePlan(pay []byte, at int, accepteGen0 bool) (enteteVuAt, bool) {
	id := u32Bits(pay, at)
	gen, index := id>>30, id&0x3fffffff
	if (gen == 0 && !accepteGen0) || index >= 8192 {
		return enteteVuAt{}, false
	}
	ty := u32Bits(pay, at+32)
	flags := u32Bits(pay, at+96) >> 20
	if ty >= 64 || flags&0xe0 != 0 {
		return enteteVuAt{}, false
	}
	n := u32Bits(pay, at+108)
	var ok bool
	switch flags >> 8 {
	case 0:
		ok = u32Bits(pay, at+64) == 0xffffffff && n == 0 && u32Bits(pay, at+140) == 0
	case 1, 3:
		ok = n == creationSizeTemoin(ty)
	}
	return enteteVuAt{enteteVu{index, gen, ty}, at}, ok
}

// balayageEntier : la passe du temoin (generation non nulle), puis, entre deux entrees retenues, une seconde passe
// qui accepte la generation 0 pour un index STRICTEMENT compris entre les deux voisines.
func balayageEntier(pay []byte) []enteteVu {
	total := len(pay) * 8
	var passe1 []enteteVuAt
	next := uint32(0)
	for at := 1; at < total-108-128; {
		e, ok := enteteSurLePlan(pay, at, false)
		if !ok || e.index < next {
			at++
			continue
		}
		passe1 = append(passe1, e)
		next = e.index + 1
		at += 108
	}
	// bornes fictives : avant la premiere et apres la derniere
	bornes := append([]enteteVuAt{{enteteVu{index: 0}, 0}}, passe1...)
	bornes = append(bornes, enteteVuAt{enteteVu{index: 8192}, total})
	out := make([]enteteVu, 0, len(passe1)+64)
	for k := 0; k+1 < len(bornes); k++ {
		a, b := bornes[k], bornes[k+1]
		if k > 0 {
			out = append(out, a.enteteVu)
		}
		lo := a.index + 1
		if k == 0 {
			lo = 0
		}
		debut := a.at + 108
		if k == 0 {
			debut = 1
		}
		next := lo
		for at := debut; at < b.at-108 && at < total-108-128; {
			e, ok := enteteSurLePlan(pay, at, true)
			if !ok || e.gen != 0 || e.index < next || e.index >= b.index {
				at++
				continue
			}
			out = append(out, e.enteteVu)
			next = e.index + 1
			at += 108
		}
	}
	return out
}

// TestSnapScanEntier : le balayage a deux passes contre la table du temoin.
func TestSnapScanEntier(t *testing.T) {
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
		var w, trouvees, justes, manquees, fausses int
		for c, W := range snaps {
			data, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != grammar.PacketTypeKeyframe {
					continue
				}
				got := balayageEntier(pk.Payload(data))
				vus := map[uint32]enteteVu{}
				for _, e := range got {
					vus[e.index] = e
				}
				w += len(W)
				trouvees += len(got)
				for s, we := range W {
					if g, ok := vus[uint32(s)]; ok && int(g.gen) == we.gen && int(g.ty) == we.ty { //nolint:gosec // borne
						justes++
					} else {
						manquees++
					}
				}
				fausses += len(got) - justesDe(got, W)
			}
		}
		fmt.Printf("%s DEUX PASSES : temoin %d ; balayage %d ; justes %d ; manquees %d ; fausses %d\n", f.id, w, trouvees, justes, manquees, fausses)
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
