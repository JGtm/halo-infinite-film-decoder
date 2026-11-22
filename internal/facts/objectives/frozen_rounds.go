package objectives

import (
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// frozen_rounds.go — LES VALEURS FIGEES DES MANCHES TERMINEES, PAR JOUEUR ET PAR MANCHE.
//
// A chaque manche terminee, le film fige la ligne de chaque joueur : la valeur de chaque emplacement du
// tableau a la fin de la manche ([types.StatRecord.Frozen]). Elle est livree au debut de la manche
// suivante, sur la ligne du joueur, puis repetee ; elle ne change pas d'une emission a l'autre. La
// derniere manche d'un match n'est jamais figee : un match a une manche n'en a aucune.
//
// Ces valeurs portent ce que la lecture en direct n'a pas en fin de manche : tirs tires et tirs au but,
// precision, degats infliges et subis, duree de vie moyenne, KDA, manches gagnees.

// Noms publies des valeurs figees ([FrozenStats]) qui ne sont pas deja ceux de [ScoreboardStats].
const (
	FrozenModeScore     = "modeScore"
	FrozenRoundsWon     = "roundsWon"
	FrozenPersonalScore = "personalScore"
	FrozenKills         = "kills"
	FrozenDeaths        = "deaths"
	FrozenAssists       = "assists"
	FrozenKDA           = "kda"
	FrozenAverageLife   = "averageLifeDuration"
	FrozenShotsFired    = "shotsFired"
	FrozenShotsHit      = "shotsHit"
	FrozenAccuracy      = "accuracy"
	FrozenDamageDealt   = "damageDealt"
	FrozenDamageTaken   = "damageTaken"
)

// FrozenStat est un emplacement publie des valeurs figees : son nom et son adresse dans le tableau.
type FrozenStat struct {
	// Name est le nom publie.
	Name string
	// Component est l'emplacement du compteur de la manche en cours dont c'est la valeur figee.
	Component StatComponent
}

// FrozenStats rend les emplacements publies des valeurs figees, dans l'ordre de leur composant. Ce sont
// ceux dont le sens est etabli, quel que soit le mode ; les emplacements propres a un mode (index 20 a 27),
// les doublons (`Spartan*`, experience de kit) et les emplacements jamais ecrits n'y sont pas.
//
// L'unite est celle du film, entiere : `kda` et `accuracy` en centiemes (precision 2 de la definition de
// l'emplacement ; `accuracy` = touches / tirs, tronque, en centiemes de pour cent ; `kda` = eliminations -
// morts + assistances / 3, tronque, en centiemes), `averageLifeDuration` en dixiemes de seconde
// (precision 1), les autres en unites.
func FrozenStats() []FrozenStat {
	at := func(name string, comp int, sideB bool) FrozenStat {
		return FrozenStat{Name: name, Component: StatComponent{Comp: comp, SideB: sideB}}
	}
	return []FrozenStat{
		at(FrozenModeScore, 0, false), at(FrozenRoundsWon, 0, true),
		at(FrozenPersonalScore, 1, true),
		at(FrozenKills, 2, false), at(FrozenDeaths, 2, true),
		at(FrozenAssists, 3, false), at(FrozenKDA, 3, true),
		at(StatSuicides, 4, false), at(StatBetrayals, 4, true),
		at(FrozenAverageLife, 5, false), at(StatMaxSpree, 5, true),
		at(StatHeadshots, 6, false), at(FrozenShotsFired, 6, true),
		at(FrozenShotsHit, 7, false), at(FrozenAccuracy, 7, true),
		at(StatMeleeKills, 8, false), at(StatGrenadeKills, 8, true),
		at(StatPowerWeaponKills, 9, false), at(FrozenDamageDealt, 9, true),
		at(FrozenDamageTaken, 10, false), at(StatSpawns, 10, true),
	}
}

// FrozenRound porte les valeurs figees d'une manche terminee d'un joueur, par nom ([FrozenStats]).
type FrozenRound struct {
	// Round est la manche terminee, 0-based.
	Round int
	// Values porte la valeur de chaque emplacement publie ; une valeur que le film declare par defaut vaut 0.
	Values map[string]int64
}

// cleFigee designe une valeur figee : le joueur, la manche terminee, le composant et le cote.
type cleFigee struct {
	owner       string
	round, comp int
	sideB       bool
}

// FrozenRoundsByOwner rend les manches figees de chaque joueur, triees par manche, et le nombre de couples
// (joueur, manche) ecartes. `owner` nomme le joueur de la ligne d'un enregistrement ("" : personne) ; les
// lignes d'equipe ne sont pas lues.
//
// Pour une valeur, la derniere emission non nulle l'emporte (une valeur figee ne change pas ; ses zeros
// sont ceux de la remise a zero d'une ligne). Un couple (joueur, manche) dont toutes les valeurs sont
// nulles n'est pas une manche figee : c'est la ligne remise a zero au debut d'une manche, ou celle d'un
// joueur qui n'a pas joue cette manche. Un couple dont la manche n'est pas une manche retenue suivie d'une
// autre manche retenue ([RealRounds]) n'est pas publie et se compte : seule une manche terminee se fige.
func FrozenRoundsByOwner(recs []types.StatRecord, owner func(types.StatRecord) string) (map[string][]FrozenRound, int) {
	l := lectureFigee{valeurs: map[cleFigee]instantValeur{}, ecrites: map[string]map[int]bool{}}
	for _, r := range recs {
		if len(r.Frozen) == 0 || IsTeamSlot(r.Slot) {
			continue
		}
		if o := owner(r); o != "" {
			l.noter(o, r)
		}
	}
	return l.manches(RealRounds(recs))
}

// instantValeur est une valeur figee non nulle et l'instant de l'enregistrement qui la porte.
type instantValeur struct {
	t int
	v int64
}

// lectureFigee accumule les valeurs figees non nulles par (joueur, manche, emplacement), et les manches
// ou un joueur a au moins une valeur non nulle.
type lectureFigee struct {
	valeurs map[cleFigee]instantValeur
	ecrites map[string]map[int]bool
}

// noter range les valeurs figees non nulles de l'enregistrement `r`, de la ligne du joueur `o`.
func (l *lectureFigee) noter(o string, r types.StatRecord) {
	for comp, vals := range r.Frozen {
		for _, f := range vals {
			for side, v := range [2]int64{f.A, f.B} {
				if v == 0 {
					continue
				}
				k := cleFigee{owner: o, round: f.Round, comp: comp, sideB: side == 1}
				if prev, vu := l.valeurs[k]; !vu || r.TimeMS >= prev.t {
					l.valeurs[k] = instantValeur{t: r.TimeMS, v: v}
				}
				if l.ecrites[o] == nil {
					l.ecrites[o] = map[int]bool{}
				}
				l.ecrites[o][f.Round] = true
			}
		}
	}
}

// manches rend les manches figees publiables de chaque joueur et le nombre de couples ecartes (cf.
// [FrozenRoundsByOwner]).
func (l *lectureFigee) manches(real map[int]bool) (map[string][]FrozenRound, int) {
	out := make(map[string][]FrozenRound, len(l.ecrites))
	ecartes := 0
	for o, rounds := range l.ecrites {
		for round := range rounds {
			if !real[round] || !real[round+1] {
				ecartes++
				continue
			}
			fr := FrozenRound{Round: round, Values: make(map[string]int64, len(FrozenStats()))}
			for _, s := range FrozenStats() {
				k := cleFigee{owner: o, round: round, comp: s.Component.Comp, sideB: s.Component.SideB}
				fr.Values[s.Name] = l.valeurs[k].v
			}
			out[o] = append(out[o], fr)
		}
		sort.Slice(out[o], func(i, j int) bool { return out[o][i].Round < out[o][j].Round })
		if len(out[o]) == 0 {
			delete(out, o)
		}
	}
	return out, ecartes
}
