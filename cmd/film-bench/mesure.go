package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

const (
	// etapeFinDuBalayage est la derniere etape de `replay.BuildFromFilmSteps` : elle separe le
	// decodage (film vers faits) de la publication (faits vers document).
	etapeFinDuBalayage = "clockOrigin"
)

// ConfigPasse regle une passe de mesure : la vitesse demandee au decodeur et, hors chronometre
// seulement, un crochet qui recoit chaque etape observee avec sa valeur (jamais pose dans une passe
// chronometree, ni dans celle du pic de memoire).
type ConfigPasse struct {
	Vitesse replay.OptionsVitesse
	Etape   func(etape string, v any)
}

// Cible est un film pret a mesurer : ses fichiers, ses metadonnees de manifeste et l'entree de
// carte sous laquelle il se decode.
type Cible struct {
	Nom     string
	Dossier string
	Meta    []decfilm.ChunkMeta
	Entree  decfilm.MapQuantEntry
}

// Resultat est ce qu'une passe produit : ses durees, et le document et les faits qu'elle a rendus
// (les empreintes se calculent plus tard, hors chronometre).
type Resultat struct {
	Passe Passe
	Doc   replay.ReplayDocument
	Faits *replay.FilmFactsFile
}

// preparer lit le manifeste d'un film et resout sa carte au catalogue.
func preparer(c Correspondance, cat *decfilm.MapQuantCatalog) (Cible, error) {
	meta, err := lireManifeste(c.Manifeste)
	if err != nil {
		return Cible{}, fmt.Errorf("film %s : %w", c.Nom, err)
	}
	entree, err := entreeDeCarte(cat, c.Carte)
	if err != nil {
		return Cible{}, fmt.Errorf("film %s : %w", c.Nom, err)
	}
	return Cible{Nom: c.Nom, Dossier: c.Dossier, Meta: meta, Entree: entree}, nil
}

// lireManifeste rend l'index des chunks d'un manifeste (numero, type, debut), dans son ordre.
func lireManifeste(chemin string) ([]decfilm.ChunkMeta, error) {
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin donne par l'operateur
	if err != nil {
		return nil, fmt.Errorf("manifeste illisible : %w", err)
	}
	var mf struct {
		Chunks []struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(brut, &mf); err != nil {
		return nil, fmt.Errorf("manifeste invalide : %w", err)
	}
	meta := make([]decfilm.ChunkMeta, len(mf.Chunks))
	for i, c := range mf.Chunks {
		meta[i] = decfilm.ChunkMeta{Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS}
	}
	return meta, nil
}

// entreeDeCarte rend l'entree du catalogue dont le nom de carte, sinon le module, est `carte`.
func entreeDeCarte(cat *decfilm.MapQuantCatalog, carte string) (decfilm.MapQuantEntry, error) {
	if e, err := cat.Lookup(carte); err == nil {
		return e, nil
	}
	for _, e := range cat.Maps {
		if e.Module == carte {
			return e, nil
		}
	}
	return decfilm.MapQuantEntry{}, fmt.Errorf("aucune carte %q au catalogue de bornes", carte)
}

// fichiersDesMorceaux rend les `chunk_NN.bin` d'un dossier, dans l'ordre de leur numero.
func fichiersDesMorceaux(dossier string) ([]string, error) {
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		return nil, fmt.Errorf("dossier illisible : %w", err)
	}
	type numerote struct {
		n    int
		path string
	}
	var morceaux []numerote
	for _, e := range entrees {
		if n, ok := filmcache.NumeroDuChunk(e.Name()); ok && !e.IsDir() {
			morceaux = append(morceaux, numerote{n, filepath.Join(dossier, e.Name())})
		}
	}
	sort.Slice(morceaux, func(i, j int) bool { return morceaux[i].n < morceaux[j].n })
	out := make([]string, len(morceaux))
	for i, m := range morceaux {
		out[i] = m.path
	}
	return out, nil
}

