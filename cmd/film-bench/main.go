// Commande film-bench : releve de vitesse du decodeur de film (releve A : le decodeur seul,
// options minimales), version par version.
//
//	GOWORK=off go run ./cmd/film-bench -films bench/films.tsv -correspondance <fichier hors depot> \
//	  -catalogue <map_quant_bounds.json> -passes 5 -sortie bench/releves/<version>.tsv
//
// Pour chaque film : decompression, chargement, decodage, publication, total, pic de memoire et
// empreintes. Le protocole et le sens de chaque colonne sont dans `bench/README.md`. Le total de
// ce releve N'EST PAS celui d'une generation complete de rejeu (qui lit aussi une base).
//
// `-fils N` demande N fils au decodeur (colonne `fils` du releve ; le nombre employe est plafonne par
// GOMAXPROCS). `-etapes <fichier>` ecrit, apres les mesures et hors de tout chronometre, la suite
// ordonnee des etapes observees de chaque film avec le SHA-256 de chaque valeur : c'est la preuve de
// l'ordre dans lequel le decodeur les publie.
//
// L'outil n'emploie que des entrees publiques du decodeur (`decfilm`, `replay`, `filmcache`,
// `memguard`). Les deux options de fils (`replay.Options.Vitesse`) n'existent qu'a partir de la
// version qui les porte : pour relever une version anterieure, l'outil se copie dans son arbre apres
// retrait de ces deux options.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// options est la ligne de commande.
type options struct {
	films          string
	correspondance string
	catalogue      string
	sortie         string
	seulement      string
	passes         int
	plafondGiB     int
	enfant         bool
	profilCPU      string
	profilTas      string
	fils           int
	etapes         string
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// lireOptions decode la ligne de commande et refuse une combinaison inutilisable.
func lireOptions(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("film-bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.films, "films", "bench/films.tsv", "jeu fixe de films (noms neutres)")
	fs.StringVar(&o.correspondance, "correspondance", "", "fichier LOCAL hors depot : nom neutre, dossier des morceaux, manifeste, carte")
	fs.StringVar(&o.catalogue, "catalogue", "", "map_quant_bounds.json")
	fs.StringVar(&o.sortie, "sortie", "", "TSV du releve a ecrire (par exemple bench/releves/vX.Y.Z.tsv) ; vide = resume seul")
	fs.StringVar(&o.seulement, "seulement", "", "noms de films a mesurer, separes par des virgules (defaut : tout le jeu)")
	fs.IntVar(&o.passes, "passes", 5, "passes chronometrees, apres une passe d'echauffement ecartee")
	fs.IntVar(&o.plafondGiB, "plafond-gio", 2, "plafond de memoire d'un film, en Gio (mesure sur le poste)")
	fs.BoolVar(&o.enfant, "enfant", false, "usage interne : une passe d'un film, sous sentinelle, pour son pic de memoire")
	fs.StringVar(&o.profilCPU, "profil-cpu", "", "profil CPU pprof d une passe d un seul film (avec -seulement)")
	fs.StringVar(&o.profilTas, "profil-tas", "", "profil du tas pprof d une passe d un seul film (avec -seulement) ; <fichier>.pic = plus haut tas")
	fs.IntVar(&o.fils, "fils", 1, "fils que le decodeur peut employer (plafonne par GOMAXPROCS et par replay.MaxFils) ; 1 = comportement actuel")
	fs.StringVar(&o.etapes, "etapes", "", "TSV de la suite ordonnee des etapes observees, avec le SHA-256 de chaque valeur, d'une passe hors chronometre par film")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	switch {
	case o.correspondance == "":
		return o, errors.New("-correspondance est requis")
	case o.catalogue == "":
		return o, errors.New("-catalogue est requis")
	case o.passes < 1:
		return o, errors.New("-passes doit etre au moins 1")
	case o.plafondGiB < 1:
		return o, errors.New("-plafond-gio doit etre au moins 1")
	case o.enfant && strings.Contains(o.seulement, ","):
		return o, errors.New("-enfant mesure un seul film")
	case o.fils < 1:
		return o, errors.New("-fils doit etre au moins 1")
	case o.etapes != "" && (o.enfant || o.profilDemande()):
		return o, errors.New("-etapes est incompatible avec -enfant, -profil-cpu et -profil-tas")
	}
	if err := o.verifierProfil(); err != nil {
		return o, err
	}
	return o, nil
}

// vitesse rend les options de vitesse demandees au decodeur.
func (o options) vitesse() replay.OptionsVitesse { return replay.OptionsVitesse{Fils: o.fils} }

// refuserSousData refuse une sortie rangee sous un dossier `data` : les donnees de l'hote ne
// s'ecrivent pas.
func refuserSousData(sortie string) error {
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Clean(sortie)), "/") {
		if strings.EqualFold(seg, "data") {
			return errors.New("-sortie sous un dossier data refusee")
		}
	}
	return nil
}

// run : le decodeur journalise des centaines de lignes par film ; elles couteraient du temps dans
// les passes chronometrees, donc seules les erreurs passent.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	slog.SetLogLoggerLevel(slog.LevelError)
	opt, err := lireOptions(args, stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			dire(stderr, "film-bench : %v\n", err)
		}
		return 2
	}
	if opt.profilDemande() {
		return profiler(ctx, opt, stdout, stderr)
	}
	if opt.enfant {
		return executerEnfant(ctx, opt, stdout, stderr)
	}
	if err := lancer(ctx, opt, stdout, stderr); err != nil {
		dire(stderr, "film-bench : %v\n", err)
		return 1
	}
	return 0
}

