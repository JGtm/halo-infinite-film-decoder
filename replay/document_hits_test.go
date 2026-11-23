package replay

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Effets de dégât du catalogue (damageeffects/data/effects.tsv).
const (
	effetArme    = 0x0000563f
	effetGrenade = 0xff768294
)

func tirDeTest(joueur int, numero uint8, ts uint64, arme uint64) grammar.FireEvent {
	return grammar.FireEvent{HasShooter: true, FilmIndex: joueur, FireNumber: numero, TimestampUS: ts, WeaponID: arme}
}

func degatDeTest(ts uint64, tag uint32, joueur int8, victime uint16) grammar.DegatLu {
	return grammar.DegatLu{TS: ts, Tag: tag, Joueur: joueur, Victime: victime, ATag: true, AVictime: true}
}

// pisteDeTest : la piste de l'unité d'index `unite` porte le slot de l'index plus 512.
func pisteDeLaTouche(unite uint32, xuid string, debut, fin int) Track {
	return Track{Slot: unite + 512, XUID: xuid, StartFrame: debut, EndFrame: fin}
}

// TestBuildHits : une touche est un message d'effet d'arme, une fois par (auteur, unité, trame), posée à la frame de
// son instant avec l'arme du dernier tir de l'auteur dans les deux secondes ; l'auteur se résout par le roster présent,
// la victime par la piste de son unité ; chaque refus a son compteur.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter une grenade ; ne pas dédoublonner les messages d'une trame ; nommer
// l'arme d'un tir de plus de deux secondes ; résoudre la victime par le slot sans décalage ; poser une touche hors de
// l'axe ; ignorer la présence de l'auteur.
func TestBuildHits(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1), pisteDeLaTouche(6, xuidC, 0, 49), pisteDeLaTouche(6, xuidD, 50, finDesActions-1)}
	const arme = 0x00000000AB00CD00
	cas := []struct {
		nom    string
		fire   []grammar.FireEvent
		degats []grammar.DegatLu
		hits   []Hit
		cov    HitCoverage
	}{
		{
			"une touche, l'arme du dernier tir", []grammar.FireEvent{tirDeTest(0, 1, instantDeLaFrame(2), arme)},
			[]grammar.DegatLu{degatDeTest(instantDeLaFrame(3), effetArme, 0, 5)},
			[]Hit{{T: 3, Shooter: xuidA, Victim: xuidB, Effect: "0000563f", Weapon: "0x00000000AB00CD00"}},
			HitCoverage{WeaponMessages: 1, Hits: 1, Published: 1},
		},
		{
			"un tir de plus de deux secondes ne nomme pas l'arme", []grammar.FireEvent{tirDeTest(0, 1, origineDesActions-1_000_000, arme)},
			[]grammar.DegatLu{degatDeTest(instantDeLaFrame(90), effetArme, 0, 5)},
			[]Hit{{T: 90, Shooter: xuidA, Victim: xuidB, Effect: "0000563f"}},
			HitCoverage{WeaponMessages: 1, Hits: 1, Published: 1},
		},
		{
			"deux messages de la même trame : une touche", nil,
			[]grammar.DegatLu{degatDeTest(instantDeLaFrame(4), effetArme, 0, 5), degatDeTest(instantDeLaFrame(4), effetArme, 0, 5)},
			[]Hit{{T: 4, Shooter: xuidA, Victim: xuidB, Effect: "0000563f"}},
			HitCoverage{WeaponMessages: 2, Hits: 1, Published: 1},
		},
		{
			"une grenade, un message sans auteur, un message sans unité blessée : aucune touche", nil,
			[]grammar.DegatLu{
				degatDeTest(instantDeLaFrame(4), effetGrenade, 0, 5),
				degatDeTest(instantDeLaFrame(4), effetArme, -1, 5),
				{TS: instantDeLaFrame(4), Tag: effetArme, ATag: true, Joueur: 0},
			},
			nil, HitCoverage{},
		},
		{
			"un auteur parti, une victime sans piste, avant l'origine, après la fin", nil,
			[]grammar.DegatLu{
				degatDeTest(instantDeLaFrame(60), effetArme, 6, 5),                          // l'humain F est parti à la frame 10
				degatDeTest(instantDeLaFrame(4), effetArme, 0, 40),                          // aucune piste pour l'unité 40
				degatDeTest(origineDesActions-1, effetArme, 0, 5),                           // avant la frame 0
				degatDeTest(origineDesActions+finDesActions*pasDesActions, effetArme, 0, 5), // après la dernière
			},
			nil, HitCoverage{WeaponMessages: 4, Hits: 4, NoShooter: 1, NoVictim: 1, BeforeOrigin: 1, AfterEnd: 1},
		},
		{
			"la victime remplacée sur la même unité : la piste de la frame", nil,
			[]grammar.DegatLu{degatDeTest(instantDeLaFrame(20), effetArme, 0, 6), degatDeTest(instantDeLaFrame(70), effetArme, 0, 6)},
			[]Hit{{T: 20, Shooter: xuidA, Victim: xuidC, Effect: "0000563f"}, {T: 70, Shooter: xuidA, Victim: xuidD, Effect: "0000563f"}},
			HitCoverage{WeaponMessages: 2, Hits: 2, Published: 2},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			hits, _, _, cov := buildHits(HitInput{Read: true, Messages: c.degats}, c.fire, roster, pistes, axe)
			if !slices.Equal(hits, c.hits) {
				t.Errorf("touches %+v, attendu %+v", hits, c.hits)
			}
			c.cov.Estimated, c.cov.MinCoverage, c.cov.MaxDuplicates, c.cov.MaxStoppedShare = true, minCouvertureDesTirs, maxDoublonsDeTouches, maxPartDeTramesArretees
			c.cov.Messages = len(c.degats)
			cov.Players, cov.Sufficient = 0, 0
			// Ces messages sont sans points : ils sont des lignes à zéro point, que TestDegatsAZeroPoint compte ; les touches ne les voient pas.
			cov.DamagesZero, cov.DamagesZeroOffUnit, cov.DamagesZeroOutOfAxis = 0, 0, 0
			if cov != c.cov {
				t.Errorf("couverture %+v, attendu %+v", cov, c.cov)
			}
		})
	}
}

