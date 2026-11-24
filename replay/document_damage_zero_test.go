package replay

import (
	"context"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// degatsDeLaGardeZero fabrique les messages et les points de la garde : des messages à points et des messages à zéro point de chaque sorte
// (doublon, auto-dégât, victime bot, unité sans piste, hors de l'axe, responsable sans piste), dans l'ordre du film.
func degatsDeLaGardeZero() ([]grammar.DegatLu, []PointsDeDegat) {
	avec := func(d grammar.DegatLu, responsable uint16) grammar.DegatLu {
		d.AResponsable, d.Responsable = true, responsable
		return d
	}
	msgs := []grammar.DegatLu{
		avec(degatDeTest(instantDeLaFrame(3), effetArme, 0, 5), 4),                  // A -> B, 25 points ; l'unité 4 est celle de A
		avec(degatDeTest(instantDeLaFrame(3), effetArme, 0, 5), 4),                  // le doublon à zéro point dans la même trame
		avec(degatDeTest(instantDeLaFrame(4), effetArme, 1, 5), 5),                  // B se blesse sans points
		degatDeTest(instantDeLaFrame(5), effetArme, 0, 9),                           // A -> bot, 10 points
		degatDeTest(instantDeLaFrame(6), effetArme, 0, 9),                           // A -> bot, zéro point
		degatDeTest(instantDeLaFrame(7), effetArme, 0, 40),                          // unité sans piste, zéro point
		avec(degatDeTest(instantDeLaFrame(8), effetArme, -1, 6), 41),                // responsable sans piste, zéro point
		degatDeTest(origineDesActions+finDesActions*pasDesActions, effetArme, 0, 5), // hors de l'axe, zéro point
		degatDeTest(instantDeLaFrame(9), effetArme, 0, 40),                          // unité sans piste, 5 points
		degatDeTest(instantDeLaFrame(10), effetArme, 1, 5),                          // B se blesse, 12,3 points
	}
	pts := []PointsDeDegat{{Bouclier: 250}, {}, {}, {Corps: 100}, {}, {}, {}, {}, {Corps: 50}, {Corps: 123}}
	return msgs, pts
}

func pistesDeLaGardeZero() []Track {
	return []Track{pisteDeLaTouche(4, xuidA, 0, finDesActions-1), pisteDeLaTouche(5, xuidB, 0, finDesActions-1),
		pisteDeLaTouche(6, xuidC, 0, finDesActions-1), pisteDeLaTouche(9, "", 0, finDesActions-1)}
}

func lireLesDegatsDeLaGarde(msgs []grammar.DegatLu, pts []PointsDeDegat) ([]Hit, []Damage, []ShotStat, HitCoverage) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	return buildHits(HitInput{Read: true, Messages: msgs, Points: pts, Stats: grammar.StatsDegats{Trames: 100, Terminees: 100}},
		nil, rosterDesActions(), pistesDeLaGardeZero(), axe)
}

// TestDegatsAZeroPoint : un message qui ne retire rien est publié (valeurs nulles) sur une unité qui a une piste et dans l'axe, avec le slot
// de la victime et celui du responsable quand il a une piste ; une unité sans piste ou un instant hors de l'axe a son compteur propre.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : omettre le slot d'une victime bot ; poser `by` sans piste du responsable ; poser un `by` hors
// responsable nommé ; compter une ligne à zéro point dans `damages` ; ne pas séparer l'unité sans piste d'un zéro point de `damagesOffUnit`.
func TestDegatsAZeroPoint(t *testing.T) {
	msgs, pts := degatsDeLaGardeZero()
	_, lignes, _, cov := lireLesDegatsDeLaGarde(msgs, pts)
	want := []Damage{
		{T: 3, Attacker: xuidA, Victim: xuidB, Effect: "0000563f", Shield: 25, Slot: 517, By: 516},
		{T: 3, Attacker: xuidA, Victim: xuidB, Effect: "0000563f", Slot: 517, By: 516},
		{T: 4, Attacker: xuidB, Victim: xuidB, Effect: "0000563f", Slot: 517, By: 517},
		{T: 5, Attacker: xuidA, Effect: "0000563f", Body: 10, Slot: 521},
		{T: 6, Attacker: xuidA, Effect: "0000563f", Slot: 521},
		{T: 8, Victim: xuidC, Effect: "0000563f", Slot: 518}, // responsable 41 sans piste : pas de `by`
		{T: 10, Attacker: xuidB, Victim: xuidB, Effect: "0000563f", Body: 12.3, Slot: 517},
	}
	if !slices.Equal(lignes, want) {
		t.Errorf("lignes %+v, attendu %+v", lignes, want)
	}
	if cov.Damages != 3 || cov.DamagesZero != 4 || cov.DamagesZeroOffUnit != 1 || cov.DamagesZeroOutOfAxis != 1 || cov.DamagesOffUnit != 1 || cov.SelfDamages != 1 {
		t.Errorf("couverture : %d avec points, %d à zéro, %d zéro hors unité, %d zéro hors axe, %d hors unité, %d sur soi ; attendu 3, 4, 1, 1, 1 et 1",
			cov.Damages, cov.DamagesZero, cov.DamagesZeroOffUnit, cov.DamagesZeroOutOfAxis, cov.DamagesOffUnit, cov.SelfDamages)
	}
}

