//go:build research

package grammar

// repliques_research_test.go — INSTRUMENT : les repliques (genre 75) et les annonces (genre 76) de la
// vue A, comptees sur des films reels, a confronter au temoin (la bibliotheque libfilmshell de Den, MIT,
// evenements `Chatter` et `VoiceLine`).
//
// POUR CHAQUE FILM, le banc joue la passe des tetes avec un lecteur de repliques inscrit
// ([canalDeRepliques]) et compte, sur les trames delta :
//
//	- les messages lus de chaque genre, ceux qui portent un son, et les trames dont la vue A s est
//	  arretee avant son terminateur (les messages qu elles portent apres l arret ne se lisent pas) ;
//	- genre 75 : par locuteur (la reference 0 de l en-tete, brute, et l index de joueur de la charge) et
//	  par identifiant de son ;
//	- genre 76 : par identifiant de son, et par bit de destinataire pose (le joueur p est le bit
//	  0x80000000 >> p, d apres le temoin).
//
// USAGE (lecture seule, aucun film = saut propre) :
//
//	REPLIQUES_FILMS=<racine data/cache> REPLIQUES_IDS='<id8>=<carte>;<id8>=<carte>' \
//	  REPLIQUES_SORTIE=<dossier> go test -tags=research ./internal/grammar -run TestRepliquesDesFilms -v
//
// `<carte>` est le nom de la carte au catalogue des bornes ; une ligne TSV par comptage est ecrite dans
// `<dossier>/<id8>.repliques.tsv` quand REPLIQUES_SORTIE est pose, et toujours journalisee.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// canalDeComptageDesRepliques compte les genres 75 et 76 de la vue A rangee de chaque trame.
type canalDeComptageDesRepliques struct {
	trames, arretees               int
	repliques, repliquesAvecSon    int
	annonces, annoncesAvecSon      int
	parLocuteur, parSonReplique    map[string]int
	parSonAnnonce, parDestinataire map[string]int
}

func nouveauCanalDeComptageDesRepliques() *canalDeComptageDesRepliques {
	return &canalDeComptageDesRepliques{parLocuteur: map[string]int{}, parSonReplique: map[string]int{},
		parSonAnnonce: map[string]int{}, parDestinataire: map[string]int{}}
}

func (*canalDeComptageDesRepliques) Interets() []Interet   { return nil }
func (*canalDeComptageDesRepliques) Clore(BilanDeMarche)   {}
func (*canalDeComptageDesRepliques) LitLesRepliques() bool { return true }

// Tete compte les messages de la vue A rangee du paquet.
func (c *canalDeComptageDesRepliques) Tete(p *lecture.Paquet) {
	c.trames++
	if p.VueA.Etat != lecture.VueTerminee {
		c.arretees++
	}
	for _, m := range p.VueA.Repliques {
		c.repliques++
		loc := "ref=absente"
		if m.Drapeaux&lecture.RepliqueRef0 != 0 {
			loc = fmt.Sprintf("ref=%d", m.Ref)
		}
		if m.Joueur != lecture.RefAbsente {
			loc += fmt.Sprintf(" joueur=%d", m.Joueur)
		}
		c.parLocuteur[loc]++
		if m.Drapeaux&lecture.RepliqueSon != 0 {
			c.repliquesAvecSon++
			c.parSonReplique[fmt.Sprintf("%08x", m.Son)]++
		}
	}
	for _, m := range p.VueA.Annonces {
		c.annonces++
		if m.Drapeaux&lecture.AnnonceSon != 0 {
			c.annoncesAvecSon++
			c.parSonAnnonce[fmt.Sprintf("%08x", m.Son)]++
		}
		for k := range 32 {
			if m.Destinataires&(0x80000000>>uint(k)) != 0 {
				c.parDestinataire[fmt.Sprintf("bit=%d", k)]++
			}
		}
	}
}

// lignes rend le comptage en lignes `rubrique<TAB>cle<TAB>compte`, triees.
func (c *canalDeComptageDesRepliques) lignes() []string {
	out := []string{
		fmt.Sprintf("total\ttrames_delta\t%d", c.trames),
		fmt.Sprintf("total\ttrames_vue_a_arretee\t%d", c.arretees),
		fmt.Sprintf("genre75\tmessages\t%d", c.repliques),
		fmt.Sprintf("genre75\tavec_son\t%d", c.repliquesAvecSon),
		fmt.Sprintf("genre76\tmessages\t%d", c.annonces),
		fmt.Sprintf("genre76\tavec_son\t%d", c.annoncesAvecSon),
	}
	for rubrique, m := range map[string]map[string]int{"genre75_locuteur": c.parLocuteur,
		"genre75_son": c.parSonReplique, "genre76_son": c.parSonAnnonce, "genre76_destinataire": c.parDestinataire} {
		cles := make([]string, 0, len(m))
		for k := range m {
			cles = append(cles, k)
		}
		slices.Sort(cles)
		for _, k := range cles {
			out = append(out, fmt.Sprintf("%s\t%s\t%d", rubrique, k, m[k]))
		}
	}
	slices.Sort(out[6:])
	return out
}

func TestRepliquesDesFilms(t *testing.T) {
	racine, liste := os.Getenv("REPLIQUES_FILMS"), os.Getenv("REPLIQUES_IDS")
	if racine == "" || liste == "" {
		t.Skip("REPLIQUES_FILMS ou REPLIQUES_IDS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(bornesDeCartes(t))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		canal := nouveauCanalDeComptageDesRepliques()
		if err := Distribuer(aj14fContexte(film, &entree), canal); err != nil {
			t.Fatalf("%s : distribution : %v", id, err)
		}
		lignes := canal.lignes()
		for _, l := range lignes {
			t.Logf("%s\t%s", id, l)
		}
		if dossier := os.Getenv("REPLIQUES_SORTIE"); dossier != "" {
			chemin := filepath.Join(dossier, id+".repliques.tsv")
			if err := os.WriteFile(chemin, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
				t.Fatalf("%s : ecriture : %v", id, err)
			}
		}
	}
}