// TestLeBilanDeTirsDitSiLesDonneesSuffisent : les tirs sont la somme des sauts de la numérotation (le retour de 255 à 0
// compte), la couverture est leur part écrite, les doublons le rapport des messages aux touches, et les deux seuils
// décident ; un joueur sans touche n'est jamais suffisant.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter les événements de tir comme les tirs ; ne pas replier la numérotation
// modulo 256 ; inverser un seuil ; juger suffisant un joueur sans touche.
func TestLeBilanDeTirsDitSiLesDonneesSuffisent(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidD, 0, finDesActions-1)} // la victime n est aucun des tireurs
	// A : numéros 250, 252, 255, 2 -> 1 + 2 + 3 + 3 = 9 tirs pour 4 écrits ; B et C : numéros 1, 2, 3 -> 3 tirs, 3 écrits.
	// D (index 7, sans présence) tire sans être jamais résolu.
	fire := []grammar.FireEvent{
		tirDeTest(0, 250, instantDeLaFrame(1), 1), tirDeTest(0, 252, instantDeLaFrame(2), 1), tirDeTest(0, 255, instantDeLaFrame(3), 1), tirDeTest(0, 2, instantDeLaFrame(4), 1),
		tirDeTest(1, 1, instantDeLaFrame(1), 1), tirDeTest(1, 2, instantDeLaFrame(2), 1), tirDeTest(1, 3, instantDeLaFrame(3), 1),
		tirDeTest(2, 1, instantDeLaFrame(1), 1), tirDeTest(2, 2, instantDeLaFrame(2), 1), tirDeTest(2, 3, instantDeLaFrame(3), 1),
		tirDeTest(7, 1, instantDeLaFrame(1), 1),
	}
	degats := []grammar.DegatLu{
		degatDeTest(instantDeLaFrame(5), effetArme, 0, 5), // A : 1 touche
		degatDeTest(instantDeLaFrame(5), effetArme, 1, 5), // B : 2 touches, 2 messages
		degatDeTest(instantDeLaFrame(6), effetArme, 1, 5),
		degatDeTest(instantDeLaFrame(7), effetArme, 2, 5), // C : 1 touche écrite en 2 messages, doublons 2,0
		degatDeTest(instantDeLaFrame(7), effetArme, 2, 5),
	}
	_, _, stats, cov := buildHits(HitInput{Read: true, Messages: degats, Stats: grammar.StatsDegats{Trames: 1000, Terminees: 1000}}, fire, roster, pistes, axe)
	want := []ShotStat{
		{XUID: xuidA, Shots: 9, ShotEvents: 4, Messages: 1, Hits: 1, Coverage: 4.0 / 9.0, Duplicates: 1},
		{XUID: xuidB, Shots: 3, ShotEvents: 3, Messages: 2, Hits: 2, Coverage: 1, Duplicates: 1, Sufficient: true},
		{XUID: xuidC, Shots: 3, ShotEvents: 3, Messages: 2, Hits: 1, Coverage: 1, Duplicates: 2},
	}
	if !slices.Equal(stats, want) {
		t.Errorf("bilan %+v, attendu %+v", stats, want)
	}
	if cov.Players != 3 || cov.Sufficient != 1 {
		t.Errorf("couverture : %d joueur(s) dont %d suffisant(s), attendu 3 et 1", cov.Players, cov.Sufficient)
	}
}

