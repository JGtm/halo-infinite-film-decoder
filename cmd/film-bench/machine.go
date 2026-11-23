package main

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// valeurInconnue : ce que le releve ecrit quand le poste ne dit pas une valeur.
const valeurInconnue = "inconnu"

// Machine decrit le poste de mesure. Le nom du poste n'y figure pas.
type Machine struct {
	Processeur string
	Coeurs     int // coeurs logiques
	MemoireMio uint64
	Systeme    string
	Go         string
	GoMaxProcs int
}

// releverMachine lit ce que le poste dit de lui ; une valeur indisponible vaut « inconnu » ou 0.
func releverMachine() Machine {
	m := Machine{
		Processeur: processeur(),
		Coeurs:     runtime.NumCPU(),
		MemoireMio: memoireMio(),
		Systeme:    runtime.GOOS + "/" + runtime.GOARCH,
		Go:         runtime.Version(),
		GoMaxProcs: runtime.GOMAXPROCS(0),
	}
	if m.Processeur == "" {
		m.Processeur = valeurInconnue
	}
	return m
}

// arbreGit rend `git describe --tags --always --dirty` du dossier courant, ou « inconnu ».
func arbreGit(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		return valeurInconnue
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		return s
	}
	return valeurInconnue
}

// dire ecrit un message de progression ou d'erreur sur un flux d'information (stderr) ; un echec
// d'ecriture sur ce flux n'a pas de suite a lui donner.
func dire(w io.Writer, format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
