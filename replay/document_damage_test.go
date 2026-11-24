package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// TestBuildDamage : une ligne par message qui touche une unité qui a une piste (zéro point compris, cf. TestDegatsAZeroPoint), posée à sa frame ; l'auteur se
// résout par le roster, la victime par la piste (absente pour un bot) ; le total infligé exclut ce qu'on se fait à soi-même,
// le subi ne compte que les joueurs ; une unité sans piste n'est ni publiée ni comptée ; un instant hors de l'axe compte
// aux totaux sans ligne.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter un dégât fait à un véhicule ; compter l'auto-dégât en infligé ; compter un
// bot en subi ; publier une ligne hors de l'axe ; oublier le dégât au bot en infligé ; ne pas arrondir au dixième.
func TestBuildDamage(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	// B (unité 5) et C (unité 6) sont des joueurs, l'unité 9 est un bot (piste sans xuid) ; l'unité 40 n'a pas de piste.
	pistes := []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1), pisteDeLaTouche(6, xuidC, 0, finDesActions-1), pisteDeLaTouche(9, "", 0, finDesActions-1)}
	coup := func(ts uint64, joueur int8, unite uint16) grammar.DegatLu {
		d := degatDeTest(ts, effetArme, joueur, unite)
		d.GenVictime = 0
		return d
	}
	messages := []grammar.DegatLu{
		coup(instantDeLaFrame(3), 0, 5),                           // A touche B
		coup(instantDeLaFrame(4), 0, 9),                           // A touche le bot
		coup(instantDeLaFrame(5), 1, 5),                           // B se touche lui-même
		coup(instantDeLaFrame(6), -1, 6),                          // le décor touche C
		coup(instantDeLaFrame(7), 0, 40),                          // A touche un véhicule : sans piste
		coup(instantDeLaFrame(8), 0, 5),                           // un coup qui ne retire rien
		coup(origineDesActions+finDesActions*pasDesActions, 0, 5), // après la dernière frame : aux totaux, sans ligne
	}
	points := []PointsDeDegat{{Bouclier: 250}, {Bouclier: 100, Corps: 55, Mesure: true}, {Corps: 123}, {Corps: 300}, {Bouclier: 999}, {}, {Bouclier: 11}}
	_, lignes, stats, cov := buildHits(HitInput{Read: true, Messages: messages, Points: points, Stats: grammar.StatsDegats{Trames: 100, Terminees: 100}},
		nil, roster, pistes, axe)
	want := []Damage{
		{T: 3, Attacker: xuidA, Victim: xuidB, Effect: "0000563f", Shield: 25, Slot: 517},
		{T: 4, Attacker: xuidA, Effect: "0000563f", Shield: 10, Body: 5.5, Measured: true, Slot: 521},
		{T: 5, Attacker: xuidB, Victim: xuidB, Effect: "0000563f", Body: 12.3, Slot: 517},
		{T: 6, Victim: xuidC, Effect: "0000563f", Body: 30, Slot: 518},
		{T: 8, Attacker: xuidA, Victim: xuidB, Effect: "0000563f", Slot: 517}, // le coup qui ne retire rien : une ligne à zéro point
	}
	if !slices.Equal(lignes, want) {
		t.Errorf("lignes %+v, attendu %+v", lignes, want)
	}
	if cov.Damages != 4 || cov.DamagesMeasured != 1 || cov.DamagesOffUnit != 1 || cov.SelfDamages != 1 || cov.DamagesZero != 1 {
		t.Errorf("couverture : %d ligne(s) dont %d mesurée(s), %d hors unité, %d sur soi, %d à zéro point ; attendu 4, 1, 1, 1 et 1",
			cov.Damages, cov.DamagesMeasured, cov.DamagesOffUnit, cov.SelfDamages, cov.DamagesZero)
	}
	totaux := map[string][2]float64{}
	for _, s := range stats {
		totaux[s.XUID] = [2]float64{s.DamageDealt, s.DamageTaken}
	}
	// A : 25 + 15,5 + 1,1 (la ligne hors axe compte) ; B : 0 infligé (soi-même), 25 + 12,3 + 1,1 subis ; C : 30 subis.
	wantTotaux := map[string][2]float64{xuidA: {41.6, 0}, xuidB: {0, 38.4}, xuidC: {0, 30}}
	for x, w := range wantTotaux {
		if got := totaux[x]; got != w {
			t.Errorf("totaux de %s : %v, attendu %v", x, got, w)
		}
	}
}
