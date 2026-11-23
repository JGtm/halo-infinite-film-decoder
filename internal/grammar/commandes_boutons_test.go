package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// commandes_boutons_test.go — LES COMMANDES D ACCROUPI ET DE SAUT, tenues dans les drapeaux de l entree de
// controle et pliees comme la commande de deplacement. Scenarios synthetiques, sans film.

// entreeDeBoutons rend l entree d une place qui porte le bloc de commande, avec ses drapeaux (-1 : champ absent du
// flux) et un couple analogique au zero exact.
func entreeDeBoutons(index, drapeaux int) EntreeDeControle {
	e := entreeDeCommande(index, zeroAnalogique)
	e.Champs.Drapeaux = drapeaux
	return e
}

// commandesDuGenre rend, dans l ordre, les commandes tenues du genre `kind`.
func commandesDuGenre(hs []types.ControlHold, kind string) []types.ControlHold {
	var out []types.ControlHold
	for _, h := range hs {
		if h.Kind == kind {
			out = append(out, h)
		}
	}
	return out
}

// TestLesBoutonsDAccroupiEtDeSautSontDeuxCommandesIndependantes : le bit de valeur 1 tient l accroupi, celui de
// valeur 4 tient le saut, chacun sur sa place ; les autres bits du champ ne tiennent rien. MUTATIONS : les deux
// valeurs permutees, un autre bit lu comme un bouton, un genre pose sur l autre -> ROUGE.
func TestLesBoutonsDAccroupiEtDeSautSontDeuxCommandesIndependantes(t *testing.T) {
	hs, st, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeBoutons(3, 0), entreeDeBoutons(5, 0)}},
		{ts: 20, entrees: []EntreeDeControle{entreeDeBoutons(3, 1), entreeDeBoutons(5, 4)}},
		{ts: 30, entrees: []EntreeDeControle{entreeDeBoutons(3, 1|4), entreeDeBoutons(5, 2|8|16)}},
		{ts: 40, entrees: []EntreeDeControle{entreeDeBoutons(3, 4), entreeDeBoutons(5, 0)}},
		{ts: 50, entrees: []EntreeDeControle{entreeDeBoutons(3, 0), entreeDeBoutons(5, 0)}},
	})
	accroupi, saut := commandesDuGenre(hs, types.MovementCrouchCommand), commandesDuGenre(hs, types.MovementJumpCommand)
	if len(accroupi) != 1 || accroupi[0].FilmIndex != 3 || accroupi[0].StartUS != 20 || accroupi[0].EndUS != 40 ||
		accroupi[0].Entries != 2 {
		t.Errorf("accroupi %+v, attendu la place 3 sur [20, 40), 2 entrees", accroupi)
	}
	if len(saut) != 2 || saut[0].FilmIndex != 5 || saut[0].StartUS != 20 || saut[0].EndUS != 30 ||
		saut[1].FilmIndex != 3 || saut[1].StartUS != 30 || saut[1].EndUS != 50 {
		t.Errorf("saut %+v, attendu la place 5 sur [20, 30) puis la place 3 sur [30, 50)", saut)
	}
	if got := len(commandesDuGenre(hs, types.MovementMoveCommand)); got != 0 {
		t.Errorf("le deplacement est au zero exact : %d commande(s) lue(s)", got)
	}
	if st.Crouch.Holds != 1 || st.Jump.Holds != 2 || st.Move.Holds != 0 || st.Entries != 10 {
		t.Errorf("compteurs %+v", st)
	}
}

// TestUnChampDeDrapeauxAbsentNeTientAucunBouton : « absent » vaut -1, dont tous les bits sont poses ; c est
// l absence qui se teste AVANT le masque, et elle dit « aucun bouton ». Un bouton tenu qui devient absent est
// relache. MUTATION : masquer sans tester l absence -> ROUGE (un saut et un accroupi sur chaque entree).
func TestUnChampDeDrapeauxAbsentNeTientAucunBouton(t *testing.T) {
	hs, _, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeBoutons(2, ChampDeControleAbsent)}},
		{ts: 20, entrees: []EntreeDeControle{entreeDeBoutons(2, 5)}},
		{ts: 30, entrees: []EntreeDeControle{entreeDeBoutons(2, ChampDeControleAbsent)}},
	})
	if len(hs) != 2 {
		t.Fatalf("commandes %+v, attendu un accroupi et un saut", hs)
	}
	for _, h := range hs {
		if h.StartUS != 20 || h.EndUS != 30 || h.EndBound != types.ContinuousFireBoundReleased {
			t.Errorf("commande %+v, attendue sur [20, 30) relachee par l entree sans champ", h)
		}
	}
	if (ChampsDeControle{Drapeaux: ChampDeControleAbsent}).drapeauPose(drapeauSaut) {
		t.Error("le champ absent ne pose aucun bit")
	}
}