// lireOctets lit les fichiers donnes, en memoire, HORS chronometre.
func lireOctets(chemins []string) ([][]byte, error) {
	out := make([][]byte, len(chemins))
	for i, p := range chemins {
		b, err := os.ReadFile(p) //nolint:gosec // chemin issu du dossier du film
		if err != nil {
			return nil, fmt.Errorf("morceau illisible : %w", err)
		}
		out[i] = b
	}
	return out, nil
}

// chronometrerDecompression rend la duree de `decfilm.Inflate` sur chaque morceau, et la somme des
// octets rendus (consommee par l'appelant pour que le travail ne soit pas elidable).
func chronometrerDecompression(brut [][]byte) (time.Duration, int) {
	runtime.GC()
	taille := 0
	debut := time.Now()
	for _, b := range brut {
		taille += len(decfilm.Inflate(b))
	}
	return time.Since(debut), taille
}

// executer fait UNE passe de mesure d'un film : decompression a part, puis chargement, decodage et
// publication, avec un total d'un seul tenant. Les options sont minimales (carte seule : ni source
// de degat, ni tableau, ni objectifs, ni zones) ; le nombre de fils vient de `cfg` et se pose sur
// `opt`, ci-dessous, et nulle part ailleurs.
func executer(ctx context.Context, c Cible, cfg ConfigPasse) (Resultat, error) {
	chemins, err := fichiersDesMorceaux(c.Dossier)
	if err != nil {
		return Resultat{}, fmt.Errorf("film %s : %w", c.Nom, err)
	}
	brut, err := lireOctets(chemins)
	if err != nil {
		return Resultat{}, fmt.Errorf("film %s : %w", c.Nom, err)
	}
	var r Resultat
	var taille int
	r.Passe.Decompression, taille = chronometrerDecompression(brut)
	if taille == 0 {
		return Resultat{}, fmt.Errorf("film %s : aucun octet decompresse", c.Nom)
	}

	runtime.GC()
	debutTotal := time.Now()
	film, err := decfilm.LoadDir(c.Dossier, c.Meta)
	if err != nil {
		return Resultat{}, fmt.Errorf("film %s : chargement : %w", c.Nom, err)
	}
	r.Passe.Chargement = time.Since(debutTotal)

	var finBalayage time.Time
	opt := replay.Options{MapQuant: &c.Entree, Vitesse: cfg.Vitesse}
	opt.Observe = func(etape string, v any) {
		if survolEtape != nil {
			survolEtape(etape)
		}
		if cfg.Etape != nil {
			cfg.Etape(etape, v)
		}
		if etape == etapeFinDuBalayage && finBalayage.IsZero() {
			finBalayage = time.Now()
		}
	}
	debut := time.Now()
	doc, faits, err := replay.BuildFromFilmAvecFaits(ctx, c.Nom, "halo_infinite", film, opt)
	fin := time.Now()
	if err != nil {
		return Resultat{}, fmt.Errorf("film %s : cuisson : %w", c.Nom, err)
	}
	if faits == nil || finBalayage.IsZero() {
		return Resultat{}, fmt.Errorf("film %s : faits absents ou fin de balayage non observee", c.Nom)
	}
	r.Passe.Decodage = finBalayage.Sub(debut)
	r.Passe.Publication = fin.Sub(finBalayage)
	r.Passe.Total = fin.Sub(debutTotal)
	r.Doc, r.Faits = doc, faits
	return r, nil
}

// passeChronometree fait une passe et n'en garde que les durees : le document et les faits sont
// abandonnes au retour, donc liberables avant que l appelant ne rende la memoire a l OS.
func passeChronometree(ctx context.Context, c Cible, v replay.OptionsVitesse) (Passe, error) {
	r, err := executer(ctx, c, ConfigPasse{Vitesse: v})
	return r.Passe, err
}