// cibles lit le jeu, la correspondance et le catalogue, verifie chaque film retenu et rend ses
// cibles dans l'ordre du jeu.
func cibles(opt options) ([]Film, []Cible, error) {
	jeu, err := lireJeu(opt.films)
	if err != nil {
		return nil, nil, err
	}
	films, err := retenir(jeu, opt.seulement)
	if err != nil {
		return nil, nil, err
	}
	cors, err := lireCorrespondance(opt.correspondance)
	if err != nil {
		return nil, nil, err
	}
	cat, err := decfilm.LoadMapQuantCatalog(opt.catalogue)
	if err != nil {
		return nil, nil, err
	}
	out := make([]Cible, 0, len(films))
	for _, f := range films {
		cor, ok := cors[f.Nom]
		if !ok {
			return nil, nil, fmt.Errorf("film %s : absent de la correspondance", f.Nom)
		}
		if err := verifierChemins(cor); err != nil {
			return nil, nil, err
		}
		fichiers, err := fichiersDesMorceaux(cor.Dossier)
		if err != nil {
			return nil, nil, fmt.Errorf("film %s : %w", f.Nom, err)
		}
		if len(fichiers) != f.Morceaux {
			return nil, nil, fmt.Errorf("film %s : %d morceaux sur disque, %d dans le jeu fixe", f.Nom, len(fichiers), f.Morceaux)
		}
		c, err := preparer(cor, cat)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, c)
	}
	return films, out, nil
}

// lancer fait le releve : un fils par film pour le pic et les empreintes, puis les passes
// chronometrees en un seul processus, l'ordre des films tourne a chaque passe.
func lancer(ctx context.Context, opt options, stdout, stderr io.Writer) error {
	for _, sortie := range []string{opt.sortie, opt.etapes} {
		if sortie == "" {
			continue
		}
		if err := refuserSousData(sortie); err != nil {
			return err
		}
	}
	films, cs, err := cibles(opt)
	if err != nil {
		return err
	}
	resultats := make([]ResultatFilm, len(films))
	for i, f := range films {
		dire(stderr, "pic de memoire : %s\n", f.Nom)
		rap, err := lancerEnfant(ctx, opt, f.Nom)
		if err != nil {
			return err
		}
		resultats[i] = ResultatFilm{Film: f, Fils: opt.vitesse().FilsEffectifs(), PicMio: float64(rap.PicOctets) / octetsParMio,
			EmpreinteDoc: rap.EmpreinteDoc, EmpreinteFaits: rap.EmpreinteFaits}
	}
	passes, err := chronometrer(ctx, cs, opt, stderr)
	if err != nil {
		return err
	}
	for i := range resultats {
		resultats[i].Stats = resumerPasses(passes[films[i].Nom])
	}
	if err := ecrireResume(stdout, resultats); err != nil {
		return err
	}
	if opt.etapes != "" {
		if err := ecrireEtapes(ctx, opt, cs, stderr); err != nil {
			return err
		}
	}
	if opt.sortie == "" {
		return nil
	}
	return rangerReleve(ctx, opt, resultats)
}

// chronometrer fait l'echauffement (passe 0, ecartee) puis `n` passes ; l'ordre des films tourne
// d'une passe a l'autre. Un seul film est en memoire a la fois.
func chronometrer(ctx context.Context, cs []Cible, opt options, stderr io.Writer) (map[string][]Passe, error) {
	noms := make([]string, len(cs))
	parNom := make(map[string]Cible, len(cs))
	for i, c := range cs {
		noms[i] = c.Nom
		parNom[c.Nom] = c
	}
	out := make(map[string][]Passe, len(cs))
	n := opt.passes
	for p := 0; p <= n; p++ {
		if p == 0 {
			dire(stderr, "passe d echauffement (ecartee)\n")
		} else {
			dire(stderr, "passe %d/%d\n", p, n)
		}
		for _, nom := range tourner(noms, p) {
			ps, err := passeChronometree(ctx, parNom[nom], opt.vitesse())
			if err != nil {
				return nil, err
			}
			if p > 0 {
				out[nom] = append(out[nom], ps)
			}
			liberer()
		}
	}
	return out, nil
}

// liberer rend la memoire d'un film a l'OS avant le suivant, hors chronometre.
func liberer() { debug.FreeOSMemory() }

// rangerReleve ecrit le TSV de la sortie.
func rangerReleve(ctx context.Context, opt options, resultats []ResultatFilm) error {
	if err := os.MkdirAll(filepath.Dir(opt.sortie), 0o750); err != nil {
		return fmt.Errorf("dossier de sortie : %w", err)
	}
	f, err := os.Create(opt.sortie) //nolint:gosec // sortie choisie par l'operateur
	if err != nil {
		return fmt.Errorf("sortie : %w", err)
	}
	ent := Entete{Date: time.Now(), Arbre: arbreGit(ctx), Machine: releverMachine(), Passes: opt.passes, PlafondGiB: opt.plafondGiB}
	if err := ecrireReleve(f, ent, resultats); err != nil {
		_ = f.Close()
		return fmt.Errorf("ecriture du releve : %w", err)
	}
	return f.Close()
}
