package replay

// document_arme_tireur_publie_test.go — L ARME D UNE TOUCHE SE CHERCHE SOUS LE TIREUR QUE LA TOUCHE PUBLIE.
//
// Le message de degat nomme un index de joueur (ici 2, C) ; l unite responsable est celle de B, que la touche publie comme tireur.
// Les tirs sont rangés sous B (leur place lue). Chercher l arme sous l index du message (C) ne trouve aucun tir ; la chercher
// sous le tireur publie trouve celui de B.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// TestLArmeDUneToucheEtDUnDegatSeCherchentSousLeTireurPublie : par le vrai assemblage, la touche et la ligne de degat dont
// l auteur est B portent l arme du dernier tir de B, meme quand l index du message est celui de C.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : chercher l arme sous l index du message ; ne corriger que les touches ou que les degats.
func TestLArmeDUneToucheEtDUnDegatSeCherchentSousLeTireurPublie(t *testing.T) {
	const arme = 0x00000000AB00CD07
	fire := []grammar.FireEvent{tirDeTest(1, 1, instantDeLaFrame(3), arme)} // la place 1 est celle de B
	message := degatDeTest(instantDeLaFrame(5), effetArme, 2, 6)            // l index du message est celui de C ...
	message.Responsable, message.AResponsable = 5, true                     // ... l unite responsable est celle de B
	a := assemblageDeTouches(true, fire, []grammar.DegatLu{message})
	a.opt.Damages.Points = []PointsDeDegat{{Corps: 100}}
	a.doc.Tracks = append(a.doc.Tracks, pisteDeLaTouche(6, xuidC, 0, finDesActions-1))
	a.poserLesTouches()
	doc := documentJSON(t, a.doc)
	hits, _ := doc["hits"].([]any)
	damage, _ := doc["damage"].([]any)
	if len(hits) != 1 || len(damage) != 1 {
		t.Fatalf("touches %v, degats %v : attendu une de chaque", hits, damage)
	}
	for nom, ligne := range map[string]any{"touche": hits[0], "degat": damage[0]} {
		m := ligne.(map[string]any)
		if m["w"] != "0x00000000AB00CD07" {
			t.Errorf("%s : arme %v, attendu celle du dernier tir de B", nom, m["w"])
		}
	}
	if hits[0].(map[string]any)["shooter"] != xuidB {
		t.Errorf("tireur publie %v, attendu B", hits[0].(map[string]any)["shooter"])
	}
}