// TestUneLectureArreteeRendTousLesJoueursInsuffisants : au-delà de 0,5 % de trames dont la vue A s'arrête sur un message non lu,
// les messages de dégât perdus rendent les touches incomparables : aucun joueur n'est suffisant, même à bonne couverture ; en
// deçà, la suffisance suit les deux autres critères ; sans trame lue, aucune.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas regarder les trames arrêtées ; comparer en nombre et non en part ; juger
// suffisant un film sans trame ; ne pas compter les vitalités non lues avec les trames arrêtées.
func TestUneLectureArreteeRendTousLesJoueursInsuffisants(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidD, 0, finDesActions-1)} // la victime n est aucun des tireurs
	fire := []grammar.FireEvent{tirDeTest(1, 1, instantDeLaFrame(1), 1), tirDeTest(1, 2, instantDeLaFrame(2), 1)}
	degats := []grammar.DegatLu{degatDeTest(instantDeLaFrame(3), effetArme, 1, 5)}
	cas := []struct {
		nom   string
		stats grammar.StatsDegats
		suff  bool
	}{
		{"lue jusqu'au bout", grammar.StatsDegats{Trames: 1000, Terminees: 1000}, true},
		{"5 arrêts sur 1000 trames : à la limite", grammar.StatsDegats{Trames: 1000, Terminees: 995, Arretees: 5}, true},
		{"6 arrêts sur 1000 trames", grammar.StatsDegats{Trames: 1000, Terminees: 994, Arretees: 6}, false},
		{"6 vitalités non lues sur 1000 trames", grammar.StatsDegats{Trames: 1000, Terminees: 994, VitalesNonLues: 6}, false},
		{"3 arrêts et 2 vitalités non lues : à la limite", grammar.StatsDegats{Trames: 1000, Terminees: 995, Arretees: 3, VitalesNonLues: 2}, true},
		{"3 arrêts et 3 vitalités non lues", grammar.StatsDegats{Trames: 1000, Terminees: 994, Arretees: 3, VitalesNonLues: 3}, false},
		{"aucune trame", grammar.StatsDegats{}, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, _, stats, cov := buildHits(HitInput{Read: true, Messages: degats, Stats: c.stats}, fire, roster, pistes, axe)
			if len(stats) != 1 || stats[0].Sufficient != c.suff || (cov.Sufficient == 1) != c.suff {
				t.Errorf("bilan %+v, couverture %+v, suffisant attendu %v", stats, cov, c.suff)
			}
		})
	}
}

