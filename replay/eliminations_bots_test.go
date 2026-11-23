package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// eliminations_bots_test.go — LES ÉLIMINATIONS DONT UN BOT EST L'AUTEUR OU LA VICTIME : tout est synthétique. Un bot n'a pas
// d'identifiant ; il se désigne par sa place (l'index de joueur du film, `roster[].filmIndex`).

// actionDeBot : une action de score posée à la place d'un bot.
func actionDeBot(name string, t, place int) ScoreAction {
	return ScoreAction{T: t, Place: placeDeTest(place), Name: name, Points: 100}
}

// TestActionsDeScoreDUnBot : un frag de bot est posé à sa place, sans xuid ; sa cible humaine garde son xuid, sa cible bot
// prend sa place ; une action de bot n'entre pas dans la comparaison au tableau.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : jeter l'action d'un bot (`noRecipient`) ; poser le xuid d'un autre ; ne pas
// poser la cible bot ; compter l'action d'un bot dans le tableau d'un joueur sans xuid.
func TestActionsDeScoreDUnBot(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	events := []grammar.EvenementDeJeuLu{
		messageDesActions(instantDeLaFrame(5), rangFrag, texteFrag, true, 0, masqueDesActions(3)), // bot -> humain
		messageDesActions(instantDeLaFrame(6), rangFrag, texteFrag, true, 3, masqueDesActions(1)), // humain -> bot
		messageDesActions(instantDeLaFrame(7), rangFrag, texteFrag, true, 3, masqueDesActions(3)), // bot -> bot (même place)
	}
	actions, cov := buildScoreActions(entreeDesActions([]string{scriptsDeLArene}, events...), rosterDesActions(), axe, nil, "")
	want := []ScoreAction{
		{T: 5, Place: placeDeTest(3), Name: "killPersonalScore", Points: 100, Target: xuidA},
		{T: 6, XUID: xuidB, Name: "killPersonalScore", Points: 100, TargetPlace: placeDeTest(3)},
		{T: 7, Place: placeDeTest(3), Name: "killPersonalScore", Points: 100, TargetPlace: placeDeTest(3)},
	}
	if !reflect.DeepEqual(actions, want) {
		t.Errorf("actions %+v, attendu %+v", actions, want)
	}
	if cov.Scored != 3 || cov.Published != 3 || cov.NoRecipient != 0 || cov.BotRecipient != 2 || cov.BotTarget != 2 || !cov.Complete {
		t.Errorf("couverture %+v : 3 publiées, 0 sans destinataire, 2 destinataires bots, 2 cibles bots, complet attendus", cov)
	}
}

// Un bot sans humain ni entrée de bot au roster (un siège que le film ne nomme pas) reste sans destinataire.
func TestActionDUnSiegeNonNommeResteSansDestinataire(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	events := []grammar.EvenementDeJeuLu{messageDesActions(instantDeLaFrame(5), rangFrag, texteFrag, true, 21, masqueDesActions(20))}
	actions, cov := buildScoreActions(entreeDesActions([]string{scriptsDeLArene}, events...), rosterDesActions(), axe, nil, "")
	if len(actions) != 1 || actions[0].Place != nil || actions[0].XUID != "" && false {
		t.Logf("actions %+v", actions)
	}
	if len(actions) != 0 || cov.NoRecipient != 1 || cov.NoTarget != 1 || cov.Complete {
		t.Errorf("actions %+v couverture %+v : aucune action, 1 sans destinataire, 1 sans cible, incomplet attendus", actions, cov)
	}
}

// Auteur bot, victime bot, bot contre bot, humain contre humain : chaque cas a ses lignes, et l'humain ne change pas.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne rien publier pour un bot ; publier un xuid vide en plus de la place ; perdre la
// place de la victime ; ajouter une ligne d'humain pour une action de bot ; ne pas compter les lignes de bots.
func TestEliminationsDeBots(t *testing.T) {
	humain := joueurDeTest("k", [][2]int{{10, 1}, {30, 2}}, nil)
	in := eliminationInput{
		Players: []PlayerScore{humain},
		Actions: []ScoreAction{
			killAction(10, "k", "vh"), // humain -> humain
			{T: 30, XUID: "k", Name: killScoreActionName, Points: 100, TargetPlace: placeDeTest(3)},             // humain -> bot
			{T: 15, Place: placeDeTest(3), Name: killScoreActionName, Points: 100, Target: "k"},                 // bot -> humain
			{T: 20, Place: placeDeTest(3), Name: killScoreActionName, Points: 100, TargetPlace: placeDeTest(5)}, // bot -> bot
			{T: 22, Place: placeDeTest(3), Name: "assistPersonalScore", Points: 50},                             // pas une élimination
			{T: 25, Place: placeDeTest(5), Name: killScoreActionName, Points: 100},                              // bot, victime inconnue
		},
	}
	got, cov := buildEliminations(in)
	want := []Elimination{
		{T: 10, XUID: "k", Victim: "vh", Headshot: nil},
		{T: 15, Place: placeDeTest(3), Victim: "k"},
		{T: 20, Place: placeDeTest(3), VictimPlace: placeDeTest(5)},
		{T: 25, Place: placeDeTest(5)},
		{T: 30, XUID: "k", VictimPlace: placeDeTest(3)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("éliminations %+v, attendu %+v", got, want)
	}
	if cov.Eliminations != 5 || cov.BotEliminations != 3 || cov.WithVictim != 4 || cov.VictimPlaces != 2 {
		t.Errorf("couverture %+v : 5 éliminations dont 3 de bots, 4 victimes dont 2 bots attendues", cov)
	}
	if cov.HeadYes+cov.HeadNo+cov.HeadUnknown != cov.Eliminations || cov.PerfectYes+cov.PerfectUnknown != cov.Eliminations {
		t.Errorf("couverture %+v : les invariants head*/perfect* ne tiennent plus", cov)
	}
}

// Sans action de bot, le calque est celui d'avant : aucune ligne de bot, aucun compteur de bot.
func TestEliminationsSansBotInchangees(t *testing.T) {
	in := eliminationInput{
		Players: []PlayerScore{joueurDeTest("k", [][2]int{{10, 1}}, nil)},
		Actions: []ScoreAction{killAction(10, "k", "vh"), {T: 12, Place: placeDeTest(3), Name: "assistPersonalScore", Points: 50}},
	}
	got, cov := buildEliminations(in)
	if len(got) != 1 || got[0].Place != nil || got[0].VictimPlace != nil || cov.BotEliminations != 0 || cov.VictimPlaces != 0 {
		t.Errorf("éliminations %+v couverture %+v : une ligne d'humain, rien de bot attendus", got, cov)
	}
}
