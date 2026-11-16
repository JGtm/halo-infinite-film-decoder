package replay

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Effets que les films du corpus nomment (damageeffects/data/effects.tsv) : le plus fréquent du film de référence
// (12 points), un qui n'a de multiplicateur du corps que 0,6 (7 points), un qui ne déborde pas (29 points), un qui
// déborde pour moitié (80 points), une grenade (zone), un effet de mêlée que la table connaît sans points.
const (
	effetDe12      = 0x17d6bd23
	effetDe7       = 0x0000d580
	effetSansDebor = 0x1b6370f9
	effetDe80      = 0xf81f1a01
	effetZone      = 0xff768294
	effetSansPoint = 0x000027cd
)

// vitalesDeTest : la vie de l'unité d'index 5, génération 1 (slot 517) : créée à l'instant 100, lue à 200 puis à 300.
func vitalesDeTest(sante0, bouclier0, sante1, bouclier1 float32) grammar.VitalitesLues {
	lire := func(ts uint64, s, b float32) grammar.LectureVitale {
		return grammar.LectureVitale{TS: ts, Slot: 517, Gen: 1, ASante: true, ABouclier: true, Sante: s, Bouclier: b}
	}
	return grammar.VitalitesLues{Lectures: []grammar.LectureVitale{
		{TS: 100, Slot: 517, Gen: 1, Creation: true}, lire(200, sante0, bouclier0), lire(300, sante1, bouclier1)}}
}

func coupDeTest(tag uint32, echelle uint8) grammar.DegatLu {
	return grammar.DegatLu{TS: 300, Tag: tag, Victime: 5, GenVictime: 1, Echelle: echelle, ATag: true, AVictime: true}
}

// TestLesPointsDUnCoupSuiventLaTableEtLaVieLue : un coup calculé retire ce que la table dit, du bouclier puis du corps,
// depuis la vie lue avant l'instant ; les coups d'une scène se suivent dans l'ordre du film ; un coup que la table ne
// chiffre pas (zone, échelle réduite, effet inconnu) partage ce que la vie perd sans qu'un coup calculé l'explique ; un
// effet connu sans points ne retire rien ; un tir à la tête qui achève retire ce qui reste.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : partir de la vie à l'instant au lieu d'avant ; ne pas déduire le premier
// coup avant le second ; ne pas partager la perte entre les coups mesurés ; dépasser ce qui reste ; compter un effet
// connu sans points ; oublier le tir à la tête ; laisser déborder l'effet qui ne déborde pas.
func TestLesPointsDUnCoupSuiventLaTableEtLaVieLue(t *testing.T) {
	cas := []struct {
		nom    string
		vie    grammar.VitalitesLues
		coups  []grammar.DegatLu
		points []PointsDeDegat
	}{
		{"un coup de 12 sur un bouclier plein", vitalesDeTest(1, 1, 1, 128.0/140), []grammar.DegatLu{coupDeTest(effetDe12, 15)},
			[]PointsDeDegat{{Bouclier: 120}}},
		{"le bouclier ne tient que quatre points : le reste déborde au corps, à 0,6", vitalesDeTest(1, 4.0/140, 1, 0),
			[]grammar.DegatLu{coupDeTest(effetDe7, 15)}, []PointsDeDegat{{Bouclier: 40, Corps: 18}}},
		{"un effet qui ne déborde pas s'arrête au bouclier", vitalesDeTest(1, 20.0/140, 1, 0), []grammar.DegatLu{coupDeTest(effetSansDebor, 15)},
			[]PointsDeDegat{{Bouclier: 200}}},
		{"un coup de 80 déborde pour moitié", vitalesDeTest(1, 50.0/140, 1, 0), []grammar.DegatLu{coupDeTest(effetDe80, 15)},
			[]PointsDeDegat{{Bouclier: 500, Corps: 150}}},
		{"deux coups à l'instant : le second part de ce que le premier a laissé", vitalesDeTest(1, 18.0/140, 1, 0),
			[]grammar.DegatLu{coupDeTest(effetDe12, 15), coupDeTest(effetDe12, 15)}, []PointsDeDegat{{Bouclier: 120}, {Bouclier: 60, Corps: 60}}},
		{"échelle réduite : mesuré sur ce que la vie perd", vitalesDeTest(1, 1, 1, 100.0/140), []grammar.DegatLu{coupDeTest(effetDe12, 7)},
			[]PointsDeDegat{{Bouclier: 400, Mesure: true}}},
		{"une zone et un effet inconnu partagent la perte", vitalesDeTest(1, 1, 1, 120.0/140), []grammar.DegatLu{coupDeTest(effetZone, 15), coupDeTest(0x12345678, 15)},
			[]PointsDeDegat{{Bouclier: 100, Mesure: true}, {Bouclier: 100, Mesure: true}}},
		{"un coup calculé puis un mesuré : le mesuré n'a que le reste de la perte", vitalesDeTest(1, 1, 1, 90.0/140),
			[]grammar.DegatLu{coupDeTest(effetDe12, 15), coupDeTest(effetZone, 15)}, []PointsDeDegat{{Bouclier: 120}, {Bouclier: 380, Mesure: true}}},
		{"un effet connu qui ne retire rien", vitalesDeTest(1, 1, 1, 1), []grammar.DegatLu{coupDeTest(effetSansPoint, 15)}, []PointsDeDegat{{}}},
		{"un tir à la tête qui achève retire tout ce qui reste du corps", vitalesDeTest(50.0/90, 0, 0, 0), []grammar.DegatLu{coupDeTest(effetDe12, 15)},
			[]PointsDeDegat{{Corps: 500}}},
		{"une unité sans vie lue part pleine", grammar.VitalitesLues{}, []grammar.DegatLu{coupDeTest(effetDe12, 15)}, []PointsDeDegat{{Bouclier: 120}}},
		{"un message sans unité blessée ne retire rien", vitalesDeTest(1, 1, 1, 0.5),
			[]grammar.DegatLu{{TS: 300, Tag: effetDe12, Victime: 5, GenVictime: 1, Echelle: 15, ATag: true}}, []PointsDeDegat{{}}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := calculerLesPointsDeDegats(c.coups, c.vie)
			if !slices.Equal(got, c.points) {
				t.Errorf("points %+v, attendu %+v", got, c.points)
			}
		})
	}
}