// TestUneToucheSurSoiNEstPasUneTouche : un message d'effet d'arme dont la victime est l'auteur (une arme à zone sur son
// propre tireur) n'est ni publié, ni compté aux touches ni aux messages du joueur, ni aux doublons : il est compté à part
// (`selfHits`, une fois par auteur, unité et trame) ; une touche sur un autre joueur de la même trame reste une touche.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : compter la touche sur soi ; la publier ; la compter aux messages du joueur.
func TestUneToucheSurSoiNEstPasUneTouche(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1), pisteDeLaTouche(6, xuidC, 0, finDesActions-1)}
	degats := []grammar.DegatLu{
		degatDeTest(instantDeLaFrame(5), effetArme, 1, 5), // B sur sa propre unité
		degatDeTest(instantDeLaFrame(5), effetArme, 1, 5), // le même message, redoublé
		degatDeTest(instantDeLaFrame(5), effetArme, 1, 6), // B sur C : une touche
	}
	hits, _, stats, cov := buildHits(HitInput{Read: true, Messages: degats, Stats: grammar.StatsDegats{Trames: 1000, Terminees: 1000}}, nil, roster, pistes, axe)
	if want := []Hit{{T: 5, Shooter: xuidB, Victim: xuidC, Effect: "0000563f"}}; !slices.Equal(hits, want) {
		t.Errorf("touches %+v, attendu %+v", hits, want)
	}
	if cov.SelfHits != 1 || cov.Hits != 1 || cov.WeaponMessages != 3 {
		t.Errorf("couverture : %d sur soi, %d touche(s), %d message(s) d'arme ; attendu 1, 1 et 3", cov.SelfHits, cov.Hits, cov.WeaponMessages)
	}
	if len(stats) != 1 || stats[0].XUID != xuidB || stats[0].Hits != 1 || stats[0].Messages != 1 {
		t.Errorf("bilan %+v, attendu B avec une touche et un message", stats)
	}
}

// assemblageDeTouches ouvre un assemblage qui n'a que ce que la pose des touches lit : un film dont l'index de tireur est (ou
// n'est pas) la place, un roster dont chaque siege est l'index, des tirs et des messages de degat.
func assemblageDeTouches(indexEstLaPlace bool, fire []grammar.FireEvent, degats []grammar.DegatLu) *assemblage {
	roster := rosterDesActions()
	for i := range roster {
		roster[i].Seat = roster[i].FilmIndex
	}
	accord := accordIndexUnite{total: 10}
	if indexEstLaPlace {
		accord.accord = 10
	}
	return &assemblage{
		ctx: context.Background(), opt: Options{Fallbacks: fallback.NouveauCompteur(),
			Damages: HitInput{Read: true, Messages: degats, Stats: grammar.StatsDegats{Trames: 1000, Terminees: 1000}}},
		fire: fire, origin: origineDesActions, step: pasDesActions,
		doc: ReplayDocument{FrameCount: finDesActions, Roster: roster, Coverage: &Coverage{},
			Tracks: []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1)}},
		indexTireur: accord,
	}
}

// TestLesTouchesLisentLesTirsDuResteDeLAssemblage : sur un film dont l'index de tireur n'est pas la place, aucun tir n'est
// attribue par l'index (ni tir ecrit, ni numerotation, ni arme) et aucun joueur n'est juge suffisant ; sur un film dont il est la
// place, les tirs, l'arme et le verdict sont ceux de l'index. La condition est decidee une fois, par `tirsDuCombat`, pour tous
// les calques du combat.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : inverser la condition ; lire les tirs bruts du film dans la pose des touches.
func TestLesTouchesLisentLesTirsDuResteDeLAssemblage(t *testing.T) {
	const arme = 0x00000000AB00CD00
	fire := []grammar.FireEvent{
		tirDeTest(0, 1, instantDeLaFrame(1), arme), tirDeTest(0, 2, instantDeLaFrame(2), arme), tirDeTest(0, 3, instantDeLaFrame(3), arme),
	}
	degats := []grammar.DegatLu{degatDeTest(instantDeLaFrame(5), effetArme, 0, 5)}
	cas := []struct {
		nom        string
		laPlace    bool
		tirs       int
		arme       string
		suffisants int
	}{
		{"l'index n'est pas la place : aucune attribution", false, 0, "", 0},
		{"l'index est la place : les tirs de l'index", true, 3, "0x00000000AB00CD00", 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := assemblageDeTouches(c.laPlace, fire, degats)
			a.poserLesTouches()
			if len(a.doc.ShotStats) != 1 || a.doc.ShotStats[0].Shots != c.tirs || a.doc.ShotStats[0].ShotEvents != c.tirs {
				t.Fatalf("bilan %+v, attendu %d tirs ecrits et numerotes", a.doc.ShotStats, c.tirs)
			}
			if len(a.doc.Hits) != 1 || a.doc.Hits[0].Weapon != c.arme {
				t.Errorf("touches %+v, arme attendue %q", a.doc.Hits, c.arme)
			}
			if a.doc.Coverage.Hits.Sufficient != c.suffisants || a.doc.ShotStats[0].Sufficient != (c.suffisants == 1) {
				t.Errorf("suffisants %d (verdict %v), attendu %d", a.doc.Coverage.Hits.Sufficient, a.doc.ShotStats[0].Sufficient, c.suffisants)
			}
		})
	}
}

