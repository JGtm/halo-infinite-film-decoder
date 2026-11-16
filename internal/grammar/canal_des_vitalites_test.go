package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// compDeVie : un composant de vie porté qui rend la valeur `payload`.
func compDeVie(payload any) CompResult { return CompResult{Ported: true, Payload: payload} }

// TestLeCanalDesVitalitesLitLaVieDesBipedes : une création est une lecture à part ; une mise à jour porte la santé, le
// bouclier ou les deux, avec la vie (slot et génération du record) et l'instant de la trame ; un composant non porté, un
// record d'un autre archétype et un delta sans vitalité ne rendent rien.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : lire la génération dans le slot ; perdre un drapeau de présence ; ranger un record
// d'un autre archétype ; ranger un delta sans vitalité ; ne pas compter les créations.
func TestLeCanalDesVitalitesLitLaVieDesBipedes(t *testing.T) {
	c := nouveauCanalDesVitalites()
	recs := []FrameRecord{
		{Type: recNew, TypeIndex: BipedTypeIndex, Slot: 517, ID: 517 | 2<<30},
		{Type: recDelta, TypeIndex: BipedTypeIndex, Slot: 517, ID: 517 | 2<<30,
			Trace: EntityTrace{Comps: []CompResult{compDeVie(BodyVitality{Health: 0.5}), compDeVie(ShieldVitality{Shield: 0.25})}}},
		{Type: recDelta, TypeIndex: BipedTypeIndex, Slot: 520, ID: 520 | 1<<30,
			Trace: EntityTrace{Comps: []CompResult{compDeVie(ShieldVitality{Shield: 1})}}},
		{Type: recDelta, TypeIndex: BipedTypeIndex, Slot: 521, ID: 521, Trace: EntityTrace{Comps: []CompResult{compDeVie(BodyVitality{Health: 1})}}},
		{Type: recDelta, TypeIndex: BipedTypeIndex, Slot: 522, ID: 522, Trace: EntityTrace{Comps: []CompResult{{Ported: false, Payload: BodyVitality{Health: 1}}}}},
		{Type: recDelta, TypeIndex: BipedTypeIndex, Slot: 523, ID: 523, Trace: EntityTrace{Comps: []CompResult{compDeVie(nil)}}},
		{Type: recDelta, TypeIndex: 36, Slot: 524, ID: 524, Trace: EntityTrace{Comps: []CompResult{compDeVie(BodyVitality{Health: 1})}}},
	}
	c.lire(1_000, recs)
	want := []LectureVitale{
		{TS: 1_000, Slot: 517, Gen: 2, Creation: true},
		{TS: 1_000, Slot: 517, Gen: 2, ASante: true, Sante: 0.5, ABouclier: true, Bouclier: 0.25},
		{TS: 1_000, Slot: 520, Gen: 1, ABouclier: true, Bouclier: 1},
		{TS: 1_000, Slot: 521, Gen: 0, ASante: true, Sante: 1},
	}
	if len(c.lu.Lectures) != len(want) {
		t.Fatalf("%d lecture(s) %+v, attendu %d", len(c.lu.Lectures), c.lu.Lectures, len(want))
	}
	for i := range want {
		if c.lu.Lectures[i] != want[i] {
			t.Errorf("lecture %d : %+v, attendu %+v", i, c.lu.Lectures[i], want[i])
		}
	}
	if c.lu.Stats.Creations != 1 || c.lu.Stats.Lectures != len(want) {
		t.Errorf("comptes %+v, attendu 1 création et %d lectures", c.lu.Stats, len(want))
	}
}

// TestLeCanalDesVitalitesRelitLesListesQueLaMarcheNeLocalisePas : sur la bobine réelle, une trame dont la liste est annoncée et
// que la marche ne localise pas est relue comme le fait le canal des morts ; celles qui restent illisibles et que la vue A
// n'arrête pas déjà sont comptées, et les autres sont lues. Les comptes se refont indépendamment, par une marche des trames
// qui relit la liste de la même façon.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas relire la liste (les trames se comptent non lues) ; compter les trames
// que la vue A arrête ; ne pas compter une trame illisible.
func TestLeCanalDesVitalitesRelitLesListesQueLaMarcheNeLocalisePas(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	marche, err := ScanMarcheDesTramesAvec(fc, LecturesDeLaMarche{})
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	m, err := NewFilmContext(film).nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	lues, recuperees, perdues, libres := 0, 0, 0, 0
	m.parcourir(func(tr *trameLue) bool {
		if tr.debut >= 0 {
			lues++
			return true
		}
		if !listeAnnoncee(&tr.paquet.VueA) {
			return true
		}
		md := &MarcheDistribuee{Paquet: tr.paquet, marche: m}
		if _, ok, libre := md.recupererLaListe(); ok {
			lues++
			recuperees++
			libres += unSi(libre)
		} else if tr.paquet.VueA.Etat != lecture.VueArretee {
			perdues++
		}
		return true
	})
	if recuperees == 0 {
		t.Fatal("aucune liste recuperee sur la bobine : le test ne garde rien")
	}
	got := marche.Vitalites.Stats
	if got.LocalisationsALargeurLibre != libres || fc.ComptesDesReplis().LocalisationsALargeurLibre != libres {
		t.Errorf("%d liste(s) a largeur libre comptee(s) au canal et %d au rapport des replis du contexte ; attendu %d", got.LocalisationsALargeurLibre,
			fc.ComptesDesReplis().LocalisationsALargeurLibre, libres)
	}
	if got.Trames != lues || got.TramesNonLues != perdues {
		t.Errorf("%d trame(s) lue(s) et %d non lue(s) ; attendu %d et %d (dont %d listes recuperees)", got.Trames, got.TramesNonLues, lues, perdues, recuperees)
	}
}

// TestLesListesALargeurLibreDesVitalitesRejoignentLeRepliDesMorts : les listes que le canal des vitalites recupere a largeur libre
// se versent au rapport des replis du contexte, sous le MEME compteur que celles du canal des morts.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas verser le compte.
func TestLesListesALargeurLibreDesVitalitesRejoignentLeRepliDesMorts(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	m := MarcheDesTrames{Vitalites: VitalitesLues{Stats: StatsVitalites{LocalisationsALargeurLibre: 3}}}
	m.verserLesReplisDesVitalites(fc)
	if got := fc.ComptesDesReplis().LocalisationsALargeurLibre; got != 3 {
		t.Fatalf("%d liste(s) a largeur libre au rapport, attendu 3", got)
	}
}