// TestUnMessageSansUniteNeSeMelePasALUniteZero : un message sans unité blessée (index 0, génération 0 par défaut) et un vrai
// message sur l'unité 0 de génération 0, au même instant, ne forment pas une scène : le vrai message est chiffré de la
// même façon quel que soit l'ordre des deux.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : regrouper sans tenir compte de la présence de l'unité blessée.
func TestUnMessageSansUniteNeSeMelePasALUniteZero(t *testing.T) {
	vie := grammar.VitalitesLues{Lectures: []grammar.LectureVitale{
		{TS: 100, Slot: 512, Gen: 0, Creation: true},
		{TS: 200, Slot: 512, Gen: 0, ASante: true, ABouclier: true, Sante: 1, Bouclier: 1},
		{TS: 300, Slot: 512, Gen: 0, ASante: true, ABouclier: true, Sante: 1, Bouclier: 128.0 / 140},
	}}
	vrai := grammar.DegatLu{TS: 300, Tag: effetDe12, Victime: 0, GenVictime: 0, Echelle: 15, ATag: true, AVictime: true}
	orphelin := grammar.DegatLu{TS: 300, Tag: effetDe12, Echelle: 15, ATag: true}
	want := PointsDeDegat{Bouclier: 120}
	if got := calculerLesPointsDeDegats([]grammar.DegatLu{orphelin, vrai}, vie); got[0] != (PointsDeDegat{}) || got[1] != want {
		t.Errorf("orphelin d'abord : %+v, attendu {} puis %+v", got, want)
	}
	if got := calculerLesPointsDeDegats([]grammar.DegatLu{vrai, orphelin}, vie); got[0] != want || got[1] != (PointsDeDegat{}) {
		t.Errorf("orphelin ensuite : %+v, attendu %+v puis {}", got, want)
	}
}