// TestLaNumerotationDesTirsEstCelleDuJoueur : deux humains successifs sur le meme index (un remplacant) ont chacun leur
// numerotation : le premier numero du remplacant ne compte pas un saut contre le dernier du partant.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : indexer la numerotation par l'index et non par le joueur resolu.
func TestLaNumerotationDesTirsEstCelleDuJoueur(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	r := resolveurDeTouches{humains: humainsParIndex(rosterDesActions()), axe: axe}
	// D (index 4) tire les numeros 250 et 251 avant de partir a la frame 49 ; E (meme index) prend le relais a la frame 50 avec les numeros 3 et 4.
	fire := []grammar.FireEvent{
		tirDeTest(4, 250, instantDeLaFrame(10), 1), tirDeTest(4, 251, instantDeLaFrame(11), 1),
		tirDeTest(4, 3, instantDeLaFrame(60), 1), tirDeTest(4, 4, instantDeLaFrame(61), 1),
	}
	bilan := bilanDesTirs{}
	compterLesTirs(fire, r, bilan)
	if d, e := bilan[xuidD], bilan[xuidE]; d == nil || e == nil || d.ShotEvents != 2 || d.Shots != 2 || e.ShotEvents != 2 || e.Shots != 2 {
		t.Fatalf("bilan D %+v, E %+v : deux tirs ecrits et deux tirs numerotes pour chacun", d, e)
	}
}

// TestLArmeDUneToucheEstCelleDuJoueurALInstant : deux humains successifs sur le meme index ont chacun leurs tirs ; la touche (et le degat) du
// second ne nomme jamais l'arme du dernier tir du partant, meme dans les deux secondes. Un bot, sans identifiant, se lit par son index.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ranger les tirs par l'index seul ; resoudre l'auteur du message a une autre frame que celui du tir.
func TestLArmeDUneToucheEstCelleDuJoueurALInstant(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1)}
	const armeD, armeE, armeBot = 0x00000000AB00CD01, 0x00000000AB00CD02, 0x00000000AB00CD03
	// D (index 4) tire a la frame 48 puis part ; E (meme index) touche a la frame 52 sans avoir tire, puis tire a la frame 60 et touche a la 62.
	// Le bot de l'index 3 tire a la frame 10 et touche a la frame 11.
	fire := []grammar.FireEvent{
		tirDeTest(4, 1, instantDeLaFrame(48), armeD), tirDeTest(4, 1, instantDeLaFrame(60), armeE), tirDeTest(3, 1, instantDeLaFrame(10), armeBot),
	}
	messages := []grammar.DegatLu{
		degatDeTest(instantDeLaFrame(52), effetArme, 4, 5), degatDeTest(instantDeLaFrame(62), effetArme, 4, 5), degatDeTest(instantDeLaFrame(11), effetArme, 3, 5),
	}
	points := []PointsDeDegat{{Corps: 100}, {Corps: 100}, {Corps: 100}}
	hits, lignes, _, _ := buildHits(HitInput{Read: true, Messages: messages, Points: points, Stats: grammar.StatsDegats{Trames: 100, Terminees: 100}}, fire, roster, pistes, axe)
	armes := map[int]string{}
	for _, h := range hits {
		armes[h.T] = h.Weapon
	}
	wantTouches := map[int]string{52: "", 62: "0x00000000AB00CD02"} // le bot ne pose aucune touche : il n'a pas de joueur
	if !maps.Equal(armes, wantTouches) {
		t.Errorf("armes des touches %v, attendu %v", armes, wantTouches)
	}
	want := map[int]string{11: "0x00000000AB00CD03", 52: "", 62: "0x00000000AB00CD02"}
	degats := map[int]string{}
	for _, l := range lignes {
		degats[l.T] = l.Weapon
	}
	if !maps.Equal(degats, want) {
		t.Errorf("armes des degats %v, attendu %v", degats, want)
	}
}

