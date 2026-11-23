package main

// etapes.go — LA SUITE ORDONNEE DES ETAPES OBSERVEES, AVEC L'EMPREINTE DE CHAQUE VALEUR.
//
// `-etapes <fichier>` rejoue UNE passe par film, hors de tout chronometre (ses durees sont
// jetees), et range pour chaque etape de `Options.Observe`, dans l'ordre ou elle est publiee, le
// SHA-256 de sa valeur serialisee en JSON. Deux relevés a des nombres de fils differents qui
// portent la meme suite prouvent que l'ordre et le contenu des etapes ne dependent pas du
// parallelisme. Une valeur que JSON ne sait pas ecrire porte une empreinte du texte de l'erreur,
// precedee de `erreur:` : elle reste comparable, mais ne prouve que l'echec.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Etape est une etape observee : son nom et l'empreinte de sa valeur.
type Etape struct {
	Nom        string
	Empreinte  string
	Serialisee bool // faux : la valeur n'a pas pu s'ecrire en JSON, l'empreinte est celle de l'erreur
}

// empreinteDeValeur rend le SHA-256 (hexadecimal) de la serialisation JSON de `v`.
func empreinteDeValeur(v any) Etape {
	blob, err := json.Marshal(v)
	if err != nil {
		blob = []byte(err.Error())
		h := sha256.Sum256(blob)
		return Etape{Empreinte: "erreur:" + hex.EncodeToString(h[:])}
	}
	h := sha256.Sum256(blob)
	return Etape{Empreinte: hex.EncodeToString(h[:]), Serialisee: true}
}

// suiteDesEtapes fait une passe du film, hors chronometre, et rend ses etapes dans l'ordre observe.
func suiteDesEtapes(ctx context.Context, c Cible, v ConfigPasse) ([]Etape, error) {
	var suite []Etape
	v.Etape = func(nom string, valeur any) {
		e := empreinteDeValeur(valeur)
		e.Nom = nom
		suite = append(suite, e)
	}
	if _, err := executer(ctx, c, v); err != nil {
		return nil, err
	}
	return suite, nil
}

// ecrireSuite ecrit la suite d'un film : une ligne par etape, `film`, `rang`, `etape`, `empreinte`.
func ecrireSuite(w io.Writer, film string, suite []Etape) error {
	for i, e := range suite {
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", film, i+1, e.Nom, e.Empreinte); err != nil {
			return err
		}
	}
	return nil
}

// ecrireEtapes range, pour chaque film dans l'ordre du jeu, la suite de ses etapes sous `opt.etapes`.
func ecrireEtapes(ctx context.Context, opt options, cs []Cible, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(opt.etapes), 0o750); err != nil {
		return fmt.Errorf("dossier des etapes : %w", err)
	}
	f, err := os.Create(opt.etapes) //nolint:gosec // sortie choisie par l'operateur
	if err != nil {
		return fmt.Errorf("etapes : %w", err)
	}
	if err := ecrireToutesLesSuites(ctx, f, opt, cs, stderr); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// ecrireToutesLesSuites fait la passe de chaque film et ecrit l'en-tete puis les suites.
func ecrireToutesLesSuites(ctx context.Context, w io.Writer, opt options, cs []Cible, stderr io.Writer) error {
	if _, err := fmt.Fprintf(w, "# film-bench : suite des etapes observees (SHA-256 de la valeur en JSON)\n# fils\t%d\nfilm\trang\tetape\tempreinte\n",
		opt.vitesse().FilsEffectifs()); err != nil {
		return err
	}
	for _, c := range cs {
		dire(stderr, "etapes : %s\n", c.Nom)
		suite, err := suiteDesEtapes(ctx, c, ConfigPasse{Vitesse: opt.vitesse()})
		if err != nil {
			return err
		}
		if err := ecrireSuite(w, c.Nom, suite); err != nil {
			return err
		}
		liberer()
	}
	return nil
}
