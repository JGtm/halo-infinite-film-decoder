//go:build research

package divergence

// vuea_research_test.go : OU LA VUE A DIFFERE DU TEMOIN (plan marche complete, A2.V, mesure B1). Pour
// chaque paquet des films donnes, la vue A lue par NOTRE marche est confrontee a celle du temoin, message
// par message ; le premier message ou les deux cessent de coincider est classe.
//
//	VUEA_RACINE=<film_chunks> VUEA_FILMS=<liste id\tbuild\t..\tniveau> VUEA_TEMOIN=<dossier des TSV>
//	VUEA_CARTE=<map_quant_bounds.json> VUEA_SORTIE=<dossier> [VUEA_ID=<id>] \
//	  go test -tags=research ./research/divergence/ -run TestVueA -v
//
// Sortie : `<sortie>/<id>_vuea.tsv` (une ligne par paquet en ecart ou non ferme chez nous) et un resume
// `<sortie>/<id>_vuea_resume.txt`.
//
// Classes (premier ecart, dans l ordre du flux de la vue A) :
//
//	ok              genres egaux, fin de vue A egale, debut de vue B egal
//	genre_diverge   les listes de genres cessent de coincider a l indice i (le message i-1 est le dernier
//	                message commun : sa FIN differe d un cote a l autre)
//	nous_en_plus    nos genres prolongent ceux du temoin, qui a lu son terminateur
//	nous_arretes    nos genres sont un prefixe de ceux du temoin (arret sur un message non lu)
//	fin_differente  genres egaux, vue A terminee des deux cotes, fins a des bits differents
//	vueB_decale     genres et fin de vue A egaux, mais le debut de vue B retenu par la marche differe
//	                (repli de localisation : l ecart est en aval de la vue A)

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

type ligneVueA struct {
	classe                string
	i                     int // indice du premier ecart
	gNous, gTem           int
	culprit               int // genre du dernier message commun (-1 : la tete)
	wDebut, wFin          int // bits du message culprit chez le temoin : debut (champ de genre), fin
	finANous, entityStart int
	vueBNous              int
	nNous, nTem           int
	etatA                 lecture.EtatDeVue
	debutMeth             lecture.DebutDeVueB
}

// classerVueA : voir l en-tete.
func classerVueA(p *lecture.Paquet, t *paquetTemoin) ligneVueA {
	g := p.VueA.Genres
	l := ligneVueA{i: -1, gNous: -1, gTem: -1, culprit: -1, wDebut: -1, wFin: -1, entityStart: t.entityStart,
		vueBNous: int(p.VueB.Debut), nNous: len(g), nTem: len(t.evts), etatA: p.VueA.Etat, debutMeth: p.Debut,
		finANous: int(p.VueA.Debut + p.VueA.Bits)}
	if p.VueA.Etat != lecture.VueTerminee {
		l.finANous = -1
	}
	cul := func(i int) { // le message i-1 du temoin
		if i > 0 && i-1 < len(t.evts) {
			l.culprit = t.evts[i-1].genre
			l.wDebut = t.evts[i-1].bit
			if i < len(t.evts) {
				l.wFin = t.evts[i].bit - 1
			} else if t.entityStart >= 0 {
				l.wFin = t.entityStart - 1
			}
		}
	}
	m := min(len(g), len(t.evts))
	for i := 0; i < m; i++ {
		if int(g[i]) != t.evts[i].genre {
			l.classe, l.i, l.gNous, l.gTem = "genre_diverge", i, int(g[i]), t.evts[i].genre
			cul(i)
			return l
		}
	}
	switch {
	case len(g) > len(t.evts):
		l.classe, l.i, l.gNous = "nous_en_plus", len(t.evts), int(g[len(t.evts)])
		cul(len(t.evts))
	case len(g) < len(t.evts):
		l.classe, l.i, l.gTem = "nous_arretes", len(g), t.evts[len(g)].genre
		cul(len(g))
	case p.VueA.Etat != lecture.VueTerminee:
		l.classe = "arret_meme_nombre"
		if len(g) > 0 {
			l.culprit = int(g[len(g)-1]) // le genre sur lequel nous nous arretons
		}
	case t.entityStart >= 0 && l.finANous != t.entityStart:
		l.classe = "fin_differente"
		if len(g) > 0 {
			l.culprit, l.wDebut = int(g[len(g)-1]), t.evts[len(g)-1].bit
		}
		l.wFin = t.entityStart - 1
	case t.entityStart >= 0 && int(p.VueB.Debut) != t.entityStart:
		l.classe = "vueB_decale"
	default:
		l.classe = "ok"
	}
	return l
}

