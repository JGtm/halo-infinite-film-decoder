package main

// enfant.go — LE PIC DE MEMOIRE SE PREND DANS UN PROCESSUS NEUF, UN FILM A LA FOIS.
//
// Le pic d'un processus qui a deja traite d'autres films melange leur tas ; l'outil se relance donc
// lui-meme (`-enfant -seulement <film>`), le fils fait une passe sous la sentinelle de `memguard`
// et rend une ligne JSON sur sa sortie standard : le pic, et les empreintes du document et des
// faits, calculees APRES la lecture du pic et hors de tout chronometre.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/memguard"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// Codes de sortie du fils.
const (
	sortieEchec     = 1
	sortiePlafond   = 3
	nomDeLOutil     = "film-bench"
	octetsParMio    = 1 << 20
	plafondMaxLigne = 64 << 10
)

// RapportEnfant est la ligne JSON que le fils rend.
type RapportEnfant struct {
	PicOctets      uint64 `json:"pic_octets"`
	Depasse        bool   `json:"depasse,omitempty"`
	EmpreinteDoc   string `json:"empreinte_document,omitempty"`
	EmpreinteFaits string `json:"empreinte_faits,omitempty"`
}

// empreintes rend les SHA-256 (hexadecimal) du document serialise en JSON et du fichier de faits.
func empreintes(r Resultat) (string, string, error) {
	blob, err := json.Marshal(r.Doc)
	if err != nil {
		return "", "", fmt.Errorf("serialisation du document : %w", err)
	}
	faits, err := replay.EncodeFilmFactsFile(r.Faits)
	if err != nil {
		return "", "", fmt.Errorf("serialisation des faits : %w", err)
	}
	d, f := sha256.Sum256(blob), sha256.Sum256(faits)
	return hex.EncodeToString(d[:]), hex.EncodeToString(f[:]), nil
}

// executerEnfant est le travail du fils : une passe du film `opt.seulement`, sous le plafond de
// mesure. Il rend le code de sortie du processus.
func executerEnfant(ctx context.Context, opt options, stdout, stderr io.Writer) int {
	cible, err := cibleUnique(opt)
	if err != nil {
		dire(stderr, "film-bench (fils) : %v\n", err)
		return sortieEchec
	}
	enc := json.NewEncoder(stdout)
	garde := memguard.Arm(ctx, nomDeLOutil, opt.plafondGiB, func(pic uint64) {
		_ = enc.Encode(RapportEnfant{PicOctets: pic, Depasse: true})
		os.Exit(sortiePlafond)
	})
	defer garde.Disarm()
	res, err := executer(ctx, cible, ConfigPasse{Vitesse: opt.vitesse()})
	pic := garde.Peak() // lu AVANT les empreintes : elles ne font pas partie du pic mesure
	if err != nil {
		dire(stderr, "film-bench (fils) : %v\n", err)
		return sortieEchec
	}
	ed, ef, err := empreintes(res)
	if err != nil {
		dire(stderr, "film-bench (fils) : %v\n", err)
		return sortieEchec
	}
	if err := enc.Encode(RapportEnfant{PicOctets: pic, EmpreinteDoc: ed, EmpreinteFaits: ef}); err != nil {
		dire(stderr, "film-bench (fils) : %v\n", err)
		return sortieEchec
	}
	return 0
}

// cibleUnique prepare le film unique que le fils doit mesurer.
func cibleUnique(opt options) (Cible, error) {
	cors, err := lireCorrespondance(opt.correspondance)
	if err != nil {
		return Cible{}, err
	}
	cor, ok := cors[opt.seulement]
	if !ok {
		return Cible{}, fmt.Errorf("film %q absent de la correspondance", opt.seulement)
	}
	if err := verifierChemins(cor); err != nil {
		return Cible{}, err
	}
	cat, err := decfilm.LoadMapQuantCatalog(opt.catalogue)
	if err != nil {
		return Cible{}, err
	}
	return preparer(cor, cat)
}

// lancerEnfant relance l'outil sur UN film et rend son rapport. Le fils herite de la
// correspondance, du catalogue et du plafond du parent.
func lancerEnfant(ctx context.Context, opt options, nom string) (RapportEnfant, error) {
	exe, err := os.Executable()
	if err != nil {
		return RapportEnfant{}, fmt.Errorf("outil introuvable pour se relancer : %w", err)
	}
	cmd := exec.CommandContext(ctx, exe, "-enfant", "-seulement", nom,
		"-correspondance", opt.correspondance, "-catalogue", opt.catalogue,
		"-plafond-gio", strconv.Itoa(opt.plafondGiB), "-fils", strconv.Itoa(opt.fils))
	var sortie, erreurs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &sortie, &erreurs
	runErr := cmd.Run()
	rapport, errLecture := lireRapport(sortie.Bytes())
	if runErr != nil {
		if rapport.Depasse {
			return rapport, fmt.Errorf("film %s : plafond de %d Gio franchi (pic %.0f Mio)", nom, opt.plafondGiB,
				float64(rapport.PicOctets)/octetsParMio)
		}
		return rapport, fmt.Errorf("film %s : le fils a echoue : %w\n%s", nom, runErr, dernieresLignes(erreurs.String(), 5))
	}
	if errLecture != nil {
		return rapport, fmt.Errorf("film %s : rapport du fils illisible : %w", nom, errLecture)
	}
	return rapport, nil
}

// lireRapport decode la derniere ligne JSON de la sortie du fils.
func lireRapport(sortie []byte) (RapportEnfant, error) {
	lignes := strings.Split(strings.TrimSpace(string(sortie)), "\n")
	derniere := strings.TrimSpace(lignes[len(lignes)-1])
	if derniere == "" || len(derniere) > plafondMaxLigne {
		return RapportEnfant{}, errors.New("aucune ligne de rapport")
	}
	var r RapportEnfant
	if err := json.Unmarshal([]byte(derniere), &r); err != nil {
		return RapportEnfant{}, err
	}
	return r, nil
}

// dernieresLignes rend les `n` dernieres lignes non vides de `s`.
func dernieresLignes(s string, n int) string {
	var l []string
	for _, x := range strings.Split(s, "\n") {
		if strings.TrimSpace(x) != "" {
			l = append(l, x)
		}
	}
	if len(l) > n {
		l = l[len(l)-n:]
	}
	return strings.Join(l, "\n")
}