// TestUneEntreeSansBlocNeDitRienDesBoutons : un bouton tenu traverse une entree sans bloc, et une entree sans
// bloc ne lui donne pas de fin ; seul le bloc suivant la lui donne.
func TestUneEntreeSansBlocNeDitRienDesBoutons(t *testing.T) {
	hs, _, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeBoutons(1, 1)}},
		{ts: 20, entrees: []EntreeDeControle{{Index: 1}}},
		{ts: 30, entrees: []EntreeDeControle{entreeDeBoutons(1, 0)}},
	})
	if len(hs) != 1 || hs[0].Kind != types.MovementCrouchCommand || hs[0].StartUS != 10 || hs[0].EndUS != 30 ||
		hs[0].Entries != 1 {
		t.Fatalf("commandes %+v, attendu un accroupi sur [10, 30) qui compte une seule entree", hs)
	}
}

// TestLesTrousDesBoutonsSontPortesCommeCeuxDuDeplacement : tenu avant et apres un trou, le bouton porte un
// passage interieur ; relache dans le trou, il finit au trou ; pose apres un trou, il commence a l entree
// qui le lit avec la borne « trou ». MUTATION : lever l inconnu avant de plier les boutons -> ROUGE.
func TestLesTrousDesBoutonsSontPortesCommeCeuxDuDeplacement(t *testing.T) {
	hs, st, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeBoutons(1, 1), entreeDeBoutons(2, 1), entreeDeBoutons(3, 0)}},
		{ts: 20, trou: true},
		{ts: 30, entrees: []EntreeDeControle{entreeDeBoutons(1, 1), entreeDeBoutons(2, 0), entreeDeBoutons(3, 4)}},
		{ts: 40, entrees: []EntreeDeControle{entreeDeBoutons(1, 0), entreeDeBoutons(3, 0)}},
	})
	if len(hs) != 3 {
		t.Fatalf("commandes %+v, attendu trois", hs)
	}
	interieur := commandesDuGenre(hs, types.MovementCrouchCommand)
	if len(interieur) != 2 || interieur[0].FilmIndex != 1 || len(interieur[0].Holes) != 1 ||
		interieur[0].Holes[0].StartUS != 20 || interieur[0].Holes[0].EndUS != 30 {
		t.Errorf("accroupi %+v, attendu la place 1 avec le passage [20, 30)", interieur)
	}
	if interieur[1].FilmIndex != 2 || interieur[1].EndUS != 20 || interieur[1].EndBound != types.ContinuousFireBoundHole {
		t.Errorf("accroupi %+v, attendu la place 2 finie a 20 dans le trou", interieur[1])
	}
	saut := commandesDuGenre(hs, types.MovementJumpCommand)
	if len(saut) != 1 || saut[0].StartUS != 30 || saut[0].StartBound != types.ContinuousFireBoundHole {
		t.Errorf("saut %+v, attendu un saut ouvert a 30 avec la borne trou", saut)
	}
	if st.Crouch.Holds != 2 || st.Crouch.WithHole != 2 || st.Crouch.InnerHoles != 1 ||
		st.Jump.Holds != 1 || st.Jump.WithHole != 1 {
		t.Errorf("compteurs %+v", st)
	}
}

// TestLesBoutonsTenusALaFinDuFilmFinissentAuDernierPaquet : la fin de la marche cloture les boutons encore
// tenus, avec la borne de fin de film.
func TestLesBoutonsTenusALaFinDuFilmFinissentAuDernierPaquet(t *testing.T) {
	hs, _, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeBoutons(4, 4)}},
		{ts: 90, entrees: []EntreeDeControle{entreeDeBoutons(4, 4)}},
	})
	if len(hs) != 1 || hs[0].Kind != types.MovementJumpCommand || hs[0].EndUS != 90 ||
		hs[0].EndBound != types.ContinuousFireBoundFilmEnd || hs[0].Entries != 2 {
		t.Fatalf("commandes %+v, attendu un saut fini a 90 avec la borne de fin de film", hs)
	}
}
