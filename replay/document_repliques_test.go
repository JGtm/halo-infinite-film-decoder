package replay

import (
	"context"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// document_repliques_test.go — les repliques des Spartans et les annonces du document, sur des donnees construites.

// axeDeRepliques est l axe de frames des cas : origine a 1 s, un pas de 40 ms, 100 frames.
var axeDeRepliques = axeDeFrames{origin: 1_000_000, step: 40_000, frames: 100}

// auTemps rend l horodatage de la frame `f` de l axe, plus `reste` microsecondes.
func auTemps(f int, reste uint64) uint64 {
	return axeDeRepliques.origin + uint64(f)*axeDeRepliques.step + reste
}

// pisteDUnite est la piste de l unite `u`, de la frame `de` a la frame `a` (0 = ouverte), au joueur `xuid` (vide : un bot).
func pisteDUnite(u uint16, xuid string, de, a int) Track {
	return Track{Slot: uint32(u) + decalageDesSlotsDUnite, XUID: xuid, StartFrame: de, EndFrame: a}
}

// replique construit une replique lue : l unite qui parle (ou -1), l index de joueur de la charge (ou -1), le son.
func replique(ts uint64, unite, joueur int, son uint32) grammar.RepliqueLue {
	m := grammar.RepliqueLue{TS: ts, Son: son, ASon: true, Ligne: 9, ALigne: true, Priorite: 7, Joueur: int8(joueur)} //nolint:gosec // cas de test
	if unite >= 0 {
		m.Locuteur, m.ALocuteur = uint16(unite), true //nolint:gosec // cas de test
	}
	return m
}

// TestLesRepliquesSePosentSousLeJoueurDeLUniteQuiParle : le joueur d une replique est celui de la piste de son unite a la frame,
// jamais celui de l index de la charge ; une unite sans reference, sans piste a cette frame ou a piste de bot donne une replique
// publiee sans joueur et comptee ; les repliques hors de l axe sont ecartees et comptees ; l identite de la couverture tient.
// MUTATIONS : le joueur pris dans l index de la charge, une piste lue hors de sa fenetre, un bot compte comme humain, une borne
// d axe decalee, une replique sans joueur ecartee -> ROUGE.
func TestLesRepliquesSePosentSousLeJoueurDeLUniteQuiParle(t *testing.T) {
	pistes := []Track{
		pisteDUnite(5, "111", 0, 50),
		pisteDUnite(5, "222", 51, 0), // la meme unite, un autre joueur apres la frame 50
		pisteDUnite(6, "", 0, 0),     // un bot
		pisteDUnite(7, "333", 0, 10),
	}
	in := grammar.RepliquesLues{
		Stats: grammar.StatsRepliques{Trames: 80, Terminees: 78, Arretees: 2, Repliques: 8},
		Repliques: []grammar.RepliqueLue{
			replique(auTemps(10, 0), 5, 2, 0xAA),       // l index de la charge nomme un autre joueur : l unite fait foi
			replique(auTemps(60, 39_999), 5, -1, 0xBB), // la meme unite, plus tard : l autre joueur
			replique(auTemps(20, 0), 6, -1, 0xCC),      // un bot
			replique(auTemps(30, 0), 7, 0, 0xDD),       // l unite 7 n a plus de piste a la frame 30
			replique(auTemps(40, 0), -1, 1, 0xEE),      // aucune reference d unite
			replique(auTemps(0, 0), 5, -1, 0x01),       // la premiere frame de l axe
			replique(auTemps(100, 0), 5, -1, 0x02),     // apres la derniere frame
			replique(axeDeRepliques.origin-1, 5, -1, 0x03),
		},
	}
	got, cov := buildChatter(in, pistes, axeDeRepliques, "")

	ligne := entierOptionnel(9, true)
	want := []Chatter{
		{T: 0, Player: "111", Sound: "00000001", Priority: 7, Line: ligne},
		{T: 10, Player: "111", Sound: "000000aa", Priority: 7, Line: ligne},
		{T: 20, Sound: "000000cc", Priority: 7, Line: ligne},
		{T: 30, Sound: "000000dd", Priority: 7, Line: ligne},
		{T: 40, Sound: "000000ee", Priority: 7, Line: ligne},
		{T: 60, Player: "222", Sound: "000000bb", Priority: 7, Line: ligne},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("repliques %+v,\nattendu %+v", got, want)
	}
	wantCov := ChatterCoverage{Frames: 80, ClosedFrames: 78, StoppedFrames: 2, Read: 8, BeforeOrigin: 1, AfterEnd: 1,
		Published: 6, BySpeaker: 3, WithoutPlayer: 3, Bots: 1, Unnamed: 6}
	if cov != wantCov {
		t.Errorf("couverture %+v, attendu %+v", cov, wantCov)
	}
	if cov.Read != cov.BeforeOrigin+cov.AfterEnd+cov.Published || cov.Published != cov.BySpeaker+cov.WithoutPlayer ||
		cov.Published != cov.Named+cov.Unnamed {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
}

// TestUneRepliqueSansSonNiIndiceNePorteQueSaPriorite : les champs que le message ne porte pas sont absents du document, et la
// priorite, toujours lue, est publiee meme a zero.
func TestUneRepliqueSansSonNiIndiceNePorteQueSaPriorite(t *testing.T) {
	in := grammar.RepliquesLues{Repliques: []grammar.RepliqueLue{{TS: auTemps(3, 0), Joueur: -1}}}
	got, _ := buildChatter(in, nil, axeDeRepliques, "")
	want := []Chatter{{T: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("repliques %+v, attendu %+v", got, want)
	}
}

// TestLesAnnoncesSeFondentParSonEtParFrame : les messages d une frame qui portent le meme son sont une ligne dont le masque est
// l union des leurs ; un autre son, une autre frame ou une annonce sans son sont d autres lignes ; hors de l axe, ecartees ;
// l identite de la couverture tient. MUTATIONS : la cle sans la frame, la cle sans le son, un masque pris au premier message
// seulement, un message repris non compte -> ROUGE.
func TestLesAnnoncesSeFondentParSonEtParFrame(t *testing.T) {
	a := func(ts uint64, son uint32, aSon bool, masque uint32) grammar.AnnonceLue {
		return grammar.AnnonceLue{TS: ts, Son: son, ASon: aSon, Destinataires: masque, Joueur: 0}
	}
	in := grammar.RepliquesLues{
		Stats: grammar.StatsRepliques{Trames: 10, Terminees: 10, Annonces: 9},
		Annonces: []grammar.AnnonceLue{
			a(auTemps(5, 0), 0x11, true, 0x01000000),
			a(auTemps(5, 100), 0x11, true, 0x04000000),
			a(auTemps(5, 200), 0x22, true, 0x01000000),
			a(auTemps(6, 0), 0x11, true, 0x01000000),
			a(auTemps(7, 0), 0, false, 0x80000000),
			a(auTemps(7, 0), 0, false, 0x40000000),
			a(auTemps(5, 300), 0x11, true, 0x01000000), // un doublon exact : repris aussi
			a(axeDeRepliques.origin-1, 0x11, true, 0x01000000),
			a(auTemps(100, 0), 0x11, true, 0x01000000),
		},
	}
	got, cov := buildAnnouncements(in, axeDeRepliques)
	want := []Announcement{
		{T: 5, Sound: "00000011", Mask: "05000000"},
		{T: 5, Sound: "00000022", Mask: "01000000"},
		{T: 6, Sound: "00000011", Mask: "01000000"},
		{T: 7, Mask: "c0000000"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("annonces %+v,\nattendu %+v", got, want)
	}
	wantCov := AnnouncementCoverage{Frames: 10, ClosedFrames: 10, Read: 9, BeforeOrigin: 1, AfterEnd: 1, Merged: 3, Published: 4}
	if cov != wantCov {
		t.Errorf("couverture %+v, attendu %+v", cov, wantCov)
	}
	if cov.Read != cov.BeforeOrigin+cov.AfterEnd+cov.Merged+cov.Published {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
}

// TestLeCalqueDesRepliquesNeSePoseQueSiLaMarcheADesTrames : la passe pose les deux calques et leurs couvertures quand la marche a
// lu au moins une trame, sur les pistes deja posees ; sans trame lue, ni calque ni couverture.
func TestLeCalqueDesRepliquesNeSePoseQueSiLaMarcheADesTrames(t *testing.T) {
	passe := func(in grammar.RepliquesLues) *assemblage {
		a := &assemblage{ctx: context.Background(), opt: Options{Repliques: in}, origin: axeDeRepliques.origin,
			step: axeDeRepliques.step}
		a.doc.FrameCount = axeDeRepliques.frames
		a.doc.Tracks = []Track{pisteDUnite(5, "111", 0, 0)}
		a.doc.Coverage = &Coverage{}
		a.poserLesRepliques()
		return a
	}
	a := passe(grammar.RepliquesLues{
		Stats:     grammar.StatsRepliques{Trames: 4, Terminees: 4, Repliques: 1, Annonces: 1},
		Repliques: []grammar.RepliqueLue{replique(auTemps(2, 0), 5, -1, 0xAA)},
		Annonces:  []grammar.AnnonceLue{{TS: auTemps(3, 0), Son: 0x11, ASon: true, Destinataires: 0x10000000, Joueur: -1}},
	})
	if len(a.doc.Chatter) != 1 || a.doc.Chatter[0].Player != "111" || a.doc.Coverage.Chatter == nil || a.doc.Coverage.Chatter.BySpeaker != 1 {
		t.Errorf("repliques %+v, couverture %+v", a.doc.Chatter, a.doc.Coverage.Chatter)
	}
	if len(a.doc.Announcements) != 1 || a.doc.Announcements[0].Mask != "10000000" || a.doc.Coverage.Announcements == nil {
		t.Errorf("annonces %+v, couverture %+v", a.doc.Announcements, a.doc.Coverage.Announcements)
	}
	vide := passe(grammar.RepliquesLues{})
	if vide.doc.Chatter != nil || vide.doc.Announcements != nil || vide.doc.Coverage.Chatter != nil || vide.doc.Coverage.Announcements != nil {
		t.Errorf("sans trame lue, rien ne doit etre pose : %+v", vide.doc)
	}
}

// TestLesRepliquesFontLAllerRetourDesFaits : les comptes du canal, chaque replique et chaque annonce se relisent a l identique ;
// une liste vide se relit nil ; un flux tronque rend une erreur.
func TestLesRepliquesFontLAllerRetourDesFaits(t *testing.T) {
	cas := []grammar.RepliquesLues{
		{},
		{Stats: grammar.StatsRepliques{Trames: 3, Terminees: 3}},
		{
			Stats: grammar.StatsRepliques{Trames: 9000, Terminees: 8990, Arretees: 10, Repliques: 3, Annonces: 2},
			Repliques: []grammar.RepliqueLue{
				{TS: 4_517_903_000, Son: 0xcafebabe, Locuteur: 300, GenLocuteur: 2, Ligne: 0xabc, Priorite: 13, Joueur: 17,
					ASon: true, ALigne: true, ALocuteur: true},
				{TS: 4_517_903_000, Priorite: 5, Joueur: -1},
				{TS: 4_517_999_000, Son: 1, Priorite: 31, Joueur: 0, ASon: true},
			},
			Annonces: []grammar.AnnonceLue{
				{TS: 4_517_000_000, Son: 0x01020304, Destinataires: 0xa0000001, Joueur: -1, ASon: true},
				{TS: 4_517_000_000, Destinataires: 0x80000000, Joueur: 3},
			},
		},
	}
	for _, c := range cas {
		w := &gwriter{}
		encodeRepliques(w, c)
		r := &greader{b: w.b}
		got := decodeRepliques(r)
		if r.err != nil || r.off != len(r.b) {
			t.Fatalf("relecture invalide : err %v, %d/%d octets", r.err, r.off, len(r.b))
		}
		if !reflect.DeepEqual(got, c) {
			t.Errorf("relu %+v, attendu %+v", got, c)
		}
		if len(w.b) > 12 {
			tronque := &greader{b: w.b[:len(w.b)-3]}
			if _ = decodeRepliques(tronque); tronque.err == nil {
				t.Errorf("un flux tronque doit rendre une erreur")
			}
		}
	}
}

// TestLeNomDeLaRepliqueVientDeLIndiceEtDuBuild : une replique prend le nom de son INDICE (pas de son son) quand le build du film a
// une disposition mesuree et que l indice a un nom prouve ; sinon elle est publiee sans nom et comptee. L identite
// published = named + unnamed tient. MUTATIONS : le nom pris du son, le garde de build retire, une replique sans indice nommee,
// un nom compte deux fois -> ROUGE.
func TestLeNomDeLaRepliqueVientDeLIndiceEtDuBuild(t *testing.T) {
	r := func(f int, son uint32, ligne uint16, aLigne bool) grammar.RepliqueLue {
		return grammar.RepliqueLue{TS: auTemps(f, 0), Son: son, ASon: true, Ligne: ligne, ALigne: aLigne, Priorite: 7, Joueur: -1}
	}
	in := grammar.RepliquesLues{Repliques: []grammar.RepliqueLue{
		r(1, 0x11, 2926, true), // indice nomme
		r(2, 0x22, 2926, true), // meme indice, autre voix : meme nom
		r(3, 0x11, 2822, true), // indice sans nom prouve
		r(4, 0x11, 0, false),   // sans indice
	}}
	got, cov := buildChatter(in, nil, axeDeRepliques, "HI_1_13_0")
	noms := []string{got[0].Name, got[1].Name, got[2].Name, got[3].Name}
	if want := []string{"seeingenemynearby", "seeingenemynearby", "", ""}; !reflect.DeepEqual(noms, want) {
		t.Errorf("noms %v, attendu %v", noms, want)
	}
	if cov.Named != 2 || cov.Unnamed != 2 || cov.Published != 4 {
		t.Errorf("couverture %+v : attendu 2 nommees, 2 sans nom sur 4", cov)
	}
	for _, build := range []string{"", "HI_1_11_0", "HI_1_14_0"} {
		got, cov := buildChatter(in, nil, axeDeRepliques, build)
		if cov.Named != 0 || cov.Unnamed != 4 || got[0].Name != "" {
			t.Errorf("build %q : aucun nom attendu, couverture %+v", build, cov)
		}
	}
}
