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