// TestLAuteurDUnMessageEstLeJoueurDeSonUnite : un message dont l'index de joueur ne résout aucun humain présent (la
// présence publiée du joueur est close, G n'en a aucune) mais dont l'unité responsable porte la piste de G est une touche
// de G, au bilan, sur l'axe et dans les dégâts ; une unité sans piste, ou celle d'un bot, laisse l'index décider.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : résoudre l'auteur par l'index seul ; prendre une unité sans xuid pour un humain ;
// ne pas poser l'auteur des lignes de dégât par l'unité.
func TestLAuteurDUnMessageEstLeJoueurDeSonUnite(t *testing.T) {
	axe := axeDeFrames{origin: origineDesActions, step: pasDesActions, frames: finDesActions}
	roster := rosterDesActions()
	pistes := []Track{pisteDeLaTouche(5, xuidB, 0, finDesActions-1), pisteDeLaTouche(8, xuidG, 0, finDesActions-1), pisteDeLaTouche(9, "", 0, finDesActions-1)}
	avecUnite := func(d grammar.DegatLu, unite uint16) grammar.DegatLu {
		d.Responsable, d.AResponsable = unite, true
		return d
	}
	messages := []grammar.DegatLu{
		avecUnite(degatDeTest(instantDeLaFrame(3), effetArme, 7, 5), 8),  // index 7 : G n'a aucune présence ; l'unité 8 est celle de G
		avecUnite(degatDeTest(instantDeLaFrame(4), effetArme, 0, 5), 40), // unité sans piste : l'index 0 résout A
		avecUnite(degatDeTest(instantDeLaFrame(5), effetArme, 0, 5), 9),  // unité d'un bot : l'index 0 résout A
		avecUnite(degatDeTest(instantDeLaFrame(6), effetArme, 7, 5), 40), // ni l'un ni l'autre : aucun auteur
	}
	points := []PointsDeDegat{{Corps: 100}, {Corps: 100}, {Corps: 100}, {Corps: 100}}
	hits, lignes, stats, cov := buildHits(HitInput{Read: true, Messages: messages, Points: points, Stats: grammar.StatsDegats{Trames: 100, Terminees: 100}}, nil, roster, pistes, axe)
	wantHits := []Hit{{T: 3, Shooter: xuidG, Victim: xuidB, Effect: "0000563f"}, {T: 4, Shooter: xuidA, Victim: xuidB, Effect: "0000563f"}, {T: 5, Shooter: xuidA, Victim: xuidB, Effect: "0000563f"}}
	if !slices.Equal(hits, wantHits) {
		t.Errorf("touches %+v, attendu %+v", hits, wantHits)
	}
	if cov.NoShooter != 1 {
		t.Errorf("%d touche(s) sans auteur, attendu 1 (le dernier message)", cov.NoShooter)
	}
	degats := map[int]string{}
	for _, l := range lignes {
		degats[l.T] = l.Attacker
	}
	if want := map[int]string{3: xuidG, 4: xuidA, 5: xuidA, 6: ""}; !maps.Equal(degats, want) {
		t.Errorf("auteurs des lignes de dégât %v, attendu %v", degats, want)
	}
	par := map[string]int{}
	for _, s := range stats {
		par[s.XUID] = s.Hits
	}
	if par[xuidG] != 1 || par[xuidA] != 2 {
		t.Errorf("touches au bilan : G %d, A %d ; attendu 1 et 2", par[xuidG], par[xuidA])
	}
}
