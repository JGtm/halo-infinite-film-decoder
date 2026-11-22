package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// scriptsGrandeBataille sont les scripts du groupe de la Grande bataille en Slayer.
var scriptsGrandeBataille = []string{"bigteambattle.lua", "btbslayerevents.lua"}

// messagesDeVehicules : une aide par IEM (rang 227 du film), un Ghost détruit (231), un message hors de l'arbre
// décalé (263), une médaille hors des intervalles décalés (178).
func messagesDeVehicules() []grammar.EvenementDeJeuLu {
	masque := masqueDesActions(0)
	return []grammar.EvenementDeJeuLu{
		messageDesActions(instantDeLaFrame(1), 227, 0, false, -1, masque),
		messageDesActions(instantDeLaFrame(2), 231, 0, false, -1, masque),
		messageDesActions(instantDeLaFrame(3), 263, 0, false, -1, masque),
		messageDesActions(instantDeLaFrame(4), 178, 0, false, -1, masque),
	}
}

// TestLeBuildDecaleLArbreDesActions : sur HI_1_11_0, les rangs de véhicule de la Grande bataille se lisent un cran
// plus loin et la couverture compte les messages traduits ; sur un autre build ou sans build, les mêmes messages se lisent
// tels quels et rien n'est compté.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas passer le build à la résolution ; un décalage de deux crans sur le
// premier intervalle ; un décalage appliqué à tous les builds ; ne pas compter `treeShift` ; le compter hors des
// intervalles.
func TestLeBuildDecaleLArbreDesActions(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	cas := []struct {
		nom        string
		build      string
		actions    []ScoreAction
		treeShift  int
		hors       int
		sansPoints int
		nonConfirm int
	}{
		{"HI_1_11_0 : décalé", "HI_1_11_0", []ScoreAction{
			{T: 1, XUID: xuidA, Name: "empAssistPersonalScoreResponse", Points: 50},
			{T: 2, XUID: xuidA, Name: "ghostDestroyResponse", Points: 50},
		}, 3, 1, 1, 0},
		{"HI_1_12_0 : tel quel", "HI_1_12_0", []ScoreAction{
			{T: 2, XUID: xuidA, Name: "falconDestroyResponse", Points: 75},
			{T: 3, XUID: xuidA, Name: "wraithHijackResponse", Points: 25},
		}, 0, 0, 2, 0},
		{"build inconnu du film : tel quel", "", []ScoreAction{
			{T: 2, XUID: xuidA, Name: "falconDestroyResponse", Points: 75},
			{T: 3, XUID: xuidA, Name: "wraithHijackResponse", Points: 25},
		}, 0, 0, 2, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			actions, cov := buildScoreActions(entreeDesActions(scriptsGrandeBataille, messagesDeVehicules()...), rosterDesActions(), axe, nil, c.build)
			if !slices.Equal(actions, c.actions) {
				t.Errorf("actions %+v, attendu %+v", actions, c.actions)
			}
			if cov.TreeShift != c.treeShift || cov.OutsideTree != c.hors || cov.Unrelated != c.sansPoints || cov.UnconfirmedRank != c.nonConfirm {
				t.Errorf("treeShift %d, hors de l'arbre %d, sans points %d, non confirmés %d ; attendu %d, %d, %d, %d",
					cov.TreeShift, cov.OutsideTree, cov.Unrelated, cov.UnconfirmedRank, c.treeShift, c.hors, c.sansPoints, c.nonConfirm)
			}
		})
	}
}

// TestLAssemblagePasseLeBuildDuFilm : le calque lit l'arbre avec le build que l'identité du film porte ; sans
// identité, le rang se lit tel quel.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : passer un build vide ou en dur à la résolution.
func TestLAssemblagePasseLeBuildDuFilm(t *testing.T) {
	cas := []struct {
		nom      string
		identite *profile.FilmIdentity
		nom0     string
		decale   int
	}{
		{"film de HI_1_11_0", &profile.FilmIdentity{Build: "HI_1_11_0"}, "empAssistPersonalScoreResponse", 3},
		{"film de HI_1_12_0", &profile.FilmIdentity{Build: "HI_1_12_0"}, "", 0},
		{"film sans identité", nil, "", 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := &assemblage{opt: Options{FilmIdentity: c.identite, ScoreActions: entreeDesActions(scriptsGrandeBataille, messagesDeVehicules()...)},
				doc: ReplayDocument{FrameCount: finDesActions, Roster: rosterDesActions(), Coverage: &Coverage{}}, origin: origineDesActions, step: pasDesActions}
			a.poserLesScoresParAction()
			if a.doc.Coverage.ScoreActions == nil || a.doc.Coverage.ScoreActions.TreeShift != c.decale {
				t.Fatalf("couverture %+v, treeShift %d attendu", a.doc.Coverage.ScoreActions, c.decale)
			}
			if c.nom0 != "" && (len(a.doc.ScoreActions) == 0 || a.doc.ScoreActions[0].Name != c.nom0) {
				t.Errorf("actions %+v, première attendue : %s", a.doc.ScoreActions, c.nom0)
			}
		})
	}
}