func TestVueA(t *testing.T) {
	racine, tdir, sortie, liste := os.Getenv("VUEA_RACINE"), os.Getenv("VUEA_TEMOIN"), os.Getenv("VUEA_SORTIE"), os.Getenv("VUEA_FILMS")
	if racine == "" || tdir == "" || sortie == "" || liste == "" {
		t.Skip("VUEA_RACINE, VUEA_FILMS, VUEA_TEMOIN, VUEA_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	var cat *profile.MapQuantCatalog
	if c := os.Getenv("VUEA_CARTE"); c != "" {
		var err error
		if cat, err = profile.LoadMapQuantCatalog(c); err != nil {
			t.Fatal(err)
		}
	}
	films, err := lireLesFilms(liste)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sortie, 0o755); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
	for _, f := range films {
		if seul := os.Getenv("VUEA_ID"); seul != "" && seul != f.id {
			continue
		}
		if err := vueAUnFilm(racine, tdir, sortie, f, cat); err != nil {
			t.Fatalf("film %s : %v", f.id, err)
		}
	}
}

func vueAUnFilm(racine, tdir, sortie string, f filmDeTest, cat *profile.MapQuantCatalog) error {
	fc, err := ouvrirFilm(racine, f.id, f.niveau, cat)
	if err != nil {
		return err
	}
	verdicts, err := lireLesVerdicts(fc)
	if err != nil {
		return err
	}
	tem, err := ouvrirTemoin(filepath.Join(tdir, f.id+".tsv"))
	if err != nil {
		return err
	}
	defer tem.fermer()
	out, err := os.Create(filepath.Join(sortie, f.id+"_vuea.tsv")) //nolint:gosec // sortie de l operateur
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	fmt.Fprint(out, "chunk\tpaquet\ttaille\tnous_ferme\ttemoin_ferme\tcause_nous\tdebut_meth\tetat_A\tclasse\ti\tg_nous\tg_tem\tgenre_culprit\tw_debut\tw_fin\tfinA_nous\tentityStart_tem\tvueB_nous\tn_nous\tn_tem\n")
	compte := map[string]int{}
	var cur *paquetTemoin
	suivant := func() (err error) { cur, err = tem.suivant(); return }
	if err := suivant(); err != nil {
		return err
	}
	for p, err := range fc.Trames(grammar.NouvelleObservation()) {
		if err != nil {
			return err
		}
		if len(p.Payload) < 2 {
			continue
		}
		for cur != nil && (cur.chunk < p.Chunk || cur.chunk == p.Chunk && cur.paquet < p.Index) {
			if err := suivant(); err != nil {
				return err
			}
		}
		if cur == nil || cur.chunk != p.Chunk || cur.paquet != p.Index {
			continue
		}
		v := verdicts[cleDePaquet{p.Chunk, p.Index}]
		l := classerVueA(p, cur)
		compte[fmt.Sprintf("%s|%s", l.classe, oui(v.fermee))]++
		if l.classe != "ok" || !v.fermee {
			fmt.Fprintf(out, "%d\t%d\t%d\t%s\t%s\t%s\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", p.Chunk, p.Index,
				len(p.Payload), oui(v.fermee), oui(cur.ferme()), nettoyer(v.cause), l.debutMeth, l.etatA, l.classe, l.i, l.gNous,
				l.gTem, l.culprit, l.wDebut, l.wFin, l.finANous, l.entityStart, l.vueBNous, l.nNous, l.nTem)
		}
		if err := suivant(); err != nil {
			return err
		}
	}
	cles := make([]string, 0, len(compte))
	for k := range compte {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	var sb strings.Builder
	for _, k := range cles {
		fmt.Fprintf(&sb, "%s\t%d\n", k, compte[k])
	}
	return os.WriteFile(filepath.Join(sortie, f.id+"_vuea_resume.txt"), []byte(sb.String()), 0o644) //nolint:gosec // sortie
}