// TestLesZeroPointNeBougentAucunCompte est la garde R-14 des dégâts : les mêmes messages lus avec leurs messages à zéro point et sans eux
// rendent les mêmes totaux de `shotStats[]`, les mêmes lignes à points et les mêmes compteurs de dégâts ; seuls les lignes et les
// compteurs à zéro point diffèrent.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter l'auto-dégât d'un zéro point dans `selfDamages` ; compter un zéro point dans
// `damagesMeasured` ou `damagesOffUnit` ; l'ajouter au bilan d'un joueur.
func TestLesZeroPointNeBougentAucunCompte(t *testing.T) {
	msgs, pts := degatsDeLaGardeZero()
	var msgsSans []grammar.DegatLu
	var ptsSans []PointsDeDegat
	for i, p := range pts {
		if p.Bouclier+p.Corps > 0 {
			msgsSans, ptsSans = append(msgsSans, msgs[i]), append(ptsSans, p)
		}
	}
	_, lignes, stats, cov := lireLesDegatsDeLaGarde(msgs, pts)
	_, lignesS, statsS, covS := lireLesDegatsDeLaGarde(msgsSans, ptsSans)
	// Les totaux de dégât (infligé, subi) ne bougent pas ; les messages et les touches d'un joueur se comptent sur les messages d'arme, que
	// les points ne décident pas (cf. TestLesTouchesIgnorentLesPointsDesDegats) : ils ne sont pas comparés ici, la liste de messages diffère.
	if len(stats) != len(statsS) {
		t.Fatalf("joueurs au bilan : %d, sans les zéro-point %d", len(stats), len(statsS))
	}
	for i := range stats {
		if stats[i].XUID != statsS[i].XUID || stats[i].DamageDealt != statsS[i].DamageDealt || stats[i].DamageTaken != statsS[i].DamageTaken {
			t.Errorf("totaux de dégât de %s : (%v, %v), sans les zéro-point (%v, %v)", stats[i].XUID,
				stats[i].DamageDealt, stats[i].DamageTaken, statsS[i].DamageDealt, statsS[i].DamageTaken)
		}
	}
	var avecPoints []Damage
	for _, l := range lignes {
		if l.Shield+l.Body > 0 {
			avecPoints = append(avecPoints, l)
		}
	}
	if !slices.Equal(avecPoints, lignesS) {
		t.Errorf("lignes à points : %+v, sans les zéro-point %+v", avecPoints, lignesS)
	}
	if cov.Damages != covS.Damages || cov.DamagesMeasured != covS.DamagesMeasured || cov.DamagesOffUnit != covS.DamagesOffUnit || cov.SelfDamages != covS.SelfDamages {
		t.Errorf("compteurs de dégâts (%d, %d, %d, %d), sans les zéro-point (%d, %d, %d, %d)",
			cov.Damages, cov.DamagesMeasured, cov.DamagesOffUnit, cov.SelfDamages,
			covS.Damages, covS.DamagesMeasured, covS.DamagesOffUnit, covS.SelfDamages)
	}
	if covS.DamagesZero+covS.DamagesZeroOffUnit+covS.DamagesZeroOutOfAxis != 0 {
		t.Errorf("sans message à zéro point, les compteurs à zéro doivent être nuls : %+v", covS)
	}
}

