package main

import (
	"slices"
	"time"
)

// Phase nomme une mesure du releve. Les cinq phases sont l'ordre des lignes du TSV.
type Phase string

const (
	// PhaseDecompression : somme de `decfilm.Inflate` sur les octets de chaque chunk, lus avant le
	// chronometre. Mesuree a part ; elle est CONTENUE dans le chargement, pas ajoutee au total.
	PhaseDecompression Phase = "decompression"
	// PhaseChargement : `decfilm.LoadDir`, lecture des fichiers, decompression et decoupage en paquets.
	PhaseChargement Phase = "chargement"
	// PhaseDecodage : du debut de `replay.BuildFromFilmAvecFaits` a l'etape observee `clockOrigin`.
	PhaseDecodage Phase = "decodage"
	// PhasePublication : de l'etape `clockOrigin` au retour de `replay.BuildFromFilmAvecFaits`.
	PhasePublication Phase = "publication"
	// PhaseTotal : chargement, decodage et publication d'un seul tenant (une horloge, pas une somme).
	PhaseTotal Phase = "total"
)

// ordreDesPhases est l'ordre d'ecriture du releve.
var ordreDesPhases = []Phase{PhaseDecompression, PhaseChargement, PhaseDecodage, PhasePublication, PhaseTotal}

// Passe regroupe les durees d'UNE passe de mesure d'un film.
type Passe struct {
	Decompression, Chargement, Decodage, Publication, Total time.Duration
}

// Duree rend la duree de la phase `ph` dans cette passe.
func (p Passe) Duree(ph Phase) time.Duration {
	switch ph {
	case PhaseDecompression:
		return p.Decompression
	case PhaseChargement:
		return p.Chargement
	case PhaseDecodage:
		return p.Decodage
	case PhasePublication:
		return p.Publication
	case PhaseTotal:
		return p.Total
	}
	return 0
}

// Stat resume une phase sur les passes chronometrees.
type Stat struct {
	Mediane, Min, Max time.Duration
	Passes            int
}

// resumer rend la mediane, le minimum et le maximum de `valeurs` ; vide rend la valeur zero.
// La mediane d'un nombre pair de valeurs est la moyenne des deux valeurs centrales.
func resumer(valeurs []time.Duration) Stat {
	if len(valeurs) == 0 {
		return Stat{}
	}
	tri := slices.Clone(valeurs)
	slices.Sort(tri)
	n := len(tri)
	med := tri[n/2]
	if n%2 == 0 {
		med = (tri[n/2-1] + tri[n/2]) / 2
	}
	return Stat{Mediane: med, Min: tri[0], Max: tri[n-1], Passes: n}
}

// resumerPasses calcule la Stat de chaque phase sur les passes d'un film.
func resumerPasses(passes []Passe) map[Phase]Stat {
	out := make(map[Phase]Stat, len(ordreDesPhases))
	for _, ph := range ordreDesPhases {
		valeurs := make([]time.Duration, len(passes))
		for i, p := range passes {
			valeurs[i] = p.Duree(ph)
		}
		out[ph] = resumer(valeurs)
	}
	return out
}

// tourner rend `noms` decale de `passe` positions (modulo leur nombre) : la passe 0 garde l'ordre
// donne, la passe 1 commence par le deuxieme film, etc. Chaque film occupe ainsi toutes les
// positions de la boucle, et l'effet d'un voisin (cache, tas) ne s'attache pas toujours au meme
// film. L'entree n'est pas modifiee.
func tourner(noms []string, passe int) []string {
	n := len(noms)
	if n == 0 {
		return nil
	}
	d := passe % n
	if d < 0 {
		d += n
	}
	out := make([]string, 0, n)
	out = append(out, noms[d:]...)
	return append(out, noms[:d]...)
}
