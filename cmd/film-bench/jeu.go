package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Film est une ligne du jeu fixe (`bench/films.tsv`) : un nom neutre et ce qu'on peut dire du film
// sans le designer. Rien ici ne permet de retrouver un identifiant de match ou de joueur.
type Film struct {
	Nom      string
	Mode     string
	Version  string
	Joueurs  int
	Morceaux int
}

// Correspondance relie un nom neutre aux fichiers LOCAUX du film. Elle vit hors du depot.
type Correspondance struct {
	Nom       string
	Dossier   string // dossier des morceaux (`chunk_NN.bin`)
	Manifeste string // manifeste du film (JSON)
	Carte     string // nom de carte ou module, resolu au catalogue de bornes
}

// lignesDeDonnees rend les lignes utiles d'un TSV, decoupees en colonnes : les lignes vides et
// celles qui commencent par `#` sont ignorees, ainsi que la ligne d'en-tete dont la premiere
// colonne est `entete`. Le second rendu donne le numero de chaque ligne, pour les erreurs.
func lignesDeDonnees(chemin, entete string) ([][]string, []int, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin donne par l'operateur
	if err != nil {
		return nil, nil, fmt.Errorf("fichier illisible (%s) : %w", filepath.Base(chemin), err)
	}
	defer func() { _ = f.Close() }()
	var lignes [][]string
	var numeros []int
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		l := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}
		cols := strings.Split(l, "\t")
		if cols[0] == entete {
			continue
		}
		lignes = append(lignes, cols)
		numeros = append(numeros, n)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("lecture de %s : %w", filepath.Base(chemin), err)
	}
	return lignes, numeros, nil
}

// lireJeu lit `bench/films.tsv` : nom, mode, version, joueurs, morceaux.
func lireJeu(chemin string) ([]Film, error) {
	lignes, numeros, err := lignesDeDonnees(chemin, "nom")
	if err != nil {
		return nil, err
	}
	var films []Film
	vus := map[string]bool{}
	for i, c := range lignes {
		if len(c) != 5 {
			return nil, fmt.Errorf("%s ligne %d : 5 colonnes attendues, %d lues", filepath.Base(chemin), numeros[i], len(c))
		}
		j, errJ := strconv.Atoi(c[3])
		m, errM := strconv.Atoi(c[4])
		if errJ != nil || errM != nil || j < 0 || m <= 0 {
			return nil, fmt.Errorf("%s ligne %d : joueurs ou morceaux illisible", filepath.Base(chemin), numeros[i])
		}
		if vus[c[0]] {
			return nil, fmt.Errorf("%s ligne %d : film %q en double", filepath.Base(chemin), numeros[i], c[0])
		}
		vus[c[0]] = true
		films = append(films, Film{Nom: c[0], Mode: c[1], Version: c[2], Joueurs: j, Morceaux: m})
	}
	if len(films) == 0 {
		return nil, fmt.Errorf("%s : aucun film", filepath.Base(chemin))
	}
	return films, nil
}

// lireCorrespondance lit le fichier local : nom, dossier des morceaux, manifeste, carte. Un chemin
// relatif se lit depuis le dossier du fichier de correspondance.
func lireCorrespondance(chemin string) (map[string]Correspondance, error) {
	lignes, numeros, err := lignesDeDonnees(chemin, "nom")
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(chemin)
	out := make(map[string]Correspondance, len(lignes))
	for i, c := range lignes {
		if len(c) != 4 {
			return nil, fmt.Errorf("correspondance ligne %d : 4 colonnes attendues, %d lues", numeros[i], len(c))
		}
		cor := Correspondance{Nom: c[0], Dossier: resoudre(base, c[1]), Manifeste: resoudre(base, c[2]), Carte: strings.TrimSpace(c[3])}
		if cor.Carte == "" {
			return nil, fmt.Errorf("correspondance ligne %d : carte vide pour %s", numeros[i], cor.Nom)
		}
		if _, dup := out[cor.Nom]; dup {
			return nil, fmt.Errorf("correspondance ligne %d : film %q en double", numeros[i], cor.Nom)
		}
		out[cor.Nom] = cor
	}
	return out, nil
}

// resoudre rend `p` tel quel s'il est absolu, sinon relatif a `base`.
func resoudre(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

// verifierChemins refuse une correspondance dont le dossier ou le manifeste manque : c'est une
// erreur, jamais un saut. Le message nomme le film, pas le chemin (celui d'un poste n'a rien a
// faire dans un journal que l'on range).
func verifierChemins(c Correspondance) error {
	if info, err := os.Stat(c.Dossier); err != nil || !info.IsDir() {
		return fmt.Errorf("film %s : dossier des morceaux introuvable", c.Nom)
	}
	if info, err := os.Stat(c.Manifeste); err != nil || info.IsDir() {
		return fmt.Errorf("film %s : manifeste introuvable", c.Nom)
	}
	return nil
}

// retenir filtre le jeu sur les noms de `seulement` (liste separee par des virgules ; vide = tout),
// dans l'ordre du jeu, et refuse un nom absent du jeu.
func retenir(jeu []Film, seulement string) ([]Film, error) {
	if strings.TrimSpace(seulement) == "" {
		return jeu, nil
	}
	voulus := map[string]bool{}
	for n := range strings.SplitSeq(seulement, ",") {
		voulus[strings.TrimSpace(n)] = true
	}
	var out []Film
	for _, f := range jeu {
		if voulus[f.Nom] {
			out = append(out, f)
			delete(voulus, f.Nom)
		}
	}
	for n := range voulus {
		return nil, fmt.Errorf("film %q absent du jeu", n)
	}
	return out, nil
}
