package objectives

import (
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// round_reset.go — LA MANCHE QUE LE FILM ANNONCE PAR UNE REMISE A ZERO SIMULTANEE DES SCORES.
//
// Quand une manche commence, le film ecrit pour chaque joueur present un enregistrement qui porte
// le score de mode ET le score personnel remis a zero, tous au meme instant. C'est le fait que le
// film emet LUI-MEME pour dire « une manche commence » : il ne depend ni du nombre d'emissions de
// la manche ni de sa duree. Mesure : le film `44e14331` ecrit ainsi sa seconde manche (803 a 859 s)
// avec les huit joueurs remis a zero a la meme milliseconde, alors que cette manche ne tire qu'un
// point de score de mode (sous [statMinRoundRun]) et 8,67 % des enregistrements de joueur de la
// manche la plus fournie (sous [statMinRoundRecordShare]).
//
// Ce critere S'AJOUTE aux deux autres (suite coherente, part d'enregistrements) : il ne retire
// jamais une manche que ceux-ci admettaient.

// statResetWindowMS : la fenetre, en millisecondes, dans laquelle les remises a zero de tous les
// joueurs presents doivent tomber pour compter comme UNE remise simultanee, et dans laquelle elles
// doivent suivre le premier enregistrement de joueur de la manche pour compter comme son debut.
//
// Une seconde est la precision du fait : le film date l'ecriture d'une manche sur un meme paquet
// (`44e14331` : les huit remises tombent a la meme milliseconde), et la seconde absorbe l'ecart
// d'un paquet voisin sans rapprocher deux remises qui seraient reellement etalees dans la manche.
const statResetWindowMS = 1000

// statMinResetPlayers : le nombre minimal de joueurs presents pour qu'une remise soit un fait de
// manche. Un seul enregistrement nul ne prouve pas qu'une manche commence ; deux joueurs remis a
// zero a la meme seconde sont deja improbables pour un ancrage fortuit, qui arrive isole.
const statMinResetPlayers = 2

// simultaneousResetRounds rend les manches, a partir de la deuxieme, dont tous les joueurs presents
// ont leur score remis a zero dans la meme seconde, au debut de la manche.
//
// Un JOUEUR PRESENT est un slot de joueur (les slots d'equipe sont exclus) qui porte au moins un
// enregistrement de la manche. Sa REMISE A ZERO est son premier enregistrement de la manche qui
// porte le score de mode (composant 0) ET le score personnel (composant 1), les deux canaux de
// chacun a zero. La manche est admise quand :
//
//   - tous les joueurs presents ont une remise a zero, et ils sont au moins [statMinResetPlayers] ;
//   - ces remises tiennent dans [statResetWindowMS] ;
//   - la premiere remise ne suit pas de plus de [statResetWindowMS] le premier enregistrement de
//     joueur de la manche (elle marque le DEBUT de la manche, pas un score nul en cours de route) ;
//   - ces joueurs sont au moins la moitie des slots de joueur du film : un fragment isole de
//     quelques slots ne tient pas pour la manche entiere.
//
// La manche 0 n'est jamais lue ici : le film ne l'ouvre pas par cette remise, et sa presence se
// decide ailleurs (repli de [contiguousRounds]).
func simultaneousResetRounds(recs []types.StatRecord) map[int]bool {
	rounds, filmSlots := resetsParManche(recs)
	out := map[int]bool{}
	for round, s := range rounds {
		if round >= 1 && len(s.slots) >= statMinResetPlayers && len(s.resets) == len(s.slots) &&
			len(s.slots)*2 >= filmSlots && s.remisesSimultanees() {
			out[round] = true
		}
	}
	return out
}

// resetsManche porte, pour une manche, ce que la lecture des remises a zero a besoin de savoir.
type resetsManche struct {
	slots   map[int]bool // joueurs presents
	resets  map[int]int  // slot -> instant de sa premiere remise a zero
	premier int          // premier instant d'un enregistrement de joueur
}

// resetsParManche range, par manche, les joueurs presents et leurs remises a zero, et rend le
// nombre de slots de joueur du film.
func resetsParManche(recs []types.StatRecord) (map[int]*resetsManche, int) {
	rounds := map[int]*resetsManche{}
	filmSlots := map[int]bool{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			continue
		}
		filmSlots[r.Slot] = true
		s := rounds[r.Round]
		if s == nil {
			s = &resetsManche{slots: map[int]bool{}, resets: map[int]int{}, premier: r.TimeMS}
			rounds[r.Round] = s
		}
		s.noter(r)
	}
	return rounds, len(filmSlots)
}

// noter range un enregistrement de joueur de la manche.
func (s *resetsManche) noter(r types.StatRecord) {
	s.slots[r.Slot] = true
	s.premier = min(s.premier, r.TimeMS)
	if !isScoreReset(r) {
		return
	}
	if t, vu := s.resets[r.Slot]; !vu || r.TimeMS < t {
		s.resets[r.Slot] = r.TimeMS
	}
}

// remisesSimultanees dit que les remises tiennent dans [statResetWindowMS] et que la premiere
// suit de moins de [statResetWindowMS] le premier enregistrement de joueur de la manche.
func (s *resetsManche) remisesSimultanees() bool {
	lo, hi := -1, -1
	for _, t := range s.resets {
		if lo < 0 || t < lo {
			lo = t
		}
		hi = max(hi, t)
	}
	return hi-lo <= statResetWindowMS && lo-s.premier <= statResetWindowMS
}

// isScoreReset dit qu'un enregistrement porte le score de mode et le score personnel, les deux
// canaux de chacun a zero : l'ecriture de la remise d'un joueur au debut d'une manche.
func isScoreReset(r types.StatRecord) bool {
	for _, comp := range [...]int{modeScoreComp, personalScoreComp} {
		v, ok := r.Comps[comp]
		if !ok || v.A != 0 || v.B != 0 {
			return false
		}
	}
	return true
}