// TestLesTouchesIgnorentLesPointsDesDegats : les touches se comptent sur les messages d'arme, jamais sur les lignes de `damage[]` ni sur leurs
// points : les mêmes messages avec tous leurs points à zéro rendent les mêmes touches et la même couverture de touches.
func TestLesTouchesIgnorentLesPointsDesDegats(t *testing.T) {
	msgs, pts := degatsDeLaGardeZero()
	h1, _, s1, c1 := lireLesDegatsDeLaGarde(msgs, pts)
	h2, _, s2, c2 := lireLesDegatsDeLaGarde(msgs, make([]PointsDeDegat, len(pts)))
	// Un joueur qui ne fait que subir des points n'est au bilan que par eux : sans points il n'y est plus, ce qui est le comportement historique.
	parXUID := map[string]ShotStat{}
	for _, s := range s2 {
		parXUID[s.XUID] = s
	}
	if !slices.Equal(h1, h2) {
		t.Errorf("touches %+v contre %+v : les points ne décident d'aucune touche", h1, h2)
	}
	for _, a := range s1 {
		b, ok := parXUID[a.XUID]
		if !ok {
			continue
		}
		a.DamageDealt, a.DamageTaken, b.DamageDealt, b.DamageTaken = 0, 0, 0, 0
		if a != b {
			t.Errorf("bilan de tir de %s : %+v contre %+v", a.XUID, a, b)
		}
	}
	if c1.Hits != c2.Hits || c1.WeaponMessages != c2.WeaponMessages || c1.Published != c2.Published || c1.SelfHits != c2.SelfHits {
		t.Errorf("couverture des touches %+v contre %+v", c1, c2)
	}
}

// TestDecalageDansLaFrame : le décalage est le reste de la division de l'instant, depuis l'origine, par le pas, en millisecondes entières ;
// zéro quand l'instant précède l'origine, que la grille n'a pas de pas, ou que la frame donnée n'est pas celle de l'instant (frame bornée).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : arrondir au lieu de tronquer ; oublier l'origine ; ignorer la frame bornée ; diviser par zéro.
func TestDecalageDansLaFrame(t *testing.T) {
	const origine, pas = 1_000_000, 100_000
	for _, c := range []struct {
		nom   string
		t     uint64
		frame int
		want  int
	}{
		{"pile sur la frame", origine + 3*pas, 3, 0},
		{"milieu", origine + 3*pas + 42_000, 3, 42},
		{"troncature", origine + 3*pas + 42_999, 3, 42},
		{"dernier instant de la frame", origine + 3*pas + 99_999, 3, 99},
		{"avant l'origine", origine - 1, 0, 0},
		{"frame bornée", origine + 9*pas + 42_000, 4, 0},
	} {
		if got := decalageDansLaFrame(c.t, origine, pas, c.frame); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.nom, got, c.want)
		}
	}
	if got := decalageDansLaFrame(origine+5, origine, 0, 0); got != 0 {
		t.Errorf("sans pas : %d, attendu 0", got)
	}
}

// TestLeTirPorteSonDecalageDansLaFrame : un tir à pied et un tir en véhicule publient `o`, l'écart en millisecondes à sa frame ; un tir pile sur
// sa frame n'en porte pas.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne poser `o` que sur la voie à pied ; le poser sur la voie véhicule sans l'origine ; le publier à
// zéro (il serait sérialisé).
func TestLeTirPorteSonDecalageDansLaFrame(t *testing.T) {
	var pos []grammar.BipedPosition
	for i := range uint64(20) {
		pos = append(pos, posAt(10, 1_000_000+i*50_000, 1, 1, 90))
	}
	events := []grammar.FireEvent{fireAt(1_237_000, 3, 90), fireAt(1_300_000, 3, 90)}
	shots, _, _ := buildShots(indexBySlot(pos), events, grilleDeTest, occupantsPlats(map[uint32]int{10: 3}), nouveauViseur(pos, nil, nil))
	if len(shots) != 2 || shots[0].T != 2 || shots[0].O != 37 || shots[1].T != 3 || shots[1].O != 0 {
		t.Fatalf("tirs à pied %+v, attendu (frame 2, 37 ms) puis (frame 3, 0 ms)", shots)
	}
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}, {T: 30, X: 20, Y: 40}})
	o := vsOrphan(20, 0x11725DC400000000)
	o.ev.TimestampUS += 42_000
	attachVehicleShots(context.Background(), doc, orphelinsSansVisee(o), vsOwn(), vsClock())
	if len(doc.Shots) != 1 || doc.Shots[0].T != 20 || doc.Shots[0].O != 42 {
		t.Fatalf("tir en véhicule %+v, attendu frame 20 et 42 ms", doc.Shots)
	}
}
