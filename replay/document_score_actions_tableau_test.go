package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// tableauDesActions rend un tableau du film dont chaque joueur n'a que son score final.
func tableauDesActions(scores map[string]int) *ScoreTimeline {
	tl := &ScoreTimeline{}
	for xuid, v := range scores {
		tl.Players = append(tl.Players, PlayerScore{XUID: xuid, Score: ScoreSeries{Total: []ScoreTick{{T: 0, V: 0}, {T: 50, V: v}}}})
	}
	return tl
}

// TestLeManqueDuTableauFaitLeVerdict : un joueur dont le score du tableau dépasse la somme de ses actions à points
// POSITIFS publiées est un manque et rend `complete` faux ; un tableau en dessous est un retard et laisse le verdict ;
// une pénalité n'entre pas dans la somme ; sans joueur au tableau rien ne se compare.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas faire dépendre `complete` du manque ; faire dépendre `complete` du retard ;
// compter les pénalités dans la somme ; ne compter que les joueurs qui ont une action ; ne pas lire le DERNIER total ;
// compter un joueur sans total au tableau comme égal ; comparer un film sans joueur au tableau.
func TestLeManqueDuTableauFaitLeVerdict(t *testing.T) {
	deuxFrags := []grammar.EvenementDeJeuLu{
		messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 1, masqueDesActions(0)),
		messageDesActions(instantDeLaFrame(4), rangFrag, texteFrag, true, 1, masqueDesActions(0)),
	}
	fragEtSuicide := []grammar.EvenementDeJeuLu{
		messageDesActions(instantDeLaFrame(3), rangFrag, texteFrag, true, 1, masqueDesActions(0)),
		messageDesActions(instantDeLaFrame(4), rangSuicide, 0, false, -1, masqueDesActions(0)),
	}
	cas := []struct {
		nom              string
		events           []grammar.EvenementDeJeuLu
		tableau          *ScoreTimeline
		compares, manque int
		retard           int
		complet          bool
	}{
		{"tableau égal à la somme", deuxFrags, tableauDesActions(map[string]int{xuidA: 200}), 1, 0, 0, true},
		{"tableau au-dessus : manque", deuxFrags, tableauDesActions(map[string]int{xuidA: 300}), 1, 1, 0, false},
		{"tableau en dessous : retard, verdict inchangé", deuxFrags, tableauDesActions(map[string]int{xuidA: 100}), 1, 0, 1, true},
		{"pénalité seule : pas de manque", fragEtSuicide, tableauDesActions(map[string]int{xuidA: 100}), 1, 0, 0, true},
		{"pénalité seule, tableau nul", fragEtSuicide[1:], tableauDesActions(map[string]int{xuidA: 0}), 1, 0, 0, true},
		{"joueur du tableau sans action et score nul", deuxFrags, tableauDesActions(map[string]int{xuidA: 200, xuidB: 0}), 2, 0, 0, true},
		{"joueur du tableau sans action et score non nul : manque", deuxFrags, tableauDesActions(map[string]int{xuidA: 200, xuidB: 25}), 2, 1, 0, false},
		{"joueur avec actions absent du tableau : retard", deuxFrags, tableauDesActions(map[string]int{xuidB: 0}), 2, 0, 1, true},
		{"film sans joueur au tableau", deuxFrags, &ScoreTimeline{}, 0, 0, 0, true},
		{"film sans tableau", deuxFrags, nil, 0, 0, 0, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, cov := buildScoreActions(entreeDesActions([]string{scriptsDeLArene}, c.events...), rosterDesActions(),
				axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}, c.tableau, "")
			if cov.ScoreboardCompared != c.compares || cov.ScoreboardMissing != c.manque || cov.ScoreboardBehind != c.retard || cov.Complete != c.complet {
				t.Errorf("comparés %d, manques %d, retards %d, complet %v ; attendu %d, %d, %d, %v",
					cov.ScoreboardCompared, cov.ScoreboardMissing, cov.ScoreboardBehind, cov.Complete, c.compares, c.manque, c.retard, c.complet)
			}
		})
	}
}

// TestLeScoreFinalDuTableauEstLeDernierTotal : le score du tableau est la dernière valeur de la série cumulée, zéro
// sans série ; un film sans joueur n'a pas de tableau.
func TestLeScoreFinalDuTableauEstLeDernierTotal(t *testing.T) {
	tl := &ScoreTimeline{Players: []PlayerScore{
		{XUID: xuidA, Score: ScoreSeries{Total: []ScoreTick{{T: 1, V: 100}, {T: 9, V: 350}}}},
		{XUID: xuidB},
	}}
	got := scoreFinalDuTableau(tl)
	if len(got) != 2 || got[xuidA] != 350 || got[xuidB] != 0 {
		t.Errorf("scores %v, attendu 350 et 0", got)
	}
	if scoreFinalDuTableau(&ScoreTimeline{}) != nil || scoreFinalDuTableau(nil) != nil {
		t.Error("un film sans joueur au tableau ne doit pas avoir de tableau à comparer")
	}
}
